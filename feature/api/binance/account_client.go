package binance

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"go_binance_futures/service/binanceapiusage"

	"github.com/adshao/go-binance/v2/futures"
)

// AccountID names the credential and private exchange state boundary.
// A secondary account never falls back to the legacy main-account client.
type AccountID string

const (
	MainAccountID AccountID = "main"
	LeadAccountID AccountID = "lead"
)

type AccountClient struct {
	id       AccountID
	client   *futures.Client
	signedMu sync.RWMutex
	reads    accountReadState
	config   tradeConfigState
}

func (a *AccountClient) ID() AccountID {
	if a == nil {
		return ""
	}
	return a.id
}

// NewAccountClient constructs an isolated adapter around a supplied SDK client.
// It has no global registration, startup hooks, database access, or live orders.
func NewAccountClient(id AccountID, client *futures.Client) (*AccountClient, error) {
	if (id != MainAccountID && id != LeadAccountID) || client == nil {
		return nil, fmt.Errorf("invalid futures account client")
	}
	return &AccountClient{id: id, client: client}, nil
}

// NewLeadAccountClient creates an inert lead adapter. Credentials must come
// from a future secure configuration workflow, never the main app.conf keys.
func NewLeadAccountClient(apiKey, apiSecret string, httpClient *http.Client) (*AccountClient, error) {
	if strings.TrimSpace(apiKey) == "" || strings.TrimSpace(apiSecret) == "" {
		return nil, fmt.Errorf("lead API credentials are required")
	}
	c := futures.NewClient(apiKey, apiSecret)
	c.SetApiEndpoint("https://fapi.binance.com")
	if httpClient == nil {
		httpClient = proxyPool.HTTPClient()
	}
	// The limiter is per Lead client; the existing V4-5 coordinator continues
	// to enforce the shared IP/request-weight budget.
	wrapped := *httpClient
	base := wrapped.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	wrapped.Transport = &leadOrderLimiter{base: base}
	c.HTTPClient = binanceapiusage.WrapClient(&wrapped, binanceapiusage.TransportConfig{Product: "futures", Environment: "mainnet", Source: "lead_trading"})
	return NewAccountClient(LeadAccountID, c)
}

func doAccountSigned[T any](a *AccountClient, ctx context.Context, window int64, call func(*futures.Client, context.Context, ...futures.RequestOption) (T, error)) (T, error) {
	var zero T
	if a == nil || a.client == nil {
		return zero, fmt.Errorf("futures account client is unavailable")
	}
	a.signedMu.Lock()
	res, err := call(a.client, ctx, futures.WithRecvWindow(window))
	a.signedMu.Unlock()
	if err == nil || !isFuturesTimestampError(err) {
		return res, err
	}
	a.signedMu.Lock()
	started := time.Now().UnixMilli()
	serverTime, syncErr := a.client.NewServerTimeService().Do(ctx)
	finished := time.Now().UnixMilli()
	if syncErr == nil {
		a.client.TimeOffset = ((started + finished) / 2) - serverTime
	}
	a.signedMu.Unlock()
	if syncErr != nil {
		return res, fmt.Errorf("account %s timestamp resync failed: %w", a.id, syncErr)
	}
	a.signedMu.Lock()
	res, err = call(a.client, ctx, futures.WithRecvWindow(window))
	a.signedMu.Unlock()
	return res, err
}
func (a *AccountClient) GetFuturesAccountContext(ctx context.Context) (*futures.Account, error) {
	return doAccountSigned(a, ctx, futuresSignedReadRecvWindow, func(c *futures.Client, ctx context.Context, o ...futures.RequestOption) (*futures.Account, error) {
		return c.NewGetAccountService().Do(ctx, o...)
	})
}
func (a *AccountClient) GetPositionContext(ctx context.Context, params PositionParams) ([]*futures.PositionRisk, error) {
	return a.getPosition(ctx, params, false)
}
func (a *AccountClient) GetPositionFreshContext(ctx context.Context, params PositionParams) ([]*futures.PositionRisk, error) {
	return a.getPosition(ctx, params, true)
}
func (a *AccountClient) getPosition(ctx context.Context, params PositionParams, fresh bool) ([]*futures.PositionRisk, error) {
	if a == nil {
		return nil, fmt.Errorf("nil account")
	}
	loader := func(ctx context.Context) ([]*futures.PositionRisk, error) {
		return doAccountSigned(a, ctx, futuresSignedReadRecvWindow, func(c *futures.Client, ctx context.Context, o ...futures.RequestOption) ([]*futures.PositionRisk, error) {
			svc := c.NewGetPositionRiskService()
			if params.Symbol != "" {
				svc = svc.Symbol(params.Symbol)
			}
			return svc.Do(ctx, o...)
		})
	}
	if !isAllAccountPositionRequest(params) {
		return loader(ctx)
	}
	if fresh {
		return a.reads.positionsFresh(ctx, loader)
	}
	return a.reads.positionsCached(ctx, loader)
}
func (a *AccountClient) GetOpenOrderContext(ctx context.Context, symbols ...string) ([]*futures.Order, error) {
	return a.getOpenOrders(ctx, false, symbols...)
}
func (a *AccountClient) GetOpenOrderFreshContext(ctx context.Context, symbols ...string) ([]*futures.Order, error) {
	return a.getOpenOrders(ctx, true, symbols...)
}
func (a *AccountClient) getOpenOrders(ctx context.Context, fresh bool, symbols ...string) ([]*futures.Order, error) {
	if a == nil {
		return nil, fmt.Errorf("nil account")
	}
	loader := func(ctx context.Context) ([]*futures.Order, error) {
		return doAccountSigned(a, ctx, futuresSignedReadRecvWindow, func(c *futures.Client, ctx context.Context, o ...futures.RequestOption) ([]*futures.Order, error) {
			svc := c.NewListOpenOrdersService()
			if len(symbols) > 0 && symbols[0] != "" {
				svc = svc.Symbol(symbols[0])
			}
			return svc.Do(ctx, o...)
		})
	}
	if len(symbols) > 0 && symbols[0] != "" {
		return loader(ctx)
	}
	if fresh {
		return a.reads.ordersFresh(ctx, loader)
	}
	return a.reads.ordersCached(ctx, loader)
}
func (a *AccountClient) InvalidateAccountReadCache() {
	if a != nil {
		a.reads.invalidate()
	}
}
func (a *AccountClient) GetOrderByClientOrderID(ctx context.Context, symbol, id string) (*futures.Order, error) {
	return doAccountSigned(a, ctx, futuresSignedReadRecvWindow, func(c *futures.Client, ctx context.Context, o ...futures.RequestOption) (*futures.Order, error) {
		return c.NewGetOrderService().Symbol(symbol).OrigClientOrderID(id).Do(ctx, o...)
	})
}
func (a *AccountClient) GetOrderByOrderID(ctx context.Context, symbol string, id int64) (*futures.Order, error) {
	return doAccountSigned(a, ctx, futuresSignedReadRecvWindow, func(c *futures.Client, ctx context.Context, o ...futures.RequestOption) (*futures.Order, error) {
		return c.NewGetOrderService().Symbol(symbol).OrderID(id).Do(ctx, o...)
	})
}
func (a *AccountClient) CreateOwnedOrder(ctx context.Context, p OwnedOrderParams) (*futures.CreateOrderResponse, error) {
	res, err := doAccountSigned(a, ctx, futuresSignedTradeRecvWindow, func(c *futures.Client, ctx context.Context, o ...futures.RequestOption) (*futures.CreateOrderResponse, error) {
		svc := c.NewCreateOrderService().Symbol(p.Symbol).Side(p.Side).PositionSide(p.PositionSide).Type(p.OrderType).Quantity(formatOwnedOrderDecimal(p.Quantity)).NewClientOrderID(p.ClientOrderID)
		if p.OrderType == futures.OrderTypeLimit {
			svc = svc.TimeInForce(futures.TimeInForceTypeGTC).Price(formatOwnedOrderDecimal(p.Price))
		}
		if p.StopPrice > 0 {
			svc = svc.StopPrice(formatOwnedOrderDecimal(p.StopPrice))
		}
		return svc.Do(ctx, o...)
	})
	a.InvalidateAccountReadCache()
	return res, err
}
func (a *AccountClient) CreateOwnedAlgoOrder(ctx context.Context, p OwnedOrderParams) (*futures.CreateAlgoOrderResp, error) {
	res, err := doAccountSigned(a, ctx, futuresSignedTradeRecvWindow, func(c *futures.Client, ctx context.Context, o ...futures.RequestOption) (*futures.CreateAlgoOrderResp, error) {
		svc := c.NewCreateAlgoOrderService().Symbol(p.Symbol).Side(p.Side).PositionSide(p.PositionSide).Type(futures.AlgoOrderType(strings.ToUpper(strings.TrimSpace(string(p.OrderType))))).Quantity(formatOwnedOrderDecimal(p.Quantity)).ClientAlgoId(p.ClientOrderID)
		if p.StopPrice > 0 {
			svc = svc.TriggerPrice(formatOwnedOrderDecimal(p.StopPrice))
		}
		if p.Price > 0 {
			svc = svc.Price(formatOwnedOrderDecimal(p.Price))
		}
		return svc.Do(ctx, o...)
	})
	a.InvalidateAccountReadCache()
	return res, err
}
func (a *AccountClient) GetAlgoOrderByClientOrderID(ctx context.Context, id string) (*futures.GetAlgoOrderResp, error) {
	return doAccountSigned(a, ctx, futuresSignedReadRecvWindow, func(c *futures.Client, ctx context.Context, o ...futures.RequestOption) (*futures.GetAlgoOrderResp, error) {
		return c.NewGetAlgoOrderService().ClientAlgoID(id).Do(ctx, o...)
	})
}
func (a *AccountClient) CancelAlgoOrder(ctx context.Context, id int64) (*futures.CancelAlgoOrderResp, error) {
	res, err := doAccountSigned(a, ctx, futuresSignedTradeRecvWindow, func(c *futures.Client, ctx context.Context, o ...futures.RequestOption) (*futures.CancelAlgoOrderResp, error) {
		return c.NewCancelAlgoOrderService().AlgoID(id).Do(ctx, o...)
	})
	a.InvalidateAccountReadCache()
	return res, err
}
func (a *AccountClient) CancelOrderContext(ctx context.Context, symbol string, id int64) (*futures.CancelOrderResponse, error) {
	res, err := doAccountSigned(a, ctx, futuresSignedTradeRecvWindow, func(c *futures.Client, ctx context.Context, o ...futures.RequestOption) (*futures.CancelOrderResponse, error) {
		return c.NewCancelOrderService().Symbol(symbol).OrderID(id).Do(ctx, o...)
	})
	a.InvalidateAccountReadCache()
	return res, err
}
func (a *AccountClient) SetLeverageContext(ctx context.Context, symbol string, leverage int) (*futures.SymbolLeverage, error) {
	res, err := doAccountSigned(a, ctx, futuresSignedTradeRecvWindow, func(c *futures.Client, ctx context.Context, o ...futures.RequestOption) (*futures.SymbolLeverage, error) {
		return c.NewChangeLeverageService().Symbol(symbol).Leverage(leverage).Do(ctx, o...)
	})
	if err == nil {
		a.InvalidateAccountReadCache()
		a.InvalidateTradeConfig(symbol)
	}
	return res, err
}
func (a *AccountClient) SetMarginTypeContext(ctx context.Context, symbol string, margin futures.MarginType) error {
	_, err := doAccountSigned(a, ctx, futuresSignedTradeRecvWindow, func(c *futures.Client, ctx context.Context, o ...futures.RequestOption) (struct{}, error) {
		return struct{}{}, c.NewChangeMarginTypeService().Symbol(symbol).MarginType(margin).Do(ctx, o...)
	})
	if err == nil {
		a.InvalidateAccountReadCache()
		a.InvalidateTradeConfig(symbol)
	}
	return err
}
func (a *AccountClient) EnsureTradeConfigContext(ctx context.Context, symbol string, margin futures.MarginType, leverage int) error {
	if a == nil {
		return errors.New("nil account")
	}
	return a.config.ensure(ctx, symbol, margin, leverage, a.SetMarginTypeContext, a.SetLeverageContext)
}
func (a *AccountClient) InvalidateTradeConfig(symbol string) {
	if a != nil {
		a.config.invalidate(symbol)
	}
}
func (a *AccountClient) GetListenKey(ctx context.Context) (string, error) {
	return doAccountSigned(a, ctx, futuresSignedReadRecvWindow, func(c *futures.Client, ctx context.Context, o ...futures.RequestOption) (string, error) {
		return c.NewStartUserStreamService().Do(ctx, o...)
	})
}
func (a *AccountClient) UpdateListenKey(ctx context.Context, key string) error {
	_, err := doAccountSigned(a, ctx, futuresSignedReadRecvWindow, func(c *futures.Client, ctx context.Context, o ...futures.RequestOption) (struct{}, error) {
		return struct{}{}, c.NewKeepaliveUserStreamService().ListenKey(key).Do(ctx, o...)
	})
	return err
}

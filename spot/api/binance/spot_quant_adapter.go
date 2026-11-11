package binance

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"go_binance_futures/service/binanceapiusage"
	"go_binance_futures/service/spotexecution"

	sdk "github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/common"
)

// QuantAdapter uses the project's Spot client (existing proxy + REST limit
// wrapper + ticker cache). It does not reuse legacy Spot rush order methods.
// No main/router registers the 06-4 processor; none of this runs on import.
type QuantAdapter struct {
	c                  *sdk.Client
	allowOrderMutation bool
}

// NewSpotQuantAdapter is read-only by default. Live execution/cancel must
// only be wired during the separately authorized 06-6 deployment gate.
func NewSpotQuantAdapter() *QuantAdapter { return &QuantAdapter{c: client} }
func NewSpotQuantAdapterWithClient(c *sdk.Client, permitOrderMutation bool) (*QuantAdapter, error) {
	if c == nil {
		return nil, fmt.Errorf("nil Spot Binance client")
	}
	// 06-4 is a Fake/Testnet development stage. Production mutations are
	// deliberately impossible even if someone mistakenly passes true.
	// Mainnet release needs a separate, explicit 06-6 code/review gate.
	if permitOrderMutation && strings.TrimRight(c.BaseURL, "/") != "https://testnet.binance.vision" {
		return nil, fmt.Errorf("Spot quant order mutations are restricted to Binance Spot Testnet in 06-4")
	}
	return &QuantAdapter{c: c, allowOrderMutation: permitOrderMutation}, nil
}

var _ spotexecution.Exchange = (*QuantAdapter)(nil)

func (a *QuantAdapter) Rules(ctx context.Context, symbol string) (spotexecution.Rules, error) {
	var out spotexecution.Rules
	ctx = binanceapiusage.WithSource(ctx, "spot_quant_rules")
	// Use the cached complete exchange info when using the shared client.
	var data *sdk.ExchangeInfo
	var err error
	if a.c == client {
		data, err = GetExchangeInfo()
	} else {
		data, err = a.c.NewExchangeInfoService().Symbols(symbol).Do(ctx)
	}
	if err != nil {
		return out, err
	}
	for _, s := range data.Symbols {
		if s.Symbol != symbol {
			continue
		}
		out.Symbol = s.Symbol
		out.Base = s.BaseAsset
		out.Quote = s.QuoteAsset
		out.Trading = s.Status == "TRADING" && s.IsSpotTradingAllowed
		for _, t := range s.OrderTypes {
			if t == "MARKET" {
				out.MarketAllowed = true
			}
		}
		if x := s.LotSizeFilter(); x != nil {
			out.Step = x.StepSize
			out.MinQty = x.MinQuantity
			out.MaxQty = x.MaxQuantity
		}
		if x := s.MarketLotSizeFilter(); x != nil {
			if x.StepSize != "" && x.StepSize != "0" && x.StepSize != "0.00000000" {
				out.Step = x.StepSize
			}
			if x.MinQuantity != "" && x.MinQuantity != "0" {
				out.MinQty = x.MinQuantity
			}
			if x.MaxQuantity != "" && x.MaxQuantity != "0" {
				out.MaxQty = x.MaxQuantity
			}
		}
		if x := s.PriceFilter(); x != nil {
			out.Price = x.TickSize
		}
		if x := s.NotionalFilter(); x != nil {
			if x.ApplyMinToMarket {
				out.MinNotional = x.MinNotional
			}
			if x.ApplyMaxToMarket {
				out.MaxNotional = x.MaxNotional
			}
		}
		// v2.8.12 has no typed MIN_NOTIONAL helper: read official filter
		// fields without assuming NOTIONAL exists on every older pair.
		for _, raw := range s.Filters {
			if raw["filterType"] != "MIN_NOTIONAL" {
				continue
			}
			apply, ok := raw["applyToMarket"].(bool)
			if !ok || !apply {
				continue
			}
			if v, ok := raw["minNotional"].(string); ok && out.MinNotional == "" {
				out.MinNotional = v
			}
		}
		return out, nil
	}
	return out, fmt.Errorf("Spot exchange symbol %s not found", symbol)
}
func (a *QuantAdapter) Price(ctx context.Context, symbol string) (string, error) {
	if a.c == client {
		res, err := GetTickerPriceContext(binanceapiusage.WithSource(ctx, "spot_quant_price"), symbol)
		if err != nil {
			return "", err
		}
		for _, r := range res {
			if r.Symbol == symbol {
				return r.Price, nil
			}
		}
		return "", fmt.Errorf("Spot price %s missing", symbol)
	}
	p, err := a.c.NewListPricesService().Symbol(symbol).Do(binanceapiusage.WithSource(ctx, "spot_quant_price"))
	if err != nil {
		return "", err
	}
	for _, v := range p {
		if v.Symbol == symbol {
			return v.Price, nil
		}
	}
	return "", fmt.Errorf("Spot price %s missing", symbol)
}
func (a *QuantAdapter) Free(ctx context.Context, asset string) (string, error) {
	info, err := a.c.NewGetAccountService().Do(binanceapiusage.WithSource(ctx, "spot_quant_balance"))
	if err != nil {
		return "", err
	}
	if !info.CanTrade {
		return "", fmt.Errorf("Spot account trading permission is disabled")
	}
	for _, balance := range info.Balances {
		if balance.Asset == asset {
			return balance.Free, nil
		}
	}
	return "0", nil
}
func convertOrder(x *sdk.Order) spotexecution.RemoteOrder {
	return spotexecution.RemoteOrder{
		Symbol: x.Symbol, Side: string(x.Side), Type: string(x.Type),
		ClientOrderID: x.ClientOrderID, ExchangeOrderID: strconv.FormatInt(x.OrderID, 10),
		Status: string(x.Status), OrigQty: x.OrigQuantity, ExecutedQty: x.ExecutedQuantity,
	}
}
func (a *QuantAdapter) FindOrder(ctx context.Context, symbol, clientID string) (*spotexecution.RemoteOrder, error) {
	order, err := a.c.NewGetOrderService().Symbol(symbol).OrigClientOrderID(clientID).Do(binanceapiusage.WithSource(ctx, "spot_quant_lookup"))
	if err != nil {
		var apiErr *common.APIError
		if errors.As(err, &apiErr) && apiErr.Code == -2013 {
			return nil, nil
		}
		return nil, err
	}
	if order == nil {
		return nil, fmt.Errorf("nil Binance Spot order response")
	}
	m := convertOrder(order)
	return &m, nil
}
func (a *QuantAdapter) Place(ctx context.Context, req spotexecution.Request) (spotexecution.RemoteOrder, error) {
	var empty spotexecution.RemoteOrder
	if a == nil || a.c == nil || !a.allowOrderMutation || strings.TrimRight(a.c.BaseURL, "/") != "https://testnet.binance.vision" {
		return empty, spotexecution.ErrDisabled
	}
	if req.Type != "MARKET" || (req.Side != "BUY" && req.Side != "SELL") || req.Quantity == "" || len(req.ClientOrderID) > 36 {
		return empty, fmt.Errorf("invalid new Spot quantity order")
	}
	order, err := a.c.NewCreateOrderService().Symbol(req.Symbol).Side(sdk.SideType(req.Side)).
		Type(sdk.OrderTypeMarket).Quantity(req.Quantity).NewClientOrderID(req.ClientOrderID).
		NewOrderRespType(sdk.NewOrderRespTypeFULL).Do(binanceapiusage.WithSource(ctx, "spot_quant_place"))
	if err != nil {
		return empty, err
	}
	if order == nil {
		return empty, fmt.Errorf("nil Binance create-order response")
	}
	return spotexecution.RemoteOrder{
		Symbol: order.Symbol, Side: string(order.Side), Type: string(order.Type),
		ClientOrderID: order.ClientOrderID, ExchangeOrderID: strconv.FormatInt(order.OrderID, 10),
		Status: string(order.Status), OrigQty: order.OrigQuantity, ExecutedQty: order.ExecutedQuantity,
	}, nil
}
func (a *QuantAdapter) Cancel(ctx context.Context, symbol, clientID string) (spotexecution.RemoteOrder, error) {
	var empty spotexecution.RemoteOrder
	if a == nil || a.c == nil || !a.allowOrderMutation || strings.TrimRight(a.c.BaseURL, "/") != "https://testnet.binance.vision" {
		return empty, spotexecution.ErrDisabled
	}
	result, err := a.c.NewCancelOrderService().Symbol(symbol).OrigClientOrderID(clientID).Do(binanceapiusage.WithSource(ctx, "spot_quant_cancel"))
	if err != nil {
		return empty, err
	}
	if result == nil {
		return empty, fmt.Errorf("nil Binance Spot cancellation response")
	}
	return spotexecution.RemoteOrder{
		Symbol: result.Symbol, Side: string(result.Side), Type: string(result.Type),
		ClientOrderID: result.OrigClientOrderID, ExchangeOrderID: strconv.FormatInt(result.OrderID, 10),
		Status: string(result.Status), OrigQty: result.OrigQuantity, ExecutedQty: result.ExecutedQuantity,
	}, nil
}
func (a *QuantAdapter) Trades(ctx context.Context, symbol, orderID string) ([]spotexecution.Trade, error) {
	id, err := strconv.ParseInt(orderID, 10, 64)
	if err != nil || id <= 0 {
		return nil, fmt.Errorf("invalid Spot exchange order ID")
	}
	trades, err := a.c.NewListTradesService().Symbol(symbol).OrderId(id).Limit(1000).Do(binanceapiusage.WithSource(ctx, "spot_quant_my_trades"))
	if err != nil {
		return nil, err
	}
	res := make([]spotexecution.Trade, 0, len(trades))
	for _, v := range trades {
		if v == nil {
			return nil, fmt.Errorf("nil Binance trade response")
		}
		side := "SELL"
		if v.IsBuyer {
			side = "BUY"
		}
		fee := v.Commission
		if strings.TrimSpace(fee) == "" {
			return nil, fmt.Errorf("Binance commission missing")
		}
		res = append(res, spotexecution.Trade{
			Symbol: v.Symbol, Side: side, OrderID: strconv.FormatInt(v.OrderID, 10),
			TradeID: strconv.FormatInt(v.ID, 10), Qty: v.Quantity, Price: v.Price,
			QuoteQty: v.QuoteQuantity, Fee: fee, FeeAsset: v.CommissionAsset, Time: v.Time,
		})
	}
	return res, nil
}

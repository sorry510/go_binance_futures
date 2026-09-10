package futuresownership

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"go_binance_futures/feature/api/binance"
	"go_binance_futures/models"

	"github.com/adshao/go-binance/v2/futures"
)

type OrderRequest struct {
	Owner         string
	Symbol        string
	PositionSide  string
	Intent        string
	Side          string
	OrderType     string
	Quantity      float64
	Price         float64
	StopPrice     float64
	SourceRef     string
	ClientOrderID string
}

type ExchangeOrder struct {
	ExchangeOrderID string
	ClientOrderID   string
	Status          string
	FilledQty       float64
	AveragePrice    float64
}

type OrderBroker interface {
	Submit(context.Context, OrderRequest, string) (ExchangeOrder, error)
	Lookup(context.Context, string, string) (ExchangeOrder, error)
	Cancel(context.Context, string, int64) error
}

type Executor struct {
	Ownership Service
	Broker    OrderBroker
}

func DefaultExecutor() Executor {
	return Executor{Ownership: DefaultService(), Broker: BinanceOrderBroker{}}
}

func (e Executor) Execute(ctx context.Context, request OrderRequest) (ExchangeOrder, error) {
	if e.Broker == nil {
		return ExchangeOrder{}, fmt.Errorf("managed order broker is required")
	}
	clientID := strings.TrimSpace(request.ClientOrderID)
	if clientID == "" {
		var err error
		clientID, err = newOwnedClientOrderID(request.Owner)
		if err != nil {
			return ExchangeOrder{}, err
		}
	}
	request.ClientOrderID = clientID
	managed, err := e.Ownership.ClaimOrder(ctx, ClaimOrderInput{
		Owner: request.Owner, Symbol: request.Symbol, PositionSide: request.PositionSide,
		Intent: request.Intent, ClientOrderID: clientID, RequestedQty: request.Quantity,
		OrderType: request.OrderType, SourceRef: request.SourceRef,
	})
	if err != nil {
		return ExchangeOrder{}, err
	}
	if managed.Status != OrderPending {
		return e.Reconcile(ctx, managed.Symbol, managed.ClientOrderID)
	}
	result, submitErr := e.Broker.Submit(ctx, request, clientID)
	if submitErr != nil {
		lookup, lookupErr := e.Broker.Lookup(ctx, managed.Symbol, clientID)
		if lookupErr == nil && strings.TrimSpace(lookup.ExchangeOrderID) != "" {
			return e.applyExchange(ctx, clientID, lookup)
		}
		_ = e.Ownership.SetOrderStatus(ctx, clientID, OrderReconcile)
		return ExchangeOrder{}, fmt.Errorf("managed order submission is uncertain; reconcile client_order_id=%s before retry: %w", clientID, submitErr)
	}
	applied, err := e.applyExchange(ctx, clientID, result)
	if err != nil {
		return applied, err
	}
	if strings.EqualFold(request.OrderType, "MARKET") && applied.FilledQty <= qtyEpsilon {
		reconciled, reconcileErr := e.Reconcile(ctx, request.Symbol, clientID)
		if reconcileErr != nil {
			return reconciled, reconcileErr
		}
		if reconciled.FilledQty <= qtyEpsilon {
			_ = e.Ownership.SetOrderStatus(ctx, clientID, OrderReconcile)
			return reconciled, fmt.Errorf("market order %s is accepted but fill quantity is not confirmed; reconciliation is required", clientID)
		}
		return reconciled, nil
	}
	return applied, nil
}

func (e Executor) Reconcile(ctx context.Context, symbol, clientOrderID string) (ExchangeOrder, error) {
	if e.Broker == nil {
		return ExchangeOrder{}, fmt.Errorf("managed order broker is required")
	}
	result, err := e.Broker.Lookup(ctx, strings.ToUpper(strings.TrimSpace(symbol)), strings.TrimSpace(clientOrderID))
	if err != nil {
		_ = e.Ownership.SetOrderStatus(ctx, clientOrderID, OrderReconcile)
		return ExchangeOrder{}, err
	}
	return e.applyExchange(ctx, clientOrderID, result)
}

func (e Executor) applyExchange(ctx context.Context, clientOrderID string, result ExchangeOrder) (ExchangeOrder, error) {
	if strings.TrimSpace(result.ExchangeOrderID) != "" {
		if err := e.Ownership.MarkOrderSubmitted(ctx, clientOrderID, result.ExchangeOrderID); err != nil {
			return result, err
		}
	}
	if result.FilledQty > 0 {
		if _, err := e.Ownership.ApplyFill(ctx, clientOrderID, result.FilledQty, result.AveragePrice); err != nil {
			return result, err
		}
	}
	switch strings.ToUpper(strings.TrimSpace(result.Status)) {
	case "FILLED":
		if err := e.Ownership.SetOrderStatus(ctx, clientOrderID, OrderFilled); err != nil {
			return result, err
		}
	case "PARTIALLY_FILLED":
		if err := e.Ownership.SetOrderStatus(ctx, clientOrderID, OrderPartiallyFilled); err != nil {
			return result, err
		}
	case "CANCELED", "EXPIRED":
		if err := e.Ownership.SetOrderStatus(ctx, clientOrderID, OrderCanceled); err != nil {
			return result, err
		}
	case "REJECTED":
		if err := e.Ownership.SetOrderStatus(ctx, clientOrderID, OrderFailed); err != nil {
			return result, err
		}
	}
	return result, nil
}

func (e Executor) Cancel(ctx context.Context, owner string, order models.FuturesManagedOrder) error {
	expectedOwner, err := normalizeOwner(owner)
	if err != nil {
		return err
	}
	if order.Owner != expectedOwner {
		return fmt.Errorf("managed order %s belongs to owner %s, not %s", order.ClientOrderID, order.Owner, expectedOwner)
	}
	orderID, err := strconv.ParseInt(strings.TrimSpace(order.ExchangeOrderID), 10, 64)
	if err != nil || orderID <= 0 {
		return fmt.Errorf("managed order %s has no valid exchange order id", order.ClientOrderID)
	}
	if err := e.Broker.Cancel(ctx, order.Symbol, orderID); err != nil {
		return err
	}
	return e.Ownership.SetOrderStatus(ctx, order.ClientOrderID, OrderCanceled)
}

func newOwnedClientOrderID(owner string) (string, error) {
	owner, err := normalizeOwner(owner)
	if err != nil {
		return "", err
	}
	prefix := map[string]string{OwnerAutoStrategy: "aut", OwnerNewCoinRush: "rush", OwnerNoticeAutoOrder: "notice", OwnerFundingRate: "fund", OwnerAgentTrade: "agt"}[owner]
	buf := make([]byte, 10)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return prefix + "_" + hex.EncodeToString(buf), nil
}

type BinanceOrderBroker struct{}

func (BinanceOrderBroker) Submit(ctx context.Context, request OrderRequest, clientOrderID string) (ExchangeOrder, error) {
	side := futures.SideTypeBuy
	if strings.EqualFold(request.Side, "SELL") {
		side = futures.SideTypeSell
	}
	positionSide := futures.PositionSideTypeLong
	if strings.EqualFold(request.PositionSide, "SHORT") {
		positionSide = futures.PositionSideTypeShort
	}
	orderType := futures.OrderType(strings.ToUpper(strings.TrimSpace(request.OrderType)))
	switch orderType {
	case futures.OrderTypeMarket, futures.OrderTypeLimit, futures.OrderType("STOP_MARKET"), futures.OrderType("TAKE_PROFIT_MARKET"):
	default:
		return ExchangeOrder{}, fmt.Errorf("unsupported managed order type %q", request.OrderType)
	}
	order, err := binance.CreateOwnedOrder(ctx, binance.OwnedOrderParams{Symbol: request.Symbol, Quantity: request.Quantity, Price: request.Price, StopPrice: request.StopPrice, Side: side, PositionSide: positionSide, OrderType: orderType, ClientOrderID: clientOrderID})
	if err != nil {
		return ExchangeOrder{}, err
	}
	return exchangeFromCreate(order), nil
}

func (BinanceOrderBroker) Lookup(ctx context.Context, symbol, clientOrderID string) (ExchangeOrder, error) {
	order, err := binance.GetOrderByClientOrderID(ctx, symbol, clientOrderID)
	if err != nil {
		return ExchangeOrder{}, err
	}
	filled, _ := strconv.ParseFloat(order.ExecutedQuantity, 64)
	avg, _ := strconv.ParseFloat(order.AvgPrice, 64)
	return ExchangeOrder{ExchangeOrderID: strconv.FormatInt(order.OrderID, 10), ClientOrderID: order.ClientOrderID, Status: string(order.Status), FilledQty: filled, AveragePrice: avg}, nil
}

func (BinanceOrderBroker) Cancel(ctx context.Context, symbol string, orderID int64) error {
	_, err := binance.CancelOrder(symbol, orderID)
	return err
}

func exchangeFromCreate(order *futures.CreateOrderResponse) ExchangeOrder {
	if order == nil {
		return ExchangeOrder{}
	}
	filled, _ := strconv.ParseFloat(order.ExecutedQuantity, 64)
	avg, _ := strconv.ParseFloat(order.AvgPrice, 64)
	return ExchangeOrder{ExchangeOrderID: strconv.FormatInt(order.OrderID, 10), ClientOrderID: order.ClientOrderID, Status: string(order.Status), FilledQty: filled, AveragePrice: avg}
}

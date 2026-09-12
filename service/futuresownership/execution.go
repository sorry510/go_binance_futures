package futuresownership

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"go_binance_futures/feature/api/binance"
	"go_binance_futures/models"

	"github.com/adshao/go-binance/v2/common"
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
	Lookup(context.Context, string, string, string) (ExchangeOrder, error)
	Cancel(context.Context, string, int64, string) error
}

type Executor struct {
	Ownership Service
	Broker    OrderBroker
}

func DefaultExecutor() Executor {
	return Executor{Ownership: DefaultService(), Broker: BinanceOrderBroker{}}
}

// validateOrderRequest keeps direction and quantity semantics independent from
// every caller. Binance Hedge Mode always receives a positive quantity; Side
// determines whether the LONG/SHORT leg is increased or reduced.
func validateOrderRequest(request OrderRequest) error {
	if math.IsNaN(request.Quantity) || math.IsInf(request.Quantity, 0) || request.Quantity <= qtyEpsilon {
		return fmt.Errorf("managed order quantity must be a finite positive value")
	}
	positionSide, err := normalizePositionSide(request.PositionSide)
	if err != nil {
		return err
	}
	intent, err := normalizeIntent(request.Intent)
	if err != nil {
		return err
	}
	side := strings.ToUpper(strings.TrimSpace(request.Side))
	if side != string(futures.SideTypeBuy) && side != string(futures.SideTypeSell) {
		return fmt.Errorf("order side must be BUY or SELL")
	}
	expected := string(futures.SideTypeBuy)
	if intent == IntentOpen {
		if positionSide == "SHORT" {
			expected = string(futures.SideTypeSell)
		}
	} else {
		// Closing/protection orders must use the opposite side while keeping
		// the original positionSide in Hedge Mode.
		expected = string(futures.SideTypeSell)
		if positionSide == "SHORT" {
			expected = string(futures.SideTypeBuy)
		}
	}
	if side != expected {
		return fmt.Errorf("invalid managed order direction: intent=%s position_side=%s requires side=%s, got %s", intent, positionSide, expected, side)
	}
	return nil
}

func deterministicSubmitRejection(err error) bool {
	var apiErr *common.APIError
	if !errors.As(err, &apiErr) || apiErr == nil {
		return false
	}
	// These Binance codes explicitly mean the execution result is unknown.
	// Everything else carrying a concrete Binance API error code is a
	// deterministic rejection and is safe to mark failed/retry with a new ID.
	switch apiErr.Code {
	case -1000, -1001, -1006, -1007:
		return false
	default:
		return apiErr.Code != 0
	}
}

func (e Executor) Execute(ctx context.Context, request OrderRequest) (ExchangeOrder, error) {
	if err := ctx.Err(); err != nil {
		return ExchangeOrder{}, err
	}
	if err := validateOrderRequest(request); err != nil {
		return ExchangeOrder{}, err
	}
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
		if deterministicSubmitRejection(submitErr) {
			_ = e.Ownership.SetOrderStatus(ctx, clientID, OrderFailed)
			return ExchangeOrder{}, fmt.Errorf("managed order rejected by exchange: %w", submitErr)
		}
		lookup, lookupErr := e.Broker.Lookup(ctx, managed.Symbol, clientID, managed.OrderType)
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
	managed, loadErr := e.Ownership.GetOrder(ctx, clientOrderID)
	if loadErr != nil {
		return ExchangeOrder{}, loadErr
	}
	result, err := e.Broker.Lookup(ctx, strings.ToUpper(strings.TrimSpace(symbol)), strings.TrimSpace(clientOrderID), managed.OrderType)
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
	if err := e.Broker.Cancel(ctx, order.Symbol, orderID, order.OrderType); err != nil {
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

func isAlgoManagedOrderType(orderType string) bool {
	switch strings.ToUpper(strings.TrimSpace(orderType)) {
	case "STOP", "TAKE_PROFIT", "STOP_MARKET", "TAKE_PROFIT_MARKET", "TRAILING_STOP_MARKET":
		return true
	default:
		return false
	}
}

func (BinanceOrderBroker) Submit(ctx context.Context, request OrderRequest, clientOrderID string) (ExchangeOrder, error) {
	side := futures.SideTypeBuy
	if strings.EqualFold(request.Side, "SELL") {
		side = futures.SideTypeSell
	}
	positionSide := futures.PositionSideTypeLong
	if strings.EqualFold(request.PositionSide, "SHORT") {
		positionSide = futures.PositionSideTypeShort
	}
	orderType := strings.ToUpper(strings.TrimSpace(request.OrderType))
	if isAlgoManagedOrderType(orderType) {
		order, err := binance.CreateOwnedAlgoOrder(ctx, binance.OwnedOrderParams{
			Symbol: request.Symbol, Quantity: request.Quantity, Price: request.Price, StopPrice: request.StopPrice,
			Side: side, PositionSide: positionSide, OrderType: futures.OrderType(orderType), ClientOrderID: clientOrderID,
		})
		if err != nil {
			return ExchangeOrder{}, err
		}
		return exchangeFromAlgoCreate(order), nil
	}
	switch futures.OrderType(orderType) {
	case futures.OrderTypeMarket, futures.OrderTypeLimit:
	default:
		return ExchangeOrder{}, fmt.Errorf("unsupported managed order type %q", request.OrderType)
	}
	order, err := binance.CreateOwnedOrder(ctx, binance.OwnedOrderParams{Symbol: request.Symbol, Quantity: request.Quantity, Price: request.Price, StopPrice: request.StopPrice, Side: side, PositionSide: positionSide, OrderType: futures.OrderType(orderType), ClientOrderID: clientOrderID})
	if err != nil {
		return ExchangeOrder{}, err
	}
	return exchangeFromCreate(order), nil
}

func (BinanceOrderBroker) Lookup(ctx context.Context, symbol, clientOrderID, orderType string) (ExchangeOrder, error) {
	if isAlgoManagedOrderType(orderType) {
		algo, err := binance.GetAlgoOrderByClientOrderID(ctx, clientOrderID)
		if err != nil {
			return ExchangeOrder{}, err
		}
		return exchangeFromAlgoLookup(ctx, algo)
	}
	order, err := binance.GetOrderByClientOrderID(ctx, symbol, clientOrderID)
	if err != nil {
		return ExchangeOrder{}, err
	}
	return exchangeFromOrder(order), nil
}

func (BinanceOrderBroker) Cancel(ctx context.Context, symbol string, orderID int64, orderType string) error {
	if isAlgoManagedOrderType(orderType) {
		_, err := binance.CancelAlgoOrder(ctx, orderID)
		return err
	}
	_, err := binance.CancelOrder(symbol, orderID)
	return err
}

func exchangeFromAlgoCreate(order *futures.CreateAlgoOrderResp) ExchangeOrder {
	if order == nil {
		return ExchangeOrder{}
	}
	return ExchangeOrder{ExchangeOrderID: strconv.FormatInt(order.AlgoId, 10), ClientOrderID: order.ClientAlgoId, Status: string(order.AlgoStatus)}
}

func exchangeFromAlgoLookup(ctx context.Context, algo *futures.GetAlgoOrderResp) (ExchangeOrder, error) {
	if algo == nil {
		return ExchangeOrder{}, nil
	}
	result := ExchangeOrder{ExchangeOrderID: strconv.FormatInt(algo.AlgoId, 10), ClientOrderID: algo.ClientAlgoId, Status: string(algo.AlgoStatus)}
	if strings.TrimSpace(algo.ActualOrderId) == "" || strings.TrimSpace(algo.ActualOrderId) == "0" {
		return result, nil
	}
	actualID, err := strconv.ParseInt(strings.TrimSpace(algo.ActualOrderId), 10, 64)
	if err != nil {
		return result, fmt.Errorf("parse actual order id for algo %d: %w", algo.AlgoId, err)
	}
	actual, err := binance.GetOrderByOrderID(ctx, algo.Symbol, actualID)
	if err != nil {
		return result, err
	}
	actualResult := exchangeFromOrder(actual)
	result.Status = actualResult.Status
	result.FilledQty = actualResult.FilledQty
	result.AveragePrice = actualResult.AveragePrice
	return result, nil
}

func exchangeFromOrder(order *futures.Order) ExchangeOrder {
	if order == nil {
		return ExchangeOrder{}
	}
	filled, _ := strconv.ParseFloat(order.ExecutedQuantity, 64)
	avg, _ := strconv.ParseFloat(order.AvgPrice, 64)
	return ExchangeOrder{ExchangeOrderID: strconv.FormatInt(order.OrderID, 10), ClientOrderID: order.ClientOrderID, Status: string(order.Status), FilledQty: filled, AveragePrice: avg}
}

func exchangeFromCreate(order *futures.CreateOrderResponse) ExchangeOrder {
	if order == nil {
		return ExchangeOrder{}
	}
	filled, _ := strconv.ParseFloat(order.ExecutedQuantity, 64)
	avg, _ := strconv.ParseFloat(order.AvgPrice, 64)
	return ExchangeOrder{ExchangeOrderID: strconv.FormatInt(order.OrderID, 10), ClientOrderID: order.ClientOrderID, Status: string(order.Status), FilledQty: filled, AveragePrice: avg}
}

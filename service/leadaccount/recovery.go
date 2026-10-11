package leadaccount

import (
	"context"
	"errors"
	"fmt"
	"github.com/adshao/go-binance/v2/futures"
	"math"
	"strconv"
	"strings"

	"go_binance_futures/service/futuresownership"
)

// ErrLeadRecoveryUncertain denotes a lookup result which cannot safely prove
// a terminal order state. It must never be treated as permission to resubmit.
var ErrLeadRecoveryUncertain = errors.New("lead order requires authoritative reconciliation")

// RecoveryEvidence describes a single account-bound exchange lookup. It is
// diagnostic only: Stage 5 must separately reconcile durable Ownership state.
type RecoveryEvidence struct {
	Order             futuresownership.ExchangeOrder
	ActualOrderID     string // populated only when Algo triggered a verifiable real order
	Terminal          bool
	RequiresReconcile bool
}

// InspectOrderRecovery only reads the Lead-bound broker. It does not mutate
// orders, unlock pendingReconcile or infer fills from REST failures.
func (a *LeadExecutionAdapter) InspectOrderRecovery(ctx context.Context, symbol, clientOrderID, orderType string) (RecoveryEvidence, error) {
	if a == nil || a.account == nil || a.account.ID() != "lead" {
		return RecoveryEvidence{}, ErrLeadOpenBlocked
	}
	if err := ctx.Err(); err != nil {
		return RecoveryEvidence{}, err
	}
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	clientOrderID = strings.TrimSpace(clientOrderID)
	if symbol == "" || clientOrderID == "" {
		return RecoveryEvidence{RequiresReconcile: true}, ErrLeadRecoveryUncertain
	}
	if isLeadAlgoOrderType(orderType) {
		return a.inspectAlgoRecovery(ctx, symbol, clientOrderID, strings.ToUpper(strings.TrimSpace(orderType)))
	}
	switch orderType {
	case "MARKET", "LIMIT":
	default:
		return RecoveryEvidence{RequiresReconcile: true}, ErrLeadRecoveryUncertain
	}
	order, err := a.account.GetOrderByClientOrderID(ctx, symbol, clientOrderID)
	if err != nil {
		return RecoveryEvidence{RequiresReconcile: true}, fmt.Errorf("%w: ordinary order lookup failed", ErrLeadRecoveryUncertain)
	}
	if order == nil || string(order.Type) != orderType {
		return RecoveryEvidence{RequiresReconcile: true}, ErrLeadRecoveryUncertain
	}
	exchange, err := verifiedLeadOrder(order, symbol, clientOrderID)
	if err != nil {
		return RecoveryEvidence{RequiresReconcile: true}, err
	}
	return ClassifyLeadRecovery(exchange, clientOrderID)
}

func isLeadAlgoOrderType(orderType string) bool {
	switch strings.ToUpper(strings.TrimSpace(orderType)) {
	case "STOP", "TAKE_PROFIT", "STOP_MARKET", "TAKE_PROFIT_MARKET", "TRAILING_STOP_MARKET":
		return true
	}
	return false
}

// verifiedLeadOrder rejects malformed quantities rather than silently treating
// an SDK ParseFloat failure as zero completed fills.
func verifiedLeadOrder(order *futures.Order, symbol, clientID string) (futuresownership.ExchangeOrder, error) {
	if order == nil || order.OrderID <= 0 || order.Symbol != symbol || order.ClientOrderID != clientID {
		return futuresownership.ExchangeOrder{}, ErrLeadRecoveryUncertain
	}
	filled, e1 := strconv.ParseFloat(order.ExecutedQuantity, 64)
	avg, e2 := strconv.ParseFloat(order.AvgPrice, 64)
	if e1 != nil || e2 != nil {
		return futuresownership.ExchangeOrder{}, ErrLeadRecoveryUncertain
	}
	return futuresownership.ExchangeOrder{
		ExchangeOrderID: strconv.FormatInt(order.OrderID, 10), ClientOrderID: clientID,
		Status: string(order.Status), FilledQty: filled, AveragePrice: avg,
	}, nil
}

// Algo receipt alone is insufficient to prove a trigger's fill. When the
// exchange lists an actual order, verify its ID, symbol and direction before
// using that order's cumulative filled quantity.
func (a *LeadExecutionAdapter) inspectAlgoRecovery(ctx context.Context, symbol, clientID, orderType string) (RecoveryEvidence, error) {
	uncertain := RecoveryEvidence{RequiresReconcile: true}
	algo, err := a.account.GetAlgoOrderByClientOrderID(ctx, clientID)
	if err != nil {
		return uncertain, fmt.Errorf("%w: Algo lookup failed", ErrLeadRecoveryUncertain)
	}
	if algo == nil || algo.AlgoId <= 0 || algo.ClientAlgoId != clientID || algo.Symbol != symbol || string(algo.OrderType) != orderType ||
		(algo.PositionSide != futures.PositionSideTypeLong && algo.PositionSide != futures.PositionSideTypeShort) ||
		(algo.Side != futures.SideTypeBuy && algo.Side != futures.SideTypeSell) {
		return uncertain, ErrLeadRecoveryUncertain
	}
	algoID := strconv.FormatInt(algo.AlgoId, 10)
	actualID := strings.TrimSpace(algo.ActualOrderId)
	if actualID == "" || actualID == "0" {
		// FINISHED without an actual order cannot prove any execution state.
		// Only explicitly canceled/expired Algo receipts are terminal evidence.
		evidence, classifyErr := ClassifyLeadRecovery(futuresownership.ExchangeOrder{
			ExchangeOrderID: algoID, ClientOrderID: clientID, Status: string(algo.AlgoStatus),
		}, clientID)
		return evidence, classifyErr
	}
	orderID, parseErr := strconv.ParseInt(actualID, 10, 64)
	if parseErr != nil || orderID <= 0 {
		return uncertain, ErrLeadRecoveryUncertain
	}
	realOrder, lookupErr := a.account.GetOrderByOrderID(ctx, symbol, orderID)
	if lookupErr != nil {
		return uncertain, fmt.Errorf("%w: triggered order lookup failed", ErrLeadRecoveryUncertain)
	}
	if realOrder == nil || realOrder.OrderID != orderID || realOrder.Symbol != symbol ||
		realOrder.Side != algo.Side || realOrder.PositionSide != algo.PositionSide {
		return uncertain, ErrLeadRecoveryUncertain
	}
	exchange, verifyErr := verifiedLeadOrder(realOrder, symbol, realOrder.ClientOrderID)
	if verifyErr != nil {
		return uncertain, verifyErr
	}
	// Preserve the *Algo* client ID and order ID for the ownership claim.
	exchange.ClientOrderID = clientID
	exchange.ExchangeOrderID = algoID
	evidence, classifyErr := ClassifyLeadRecovery(exchange, clientID)
	evidence.ActualOrderID = actualID
	return evidence, classifyErr
}

// ClassifyLeadRecovery is shared by ordinary and Algo STOP/TP evidence.
// An Algo with no actual order yet is pending even if its trigger was accepted.
func ClassifyLeadRecovery(order futuresownership.ExchangeOrder, clientOrderID string) (RecoveryEvidence, error) {
	evidence := RecoveryEvidence{Order: order, RequiresReconcile: true}
	if strings.TrimSpace(order.ExchangeOrderID) == "" || strings.TrimSpace(order.ClientOrderID) != strings.TrimSpace(clientOrderID) ||
		math.IsNaN(order.FilledQty) || math.IsInf(order.FilledQty, 0) || order.FilledQty < 0 ||
		math.IsNaN(order.AveragePrice) || math.IsInf(order.AveragePrice, 0) || order.AveragePrice < 0 ||
		(order.FilledQty > 0 && order.AveragePrice <= 0) {
		return evidence, ErrLeadRecoveryUncertain
	}
	status := strings.ToUpper(strings.TrimSpace(order.Status))
	// Binance terminal FILLED always has cumulative quantity; a rejected
	// order cannot legitimately contain a completed fill. Do not trust an
	// inconsistent HTTP response enough to classify it as authoritative.
	if (status == "FILLED" && order.FilledQty <= 0) ||
		(status == "REJECTED" && order.FilledQty != 0) ||
		(status == "PARTIALLY_FILLED" && order.FilledQty <= 0) ||
		(status == "NEW" && order.FilledQty != 0) {
		return evidence, ErrLeadRecoveryUncertain
	}
	switch status {
	case "FILLED", "CANCELED", "EXPIRED", "REJECTED", "EXPIRED_IN_MATCH", "CANCELLED":
		// Even a terminal exchange status needs Stage 5 ownership/position reconciliation.
		evidence.Terminal = true
	case "NEW", "PARTIALLY_FILLED", "PENDING_NEW", "PENDING_CANCEL", "TRIGGERED", "NOT_TRIGGERED", "WORKING":
	default:
		return evidence, ErrLeadRecoveryUncertain
	}
	return evidence, nil
}

package agenttrade

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	binanceapi "go_binance_futures/feature/api/binance"
	"go_binance_futures/models"
	futuresownership "go_binance_futures/service/futuresownership"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/client/orm"
)

type OwnershipLifecycle struct {
	Ownership  futuresownership.Service
	Executor   futuresownership.Executor
	AccountQty func(context.Context, string, string) (float64, error)
}

func DefaultOwnershipLifecycle() OwnershipLifecycle {
	return OwnershipLifecycle{Ownership: futuresownership.DefaultService(), Executor: futuresownership.DefaultExecutor()}
}

func (l OwnershipLifecycle) ownership() futuresownership.Service {
	return l.Ownership
}

func (l OwnershipLifecycle) executor() futuresownership.Executor {
	if l.Executor.Broker == nil {
		return futuresownership.DefaultExecutor()
	}
	return l.Executor
}

func (l OwnershipLifecycle) EnsureProtection(ctx context.Context, proposal models.AgentTradeProposal) (ProtectionResult, error) {
	position, err := l.ownership().GetPosition(ctx, futuresownership.OwnerAgentTrade, proposal.Symbol, proposal.Side)
	if err == orm.ErrNoRows {
		return ProtectionResult{}, ErrManagedPositionClosed
	}
	if err != nil {
		return ProtectionResult{}, fmt.Errorf("load agent managed position: %w", err)
	}
	if position.ManagedQty <= 0 || proposal.StopLoss <= 0 {
		return ProtectionResult{}, fmt.Errorf("managed quantity and stop loss must be positive")
	}
	side := closeSide(proposal.Side)
	stopClientID, err := l.nextProtectionClientOrderID(ctx, proposal, futuresownership.IntentStopLoss, "sl", position.ManagedQty)
	if err != nil {
		return ProtectionResult{}, err
	}
	result := ProtectionResult{StopClientOrderID: stopClientID}
	stop, err := l.submitProtection(ctx, proposal, position.ManagedQty, proposal.StopLoss, futuresownership.IntentStopLoss, "STOP_MARKET", side, result.StopClientOrderID)
	if err != nil {
		return result, fmt.Errorf("create protective stop: %w", err)
	}
	result.StopExchangeOrderID = stop.ExchangeOrderID

	var targets []float64
	if err := json.Unmarshal([]byte(proposal.TakeProfitsJSON), &targets); err == nil && len(targets) > 0 && targets[0] > 0 {
		tpClientID, idErr := l.nextProtectionClientOrderID(ctx, proposal, futuresownership.IntentTakeProfit, "tp", position.ManagedQty)
		if idErr != nil {
			result.TakeProfitError = idErr.Error()
			return result, nil
		}
		result.TakeProfitClientOrderID = tpClientID
		tp, tpErr := l.submitProtection(ctx, proposal, position.ManagedQty, targets[0], futuresownership.IntentTakeProfit, "TAKE_PROFIT_MARKET", side, result.TakeProfitClientOrderID)
		if tpErr != nil {
			result.TakeProfitError = tpErr.Error()
		} else {
			result.TakeProfitExchangeOrderID = tp.ExchangeOrderID
		}
	}
	return result, nil
}

func (l OwnershipLifecycle) submitProtection(ctx context.Context, proposal models.AgentTradeProposal, qty, stopPrice float64, intent, orderType, side, clientID string) (futuresownership.ExchangeOrder, error) {
	request := futuresownership.OrderRequest{
		Owner: futuresownership.OwnerAgentTrade, Symbol: proposal.Symbol, PositionSide: proposal.Side,
		Intent: intent, Side: side, OrderType: orderType, Quantity: qty, StopPrice: stopPrice,
		SourceRef: proposal.ProposalID, ClientOrderID: clientID,
	}
	return l.executor().Execute(ctx, request)
}

func (l OwnershipLifecycle) nextProtectionClientOrderID(ctx context.Context, proposal models.AgentTradeProposal, intent, kind string, quantity float64) (string, error) {
	orders, err := l.ownership().ListOrders(ctx, futuresownership.OwnerAgentTrade, 500)
	if err != nil {
		return "", err
	}
	attempts := 0
	for _, managed := range orders {
		if managed.SourceRef != proposal.ProposalID || managed.Intent != intent {
			continue
		}
		attempts++
		if managed.Status == futuresownership.OrderFilled {
			// A terminal protective order must never be treated as active protection.
			// If EnsureProtection is running, a managed position still exists, so a
			// filled/terminal protection needs a fresh attempt for the remainder.
			continue
		}
		if managed.Status == futuresownership.OrderPending || managed.Status == futuresownership.OrderSubmitted || managed.Status == futuresownership.OrderPartiallyFilled || managed.Status == futuresownership.OrderReconcile {
			if math.Abs(managed.RequestedQty-quantity) <= 1e-12 {
				return managed.ClientOrderID, nil
			}
			if strings.TrimSpace(managed.ExchangeOrderID) == "" {
				return "", fmt.Errorf("cannot resize unresolved %s protection %s from %.12f to %.12f", intent, managed.ClientOrderID, managed.RequestedQty, quantity)
			}
			if err := l.executor().Cancel(ctx, futuresownership.OwnerAgentTrade, managed); err != nil {
				return "", fmt.Errorf("cancel stale %s protection %s before resize: %w", intent, managed.ClientOrderID, err)
			}
		}
	}
	return lifecycleAttemptClientOrderID(kind, proposal.ProposalID, attempts+1), nil
}

func (l OwnershipLifecycle) Close(ctx context.Context, proposal models.AgentTradeProposal) (CloseResult, error) {
	position, err := l.ownership().GetPosition(ctx, futuresownership.OwnerAgentTrade, proposal.Symbol, proposal.Side)
	if err == orm.ErrNoRows {
		if cleanupErr := l.cancelProtection(ctx, proposal.ProposalID, ""); cleanupErr != nil {
			return CloseResult{AlreadyClosed: true}, fmt.Errorf("managed position is closed but protection cleanup failed: %w", cleanupErr)
		}
		return CloseResult{AlreadyClosed: true}, nil
	}
	if err != nil {
		return CloseResult{}, err
	}
	if err := l.cancelProtection(ctx, proposal.ProposalID, futuresownership.IntentTakeProfit); err != nil {
		return CloseResult{}, fmt.Errorf("cancel take profit before close: %w", err)
	}
	accountQtyFn := l.AccountQty
	if accountQtyFn == nil {
		accountQtyFn = currentAgentAccountQty
	}
	accountQty, err := accountQtyFn(ctx, proposal.Symbol, proposal.Side)
	if err != nil {
		return CloseResult{}, err
	}
	closeQty := math.Min(position.ManagedQty, math.Abs(accountQty))
	if closeQty <= 1e-12 {
		_, _ = l.ownership().ReconcilePosition(ctx, futuresownership.OwnerAgentTrade, proposal.Symbol, proposal.Side, 0)
		_ = l.cancelProtection(ctx, proposal.ProposalID, "")
		return CloseResult{AlreadyClosed: true}, nil
	}
	clientID, err := l.nextCloseClientOrderID(ctx, proposal)
	if err != nil {
		return CloseResult{}, err
	}
	// Reconciliation above may have completed an earlier close attempt. Reload
	// ownership before calculating the next mutation quantity.
	position, err = l.ownership().GetPosition(ctx, futuresownership.OwnerAgentTrade, proposal.Symbol, proposal.Side)
	if err == orm.ErrNoRows {
		_ = l.cancelProtection(ctx, proposal.ProposalID, "")
		return CloseResult{AlreadyClosed: true}, nil
	}
	if err != nil {
		return CloseResult{}, err
	}
	accountQty, err = accountQtyFn(ctx, proposal.Symbol, proposal.Side)
	if err != nil {
		return CloseResult{}, err
	}
	closeQty = math.Min(position.ManagedQty, math.Abs(accountQty))
	if closeQty <= 1e-12 {
		_, _ = l.ownership().ReconcilePosition(ctx, futuresownership.OwnerAgentTrade, proposal.Symbol, proposal.Side, 0)
		_ = l.cancelProtection(ctx, proposal.ProposalID, "")
		return CloseResult{AlreadyClosed: true}, nil
	}
	order, err := l.executor().Execute(ctx, futuresownership.OrderRequest{
		Owner: futuresownership.OwnerAgentTrade, Symbol: proposal.Symbol, PositionSide: proposal.Side,
		Intent: futuresownership.IntentClose, Side: closeSide(proposal.Side), OrderType: string(futures.OrderTypeMarket),
		Quantity: closeQty, SourceRef: proposal.ProposalID, ClientOrderID: clientID,
	})
	if err != nil {
		return CloseResult{ClientOrderID: clientID}, err
	}
	remaining, positionErr := l.ownership().GetPosition(ctx, futuresownership.OwnerAgentTrade, proposal.Symbol, proposal.Side)
	if positionErr != orm.ErrNoRows && positionErr != nil {
		return CloseResult{}, positionErr
	}
	if positionErr == nil && remaining.ManagedQty > 1e-12 {
		return CloseResult{ClientOrderID: clientID, ExchangeOrderID: order.ExchangeOrderID, FilledQty: order.FilledQty}, fmt.Errorf("managed close is only partially filled; %.12f remains", remaining.ManagedQty)
	}
	if err := l.cancelProtection(ctx, proposal.ProposalID, ""); err != nil {
		return CloseResult{ClientOrderID: clientID, ExchangeOrderID: order.ExchangeOrderID, FilledQty: order.FilledQty}, fmt.Errorf("position closed but protection cleanup failed: %w", err)
	}
	return CloseResult{ClientOrderID: clientID, ExchangeOrderID: order.ExchangeOrderID, FilledQty: order.FilledQty}, nil
}

func (l OwnershipLifecycle) nextCloseClientOrderID(ctx context.Context, proposal models.AgentTradeProposal) (string, error) {
	orders, err := l.ownership().ListOrders(ctx, futuresownership.OwnerAgentTrade, 500)
	if err != nil {
		return "", err
	}
	attempts := 0
	for _, managed := range orders {
		if managed.SourceRef != proposal.ProposalID || managed.Intent != futuresownership.IntentClose {
			continue
		}
		attempts++
		switch managed.Status {
		case futuresownership.OrderPending, futuresownership.OrderSubmitted, futuresownership.OrderPartiallyFilled, futuresownership.OrderReconcile:
			reconciled, reconcileErr := l.executor().Reconcile(ctx, managed.Symbol, managed.ClientOrderID)
			if reconcileErr != nil {
				return "", fmt.Errorf("reconcile existing managed close %s before retry: %w", managed.ClientOrderID, reconcileErr)
			}
			switch strings.ToUpper(strings.TrimSpace(reconciled.Status)) {
			case "FILLED", "CANCELED", "EXPIRED", "REJECTED":
				// Terminal: a remaining managed quantity may use a new attempt ID.
			default:
				return "", fmt.Errorf("managed close %s is still active with exchange status %s", managed.ClientOrderID, reconciled.Status)
			}
		}
	}
	return lifecycleAttemptClientOrderID("close", proposal.ProposalID, attempts+1), nil
}

func lifecycleAttemptClientOrderID(kind, proposalID string, attempt int) string {
	if attempt < 1 {
		attempt = 1
	}
	suffix := fmt.Sprintf("_%d", attempt)
	base := lifecycleClientOrderID(kind, proposalID)
	if len(base)+len(suffix) > 36 {
		base = base[:36-len(suffix)]
	}
	return base + suffix
}

func (l OwnershipLifecycle) cancelProtection(ctx context.Context, proposalID, onlyIntent string) error {
	orders, err := l.ownership().ListOrders(ctx, futuresownership.OwnerAgentTrade, 500)
	if err != nil {
		return err
	}
	for _, order := range orders {
		if order.SourceRef != proposalID || (order.Intent != futuresownership.IntentStopLoss && order.Intent != futuresownership.IntentTakeProfit) {
			continue
		}
		if onlyIntent != "" && order.Intent != onlyIntent {
			continue
		}
		if order.Status != futuresownership.OrderPending && order.Status != futuresownership.OrderSubmitted && order.Status != futuresownership.OrderPartiallyFilled && order.Status != futuresownership.OrderReconcile {
			continue
		}
		if err := l.executor().Cancel(ctx, futuresownership.OwnerAgentTrade, order); err != nil {
			return err
		}
	}
	return nil
}

func currentAgentAccountQty(ctx context.Context, symbol, side string) (float64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	rows, err := binanceapi.GetPosition(binanceapi.PositionParams{Symbol: strings.ToUpper(strings.TrimSpace(symbol))})
	if err != nil {
		return 0, fmt.Errorf("load Binance position before agent close: %w", err)
	}
	for _, row := range rows {
		if !strings.EqualFold(string(row.PositionSide), side) {
			continue
		}
		qty, parseErr := strconv.ParseFloat(row.PositionAmt, 64)
		if parseErr != nil {
			return 0, parseErr
		}
		return math.Abs(qty), nil
	}
	return 0, nil
}

func closeSide(positionSide string) string {
	if strings.EqualFold(positionSide, "SHORT") {
		return string(futures.SideTypeBuy)
	}
	return string(futures.SideTypeSell)
}

func lifecycleClientOrderID(kind, proposalID string) string {
	value := "agt_" + kind + "_" + strings.TrimPrefix(strings.TrimSpace(proposalID), "tp_")
	if len(value) > 36 {
		value = value[:36]
	}
	return value
}

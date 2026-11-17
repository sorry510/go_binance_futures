package leadaccount

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	binance "go_binance_futures/feature/api/binance"
	"go_binance_futures/service/futuresownership"
)

// ExecuteManagedClose handles only verified Lead-owned exits. The regular
// new-entry switch does not block exits, but broken Lead identity/WS or
// uncertain order reconciliation does.
func (a *LeadExecutionAdapter) ExecuteManagedClose(ctx context.Context, request futuresownership.OrderRequest) (futuresownership.ExchangeOrder, error) {
	if a == nil || a.risk == nil || a.source == nil || a.account == nil || a.account.ID() != binance.LeadAccountID {
		return futuresownership.ExchangeOrder{}, ErrLeadOpenBlocked
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return futuresownership.ExchangeOrder{}, err
	}
	if a.pendingReconcile {
		return futuresownership.ExchangeOrder{}, ErrLeadPendingReconcile
	}
	if a.simulateEvidence == nil && !leadTradeSessionHealthyFor(a.account) {
		return futuresownership.ExchangeOrder{}, ErrLeadOpenBlocked
	}
	if a.guard != nil && !a.guard.permitsExitWrite() {
		return futuresownership.ExchangeOrder{}, ErrLeadOpenBlocked
	}
	a.risk.mu.RLock()
	state := a.risk.state
	a.risk.mu.RUnlock()
	if !state.Stage7Authorized || !state.PortfolioBound || !state.ReadOnlyVerified || !state.WSHealthy {
		return futuresownership.ExchangeOrder{}, ErrLeadOpenBlocked
	}
	if request.Owner != futuresownership.OwnerAutoStrategy ||
		(request.Intent != futuresownership.IntentClose && request.Intent != futuresownership.IntentStopLoss && request.Intent != futuresownership.IntentTakeProfit) ||
		(request.OrderType != "MARKET" && request.OrderType != "LIMIT" && request.OrderType != "STOP_MARKET" && request.OrderType != "TAKE_PROFIT_MARKET") {
		return futuresownership.ExchangeOrder{}, ErrLeadOpenBlocked
	}
	positionSide := strings.ToUpper(strings.TrimSpace(request.PositionSide))
	if positionSide != "LONG" && positionSide != "SHORT" {
		return futuresownership.ExchangeOrder{}, ErrLeadOpenBlocked
	}
	if (positionSide == "LONG" && request.Side != "SELL") || (positionSide == "SHORT" && request.Side != "BUY") ||
		!riskFinitePositive(request.Quantity) {
		return futuresownership.ExchangeOrder{}, ErrLeadOpenBlocked
	}
	owned, err := a.ownership.Ownership.GetPosition(ctx, futuresownership.OwnerAutoStrategy, request.Symbol, positionSide)
	if err != nil || owned.AccountID != string(binance.LeadAccountID) || !riskFinitePositive(owned.ManagedQty) {
		return futuresownership.ExchangeOrder{}, ErrLeadOpenBlocked
	}
	// Re-read the actual Lead account leg; do not use main, cached or self-reported
	// quantity. Manual adds may never be included as managed amount; manual
	// reductions constrain every attempted close quantity.
	live, err := a.source.Reader.GetPositionFreshContext(ctx, binance.PositionParams{Symbol: request.Symbol})
	if err != nil {
		return futuresownership.ExchangeOrder{}, fmt.Errorf("Lead managed close live position unavailable")
	}
	liveQty := 0.0
	markPrice := 0.0
	for _, p := range live {
		if p == nil || !strings.EqualFold(p.Symbol, request.Symbol) || strings.ToUpper(p.PositionSide) != positionSide {
			continue
		}
		quantity, parseErr := strconv.ParseFloat(p.PositionAmt, 64)
		if parseErr != nil || math.IsInf(quantity, 0) || math.IsNaN(quantity) {
			return futuresownership.ExchangeOrder{}, ErrLeadOpenBlocked
		}
		mark, markErr := strconv.ParseFloat(p.MarkPrice, 64)
		if markErr != nil || !riskFinitePositive(mark) {
			return futuresownership.ExchangeOrder{}, ErrLeadOpenBlocked
		}
		if markPrice != 0 && markPrice != mark {
			return futuresownership.ExchangeOrder{}, ErrLeadOpenBlocked
		}
		markPrice = mark
		liveQty += math.Abs(quantity)
	}
	if liveQty <= 0 || request.Quantity > math.Min(owned.ManagedQty, liveQty)+1e-10 {
		return futuresownership.ExchangeOrder{}, ErrLeadOpenBlocked
	}
	if err := a.verifyCloseRules(ctx, request, markPrice, math.Min(owned.ManagedQty, liveQty), liveQty); err != nil {
		return futuresownership.ExchangeOrder{}, fmt.Errorf("%w: %v", ErrLeadOpenBlocked, err)
	}
	// An acknowledged TP/SL or close request must not block unrelated
	// orders. Only submission/ledger uncertainty triggers recovery.
	executed, execErr := a.ownership.Execute(ctx, request)
	if execErr != nil {
		a.pendingReconcile = true
		if a.guard != nil {
			a.guard.RecordFault(binance.LeadAccountID, FaultPendingReconcile)
		}
	}
	return executed, execErr
}

package leadaccount

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	binance "go_binance_futures/feature/api/binance"
	"go_binance_futures/models"
	"go_binance_futures/service/futuresownership"
)

// CancelExpiredStrategyOpens implements the same timeout policy as Main:
// only managed, outstanding OPEN orders with a remaining quantity are
// eligible. The existing executor owns both exchange cancel and local state.
// Pausing NEW opens must not prevent maintaining existing orders.
func (a *LeadExecutionAdapter) CancelExpiredStrategyOpens(ctx context.Context, timeoutSeconds int64) error {
	if a == nil || a.account == nil || a.account.ID() != binance.LeadAccountID || a.risk == nil ||
		timeoutSeconds <= 0 {
		return ErrLeadOpenBlocked
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	a.risk.mu.RLock()
	state := a.risk.state
	a.risk.mu.RUnlock()
	if !state.Stage7Authorized || !state.PortfolioBound || !state.ReadOnlyVerified || !state.WSHealthy ||
		(a.guard != nil && !a.guard.permitsExitWrite()) {
		return ErrLeadOpenBlocked
	}
	orders, err := a.ownership.Ownership.ActiveOrders(ctx, futuresownership.OwnerAutoStrategy)
	if err != nil {
		return err
	}
	now := time.Now().UnixMilli()
	for _, current := range orders {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !leadExpiredOpeningOrder(current, now, timeoutSeconds) {
			continue
		}
		if strings.TrimSpace(current.ExchangeOrderID) == "" {
			if _, err := a.ownership.Reconcile(ctx, current.Symbol, current.ClientOrderID); err != nil {
				return fmt.Errorf("Lead unknown open order %s requires lookup: %w", current.ClientOrderID, err)
			}
			current, err = a.ownership.Ownership.GetOrder(ctx, current.ClientOrderID)
			if err != nil {
				return err
			}
			if !leadExpiredOpeningOrder(current, now, timeoutSeconds) ||
				strings.TrimSpace(current.ExchangeOrderID) == "" {
				continue
			}
		}
		if err := a.ownership.Cancel(ctx, futuresownership.OwnerAutoStrategy, current); err != nil {
			// Exchange cancellation is not proof the order did not fill.
			// Query the original clientOrderId before any further action.
			if _, lookupErr := a.ownership.Reconcile(ctx, current.Symbol, current.ClientOrderID); lookupErr != nil {
				return errors.Join(err, lookupErr)
			}
		}
	}
	return nil
}

func leadExpiredOpeningOrder(o models.FuturesManagedOrder, nowMillis, timeoutSeconds int64) bool {
	if o.AccountID != string(binance.LeadAccountID) ||
		o.Owner != futuresownership.OwnerAutoStrategy ||
		o.Intent != futuresownership.IntentOpen || timeoutSeconds <= 0 ||
		nowMillis < o.CreatedAt+timeoutSeconds*1000 ||
		o.RequestedQty <= 0 || o.RequestedQty-o.FilledQty <= 1e-8 {
		return false
	}
	switch o.Status {
	case futuresownership.OrderPending, futuresownership.OrderSubmitted,
		futuresownership.OrderPartiallyFilled, futuresownership.OrderReconcile:
		return true
	default:
		return false
	}
}

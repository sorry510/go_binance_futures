package leadaccount

import (
	"context"
	"errors"
	"fmt"

	"go_binance_futures/feature/api/binance"
	"go_binance_futures/service/futuresownership"
)

// ErrLiveExecutionLocked is a deliberate Stage 4-3/7 fail-closed boundary.
// Neither ordinary config flags nor a successful read-only verification can
// authorize mutations. Live authorization must be introduced separately.
var ErrLiveExecutionLocked = errors.New("lead live execution not authorized")

// ReadOnlyOrderBroker allows recovery/reconciliation lookups for the explicitly
// bound Lead key but never sends an order or cancellation to Binance.
// All write routes, including algo TP/SL and cancellation, fail closed.
type ReadOnlyOrderBroker struct {
	account *binance.AccountClient
}

func NewReadOnlyOrderBroker(account *binance.AccountClient) (*ReadOnlyOrderBroker, error) {
	if account == nil || account.ID() != binance.LeadAccountID {
		return nil, fmt.Errorf("read-only lead broker requires a lead account")
	}
	return &ReadOnlyOrderBroker{account: account}, nil
}

func (b *ReadOnlyOrderBroker) Submit(ctx context.Context, _ futuresownership.OrderRequest, _ string) (futuresownership.ExchangeOrder, error) {
	if err := ctx.Err(); err != nil {
		return futuresownership.ExchangeOrder{}, err
	}
	return futuresownership.ExchangeOrder{}, ErrLiveExecutionLocked
}

func (b *ReadOnlyOrderBroker) Cancel(ctx context.Context, _ string, _ int64, _ string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return ErrLiveExecutionLocked
}

func (b *ReadOnlyOrderBroker) Lookup(ctx context.Context, symbol, clientID, orderType string) (futuresownership.ExchangeOrder, error) {
	if err := ctx.Err(); err != nil {
		return futuresownership.ExchangeOrder{}, err
	}
	if b == nil || b.account == nil || b.account.ID() != binance.LeadAccountID {
		return futuresownership.ExchangeOrder{}, fmt.Errorf("lead account broker is unavailable")
	}
	return (futuresownership.BinanceOrderBroker{Account: b.account}).Lookup(ctx, symbol, clientID, orderType)
}

// NewReadOnlyExecutor binds the lead-specific ownership ledger and broker.
// It intentionally cannot execute a live order, even when a caller invokes
// Executor.Execute directly; production activation remains gated by Stage 7.
func NewReadOnlyExecutor(account *binance.AccountClient) (futuresownership.Executor, error) {
	broker, err := NewReadOnlyOrderBroker(account)
	if err != nil {
		return futuresownership.Executor{}, err
	}
	ownership, err := futuresownership.BindAccount(binance.LeadAccountID)
	if err != nil {
		return futuresownership.Executor{}, err
	}
	return futuresownership.Executor{Ownership: ownership, Broker: broker}, nil
}

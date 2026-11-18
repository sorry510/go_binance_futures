package coinmexecution

import (
	"context"
	"errors"
	"testing"

	"go_binance_futures/feature/api/binance"
)

// O2: a full userTrades result is ambiguous (there may be later pages).
// It must never be interpreted as complete, nor left in a "FILLED" state
// without confirmed local Fills.
func TestCM5FullUserTradesPageLeavesExplicitAwaitingFills(t *testing.T) {
	db := cm5Fixture(t)
	m := newMockCM5()
	p := NewProcessor(db, m)
	ctx := context.Background()
	local, err := p.Open(ctx, OpenRequest{Symbol: "BTCUSD_PERP", PositionSide: "LONG", ClientOrderID: "cm5fullpageguard", Type: "MARKET"})
	if err != nil {
		t.Fatal(err)
	}
	ack := m.orders[local.ClientOrderID]
	ack.Status = "FILLED"
	ack.FilledContracts = "10"
	m.trades[ack.ExchangeOrderID] = make([]binance.CoinMTrade, 1000)
	got, err := p.Reconcile(ctx, local.ClientOrderID)
	if !errors.Is(err, ErrUncertain) {
		t.Fatalf("1000-trade page must block settlement: %+v %v", got, err)
	}
	if got.Status != "AWAITING_FILLS" {
		t.Fatalf("incomplete page incorrectly recorded as settled: %+v", got)
	}
	persisted, err := p.FindOrder(local.ClientOrderID)
	if err != nil || persisted.Status != "AWAITING_FILLS" {
		t.Fatalf("incomplete page status not durable: %+v %v", persisted, err)
	}
	expectCM5(t, db, "SELECT COUNT(*) FROM coin_m_trade_fills", 0)
	// Reconcile can retry the original ID, but must never POST another order.
	if len(m.calls) != 1 {
		t.Fatalf("ambiguous page posted a duplicate: %+v", m.calls)
	}
}

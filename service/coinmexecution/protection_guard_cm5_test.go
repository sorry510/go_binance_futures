package coinmexecution

import (
	"context"
	"errors"
	"testing"

	"go_binance_futures/feature/api/binance"
)

func TestCM5AlgoChildMissingVersusMalformedFailClosed(t *testing.T) {
	for _, tc := range []struct {
		raw    string
		absent bool
	}{
		{"", true}, {"null", true}, {"0", true}, {"\"\"", true},
		{"123", false}, {"\"123\"", false},
		{"\"not-a-number\"", false}, {"{}", false}, {"-1", false},
	} {
		if got := algoChildAbsent([]byte(tc.raw)); got != tc.absent {
			t.Errorf("actualOrderId=%q absent=%v want=%v", tc.raw, got, tc.absent)
		}
		err := algoChildPresentOrUnknown([]byte(tc.raw))
		if tc.absent && err != nil {
			t.Errorf("%q explicit absent child rejected: %v", tc.raw, err)
		}
		if !tc.absent && !errors.Is(err, ErrUncertain) {
			t.Errorf("%q unexpectedly accepted child: %v", tc.raw, err)
		}
	}
}

func TestCM5WrongTriggeredChildNeverCancelsValidSibling(t *testing.T) {
	db := cm5Fixture(t)
	m := newMockCM5()
	p := NewProcessor(db, m)
	ctx := context.Background()
	entry, err := p.Open(ctx, OpenRequest{Symbol: "BTCUSD_PERP", PositionSide: "LONG", ClientOrderID: "cm5childorderguard", Type: "MARKET"})
	if err != nil {
		t.Fatal(err)
	}
	m.qty["LONG"] = "10"
	if _, err = p.ApplyFill(sampleTrade(m.orders[entry.ClientOrderID], 9021, "10", "50000", "0")); err != nil {
		t.Fatal(err)
	}
	stop, err := p.PlaceProtection(ctx, ProtectionRequest{PositionID: entry.PositionID, ClientAlgoID: "cm5guardstop", Type: "STOP_MARKET", TriggerPrice: "45000"})
	if err != nil {
		t.Fatal(err)
	}
	take, err := p.PlaceProtection(ctx, ProtectionRequest{PositionID: entry.PositionID, ClientAlgoID: "cm5guardtake", Type: "TAKE_PROFIT_MARKET", TriggerPrice: "55000"})
	if err != nil {
		t.Fatal(err)
	}
	m.algoTriggerID = stop.ClientOrderID
	m.algoState = "TRIGGERED"
	m.algoChildID = 90991
	// This remote child belongs to a different hedge side, even though
	// its ID appears in the owned STOP's Algo response.
	m.orders["fake-child"] = &OrderAck{ExchangeOrderID: m.algoChildID, ClientOrderID: "foreign-order",
		Symbol: "BTCUSD_PERP", PositionSide: "SHORT", Side: "BUY", Status: "FILLED", FilledContracts: "10"}
	if _, err = p.ReconcileProtection(ctx, stop.ClientOrderID); err == nil {
		t.Fatal("foreign child was accepted")
	}
	sibling, err := p.FindOrder(take.ClientOrderID)
	if err != nil {
		t.Fatal(err)
	}
	if sibling.Status != "ACTIVE" {
		t.Fatalf("legitimate sibling protection was canceled before child ownership validation: %+v", sibling)
	}
	if len(m.openAlgos) != 2 {
		t.Fatalf("malformed child canceled an existing protective Algo; open=%d", len(m.openAlgos))
	}
	expectCM5(t, db, "SELECT COUNT(*) FROM coin_m_trade_fills", 1)
	// Both protectors remain owned and available for explicit recovery.
	_ = binance.CoinMTrade{}
}

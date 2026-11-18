package coinmexecution

import (
	"context"
	"errors"
	"testing"

	"go_binance_futures/feature/api/binance"
)

func TestCM5TriggeredAlgoChildConfirmedBeforeFillAttribution(t *testing.T) {
	db := cm5Fixture(t)
	m := newMockCM5()
	p := NewProcessor(db, m)
	ctx := context.Background()
	entry, err := p.Open(ctx, OpenRequest{Symbol: "BTCUSD_PERP", PositionSide: "LONG", ClientOrderID: "cm5algoentry", Type: "MARKET"})
	if err != nil {
		t.Fatal(err)
	}
	m.qty["LONG"] = "10"
	if _, err = p.ApplyFill(sampleTrade(m.orders[entry.ClientOrderID], 901, "10", "50000", "0")); err != nil {
		t.Fatal(err)
	}
	protection, err := p.PlaceProtection(ctx, ProtectionRequest{PositionID: entry.PositionID,
		ClientAlgoID: "cm5stopactual", Type: "STOP_MARKET", TriggerPrice: "45000"})
	if err != nil {
		t.Fatal(err)
	}
	// Fresh Algo is NOT the same identity as the resulting ordinary order.
	if protection.ExchangeOrderID == "" || protection.Status != "ACTIVE" {
		t.Fatalf("missing owned Algo identity: %+v", protection)
	}
	m.algoChildID = 99009
	m.algoState = "TRIGGERED"
	m.orders["child"] = &OrderAck{ClientOrderID: "exchange-allocated-child", ExchangeOrderID: m.algoChildID,
		Symbol: "BTCUSD_PERP", Side: "SELL", PositionSide: "LONG", Status: "FILLED", FilledContracts: "10"}
	childFill := binance.CoinMTrade{Symbol: "BTCUSD_PERP", OrderID: m.algoChildID, TradeID: 902,
		Side: "SELL", PositionSide: "LONG", Quantity: "10", Price: "45000",
		RealizedPnL: "-0.002222222222222222", MarginAsset: "BTC", Commission: "0.0002",
		CommissionAsset: "BTC", Time: 1700000000000}
	m.trades[m.algoChildID] = []binance.CoinMTrade{childFill}
	related, err := p.ReconcileProtection(ctx, protection.ClientOrderID)
	if err != nil || related.ExchangeOrderID != "99009" || related.Status != "FILLED" {
		t.Fatalf("verified normal child not attached to owned Algo: %+v %v", related, err)
	}
	pos, err := loadPosition(db, entry.PositionID)
	if err != nil || pos.Status != "CLOSED" || pos.RemainingContracts != "0" || pos.RealizedPnL != "-0.002222222222222222" {
		t.Fatalf("confirmed Algo child Fill not applied as CLOSE: %+v %v", pos, err)
	}
	// Reprocessing exact same real child ID is idempotent.
	again, err := p.ReconcileProtection(ctx, protection.ClientOrderID)
	if err != nil || again.Status != "FILLED" {
		t.Fatalf("repeated Algo lookup regressed terminal state: %+v %v", again, err)
	}
	expectCM5(t, db, "SELECT COUNT(*) FROM coin_m_trade_fills", 2)
}
func TestCM5AlgoChildIdentityFailureDoesNotClaimUnmanagedTrade(t *testing.T) {
	for _, scenario := range []struct {
		name   string
		modify func(*mockCM5)
	}{
		{"missing_triggered_child_id", func(m *mockCM5) { m.algoState = "TRIGGERED" }},
		{"child_from_wrong_symbol", func(m *mockCM5) {
			m.algoState = "TRIGGERED"
			m.algoChildID = 99001
			m.orders["child"] = &OrderAck{ExchangeOrderID: 99001, Symbol: "ETHUSD_PERP",
				Side: "SELL", PositionSide: "LONG", Status: "FILLED", FilledContracts: "10"}
		}},
		{"child_from_wrong_side", func(m *mockCM5) {
			m.algoState = "TRIGGERED"
			m.algoChildID = 99002
			m.orders["child"] = &OrderAck{ExchangeOrderID: 99002, Symbol: "BTCUSD_PERP",
				Side: "BUY", PositionSide: "LONG", Status: "FILLED", FilledContracts: "10"}
		}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			db := cm5Fixture(t)
			m := newMockCM5()
			p := NewProcessor(db, m)
			ctx := context.Background()
			o, err := p.Open(ctx, OpenRequest{Symbol: "BTCUSD_PERP", PositionSide: "LONG", ClientOrderID: "cm5childguardentry", Type: "MARKET"})
			if err != nil {
				t.Fatal(err)
			}
			m.qty["LONG"] = "10"
			if _, err = p.ApplyFill(sampleTrade(m.orders[o.ClientOrderID], 999, "10", "50000", "0")); err != nil {
				t.Fatal(err)
			}
			alg, err := p.PlaceProtection(ctx, ProtectionRequest{PositionID: o.PositionID, ClientAlgoID: "cm5childguardprotect",
				Type: "STOP_MARKET", TriggerPrice: "45000"})
			if err != nil {
				t.Fatal(err)
			}
			scenario.modify(m)
			if _, err = p.ReconcileProtection(ctx, alg.ClientOrderID); err == nil || !errors.Is(err, ErrUncertain) && scenario.name == "missing_triggered_child_id" {
				t.Fatalf("unverified child identity accepted: %v", err)
			}
			expectCM5(t, db, "SELECT COUNT(*) FROM coin_m_trade_fills", 1)
			pos, err := loadPosition(db, o.PositionID)
			if err != nil || pos.Status != "OPEN" || pos.RemainingContracts != "10" {
				t.Fatalf("unmanaged child affected owned position: %+v %v", pos, err)
			}
		})
	}
}

func TestCM5StopAndTakeProfitPairedCancelSiblingOnTrigger(t *testing.T) {
	db := cm5Fixture(t)
	m := newMockCM5()
	p := NewProcessor(db, m)
	ctx := context.Background()
	opening, err := p.Open(ctx, OpenRequest{Symbol: "BTCUSD_PERP", PositionSide: "LONG", ClientOrderID: "cm5pairentry", Type: "MARKET"})
	if err != nil {
		t.Fatal(err)
	}
	m.qty["LONG"] = "10"
	if _, err = p.ApplyFill(sampleTrade(m.orders[opening.ClientOrderID], 1101, "10", "50000", "0")); err != nil {
		t.Fatal(err)
	}
	stop, err := p.PlaceProtection(ctx, ProtectionRequest{PositionID: opening.PositionID,
		ClientAlgoID: "cm5pairstop", Type: "STOP_MARKET", TriggerPrice: "45000"})
	if err != nil {
		t.Fatalf("create STOP: %v", err)
	}
	take, err := p.PlaceProtection(ctx, ProtectionRequest{PositionID: opening.PositionID,
		ClientAlgoID: "cm5pairtake", Type: "TAKE_PROFIT_MARKET", TriggerPrice: "55000"})
	if err != nil || take.Status != "ACTIVE" || stop.Status != "ACTIVE" || len(m.openAlgos) != 2 {
		t.Fatalf("independent STOP + TP should both exist: %+v %+v err=%v remote=%d", stop, take, err, len(m.openAlgos))
	}
	if _, err = p.PlaceProtection(ctx, ProtectionRequest{PositionID: opening.PositionID,
		ClientAlgoID: "cm5pairstopduplicate", Type: "STOP_MARKET", TriggerPrice: "44000"}); err == nil {
		t.Fatal("duplicate same-type STOP accepted")
	}
	// STOP triggers first, with independently confirmed normal child order ID.
	m.algoTriggerID = stop.ClientOrderID
	m.algoState = "TRIGGERED"
	m.algoChildID = 99015
	m.orders["stop_child"] = &OrderAck{ClientOrderID: "exchange-child-after-stop", ExchangeOrderID: 99015,
		Symbol: "BTCUSD_PERP", PositionSide: "LONG", Side: "SELL", Status: "FILLED", FilledContracts: "10"}
	m.trades[99015] = []binance.CoinMTrade{{
		Symbol: "BTCUSD_PERP", OrderID: 99015, TradeID: 1102, Side: "SELL", PositionSide: "LONG",
		Quantity: "10", Price: "45000", RealizedPnL: "-0.002222222222222222",
		MarginAsset: "BTC", Commission: "0.0001", CommissionAsset: "BTC", Time: 1700000000000,
	}}
	stopFilled, err := p.ReconcileProtection(ctx, stop.ClientOrderID)
	if err != nil || stopFilled.Status != "FILLED" {
		t.Fatalf("triggered STOP not reconciled: %+v %v", stopFilled, err)
	}
	sibling, err := p.FindOrder(take.ClientOrderID)
	if err != nil || sibling.Status != "CANCELED" || len(m.openAlgos) != 1 {
		t.Fatalf("untriggered TAKE_PROFIT sibling remained active: %+v %v remainingRemote=%d", sibling, err, len(m.openAlgos))
	}
	pos, err := loadPosition(db, opening.PositionID)
	if err != nil || pos.Status != "CLOSED" || pos.RemainingContracts != "0" {
		t.Fatalf("paired Algo position not closed by verified child Fill: %+v %v", pos, err)
	}
	expectCM5(t, db, "SELECT COUNT(*) FROM coin_m_trade_fills", 2)
}
func TestCM5UnmanagedRemoteAlgoBlocksEntry(t *testing.T) {
	db := cm5Fixture(t)
	m := newMockCM5()
	m.openAlgos = []binance.CoinMAlgoOrderResponse{{
		ClientAlgoID: "unmanaged-stop", AlgoID: 98765, Symbol: "BTCUSD_PERP",
		PositionSide: "LONG", Side: "SELL", OrderType: "STOP_MARKET", AlgoStatus: "NEW",
	}}
	p := NewProcessor(db, m)
	if _, err := p.Open(context.Background(), OpenRequest{Symbol: "BTCUSD_PERP",
		PositionSide: "LONG", ClientOrderID: "cm5shouldblock", Type: "MARKET"}); err == nil {
		t.Fatal("unmanaged remote active Algo did not prevent a new COIN-M entry")
	}
	expectCM5(t, db, "SELECT COUNT(*) FROM coin_m_managed_orders", 0)
}

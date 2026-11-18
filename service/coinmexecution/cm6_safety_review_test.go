package coinmexecution

import (
	"context"
	"errors"
	"testing"

	"go_binance_futures/models"
	"go_binance_futures/service/coinmquant"
)

func TestCM6CloseRulesCannotBypassTakeProfitStopLoss(t *testing.T) {
	marketCalls := 0
	strategy := coinMStrategy{Market: func(_ context.Context, _ models.DeliverySymbols) (map[string]interface{}, error) {
		marketCalls++
		return map[string]interface{}{"NowPrice": float64(55000)}, nil
	}}
	coin := models.DeliverySymbols{Symbol: "BTCUSD_PERP", Leverage: 4, Profit: "8", Loss: "6",
		Strategy: `[{"name":"force","type":"close_long","enable":true,"code":"true"}]`}
	mark, pnl := "55000", "0.00035" // Initial margin 0.005 BTC; this is 7% ROI
	snapshot := coinmquant.PositionSummary{
		CoinMManagedPosition: models.CoinMManagedPosition{
			ID: 1, Symbol: "BTCUSD_PERP", PositionSide: "LONG", Status: "OPEN",
			ContractSize: "100", RemainingContracts: "10", EntryPrice: "50000",
		},
		MarkPrice: &mark, UnrealizedPnL: &pnl, PnLStatus: "verified",
	}
	exit, err := strategy.Exit(context.Background(), coin, snapshot)
	if err != nil || exit {
		t.Fatalf("close_long=true bypassed 8%% ROI threshold: %v %v", exit, err)
	}
	if marketCalls != 0 {
		t.Fatalf("nontriggered ROI did unnecessary market API evaluation: %d", marketCalls)
	}
	pnl = "0.0005" // +10% ROI: only now may explicit rule authorize CLOSE
	exit, err = strategy.Exit(context.Background(), coin, snapshot)
	if err != nil || !exit {
		t.Fatalf("verified +10%% LONG ROI should allow matching close_long: %v %v", exit, err)
	}
	coin.Strategy = `[{"name":"force","type":"close_short","enable":true,"code":"true"}]`
	snapshot.PositionSide = "SHORT"
	pnl = "0.0003" // +6% ROI: still below +8% trigger
	exit, err = strategy.Exit(context.Background(), coin, snapshot)
	if err != nil || exit {
		t.Fatalf("SHORT close bypassed +8%%: %v %v", exit, err)
	}
	pnl = "0.0005"
	exit, err = strategy.Exit(context.Background(), coin, snapshot)
	if err != nil || !exit {
		t.Fatalf("SHORT +10%% ROI failed to evaluate close_short: %v %v", exit, err)
	}
	coin.Profit = "0"
	coin.Loss = "0"
	exit, err = strategy.Exit(context.Background(), coin, snapshot)
	if err != nil || exit {
		t.Fatalf("disabled TP/SL allowed unconditional close: %v %v", exit, err)
	}
	coin.Profit = "-8"
	if _, err = strategy.Exit(context.Background(), coin, snapshot); err == nil {
		t.Fatal("invalid negative ROI threshold accepted")
	}
}
func TestCM6CancelProtectionChecksSavedAlgoIdentity(t *testing.T) {
	for _, tc := range []struct {
		name    string
		corrupt func(*mockCM5)
	}{
		{"wrong_remote_order_type", func(m *mockCM5) { m.openAlgos[0].OrderType = "TAKE_PROFIT_MARKET" }},
		{"wrong_remote_algo_id", func(m *mockCM5) { m.openAlgos[0].AlgoID += 123 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := cm5Fixture(t)
			m := newMockCM5()
			p := NewProcessor(db, m)
			ctx := context.Background()
			opened, err := p.Open(ctx, OpenRequest{Symbol: "BTCUSD_PERP", PositionSide: "LONG",
				Type: "MARKET", ClientOrderID: "cm6cancelidentityopen"})
			if err != nil {
				t.Fatal(err)
			}
			m.qty["LONG"] = "10"
			if _, err = p.ApplyFill(sampleTrade(m.orders[opened.ClientOrderID], 71101, "10", "50000", "0")); err != nil {
				t.Fatal(err)
			}
			stop, err := p.PlaceProtection(ctx, ProtectionRequest{PositionID: opened.PositionID,
				ClientAlgoID: "cm6cancelidentitystop", Type: "STOP_MARKET", TriggerPrice: "45000"})
			if err != nil || stop.Status != "ACTIVE" {
				t.Fatalf("STOP setup failed %+v %v", stop, err)
			}
			tc.corrupt(m)
			_, err = p.CancelProtection(ctx, stop.ClientOrderID)
			if !errors.Is(err, ErrUncertain) {
				t.Fatalf("incorrect remote identity accepted as canceled: %v", err)
			}
			saved, err := p.FindOrder(stop.ClientOrderID)
			if err != nil || saved.Status != "UNKNOWN" {
				t.Fatalf("remote identity mismatch not left uncertain: %+v %v", saved, err)
			}
			if _, err = p.Close(ctx, CloseRequest{PositionID: opened.PositionID,
				ClientOrderID: "cm6unsafeclose", Type: "MARKET"}); err == nil {
				t.Fatal("CLOSE submitted despite unresolved protective Algo identity")
			}
			if len(m.calls) != 1 {
				t.Fatalf("unverified cancellation permitted new regular order: %+v", m.calls)
			}
		})
	}
}

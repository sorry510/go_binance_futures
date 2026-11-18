package coinmexecution

import (
	"context"
	"strings"
	"testing"
	"time"

	"go_binance_futures/service/coinmquant"
)

// Real REST/WS transactions are deliberately NOT used. Exercise CM-6 Runner,
// CM-5 Processor and coin-m_managed_* Fill evidence end to end via an
// isolated database and deterministic Fake DAPI gateway.
func TestCM7FakeManagedOpenProtectCloseFullLifecycle(t *testing.T) {
	for _, side := range []string{"LONG", "SHORT"} {
		t.Run(side, func(t *testing.T) {
			r, processor, remote, signals, db, now := cm6RunnerFixture(t)
			ctx := context.Background()
			signals.entrySide = side
			// Global+symbol gates and stable COIN-M ID create only one order.
			if err := r.Tick(ctx); err != nil {
				t.Fatal(err)
			}
			if len(remote.calls) != 1 || remote.calls[0].PositionSide != side ||
				!strings.HasPrefix(remote.calls[0].ClientOrderID, "cm6") {
				t.Fatalf("wrong first DAPI intention: %+v", remote.calls)
			}
			openAck := remote.orders[remote.calls[0].ClientOrderID]
			if openAck == nil {
				t.Fatal("local open ID missing in Fake DAPI")
			}
			remote.qty[side] = "10"
			tr := sampleTrade(openAck, 79001, "10", "50000", "0")
			if side == "SHORT" {
				tr.Side = "SELL"
			}
			applied, err := processor.ApplyFill(tr)
			if err != nil || !applied {
				t.Fatalf("CM7 open Fill rejected: %v %t", err, applied)
			}
			applied, err = processor.ApplyFill(tr)
			if err != nil || applied {
				t.Fatalf("duplicate exchange trade ID not idempotent: %v %t", err, applied)
			}
			expectCM5(t, db, "SELECT COUNT(*) FROM coin_m_trade_fills", 1)

			// Closing entries OFF must not affect the already-owned position or
			// prevent STOP+TP from being installed.
			if _, err := db.Raw("UPDATE config SET coin_m_quant_enable=0").Exec(); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Raw("UPDATE delivery_symbols SET enable=0").Exec(); err != nil {
				t.Fatal(err)
			}
			if need, err := CoinMNeedsRuntime(db); err != nil || !need {
				t.Fatalf("existing Managed exit must stay live with both switches OFF: %v %t", err, need)
			}
			signals.exit = false
			*now = now.Add(11 * time.Second)
			if err := r.Tick(ctx); err != nil {
				t.Fatalf("STOP placement: %v", err)
			}
			if len(remote.openAlgos) != 1 || remote.openAlgos[0].OrderType != "STOP_MARKET" {
				t.Fatalf("missing owned STOP: %+v", remote.openAlgos)
			}
			// The 2-second Runner must first reconcile the existing STOP
			// before it can create the other half of the protective pair.
			for attempt := 0; attempt < 20 && len(remote.openAlgos) < 2; attempt++ {
				*now = now.Add(2 * time.Second)
				if err := r.Tick(ctx); err != nil {
					t.Fatalf("STOP reconciliation/TP tick %d: %v", attempt, err)
				}
			}
			if len(remote.openAlgos) != 2 {
				t.Fatalf("missing both owned protectors after realistic 2s tick cadence: %+v", remote.openAlgos)
			}
			stopCount, takeCount := 0, 0
			for _, a := range remote.openAlgos {
				switch a.OrderType {
				case "STOP_MARKET":
					stopCount++
				case "TAKE_PROFIT_MARKET":
					takeCount++
				}
			}
			if stopCount != 1 || takeCount != 1 {
				t.Fatalf("duplicate or wrong protective Algos: %+v", remote.openAlgos)
			}

			// A legitimate strategy exit cancels both owned protection IDs before
			// a unique CLOSE intent. This must work even though both switches OFF.
			signals.exit = true
			for attempt := 0; attempt < 25 && len(remote.calls) < 2; attempt++ {
				*now = now.Add(2 * time.Second)
				if err := r.Tick(ctx); err != nil {
					t.Fatalf("exit/reconcile attempt %d: %v", attempt, err)
				}
			}
			if len(remote.calls) != 2 {
				t.Fatalf("existing managed position could not exit after STOP+TP: %+v", remote.calls)
			}
			closeRequest := remote.calls[1]
			expectedSide := "SELL"
			if side == "SHORT" {
				expectedSide = "BUY"
			}
			if closeRequest.PositionSide != side || closeRequest.Side != expectedSide {
				t.Fatalf("wrong hedge-side close %+v", closeRequest)
			}
			if len(remote.openAlgos) != 0 {
				t.Fatalf("CLOSE sent before all owned protections canceled: %+v", remote.openAlgos)
			}
			closeAck := remote.orders[closeRequest.ClientOrderID]
			if closeAck == nil {
				t.Fatal("missing close acknowledgement")
			}
			closeFill := sampleTrade(closeAck, 79002, "10", "55000", "0.0018")
			if side == "SHORT" {
				closeFill.Price = "45000"
				closeFill.Side = "BUY"
			}
			if ok, err := processor.ApplyFill(closeFill); err != nil || !ok {
				t.Fatalf("closing Fill not recorded %t %v", ok, err)
			}
			expectCM5(t, db, "SELECT COUNT(*) FROM coin_m_trade_fills", 2)
			expectCM5(t, db, "SELECT COUNT(*) FROM coin_m_managed_positions WHERE status='CLOSED'", 1)
			summaries, err := coinmquant.PositionSummaries(db, coinmquant.LedgerFilter{Symbol: "BTCUSD_PERP"}, nil)
			if err != nil || len(summaries.List) != 1 {
				t.Fatalf("bad COIN-M read model: %+v %v", summaries, err)
			}
			summary := summaries.List[0]
			if summary.PnLStatus != "verified" || summary.TotalPnL == nil ||
				summary.RemainingContracts != "0" || summary.PositionSide != side {
				t.Fatalf("closed managed cycle not verified: %+v", summary)
			}
			snapshot, err := ReadRuntimeSummary(db)
			if err != nil || snapshot.GlobalEntryEnabled || snapshot.ManagedCyclesNeedingWork != 0 ||
				snapshot.OrdersNeedingReconciliation != 0 || !snapshot.ReadOnly {
				t.Fatalf("local status still shows false positions or live execution: %+v %v", snapshot, err)
			}
			// Complete lifecycle, switches OFF: must not re-open or POST on next tick.
			*now = now.Add(2 * time.Second)
			if err := r.Tick(ctx); err != nil {
				t.Fatalf("post-close tick: %v", err)
			}
			if len(remote.calls) != 2 {
				t.Fatalf("re-opened despite both switches off: %+v", remote.calls)
			}
		})
	}
}

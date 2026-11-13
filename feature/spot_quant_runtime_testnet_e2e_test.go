package feature

import (
	"context"
	"errors"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"go_binance_futures/service/spotexecution"
	"go_binance_futures/service/spotledger"
	spotbinance "go_binance_futures/spot/api/binance"
)

// Opt-in: ONE real Testnet auto-strategy cycle using an isolated TEMP SQLite.
// No live main app, no Mainnet key, no developer/production database.
// NEVER run implicitly from go test ./...; this test makes real TESTNET orders.
func TestStage0675RealTestnetAutomaticRunnerCycle(t *testing.T) {
	if os.Getenv("SPOT_QUANT_TESTNET_AUTO_E2E") != "1" {
		t.Skip("real Spot Testnet automatic BUY/SELL requires explicit opt-in")
	}
	runner, _, _, db := spot0673Fixture(t)
	spot0673Enable(t, db)
	// Only a tiny, deterministic TESTNET strategy in the isolated fixture.
	// The user's real strategy rules and production DB are not modified.
	rules := `[{"name":"testnet buy","type":"long","enable":true,"code":"NowPrice > 0"},
 {"name":"testnet close","type":"close_long","enable":true,"code":"NowPrice > 0"}]`
	if _, err := db.Raw(`UPDATE spot_symbols SET usdt='20',
    profit='0.000001', loss='0.000001',strategy=?,technology='{}'
    WHERE symbol='BTCUSDT'`, rules).Exec(); err != nil {
		t.Fatal(err)
	}
	exchange, err := spotbinance.NewSpotQuantTestnetTradingAdapterFromConfig()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 140*time.Second)
	defer cancel()
	if err := exchange.SyncSpotMarketServerTime(ctx); err != nil {
		t.Fatalf("Spot Testnet time synchronization: %v", err)
	}
	strategy, err := SpotMarketStrategy(exchange)
	if err != nil {
		t.Fatal(err)
	}
	processor, err := spotexecution.NewProcessor(exchange, spotexecution.DBGate{DB: db}, runner.ledger)
	if err != nil {
		t.Fatal(err)
	}
	ready := &atomic.Bool{}
	runner.strategy = strategy
	runner.exec = spotQuantLiveExecutor{processor: processor, wsReady: ready}
	runner.orderPrefix = "sqt"
	runner.spot3dOpen = SpotMarketThreeDayOpen(exchange)
	runner.clock = time.Now

	streamCtx, streamCancel := context.WithCancel(context.Background())
	streamErr := make(chan error, 1)
	subscribed := make(chan struct{}, 1)
	go func() {
		streamErr <- exchange.RunSpotQuantUserStreamWithStatus(streamCtx, processor, func(v bool) {
			ready.Store(v)
			if v {
				select {
				case subscribed <- struct{}{}:
				default:
				}
			}
		})
	}()
	defer func() {
		streamCancel()
		select {
		case <-streamErr:
		case <-time.After(4 * time.Second):
		}
	}()
	select {
	case <-subscribed:
		t.Log("AUTO_TESTNET_SIGNED_WS_READY=PASS")
	case err := <-streamErr:
		t.Fatalf("Testnet signed User WS not available, no orders sent: %v", err)
	case <-time.After(25 * time.Second):
		t.Fatal("Testnet WS subscription not acknowledged; no orders sent")
	}
	buySeen, sellSeen := false, false
	var openPositionID int64
	for tick := 0; tick < 14; tick++ {
		if err := ctx.Err(); err != nil {
			t.Fatalf("Testnet E2E exceeded bounded interval: %v", err)
		}
		if !ready.Load() {
			t.Fatal("signed Testnet WS disconnected: no new order permitted")
		}
		err := runner.Run(ctx)
		if err != nil && !errors.Is(err, spotexecution.ErrNeedsReview) {
			t.Fatalf("Testnet runner tick %d failed (check prior orders before ever retrying POST): %v", tick, err)
		}
		page, e := runner.ledger.ListPositions(spotledger.PositionFilter{Symbol: "BTCUSDT", Page: 1, Limit: 10})
		if e != nil {
			t.Fatal(e)
		}
		if len(page.List) > 1 {
			t.Fatalf("unexpected multiple Testnet strategy cycles: %d", len(page.List))
		}
		if len(page.List) == 1 {
			openPositionID = page.List[0].ID
			fills, e := runner.ledger.ListFills(openPositionID)
			if e != nil {
				t.Fatal(e)
			}
			for _, fill := range fills {
				if fill.Side == "BUY" {
					buySeen = true
				}
				if fill.Side == "SELL" {
					sellSeen = true
				}
			}
			state, e := runner.ledger.Accounting(openPositionID, "")
			if e == nil && state.Position.Status == "CLOSED" && buySeen && sellSeen {
				t.Logf("AUTO_TESTNET_REAL_CYCLE_PASS ticks=%d position=%d BUYFill=true SELLFill=true CLOSED=true",
					tick+1, openPositionID)
				return
			}
		}
		time.Sleep(1200 * time.Millisecond)
	}
	t.Fatalf("Testnet automatic cycle incomplete, strategy inventory may remain: BUYFill=%t SELLFill=%t position=%d", buySeen, sellSeen, openPositionID)
}

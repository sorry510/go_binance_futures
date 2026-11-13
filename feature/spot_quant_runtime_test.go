package feature

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"go_binance_futures/service/spotexecution"
	"go_binance_futures/service/spotledger"
)

func TestSpotQuantLiveWSGateAndFakeCycle(t *testing.T) {
	r, ex, processor, db := spot0673Fixture(t)
	spot0673Enable(t, db)
	ready := &atomic.Bool{}
	r.exec = spotQuantLiveExecutor{processor: processor, wsReady: ready}
	if err := r.Run(context.Background()); !errors.Is(err, spotexecution.ErrDisabled) || ex.places != 0 {
		t.Fatalf("WS not authenticated but BUY proceeded: %v places=%d", err, ex.places)
	}
	ready.Store(true)
	if err := r.Run(context.Background()); err != nil || ex.places != 1 {
		t.Fatalf("signed WS BUY: %v places=%d", err, ex.places)
	}
	pos, ok, err := r.activeSpotPosition("BTCUSDT")
	if err != nil || !ok || pos.Status != "OPEN" {
		t.Fatalf("BUY Fill missing: %v %+v", err, pos)
	}
	ready.Store(false)
	ex.price = "110"
	if err := r.Run(context.Background()); !errors.Is(err, spotexecution.ErrDisabled) || ex.places != 1 {
		t.Fatalf("WS disconnected but SELL proceeded: %v places=%d", err, ex.places)
	}
	ready.Store(true)
	if err := r.Run(context.Background()); err != nil || ex.places != 2 {
		t.Fatalf("WS recovery did not allow verified SELL: %v places=%d", err, ex.places)
	}
	snapshot, err := r.ledger.Accounting(pos.ID, "110")
	if err != nil || snapshot.Position.Status != "CLOSED" {
		t.Fatalf("not CLOSED: %v %+v", err, snapshot)
	}
	for i := 0; i < 4; i++ {
		if err := r.Run(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if ex.places != 2 {
		t.Fatalf("duplicate orders after CLOSE: %d", ex.places)
	}
}

func TestSpotQuantLiveBudgetComesFromSymbolNotHardcoded(t *testing.T) {
	r, ex, processor, db := spot0673Fixture(t)
	spot0673Enable(t, db)
	if _, err := db.Raw("UPDATE spot_symbols SET usdt='100' WHERE symbol='BTCUSDT'").Exec(); err != nil {
		t.Fatal(err)
	}
	ready := &atomic.Bool{}
	ready.Store(true)
	r.exec = spotQuantLiveExecutor{processor: processor, wsReady: ready}
	if err := r.Run(context.Background()); err != nil || ex.places != 1 {
		t.Fatalf("symbol-configured budget rejected: %v places=%d", err, ex.places)
	}
}

func TestSpotQuantLiveEnvironmentOrderIdentity(t *testing.T) {
	r, ex, processor, db := spot0673Fixture(t)
	spot0673Enable(t, db)
	ready := &atomic.Bool{}
	ready.Store(true)
	r.exec = spotQuantLiveExecutor{processor: processor, wsReady: ready}
	r.orderPrefix = "sqt"
	open := time.Unix(1794517200, 0).UTC()
	testID := r.buyClientID("BTCUSDT", "1m", open)
	r.orderPrefix = "sqm"
	mainID := r.buyClientID("BTCUSDT", "1m", open)
	if len(testID) != 34 || len(mainID) != 34 || testID == mainID ||
		!strings.HasPrefix(testID, "sqt") || !strings.HasPrefix(mainID, "sqm") {
		t.Fatalf("Spot Testnet/Mainnet IDs not separate: %q %q", testID, mainID)
	}
	r.orderPrefix = "sqt"
	if err := r.Run(context.Background()); err != nil || ex.places != 1 {
		t.Fatalf("Testnet BUY: %v places=%d", err, ex.places)
	}
	ex.price = "110"
	r.orderPrefix = "sqm"
	if err := r.Run(context.Background()); !errors.Is(err, spotledger.ErrNeedsReview) || ex.places != 1 {
		t.Fatalf("mainnet may have sold testnet position: %v places=%d", err, ex.places)
	}
	r.orderPrefix = "sqt"
	if err := r.Run(context.Background()); err != nil || ex.places != 2 {
		t.Fatalf("correct environment SELL: %v places=%d", err, ex.places)
	}
}

func TestSpotQuantLiveSignedURLRedaction(t *testing.T) {
	message := `Get "https://api.binance.com/api/v3/order?symbol=BTCUSDT&signature=abcdef1234567890&api_key=x1": timeout`
	redacted := spotQuantSafeMessage(message)
	if strings.Contains(redacted, "abcdef1234567890") ||
		strings.Contains(redacted, "api_key=x1") ||
		!strings.Contains(redacted, "symbol=BTCUSDT") ||
		!strings.Contains(redacted, "[REDACTED]") {
		t.Fatalf("Spot signed URL leaked in logs: %s", redacted)
	}
}

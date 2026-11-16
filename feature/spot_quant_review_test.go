package feature

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"go_binance_futures/models"
	"go_binance_futures/service/spotexecution"
	"go_binance_futures/service/spotledger"
	spotbinance "go_binance_futures/spot/api/binance"

	sdk "github.com/adshao/go-binance/v2"
)

type spotReviewRecoveryExec struct {
	spotQuantExecutor
	calls int
}

func (e *spotReviewRecoveryExec) Reconcile(_ context.Context, _ int64) (models.SpotManagedOrder, error) {
	e.calls++
	return models.SpotManagedOrder{}, spotledger.ErrNeedsReview
}

func TestSpotReviewDisabledRecoveryBackoffPreservesOwnership(t *testing.T) {
	r, ex, _, db := spotRunnerFixture(t)
	spotRunnerSetSwitches(t, db, 1, 1)
	if err := r.Run(context.Background()); err != nil || ex.buys != 1 {
		t.Fatalf("seed managed BUY: %v buys=%d", err, ex.buys)
	}
	pos, ok, err := r.activeSpotPosition("BTCUSDT")
	if err != nil || !ok {
		t.Fatal("missing Fill-owned position:", err)
	}
	if _, _, err := r.ledger.CreateOrder(spotledger.OrderIntent{
		PositionID: pos.ID, Symbol: "BTCUSDT", Side: "SELL", Type: "MARKET",
		ClientOrderID: "pending_spot_review_exit", RequestedQty: "1",
	}); err != nil {
		t.Fatal(err)
	}
	recovery := &spotReviewRecoveryExec{spotQuantExecutor: ex}
	r.exec = recovery
	spotRunnerSetSwitches(t, db, 0, 0)
	if err := r.Run(context.Background()); !errors.Is(err, spotledger.ErrNeedsReview) || recovery.calls != 1 {
		t.Fatalf("first OFF reconciliation failure: %v queries=%d", err, recovery.calls)
	}
	if err := r.Run(context.Background()); err != nil || recovery.calls != 1 {
		t.Fatalf("permanent OFF error should cool down: %v queries=%d", err, recovery.calls)
	}
	if r.offRetryDelay["BTCUSDT"] != 5*time.Second {
		t.Fatal("initial cooldown not 5s")
	}
	r.offRetryAt["BTCUSDT"] = time.Now().Add(-time.Second)
	if err := r.Run(context.Background()); !errors.Is(err, spotledger.ErrNeedsReview) || recovery.calls != 2 {
		t.Fatalf("OFF error should retry after cooldown: %v queries=%d", err, recovery.calls)
	}
	if r.offRetryDelay["BTCUSDT"] != 10*time.Second || ex.buys != 1 || ex.sells != 0 {
		t.Fatalf("unexpected OFF retries or trade: delay=%v buys=%d sells=%d", r.offRetryDelay["BTCUSDT"], ex.buys, ex.sells)
	}
}

func TestSpotReviewDisabledPositionWSNotReadyFailsClosed(t *testing.T) {
	r, ex, processor, db := spot0673Fixture(t)
	spot0673Enable(t, db)
	ready := &atomic.Bool{}
	ready.Store(true)
	r.exec = spotQuantLiveExecutor{processor: processor, wsReady: ready}
	if err := r.Run(context.Background()); err != nil || ex.places != 1 {
		t.Fatalf("seed Fill: %v orders=%d", err, ex.places)
	}
	spotRunnerSetSwitches(t, db, 0, 0)
	ex.price = "110"
	ready.Store(false)
	if err := r.Run(context.Background()); !errors.Is(err, spotexecution.ErrDisabled) || ex.places != 1 {
		t.Fatalf("WS down allowed SELL: %v orders=%d", err, ex.places)
	}
	ready.Store(true)
	if err := r.Run(context.Background()); err != nil || ex.places != 2 {
		t.Fatalf("WS restored but managed exit failed: %v orders=%d", err, ex.places)
	}
}

func TestSpotReviewIdleRuntimeDoesNotUseNetwork(t *testing.T) {
	rt := &spotQuantRuntime{needsWork: func() (bool, error) { return false, nil }}
	rt.ready.Store(true)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); rt.streamLoop(ctx) }()
	deadline := time.After(500 * time.Millisecond)
	for rt.ready.Load() {
		select {
		case <-deadline:
			cancel()
			t.Fatal("idle runtime kept WS ready")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("idle loop did not stop on cancellation")
	}
}

type spotReviewTickerTransport struct{ closedAt int64 }

func (tr spotReviewTickerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host != "api.binance.com" || req.URL.Path != "/api/v3/ticker/24hr" {
		return nil, fmt.Errorf("unexpected external path %s", req.URL.String())
	}
	payload := fmt.Sprintf(`[{"symbol":"BTCUSDT","lastPrice":"101","openPrice":"95","highPrice":"105","lowPrice":"90","priceChangePercent":"4","closeTime":%d}]`, tr.closedAt)
	return &http.Response{StatusCode: 200, Status: "200 OK", Body: io.NopCloser(strings.NewReader(payload)), Header: http.Header{}, Request: req}, nil
}

func TestSpotReviewStrategyUsesExchangeTickerTimeAndIndependentNow(t *testing.T) {
	makeStrategy := func(age time.Duration) func(models.SpotSymbols) (map[string]interface{}, error) {
		client := sdk.NewClient("test_key", "test_secret")
		client.SetApiEndpoint("https://api.binance.com")
		client.HTTPClient = &http.Client{Transport: spotReviewTickerTransport{closedAt: time.Now().Add(-age).UnixMilli()}}
		adapter, err := spotbinance.NewSpotQuantAdapterWithClient(client, true)
		if err != nil {
			t.Fatal(err)
		}
		evaluator, err := SpotMarketStrategy(adapter)
		if err != nil {
			t.Fatal(err)
		}
		return evaluator.LoadEnvironment
	}
	coin := models.SpotSymbols{Symbol: "BTCUSDT", Type: "USDT", Technology: "{}"}
	env, err := makeStrategy(4 * time.Minute)(coin)
	if err != nil || env["NowPrice"] != 101.0 {
		t.Fatalf("fresh ticker not accepted: %v env=%v", err, env)
	}
	if _, err := makeStrategy(6 * time.Minute)(coin); err == nil {
		t.Fatal("stale ticker incorrectly accepted")
	}
}

package binance

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/adshao/go-binance/v2/futures"
)

func testLeadCache(t *testing.T, load LeadSymbolLoader) *LeadSymbolCache {
	t.Helper()
	a, err := NewAccountClient(LeadAccountID, futures.NewClient("test", "test"))
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewLeadSymbolCache(a, load)
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func TestStage3LeadSymbolCacheHourlyAndFailClosed(t *testing.T) {
	now := time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC)
	var calls atomic.Int32
	failure := false
	cache := testLeadCache(t, func(context.Context) ([]LeadTradingSymbol, error) {
		calls.Add(1)
		if failure {
			return nil, errors.New("network unavailable")
		}
		return []LeadTradingSymbol{{Symbol: " btcusdt ", QuoteAsset: "USDT"}, {Symbol: "ETHUSDT", QuoteAsset: "USDT"}, {Symbol: "XRPUSDC", QuoteAsset: "USDC"}}, nil
	})
	cache.now = func() time.Time { return now }
	if cache.Allows("BTCUSDT") {
		t.Fatal("uninitialized cache admitted open")
	}
	if err := cache.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !cache.Allows("btcusdt") || !cache.Allows("ETHUSDT") || cache.Allows("XRPUSDC") {
		t.Fatal("whitelist invalid")
	}
	if s := cache.Status(); !s.Ready || s.Count != 2 {
		t.Fatalf("status=%+v", s)
	}
	for i := 0; i < 2000; i++ {
		if !cache.Allows("BTCUSDT") {
			t.Fatal("lost whitelist")
		}
		if _, err := cache.Snapshot(); err != nil {
			t.Fatal(err)
		}
		if err := cache.Refresh(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("trade loop caused %d SAPI calls", calls.Load())
	}
	now = now.Add(time.Hour)
	failure = true
	if err := cache.Refresh(context.Background()); err == nil {
		t.Fatal("expected transient failure")
	}
	if !cache.Allows("BTCUSDT") {
		t.Fatal("transient failure erased unexpired snapshot")
	}
	if err := cache.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatalf("retry not throttled: calls=%d", calls.Load())
	}
	now = now.Add(2 * time.Hour)
	if cache.Allows("BTCUSDT") {
		t.Fatal("expired snapshot allows new open")
	}
	if _, err := cache.Snapshot(); !errors.Is(err, ErrLeadSymbolNotReady) {
		t.Fatalf("expected fail-closed: %v", err)
	}
	failure = false
	if err := cache.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !cache.Allows("BTCUSDT") {
		t.Fatal("successful refresh did not restore whitelist")
	}
	cache.Invalidate()
	if cache.Allows("BTCUSDT") {
		t.Fatal("invalidated cache remains allowed")
	}
}
func TestStage3LeadSymbolCacheConcurrentRefreshAndCredentialFence(t *testing.T) {
	var calls atomic.Int32
	start := make(chan struct{})
	release := make(chan struct{})
	cache := testLeadCache(t, func(context.Context) ([]LeadTradingSymbol, error) {
		calls.Add(1)
		close(start)
		<-release
		return []LeadTradingSymbol{{Symbol: "BTCUSDT", QuoteAsset: "USDT"}}, nil
	})
	var wg sync.WaitGroup
	wg.Add(1)
	result := make(chan error, 1)
	go func() { defer wg.Done(); result <- cache.Refresh(context.Background()) }()
	<-start
	cache.Invalidate()
	close(release)
	wg.Wait()
	if err := <-result; !errors.Is(err, ErrLeadSymbolNotReady) {
		t.Fatalf("stale success accepted: %v", err)
	}
	if cache.Allows("BTCUSDT") {
		t.Fatal("stale in-flight response repopulated cache")
	}
	if calls.Load() != 1 {
		t.Fatalf("calls=%d", calls.Load())
	}
}
func TestStage3LeadSymbolCacheConcurrentReadersSingleRefresh(t *testing.T) {
	var calls atomic.Int32
	cache := testLeadCache(t, func(context.Context) ([]LeadTradingSymbol, error) {
		calls.Add(1)
		return []LeadTradingSymbol{{Symbol: "BTCUSDT", QuoteAsset: "USDT"}}, nil
	})
	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _ = cache.Refresh(context.Background()) }()
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatalf("refreshes=%d want 1", calls.Load())
	}
}
func TestStage3LeadSymbolCacheForbiddenAndEmpty(t *testing.T) {
	for _, tc := range []struct {
		name    string
		err     error
		symbols []LeadTradingSymbol
	}{
		{"forbidden", ErrLeadSAPIUnauthorized, nil},
		{"empty", nil, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cache := testLeadCache(t, func(context.Context) ([]LeadTradingSymbol, error) { return tc.symbols, tc.err })
			if err := cache.Refresh(context.Background()); err == nil {
				t.Fatal("invalid response accepted")
			}
			if cache.Allows("BTCUSDT") || cache.Status().Ready {
				t.Fatal("invalid response allowed open")
			}
		})
	}
}

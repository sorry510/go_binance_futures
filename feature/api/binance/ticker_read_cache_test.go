package binance

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/adshao/go-binance/v2/futures"
)

func TestFuturesTickerSnapshotFreshness(t *testing.T) {
	futuresTickerSnapshot.Lock()
	oldItems := futuresTickerSnapshot.Items
	futuresTickerSnapshot.Items = make(map[string]futuresTickerSnapshotEntry)
	futuresTickerSnapshot.Unlock()
	oldFullAt := futuresTickerLastFullAt.Load()
	t.Cleanup(func() {
		futuresTickerSnapshot.Lock()
		futuresTickerSnapshot.Items = oldItems
		futuresTickerSnapshot.Unlock()
		futuresTickerLastFullAt.Store(oldFullAt)
	})

	now := time.Now()
	storeFuturesTickerEvents(futures.WsAllMarketTickerEvent{
		&futures.WsMarketTickerEvent{Symbol: "BTCUSDT", ClosePrice: "101", Time: now.UnixMilli()},
	}, now)

	price, ok := GetFreshFuturesTickerPrice("btcusdt")
	if !ok || price != "101" {
		t.Fatalf("fresh ticker price ok=%v price=%q", ok, price)
	}
	if !FuturesTickerWSFresh() {
		t.Fatal("fresh all-market ticker event must mark WS fresh")
	}

	futuresTickerSnapshot.Lock()
	entry := futuresTickerSnapshot.Items["BTCUSDT"]
	entry.ReceivedAt = now.Add(-10 * time.Second).UnixMilli()
	futuresTickerSnapshot.Items["BTCUSDT"] = entry
	futuresTickerSnapshot.Unlock()
	if _, ok := GetFreshFuturesTickerPrice("BTCUSDT"); ok {
		t.Fatal("stale per-symbol ticker must not be served")
	}
}

func TestFuturesTickerRESTFallbackCache(t *testing.T) {
	futuresTickerRESTCache.Lock()
	old := futuresTickerRESTCache.items
	futuresTickerRESTCache.items = make(map[string]futuresTickerRESTCacheEntry)
	futuresTickerRESTCache.Unlock()
	t.Cleanup(func() {
		futuresTickerRESTCache.Lock()
		futuresTickerRESTCache.items = old
		futuresTickerRESTCache.Unlock()
	})

	var calls atomic.Int32
	loader := func(context.Context) ([]*futures.SymbolPrice, error) {
		calls.Add(1)
		return []*futures.SymbolPrice{{Symbol: "BTCUSDT", Price: "100"}}, nil
	}
	first, err := getFuturesTickerRESTCached(context.Background(), "BTCUSDT", loader)
	if err != nil {
		t.Fatal(err)
	}
	second, err := getFuturesTickerRESTCached(context.Background(), "BTCUSDT", loader)
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("ticker REST loader calls=%d want=1", calls.Load())
	}
	first[0].Price = "mutated"
	if second[0].Price != "100" {
		t.Fatalf("ticker cache leaked caller mutation: %+v", second)
	}
}

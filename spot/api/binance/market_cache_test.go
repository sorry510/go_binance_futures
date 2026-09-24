package binance

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	spotbinance "github.com/adshao/go-binance/v2"
)

func TestSpotTickerSnapshotFreshness(t *testing.T) {
	spotTickerSnapshot.Lock()
	oldItems := spotTickerSnapshot.Items
	spotTickerSnapshot.Items = make(map[string]spotTickerSnapshotEntry)
	spotTickerSnapshot.Unlock()
	oldUpdateAt := spotTickerLastUpdateAt.Load()
	t.Cleanup(func() {
		spotTickerSnapshot.Lock()
		spotTickerSnapshot.Items = oldItems
		spotTickerSnapshot.Unlock()
		spotTickerLastUpdateAt.Store(oldUpdateAt)
	})

	now := time.Now()
	storeSpotTickerEvents(spotbinance.WsAllMarketsStatEvent{
		&spotbinance.WsMarketStatEvent{Symbol: "ETHUSDT", LastPrice: "25"},
	}, now)
	price, ok := GetFreshSpotTickerPrice("ethusdt")
	if !ok || price != "25" {
		t.Fatalf("fresh spot ticker ok=%v price=%q", ok, price)
	}
	if !SpotTickerWSFresh() {
		t.Fatal("fresh spot all-market event must mark WS fresh")
	}

	spotTickerSnapshot.Lock()
	entry := spotTickerSnapshot.Items["ETHUSDT"]
	entry.ReceivedAt = now.Add(-10 * time.Second).UnixMilli()
	spotTickerSnapshot.Items["ETHUSDT"] = entry
	spotTickerSnapshot.Unlock()
	if _, ok := GetFreshSpotTickerPrice("ETHUSDT"); ok {
		t.Fatal("stale spot ticker must not be served")
	}
}

func TestSpotTickerRESTFallbackCache(t *testing.T) {
	spotTickerRESTCache.Lock()
	old := spotTickerRESTCache.items
	spotTickerRESTCache.items = make(map[string]spotTickerRESTCacheEntry)
	spotTickerRESTCache.Unlock()
	t.Cleanup(func() {
		spotTickerRESTCache.Lock()
		spotTickerRESTCache.items = old
		spotTickerRESTCache.Unlock()
	})

	var calls atomic.Int32
	loader := func(context.Context) ([]*spotbinance.SymbolPrice, error) {
		calls.Add(1)
		return []*spotbinance.SymbolPrice{{Symbol: "ETHUSDT", Price: "20"}}, nil
	}
	first, err := getSpotTickerRESTCached(context.Background(), "ETHUSDT", loader)
	if err != nil {
		t.Fatal(err)
	}
	second, err := getSpotTickerRESTCached(context.Background(), "ETHUSDT", loader)
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("spot ticker REST loader calls=%d want=1", calls.Load())
	}
	first[0].Price = "mutated"
	if second[0].Price != "20" {
		t.Fatalf("spot ticker cache leaked caller mutation: %+v", second)
	}
}

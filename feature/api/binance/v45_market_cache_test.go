package binance

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/adshao/go-binance/v2/futures"
)

func TestMarkPriceSnapshotUsesFreshAllMarketEvent(t *testing.T) {
	futuresMarkPriceData.Lock()
	oldItems := futuresMarkPriceData.Items
	futuresMarkPriceData.Items = make(map[string]markPriceSnapshotEntry)
	futuresMarkPriceData.Unlock()
	oldFullAt := futuresMarkPriceLastFullAt.Load()
	defer func() {
		futuresMarkPriceData.Lock()
		futuresMarkPriceData.Items = oldItems
		futuresMarkPriceData.Unlock()
		futuresMarkPriceLastFullAt.Store(oldFullAt)
	}()

	now := time.Now()
	storeFuturesMarkPriceEvents(futures.WsAllMarkPriceEvent{
		&futures.WsMarkPriceEvent{Symbol: "BTCUSDT", MarkPrice: "100", FundingRate: "0.001", Time: now.UnixMilli()},
		&futures.WsMarkPriceEvent{Symbol: "ETHUSDT", MarkPrice: "10", FundingRate: "-0.002", Time: now.UnixMilli()},
	}, now)

	rows, ok := loadFreshFuturesPremiumIndexSnapshot("", time.Second)
	if !ok || len(rows) != 2 {
		t.Fatalf("bulk snapshot ok=%v rows=%+v", ok, rows)
	}
	one, ok := loadFreshFuturesPremiumIndexSnapshot("BTCUSDT", time.Second)
	if !ok || len(one) != 1 || one[0].MarkPrice != "100" || one[0].LastFundingRate != "0.001" {
		t.Fatalf("single snapshot ok=%v rows=%+v", ok, one)
	}

	futuresMarkPriceData.Lock()
	entry := futuresMarkPriceData.Items["BTCUSDT"]
	entry.ReceivedAt = now.Add(-10 * time.Second).UnixMilli()
	futuresMarkPriceData.Items["BTCUSDT"] = entry
	futuresMarkPriceData.Unlock()
	if _, ok := loadFreshFuturesPremiumIndexSnapshot("BTCUSDT", time.Second); ok {
		t.Fatal("stale mark-price snapshot must fail closed")
	}
}

func TestLiveKlineWSEventUpdatesCurrentAndAppendsNewBar(t *testing.T) {
	key := liveKlineCacheKey("BTCUSDT", "1m", 2)
	liveKlineCacheMu.Lock()
	old := liveKlineCache
	liveKlineCache = map[string]liveKlineCacheEntry{
		key: {
			rows: []*futures.Kline{
				{OpenTime: 60_000, CloseTime: 119_999, Open: "2", Close: "2"},
				{OpenTime: 0, CloseTime: 59_999, Open: "1", Close: "1"},
			},
			expiresAt: time.Now().Add(time.Minute),
			maxLimit:  2,
		},
	}
	liveKlineCacheMu.Unlock()
	defer func() {
		liveKlineCacheMu.Lock()
		liveKlineCache = old
		liveKlineCacheMu.Unlock()
	}()

	now := time.Now()
	applyLiveKlineWSEvent(&futures.WsKlineEvent{
		Symbol: "BTCUSDT",
		Kline: futures.WsKline{
			StartTime: 60_000, EndTime: 119_999, Interval: "1m",
			Open: "2", High: "2.5", Low: "1.9", Close: "2.4",
		},
	}, now)
	rows, _, ok, wsFresh := loadLiveKlineCache(key, 2)
	if !ok || !wsFresh || rows[0].Close != "2.4" {
		t.Fatalf("current bar update failed: ok=%v ws=%v rows=%+v", ok, wsFresh, rows)
	}

	applyLiveKlineWSEvent(&futures.WsKlineEvent{
		Symbol: "BTCUSDT",
		Kline: futures.WsKline{
			StartTime: 120_000, EndTime: 179_999, Interval: "1m",
			Open: "2.4", High: "2.6", Low: "2.3", Close: "2.5",
		},
	}, now.Add(time.Millisecond))
	rows, _, ok, wsFresh = loadLiveKlineCache(key, 2)
	if !ok || !wsFresh || len(rows) != 2 || rows[0].OpenTime != 120_000 || rows[1].OpenTime != 60_000 {
		t.Fatalf("new bar append failed: %+v", rows)
	}
}

func TestLiveKlineWSGapExpiresCacheForRESTBootstrap(t *testing.T) {
	key := liveKlineCacheKey("ETHUSDT", "1m", 2)
	liveKlineCacheMu.Lock()
	old := liveKlineCache
	liveKlineCache = map[string]liveKlineCacheEntry{
		key: {
			rows:      []*futures.Kline{{OpenTime: 60_000, CloseTime: 119_999, Close: "2"}},
			expiresAt: time.Now().Add(time.Minute),
			maxLimit:  2,
		},
	}
	liveKlineCacheMu.Unlock()
	defer func() {
		liveKlineCacheMu.Lock()
		liveKlineCache = old
		liveKlineCacheMu.Unlock()
	}()

	applyLiveKlineWSEvent(&futures.WsKlineEvent{
		Symbol: "ETHUSDT",
		Kline: futures.WsKline{
			StartTime: 180_000, EndTime: 239_999, Interval: "1m", Close: "3",
		},
	}, time.Now())

	if _, _, ok, _ := loadLiveKlineCache(key, 2); ok {
		t.Fatal("gap must expire incremental cache so next caller REST-bootstraps")
	}
}

func resetIndicatorCachesForTest() {
	openInterestCacheMu.Lock()
	openInterestCache = make(map[string]openInterestCacheEntry)
	openInterestCacheMu.Unlock()
	openInterestStatsCacheMu.Lock()
	openInterestStatsCache = make(map[string]openInterestStatsCacheEntry)
	openInterestStatsCacheMu.Unlock()
	takerRatioCacheMu.Lock()
	takerRatioCache = make(map[string]takerRatioCacheEntry)
	takerRatioCacheMu.Unlock()
}

func TestOpenInterestCacheClonesAndCoalesces(t *testing.T) {
	resetIndicatorCachesForTest()
	var calls atomic.Int32
	loader := func(context.Context) (*futures.OpenInterest, error) {
		calls.Add(1)
		time.Sleep(20 * time.Millisecond)
		return &futures.OpenInterest{Symbol: "BTCUSDT", OpenInterest: "123"}, nil
	}

	const workers = 8
	results := make([]*futures.OpenInterest, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			row, err := getOpenInterestCached(context.Background(), "BTCUSDT", loader)
			if err != nil {
				t.Errorf("worker %d: %v", index, err)
				return
			}
			results[index] = row
		}(i)
	}
	wg.Wait()
	if got := calls.Load(); got != 1 {
		t.Fatalf("REST loader calls=%d want=1", got)
	}
	results[0].OpenInterest = "mutated"
	row, err := getOpenInterestCached(context.Background(), "BTCUSDT", loader)
	if err != nil {
		t.Fatal(err)
	}
	if row.OpenInterest != "123" {
		t.Fatalf("cached value leaked caller mutation: %+v", row)
	}
}

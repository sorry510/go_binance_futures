package binance

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/adshao/go-binance/v2/futures"
)

func TestSpotQuantMarketSnapshotCacheIsBoundedAndEnvironmentIsolated(t *testing.T) {
	mainnet, testnet := newSpotQuantMarketCache(), newSpotQuantMarketCache()
	var mainCalls, testCalls atomic.Int32
	loadMain := func() (SpotMarketSnapshot, error) {
		mainCalls.Add(1)
		return SpotMarketSnapshot{Symbol: "BTCUSDT", Price: "100", AsOf: time.Now()}, nil
	}
	loadTest := func() (SpotMarketSnapshot, error) {
		testCalls.Add(1)
		return SpotMarketSnapshot{Symbol: "BTCUSDT", Price: "101", AsOf: time.Now()}, nil
	}
	for i := 0; i < 4; i++ {
		snapshot, err := mainnet.snapshot("BTCUSDT", loadMain)
		if err != nil || snapshot.Price != "100" {
			t.Fatalf("mainnet snapshot: %+v %v", snapshot, err)
		}
		snapshot, err = testnet.snapshot("BTCUSDT", loadTest)
		if err != nil || snapshot.Price != "101" {
			t.Fatalf("testnet snapshot contaminated by mainnet: %+v %v", snapshot, err)
		}
	}
	if mainCalls.Load() != 1 || testCalls.Load() != 1 {
		t.Fatalf("cache missed: main=%d test=%d", mainCalls.Load(), testCalls.Load())
	}
	mainnet.mu.Lock()
	entry := mainnet.snapshots["BTCUSDT"]
	entry.until = time.Now().Add(-time.Second)
	mainnet.snapshots["BTCUSDT"] = entry
	mainnet.mu.Unlock()
	if _, err := mainnet.snapshot("BTCUSDT", loadMain); err != nil || mainCalls.Load() != 2 {
		t.Fatalf("expired snapshot not reloaded: %v calls=%d", err, mainCalls.Load())
	}
	if testCalls.Load() != 1 {
		t.Fatal("mainnet expiry touched testnet cache")
	}
}

func TestSpotQuantMarketCacheDoesNotCacheFailures(t *testing.T) {
	cache := newSpotQuantMarketCache()
	var calls int
	fetch := func() (SpotMarketSnapshot, error) {
		calls++
		if calls == 1 {
			return SpotMarketSnapshot{}, errors.New("temporary API failure")
		}
		return SpotMarketSnapshot{Symbol: "BTCUSDT", Price: "100"}, nil
	}
	if _, err := cache.snapshot("BTCUSDT", fetch); err == nil {
		t.Fatal("first fetch failure ignored")
	}
	if _, err := cache.snapshot("BTCUSDT", fetch); err != nil || calls != 2 {
		t.Fatalf("cached failed response: err=%v calls=%d", err, calls)
	}
}

func TestSpotQuantMarketCacheConcurrentRequestsCoalesced(t *testing.T) {
	cache := newSpotQuantMarketCache()
	var calls atomic.Int32
	const parallel = 24
	var wg sync.WaitGroup
	errs := make(chan error, parallel)
	for i := 0; i < parallel; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			snap, err := cache.snapshot("ETHUSDT", func() (SpotMarketSnapshot, error) {
				calls.Add(1)
				time.Sleep(20 * time.Millisecond)
				return SpotMarketSnapshot{Symbol: "ETHUSDT", Price: "100"}, nil
			})
			if err != nil {
				errs <- err
			} else if snap.Symbol != "ETHUSDT" {
				errs <- errors.New("cross-symbol cache value")
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("concurrent identical snapshot made %d calls", calls.Load())
	}
}

func TestSpotQuantMarketKlinesCacheClonesAndDoesNotCrossInterval(t *testing.T) {
	cache := newSpotQuantMarketCache()
	var calls int
	fetch := func() ([]*futures.Kline, error) {
		calls++
		return []*futures.Kline{{OpenTime: time.Now().UnixMilli(), CloseTime: time.Now().Add(time.Minute).UnixMilli(), Close: "100"}}, nil
	}
	first, err := cache.candles("BTCUSDT", "1m", 1, fetch)
	if err != nil {
		t.Fatal(err)
	}
	first[0].Close = "999"
	second, err := cache.candles("BTCUSDT", "1m", 1, fetch)
	if err != nil || len(second) != 1 || second[0].Close != "100" || calls != 1 {
		t.Fatalf("cache reused mutable Kline: %+v calls=%d err=%v", second, calls, err)
	}
	_, _ = cache.candles("BTCUSDT", "5m", 1, fetch)
	_, _ = cache.candles("BTCUSDT", "1m", 2, fetch)
	if calls != 3 {
		t.Fatalf("different interval/limit not isolated: calls=%d", calls)
	}
}

func TestSpotQuantMarketKlinesNeverReuseAfterCandleClose(t *testing.T) {
	cache := newSpotQuantMarketCache()
	var calls int
	fetch := func() ([]*futures.Kline, error) {
		calls++
		return []*futures.Kline{{OpenTime: time.Now().Add(-time.Minute).UnixMilli(), CloseTime: time.Now().Add(-time.Second).UnixMilli(), Close: "100"}}, nil
	}
	for i := 0; i < 3; i++ {
		if _, err := cache.candles("BTCUSDT", "1m", 1, fetch); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 3 {
		t.Fatalf("cached completed/stale candle: %d calls", calls)
	}
	if len(cache.klines) != 0 {
		t.Fatal("past candle retained as fresh")
	}
}

func TestSpotQuantMarketKlinesCacheOnlyThisEnvironment(t *testing.T) {
	mainnet, testnet := newSpotQuantMarketCache(), newSpotQuantMarketCache()
	fetch := func(price string) func() ([]*futures.Kline, error) {
		return func() ([]*futures.Kline, error) {
			return []*futures.Kline{{CloseTime: time.Now().Add(time.Minute).UnixMilli(), Close: price}}, nil
		}
	}
	main, err := mainnet.candles("BTCUSDT", "1m", 1, fetch("100"))
	if err != nil {
		t.Fatal(err)
	}
	test, err := testnet.candles("BTCUSDT", "1m", 1, fetch("50"))
	if err != nil {
		t.Fatal(err)
	}
	if main[0].Close != "100" || test[0].Close != "50" {
		t.Fatal("mixed market environments")
	}
}

func TestSpotQuantMarketSnapshotLRUKeepsRecentlyUsed(t *testing.T) {
	c := newSpotQuantMarketCache()
	now := time.Now()
	for i := 0; i < spotQuantCacheMax; i++ {
		key := fmt.Sprintf("TOKEN%03dUSDT", i)
		c.snapshots[key] = spotQuantSnapshotEntry{
			value:    SpotMarketSnapshot{Symbol: key, Price: "100"},
			until:    now.Add(time.Minute),
			lastUsed: now.Add(-time.Duration(spotQuantCacheMax-i) * time.Millisecond),
		}
	}
	_, err := c.snapshot("TOKEN000USDT", func() (SpotMarketSnapshot, error) {
		t.Fatal("cache failed to serve hot snapshot")
		return SpotMarketSnapshot{}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.snapshot("NEWUSDT", func() (SpotMarketSnapshot, error) {
		return SpotMarketSnapshot{Symbol: "NEWUSDT", Price: "100"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(c.snapshots) != spotQuantCacheMax {
		t.Fatalf("cache exceeds bounded size: %d", len(c.snapshots))
	}
	if _, found := c.snapshots["TOKEN000USDT"]; !found {
		t.Fatal("evicted recently accessed snapshot")
	}
	if _, found := c.snapshots["TOKEN001USDT"]; found {
		t.Fatal("oldest snapshot was not evicted")
	}
}

func TestSpotQuantMarketKlineLRUKeepsRecentlyUsed(t *testing.T) {
	c := newSpotQuantMarketCache()
	now := time.Now()
	for i := 0; i < spotQuantCacheMax; i++ {
		key := fmt.Sprintf("TOKEN%03dUSDT|1m|1", i)
		c.klines[key] = spotQuantKlineEntry{
			value:    []*futures.Kline{{Close: "100", CloseTime: now.Add(time.Minute).UnixMilli()}},
			until:    now.Add(time.Minute),
			lastUsed: now.Add(-time.Duration(spotQuantCacheMax-i) * time.Millisecond),
		}
	}
	if _, err := c.candles("TOKEN000USDT", "1m", 1, func() ([]*futures.Kline, error) {
		t.Fatal("cache failed to serve hot Kline")
		return nil, nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.candles("NEWUSDT", "1m", 1, func() ([]*futures.Kline, error) {
		return []*futures.Kline{{Close: "100", CloseTime: now.Add(time.Minute).UnixMilli()}}, nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(c.klines) != spotQuantCacheMax {
		t.Fatalf("Kline cache exceeds bounded size: %d", len(c.klines))
	}
	if _, found := c.klines["TOKEN000USDT|1m|1"]; !found {
		t.Fatal("evicted recently used Kline")
	}
	if _, found := c.klines["TOKEN001USDT|1m|1"]; found {
		t.Fatal("oldest Kline was not evicted")
	}
}

package binance

import (
	"testing"
	"time"

	spotbinance "github.com/adshao/go-binance/v2"
)

func withSpotKlineCache(t *testing.T, entries map[string]spotLiveKlineCacheEntry) {
	t.Helper()
	spotLiveKlineCacheMu.Lock()
	old := spotLiveKlineCache
	spotLiveKlineCache = entries
	spotLiveKlineCacheMu.Unlock()
	t.Cleanup(func() {
		spotLiveKlineCacheMu.Lock()
		spotLiveKlineCache = old
		spotLiveKlineCacheMu.Unlock()
	})
}

func TestSpotLiveKlineWSEventUpdatesCurrentAndAppendsNewBar(t *testing.T) {
	key := spotLiveKlineCacheKey("BTCUSDT", "1m")
	withSpotKlineCache(t, map[string]spotLiveKlineCacheEntry{
		key: {
			rows: []*spotbinance.Kline{
				{OpenTime: 60_000, CloseTime: 119_999, Open: "2", Close: "2"},
				{OpenTime: 0, CloseTime: 59_999, Open: "1", Close: "1"},
			},
			expiresAt: time.Now().Add(time.Minute),
			maxLimit:  2,
		},
	})

	now := time.Now()
	applySpotLiveKlineWSEvent(&spotbinance.WsKlineEvent{
		Symbol: "BTCUSDT",
		Kline: spotbinance.WsKline{
			StartTime: 60_000, EndTime: 119_999, Interval: "1m",
			Open: "2", High: "2.5", Low: "1.9", Close: "2.4",
		},
	}, now)
	rows, _, ok, wsFresh := loadSpotLiveKlineCache(key, 2)
	if !ok || !wsFresh || len(rows) != 2 || rows[0].Close != "2.4" {
		t.Fatalf("current bar update failed: ok=%v ws=%v rows=%+v", ok, wsFresh, rows)
	}

	applySpotLiveKlineWSEvent(&spotbinance.WsKlineEvent{
		Symbol: "BTCUSDT",
		Kline: spotbinance.WsKline{
			StartTime: 120_000, EndTime: 179_999, Interval: "1m",
			Open: "2.4", High: "2.6", Low: "2.3", Close: "2.5",
		},
	}, now.Add(time.Millisecond))
	rows, _, ok, wsFresh = loadSpotLiveKlineCache(key, 2)
	if !ok || !wsFresh || len(rows) != 2 || rows[0].OpenTime != 120_000 || rows[1].OpenTime != 60_000 {
		t.Fatalf("new bar append failed: %+v", rows)
	}
}

func TestSpotLiveKlineWSGapExpiresCache(t *testing.T) {
	key := spotLiveKlineCacheKey("ETHUSDT", "1m")
	withSpotKlineCache(t, map[string]spotLiveKlineCacheEntry{
		key: {
			rows:      []*spotbinance.Kline{{OpenTime: 60_000, CloseTime: 119_999, Close: "2"}},
			expiresAt: time.Now().Add(time.Minute),
			maxLimit:  2,
		},
	})

	applySpotLiveKlineWSEvent(&spotbinance.WsKlineEvent{
		Symbol: "ETHUSDT",
		Kline: spotbinance.WsKline{
			StartTime: 180_000, EndTime: 239_999, Interval: "1m", Close: "3",
		},
	}, time.Now())
	if _, _, ok, _ := loadSpotLiveKlineCache(key, 2); ok {
		t.Fatal("gap must expire spot incremental cache so next caller REST-bootstraps")
	}
}

func TestSpotLiveKlineCanonicalCacheServesSmallerLimit(t *testing.T) {
	key := spotLiveKlineCacheKey("SOLUSDT", "5m")
	rows := make([]*spotbinance.Kline, 10)
	for i := range rows {
		rows[i] = &spotbinance.Kline{OpenTime: int64(10 - i)}
	}
	withSpotKlineCache(t, map[string]spotLiveKlineCacheEntry{
		key: {rows: rows, expiresAt: time.Now().Add(time.Minute), maxLimit: 10},
	})
	got, _, ok, _ := loadSpotLiveKlineCache(key, 3)
	if !ok || len(got) != 3 || got[0].OpenTime != 10 || got[2].OpenTime != 8 {
		t.Fatalf("smaller limit should reuse canonical cache: %+v", got)
	}
}

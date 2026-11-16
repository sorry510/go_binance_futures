package binance

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"golang.org/x/sync/singleflight"
)

// This cache belongs to ONE QuantAdapter and therefore ONE verified Binance
// account environment. Never use the shared mainnet WS/Kline cache for Testnet.
const (
	spotQuantSnapshotTTL = 4 * time.Second
	spotQuantKlineTTL    = 7 * time.Second
	spotQuantCacheMax    = 256
)

type spotQuantSnapshotEntry struct {
	value    SpotMarketSnapshot
	until    time.Time
	lastUsed time.Time
}

type spotQuantKlineEntry struct {
	value    []*futures.Kline
	until    time.Time
	lastUsed time.Time
}

type spotQuantMarketCache struct {
	mu        sync.Mutex
	snapshots map[string]spotQuantSnapshotEntry
	klines    map[string]spotQuantKlineEntry
	snapSF    singleflight.Group
	klineSF   singleflight.Group
}

func newSpotQuantMarketCache() *spotQuantMarketCache {
	return &spotQuantMarketCache{
		snapshots: make(map[string]spotQuantSnapshotEntry),
		klines:    make(map[string]spotQuantKlineEntry),
	}
}

func (c *spotQuantMarketCache) loadSnapshot(symbol string) (SpotMarketSnapshot, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	item, ok := c.snapshots[symbol]
	if !ok || !time.Now().Before(item.until) {
		return SpotMarketSnapshot{}, false
	}
	item.lastUsed = time.Now()
	c.snapshots[symbol] = item
	return item.value, true
}

func (c *spotQuantMarketCache) loadKlines(key string) ([]*futures.Kline, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	item, ok := c.klines[key]
	if !ok || !time.Now().Before(item.until) {
		return nil, false
	}
	item.lastUsed = time.Now()
	c.klines[key] = item
	return cloneSpotQuantKlines(item.value), true
}

// Keep hot symbols when bounded caches fill. Expired entries are preferred
// for eviction, otherwise remove the least recently accessed entry.
func evictOldestSnapshot(entries map[string]spotQuantSnapshotEntry) {
	var oldestKey string
	var oldest time.Time
	now := time.Now()
	for key, item := range entries {
		if !item.until.After(now) {
			delete(entries, key)
			return
		}
		if oldestKey == "" || item.lastUsed.Before(oldest) {
			oldestKey, oldest = key, item.lastUsed
		}
	}
	if oldestKey != "" {
		delete(entries, oldestKey)
	}
}

func evictOldestKlines(entries map[string]spotQuantKlineEntry) {
	var oldestKey string
	var oldest time.Time
	now := time.Now()
	for key, item := range entries {
		if !item.until.After(now) {
			delete(entries, key)
			return
		}
		if oldestKey == "" || item.lastUsed.Before(oldest) {
			oldestKey, oldest = key, item.lastUsed
		}
	}
	if oldestKey != "" {
		delete(entries, oldestKey)
	}
}

func (c *spotQuantMarketCache) snapshot(symbol string, fetch func() (SpotMarketSnapshot, error)) (SpotMarketSnapshot, error) {
	if c == nil {
		return fetch()
	}
	if tick, ok := c.loadSnapshot(symbol); ok {
		return tick, nil
	}
	value, err, _ := c.snapSF.Do(symbol, func() (interface{}, error) {
		if tick, ok := c.loadSnapshot(symbol); ok {
			return tick, nil
		}
		tick, err := fetch()
		if err != nil {
			return nil, err // never cache a failed or stale price response
		}
		c.mu.Lock()
		now := time.Now()
		c.snapshots[symbol] = spotQuantSnapshotEntry{value: tick, until: now.Add(spotQuantSnapshotTTL), lastUsed: now}
		if len(c.snapshots) > spotQuantCacheMax {
			evictOldestSnapshot(c.snapshots)
		}
		c.mu.Unlock()
		return tick, nil
	})
	if err != nil {
		return SpotMarketSnapshot{}, err
	}
	return value.(SpotMarketSnapshot), nil
}

func cloneSpotQuantKlines(rows []*futures.Kline) []*futures.Kline {
	out := make([]*futures.Kline, len(rows))
	for i, row := range rows {
		if row != nil {
			copy := *row
			out[i] = &copy
		}
	}
	return out
}

func (c *spotQuantMarketCache) candles(symbol, interval string, limit int, fetch func() ([]*futures.Kline, error)) ([]*futures.Kline, error) {
	if c == nil {
		return fetch()
	}
	key := symbol + "|" + interval + "|" + strconv.Itoa(limit)
	if rows, ok := c.loadKlines(key); ok {
		return rows, nil
	}
	value, err, _ := c.klineSF.Do(key, func() (interface{}, error) {
		if rows, ok := c.loadKlines(key); ok {
			return rows, nil
		}
		rows, err := fetch()
		if err != nil {
			return nil, err
		}
		if len(rows) == 0 || rows[len(rows)-1] == nil {
			return nil, fmt.Errorf("empty Spot Quant market candles")
		}
		ttl := spotQuantKlineTTL
		// Never reuse the current bar after the exchange candle boundary.
		// Latest candles from REST are ascending by OpenTime.
		remaining := time.Until(time.UnixMilli(rows[len(rows)-1].CloseTime + 1))
		if remaining < ttl {
			ttl = remaining
		}
		immutable := cloneSpotQuantKlines(rows)
		if ttl > 0 {
			c.mu.Lock()
			now := time.Now()
			c.klines[key] = spotQuantKlineEntry{value: immutable, until: now.Add(ttl), lastUsed: now}
			if len(c.klines) > spotQuantCacheMax {
				evictOldestKlines(c.klines)
			}
			c.mu.Unlock()
		}
		return immutable, nil
	})
	if err != nil {
		return nil, err
	}
	return cloneSpotQuantKlines(value.([]*futures.Kline)), nil
}

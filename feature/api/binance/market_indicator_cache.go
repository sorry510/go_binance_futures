package binance

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"go_binance_futures/service/binanceapiusage"

	"github.com/adshao/go-binance/v2/futures"
	"golang.org/x/sync/singleflight"
)

const (
	openInterestCacheTTL      = 2 * time.Second
	marketRatioCacheTTL       = 30 * time.Second
	marketIndicatorMaxEntries = 256
)

type openInterestCacheEntry struct {
	data      *futures.OpenInterest
	expiresAt time.Time
}

type openInterestStatsCacheEntry struct {
	data      []*futures.OpenInterestStatistic
	expiresAt time.Time
}

type takerRatioCacheEntry struct {
	data      []*futures.TakerLongShortRatio
	expiresAt time.Time
}

var (
	openInterestCacheMu sync.Mutex
	openInterestCache   = make(map[string]openInterestCacheEntry)
	openInterestSF      singleflight.Group

	openInterestStatsCacheMu sync.Mutex
	openInterestStatsCache   = make(map[string]openInterestStatsCacheEntry)
	openInterestStatsSF      singleflight.Group

	takerRatioCacheMu sync.Mutex
	takerRatioCache   = make(map[string]takerRatioCacheEntry)
	takerRatioSF      singleflight.Group
)

func cloneOpenInterest(data *futures.OpenInterest) *futures.OpenInterest {
	if data == nil {
		return nil
	}
	out := *data
	return &out
}

func cloneOpenInterestStats(data []*futures.OpenInterestStatistic) []*futures.OpenInterestStatistic {
	out := make([]*futures.OpenInterestStatistic, 0, len(data))
	for _, row := range data {
		if row == nil {
			out = append(out, nil)
			continue
		}
		copied := *row
		out = append(out, &copied)
	}
	return out
}

func cloneTakerRatios(data []*futures.TakerLongShortRatio) []*futures.TakerLongShortRatio {
	out := make([]*futures.TakerLongShortRatio, 0, len(data))
	for _, row := range data {
		if row == nil {
			out = append(out, nil)
			continue
		}
		copied := *row
		out = append(out, &copied)
	}
	return out
}

func getOpenInterestCached(ctx context.Context, symbol string, loader func(context.Context) (*futures.OpenInterest, error)) (*futures.OpenInterest, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	source := binanceapiusage.SourceFromContext(ctx)
	now := time.Now()

	openInterestCacheMu.Lock()
	entry, ok := openInterestCache[symbol]
	if ok && entry.expiresAt.After(now) {
		data := cloneOpenInterest(entry.data)
		openInterestCacheMu.Unlock()
		binanceapiusage.RecordOptimization(source, "cache_hit", 1)
		binanceapiusage.RecordOptimization(source, "prevented_duplicate", 1)
		return data, nil
	}
	if ok {
		delete(openInterestCache, symbol)
	}
	openInterestCacheMu.Unlock()

	value, err, shared := openInterestSF.Do(symbol, func() (interface{}, error) {
		data, loadErr := loader(ctx)
		if loadErr == nil && data != nil {
			openInterestCacheMu.Lock()
			evictOldestOpenInterestLocked(now)
			openInterestCache[symbol] = openInterestCacheEntry{data: cloneOpenInterest(data), expiresAt: time.Now().Add(openInterestCacheTTL)}
			openInterestCacheMu.Unlock()
		}
		return data, loadErr
	})
	if shared {
		binanceapiusage.RecordOptimization(source, "coalesced", 1)
		binanceapiusage.RecordOptimization(source, "prevented_duplicate", 1)
	}
	if err != nil {
		return nil, err
	}
	data, _ := value.(*futures.OpenInterest)
	return cloneOpenInterest(data), nil
}

func getOpenInterestStatsCached(ctx context.Context, symbol, period string, limit int, loader func(context.Context) ([]*futures.OpenInterestStatistic, error)) ([]*futures.OpenInterestStatistic, error) {
	key := strings.ToUpper(strings.TrimSpace(symbol)) + "|" + strings.TrimSpace(period) + "|" + strconv.Itoa(limit)
	source := binanceapiusage.SourceFromContext(ctx)
	now := time.Now()

	openInterestStatsCacheMu.Lock()
	entry, ok := openInterestStatsCache[key]
	if ok && entry.expiresAt.After(now) {
		data := cloneOpenInterestStats(entry.data)
		openInterestStatsCacheMu.Unlock()
		binanceapiusage.RecordOptimization(source, "cache_hit", 1)
		binanceapiusage.RecordOptimization(source, "prevented_duplicate", 1)
		return data, nil
	}
	if ok {
		delete(openInterestStatsCache, key)
	}
	openInterestStatsCacheMu.Unlock()

	value, err, shared := openInterestStatsSF.Do(key, func() (interface{}, error) {
		data, loadErr := loader(ctx)
		if loadErr == nil {
			openInterestStatsCacheMu.Lock()
			evictOldestOpenInterestStatsLocked(now)
			openInterestStatsCache[key] = openInterestStatsCacheEntry{data: cloneOpenInterestStats(data), expiresAt: time.Now().Add(marketRatioCacheTTL)}
			openInterestStatsCacheMu.Unlock()
		}
		return data, loadErr
	})
	if shared {
		binanceapiusage.RecordOptimization(source, "coalesced", 1)
		binanceapiusage.RecordOptimization(source, "prevented_duplicate", 1)
	}
	if err != nil {
		return nil, err
	}
	data, _ := value.([]*futures.OpenInterestStatistic)
	return cloneOpenInterestStats(data), nil
}

func getTakerRatioCached(ctx context.Context, symbol, period string, limit uint32, loader func(context.Context) ([]*futures.TakerLongShortRatio, error)) ([]*futures.TakerLongShortRatio, error) {
	key := strings.ToUpper(strings.TrimSpace(symbol)) + "|" + strings.TrimSpace(period) + "|" + strconv.FormatUint(uint64(limit), 10)
	source := binanceapiusage.SourceFromContext(ctx)
	now := time.Now()

	takerRatioCacheMu.Lock()
	entry, ok := takerRatioCache[key]
	if ok && entry.expiresAt.After(now) {
		data := cloneTakerRatios(entry.data)
		takerRatioCacheMu.Unlock()
		binanceapiusage.RecordOptimization(source, "cache_hit", 1)
		binanceapiusage.RecordOptimization(source, "prevented_duplicate", 1)
		return data, nil
	}
	if ok {
		delete(takerRatioCache, key)
	}
	takerRatioCacheMu.Unlock()

	value, err, shared := takerRatioSF.Do(key, func() (interface{}, error) {
		data, loadErr := loader(ctx)
		if loadErr == nil {
			takerRatioCacheMu.Lock()
			evictOldestTakerRatioLocked(now)
			takerRatioCache[key] = takerRatioCacheEntry{data: cloneTakerRatios(data), expiresAt: time.Now().Add(marketRatioCacheTTL)}
			takerRatioCacheMu.Unlock()
		}
		return data, loadErr
	})
	if shared {
		binanceapiusage.RecordOptimization(source, "coalesced", 1)
		binanceapiusage.RecordOptimization(source, "prevented_duplicate", 1)
	}
	if err != nil {
		return nil, err
	}
	data, _ := value.([]*futures.TakerLongShortRatio)
	return cloneTakerRatios(data), nil
}

func evictOldestOpenInterestLocked(now time.Time) {
	for key, entry := range openInterestCache {
		if !entry.expiresAt.After(now) {
			delete(openInterestCache, key)
		}
	}
	if len(openInterestCache) < marketIndicatorMaxEntries {
		return
	}
	for key := range openInterestCache {
		delete(openInterestCache, key)
		break
	}
}

func evictOldestOpenInterestStatsLocked(now time.Time) {
	for key, entry := range openInterestStatsCache {
		if !entry.expiresAt.After(now) {
			delete(openInterestStatsCache, key)
		}
	}
	if len(openInterestStatsCache) < marketIndicatorMaxEntries {
		return
	}
	for key := range openInterestStatsCache {
		delete(openInterestStatsCache, key)
		break
	}
}

func evictOldestTakerRatioLocked(now time.Time) {
	for key, entry := range takerRatioCache {
		if !entry.expiresAt.After(now) {
			delete(takerRatioCache, key)
		}
	}
	if len(takerRatioCache) < marketIndicatorMaxEntries {
		return
	}
	for key := range takerRatioCache {
		delete(takerRatioCache, key)
		break
	}
}

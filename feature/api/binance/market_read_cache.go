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
	futuresDepthCacheTTL        = 750 * time.Millisecond
	futuresDepthCacheMaxEntries = 256
	futuresExchangeInfoTTL      = 12 * time.Hour
)

type depthCacheEntry struct {
	data      *futures.DepthResponse
	expiresAt time.Time
}

var futuresDepthCache = struct {
	sync.Mutex
	items map[string]depthCacheEntry
}{items: make(map[string]depthCacheEntry)}

var futuresDepthSF singleflight.Group

type futuresExchangeInfoCacheEntry struct {
	data      *futures.ExchangeInfo
	expiresAt time.Time
}

var (
	futuresExchangeInfoMu    sync.RWMutex
	futuresExchangeInfoCache futuresExchangeInfoCacheEntry
	futuresExchangeInfoSF    singleflight.Group
)

func cloneDepthResponse(data *futures.DepthResponse) *futures.DepthResponse {
	if data == nil {
		return nil
	}
	out := *data
	out.Bids = append([]futures.Bid(nil), data.Bids...)
	out.Asks = append([]futures.Ask(nil), data.Asks...)
	return &out
}

func futuresDepthCacheKey(symbol string, limit int) string {
	return strings.ToUpper(strings.TrimSpace(symbol)) + "|" + strconv.Itoa(limit)
}

func loadDepthCache(key string) (*futures.DepthResponse, bool) {
	now := time.Now()
	futuresDepthCache.Lock()
	defer futuresDepthCache.Unlock()
	entry, ok := futuresDepthCache.items[key]
	if !ok {
		return nil, false
	}
	if !entry.expiresAt.After(now) {
		delete(futuresDepthCache.items, key)
		return nil, false
	}
	return cloneDepthResponse(entry.data), true
}

func storeDepthCache(key string, data *futures.DepthResponse) {
	now := time.Now()
	futuresDepthCache.Lock()
	defer futuresDepthCache.Unlock()
	for cacheKey, entry := range futuresDepthCache.items {
		if !entry.expiresAt.After(now) {
			delete(futuresDepthCache.items, cacheKey)
		}
	}
	if len(futuresDepthCache.items) >= futuresDepthCacheMaxEntries {
		var oldestKey string
		var oldest time.Time
		for cacheKey, entry := range futuresDepthCache.items {
			if oldestKey == "" || entry.expiresAt.Before(oldest) {
				oldestKey, oldest = cacheKey, entry.expiresAt
			}
		}
		if oldestKey != "" {
			delete(futuresDepthCache.items, oldestKey)
		}
	}
	futuresDepthCache.items[key] = depthCacheEntry{
		data: cloneDepthResponse(data), expiresAt: now.Add(futuresDepthCacheTTL),
	}
}

func getDepthCached(ctx context.Context, symbol string, limit int, loader func(context.Context) (*futures.DepthResponse, error)) (*futures.DepthResponse, error) {
	key := futuresDepthCacheKey(symbol, limit)
	source := binanceapiusage.SourceFromContext(ctx)
	if data, ok := loadDepthCache(key); ok {
		binanceapiusage.RecordOptimization(source, "cache_hit", 1)
		binanceapiusage.RecordOptimization(source, "prevented_duplicate", 1)
		return data, nil
	}
	value, err, shared := futuresDepthSF.Do(key, func() (interface{}, error) {
		if data, ok := loadDepthCache(key); ok {
			return data, nil
		}
		data, loadErr := loader(ctx)
		if loadErr == nil && data != nil {
			storeDepthCache(key, data)
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
	data, _ := value.(*futures.DepthResponse)
	return cloneDepthResponse(data), nil
}

func loadFuturesExchangeInfoCache() (*futures.ExchangeInfo, bool) {
	futuresExchangeInfoMu.RLock()
	entry := futuresExchangeInfoCache
	futuresExchangeInfoMu.RUnlock()
	if entry.data == nil || !entry.expiresAt.After(time.Now()) {
		return nil, false
	}
	return entry.data, true
}

func storeFuturesExchangeInfoCache(data *futures.ExchangeInfo) {
	if data == nil {
		return
	}
	futuresExchangeInfoMu.Lock()
	futuresExchangeInfoCache = futuresExchangeInfoCacheEntry{
		data: data, expiresAt: time.Now().Add(futuresExchangeInfoTTL),
	}
	futuresExchangeInfoMu.Unlock()
}

func getFuturesExchangeInfoCached(ctx context.Context, loader func(context.Context) (*futures.ExchangeInfo, error)) (*futures.ExchangeInfo, error) {
	source := binanceapiusage.SourceFromContext(ctx)
	if data, ok := loadFuturesExchangeInfoCache(); ok {
		binanceapiusage.RecordOptimization(source, "cache_hit", 1)
		binanceapiusage.RecordOptimization(source, "prevented_duplicate", 1)
		return data, nil
	}
	value, err, shared := futuresExchangeInfoSF.Do("exchange_info", func() (interface{}, error) {
		if data, ok := loadFuturesExchangeInfoCache(); ok {
			return data, nil
		}
		data, loadErr := loader(ctx)
		if loadErr == nil {
			storeFuturesExchangeInfoCache(data)
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
	data, _ := value.(*futures.ExchangeInfo)
	return data, nil
}

func resetMarketReadCacheForTest() {
	futuresDepthCache.Lock()
	futuresDepthCache.items = make(map[string]depthCacheEntry)
	futuresDepthCache.Unlock()
	futuresExchangeInfoMu.Lock()
	futuresExchangeInfoCache = futuresExchangeInfoCacheEntry{}
	futuresExchangeInfoMu.Unlock()
}

package binance

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go_binance_futures/service/binanceapiusage"

	spotbinance "github.com/adshao/go-binance/v2"
	"golang.org/x/sync/singleflight"
)

const (
	spotTickerSnapshotMaxAge = 3 * time.Second
	spotExchangeInfoTTL      = 12 * time.Hour
)

type spotTickerSnapshotEntry struct {
	Price      string
	ReceivedAt int64
}

var spotTickerSnapshot = struct {
	sync.RWMutex
	Items map[string]spotTickerSnapshotEntry
}{Items: make(map[string]spotTickerSnapshotEntry)}
var spotTickerLastUpdateAt atomic.Int64

func SpotTickerWSFresh() bool {
	last := spotTickerLastUpdateAt.Load()
	return last > 0 && time.Since(time.UnixMilli(last)) <= 5*time.Second
}

type spotExchangeInfoCacheEntry struct {
	data      *spotbinance.ExchangeInfo
	expiresAt time.Time
}

var (
	spotExchangeInfoMu    sync.RWMutex
	spotExchangeInfoCache spotExchangeInfoCacheEntry
	spotExchangeInfoSF    singleflight.Group
)

func storeSpotTickerPrice(symbol, price string, receivedAt time.Time) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	price = strings.TrimSpace(price)
	if symbol == "" || price == "" {
		return
	}
	spotTickerLastUpdateAt.Store(receivedAt.UnixMilli())
	spotTickerSnapshot.Lock()
	spotTickerSnapshot.Items[symbol] = spotTickerSnapshotEntry{
		Price: price, ReceivedAt: receivedAt.UnixMilli(),
	}
	spotTickerSnapshot.Unlock()
}

func storeSpotTickerEvents(events spotbinance.WsAllMarketsStatEvent, receivedAt time.Time) {
	if len(events) == 0 {
		return
	}
	receivedMS := receivedAt.UnixMilli()
	stored := 0
	spotTickerSnapshot.Lock()
	for _, ticker := range events {
		if ticker == nil {
			continue
		}
		symbol := strings.ToUpper(strings.TrimSpace(ticker.Symbol))
		price := strings.TrimSpace(ticker.LastPrice)
		if symbol == "" || price == "" {
			continue
		}
		spotTickerSnapshot.Items[symbol] = spotTickerSnapshotEntry{
			Price: price, ReceivedAt: receivedMS,
		}
		stored++
	}
	spotTickerSnapshot.Unlock()
	if stored > 0 {
		spotTickerLastUpdateAt.Store(receivedMS)
	}
}

func GetFreshSpotTickerPrice(symbol string) (string, bool) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	spotTickerSnapshot.RLock()
	entry, ok := spotTickerSnapshot.Items[symbol]
	spotTickerSnapshot.RUnlock()
	if !ok || entry.ReceivedAt <= 0 || time.Since(time.UnixMilli(entry.ReceivedAt)) > spotTickerSnapshotMaxAge {
		return "", false
	}
	return entry.Price, true
}

func loadSpotExchangeInfoCache() (*spotbinance.ExchangeInfo, bool) {
	spotExchangeInfoMu.RLock()
	entry := spotExchangeInfoCache
	spotExchangeInfoMu.RUnlock()
	if entry.data == nil || !entry.expiresAt.After(time.Now()) {
		return nil, false
	}
	return entry.data, true
}

func storeSpotExchangeInfoCache(data *spotbinance.ExchangeInfo) {
	if data == nil {
		return
	}
	spotExchangeInfoMu.Lock()
	spotExchangeInfoCache = spotExchangeInfoCacheEntry{
		data: data, expiresAt: time.Now().Add(spotExchangeInfoTTL),
	}
	spotExchangeInfoMu.Unlock()
}

func getSpotExchangeInfoCached(ctx context.Context, loader func(context.Context) (*spotbinance.ExchangeInfo, error)) (*spotbinance.ExchangeInfo, error) {
	source := binanceapiusage.SourceFromContext(ctx)
	if data, ok := loadSpotExchangeInfoCache(); ok {
		binanceapiusage.RecordOptimization(source, "cache_hit", 1)
		binanceapiusage.RecordOptimization(source, "prevented_duplicate", 1)
		return data, nil
	}
	value, err, shared := spotExchangeInfoSF.Do("spot_exchange_info", func() (interface{}, error) {
		if data, ok := loadSpotExchangeInfoCache(); ok {
			return data, nil
		}
		data, loadErr := loader(ctx)
		if loadErr == nil {
			storeSpotExchangeInfoCache(data)
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
	data, _ := value.(*spotbinance.ExchangeInfo)
	return data, nil
}

const spotTickerRESTCacheTTL = time.Second

type spotTickerRESTCacheEntry struct {
	rows      []*spotbinance.SymbolPrice
	expiresAt time.Time
}

var spotTickerRESTCache = struct {
	sync.Mutex
	items map[string]spotTickerRESTCacheEntry
}{items: make(map[string]spotTickerRESTCacheEntry)}

var spotTickerRESTSF singleflight.Group

func cloneSpotSymbolPrices(rows []*spotbinance.SymbolPrice) []*spotbinance.SymbolPrice {
	out := make([]*spotbinance.SymbolPrice, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			out = append(out, nil)
			continue
		}
		copied := *row
		out = append(out, &copied)
	}
	return out
}

func getSpotTickerRESTCached(ctx context.Context, symbol string, loader func(context.Context) ([]*spotbinance.SymbolPrice, error)) ([]*spotbinance.SymbolPrice, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	source := binanceapiusage.SourceFromContext(ctx)
	now := time.Now()

	spotTickerRESTCache.Lock()
	entry, ok := spotTickerRESTCache.items[symbol]
	if ok && entry.expiresAt.After(now) {
		rows := cloneSpotSymbolPrices(entry.rows)
		spotTickerRESTCache.Unlock()
		binanceapiusage.RecordOptimization(source, "cache_hit", 1)
		binanceapiusage.RecordOptimization(source, "prevented_duplicate", 1)
		return rows, nil
	}
	if ok {
		delete(spotTickerRESTCache.items, symbol)
	}
	spotTickerRESTCache.Unlock()

	value, err, shared := spotTickerRESTSF.Do(symbol, func() (interface{}, error) {
		spotTickerRESTCache.Lock()
		entry, ok := spotTickerRESTCache.items[symbol]
		if ok && entry.expiresAt.After(time.Now()) {
			rows := cloneSpotSymbolPrices(entry.rows)
			spotTickerRESTCache.Unlock()
			return rows, nil
		}
		spotTickerRESTCache.Unlock()

		rows, loadErr := loader(ctx)
		if loadErr == nil {
			spotTickerRESTCache.Lock()
			if len(spotTickerRESTCache.items) >= 256 {
				for key, item := range spotTickerRESTCache.items {
					if !item.expiresAt.After(time.Now()) {
						delete(spotTickerRESTCache.items, key)
					}
				}
				if len(spotTickerRESTCache.items) >= 256 {
					for key := range spotTickerRESTCache.items {
						delete(spotTickerRESTCache.items, key)
						break
					}
				}
			}
			spotTickerRESTCache.items[symbol] = spotTickerRESTCacheEntry{
				rows: cloneSpotSymbolPrices(rows), expiresAt: time.Now().Add(spotTickerRESTCacheTTL),
			}
			spotTickerRESTCache.Unlock()
		}
		return rows, loadErr
	})
	if shared {
		binanceapiusage.RecordOptimization(source, "coalesced", 1)
		binanceapiusage.RecordOptimization(source, "prevented_duplicate", 1)
	}
	if err != nil {
		return nil, err
	}
	rows, _ := value.([]*spotbinance.SymbolPrice)
	return cloneSpotSymbolPrices(rows), nil
}

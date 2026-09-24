package binance

import (
	"context"
	"strings"
	"sync"
	"time"

	"go_binance_futures/service/binanceapiusage"

	"github.com/adshao/go-binance/v2/futures"
	"golang.org/x/sync/singleflight"
)

const (
	futuresTickerRESTCacheTTL = time.Second
	futuresTickerRESTCacheMax = 256
)

type futuresTickerRESTCacheEntry struct {
	rows      []*futures.SymbolPrice
	expiresAt time.Time
}

var futuresTickerRESTCache = struct {
	sync.Mutex
	items map[string]futuresTickerRESTCacheEntry
}{items: make(map[string]futuresTickerRESTCacheEntry)}

var futuresTickerRESTSF singleflight.Group

func cloneFuturesSymbolPrices(rows []*futures.SymbolPrice) []*futures.SymbolPrice {
	out := make([]*futures.SymbolPrice, 0, len(rows))
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

func getFuturesTickerRESTCached(ctx context.Context, symbol string, loader func(context.Context) ([]*futures.SymbolPrice, error)) ([]*futures.SymbolPrice, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	source := binanceapiusage.SourceFromContext(ctx)
	now := time.Now()

	futuresTickerRESTCache.Lock()
	entry, ok := futuresTickerRESTCache.items[symbol]
	if ok && entry.expiresAt.After(now) {
		rows := cloneFuturesSymbolPrices(entry.rows)
		futuresTickerRESTCache.Unlock()
		binanceapiusage.RecordOptimization(source, "cache_hit", 1)
		binanceapiusage.RecordOptimization(source, "prevented_duplicate", 1)
		return rows, nil
	}
	if ok {
		delete(futuresTickerRESTCache.items, symbol)
	}
	futuresTickerRESTCache.Unlock()

	value, err, shared := futuresTickerRESTSF.Do(symbol, func() (interface{}, error) {
		futuresTickerRESTCache.Lock()
		entry, ok := futuresTickerRESTCache.items[symbol]
		if ok && entry.expiresAt.After(time.Now()) {
			rows := cloneFuturesSymbolPrices(entry.rows)
			futuresTickerRESTCache.Unlock()
			return rows, nil
		}
		futuresTickerRESTCache.Unlock()

		rows, loadErr := loader(ctx)
		if loadErr == nil {
			futuresTickerRESTCache.Lock()
			if len(futuresTickerRESTCache.items) >= futuresTickerRESTCacheMax {
				for key, item := range futuresTickerRESTCache.items {
					if !item.expiresAt.After(time.Now()) {
						delete(futuresTickerRESTCache.items, key)
					}
				}
				if len(futuresTickerRESTCache.items) >= futuresTickerRESTCacheMax {
					for key := range futuresTickerRESTCache.items {
						delete(futuresTickerRESTCache.items, key)
						break
					}
				}
			}
			futuresTickerRESTCache.items[symbol] = futuresTickerRESTCacheEntry{
				rows: cloneFuturesSymbolPrices(rows), expiresAt: time.Now().Add(futuresTickerRESTCacheTTL),
			}
			futuresTickerRESTCache.Unlock()
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
	rows, _ := value.([]*futures.SymbolPrice)
	return cloneFuturesSymbolPrices(rows), nil
}

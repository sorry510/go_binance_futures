package binance

import (
	"context"
	"strings"
	"sync"
	"time"

	"go_binance_futures/service/binanceapiusage"

	"golang.org/x/sync/singleflight"
)

const spotAllPricesFallbackTTL = 10 * time.Second

var (
	spotAllPricesMu      sync.RWMutex
	spotAllPricesExpires time.Time
	spotAllPrices        map[string]string
	spotAllPricesSF      singleflight.Group
)

// GetSpotPricesBulkFallback fetches all spot prices in a single REST request
// only when DB/WS snapshots cannot supply some requested symbols. The result
// is cached and coalesced globally so a 3-second UI refresh cannot trigger one
// Binance REST request per listened coin.
func GetSpotPricesBulkFallback(ctx context.Context, symbols []string) (map[string]string, error) {
	if len(symbols) == 0 {
		return map[string]string{}, nil
	}
	wanted := make(map[string]struct{}, len(symbols))
	for _, name := range symbols {
		name = strings.ToUpper(strings.TrimSpace(name))
		if name != "" {
			wanted[name] = struct{}{}
		}
	}

	spotAllPricesMu.RLock()
	cached := spotAllPrices
	valid := cached != nil && time.Now().Before(spotAllPricesExpires)
	spotAllPricesMu.RUnlock()
	if !valid {
		value, err, _ := spotAllPricesSF.Do("all_spot_prices", func() (interface{}, error) {
			spotAllPricesMu.RLock()
			existing := spotAllPrices
			fresh := existing != nil && time.Now().Before(spotAllPricesExpires)
			spotAllPricesMu.RUnlock()
			if fresh {
				return existing, nil
			}
			contextWithSource := binanceapiusage.WithSource(ctx, "spot_listen_price_fallback")
			// No Symbol() means one bulk price snapshot, not N per-symbol requests.
			rows, fetchErr := client.NewListPricesService().Do(contextWithSource)
			if fetchErr != nil {
				return nil, fetchErr
			}
			prices := make(map[string]string, len(rows))
			for _, row := range rows {
				if row != nil && row.Symbol != "" && row.Price != "" {
					prices[row.Symbol] = row.Price
				}
			}
			spotAllPricesMu.Lock()
			spotAllPrices = prices
			spotAllPricesExpires = time.Now().Add(spotAllPricesFallbackTTL)
			spotAllPricesMu.Unlock()
			return prices, nil
		})
		if err != nil {
			return nil, err
		}
		cached = value.(map[string]string)
	}

	result := make(map[string]string, len(wanted))
	for symbol := range wanted {
		if price := cached[symbol]; price != "" && price != "0" {
			result[symbol] = price
		}
	}
	return result, nil
}

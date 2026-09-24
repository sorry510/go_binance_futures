package binance

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"go_binance_futures/service/binanceapiusage"

	"github.com/adshao/go-binance/v2/common"
	"github.com/adshao/go-binance/v2/futures"
	"golang.org/x/sync/singleflight"
)

const runtimeTradeConfigTTL = 5 * time.Second

type runtimeTradeConfigEntry struct {
	marginType futures.MarginType
	leverage   int
	expiresAt  time.Time
}

var runtimeTradeConfigCache = struct {
	sync.RWMutex
	items map[string]runtimeTradeConfigEntry
}{items: make(map[string]runtimeTradeConfigEntry)}

var runtimeTradeConfigSF singleflight.Group

func EnsureTradeConfigContext(ctx context.Context, symbol string, marginType futures.MarginType, leverage int) error {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" || leverage <= 0 {
		return fmt.Errorf("invalid futures trade config symbol=%q leverage=%d", symbol, leverage)
	}
	source := binanceapiusage.SourceFromContext(ctx)
	if hasRuntimeTradeConfig(symbol, marginType, leverage) {
		binanceapiusage.RecordOptimization(source, "cache_hit", 1)
		binanceapiusage.RecordOptimization(source, "prevented_duplicate", 2)
		return nil
	}

	key := symbol + "|" + string(marginType) + "|" + fmt.Sprintf("%d", leverage)
	_, err, shared := runtimeTradeConfigSF.Do(key, func() (interface{}, error) {
		if hasRuntimeTradeConfig(symbol, marginType, leverage) {
			return nil, nil
		}

		marginErr := SetMarginTypeContext(ctx, symbol, marginType)
		if marginErr != nil && !isMarginTypeAlreadyConfigured(marginErr) {
			return nil, marginErr
		}
		if _, leverageErr := SetLeverageContext(ctx, symbol, leverage); leverageErr != nil {
			return nil, leverageErr
		}
		runtimeTradeConfigCache.Lock()
		runtimeTradeConfigCache.items[symbol] = runtimeTradeConfigEntry{
			marginType: marginType,
			leverage:   leverage,
			expiresAt:  time.Now().Add(runtimeTradeConfigTTL),
		}
		runtimeTradeConfigCache.Unlock()
		return nil, nil
	})
	if shared {
		binanceapiusage.RecordOptimization(source, "coalesced", 1)
		binanceapiusage.RecordOptimization(source, "prevented_duplicate", 2)
	}
	return err
}

func hasRuntimeTradeConfig(symbol string, marginType futures.MarginType, leverage int) bool {
	runtimeTradeConfigCache.RLock()
	entry, ok := runtimeTradeConfigCache.items[symbol]
	runtimeTradeConfigCache.RUnlock()
	return ok && entry.expiresAt.After(time.Now()) && entry.marginType == marginType && entry.leverage == leverage
}

// InvalidateTradeConfig drops the short-lived successful-state hint for symbol.
// Call this whenever another path or User Data WS reports that Binance account
// configuration may have changed.
func InvalidateTradeConfig(symbol string) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" {
		return
	}
	runtimeTradeConfigCache.Lock()
	delete(runtimeTradeConfigCache.items, symbol)
	runtimeTradeConfigCache.Unlock()
}

func isMarginTypeAlreadyConfigured(err error) bool {
	var apiErr *common.APIError
	return errors.As(err, &apiErr) && apiErr != nil && apiErr.Code == -4046
}

func resetRuntimeTradeConfigCacheForTest() {
	runtimeTradeConfigCache.Lock()
	runtimeTradeConfigCache.items = make(map[string]runtimeTradeConfigEntry)
	runtimeTradeConfigCache.Unlock()
}

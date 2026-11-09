package coin

import (
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/models"
	"go_binance_futures/service/binanceapiusage"
	"go_binance_futures/service/tradefills"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"golang.org/x/sync/singleflight"
)

const recentAccountOrdersCacheTTL = 10 * time.Second

type recentAccountOrdersCacheEntry struct {
	symbols   map[string]bool
	expiresAt time.Time
}

var recentAccountOrdersCache = struct {
	sync.RWMutex
	items map[int64]recentAccountOrdersCacheEntry
}{items: make(map[int64]recentAccountOrdersCacheEntry)}

var recentAccountOrdersSF singleflight.Group

// 最近 minute 分钟交易过的币种订单。
// Legacy selector 保留 Binance 账户历史语义，但把 2 秒循环里的重复读取折叠为 10 秒缓存。
func getLimitMinOrder(minute int64) (symbols map[string]bool) {
	if minute <= 0 {
		return map[string]bool{}
	}
	if cached, ok := loadRecentAccountOrdersCache(minute); ok {
		binanceapiusage.RecordOptimization("legacy_coin_selector", "cache_hit", 1)
		binanceapiusage.RecordOptimization("legacy_coin_selector", "prevented_duplicate", 1)
		return cached
	}

	key := strconv.FormatInt(minute, 10)
	value, _, shared := recentAccountOrdersSF.Do(key, func() (interface{}, error) {
		if cached, ok := loadRecentAccountOrdersCache(minute); ok {
			return cached, nil
		}
		nowTime := time.Now().UnixMilli()
		orders, err := binance.GetOrders(binance.ListOrderParams{
			StartTime: nowTime - minute*60*1000,
		})
		if err != nil {
			return map[string]bool{}, err
		}
		result := make(map[string]bool, len(orders))
		for _, order := range orders {
			result[order.Symbol] = true
		}
		storeRecentAccountOrdersCache(minute, result)
		return result, nil
	})
	if shared {
		binanceapiusage.RecordOptimization("legacy_coin_selector", "coalesced", 1)
		binanceapiusage.RecordOptimization("legacy_coin_selector", "prevented_duplicate", 1)
	}
	result, _ := value.(map[string]bool)
	return cloneSymbolSet(result)
}

func loadRecentAccountOrdersCache(minute int64) (map[string]bool, bool) {
	recentAccountOrdersCache.RLock()
	entry, ok := recentAccountOrdersCache.items[minute]
	recentAccountOrdersCache.RUnlock()
	if !ok || !entry.expiresAt.After(time.Now()) {
		return nil, false
	}
	return cloneSymbolSet(entry.symbols), true
}

func storeRecentAccountOrdersCache(minute int64, symbols map[string]bool) {
	recentAccountOrdersCache.Lock()
	recentAccountOrdersCache.items[minute] = recentAccountOrdersCacheEntry{
		symbols: cloneSymbolSet(symbols), expiresAt: time.Now().Add(recentAccountOrdersCacheTTL),
	}
	recentAccountOrdersCache.Unlock()
}

func cloneSymbolSet(input map[string]bool) map[string]bool {
	output := make(map[string]bool, len(input))
	for symbol, enabled := range input {
		output[symbol] = enabled
	}
	return output
}

// getLimitMinManagedClose excludes genuinely recent exchange CLOSE/TP/SL
// fills using immutable trade_time; orders may be updated hours later. It never queries historical `order`, and never treats an order
// acknowledgement as a trade. A canceled partial fill is still a real close.
// On DB errors callers fail closed and do not select any new entry symbols.
func getLimitMinManagedClose(minute int64) (map[string]bool, error) {
	return getLimitMinManagedCloseWithOrm(orm.NewOrm(), minute)
}
func getLimitMinManagedCloseWithOrm(o orm.Ormer, minute int64) (map[string]bool, error) {
	excluded := make(map[string]bool)
	if minute <= 0 {
		return excluded, nil
	}
	now := time.Now().UnixMilli()
	return tradefills.RecentClosedSymbols(o, "main", now-minute*60*1000, now)
}

func GetRandArr(arr []*models.Symbols, num int) (result []*models.Symbols) {
	if num >= len(arr) {
		return arr
	}

	indices := make(map[int]bool, len(arr))
	for len(indices) < num {
		index := rand.Intn(len(arr))
		indices[index] = true
	}

	result = make([]*models.Symbols, num)
	i := 0
	for k := range indices {
		result[i] = arr[k]
		i++
	}

	return result
}

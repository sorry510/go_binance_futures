package coin

import (
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/models"
	"math/rand"
	"sort"
	"time"
)

type TradeCoin1 struct {
}

// 最近5min交易过的币种订单
func getLimitMinOrder(minute int64) (symbols map[string]bool) {
	nowTime := time.Now().Unix() * 1000 // 毫秒时间戳
	orders, _ := binance.GetOrders(binance.ListOrderParams{
		StartTime: nowTime - minute * 60 * 1000,
	})
	symbols = make(map[string]bool)
	for _, order := range orders {
		symbols[order.Symbol] = true
	}
	return symbols
}

// 策略: 从涨幅榜中前6个里面随机选取2个，从跌幅榜中的前6个里面随机选取2个, 最近5min交易过的币不再交易
func (tradeCoin1 TradeCoin1) SelectCoins(allCoins []*models.Symbols) (coins []*models.Symbols) {
	exclude_symbols_map := getLimitMinOrder(5)
	sort.SliceStable(allCoins, func(i, j int) bool {
		return allCoins[i].PercentChange < allCoins[j].PercentChange // 涨幅从小到大排序
	})
	
	filterCoins := []*models.Symbols{}
	for _, coin := range allCoins {
		if _, exist := exclude_symbols_map[coin.Symbol]; exist {
			continue
		} // 最近交易过的排除
		if coin.Enable == 1 { // 非必要，allCoins 已经排除了这些
			filterCoins = append(filterCoins, coin)
		}
	}
	sliceLength := 6
	if len(filterCoins) < sliceLength {
		sliceLength = len(filterCoins)
	}
	res1 := GetRandArr(filterCoins[:sliceLength], 2)
	res2 := GetRandArr(filterCoins[len(filterCoins) - sliceLength:], 2)
	coins = append(coins, res1...)
	coins = append(coins, res2...)
	return coins
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
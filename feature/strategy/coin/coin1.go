package coin

import (
	"go_binance_futures/models"
	"sort"
)

type TradeCoin1 struct {
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
		if coin.Enable == 1 { // 只获取允许交易的币
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

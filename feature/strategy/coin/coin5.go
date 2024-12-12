package coin

import (
	"go_binance_futures/models"
)

type TradeCoin5 struct {
}

// 策略: 最近5min交易过的币不再交易,随机选取5个
func (tradeCoin5 TradeCoin5) SelectCoins(allCoins []*models.Symbols) (coins []*models.Symbols) {
	exclude_symbols_map := getLimitMinLocalOrder(5)
	// sort.SliceStable(allCoins, func(i, j int) bool {
	// 	return allCoins[i].PercentChange < allCoins[j].PercentChange // 涨幅从小到大排序
	// })
	
	filterCoins := []*models.Symbols{}
	for _, coin := range allCoins {
		if _, exist := exclude_symbols_map[coin.Symbol]; exist {
			continue
		} // 最近交易过的排除
		if coin.Enable == 1 { // 只获取允许交易的币
			filterCoins = append(filterCoins, coin)
		}
	}

	coins = GetRandArr(filterCoins, 5)
	return coins
}

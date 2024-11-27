package coin

import (
	"go_binance_futures/models"
	"sort"
)

type TradeCoin6 struct {
}

// 策略: 最近10min交易过的币不再交易,从交易额最高的前150个币中随机选取5个
func (tradeCoin TradeCoin6) SelectCoins(allCoins []*models.Symbols) (coins []*models.Symbols) {
	exclude_symbols_map := getLimitMinOrder(10)
	filterCoins := []*models.Symbols{}
	for _, coin := range allCoins {
		if _, exist := exclude_symbols_map[coin.Symbol]; exist {
			continue
		} // 最近交易过的排除
		if coin.Enable == 1 { // 只获取允许交易的币
			filterCoins = append(filterCoins, coin)
		}
	}
	
	sort.SliceStable(filterCoins, func(i, j int) bool {
		return filterCoins[i].QuoteVolume > filterCoins[j].QuoteVolume // 交易金额从高到低
	})
	
	maxCount := len(filterCoins)
	if maxCount >= 200 {
		maxCount = 200
	}

	coins = GetRandArr(filterCoins[0:maxCount], 5)
	return coins
}

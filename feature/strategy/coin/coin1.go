package coin

import (
	"go_binance_futures/models"
	"sort"
)

type TradeCoin1 struct {
}

func (tradeCoin1 TradeCoin1) SelectCoins(allCoins []*models.Symbols) (coins []*models.Symbols) {
	sort.SliceStable(allCoins, func(i, j int) bool {
		return allCoins[i].PercentChange < allCoins[j].PercentChange // 涨幅从小到大排序
	})
	
	for _, coin := range allCoins {
		if coin.Enable == 1 {
			coins = append(coins, coin)
		}
	}
	return coins[0:2]
}
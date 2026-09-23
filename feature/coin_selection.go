package feature

import (
	"go_binance_futures/feature/strategy/coin"
	"go_binance_futures/models"
	"go_binance_futures/scanner"
)

func selectConfiguredCoins(systemConfig *models.Config, allCoins []*models.Symbols, mode scanner.SmartLocalV2Mode) []*models.Symbols {
	if systemConfig == nil {
		return []*models.Symbols{}
	}
	if systemConfig.FutureStrategyCoin == "smart_local_v2" {
		return coin.SelectSmartLocalV2CoinsForMode(allCoins, mode)
	}
	return GetCoinStrategy(systemConfig.FutureStrategyCoin).SelectCoins(allCoins)
}

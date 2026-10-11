package feature

import (
	"fmt"
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/feature/strategy/coin"
	"go_binance_futures/models"
	"go_binance_futures/scanner"
	"strings"
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

// selectLeadTradeCoins uses a full Lead-specific eligible universe BEFORE the
// Top60 scoring/round-robin step. Filtering Main's selected 5/60 would starve
// Lead of otherwise eligible contracts outside Main's pool.
func selectLeadTradeCoins(systemConfig *models.Config, allCoins []*models.Symbols, cache *binance.LeadSymbolCache) ([]*models.Symbols, error) {
	allowed, err := cache.Snapshot()
	if err != nil {
		// Whitelist controls NEW OPENS only. A temporary refresh failure must
		// never prevent account positions from reaching the exit loop.
		return []*models.Symbols{}, nil
	}
	filtered := leadEligibleUniverse(allCoins, allowed)
	if systemConfig == nil {
		return nil, fmt.Errorf("shared trading config missing")
	}
	if systemConfig.FutureStrategyCoin != "smart_local_v2" {
		// Other existing selectors remain available, but cannot escape the
		// Lead whitelist. The interface still applies the original rules.
		selected := selectConfiguredCoins(systemConfig, filtered, scanner.SmartLocalV2ModeLead)
		return selected, nil
	}
	return coin.SelectSmartLocalV2CoinsForMode(filtered, scanner.SmartLocalV2ModeLead), nil
}

// leadEligibleUniverse filters the FULL local market universe, not the Main
// Top60. No SAPI request occurs here.
func leadEligibleUniverse(all []*models.Symbols, allowed map[string]binance.LeadTradingSymbol) []*models.Symbols {
	filtered := make([]*models.Symbols, 0, len(all))
	for _, sym := range all {
		if sym == nil {
			continue
		}
		if _, ok := allowed[strings.ToUpper(strings.TrimSpace(sym.Symbol))]; ok {
			filtered = append(filtered, sym)
		}
	}
	return filtered
}

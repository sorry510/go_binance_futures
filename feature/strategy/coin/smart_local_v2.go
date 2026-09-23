package coin

import (
	"context"

	"go_binance_futures/models"
	"go_binance_futures/scanner"

	"github.com/beego/beego/v2/core/logs"
)

type SmartLocalV2 struct{}

// SelectCoins implements the legacy CoinStrategy interface and therefore uses
// the real-trade round-robin scope. Test/paper trading must go through
// SelectSmartLocalV2CoinsForMode(..., SmartLocalV2ModeTest), normally via the
// shared feature.selectConfiguredCoins entry point, so it cannot consume the
// live trading rotation.
func (SmartLocalV2) SelectCoins(allCoins []*models.Symbols) []*models.Symbols {
	return SelectSmartLocalV2CoinsForMode(allCoins, scanner.SmartLocalV2ModeTrade)
}

func SelectSmartLocalV2CoinsForMode(allCoins []*models.Symbols, mode scanner.SmartLocalV2Mode) []*models.Symbols {
	result, err := scanner.SmartLocalV2FromSymbolsWithModeCooldown(context.Background(), allCoins, mode, scanner.SmartLocalV2Options{
		Limit: scanner.DefaultSmartLocalV2PoolSize,
	})
	if err != nil {
		logs.Error("smart_local_v2 selector failed:", "mode=", mode, "error=", err)
		return []*models.Symbols{}
	}
	batch, rotation := scanner.NextSmartLocalV2BatchFor(string(mode), result.Candidates, scanner.DefaultSmartLocalV2BatchSize)

	bySymbol := make(map[string]*models.Symbols, len(allCoins))
	for _, item := range allCoins {
		if item != nil {
			bySymbol[item.Symbol] = item
		}
	}
	selected := make([]*models.Symbols, 0, len(batch))
	for _, candidate := range batch {
		if item := bySymbol[candidate.Symbol]; item != nil {
			selected = append(selected, item)
		}
	}

	logs.Debug("smart_local_v2 round-robin batch:",
		"pool_size=", len(result.Candidates),
		"batch_size=", len(selected),
		"sequence=", rotation.Sequence,
		"ranks=", rotation.BatchRanks,
		"symbols=", selectedSymbols(selected),
	)
	return selected
}

func selectedSymbols(items []*models.Symbols) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		if item != nil {
			result = append(result, item.Symbol)
		}
	}
	return result
}

package market

import (
	"strings"
	"testing"

	"go_binance_futures/models"
	markettypes "go_binance_futures/types"
)

func regimeTestSymbols(change float64) []models.Symbols {
	return []models.Symbols{
		{Symbol: "BTCUSDT", PercentChange: change, Open: "100", High: "110", Low: "95", QuoteVolume: 1000},
		{Symbol: "ETHUSDT", PercentChange: change, Open: "100", High: "108", Low: "96", QuoteVolume: 800},
		{Symbol: "SOLUSDT", PercentChange: change, Open: "100", High: "107", Low: "97", QuoteVolume: 600},
		{Symbol: "BNBUSDT", PercentChange: change, Open: "100", High: "106", Low: "98", QuoteVolume: 500},
		{Symbol: "DOGEUSDT", PercentChange: change, Open: "100", High: "105", Low: "99", QuoteVolume: 300},
	}
}

func TestCalculateAlgorithmCondition(t *testing.T) {
	condition, err := CalculateAlgorithmCondition(regimeTestSymbols(10))
	if err != nil {
		t.Fatal(err)
	}
	if condition != markettypes.MarketConditionStrongBull {
		t.Fatalf("condition=%d, want strong bull", condition)
	}
	condition, err = CalculateAlgorithmCondition(regimeTestSymbols(-10))
	if err != nil {
		t.Fatal(err)
	}
	if condition != markettypes.MarketConditionStrongBear {
		t.Fatalf("condition=%d, want strong bear", condition)
	}
}
func TestBuildRegimeSnapshot(t *testing.T) {
	symbols := regimeTestSymbols(2)
	symbols[4].PercentChange = -1
	snapshot := BuildRegimeSnapshot(symbols)
	if snapshot.SymbolCount != 5 || snapshot.AdvancingCount != 4 || snapshot.DecliningCount != 1 {
		t.Fatalf("unexpected breadth: %+v", snapshot)
	}
	if len(snapshot.MajorSymbols) != 4 || len(snapshot.TopGainers) != 5 || len(snapshot.TopLosers) != 5 {
		t.Fatalf("unexpected snapshot lists: %+v", snapshot)
	}
	if snapshot.AverageRange <= 0 || snapshot.QuoteVolumeWeightedChange == 0 {
		t.Fatalf("expected derived metrics: %+v", snapshot)
	}
}

func TestSanitizeReason(t *testing.T) {
	reason := " line1\nline2 " + strings.Repeat("测", 250)
	result := SanitizeReason(reason)
	if strings.Contains(result, "\n") {
		t.Fatalf("reason still contains newline: %q", result)
	}
	if len([]rune(result)) != 200 {
		t.Fatalf("reason rune length=%d, want 200", len([]rune(result)))
	}
}

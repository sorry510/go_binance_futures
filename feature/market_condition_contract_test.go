package feature

import (
	"testing"

	"go_binance_futures/models"
	markettypes "go_binance_futures/types"
)

func TestParseMarketConditionLLMResponseContract(t *testing.T) {
	result, err := parseMarketConditionLLMResponse(`prefix {"market_condition":7,"confidence":0.82,"reason":"市场分化"} suffix`)
	if err != nil {
		t.Fatal(err)
	}
	if result.MarketCondition != 7 || result.Confidence != 0.82 || result.Reason != "市场分化" {
		t.Fatalf("unexpected parsed result: %+v", result)
	}
}

func TestParseMarketConditionLLMResponseRejectsInvalidOutput(t *testing.T) {
	cases := []string{
		`{"market_condition":99,"confidence":0.5,"reason":"bad"}`,
		`{"market_condition":3,"confidence":1.5,"reason":"bad"}`,
		`{"market_condition":3,"confidence":0.5,"reason":""}`,
		`not-json`,
	}
	for _, input := range cases {
		if _, err := parseMarketConditionLLMResponse(input); err == nil {
			t.Fatalf("expected invalid output to fail: %s", input)
		}
	}
}

func TestCalculateLegacyMarketConditionContract(t *testing.T) {
	bullSymbols := []models.Symbols{
		{Symbol: "BTCUSDT", PercentChange: 10},
		{Symbol: "ETHUSDT", PercentChange: 10},
		{Symbol: "SOLUSDT", PercentChange: 10},
		{Symbol: "BNBUSDT", PercentChange: 10},
		{Symbol: "DOGEUSDT", PercentChange: 10},
	}
	condition, err := calculateLegacyMarketCondition(bullSymbols)
	if err != nil {
		t.Fatal(err)
	}
	if condition != markettypes.MarketConditionStrongBull {
		t.Fatalf("condition = %d, want strong bull", condition)
	}

	for index := range bullSymbols {
		bullSymbols[index].PercentChange = -10
	}
	condition, err = calculateLegacyMarketCondition(bullSymbols)
	if err != nil {
		t.Fatal(err)
	}
	if condition != markettypes.MarketConditionStrongBear {
		t.Fatalf("condition = %d, want strong bear", condition)
	}
}

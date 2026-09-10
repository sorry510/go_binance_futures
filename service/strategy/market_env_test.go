package strategy

import (
	"strings"
	"testing"
)

func TestStrategyUsesMarketConditionExactIdentifier(t *testing.T) {
	if !StrategyUsesMarketCondition(`[{"name":"x","code":"MarketCondition == \"2\"","type":"long","enable":true}]`) {
		t.Fatal("MarketCondition reference was not detected")
	}
	if StrategyUsesMarketCondition(`[{"name":"x","code":"OtherMarketCondition == 2","type":"long","enable":true}]`) {
		t.Fatal("partial identifier must not be detected")
	}
}

func TestRemoveDeprecatedMarketEnvKeepsRemainingLogicValid(t *testing.T) {
	raw := `[{"name":"open","type":"long","fullScreen":true,"enable":true,"code":"let ok = (MarketCondition == \"1\" || MarketCondition == \"2\") && BasicTrend >= 0.3 && NowPrice > NowSymbolOpen;\nok"},{"name":"close","type":"close_long","fullScreen":false,"enable":true,"code":"let market_shock = BasicTrend < -4 && ROI < 0;\nmarket_shock || ROI > 10"}]`
	updated, changed, err := RemoveDeprecatedMarketEnvFromStrategyJSON(raw)
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	if strings.Contains(updated, "BasicTrend") || strings.Contains(updated, "BTCUSDT") {
		t.Fatalf("deprecated globals remain: %s", updated)
	}
	if !strings.Contains(updated, "MarketCondition") || !strings.Contains(updated, "NowPrice") || !strings.Contains(updated, "let market_shock = false;") {
		t.Fatalf("remaining strategy logic was not preserved: %s", updated)
	}
	if !strings.Contains(updated, `"fullScreen":true`) {
		t.Fatalf("fullScreen flag was lost during migration: %s", updated)
	}
}

package permission

import (
	"strings"
	"testing"
)

func TestWriteRequiresSkillToolWhitelist(t *testing.T) {
	policy := AllowWritesFor(map[string][]string{
		"strategy_builder": {"save_draft"},
	})
	if err := policy.Allow("strategy_builder", "read_market", RiskRead); err != nil {
		t.Fatalf("read should be allowed: %v", err)
	}
	if err := policy.Allow("strategy_builder", "save_draft", RiskWrite); err != nil {
		t.Fatalf("whitelisted write should be allowed: %v", err)
	}
	if err := policy.Allow("symbol_analysis", "save_draft", RiskWrite); err == nil || !strings.Contains(err.Error(), "not whitelisted") {
		t.Fatalf("write should require matching skill whitelist: %v", err)
	}
}

func TestTradeIsGloballyDisabled(t *testing.T) {
	policy := StaticPolicy{MaxRisk: RiskTrade, TradeEnabled: false}
	err := policy.Allow("any", "place_order", RiskTrade)
	if err == nil || !strings.Contains(err.Error(), "globally disabled") {
		t.Fatalf("trade should be globally disabled: %v", err)
	}
}

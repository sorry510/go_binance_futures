package app

import (
	"testing"

	"go_binance_futures/agent/skill"
	workflowSkills "go_binance_futures/agent/skills/workflows"
)

func TestNativeSkillChatDefaults(t *testing.T) {
	disabled := map[string]bool{
		"alert_analysis": true, "market_regime": true, "strategy_builder": true,
		"alert_triage": true, "strategy_experiment_propose": true, "strategy_experiment_summary": true,
	}
	for _, item := range AvailableSkillImplementations() {
		want := 1
		if disabled[item.Name] {
			want = 0
		}
		if item.ChatDefault != want {
			t.Fatalf("chat default for %s = %d, want %d", item.Name, item.ChatDefault, want)
		}
	}
}

func TestDefaultChatEnabledWorkflowSkillsImplementChatAdapter(t *testing.T) {
	for _, definition := range []*workflowSkills.Definition{
		workflowSkills.MarketScan(), workflowSkills.StrategyReview(), workflowSkills.DailyMarketBrief(),
	} {
		wrapped := newWorkflowChatSkill(definition)
		adapter, ok := any(wrapped).(skill.ChatAdapter)
		if !ok || !adapter.ChatEnabled() {
			t.Fatalf("workflow %s must support ChatAdapter", definition.Name())
		}
	}
}

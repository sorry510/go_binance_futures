package app

import (
	"encoding/json"
	"strings"
	"testing"

	agentruntime "go_binance_futures/agent/runtime"
	"go_binance_futures/agent/skills/symbolanalysis"
	"go_binance_futures/agent/task"
)

func TestChatAssistantTextFormatsSymbolAnalysisAsMarkdown(t *testing.T) {
	condition := 2
	stop := 98.5
	plan := symbolanalysis.TradingPlanV1{
		Version: "trading_plan_v1", Symbol: "BTCUSDT", AsOf: "2026-09-05T03:00:00Z",
		MarketCondition: &condition, Direction: "long", Confidence: 0.78,
		Summary: "短周期结构偏多。", EntryZones: []symbolanalysis.PriceZone{{Low: 100, High: 101}},
		StopLoss: &stop, TakeProfits: []float64{105}, LongTrigger: "回踩企稳",
		Risks: []string{"波动扩大"}, InvalidationConditions: []string{"跌破止损"},
		Evidence: []symbolanalysis.Evidence{{Source: "get_symbol_analysis_context", Finding: "多周期偏多"}}, DataMissing: []string{},
	}
	raw, _ := json.Marshal(plan)
	text := chatAssistantText(&agentruntime.Result{Summary: "summary", Raw: raw}, &task.Task{Skill: symbolanalysis.Name})
	if !strings.Contains(text, "## BTCUSDT 单币分析") || !strings.Contains(text, "### 交易计划") {
		t.Fatalf("unexpected chat markdown: %s", text)
	}
	if strings.Contains(text, `{"version"`) {
		t.Fatalf("chat output must not expose compact JSON: %s", text)
	}
}

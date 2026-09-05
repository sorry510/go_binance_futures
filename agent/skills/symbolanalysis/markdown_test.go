package symbolanalysis

import (
	"strings"
	"testing"
)

func TestFormatMarkdown(t *testing.T) {
	condition := 2
	stop := 98.5
	plan := TradingPlanV1{
		Symbol: "BTCUSDT", AsOf: "2026-09-05T03:00:00Z", MarketCondition: &condition,
		Direction: "long", Confidence: 0.78, Summary: "短周期结构偏多。",
		EntryZones: []PriceZone{{Low: 100, High: 101}}, StopLoss: &stop, TakeProfits: []float64{105, 108},
		LongTrigger: "回踩企稳后放量", Risks: []string{"波动扩大"}, InvalidationConditions: []string{"跌破止损"},
		Evidence: []Evidence{{Source: "get_symbol_analysis_context", Finding: "多周期结构偏多"}}, DataMissing: []string{"depth"},
	}
	text := FormatMarkdown(plan)
	for _, want := range []string{"## BTCUSDT 单币分析", "🟢 做多", "78%", "偏多头（2）", "**入场区 1**：100 - 101", "**止损**：98.5", "**止盈 2**：108", "### 风险与失效条件", "### 证据", "### 数据缺失"} {
		if !strings.Contains(text, want) {
			t.Fatalf("markdown missing %q:\n%s", want, text)
		}
	}
}

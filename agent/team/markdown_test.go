package team

import (
	"strings"
	"testing"
)

func TestFormatMarkdownProducesReadableTeamSummary(t *testing.T) {
	text := FormatMarkdown(ResultV1{
		Version: "symbol_analysis_team_v1", Team: SymbolAnalysisTeam, Symbol: "BTCUSDT", Status: "succeeded",
		Direction: "long", Confidence: 0.76, Summary: "技术与资金流一致偏多。",
		Consensus: []string{"趋势偏多"}, Disagreements: []string{"短线波动较大"},
		Evidence: []Evidence{{Role: "technical_analyst", Source: "get_symbol_analysis_context", Finding: "higher lows"}},
	})
	for _, want := range []string{"## BTCUSDT 多智能体分析", "**方向**：long", "### 综合结论", "### Evidence", "technical_analyst"} {
		if !strings.Contains(text, want) {
			t.Fatalf("markdown missing %q:\n%s", want, text)
		}
	}
}

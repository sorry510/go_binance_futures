package app

import (
	"encoding/json"
	"strings"
	"testing"

	"go_binance_futures/agent/task"
	agentteam "go_binance_futures/agent/team"
)

func TestChatAssistantTextFormatsTeamResultAsMarkdown(t *testing.T) {
	raw, err := json.Marshal(agentteam.ResultV1{
		Version: "symbol_analysis_team_v1", Team: agentteam.SymbolAnalysisTeam, Symbol: "BTCUSDT",
		Status: "succeeded", Direction: "long", Confidence: 0.8, Summary: "team summary",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := chatAssistantText(nil, &task.Task{Skill: agentteam.SymbolAnalysisTeam, Result: raw})
	if !strings.Contains(text, "## BTCUSDT 多智能体分析") || !strings.Contains(text, "team summary") {
		t.Fatalf("unexpected team chat markdown:\n%s", text)
	}
}

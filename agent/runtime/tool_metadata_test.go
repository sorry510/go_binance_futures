package runtime

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"go_binance_futures/agent/permission"
	"go_binance_futures/agent/skill"
	"go_binance_futures/agent/tools"
	"go_binance_futures/llm"
)

func TestRunnerAppliesPerToolTimeout(t *testing.T) {
	client := &fakeLLMClient{items: []fakeLLMItem{
		{response: &llm.Response{Content: `{"action":"tool","tool":"slow","arguments":{}}`}},
		{response: &llm.Response{Content: `{"action":"final","result":{"handled":true}}`}},
	}}
	slow := tools.Func{ToolName: "slow", ToolRisk: permission.RiskRead, ToolMetadata: tools.Metadata{Timeout: 20 * time.Millisecond}, ExecuteFunc: func(ctx context.Context, _ json.RawMessage) (any, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}}
	runner, _ := newTestRunner(t, client, skill.Definition{SkillName: "test", AllowedTools: []string{"slow"}, Rounds: 3}, slow)
	if _, err := runner.Run(context.Background(), Request{Skill: "test", Input: "run"}); err != nil {
		t.Fatal(err)
	}
	second := client.request(1)
	if !strings.Contains(second.Messages[len(second.Messages)-1].Content, "context deadline exceeded") {
		t.Fatalf("timeout was not returned to agent: %+v", second.Messages)
	}
}

func TestRunnerAppliesPerToolResultLimit(t *testing.T) {
	client := &fakeLLMClient{items: []fakeLLMItem{{response: &llm.Response{Content: `{"action":"tool","tool":"large","arguments":{}}`}}}}
	large := tools.Func{ToolName: "large", ToolRisk: permission.RiskRead, ToolMetadata: tools.Metadata{MaxResultBytes: 32}, ExecuteFunc: func(context.Context, json.RawMessage) (any, error) {
		return strings.Repeat("x", 100), nil
	}}
	runner, _ := newTestRunner(t, client, skill.Definition{SkillName: "test", AllowedTools: []string{"large"}, Rounds: 2}, large)
	_, err := runner.Run(context.Background(), Request{Skill: "test", Input: "run"})
	if err == nil || !strings.Contains(err.Error(), "exceeds 32 bytes") {
		t.Fatalf("unexpected error: %v", err)
	}
}

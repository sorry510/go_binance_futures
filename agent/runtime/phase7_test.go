package runtime

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"go_binance_futures/agent/permission"
	"go_binance_futures/agent/skill"
	"go_binance_futures/agent/task"
	"go_binance_futures/agent/tools"
	"go_binance_futures/llm"
)

func TestRunnerEnforcesTokenBudget(t *testing.T) {
	client := &fakeLLMClient{items: []fakeLLMItem{{response: &llm.Response{
		Content: `{"action":"final","result":{"ok":true}}`, Usage: llm.Usage{TotalTokens: 101},
	}}}}
	skills := skill.NewRegistry()
	_ = skills.Register(skill.Definition{SkillName: "budget", Rounds: 2})
	runner, err := NewRunner(Config{
		Client: client, Skills: skills, Tasks: task.NewMemoryStore(), MaxTotalTokens: 100,
		Retry: RetryPolicy{MaxAttempts: 1}, Timeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = runner.Run(context.Background(), Request{Skill: "budget", Input: `{}`})
	if err == nil || !strings.Contains(err.Error(), "total tokens") {
		t.Fatalf("expected token budget error, got %v", err)
	}
}

func TestRunnerEnforcesGlobalToolCallBudget(t *testing.T) {
	client := &fakeLLMClient{items: []fakeLLMItem{
		{response: &llm.Response{Content: `{"action":"tool","tool":"echo","arguments":{}}`}},
		{response: &llm.Response{Content: `{"action":"tool","tool":"echo","arguments":{}}`}},
	}}
	skills := skill.NewRegistry()
	_ = skills.Register(skill.Definition{SkillName: "budget", AllowedTools: []string{"echo"}, Rounds: 3})
	registry := tools.NewRegistry()
	calls := 0
	_ = registry.Register(tools.Func{ToolName: "echo", ToolRisk: permission.RiskRead,
		ExecuteFunc: func(context.Context, json.RawMessage) (any, error) { calls++; return map[string]bool{"ok": true}, nil }})
	runner, err := NewRunner(Config{
		Client: client, Skills: skills, Tools: registry, Tasks: task.NewMemoryStore(), MaxToolCalls: 1,
		Retry: RetryPolicy{MaxAttempts: 1}, Timeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = runner.Run(context.Background(), Request{Skill: "budget", Input: `{}`})
	if err == nil || !strings.Contains(err.Error(), "tool calls") {
		t.Fatalf("expected tool budget error, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("tool executed %d times, want 1", calls)
	}
}

func TestRunnerWritePolicyRequiresSecondWhitelist(t *testing.T) {
	newRunner := func(policy permission.Policy) *DefaultRunner {
		client := &fakeLLMClient{items: []fakeLLMItem{
			{response: &llm.Response{Content: `{"action":"tool","tool":"save_draft","arguments":{}}`}},
			{response: &llm.Response{Content: `{"action":"final","result":{"ok":true}}`}},
		}}
		skills := skill.NewRegistry()
		_ = skills.Register(skill.Definition{SkillName: "writer", AllowedTools: []string{"save_draft"}, Rounds: 3})
		registry := tools.NewRegistry()
		_ = registry.Register(tools.Func{ToolName: "save_draft", ToolRisk: permission.RiskWrite,
			ExecuteFunc: func(context.Context, json.RawMessage) (any, error) { return map[string]bool{"saved": true}, nil }})
		runner, err := NewRunner(Config{Client: client, Skills: skills, Tools: registry, Policy: policy,
			Retry: RetryPolicy{MaxAttempts: 1}, Timeout: time.Second})
		if err != nil {
			t.Fatal(err)
		}
		return runner
	}
	_, err := newRunner(permission.AllowWritesFor(nil)).Run(context.Background(), Request{Skill: "writer", Input: `{}`})
	if err == nil || !strings.Contains(err.Error(), "not whitelisted") {
		t.Fatalf("expected write whitelist rejection, got %v", err)
	}
	if _, err := newRunner(permission.AllowWritesFor(map[string][]string{"writer": {"save_draft"}})).Run(context.Background(), Request{Skill: "writer", Input: `{}`}); err != nil {
		t.Fatalf("whitelisted write should succeed: %v", err)
	}
}

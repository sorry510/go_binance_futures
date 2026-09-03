package runtime

import (
	"context"
	"encoding/json"
	"strings"
	"sync/atomic"
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

func TestRunnerTrimsPerToolResultLimitIntoPartialEnvelope(t *testing.T) {
	client := &fakeLLMClient{items: []fakeLLMItem{
		{response: &llm.Response{Content: `{"action":"tool","tool":"large","arguments":{}}`}},
		{response: &llm.Response{Content: `{"action":"final","result":{"handled":true}}`}},
	}}
	large := tools.Func{ToolName: "large", ToolRisk: permission.RiskRead, ToolMetadata: tools.Metadata{MaxResultBytes: 32}, ExecuteFunc: func(context.Context, json.RawMessage) (any, error) {
		return strings.Repeat("x", 100), nil
	}}
	runner, _ := newTestRunner(t, client, skill.Definition{SkillName: "test", AllowedTools: []string{"large"}, Rounds: 2}, large)
	if _, err := runner.Run(context.Background(), Request{Skill: "test", Input: "run"}); err != nil {
		t.Fatal(err)
	}
	second := client.request(1)
	message := second.Messages[len(second.Messages)-1].Content
	if !strings.Contains(message, `"partial":true`) || !strings.Contains(message, "result trimmed") {
		t.Fatalf("partial envelope missing: %s", message)
	}
}

func TestRunnerToolRuntimeValidatesSchemaBeforeNativeExecution(t *testing.T) {
	client := &fakeLLMClient{items: []fakeLLMItem{
		{response: &llm.Response{Content: `{"action":"tool","tool":"schema","arguments":{"symbol":7}}`}},
		{response: &llm.Response{Content: `{"action":"final","result":{"handled":true}}`}},
	}}
	var calls int
	schemaTool := tools.Func{ToolName: "schema", ToolRisk: permission.RiskRead, ToolMetadata: tools.Metadata{
		InputSchema: json.RawMessage(`{"type":"object","required":["symbol"],"additionalProperties":false,"properties":{"symbol":{"type":"string"}}}`),
	}, ExecuteFunc: func(context.Context, json.RawMessage) (any, error) {
		calls++
		return map[string]bool{"ok": true}, nil
	}}
	runner, _ := newTestRunner(t, client, skill.Definition{SkillName: "test", AllowedTools: []string{"schema"}, Rounds: 2}, schemaTool)
	if _, err := runner.Run(context.Background(), Request{Skill: "test", Input: "run"}); err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatalf("native tool executed despite invalid schema: %d", calls)
	}
	second := client.request(1)
	if !strings.Contains(second.Messages[len(second.Messages)-1].Content, `"error_type":"invalid_input"`) {
		t.Fatalf("invalid_input envelope missing: %+v", second.Messages)
	}
}

func TestRunnerToolRuntimePersistsCacheTrace(t *testing.T) {
	client := &fakeLLMClient{items: []fakeLLMItem{
		{response: &llm.Response{Content: `{"action":"tool","tool":"cached","arguments":{"symbol":"BTCUSDT"}}`}},
		{response: &llm.Response{Content: `{"action":"tool","tool":"cached","arguments":{"symbol":"BTCUSDT"}}`}},
		{response: &llm.Response{Content: `{"action":"final","result":{"handled":true}}`}},
	}}
	var calls int
	cached := tools.Func{ToolName: "cached", ToolRisk: permission.RiskRead, ToolMetadata: tools.Metadata{Idempotent: true, CacheTTL: time.Minute}, ExecuteFunc: func(context.Context, json.RawMessage) (any, error) {
		calls++
		return map[string]any{"symbol": "BTCUSDT", "as_of": time.Now().UTC().Format(time.RFC3339)}, nil
	}}
	runner, store := newTestRunner(t, client, skill.Definition{SkillName: "test", AllowedTools: []string{"cached"}, Rounds: 3}, cached)
	result, err := runner.Run(context.Background(), Request{Skill: "test", Input: "run"})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("cached native tool calls = %d, want 1", calls)
	}
	stored, err := store.Get(context.Background(), result.TaskID)
	if err != nil {
		t.Fatal(err)
	}
	var steps []ExecutionStep
	if err := json.Unmarshal(stored.Steps, &steps); err != nil {
		t.Fatal(err)
	}
	var toolTraces int
	var cacheHit bool
	for _, step := range steps {
		if step.ToolTrace == nil {
			continue
		}
		toolTraces++
		if step.ToolTrace.CacheHit {
			cacheHit = true
		}
		if step.ToolTrace.CallBudget <= 0 || step.ToolTrace.ArgumentsHash == "" {
			t.Fatalf("tool budget/arguments trace missing: %+v", step.ToolTrace)
		}
	}
	if toolTraces != 2 || !cacheHit {
		t.Fatalf("unexpected tool traces count=%d cacheHit=%v steps=%+v", toolTraces, cacheHit, steps)
	}
}

func TestRunnerExecutesSafeParallelToolDecision(t *testing.T) {
	client := &fakeLLMClient{items: []fakeLLMItem{
		{response: &llm.Response{Content: `{"action":"parallel_tools","tools":[{"tool":"left","arguments":{}},{"tool":"right","arguments":{}}]}`}},
		{response: &llm.Response{Content: `{"action":"final","result":{"handled":true}}`}},
	}}
	var active atomic.Int32
	var maxActive atomic.Int32
	makeTool := func(name string) tools.Tool {
		return tools.Func{ToolName: name, ToolRisk: permission.RiskRead, ToolMetadata: tools.Metadata{Idempotent: true}, ExecuteFunc: func(context.Context, json.RawMessage) (any, error) {
			now := active.Add(1)
			for {
				old := maxActive.Load()
				if now <= old || maxActive.CompareAndSwap(old, now) {
					break
				}
			}
			time.Sleep(30 * time.Millisecond)
			active.Add(-1)
			return map[string]string{"tool": name}, nil
		}}
	}
	runner, store := newTestRunner(t, client, skill.Definition{SkillName: "test", AllowedTools: []string{"left", "right"}, Rounds: 2}, makeTool("left"), makeTool("right"))
	result, err := runner.Run(context.Background(), Request{Skill: "test", Input: "run"})
	if err != nil {
		t.Fatal(err)
	}
	if maxActive.Load() < 2 {
		t.Fatalf("parallel tools did not overlap: max=%d", maxActive.Load())
	}
	stored, err := store.Get(context.Background(), result.TaskID)
	if err != nil {
		t.Fatal(err)
	}
	var steps []ExecutionStep
	if err := json.Unmarshal(stored.Steps, &steps); err != nil {
		t.Fatal(err)
	}
	parallelSteps, tracedTools := 0, 0
	for _, step := range steps {
		if step.Type == StepParallelTools {
			parallelSteps++
		}
		if step.ToolTrace != nil {
			tracedTools++
		}
	}
	if parallelSteps != 1 || tracedTools != 2 {
		t.Fatalf("parallel trace missing: parent=%d tools=%d steps=%+v", parallelSteps, tracedTools, steps)
	}
	second := client.request(1)
	toolResults := 0
	for _, message := range second.Messages {
		if strings.HasPrefix(message.Content, "TOOL_RESULT\n") {
			toolResults++
		}
	}
	if toolResults != 2 {
		t.Fatalf("parallel results missing from LLM context: %d", toolResults)
	}
}

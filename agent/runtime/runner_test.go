package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go_binance_futures/agent/permission"
	"go_binance_futures/agent/skill"
	"go_binance_futures/agent/task"
	"go_binance_futures/agent/tools"
	"go_binance_futures/agent/validator"
	"go_binance_futures/llm"
)

type fakeLLMItem struct {
	response *llm.Response
	err      error
}

type fakeLLMClient struct {
	mu       sync.Mutex
	items    []fakeLLMItem
	requests []llm.Request
}

func (client *fakeLLMClient) Provider() llm.Provider { return llm.ProviderOpenAICompatible }

func (client *fakeLLMClient) Generate(_ context.Context, request llm.Request) (*llm.Response, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	client.requests = append(client.requests, request)
	index := len(client.requests) - 1
	if index >= len(client.items) {
		return nil, fmt.Errorf("unexpected LLM call %d", index+1)
	}
	return client.items[index].response, client.items[index].err
}

func (client *fakeLLMClient) request(index int) llm.Request {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.requests[index]
}

func newTestRunner(t *testing.T, client llm.Client, definition skill.Skill, registeredTools ...tools.Tool) (*DefaultRunner, *task.MemoryStore) {
	t.Helper()
	skills := skill.NewRegistry()
	if err := skills.Register(definition); err != nil {
		t.Fatal(err)
	}
	toolRegistry := tools.NewRegistry()
	for _, tool := range registeredTools {
		if err := toolRegistry.Register(tool); err != nil {
			t.Fatal(err)
		}
	}
	store := task.NewMemoryStore()
	policy := permission.AllowReadOnly()
	runner, err := NewRunner(Config{
		Client: client, Skills: skills, Tools: toolRegistry, Tasks: store, Policy: policy,
		Retry: RetryPolicy{MaxAttempts: 1}, Timeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	return runner, store
}

func TestRunnerFinalCompletesTask(t *testing.T) {
	client := &fakeLLMClient{items: []fakeLLMItem{{response: &llm.Response{
		Model: "fake", Content: `{"action":"final","summary":"done","result":{"ok":true}}`,
	}}}}
	definition := skill.Definition{SkillName: "test", Prompt: "test", Rounds: 2}
	runner, store := newTestRunner(t, client, definition)

	result, err := runner.Run(context.Background(), Request{Skill: "test", Input: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Summary != "done" || !strings.Contains(string(result.Raw), `"ok":true`) {
		t.Fatalf("unexpected result: %+v", result)
	}
	stored, err := store.Get(context.Background(), result.TaskID)
	if err != nil || stored.Status != task.StatusSucceeded || stored.Progress != 100 {
		t.Fatalf("unexpected stored task: %+v err=%v", stored, err)
	}
}

func TestRunnerToolThenFinal(t *testing.T) {
	client := &fakeLLMClient{items: []fakeLLMItem{
		{response: &llm.Response{Content: `{"action":"tool","tool":"echo","arguments":{"value":7}}`}},
		{response: &llm.Response{Content: `{"action":"final","result":{"ok":true}}`}},
	}}
	var calls atomic.Int32
	echoTool := tools.Func{
		ToolName: "echo", ToolRisk: permission.RiskRead,
		ExecuteFunc: func(_ context.Context, args json.RawMessage) (any, error) {
			calls.Add(1)
			var input map[string]int
			if err := json.Unmarshal(args, &input); err != nil {
				return nil, err
			}
			return input, nil
		},
	}
	definition := skill.Definition{SkillName: "test", AllowedTools: []string{"echo"}, Rounds: 3}
	runner, _ := newTestRunner(t, client, definition, echoTool)

	if _, err := runner.Run(context.Background(), Request{Skill: "test", Input: "hello"}); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("tool calls = %d, want 1", calls.Load())
	}
	second := client.request(1)
	if len(second.Messages) == 0 || !strings.Contains(second.Messages[len(second.Messages)-1].Content, "TOOL_RESULT") {
		t.Fatalf("second request missing tool result: %+v", second.Messages)
	}
}

func TestRunnerRejectsUnavailableOrUnauthorizedTools(t *testing.T) {
	cases := []struct {
		name       string
		allowed    []string
		registered []tools.Tool
		want       string
	}{
		{name: "not allowed", allowed: nil, registered: []tools.Tool{tools.Func{ToolName: "echo", ToolRisk: permission.RiskRead, ExecuteFunc: func(context.Context, json.RawMessage) (any, error) { return nil, nil }}}, want: "does not allow"},
		{name: "not registered", allowed: []string{"echo"}, want: "not registered"},
		{name: "risk denied", allowed: []string{"trade"}, registered: []tools.Tool{tools.Func{ToolName: "trade", ToolRisk: permission.RiskTrade, ExecuteFunc: func(context.Context, json.RawMessage) (any, error) { return nil, nil }}}, want: "exceeds allowed risk"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			client := &fakeLLMClient{items: []fakeLLMItem{{response: &llm.Response{Content: `{"action":"tool","tool":"` + map[bool]string{true: "trade", false: "echo"}[testCase.name == "risk denied"] + `","arguments":{}}`}}}}
			definition := skill.Definition{SkillName: "test", AllowedTools: testCase.allowed, Rounds: 2}
			runner, _ := newTestRunner(t, client, definition, testCase.registered...)
			_, err := runner.Run(context.Background(), Request{Skill: "test", Input: "hello"})
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("error = %v, want containing %q", err, testCase.want)
			}
		})
	}
}

func TestRunnerReturnsToolErrorToAgent(t *testing.T) {
	client := &fakeLLMClient{items: []fakeLLMItem{
		{response: &llm.Response{Content: `{"action":"tool","tool":"strict","arguments":{"bad":1}}`}},
		{response: &llm.Response{Content: `{"action":"final","result":{"repaired":true}}`}},
	}}
	strictTool := tools.Func{
		ToolName: "strict", ToolRisk: permission.RiskRead,
		ExecuteFunc: func(_ context.Context, _ json.RawMessage) (any, error) {
			return nil, fmt.Errorf("invalid tool arguments")
		},
	}
	definition := skill.Definition{SkillName: "test", AllowedTools: []string{"strict"}, Rounds: 3}
	runner, _ := newTestRunner(t, client, definition, strictTool)

	if _, err := runner.Run(context.Background(), Request{Skill: "test", Input: "hello"}); err != nil {
		t.Fatal(err)
	}
	second := client.request(1)
	last := second.Messages[len(second.Messages)-1].Content
	if !strings.Contains(last, `"ok":false`) || !strings.Contains(last, "invalid tool arguments") {
		t.Fatalf("tool error was not returned to agent: %s", last)
	}
}

func TestRunnerRepairsInvalidFinal(t *testing.T) {
	client := &fakeLLMClient{items: []fakeLLMItem{
		{response: &llm.Response{Content: `{"action":"final","result":{"version":1}}`}},
		{response: &llm.Response{Content: `{"action":"final","result":{"version":2}}`}},
	}}
	definition := skill.Definition{
		SkillName: "test", Rounds: 3,
		FinalValidator: validator.Func(func(_ context.Context, raw json.RawMessage) (any, error) {
			var payload struct {
				Version int `json:"version"`
			}
			if err := json.Unmarshal(raw, &payload); err != nil {
				return nil, err
			}
			if payload.Version != 2 {
				return nil, fmt.Errorf("version must be 2")
			}
			return payload, nil
		}),
	}
	runner, _ := newTestRunner(t, client, definition)
	result, err := runner.Run(context.Background(), Request{Skill: "test", Input: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(result.Raw), `"version":2`) {
		t.Fatalf("unexpected final result: %s", result.Raw)
	}
	second := client.request(1)
	if !strings.Contains(second.Messages[len(second.Messages)-1].Content, "AGENT_FEEDBACK") {
		t.Fatalf("validation feedback missing: %+v", second.Messages)
	}
}

func TestRunnerStopsAtMaxRounds(t *testing.T) {
	client := &fakeLLMClient{items: []fakeLLMItem{
		{response: &llm.Response{Content: `{"action":"final","result":{"ok":false}}`}},
		{response: &llm.Response{Content: `{"action":"final","result":{"ok":false}}`}},
	}}
	definition := skill.Definition{
		SkillName: "test", Rounds: 2,
		FinalValidator: validator.Func(func(_ context.Context, _ json.RawMessage) (any, error) {
			return nil, fmt.Errorf("always invalid")
		}),
	}
	runner, _ := newTestRunner(t, client, definition)
	_, err := runner.Run(context.Background(), Request{Skill: "test", Input: "hello"})
	if err == nil || !strings.Contains(err.Error(), "maximum 2 rounds") {
		t.Fatalf("unexpected error: %v", err)
	}
}

type blockingClient struct {
	started chan struct{}
}

func (client *blockingClient) Provider() llm.Provider { return llm.ProviderOpenAICompatible }
func (client *blockingClient) Generate(ctx context.Context, _ llm.Request) (*llm.Response, error) {
	select {
	case client.started <- struct{}{}:
	default:
	}
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestRunnerCancellationMarksTaskCancelled(t *testing.T) {
	client := &blockingClient{started: make(chan struct{}, 1)}
	skills := skill.NewRegistry()
	if err := skills.Register(skill.Definition{SkillName: "test", Rounds: 2}); err != nil {
		t.Fatal(err)
	}
	store := task.NewMemoryStore()
	events := make(chan task.Event, 8)
	policy := permission.AllowReadOnly()
	runner, err := NewRunner(Config{
		Client: client, Skills: skills, Tasks: store, Policy: policy,
		EventHook: func(event task.Event) { events <- event }, Timeout: time.Second,
		Retry: RetryPolicy{MaxAttempts: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	resultC := make(chan error, 1)
	go func() {
		_, runErr := runner.Run(ctx, Request{Skill: "test", Input: "hello"})
		resultC <- runErr
	}()
	<-client.started
	cancel()
	if runErr := <-resultC; runErr == nil || !strings.Contains(runErr.Error(), "context canceled") {
		t.Fatalf("unexpected cancellation error: %v", runErr)
	}

	var taskID string
	for len(events) > 0 {
		event := <-events
		if event.TaskID != "" {
			taskID = event.TaskID
		}
	}
	stored, err := store.Get(context.Background(), taskID)
	if err != nil || stored.Status != task.StatusCancelled {
		t.Fatalf("unexpected cancelled task: %+v err=%v", stored, err)
	}
}

type echoClient struct{}

func (*echoClient) Provider() llm.Provider { return llm.ProviderOpenAICompatible }
func (*echoClient) Generate(_ context.Context, request llm.Request) (*llm.Response, error) {
	input := ""
	if len(request.Messages) > 0 {
		input = request.Messages[0].Content
	}
	payload, _ := json.Marshal(map[string]any{
		"action": "final", "result": map[string]string{"input": input},
	})
	return &llm.Response{Content: string(payload)}, nil
}

func TestRunnerConcurrentTasksAreIsolated(t *testing.T) {
	definition := skill.Definition{SkillName: "test", Rounds: 2}
	runner, _ := newTestRunner(t, &echoClient{}, definition)
	type outcome struct {
		input  string
		result *Result
		err    error
	}
	const count = 20
	outcomes := make(chan outcome, count)
	var wait sync.WaitGroup
	for index := 0; index < count; index++ {
		input := fmt.Sprintf("input-%d", index)
		wait.Add(1)
		go func() {
			defer wait.Done()
			result, err := runner.Run(context.Background(), Request{Skill: "test", Input: input})
			outcomes <- outcome{input: input, result: result, err: err}
		}()
	}
	wait.Wait()
	close(outcomes)
	seenIDs := make(map[string]bool, count)
	for item := range outcomes {
		if item.err != nil {
			t.Fatal(item.err)
		}
		if item.result == nil || seenIDs[item.result.TaskID] {
			t.Fatalf("duplicate or empty task id: %+v", item.result)
		}
		seenIDs[item.result.TaskID] = true
		var payload struct {
			Input string `json:"input"`
		}
		if err := json.Unmarshal(item.result.Raw, &payload); err != nil {
			t.Fatal(err)
		}
		if payload.Input != item.input {
			t.Fatalf("task context leaked: got %q want %q", payload.Input, item.input)
		}
	}
	if len(seenIDs) != count {
		t.Fatalf("task count = %d, want %d", len(seenIDs), count)
	}
}

func TestRunnerRetriesRecoverableLLMError(t *testing.T) {
	client := &fakeLLMClient{items: []fakeLLMItem{
		{err: &llm.HTTPError{StatusCode: 429, Body: "rate limited"}},
		{response: &llm.Response{Content: `{"action":"final","result":{"ok":true}}`}},
	}}
	skills := skill.NewRegistry()
	if err := skills.Register(skill.Definition{SkillName: "test", Rounds: 2}); err != nil {
		t.Fatal(err)
	}
	policy := permission.AllowReadOnly()
	runner, err := NewRunner(Config{
		Client: client, Skills: skills, Policy: policy,
		Retry: RetryPolicy{MaxAttempts: 2}, Timeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(context.Background(), Request{Skill: "test", Input: "hello"}); err != nil {
		t.Fatal(err)
	}
	client.mu.Lock()
	calls := len(client.requests)
	client.mu.Unlock()
	if calls != 2 {
		t.Fatalf("LLM calls = %d, want 2", calls)
	}
}

func TestRunnerRetriesEOFTransportError(t *testing.T) {
	client := &fakeLLMClient{items: []fakeLLMItem{
		{err: fmt.Errorf("send llm request: %w", io.EOF)},
		{response: &llm.Response{Content: `{"action":"final","result":{"ok":true}}`}},
	}}
	skills := skill.NewRegistry()
	if err := skills.Register(skill.Definition{SkillName: "test", Rounds: 2}); err != nil {
		t.Fatal(err)
	}
	runner, err := NewRunner(Config{
		Client: client, Skills: skills, Retry: RetryPolicy{MaxAttempts: 2}, Timeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(context.Background(), Request{Skill: "test", Input: "hello"}); err != nil {
		t.Fatal(err)
	}
	client.mu.Lock()
	calls := len(client.requests)
	client.mu.Unlock()
	if calls != 2 {
		t.Fatalf("LLM calls = %d, want 2", calls)
	}
}

func TestRunnerEnforcesRequiredToolsBeforeFinal(t *testing.T) {
	client := &fakeLLMClient{items: []fakeLLMItem{
		{response: &llm.Response{Content: `{"action":"final","result":{"ok":true}}`}},
		{response: &llm.Response{Content: `{"action":"tool","tool":"echo","arguments":{}}`}},
		{response: &llm.Response{Content: `{"action":"final","result":{"ok":true}}`}},
	}}
	echo := tools.Func{ToolName: "echo", ToolRisk: permission.RiskRead, ExecuteFunc: func(context.Context, json.RawMessage) (any, error) {
		return map[string]bool{"ok": true}, nil
	}}
	definition := skill.Definition{
		SkillName: "required", AllowedTools: []string{"echo"}, Rounds: 4,
		RequiredToolsFunc: func(skill.Request) []string { return []string{"echo"} },
	}
	runner, _ := newTestRunner(t, client, definition, echo)
	if _, err := runner.Run(context.Background(), Request{Skill: "required", Input: "test"}); err != nil {
		t.Fatal(err)
	}
	if len(client.requests) != 3 {
		t.Fatalf("LLM calls = %d, want 3", len(client.requests))
	}
	feedback := client.request(1).Messages
	if len(feedback) == 0 || !strings.Contains(feedback[len(feedback)-1].Content, "required_tools") {
		t.Fatalf("required tool feedback missing: %+v", feedback)
	}
}

func TestRunnerHooksReceiveMessagesAndValidationCandidates(t *testing.T) {
	client := &fakeLLMClient{items: []fakeLLMItem{
		{response: &llm.Response{Content: `{"action":"final","result":{"version":1}}`}},
		{response: &llm.Response{Content: `{"action":"final","result":{"version":2}}`}},
	}}
	definition := skill.Definition{SkillName: "hooks", Rounds: 3, FinalValidator: validator.Func(func(_ context.Context, raw json.RawMessage) (any, error) {
		var value struct {
			Version int `json:"version"`
		}
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, err
		}
		if value.Version != 2 {
			return nil, fmt.Errorf("version must be 2")
		}
		return value, nil
	})}
	skills := skill.NewRegistry()
	if err := skills.Register(definition); err != nil {
		t.Fatal(err)
	}
	var messages []llm.Message
	var validations []error
	runner, err := NewRunner(Config{
		Client: client, Skills: skills, Retry: RetryPolicy{MaxAttempts: 1}, Timeout: time.Second,
		MessageHook:    func(_ string, message llm.Message) { messages = append(messages, message) },
		ValidationHook: func(_ string, _ json.RawMessage, err error) { validations = append(validations, err) },
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(context.Background(), Request{Skill: "hooks", Input: "test"}); err != nil {
		t.Fatal(err)
	}
	if len(validations) != 2 || validations[0] == nil || validations[1] != nil {
		t.Fatalf("unexpected validation hooks: %#v", validations)
	}
	if len(messages) < 3 || !strings.Contains(messages[1].Content, "AGENT_FEEDBACK") {
		t.Fatalf("unexpected message hooks: %+v", messages)
	}
}

func TestRunnerRepairsTruncatedResponse(t *testing.T) {
	client := &fakeLLMClient{items: []fakeLLMItem{
		{response: &llm.Response{Content: `{"action":"final"`, FinishReason: "length"}},
		{response: &llm.Response{Content: `{"action":"final","result":{"ok":true}}`}},
	}}
	definition := skill.Definition{SkillName: "truncate", Rounds: 3}
	runner, _ := newTestRunner(t, client, definition)
	if _, err := runner.Run(context.Background(), Request{Skill: "truncate", Input: "test"}); err != nil {
		t.Fatal(err)
	}
	request := client.request(1)
	if !strings.Contains(request.Messages[len(request.Messages)-1].Content, "truncated_response") {
		t.Fatalf("truncation feedback missing: %+v", request.Messages)
	}
}

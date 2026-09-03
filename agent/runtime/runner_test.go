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

	"go_binance_futures/agent/contextengine"
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
		{name: "risk denied", allowed: []string{"trade"}, registered: []tools.Tool{tools.Func{ToolName: "trade", ToolRisk: permission.RiskTrade, ExecuteFunc: func(context.Context, json.RawMessage) (any, error) { return nil, nil }}}, want: "globally disabled"},
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

type checkpointClient struct {
	mu       sync.Mutex
	calls    int
	blocking chan struct{}
}

func (client *checkpointClient) Provider() llm.Provider { return llm.ProviderOpenAICompatible }
func (client *checkpointClient) Generate(ctx context.Context, _ llm.Request) (*llm.Response, error) {
	client.mu.Lock()
	client.calls++
	call := client.calls
	client.mu.Unlock()
	if call == 1 {
		return &llm.Response{Content: `{"action":"tool","tool":"echo","arguments":{"value":7}}`}, nil
	}
	select {
	case client.blocking <- struct{}{}:
	default:
	}
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestRunnerResumeFromSafeToolCheckpointDoesNotRepeatTool(t *testing.T) {
	client := &checkpointClient{blocking: make(chan struct{}, 1)}
	var toolCalls atomic.Int32
	echoTool := tools.Func{
		ToolName: "echo", ToolRisk: permission.RiskRead,
		ToolMetadata: tools.Metadata{Idempotent: true},
		ExecuteFunc: func(_ context.Context, args json.RawMessage) (any, error) {
			toolCalls.Add(1)
			var value map[string]int
			if err := json.Unmarshal(args, &value); err != nil {
				return nil, err
			}
			return value, nil
		},
	}
	definition := skill.Definition{SkillName: "resume", AllowedTools: []string{"echo"}, Rounds: 4}
	runner, store := newTestRunner(t, client, definition, echoTool)
	ctx, cancel := context.WithCancel(context.Background())
	resultC := make(chan error, 1)
	go func() {
		_, err := runner.Run(ctx, Request{TaskID: "resume-task", Skill: "resume", Input: "test", Metadata: map[string]any{"scheduler_job": "market_regime", "ignored_complex": []string{"not", "persisted"}}})
		resultC <- err
	}()
	<-client.blocking
	cancel()
	if err := <-resultC; err == nil || !strings.Contains(err.Error(), "context canceled") {
		t.Fatalf("unexpected cancellation: %v", err)
	}
	cancelled, err := store.Get(context.Background(), "resume-task")
	if err != nil {
		t.Fatal(err)
	}
	if cancelled.Status != task.StatusCancelled || cancelled.CheckpointJSON == "" {
		t.Fatalf("expected resumable cancelled task: %+v", cancelled)
	}
	if toolCalls.Load() != 1 {
		t.Fatalf("tool calls before resume = %d, want 1", toolCalls.Load())
	}

	resumeClient := &fakeLLMClient{items: []fakeLLMItem{{response: &llm.Response{Content: `{"action":"final","result":{"ok":true}}`}}}}
	resumeRunner, err := NewRunner(Config{
		Client: resumeClient, Skills: runner.cfg.Skills, Tools: runner.cfg.Tools, Tasks: store,
		Policy: permission.AllowReadOnly(), Retry: RetryPolicy{MaxAttempts: 1}, Timeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	resumeRequest, err := resumeRunner.ResumeRequest(context.Background(), "resume-task")
	if err != nil {
		t.Fatal(err)
	}
	if resumeRequest.Metadata["scheduler_job"] != "market_regime" {
		t.Fatalf("runtime-owned resume metadata was not preserved: %+v", resumeRequest.Metadata)
	}
	if _, exists := resumeRequest.Metadata["ignored_complex"]; exists {
		t.Fatalf("arbitrary skill metadata leaked into checkpoint: %+v", resumeRequest.Metadata)
	}
	result, err := resumeRunner.Resume(context.Background(), "resume-task")
	if err != nil {
		t.Fatal(err)
	}
	if result == nil || toolCalls.Load() != 1 {
		t.Fatalf("resume repeated safe tool: result=%+v calls=%d", result, toolCalls.Load())
	}
	stored, err := store.Get(context.Background(), "resume-task")
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != task.StatusSucceeded || stored.ResumeCount != 1 || stored.ExecutionMode != string(ExecutionModeReact) {
		t.Fatalf("unexpected resumed task: %+v", stored)
	}
	var steps []ExecutionStep
	if err := json.Unmarshal(stored.Steps, &steps); err != nil || len(steps) < 4 {
		t.Fatalf("execution steps not persisted: %s err=%v", stored.Steps, err)
	}
	foundCheckpoint := false
	for _, event := range stored.Events {
		if event.StepID != "" && event.Checkpoint {
			foundCheckpoint = true
			break
		}
	}
	if !foundCheckpoint {
		t.Fatalf("checkpoint step event missing: %+v", stored.Events)
	}
}

func TestRunnerUnsafeToolClearsRecoveryCheckpointBeforeExecution(t *testing.T) {
	client := &fakeLLMClient{items: []fakeLLMItem{{response: &llm.Response{Content: `{"action":"tool","tool":"write","arguments":{}}`}}}}
	writeTool := tools.Func{
		ToolName: "write", ToolRisk: permission.RiskWrite,
		ToolMetadata: tools.Metadata{Idempotent: true},
		ExecuteFunc:  func(context.Context, json.RawMessage) (any, error) { return map[string]bool{"ok": true}, nil },
	}
	skills := skill.NewRegistry()
	if err := skills.Register(skill.Definition{SkillName: "unsafe", AllowedTools: []string{"write"}, Rounds: 1}); err != nil {
		t.Fatal(err)
	}
	store := task.NewMemoryStore()
	runner, err := NewRunner(Config{
		Client: client, Skills: skills, Tools: func() *tools.Registry {
			registry := tools.NewRegistry()
			_ = registry.Register(writeTool)
			return registry
		}(),
		Tasks: store, Policy: permission.AllowWritesFor(map[string][]string{"unsafe": {"write"}}),
		Retry: RetryPolicy{MaxAttempts: 1}, Timeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = runner.Run(context.Background(), Request{TaskID: "unsafe-task", Skill: "unsafe", Input: "test"})
	stored, err := store.Get(context.Background(), "unsafe-task")
	if err != nil {
		t.Fatal(err)
	}
	if stored.CheckpointJSON != "" {
		t.Fatalf("unsafe tool left resumable checkpoint: %s", stored.CheckpointJSON)
	}
}

type fakePlanner struct {
	plan Plan
	err  error
}

func (planner fakePlanner) Plan(context.Context, PlanRequest) (Plan, error) {
	return planner.plan, planner.err
}

func TestPlanExecuteUsesNormalToolPermissionBoundary(t *testing.T) {
	skills := skill.NewRegistry()
	if err := skills.Register(skill.Definition{SkillName: "planned", Mode: string(ExecutionModePlanExecute), AllowedTools: []string{"write"}, Rounds: 2}); err != nil {
		t.Fatal(err)
	}
	registry := tools.NewRegistry()
	if err := registry.Register(tools.Func{
		ToolName: "write", ToolRisk: permission.RiskWrite, ToolMetadata: tools.Metadata{Idempotent: true},
		ExecuteFunc: func(context.Context, json.RawMessage) (any, error) { return map[string]bool{"ok": true}, nil },
	}); err != nil {
		t.Fatal(err)
	}
	runner, err := NewRunner(Config{
		Client: &fakeLLMClient{}, Skills: skills, Tools: registry, Tasks: task.NewMemoryStore(),
		Policy: permission.AllowReadOnly(), Planner: fakePlanner{plan: Plan{Steps: []PlannedStep{{StepID: "write-1", Type: StepTool, Tool: "write"}}}},
		Retry: RetryPolicy{MaxAttempts: 1}, Timeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = runner.Run(context.Background(), Request{TaskID: "plan-denied", Skill: "planned", Input: "test"})
	if err == nil || !strings.Contains(err.Error(), "exceeds allowed risk") {
		t.Fatalf("planner bypassed tool permission: %v", err)
	}
}

func TestPlanExecuteRunsPlannedReadToolThenReactFinal(t *testing.T) {
	skills := skill.NewRegistry()
	if err := skills.Register(skill.Definition{SkillName: "planned", Mode: string(ExecutionModePlanExecute), AllowedTools: []string{"echo"}, Rounds: 2}); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	registry := tools.NewRegistry()
	if err := registry.Register(tools.Func{
		ToolName: "echo", ToolRisk: permission.RiskRead, ToolMetadata: tools.Metadata{Idempotent: true},
		ExecuteFunc: func(context.Context, json.RawMessage) (any, error) {
			calls.Add(1)
			return map[string]bool{"ok": true}, nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	store := task.NewMemoryStore()
	client := &fakeLLMClient{items: []fakeLLMItem{{response: &llm.Response{Content: `{"action":"final","result":{"ok":true}}`}}}}
	runner, err := NewRunner(Config{
		Client: client, Skills: skills, Tools: registry, Tasks: store, Policy: permission.AllowReadOnly(),
		Planner: fakePlanner{plan: Plan{Summary: "prefetch", Steps: []PlannedStep{{StepID: "read-1", Type: StepTool, Tool: "echo", Arguments: json.RawMessage(`{}`)}}}},
		Retry:   RetryPolicy{MaxAttempts: 1}, Timeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := runner.Run(context.Background(), Request{TaskID: "plan-success", Skill: "planned", Input: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if result == nil || calls.Load() != 1 {
		t.Fatalf("unexpected plan execution: result=%+v calls=%d", result, calls.Load())
	}
	stored, err := store.Get(context.Background(), "plan-success")
	if err != nil {
		t.Fatal(err)
	}
	if stored.ExecutionMode != string(ExecutionModePlanExecute) || !strings.Contains(string(stored.Plan), "read-1") {
		t.Fatalf("plan was not tracked: mode=%s plan=%s", stored.ExecutionMode, stored.Plan)
	}
}

func TestPlanExecuteRejectsPlanOverToolBudget(t *testing.T) {
	skills := skill.NewRegistry()
	if err := skills.Register(skill.Definition{SkillName: "budget-plan", Mode: string(ExecutionModePlanExecute), AllowedTools: []string{"echo"}, Rounds: 2}); err != nil {
		t.Fatal(err)
	}
	registry := tools.NewRegistry()
	_ = registry.Register(tools.Func{ToolName: "echo", ToolRisk: permission.RiskRead, ToolMetadata: tools.Metadata{Idempotent: true}})
	runner, err := NewRunner(Config{
		Client: &fakeLLMClient{}, Skills: skills, Tools: registry, Tasks: task.NewMemoryStore(), Policy: permission.AllowReadOnly(), MaxToolCalls: 1,
		Planner: fakePlanner{plan: Plan{Steps: []PlannedStep{{Tool: "echo"}, {Tool: "echo"}}}}, Retry: RetryPolicy{MaxAttempts: 1}, Timeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = runner.Run(context.Background(), Request{TaskID: "plan-budget", Skill: "budget-plan", Input: "test"})
	if err == nil || !strings.Contains(err.Error(), "budget allows 1") {
		t.Fatalf("planner bypassed tool budget: %v", err)
	}
}

type blockingPlanner struct {
	started chan struct{}
}

func (planner blockingPlanner) Plan(ctx context.Context, _ PlanRequest) (Plan, error) {
	select {
	case planner.started <- struct{}{}:
	default:
	}
	<-ctx.Done()
	return Plan{}, ctx.Err()
}

func TestPlanExecuteRejectsSelfDependency(t *testing.T) {
	plan := Plan{Steps: []PlannedStep{{StepID: "self", Type: StepTool, Tool: "echo", DependsOn: []string{"self"}}}}
	err := validatePlan(plan, map[string]bool{"echo": true}, 1)
	if err == nil || !strings.Contains(err.Error(), "cannot depend on itself") {
		t.Fatalf("self dependency was accepted: %v", err)
	}
}

func TestPlanExecutePlannerTimeoutUsesRuntimeTimeoutState(t *testing.T) {
	skills := skill.NewRegistry()
	if err := skills.Register(skill.Definition{SkillName: "planned-timeout", Mode: string(ExecutionModePlanExecute), Rounds: 2}); err != nil {
		t.Fatal(err)
	}
	store := task.NewMemoryStore()
	runner, err := NewRunner(Config{
		Client: &fakeLLMClient{}, Skills: skills, Tasks: store, Policy: permission.AllowReadOnly(),
		Planner: blockingPlanner{started: make(chan struct{}, 1)}, Timeout: 25 * time.Millisecond,
		Retry: RetryPolicy{MaxAttempts: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = runner.Run(context.Background(), Request{TaskID: "plan-timeout", Skill: "planned-timeout", Input: "test"})
	if err == nil || !strings.Contains(err.Error(), "deadline exceeded") {
		t.Fatalf("unexpected planner timeout error: %v", err)
	}
	stored, getErr := store.Get(context.Background(), "plan-timeout")
	if getErr != nil {
		t.Fatal(getErr)
	}
	if stored.Status != task.StatusFailed || stored.Stage != "timeout" {
		t.Fatalf("planner timeout bypassed runtime terminal state: %+v", stored)
	}
}

func TestPlanExecutePlannerCancellationUsesRuntimeCancelledState(t *testing.T) {
	started := make(chan struct{}, 1)
	skills := skill.NewRegistry()
	if err := skills.Register(skill.Definition{SkillName: "planned-cancel", Mode: string(ExecutionModePlanExecute), Rounds: 2}); err != nil {
		t.Fatal(err)
	}
	store := task.NewMemoryStore()
	runner, err := NewRunner(Config{
		Client: &fakeLLMClient{}, Skills: skills, Tasks: store, Policy: permission.AllowReadOnly(),
		Planner: blockingPlanner{started: started}, Timeout: time.Second, Retry: RetryPolicy{MaxAttempts: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	resultC := make(chan error, 1)
	go func() {
		_, runErr := runner.Run(ctx, Request{TaskID: "plan-cancel", Skill: "planned-cancel", Input: "test"})
		resultC <- runErr
	}()
	<-started
	cancel()
	if runErr := <-resultC; runErr == nil || !strings.Contains(runErr.Error(), "context canceled") {
		t.Fatalf("unexpected planner cancellation error: %v", runErr)
	}
	stored, getErr := store.Get(context.Background(), "plan-cancel")
	if getErr != nil {
		t.Fatal(getErr)
	}
	if stored.Status != task.StatusCancelled || stored.Stage != "cancelled" {
		t.Fatalf("planner cancellation bypassed runtime terminal state: %+v", stored)
	}
}

func TestRunnerPersistsStructuredEvidenceAndContextTrace(t *testing.T) {
	client := &fakeLLMClient{items: []fakeLLMItem{
		{response: &llm.Response{Content: `{"action":"tool","tool":"get_symbol_snapshot","arguments":{}}`}},
		{response: &llm.Response{Content: `{"action":"final","result":{"ok":true}}`}},
	}}
	old := time.Now().UTC().Add(-10 * time.Minute).UnixMilli()
	snapshotTool := tools.Func{
		ToolName: "get_symbol_snapshot", ToolRisk: permission.RiskRead, ToolMetadata: tools.Metadata{Idempotent: true},
		ExecuteFunc: func(context.Context, json.RawMessage) (any, error) {
			return map[string]any{"symbol": "BTCUSDT", "price": 100.0, "updated_at_ms": old}, nil
		},
	}
	definition := skill.Definition{SkillName: "evidence", AllowedTools: []string{"get_symbol_snapshot"}, Rounds: 3}
	runner, store := newTestRunner(t, client, definition, snapshotTool)
	result, err := runner.Run(context.Background(), Request{TaskID: "evidence-task", Skill: "evidence", Input: "analyze"})
	if err != nil {
		t.Fatal(err)
	}
	second := client.request(1)
	joined := ""
	for _, message := range second.Messages {
		joined += message.Content + "\n"
	}
	if !strings.Contains(joined, `"evidence"`) || !strings.Contains(joined, "CONTEXT_FRESHNESS") || !strings.Contains(joined, "status=stale") {
		t.Fatalf("structured/stale evidence was not injected into LLM context: %s", joined)
	}
	stored, err := store.Get(context.Background(), result.TaskID)
	if err != nil {
		t.Fatal(err)
	}
	var steps []ExecutionStep
	if err := json.Unmarshal(stored.Steps, &steps); err != nil {
		t.Fatal(err)
	}
	foundEvidence := false
	foundTrace := false
	for _, step := range steps {
		if step.Type == StepTool && len(step.Evidence) == 1 {
			foundEvidence = step.Evidence[0].Source == "get_symbol_snapshot" && step.Evidence[0].ContentHash != "" && step.Evidence[0].Freshness == contextengine.FreshnessStale
		}
		if step.Type == StepLLM && step.ContextTrace != nil && step.ContextTrace.SelectedBlocks > 0 {
			foundTrace = true
		}
	}
	if !foundEvidence || !foundTrace {
		t.Fatalf("V2-2 audit data not persisted in steps: %s", stored.Steps)
	}
}

func TestRunnerTrimsLowPriorityHistoryInsteadOfFailingContext(t *testing.T) {
	client := &fakeLLMClient{items: []fakeLLMItem{{response: &llm.Response{Content: `{"action":"final","result":{"ok":true}}`}}}}
	definition := skill.Definition{
		SkillName: "trim-context", Rounds: 2,
		BuildInputFunc: func(context.Context, skill.Request) ([]llm.Message, error) {
			return []llm.Message{
				{Role: llm.RoleUser, Content: strings.Repeat("old-history-a ", 80)},
				{Role: llm.RoleAssistant, Content: strings.Repeat("old-history-b ", 80)},
				{Role: llm.RoleUser, Content: "current BTCUSDT task"},
			}, nil
		},
	}
	skills := skill.NewRegistry()
	if err := skills.Register(definition); err != nil {
		t.Fatal(err)
	}
	store := task.NewMemoryStore()
	runner, err := NewRunner(Config{
		Client: client, Skills: skills, Tools: tools.NewRegistry(), Tasks: store,
		Policy: permission.AllowReadOnly(), MaxContextTokens: 32, MaxContextBytes: 4096,
		Retry: RetryPolicy{MaxAttempts: 1}, Timeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := runner.Run(context.Background(), Request{TaskID: "trim-context-task", Skill: "trim-context", Input: "ignored"})
	if err != nil {
		t.Fatal(err)
	}
	request := client.request(0)
	joined := ""
	for _, message := range request.Messages {
		joined += message.Content
	}
	if !strings.Contains(joined, "current BTCUSDT task") || strings.Contains(joined, "old-history-a") {
		t.Fatalf("context priority trimming is wrong: %q", joined)
	}
	stored, err := store.Get(context.Background(), result.TaskID)
	if err != nil {
		t.Fatal(err)
	}
	foundTrimEvent := false
	for _, event := range stored.Events {
		if event.Stage == "context_trimmed" {
			foundTrimEvent = true
			break
		}
	}
	if !foundTrimEvent {
		t.Fatalf("context trim event missing: %+v", stored.Events)
	}
}

func TestRunnerLoadsOnlyActivatedAndRequestedSkillResources(t *testing.T) {
	client := &fakeLLMClient{items: []fakeLLMItem{{response: &llm.Response{Content: `{"action":"final","result":{"ok":true}}`}}}}
	loaded := []string{}
	definition := skill.Definition{
		SkillName: "resources", Rounds: 2,
		ContextResourcesFunc: func(skill.Request) []contextengine.Resource {
			return []contextengine.Resource{
				{ID: "skill-md", Type: contextengine.BlockSkillInstruction, Disclosure: contextengine.DisclosureActivation, Load: func(context.Context) (string, error) {
					loaded = append(loaded, "skill")
					return "activated skill instructions", nil
				}},
				{ID: "reference", Type: contextengine.BlockSkillInstruction, Disclosure: contextengine.DisclosureOnDemand, Load: func(context.Context) (string, error) {
					loaded = append(loaded, "reference")
					return "requested reference content", nil
				}},
			}
		},
	}
	runner, _ := newTestRunner(t, client, definition)
	if _, err := runner.Run(context.Background(), Request{TaskID: "resources-task", Skill: "resources", Input: "hello", Metadata: map[string]any{"context_resource_ids": []string{"reference"}}}); err != nil {
		t.Fatal(err)
	}
	if strings.Join(loaded, ",") != "skill,reference" {
		t.Fatalf("unexpected progressive disclosure loads: %v", loaded)
	}
	request := client.request(0)
	joined := ""
	for _, message := range request.Messages {
		joined += message.Content + "\n"
	}
	if !strings.Contains(joined, "activated skill instructions") || !strings.Contains(joined, "requested reference content") {
		t.Fatalf("resource context missing: %q", joined)
	}
}

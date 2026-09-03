package toolruntime

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"go_binance_futures/agent/contextengine"
	"go_binance_futures/agent/permission"
	"go_binance_futures/agent/tools"
)

type typedResult struct {
	Symbol      string   `json:"symbol"`
	AsOf        string   `json:"as_of"`
	DataMissing []string `json:"data_missing"`
}

func testRuntime(t *testing.T, policy permission.Policy, definitions ...tools.Tool) *Runtime {
	t.Helper()
	registry := tools.NewRegistry()
	for _, definition := range definitions {
		if err := registry.Register(definition); err != nil {
			t.Fatal(err)
		}
	}
	runtime, err := New(Config{Registry: registry, Policy: policy, ContextEngine: contextengine.New(), DefaultMaxResultBytes: 1024})
	if err != nil {
		t.Fatal(err)
	}
	return runtime
}

func TestDescriptorUsesSystemMetadata(t *testing.T) {
	tool := tools.Func{ToolName: "read", ToolDescription: "desc", ToolRisk: permission.RiskRead, ToolMetadata: tools.Metadata{Idempotent: true, Timeout: time.Second, CacheTTL: 2 * time.Second, ProviderRef: "local", InputSchema: json.RawMessage(`{"type":"object"}`)}}
	runtime := testRuntime(t, permission.AllowReadOnly(), tool)
	descriptor, ok := runtime.Descriptor("read")
	if !ok || descriptor.CanonicalName != "read" || descriptor.SourceType != SourceNative || descriptor.Risk != permission.RiskRead || !descriptor.Idempotent || descriptor.TimeoutMs != 1000 || descriptor.CachePolicy.TTLms != 2000 || descriptor.ProviderRef != "local" {
		t.Fatalf("unexpected descriptor: %+v", descriptor)
	}
}

func TestInputSchemaRejectsBeforeToolExecution(t *testing.T) {
	var calls atomic.Int32
	tool := tools.Func{ToolName: "schema", ToolRisk: permission.RiskRead, ToolMetadata: tools.Metadata{InputSchema: json.RawMessage(`{"type":"object","required":["symbol"],"additionalProperties":false,"properties":{"symbol":{"type":"string"}}}`)}, ExecuteFunc: func(context.Context, json.RawMessage) (any, error) {
		calls.Add(1)
		return map[string]any{"ok": true}, nil
	}}
	runtime := testRuntime(t, permission.AllowReadOnly(), tool)
	result, err := runtime.Execute(context.Background(), ExecuteRequest{SkillName: "test", AllowedTools: map[string]bool{"schema": true}, ToolName: "schema", Arguments: json.RawMessage(`{"symbol":7,"extra":true}`)})
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatalf("tool executed %d times", calls.Load())
	}
	if result.Envelope.ErrorType != ErrorInvalidInput || result.ToolError == nil {
		t.Fatalf("unexpected result: %+v err=%v", result.Envelope, result.ToolError)
	}
}

func TestOutputSchemaRejectsInvalidToolResult(t *testing.T) {
	var calls atomic.Int32
	tool := tools.Func{ToolName: "output_schema", ToolRisk: permission.RiskRead, ToolMetadata: tools.Metadata{
		OutputSchema: json.RawMessage(`{"type":"object","required":["symbol"],"properties":{"symbol":{"type":"string"}}}`),
	}, ExecuteFunc: func(context.Context, json.RawMessage) (any, error) {
		calls.Add(1)
		return map[string]any{"symbol": 7}, nil
	}}
	runtime := testRuntime(t, permission.AllowReadOnly(), tool)
	result, err := runtime.Execute(context.Background(), ExecuteRequest{SkillName: "test", AllowedTools: map[string]bool{"output_schema": true}, ToolName: "output_schema", Arguments: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("tool calls = %d, want 1", calls.Load())
	}
	if result.Envelope.ErrorType != ErrorInternal || result.ToolError == nil || result.Value != nil || len(result.Raw) != 0 {
		t.Fatalf("invalid output must not escape as a successful result: envelope=%+v value=%#v raw=%s err=%v", result.Envelope, result.Value, result.Raw, result.ToolError)
	}
}

func TestReadIdempotentCacheRestoresConcreteType(t *testing.T) {
	var calls atomic.Int32
	tool := tools.Func{ToolName: "cached", ToolRisk: permission.RiskRead, ToolMetadata: tools.Metadata{Idempotent: true, CacheTTL: time.Minute, OutputSchema: json.RawMessage(`{"type":"object"}`)}, ExecuteFunc: func(context.Context, json.RawMessage) (any, error) {
		calls.Add(1)
		return typedResult{Symbol: "BTCUSDT", AsOf: time.Now().UTC().Format(time.RFC3339)}, nil
	}, RestoreCheckpointFunc: func(raw json.RawMessage) (any, error) {
		var value typedResult
		err := json.Unmarshal(raw, &value)
		return value, err
	}}
	runtime := testRuntime(t, permission.AllowReadOnly(), tool)
	req := ExecuteRequest{SkillName: "test", AllowedTools: map[string]bool{"cached": true}, ToolName: "cached", Arguments: json.RawMessage(`{}`)}
	first, err := runtime.Execute(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	second, err := runtime.Execute(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 || first.Envelope.CacheHit || !second.Envelope.CacheHit {
		t.Fatalf("cache behavior calls=%d first=%v second=%v", calls.Load(), first.Envelope.CacheHit, second.Envelope.CacheHit)
	}
	if _, ok := second.Value.(typedResult); !ok {
		t.Fatalf("cached concrete type = %T", second.Value)
	}
}

func TestTimeoutAndPartialAreStructured(t *testing.T) {
	slow := tools.Func{ToolName: "slow", ToolRisk: permission.RiskRead, ToolMetadata: tools.Metadata{Timeout: 20 * time.Millisecond}, ExecuteFunc: func(ctx context.Context, _ json.RawMessage) (any, error) { <-ctx.Done(); return nil, ctx.Err() }}
	runtime := testRuntime(t, permission.AllowReadOnly(), slow)
	result, err := runtime.Execute(context.Background(), ExecuteRequest{SkillName: "test", AllowedTools: map[string]bool{"slow": true}, ToolName: "slow", Arguments: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	if result.Envelope.ErrorType != ErrorTimeout || result.ToolError == nil || !strings.Contains(result.Envelope.Warnings[0], "deadline") {
		t.Fatalf("unexpected timeout envelope: %+v", result.Envelope)
	}

	partialTool := tools.Func{ToolName: "partial", ToolRisk: permission.RiskRead, ToolMetadata: tools.Metadata{MaxResultBytes: 48}, ExecuteFunc: func(context.Context, json.RawMessage) (any, error) {
		return typedResult{Symbol: "BTCUSDT", AsOf: time.Now().UTC().Format(time.RFC3339), DataMissing: []string{"depth"}}, nil
	}}
	runtime = testRuntime(t, permission.AllowReadOnly(), partialTool)
	partial, err := runtime.Execute(context.Background(), ExecuteRequest{SkillName: "test", AllowedTools: map[string]bool{"partial": true}, ToolName: "partial", Arguments: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	if !partial.Envelope.Partial || partial.Envelope.ErrorType != ErrorPartial || len(partial.Envelope.Warnings) == 0 {
		t.Fatalf("unexpected partial envelope: %+v", partial.Envelope)
	}
}

func TestStaleEvidenceIsStructured(t *testing.T) {
	stale := tools.Func{ToolName: "get_symbol_snapshot", ToolRisk: permission.RiskRead, ExecuteFunc: func(context.Context, json.RawMessage) (any, error) {
		return map[string]any{"symbol": "BTCUSDT", "updateTime": time.Now().Add(-10 * time.Minute).UnixMilli()}, nil
	}}
	runtime := testRuntime(t, permission.AllowReadOnly(), stale)
	result, err := runtime.Execute(context.Background(), ExecuteRequest{SkillName: "test", AllowedTools: map[string]bool{"get_symbol_snapshot": true}, ToolName: "get_symbol_snapshot", Arguments: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	if result.Envelope.ErrorType != ErrorStale || len(result.Evidence) == 0 || result.Evidence[0].Freshness != contextengine.FreshnessStale {
		t.Fatalf("unexpected stale result: %+v evidence=%+v", result.Envelope, result.Evidence)
	}
}

func TestPermissionUsesRegisteredRisk(t *testing.T) {
	write := tools.Func{ToolName: "write", ToolRisk: permission.RiskWrite, ExecuteFunc: func(context.Context, json.RawMessage) (any, error) { return map[string]bool{"ok": true}, nil }}
	runtime := testRuntime(t, permission.AllowReadOnly(), write)
	_, err := runtime.Execute(context.Background(), ExecuteRequest{SkillName: "test", AllowedTools: map[string]bool{"write": true}, ToolName: "write", Arguments: json.RawMessage(`{"risk":"read"}`)})
	if err == nil || TypeOf(err) != ErrorPermission {
		t.Fatalf("expected permission error, got %v", err)
	}
}

func TestBatchRunsSafeReadsInParallelAndWritesSequentially(t *testing.T) {
	var active atomic.Int32
	var maxActive atomic.Int32
	read := tools.Func{ToolName: "read", ToolRisk: permission.RiskRead, ToolMetadata: tools.Metadata{Idempotent: true}, ExecuteFunc: func(context.Context, json.RawMessage) (any, error) {
		now := active.Add(1)
		for {
			old := maxActive.Load()
			if now <= old || maxActive.CompareAndSwap(old, now) {
				break
			}
		}
		time.Sleep(30 * time.Millisecond)
		active.Add(-1)
		return map[string]bool{"ok": true}, nil
	}}
	runtime := testRuntime(t, permission.AllowReadOnly(), read)
	requests := []ExecuteRequest{{SkillName: "test", AllowedTools: map[string]bool{"read": true}, ToolName: "read", Arguments: json.RawMessage(`{"n":1}`)}, {SkillName: "test", AllowedTools: map[string]bool{"read": true}, ToolName: "read", Arguments: json.RawMessage(`{"n":2}`)}}
	results := runtime.ExecuteBatch(context.Background(), []BatchRequest{{Request: requests[0]}, {Request: requests[1]}}, 2)
	if len(results) != 2 || maxActive.Load() < 2 || results[0].Err != nil || results[1].Err != nil {
		t.Fatalf("reads did not run in parallel: max=%d results=%+v", maxActive.Load(), results)
	}

	active.Store(0)
	maxActive.Store(0)
	write := tools.Func{ToolName: "write", ToolRisk: permission.RiskWrite, ToolMetadata: tools.Metadata{Idempotent: false}, ExecuteFunc: read.ExecuteFunc}
	policy := permission.AllowWritesFor(map[string][]string{"test": {"write"}})
	runtime = testRuntime(t, policy, write)
	results = runtime.ExecuteBatch(context.Background(), []BatchRequest{{Request: ExecuteRequest{SkillName: "test", AllowedTools: map[string]bool{"write": true}, ToolName: "write", Arguments: json.RawMessage(`{}`)}}, {Request: ExecuteRequest{SkillName: "test", AllowedTools: map[string]bool{"write": true}, ToolName: "write", Arguments: json.RawMessage(`{}`)}}}, 2)
	if maxActive.Load() != 1 || results[0].Err != nil || results[1].Err != nil {
		t.Fatalf("writes must be sequential: max=%d results=%+v", maxActive.Load(), results)
	}
}

func TestFatalLookupErrorTaxonomy(t *testing.T) {
	runtime := testRuntime(t, permission.AllowReadOnly())
	_, err := runtime.Execute(context.Background(), ExecuteRequest{SkillName: "test", ToolName: "missing"})
	if err == nil || TypeOf(err) != ErrorNotFound || !errors.As(err, new(*Error)) {
		t.Fatalf("unexpected error: %v", err)
	}
}

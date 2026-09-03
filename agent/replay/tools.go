package replay

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go_binance_futures/agent/permission"
	agenttools "go_binance_futures/agent/tools"
	symbolanalysisservice "go_binance_futures/service/symbolanalysis"
)

type fixtureTool struct {
	name     string
	metadata ToolMetadata
	mu       sync.Mutex
	steps    []ToolStep
	index    int
}

func (tool *fixtureTool) Name() string        { return tool.name }
func (tool *fixtureTool) Description() string { return "fixed replay fixture tool" }
func (tool *fixtureTool) Risk() permission.RiskLevel {
	if tool.metadata.Risk == "" {
		return permission.RiskRead
	}
	return tool.metadata.Risk
}
func (tool *fixtureTool) Metadata() agenttools.Metadata {
	return agenttools.Metadata{
		InputSchema: tool.metadata.InputSchema, OutputSchema: tool.metadata.OutputSchema,
		Timeout: time.Duration(tool.metadata.TimeoutMs) * time.Millisecond, MaxResultBytes: tool.metadata.MaxResultBytes,
		Idempotent: tool.metadata.Idempotent || tool.metadata.Risk == "" || tool.metadata.Risk == permission.RiskRead,
		SourceType: tool.metadata.SourceType, ProviderRef: tool.metadata.ProviderRef,
		CacheTTL: time.Duration(tool.metadata.CacheTTLms) * time.Millisecond,
	}
}

func (tool *fixtureTool) Execute(ctx context.Context, _ json.RawMessage) (any, error) {
	tool.mu.Lock()
	if tool.index >= len(tool.steps) {
		tool.mu.Unlock()
		return nil, fmt.Errorf("replay tool %s has no step %d", tool.name, tool.index+1)
	}
	step := tool.steps[tool.index]
	tool.index++
	tool.mu.Unlock()
	if step.DelayMs > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(step.DelayMs) * time.Millisecond):
		}
	}
	if step.Error != "" {
		return nil, fmt.Errorf("%s", step.Error)
	}
	return decodeToolResult(tool.name, step.Result)
}

func decodeToolResult(name string, raw json.RawMessage) (any, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	if name == "get_symbol_analysis_context" {
		var value symbolanalysisservice.Context
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, err
		}
		return value, nil
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	return value, nil
}

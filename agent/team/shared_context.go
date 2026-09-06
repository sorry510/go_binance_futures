package team

import (
	"context"
	"encoding/json"
	"fmt"

	agentruntime "go_binance_futures/agent/runtime"
	"go_binance_futures/agent/toolruntime"
)

type ToolSharedContextExecutor struct {
	Runtime  *toolruntime.Runtime
	Observer agentruntime.Observer
}

func (executor ToolSharedContextExecutor) Execute(ctx context.Context, taskID, symbol string, budget int) (SharedContextResult, error) {
	if executor.Runtime == nil {
		return SharedContextResult{}, fmt.Errorf("team shared-context tool runtime is required")
	}
	arguments, _ := json.Marshal(map[string]string{"symbol": symbol})
	result, err := executor.Runtime.Execute(ctx, toolruntime.ExecuteRequest{
		SkillName: SymbolAnalysisTeam, AllowedTools: map[string]bool{"get_symbol_analysis_context": true},
		ToolName: "get_symbol_analysis_context", Arguments: arguments, CallIndex: 1, CallBudget: budget,
	})
	if err != nil {
		return SharedContextResult{}, err
	}
	status := "success"
	errorText := ""
	if result.ToolError != nil {
		status, errorText = "error", result.ToolError.Error()
	}
	if executor.Observer != nil {
		executor.Observer.Observe(agentruntime.Observation{
			Type: "tool_call", TaskID: taskID, Skill: SymbolAnalysisTeam, Tool: result.Descriptor.CanonicalName,
			ToolSource: string(result.Descriptor.SourceType), ProviderRef: result.Descriptor.ProviderRef,
			ProtocolVersion: result.Descriptor.ProtocolVersion, CatalogHash: result.Descriptor.CatalogHash,
			SchemaHash: result.Descriptor.SchemaHash, Status: status, Error: errorText,
			DurationMs: result.Trace.DurationMs, CacheHit: result.Trace.CacheHit, Partial: result.Trace.Partial,
			RawSize: result.Trace.RawSize, ContentHash: result.Trace.ContentHash, EvidenceCount: len(result.Evidence),
		})
	}
	if result.ToolError != nil {
		return SharedContextResult{}, result.ToolError
	}
	return SharedContextResult{
		Raw: append(json.RawMessage(nil), result.Raw...), ContentHash: result.Trace.ContentHash,
		DurationMs: result.Trace.DurationMs, Partial: result.Trace.Partial,
	}, nil
}

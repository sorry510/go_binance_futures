package app

import (
	"context"
	"fmt"

	"go_binance_futures/agent/contextengine"
	"go_binance_futures/agent/memory"
	agentruntime "go_binance_futures/agent/runtime"
	"go_binance_futures/agent/skill"
	"go_binance_futures/agent/task"
)

var defaultMemoryService = memory.Service{}

func MemoryContext(ctx context.Context, skillName string, req skill.Request) ([]contextengine.ContextBlock, error) {
	return defaultMemoryService.Context(ctx, skillName, req)
}

func MemoryWrite(ctx context.Context, req agentruntime.Request, item *task.Task, result *agentruntime.Result) ([]string, error) {
	summary := ""
	if result != nil {
		summary = result.Summary
	}
	stored, err := defaultMemoryService.PersistTaskSummary(ctx, req.Skill, req.Input, req.Metadata, item, summary)
	if err != nil || stored == nil {
		return nil, err
	}
	return []string{fmt.Sprintf("memory:%d", stored.ID)}, nil
}

package eval

import (
	"context"
	"fmt"

	"go_binance_futures/agent/permission"
	agentruntime "go_binance_futures/agent/runtime"
	"go_binance_futures/agent/skill"
	"go_binance_futures/agent/task"
	"go_binance_futures/agent/tools"
	"go_binance_futures/llm"
)

type ShadowResult struct {
	Result *agentruntime.Result `json:"-"`
	Task   *task.Task           `json:"task,omitempty"`
	Err    error                `json:"-"`
}

func RunShadow(ctx context.Context, client llm.Client, definition skill.Skill, registry *tools.Registry, req agentruntime.Request, config agentruntime.Config) ShadowResult {
	if definition == nil || client == nil {
		return ShadowResult{Err: fmt.Errorf("shadow requires client and skill")}
	}
	info := skill.ResolveVersionInfo(definition, definition.SystemPrompt())
	if info.Source != skill.DefaultSource {
		return ShadowResult{Err: fmt.Errorf("portable/imported skills are not shadow-enabled before V2-6")}
	}
	if registry == nil {
		registry = tools.NewRegistry()
	}
	for _, name := range definition.Tools() {
		selected, ok := registry.Get(name)
		if !ok {
			continue
		}
		if selected.Risk() != permission.RiskRead || !selected.Metadata().Idempotent {
			return ShadowResult{Err: fmt.Errorf("shadow rejects non-read/non-idempotent tool %q", name)}
		}
	}
	skills := skill.NewRegistry()
	if err := skills.Register(definition); err != nil {
		return ShadowResult{Err: err}
	}
	store := task.NewMemoryStore()
	config.Client, config.Skills, config.Tools, config.Tasks = client, skills, registry, store
	config.Policy, config.ToolRuntime = permission.AllowReadOnly(), nil
	config.Observer, config.EventHook, config.MessageHook, config.ValidationHook = nil, nil, nil, nil
	runner, err := agentruntime.NewRunner(config)
	if err != nil {
		return ShadowResult{Err: err}
	}
	req.Skill = definition.Name()
	req.TaskID = ""
	result, runErr := runner.Run(ctx, req)
	if result == nil {
		return ShadowResult{Result: result, Err: runErr}
	}
	stored, _ := store.Get(context.Background(), result.TaskID)
	return ShadowResult{Result: result, Task: stored, Err: runErr}
}

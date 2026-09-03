package replay

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go_binance_futures/agent/permission"
	agentruntime "go_binance_futures/agent/runtime"
	"go_binance_futures/agent/skill"
	"go_binance_futures/agent/task"
	agenttools "go_binance_futures/agent/tools"
)

type RunResult struct {
	Result *agentruntime.Result
	Task   *task.Task
	Err    error
}

func Run(ctx context.Context, fixture Fixture, definition skill.Skill) RunResult {
	if definition == nil || definition.Name() != fixture.Skill {
		return RunResult{Err: fmt.Errorf("fixture skill %q does not match definition", fixture.Skill)}
	}
	skills := skill.NewRegistry()
	if err := skills.Register(definition); err != nil {
		return RunResult{Err: err}
	}
	toolRegistry := agenttools.NewRegistry()
	for name, steps := range fixture.Tools {
		if err := toolRegistry.Register(&fixtureTool{name: name, metadata: fixture.ToolMetadata[name], steps: append([]ToolStep(nil), steps...)}); err != nil {
			return RunResult{Err: err}
		}
	}
	store := task.NewMemoryStore()
	timeout := 2 * time.Second
	if fixture.TimeoutMs > 0 {
		timeout = time.Duration(fixture.TimeoutMs) * time.Millisecond
	}
	cfg := agentruntime.Config{
		Client: &scriptedClient{steps: append([]LLMStep(nil), fixture.LLM...), modelConfigID: fixture.ModelConfigID},
		Skills: skills, Tools: toolRegistry, Tasks: store, Policy: permission.AllowReadOnly(), Timeout: timeout,
		Retry: agentruntime.RetryPolicy{MaxAttempts: 1},
	}
	if fixture.MaxContextBytes > 0 {
		cfg.MaxContextBytes = fixture.MaxContextBytes
	}
	runner, err := agentruntime.NewRunner(cfg)
	if err != nil {
		return RunResult{Err: err}
	}
	taskID := "replay_" + sanitizeName(fixture.Name)
	result, runErr := runner.Run(ctx, agentruntime.Request{TaskID: taskID, Skill: fixture.Skill, Input: fixture.Input})
	stored, getErr := store.Get(context.Background(), taskID)
	if getErr != nil {
		return RunResult{Result: result, Err: runErr}
	}
	return RunResult{Result: result, Task: stored, Err: runErr}
}

func sanitizeName(value string) string {
	value = strings.TrimSpace(value)
	var out strings.Builder
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '-' || char == '_' {
			out.WriteRune(char)
		} else {
			out.WriteByte('_')
		}
	}
	if out.Len() == 0 {
		return "case"
	}
	return out.String()
}

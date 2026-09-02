package runtime

import (
	"context"
	"fmt"

	"go_binance_futures/agent/permission"
	"go_binance_futures/agent/task"
	"go_binance_futures/agent/tools"
)

type DefaultRunner struct {
	cfg Config
}

func NewRunner(cfg Config) (*DefaultRunner, error) {
	defaults := DefaultConfig()
	if cfg.Client == nil {
		return nil, fmt.Errorf("agent runtime requires an LLM client")
	}
	if cfg.Skills == nil {
		return nil, fmt.Errorf("agent runtime requires a skill registry")
	}
	if cfg.Tools == nil {
		cfg.Tools = tools.NewRegistry()
	}
	if cfg.Tasks == nil {
		cfg.Tasks = task.NewMemoryStore()
	}
	if cfg.Policy == nil {
		policy := permission.AllowReadOnly()
		cfg.Policy = policy
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaults.Timeout
	}
	if cfg.DefaultMaxRounds <= 0 {
		cfg.DefaultMaxRounds = defaults.DefaultMaxRounds
	}
	if cfg.MaxContextBytes <= 0 {
		cfg.MaxContextBytes = defaults.MaxContextBytes
	}
	if cfg.MaxToolResultBytes <= 0 {
		cfg.MaxToolResultBytes = defaults.MaxToolResultBytes
	}
	if cfg.MaxToolCalls <= 0 {
		cfg.MaxToolCalls = defaults.MaxToolCalls
	}
	if cfg.MaxTotalTokens <= 0 {
		cfg.MaxTotalTokens = defaults.MaxTotalTokens
	}
	if cfg.Retry.MaxAttempts <= 0 {
		cfg.Retry.MaxAttempts = defaults.Retry.MaxAttempts
	}
	if cfg.Retry.Delay < 0 {
		return nil, fmt.Errorf("agent retry delay cannot be negative")
	}
	return &DefaultRunner{cfg: cfg}, nil
}

func (runner *DefaultRunner) Run(ctx context.Context, req Request) (*Result, error) {
	return (&coordinator{runner: runner}).run(ctx, req)
}

// Resume continues a cancelled, timed-out or restart-interrupted task from its
// last safe checkpoint. The caller must construct this runner with the task's
// frozen model configuration before calling Resume.
func (runner *DefaultRunner) Resume(ctx context.Context, taskID string) (*Result, error) {
	return (&coordinator{runner: runner}).resume(ctx, taskID)
}

// ValidateResume performs the same safety checks as Resume without mutating the task.
func (runner *DefaultRunner) ValidateResume(ctx context.Context, taskID string) error {
	return (&coordinator{runner: runner}).validateResume(ctx, taskID)
}

// ResumeRequest returns the frozen request identity reconstructed from a safe
// checkpoint. It is intended for lifecycle owners that need the original
// Runtime-owned metadata when invoking completion hooks after Resume.
func (runner *DefaultRunner) ResumeRequest(ctx context.Context, taskID string) (Request, error) {
	session, err := (&coordinator{runner: runner}).prepareResumeState(ctx, taskID, false)
	if err != nil {
		return Request{}, err
	}
	return session.req, nil
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}

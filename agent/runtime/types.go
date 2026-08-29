package runtime

import (
	"context"
	"encoding/json"
	"time"

	"go_binance_futures/agent/permission"
	"go_binance_futures/agent/skill"
	"go_binance_futures/agent/task"
	"go_binance_futures/agent/tools"
	"go_binance_futures/llm"
)

type Request struct {
	Skill          string         `json:"skill"`
	Input          string         `json:"input"`
	ConversationID string         `json:"conversation_id,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

type Result struct {
	TaskID  string          `json:"task_id"`
	Skill   string          `json:"skill"`
	Summary string          `json:"summary,omitempty"`
	Raw     json.RawMessage `json:"raw"`
	Value   any             `json:"value,omitempty"`
	Usage   task.Usage      `json:"usage,omitempty"`
}

type RetryPolicy struct {
	MaxAttempts int
	Delay       time.Duration
}

type Config struct {
	Client             llm.Client
	Skills             *skill.Registry
	Tools              *tools.Registry
	Tasks              task.Store
	Policy             permission.Policy
	Timeout            time.Duration
	DefaultMaxRounds   int
	MaxContextBytes    int
	MaxToolResultBytes int
	MaxToolCalls       int
	Retry              RetryPolicy
	EventHook          func(task.Event)
}

type Runner interface {
	Run(ctx context.Context, req Request) (*Result, error)
}

func DefaultConfig() Config {
	return Config{
		Timeout:            2 * time.Minute,
		DefaultMaxRounds:   8,
		MaxContextBytes:    256 * 1024,
		MaxToolResultBytes: 128 * 1024,
		MaxToolCalls:       20,
		Retry:              RetryPolicy{MaxAttempts: 2, Delay: time.Second},
	}
}

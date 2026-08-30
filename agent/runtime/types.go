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
	TaskID         string         `json:"task_id,omitempty"`
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

type Budget struct {
	MaxToolCalls   int `json:"max_tool_calls"`
	MaxTotalTokens int `json:"max_total_tokens"`
}

type BudgetProvider func(skill string) Budget

type Observation struct {
	Type           string     `json:"type"`
	TaskID         string     `json:"task_id"`
	ConversationID string     `json:"conversation_id,omitempty"`
	Skill          string     `json:"skill"`
	Provider       string     `json:"provider,omitempty"`
	Model          string     `json:"model,omitempty"`
	Tool           string     `json:"tool,omitempty"`
	Status         string     `json:"status,omitempty"`
	ErrorType      string     `json:"error_type,omitempty"`
	Error          string     `json:"error,omitempty"`
	Round          int        `json:"round,omitempty"`
	DurationMs     int64      `json:"duration_ms,omitempty"`
	Usage          task.Usage `json:"usage,omitempty"`
}

type Observer interface {
	Observe(Observation)
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
	MaxTotalTokens     int
	BudgetProvider     BudgetProvider
	Observer           Observer
	Retry              RetryPolicy
	EventHook          func(task.Event)
	MessageHook        func(taskID string, message llm.Message)
	ValidationHook     func(taskID string, raw json.RawMessage, err error)
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
		MaxTotalTokens:     120000,
		Retry:              RetryPolicy{MaxAttempts: 2, Delay: time.Second},
	}
}

package team

import (
	"context"
	"encoding/json"
	"time"

	agentruntime "go_binance_futures/agent/runtime"
	"go_binance_futures/agent/task"
)

const SymbolAnalysisTeam = "symbol_analysis_team"

type Input struct {
	Symbol string `json:"symbol"`
	Prompt string `json:"prompt,omitempty"`
}

type ChildTask struct {
	TaskID      string     `json:"task_id"`
	Role        string     `json:"role"`
	Skill       string     `json:"skill"`
	Status      string     `json:"status"`
	DurationMs  int64      `json:"duration_ms"`
	Usage       task.Usage `json:"usage,omitempty"`
	DataMissing []string   `json:"data_missing,omitempty"`
	Error       string     `json:"error,omitempty"`
}

type Evidence struct {
	Role    string `json:"role"`
	Source  string `json:"source"`
	Finding string `json:"finding"`
}

type ResultV1 struct {
	Version       string     `json:"version"`
	Team          string     `json:"team"`
	Symbol        string     `json:"symbol"`
	Status        string     `json:"status"`
	Direction     string     `json:"direction"`
	Confidence    float64    `json:"confidence"`
	Summary       string     `json:"summary"`
	Consensus     []string   `json:"consensus"`
	Disagreements []string   `json:"disagreements"`
	DataMissing   []string   `json:"data_missing"`
	Evidence      []Evidence `json:"evidence"`
}

type TaskManager interface {
	StartLinked(agentruntime.Request, task.Linkage) (*task.Task, error)
	Get(context.Context, string) (*task.Task, error)
	Cancel(context.Context, string) error
}

type StartOptions struct {
	ConversationID string
}

type Config struct {
	Manager        TaskManager
	Store          task.Store
	SharedContext  SharedContextExecutor
	Observer       agentruntime.Observer
	CompletionHook func(*task.Task)
	PollInterval   time.Duration
	ChildTimeout   time.Duration
	MaxConcurrency int
	MaxTotalTokens int
	MaxToolCalls   int
}

type SharedContextResult struct {
	Raw         json.RawMessage
	ContentHash string
	DurationMs  int64
	Partial     bool
}

type SharedContextExecutor interface {
	Execute(context.Context, string, string, int) (SharedContextResult, error)
}

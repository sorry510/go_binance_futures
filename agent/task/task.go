package task

import (
	"encoding/json"
	"time"
)

type Status string

const (
	StatusQueued      Status = "queued"
	StatusRunning     Status = "running"
	StatusWaitingLLM  Status = "waiting_llm"
	StatusWaitingTool Status = "waiting_tool"
	StatusValidating  Status = "validating"
	StatusSucceeded   Status = "succeeded"
	StatusFailed      Status = "failed"
	StatusCancelled   Status = "cancelled"
	StatusInterrupted Status = "interrupted"
)

type Event struct {
	TaskID     string    `json:"task_id"`
	Stage      string    `json:"stage"`
	Progress   int       `json:"progress"`
	Round      int       `json:"round,omitempty"`
	Message    string    `json:"message,omitempty"`
	Skill      string    `json:"skill,omitempty"`
	Tool       string    `json:"tool,omitempty"`
	Status     string    `json:"status,omitempty"`
	DurationMs int64     `json:"duration_ms,omitempty"`
	Time       time.Time `json:"time"`
}

type Usage struct {
	InputTokens  int `json:"input_tokens,omitempty"`
	OutputTokens int `json:"output_tokens,omitempty"`
	TotalTokens  int `json:"total_tokens,omitempty"`
}

type Task struct {
	ID          string          `json:"id"`
	Skill          string          `json:"skill"`
	ConversationID string          `json:"conversation_id,omitempty"`
	Status      Status          `json:"status"`
	Stage       string          `json:"stage"`
	Progress    int             `json:"progress"`
	Input       string          `json:"input"`
	Result      json.RawMessage `json:"result,omitempty"`
	Error       string          `json:"error,omitempty"`
	Round       int             `json:"round"`
	MaxRounds   int             `json:"max_rounds"`
	Provider    string          `json:"provider,omitempty"`
	Model       string          `json:"model,omitempty"`
	Usage       Usage           `json:"usage,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	StartedAt   *time.Time      `json:"started_at,omitempty"`
	UpdatedAt   time.Time       `json:"updated_at"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
	Events         []Event         `json:"events,omitempty"`
}

func IsRunningStatus(status Status) bool {
	switch status {
	case StatusQueued, StatusRunning, StatusWaitingLLM, StatusWaitingTool, StatusValidating:
		return true
	default:
		return false
	}
}

func IsTerminalStatus(status Status) bool {
	return status == StatusSucceeded || status == StatusFailed || status == StatusCancelled || status == StatusInterrupted
}

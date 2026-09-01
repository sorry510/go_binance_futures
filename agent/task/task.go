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
	StepID     string    `json:"step_id,omitempty"`
	StepType   string    `json:"step_type,omitempty"`
	Stage      string    `json:"stage"`
	Progress   int       `json:"progress"`
	Round      int       `json:"round,omitempty"`
	Message    string    `json:"message,omitempty"`
	Skill      string    `json:"skill,omitempty"`
	Tool       string    `json:"tool,omitempty"`
	Status     string    `json:"status,omitempty"`
	ErrorType  string    `json:"error_type,omitempty"`
	Checkpoint bool      `json:"checkpoint,omitempty"`
	DurationMs int64     `json:"duration_ms,omitempty"`
	Time       time.Time `json:"time"`
}

type Usage struct {
	InputTokens  int `json:"input_tokens,omitempty"`
	OutputTokens int `json:"output_tokens,omitempty"`
	TotalTokens  int `json:"total_tokens,omitempty"`
}

type VersionMetadata struct {
	RuntimeVersion        string `json:"runtime_version"`
	SkillVersion          string `json:"skill_version"`
	PromptVersion         string `json:"prompt_version"`
	PromptHash            string `json:"prompt_hash"`
	ModelConfigID         int64  `json:"model_config_id"`
	InputContractVersion  string `json:"input_contract_version"`
	OutputContractVersion string `json:"output_contract_version"`
	SkillSource           string `json:"skill_source"`
	SkillSourceVersion    string `json:"skill_source_version,omitempty"`
}

type Task struct {
	ID                    string          `json:"id"`
	Skill                 string          `json:"skill"`
	ConversationID        string          `json:"conversation_id,omitempty"`
	Status                Status          `json:"status"`
	Stage                 string          `json:"stage"`
	Progress              int             `json:"progress"`
	Input                 string          `json:"input"`
	Result                json.RawMessage `json:"result,omitempty"`
	Error                 string          `json:"error,omitempty"`
	Round                 int             `json:"round"`
	MaxRounds             int             `json:"max_rounds"`
	Provider              string          `json:"provider,omitempty"`
	Model                 string          `json:"model,omitempty"`
	ExecutionMode         string          `json:"execution_mode,omitempty"`
	Plan                  json.RawMessage `json:"plan,omitempty"`
	Steps                 json.RawMessage `json:"steps,omitempty"`
	ResumeCount           int             `json:"resume_count,omitempty"`
	CheckpointJSON        string          `json:"-"`
	RuntimeVersion        string          `json:"runtime_version"`
	SkillVersion          string          `json:"skill_version"`
	PromptVersion         string          `json:"prompt_version"`
	PromptHash            string          `json:"prompt_hash"`
	ModelConfigID         int64           `json:"model_config_id"`
	InputContractVersion  string          `json:"input_contract_version"`
	OutputContractVersion string          `json:"output_contract_version"`
	SkillSource           string          `json:"skill_source"`
	SkillSourceVersion    string          `json:"skill_source_version,omitempty"`
	Usage                 Usage           `json:"usage,omitempty"`
	CreatedAt             time.Time       `json:"created_at"`
	StartedAt             *time.Time      `json:"started_at,omitempty"`
	UpdatedAt             time.Time       `json:"updated_at"`
	CompletedAt           *time.Time      `json:"completed_at,omitempty"`
	Events                []Event         `json:"events,omitempty"`
}

func (item *Task) ApplyVersionMetadata(value VersionMetadata) {
	if item == nil {
		return
	}
	item.RuntimeVersion = value.RuntimeVersion
	item.SkillVersion = value.SkillVersion
	item.PromptVersion = value.PromptVersion
	item.PromptHash = value.PromptHash
	item.ModelConfigID = value.ModelConfigID
	item.InputContractVersion = value.InputContractVersion
	item.OutputContractVersion = value.OutputContractVersion
	item.SkillSource = value.SkillSource
	item.SkillSourceVersion = value.SkillSourceVersion
}

func (item *Task) VersionMetadata() VersionMetadata {
	if item == nil {
		return VersionMetadata{}
	}
	return VersionMetadata{
		RuntimeVersion: item.RuntimeVersion, SkillVersion: item.SkillVersion, PromptVersion: item.PromptVersion,
		PromptHash: item.PromptHash, ModelConfigID: item.ModelConfigID, InputContractVersion: item.InputContractVersion,
		OutputContractVersion: item.OutputContractVersion, SkillSource: item.SkillSource, SkillSourceVersion: item.SkillSourceVersion,
	}
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

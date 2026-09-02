package runtime

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go_binance_futures/agent/task"
	"go_binance_futures/llm"
)

type ExecutionMode string

const (
	ExecutionModeReact       ExecutionMode = "react"
	ExecutionModeWorkflow    ExecutionMode = "workflow"
	ExecutionModePlanExecute ExecutionMode = "plan_execute"
)

type StepType string

const (
	StepPlan          StepType = "plan"
	StepBuildContext  StepType = "build_context"
	StepLLM           StepType = "llm"
	StepTool          StepType = "tool"
	StepParallelTools StepType = "parallel_tools"
	StepValidate      StepType = "validate"
	StepApproval      StepType = "approval"
	StepFinalize      StepType = "finalize"
)

type StepStatus string

const (
	StepPending   StepStatus = "pending"
	StepRunning   StepStatus = "running"
	StepSucceeded StepStatus = "succeeded"
	StepFailed    StepStatus = "failed"
	StepSkipped   StepStatus = "skipped"
)

type ExecutionStep struct {
	StepID        string     `json:"step_id"`
	Type          StepType   `json:"type"`
	Status        StepStatus `json:"status"`
	Attempt       int        `json:"attempt"`
	DependsOn     []string   `json:"depends_on,omitempty"`
	InputSummary  string     `json:"input_summary,omitempty"`
	OutputSummary string     `json:"output_summary,omitempty"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	ErrorType     string     `json:"error_type,omitempty"`
	Error         string     `json:"error,omitempty"`
	Checkpoint    bool       `json:"checkpoint,omitempty"`
}

type RunState struct {
	Version         string                     `json:"version"`
	TaskID          string                     `json:"task_id"`
	Skill           string                     `json:"skill"`
	Mode            ExecutionMode              `json:"mode"`
	Snapshot        ExecutionSnapshot          `json:"snapshot"`
	Messages        []llm.Message              `json:"messages"`
	Round           int                        `json:"round"`
	NextRound       int                        `json:"next_round"`
	MaxRounds       int                        `json:"max_rounds"`
	ToolCalls       int                        `json:"tool_calls"`
	MaxToolCalls    int                        `json:"max_tool_calls"`
	MaxTotalTokens  int                        `json:"max_total_tokens"`
	RequiredTools   map[string]bool            `json:"required_tools,omitempty"`
	SuccessfulTools map[string]bool            `json:"successful_tools,omitempty"`
	ToolResults     map[string]json.RawMessage `json:"tool_results,omitempty"`
	Plan            *Plan                      `json:"plan,omitempty"`
	PlanCursor      int                        `json:"plan_cursor,omitempty"`
	Steps           []ExecutionStep            `json:"steps"`
	StepSequence    int                        `json:"step_sequence"`
	ResumeSafe      bool                       `json:"resume_safe"`
	CheckpointAt    *time.Time                 `json:"checkpoint_at,omitempty"`
	ResumeMetadata  map[string]string          `json:"resume_metadata,omitempty"`
}

const runStateVersion = "runtime_state_v1"

func newRunState(taskID, skillName string, mode ExecutionMode, snapshot ExecutionSnapshot, maxRounds, maxToolCalls, maxTotalTokens int) *RunState {
	return &RunState{
		Version: runStateVersion, TaskID: taskID, Skill: skillName, Mode: mode, Snapshot: snapshot,
		NextRound: 1, MaxRounds: maxRounds, MaxToolCalls: maxToolCalls, MaxTotalTokens: maxTotalTokens,
		RequiredTools: map[string]bool{}, SuccessfulTools: map[string]bool{}, ToolResults: map[string]json.RawMessage{},
		Steps: []ExecutionStep{}, ResumeSafe: true, ResumeMetadata: map[string]string{},
	}
}

func (state *RunState) startStep(stepType StepType, attempt int, inputSummary string, dependsOn ...string) string {
	state.StepSequence++
	now := time.Now().UTC()
	stepID := fmt.Sprintf("step-%03d", state.StepSequence)
	state.Steps = append(state.Steps, ExecutionStep{
		StepID: stepID, Type: stepType, Status: StepRunning, Attempt: attempt,
		DependsOn: append([]string(nil), dependsOn...), InputSummary: strings.TrimSpace(inputSummary), StartedAt: &now,
	})
	return stepID
}

func (state *RunState) finishStep(stepID string, status StepStatus, outputSummary, errorType string, err error) {
	for index := range state.Steps {
		step := &state.Steps[index]
		if step.StepID != stepID {
			continue
		}
		now := time.Now().UTC()
		step.Status = status
		step.OutputSummary = strings.TrimSpace(outputSummary)
		step.ErrorType = strings.TrimSpace(errorType)
		if err != nil {
			step.Error = err.Error()
		}
		step.CompletedAt = &now
		return
	}
}

func (state *RunState) markCheckpoint(stepID string) {
	now := time.Now().UTC()
	state.CheckpointAt = &now
	for index := range state.Steps {
		if state.Steps[index].StepID == stepID {
			state.Steps[index].Checkpoint = true
			return
		}
	}
}

func (state *RunState) step(stepID string) *ExecutionStep {
	for index := range state.Steps {
		if state.Steps[index].StepID == stepID {
			return &state.Steps[index]
		}
	}
	return nil
}

func (state *RunState) syncTask(item *task.Task) {
	if state == nil || item == nil {
		return
	}
	item.ExecutionMode = string(state.Mode)
	if state.Plan != nil {
		if encodedPlan, err := json.Marshal(state.Plan); err == nil {
			item.Plan = encodedPlan
		}
	}
	encoded, err := json.Marshal(state.Steps)
	if err == nil {
		item.Steps = encoded
	}
}

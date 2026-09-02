package runtime

import (
	"context"
	"encoding/json"
)

type PlannedStep struct {
	StepID    string          `json:"step_id"`
	Type      StepType        `json:"type"`
	Tool      string          `json:"tool,omitempty"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
	DependsOn []string        `json:"depends_on,omitempty"`
}

type Plan struct {
	Summary string        `json:"summary,omitempty"`
	Steps   []PlannedStep `json:"steps"`
}

type PlanRequest struct {
	TaskID       string   `json:"task_id"`
	Skill        string   `json:"skill"`
	Input        string   `json:"input"`
	AllowedTools []string `json:"allowed_tools"`
	MaxToolCalls int      `json:"max_tool_calls"`
}

// Planner only proposes a constrained execution plan. It never receives a Tool
// registry or Permission policy and therefore cannot execute tools by itself.
type Planner interface {
	Plan(ctx context.Context, request PlanRequest) (Plan, error)
}

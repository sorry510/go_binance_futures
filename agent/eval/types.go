package eval

import (
	"encoding/json"
	"time"

	"go_binance_futures/agent/task"
)

const CaseVersion = "agent_eval_v1"

type FactRule struct {
	Path     string          `json:"path"`
	Equals   json.RawMessage `json:"equals,omitempty"`
	Contains string          `json:"contains,omitempty"`
	Exists   *bool           `json:"exists,omitempty"`
	Critical bool            `json:"critical,omitempty"`
}

type FreshnessExpectation struct {
	Required    bool     `json:"required,omitempty"`
	OutputPaths []string `json:"output_paths,omitempty"`
	Terms       []string `json:"terms,omitempty"`
}

type Expectations struct {
	Status                    task.Status          `json:"status,omitempty"`
	OutputContract            string               `json:"output_contract,omitempty"`
	RequiredFacts             []FactRule           `json:"required_facts,omitempty"`
	ForbiddenFacts            []FactRule           `json:"forbidden_facts,omitempty"`
	AllowedDirections         []string             `json:"allowed_directions,omitempty"`
	RequiredTools             []string             `json:"required_tools,omitempty"`
	ForbiddenTools            []string             `json:"forbidden_tools,omitempty"`
	RequiredEvidenceSources   []string             `json:"required_evidence_sources,omitempty"`
	MaxRepairs                int                  `json:"max_repairs,omitempty"`
	MaxTokens                 int                  `json:"max_tokens,omitempty"`
	MaxDurationMs             int64                `json:"max_duration_ms,omitempty"`
	FreshnessHonesty          FreshnessExpectation `json:"freshness_honesty,omitempty"`
	RequireMCPFailureRecovery bool                 `json:"require_mcp_failure_recovery,omitempty"`
	InstructionRules          []FactRule           `json:"instruction_rules,omitempty"`
	ExpectedSelectedSkill     string               `json:"expected_selected_skill,omitempty"`
	RequirePermissionDenial   bool                 `json:"require_permission_denial,omitempty"`
}

type Case struct {
	Version      string       `json:"version"`
	Name         string       `json:"name"`
	Skill        string       `json:"skill"`
	Fixture      string       `json:"fixture,omitempty"`
	Tags         []string     `json:"tags,omitempty"`
	Expectations Expectations `json:"expectations"`
	SourcePath   string       `json:"-"`
}

type Dimension struct {
	Name       string   `json:"name"`
	Score      float64  `json:"score"`
	MaxScore   float64  `json:"max_score"`
	Applicable bool     `json:"applicable"`
	Passed     bool     `json:"passed"`
	Critical   bool     `json:"critical,omitempty"`
	Details    []string `json:"details,omitempty"`
}

type Report struct {
	CaseName         string               `json:"case_name"`
	Skill            string               `json:"skill"`
	Score            float64              `json:"score"`
	Passed           bool                 `json:"passed"`
	CriticalFailures []string             `json:"critical_failures,omitempty"`
	Dimensions       []Dimension          `json:"dimensions"`
	Identity         task.VersionMetadata `json:"identity"`
	Duration         time.Duration        `json:"-"`
	DurationMs       int64                `json:"duration_ms"`
	Error            string               `json:"error,omitempty"`
}

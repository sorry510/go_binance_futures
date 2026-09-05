package skill

import (
	"context"

	"go_binance_futures/agent/contextengine"
	"go_binance_futures/agent/validator"
	"go_binance_futures/llm"
)

type Request struct {
	Input          string         `json:"input"`
	ConversationID string         `json:"conversation_id,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

type ToolRequirementProvider interface {
	RequiredTools(req Request) []string
}

type InputValidator interface {
	ValidateInput(req Request) error
}

type RequestValidatorProvider interface {
	ValidatorFor(req Request) validator.FinalValidator
}

type RunValidatorProvider interface {
	ValidatorForRun(req Request, toolResults map[string]any) validator.FinalValidator
}

type StructuredEvidenceValidatorProvider interface {
	ValidatorForRunWithEvidence(req Request, toolResults map[string]any, evidence map[string]contextengine.Evidence) validator.FinalValidator
}

type ModelRequirementProvider interface {
	ModelRequirements() llm.ModelRequirements
}

type ExecutionModeProvider interface {
	ExecutionMode() string
}

type ContextResourceProvider interface {
	ContextResources(req Request) []contextengine.Resource
}

type Skill interface {
	Name() string
	SystemPrompt() string
	Tools() []string
	MaxRounds() int
	BuildInput(ctx context.Context, req Request) ([]llm.Message, error)
	Validator() validator.FinalValidator
}

type Definition struct {
	SkillName              string
	Prompt                 string
	AllowedTools           []string
	Rounds                 int
	Mode                   string
	Version                VersionInfo
	BuildInputFunc         func(context.Context, Request) ([]llm.Message, error)
	FinalValidator         validator.FinalValidator
	RequiredToolsFunc      func(Request) []string
	ContextResourcesFunc   func(Request) []contextengine.Resource
	ModelRequirementsValue llm.ModelRequirements
}

func (definition Definition) Name() string         { return definition.SkillName }
func (definition Definition) SystemPrompt() string { return definition.Prompt }
func (definition Definition) Tools() []string {
	return append([]string(nil), definition.AllowedTools...)
}
func (definition Definition) MaxRounds() int        { return definition.Rounds }
func (definition Definition) ExecutionMode() string { return definition.Mode }
func (definition Definition) ModelRequirements() llm.ModelRequirements {
	return definition.ModelRequirementsValue
}
func (definition Definition) VersionInfo() VersionInfo { return definition.Version }
func (definition Definition) BuildInput(ctx context.Context, req Request) ([]llm.Message, error) {
	if definition.BuildInputFunc != nil {
		return definition.BuildInputFunc(ctx, req)
	}
	return []llm.Message{{Role: llm.RoleUser, Content: req.Input}}, nil
}

func (definition Definition) Validator() validator.FinalValidator {
	if definition.FinalValidator != nil {
		return definition.FinalValidator
	}
	return validator.Passthrough{}
}

func (definition Definition) RequiredTools(req Request) []string {
	if definition.RequiredToolsFunc == nil {
		return nil
	}
	return append([]string(nil), definition.RequiredToolsFunc(req)...)
}

func (definition Definition) ContextResources(req Request) []contextengine.Resource {
	if definition.ContextResourcesFunc == nil {
		return nil
	}
	return append([]contextengine.Resource(nil), definition.ContextResourcesFunc(req)...)
}

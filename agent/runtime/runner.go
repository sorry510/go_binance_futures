package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"go_binance_futures/agent/permission"
	"go_binance_futures/agent/skill"
	"go_binance_futures/agent/task"
	"go_binance_futures/agent/tools"
	"go_binance_futures/llm"
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
	if strings.TrimSpace(req.Skill) == "" {
		return nil, fmt.Errorf("agent skill is required")
	}
	selectedSkill, exists := runner.cfg.Skills.Get(req.Skill)
	if !exists {
		return nil, fmt.Errorf("skill %q is not registered", req.Skill)
	}
	snapshot := req.ExecutionSnapshot
	if snapshot == nil {
		frozen := FreezeExecution(selectedSkill, runner.cfg.Client)
		snapshot = &frozen
	}
	systemPrompt := snapshot.SystemPrompt
	maxRounds := selectedSkill.MaxRounds()
	if maxRounds <= 0 {
		maxRounds = runner.cfg.DefaultMaxRounds
	}
	maxToolCalls := runner.cfg.MaxToolCalls
	maxTotalTokens := runner.cfg.MaxTotalTokens
	if runner.cfg.BudgetProvider != nil {
		budget := runner.cfg.BudgetProvider(selectedSkill.Name())
		if budget.MaxToolCalls > 0 {
			maxToolCalls = budget.MaxToolCalls
		}
		if budget.MaxTotalTokens > 0 {
			maxTotalTokens = budget.MaxTotalTokens
		}
	}

	now := time.Now().UTC()
	taskID := strings.TrimSpace(req.TaskID)
	if taskID == "" {
		taskID = newTaskID()
	}
	currentTask := &task.Task{
		ID:             taskID,
		Skill:          selectedSkill.Name(),
		ConversationID: strings.TrimSpace(req.ConversationID),
		Status:         task.StatusQueued,
		Stage:          "queued",
		Progress:       0,
		Input:          req.Input,
		MaxRounds:      maxRounds,
		Provider:       string(runner.cfg.Client.Provider()),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	currentTask.ApplyVersionMetadata(snapshot.Version)
	runStarted := time.Now()
	runner.observe(Observation{
		Type: "task_started", TaskID: currentTask.ID, ConversationID: currentTask.ConversationID,
		Skill: currentTask.Skill, Provider: currentTask.Provider, Status: string(currentTask.Status),
	})
	defer func() {
		errorType := ""
		if currentTask.Status != task.StatusSucceeded {
			errorType = currentTask.Stage
		}
		runner.observe(Observation{
			Type: "task_finished", TaskID: currentTask.ID, ConversationID: currentTask.ConversationID,
			Skill: currentTask.Skill, Provider: currentTask.Provider, Model: currentTask.Model,
			Status: string(currentTask.Status), ErrorType: errorType, Error: currentTask.Error,
			Round: currentTask.Round, DurationMs: elapsedMilliseconds(runStarted), Usage: currentTask.Usage,
		})
	}()
	runner.record(currentTask, task.StatusQueued, "queued", 0, "agent task created")

	runCtx, cancel := context.WithTimeout(ctx, runner.cfg.Timeout)
	defer cancel()
	startedAt := time.Now().UTC()
	currentTask.StartedAt = &startedAt
	runner.record(currentTask, task.StatusRunning, "building_input", 3, "building skill input")

	skillRequest := skill.Request{Input: req.Input, ConversationID: req.ConversationID, Metadata: req.Metadata}
	messages, err := selectedSkill.BuildInput(runCtx, skillRequest)
	if err != nil {
		return nil, runner.fail(currentTask, "build_input_failed", err)
	}
	allowedTools := make(map[string]bool, len(selectedSkill.Tools()))
	for _, name := range selectedSkill.Tools() {
		allowedTools[strings.TrimSpace(name)] = true
	}
	requiredTools := make(map[string]bool)
	if provider, ok := selectedSkill.(skill.ToolRequirementProvider); ok {
		for _, name := range provider.RequiredTools(skillRequest) {
			name = strings.TrimSpace(name)
			if name != "" {
				requiredTools[name] = true
			}
		}
	}
	successfulTools := make(map[string]bool, len(requiredTools))
	toolResults := make(map[string]any)
	finalValidator := selectedSkill.Validator()
	if provider, ok := selectedSkill.(skill.RequestValidatorProvider); ok {
		finalValidator = provider.ValidatorFor(skillRequest)
	}
	toolCalls := 0

	for round := 1; round <= maxRounds; round++ {
		if err := runCtx.Err(); err != nil {
			return nil, runner.finishContextError(currentTask, err)
		}
		currentTask.Round = round
		if contextSize(systemPrompt, messages) > runner.cfg.MaxContextBytes {
			return nil, runner.fail(currentTask, "context_too_large", fmt.Errorf("agent context exceeds %d bytes", runner.cfg.MaxContextBytes))
		}

		progress := 5 + (round-1)*80/maxRounds
		runner.record(currentTask, task.StatusWaitingLLM, "waiting_llm", progress, fmt.Sprintf("agent round %d/%d", round, maxRounds))
		response, err := runner.generateWithRetry(runCtx, llm.Request{
			System: systemPrompt, Messages: messages,
		}, currentTask, progress)
		if err != nil {
			if runCtx.Err() != nil {
				return nil, runner.finishContextError(currentTask, runCtx.Err())
			}
			return nil, runner.fail(currentTask, "llm_failed", err)
		}
		if response == nil {
			return nil, runner.fail(currentTask, "llm_failed", fmt.Errorf("LLM returned an empty response"))
		}
		mergeUsage(&currentTask.Usage, response.Usage)
		if maxTotalTokens > 0 && currentTask.Usage.TotalTokens > maxTotalTokens {
			return nil, runner.fail(currentTask, "token_budget_exceeded", fmt.Errorf("agent exceeded %d total tokens", maxTotalTokens))
		}
		if response.Model != "" {
			currentTask.Model = response.Model
		}
		runner.appendRuntimeMessage(currentTask.ID, &messages, llm.Message{Role: llm.RoleAssistant, Content: response.Content})
		if isTruncatedFinishReason(response.FinishReason) {
			runner.observeRepair(currentTask, "truncated_response")
			feedback := repairFeedback("truncated_response", "LLM response was truncated by the output token limit")
			runner.appendRuntimeMessage(currentTask.ID, &messages, feedback)
			runner.record(currentTask, task.StatusRunning, "repairing_response", progress+1, "LLM response was truncated")
			continue
		}

		decision, err := parseDecision(response.Content)
		if err != nil {
			runner.observeRepair(currentTask, "decision_protocol")
			feedback := repairFeedback("decision_protocol", err.Error())
			runner.appendRuntimeMessage(currentTask.ID, &messages, feedback)
			runner.record(currentTask, task.StatusRunning, "repairing_decision", progress+1, err.Error())
			continue
		}

		switch decision.Action {
		case "error":
			return nil, runner.fail(currentTask, "agent_error", fmt.Errorf("agent error: %s", decision.Error))
		case "tool":
			toolCalls++
			if toolCalls > maxToolCalls {
				return nil, runner.fail(currentTask, "tool_limit_exceeded", fmt.Errorf("agent exceeded %d tool calls", maxToolCalls))
			}
			if !allowedTools[decision.Tool] {
				return nil, runner.fail(currentTask, "tool_not_allowed", fmt.Errorf("skill %q does not allow tool %q", selectedSkill.Name(), decision.Tool))
			}
			selectedTool, exists := runner.cfg.Tools.Get(decision.Tool)
			if !exists {
				return nil, runner.fail(currentTask, "tool_not_registered", fmt.Errorf("tool %q is not registered", decision.Tool))
			}
			if err := runner.cfg.Policy.Allow(selectedSkill.Name(), selectedTool.Name(), selectedTool.Risk()); err != nil {
				return nil, runner.fail(currentTask, "tool_permission_denied", err)
			}
			arguments := decision.Arguments
			if len(arguments) == 0 || string(arguments) == "null" {
				arguments = json.RawMessage(`{}`)
			}
			runner.recordToolWaiting(currentTask, "waiting_tool", progress+2, selectedTool.Name(), "calling "+selectedTool.Name())
			toolCtx := runCtx
			cancelTool := func() {}
			metadata := selectedTool.Metadata()
			if metadata.Timeout > 0 {
				toolCtx, cancelTool = context.WithTimeout(runCtx, metadata.Timeout)
			}
			toolStarted := time.Now()
			value, toolErr := selectedTool.Execute(toolCtx, arguments)
			cancelTool()
			duration := time.Since(toolStarted)
			toolStatus, toolErrorType, toolError := "success", "", ""
			if toolErr != nil {
				toolStatus = "error"
				toolErrorType = classifyError(toolErr)
				toolError = toolErr.Error()
			}
			runner.observe(Observation{
				Type: "tool_call", TaskID: currentTask.ID, ConversationID: currentTask.ConversationID,
				Skill: currentTask.Skill, Provider: currentTask.Provider, Model: currentTask.Model, Tool: selectedTool.Name(),
				Status: toolStatus, ErrorType: toolErrorType, Error: toolError, Round: currentTask.Round, DurationMs: duration.Milliseconds(),
			})
			maxResultBytes := runner.cfg.MaxToolResultBytes
			if metadata.MaxResultBytes > 0 && (maxResultBytes <= 0 || metadata.MaxResultBytes < maxResultBytes) {
				maxResultBytes = metadata.MaxResultBytes
			}
			toolMessage, err := buildToolResultMessage(selectedTool.Name(), value, toolErr, maxResultBytes)
			if err != nil {
				return nil, runner.fail(currentTask, "tool_result_failed", err)
			}
			runner.appendRuntimeMessage(currentTask.ID, &messages, toolMessage)
			if toolErr == nil {
				successfulTools[selectedTool.Name()] = true
				toolResults[selectedTool.Name()] = value
			}
			outcome := "success"
			if toolErr != nil {
				outcome = "error"
			}
			runner.recordTool(currentTask, "tool_result", progress+3, selectedTool.Name(), outcome, duration, "tool result received")

		case "final":
			missingTools := make([]string, 0, len(requiredTools))
			for name := range requiredTools {
				if !successfulTools[name] {
					missingTools = append(missingTools, name)
				}
			}
			if len(missingTools) > 0 {
				runner.observeRepair(currentTask, "required_tools")
				sort.Strings(missingTools)
				err := fmt.Errorf("required tools must succeed before final: %s", strings.Join(missingTools, ", "))
				feedback := repairFeedback("required_tools", err.Error())
				runner.appendRuntimeMessage(currentTask.ID, &messages, feedback)
				runner.record(currentTask, task.StatusRunning, "repairing_required_tools", min(progress+4, 95), err.Error())
				continue
			}
			runner.record(currentTask, task.StatusValidating, "validating", min(progress+4, 95), "validating final result")
			validatorForFinal := finalValidator
			if provider, ok := selectedSkill.(skill.RunValidatorProvider); ok {
				validatorForFinal = provider.ValidatorForRun(skillRequest, toolResults)
			}
			validationStarted := time.Now()
			value, err := validatorForFinal.Validate(runCtx, decision.Result)
			validationStatus, validationError := "success", ""
			if err != nil {
				validationStatus = "error"
				validationError = err.Error()
			}
			runner.observe(Observation{
				Type: "validation", TaskID: currentTask.ID, ConversationID: currentTask.ConversationID,
				Skill: currentTask.Skill, Provider: currentTask.Provider, Model: currentTask.Model,
				Status: validationStatus, ErrorType: map[bool]string{true: "validation_error", false: ""}[err != nil],
				Error: validationError, Round: currentTask.Round, DurationMs: elapsedMilliseconds(validationStarted),
			})
			if runner.cfg.ValidationHook != nil {
				runner.cfg.ValidationHook(currentTask.ID, append(json.RawMessage(nil), decision.Result...), err)
			}
			if err != nil {
				runner.observeRepair(currentTask, "final_validation")
				feedback := repairFeedback("final_validation", err.Error())
				runner.appendRuntimeMessage(currentTask.ID, &messages, feedback)
				runner.record(currentTask, task.StatusRunning, "repairing_final", min(progress+5, 96), err.Error())
				continue
			}
			currentTask.Result = append([]byte(nil), decision.Result...)
			runner.succeed(currentTask, "completed", "agent completed")
			return &Result{
				TaskID: currentTask.ID, Skill: currentTask.Skill, Summary: decision.Summary,
				Raw: append([]byte(nil), decision.Result...), Value: value, Usage: currentTask.Usage,
			}, nil
		}
	}

	return nil, runner.fail(currentTask, "max_rounds", fmt.Errorf("agent reached maximum %d rounds", maxRounds))
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}

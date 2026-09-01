package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"go_binance_futures/agent/skill"
	"go_binance_futures/agent/task"
	"go_binance_futures/llm"
)

type reactExecutor struct {
	runner *DefaultRunner
}

func (executor *reactExecutor) execute(ctx context.Context, session *runSession) (*Result, error) {
	state := session.state
	item := session.currentTask
	messages := state.Messages
	startRound := state.NextRound
	if startRound < 1 {
		startRound = 1
	}
	for round := startRound; round <= state.MaxRounds; round++ {
		if err := ctx.Err(); err != nil {
			return nil, executor.runner.finishContextError(item, err)
		}
		state.Round = round
		state.NextRound = round
		item.Round = round
		if contextSize(state.Snapshot.SystemPrompt, messages) > executor.runner.cfg.MaxContextBytes {
			return nil, executor.runner.fail(item, "context_too_large", fmt.Errorf("agent context exceeds %d bytes", executor.runner.cfg.MaxContextBytes))
		}

		progress := 5 + (round-1)*80/state.MaxRounds
		llmStep := state.startStep(StepLLM, 1, fmt.Sprintf("agent round %d/%d", round, state.MaxRounds))
		executor.runner.recordStep(item, state, llmStep, task.StatusWaitingLLM, "waiting_llm", progress, fmt.Sprintf("agent round %d/%d", round, state.MaxRounds), "", false)
		response, err := executor.runner.generateWithRetry(ctx, llm.Request{System: state.Snapshot.SystemPrompt, Messages: messages}, item, state, llmStep, progress)
		if err != nil {
			state.finishStep(llmStep, StepFailed, "", classifyError(err), err)
			state.syncTask(item)
			if ctx.Err() != nil {
				return nil, executor.runner.finishContextError(item, ctx.Err())
			}
			return nil, executor.runner.fail(item, "llm_failed", err)
		}
		if response == nil {
			err := fmt.Errorf("LLM returned an empty response")
			state.finishStep(llmStep, StepFailed, "", "llm_empty_response", err)
			state.syncTask(item)
			return nil, executor.runner.fail(item, "llm_failed", err)
		}
		mergeUsage(&item.Usage, response.Usage)
		if state.MaxTotalTokens > 0 && item.Usage.TotalTokens > state.MaxTotalTokens {
			err := fmt.Errorf("agent exceeded %d total tokens", state.MaxTotalTokens)
			state.finishStep(llmStep, StepFailed, "", "token_budget_exceeded", err)
			state.syncTask(item)
			return nil, executor.runner.fail(item, "token_budget_exceeded", err)
		}
		if response.Model != "" {
			item.Model = response.Model
		}
		executor.runner.appendRuntimeMessage(item.ID, &messages, llm.Message{Role: llm.RoleAssistant, Content: response.Content})
		state.Messages = messages
		state.finishStep(llmStep, StepSucceeded, "LLM response received", "", nil)
		state.syncTask(item)

		if isTruncatedFinishReason(response.FinishReason) {
			executor.runner.observeRepair(item, "truncated_response")
			feedback := repairFeedback("truncated_response", "LLM response was truncated by the output token limit")
			executor.runner.appendRuntimeMessage(item.ID, &messages, feedback)
			state.Messages = messages
			executor.runner.recordStep(item, state, llmStep, task.StatusRunning, "repairing_response", progress+1, "LLM response was truncated", "truncated_response", false)
			if err := executor.runner.saveCheckpoint(ctx, item, state, llmStep, round+1); err != nil {
				return nil, executor.runner.fail(item, "checkpoint_failed", err)
			}
			continue
		}

		decision, err := parseDecision(response.Content)
		if err != nil {
			executor.runner.observeRepair(item, "decision_protocol")
			feedback := repairFeedback("decision_protocol", err.Error())
			executor.runner.appendRuntimeMessage(item.ID, &messages, feedback)
			state.Messages = messages
			executor.runner.recordStep(item, state, llmStep, task.StatusRunning, "repairing_decision", progress+1, err.Error(), "decision_protocol", false)
			if checkpointErr := executor.runner.saveCheckpoint(ctx, item, state, llmStep, round+1); checkpointErr != nil {
				return nil, executor.runner.fail(item, "checkpoint_failed", checkpointErr)
			}
			continue
		}

		switch decision.Action {
		case "error":
			return nil, executor.runner.fail(item, "agent_error", fmt.Errorf("agent error: %s", decision.Error))
		case "tool":
			if err := executor.executeTool(ctx, session, &messages, decision, progress, round, llmStep, nil); err != nil {
				return nil, err
			}
		case "final":
			result, done, err := executor.executeFinal(ctx, session, &messages, decision, progress, round, llmStep)
			if err != nil {
				return nil, err
			}
			if done {
				return result, nil
			}
		}
		state.Messages = messages
	}
	return nil, executor.runner.fail(item, "max_rounds", fmt.Errorf("agent reached maximum %d rounds", state.MaxRounds))
}

func (executor *reactExecutor) executeTool(ctx context.Context, session *runSession, messages *[]llm.Message, decision decision, progress, round int, dependsOn string, beforeCheckpoint func()) error {
	state, item := session.state, session.currentTask
	state.ToolCalls++
	if state.ToolCalls > state.MaxToolCalls {
		return executor.runner.fail(item, "tool_limit_exceeded", fmt.Errorf("agent exceeded %d tool calls", state.MaxToolCalls))
	}
	if !session.allowedTools[decision.Tool] {
		return executor.runner.fail(item, "tool_not_allowed", fmt.Errorf("skill %q does not allow tool %q", session.selectedSkill.Name(), decision.Tool))
	}
	selectedTool, exists := executor.runner.cfg.Tools.Get(decision.Tool)
	if !exists {
		return executor.runner.fail(item, "tool_not_registered", fmt.Errorf("tool %q is not registered", decision.Tool))
	}
	if err := executor.runner.cfg.Policy.Allow(session.selectedSkill.Name(), selectedTool.Name(), selectedTool.Risk()); err != nil {
		return executor.runner.fail(item, "tool_permission_denied", err)
	}
	arguments := decision.Arguments
	if len(arguments) == 0 || string(arguments) == "null" {
		arguments = json.RawMessage(`{}`)
	}
	stepID := state.startStep(StepTool, 1, selectedTool.Name(), dependsOn)
	executor.runner.recordToolStep(item, state, stepID, task.StatusWaitingTool, "waiting_tool", progress+2, selectedTool.Name(), "running", "", false, 0, "calling "+selectedTool.Name())

	safeCheckpoint := toolCreatesSafeCheckpoint(selectedTool)
	if !safeCheckpoint {
		if err := executor.runner.clearCheckpoint(ctx, item, state); err != nil {
			state.finishStep(stepID, StepFailed, "", "checkpoint_clear_failed", err)
			return executor.runner.fail(item, "checkpoint_failed", err)
		}
	}
	toolCtx := ctx
	cancelTool := func() {}
	metadata := selectedTool.Metadata()
	if metadata.Timeout > 0 {
		toolCtx, cancelTool = context.WithTimeout(ctx, metadata.Timeout)
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
	executor.runner.observe(Observation{
		Type: "tool_call", TaskID: item.ID, ConversationID: item.ConversationID,
		Skill: item.Skill, Provider: item.Provider, Model: item.Model, Tool: selectedTool.Name(),
		Status: toolStatus, ErrorType: toolErrorType, Error: toolError, Round: item.Round, DurationMs: duration.Milliseconds(),
	})
	maxResultBytes := executor.runner.cfg.MaxToolResultBytes
	if metadata.MaxResultBytes > 0 && (maxResultBytes <= 0 || metadata.MaxResultBytes < maxResultBytes) {
		maxResultBytes = metadata.MaxResultBytes
	}
	toolMessage, err := buildToolResultMessage(selectedTool.Name(), value, toolErr, maxResultBytes)
	if err != nil {
		state.finishStep(stepID, StepFailed, "", "tool_result_failed", err)
		state.syncTask(item)
		return executor.runner.fail(item, "tool_result_failed", err)
	}
	executor.runner.appendRuntimeMessage(item.ID, messages, toolMessage)
	state.Messages = *messages
	if toolErr == nil {
		state.SuccessfulTools[selectedTool.Name()] = true
		session.toolResults[selectedTool.Name()] = value
		if safeCheckpoint {
			raw, marshalErr := json.Marshal(value)
			if marshalErr != nil {
				state.finishStep(stepID, StepFailed, "", "checkpoint_encode_failed", marshalErr)
				state.syncTask(item)
				return executor.runner.fail(item, "checkpoint_failed", marshalErr)
			}
			state.ToolResults[selectedTool.Name()] = raw
		}
	}
	stepStatus := StepSucceeded
	if toolErr != nil {
		stepStatus = StepFailed
	}
	state.finishStep(stepID, stepStatus, "tool result received", toolErrorType, toolErr)
	executor.runner.recordToolStep(item, state, stepID, task.StatusRunning, "tool_result", progress+3, selectedTool.Name(), toolStatus, toolErrorType, false, duration, "tool result received")
	if safeCheckpoint {
		if beforeCheckpoint != nil {
			beforeCheckpoint()
		}
		if err := executor.runner.saveCheckpoint(ctx, item, state, stepID, round+1); err != nil {
			return executor.runner.fail(item, "checkpoint_failed", err)
		}
		executor.runner.recordToolStep(item, state, stepID, task.StatusRunning, "tool_checkpoint", progress+3, selectedTool.Name(), toolStatus, toolErrorType, true, duration, "safe tool checkpoint saved")
	}
	return nil
}

func (executor *reactExecutor) executeFinal(ctx context.Context, session *runSession, messages *[]llm.Message, decision decision, progress, round int, dependsOn string) (*Result, bool, error) {
	state, item := session.state, session.currentTask
	missingTools := make([]string, 0, len(state.RequiredTools))
	for name := range state.RequiredTools {
		if !state.SuccessfulTools[name] {
			missingTools = append(missingTools, name)
		}
	}
	if len(missingTools) > 0 {
		executor.runner.observeRepair(item, "required_tools")
		sort.Strings(missingTools)
		err := fmt.Errorf("required tools must succeed before final: %s", strings.Join(missingTools, ", "))
		feedback := repairFeedback("required_tools", err.Error())
		executor.runner.appendRuntimeMessage(item.ID, messages, feedback)
		state.Messages = *messages
		executor.runner.recordStep(item, state, dependsOn, task.StatusRunning, "repairing_required_tools", min(progress+4, 95), err.Error(), "required_tools", false)
		if checkpointErr := executor.runner.saveCheckpoint(ctx, item, state, dependsOn, round+1); checkpointErr != nil {
			return nil, false, executor.runner.fail(item, "checkpoint_failed", checkpointErr)
		}
		return nil, false, nil
	}

	stepID := state.startStep(StepValidate, 1, "validate final result", dependsOn)
	executor.runner.recordStep(item, state, stepID, task.StatusValidating, "validating", min(progress+4, 95), "validating final result", "", false)
	validatorForFinal := session.finalValidator
	if provider, ok := session.selectedSkill.(skill.RunValidatorProvider); ok {
		validatorForFinal = provider.ValidatorForRun(session.skillRequest, session.toolResults)
	}
	validationStarted := time.Now()
	value, err := validatorForFinal.Validate(ctx, decision.Result)
	validationStatus, validationError := "success", ""
	if err != nil {
		validationStatus = "error"
		validationError = err.Error()
	}
	executor.runner.observe(Observation{
		Type: "validation", TaskID: item.ID, ConversationID: item.ConversationID,
		Skill: item.Skill, Provider: item.Provider, Model: item.Model,
		Status: validationStatus, ErrorType: map[bool]string{true: "validation_error", false: ""}[err != nil],
		Error: validationError, Round: item.Round, DurationMs: elapsedMilliseconds(validationStarted),
	})
	if executor.runner.cfg.ValidationHook != nil {
		executor.runner.cfg.ValidationHook(item.ID, append(json.RawMessage(nil), decision.Result...), err)
	}
	if err != nil {
		state.finishStep(stepID, StepFailed, "validation failed", "validation_error", err)
		executor.runner.observeRepair(item, "final_validation")
		feedback := repairFeedback("final_validation", err.Error())
		executor.runner.appendRuntimeMessage(item.ID, messages, feedback)
		state.Messages = *messages
		executor.runner.recordStep(item, state, stepID, task.StatusRunning, "repairing_final", min(progress+5, 96), err.Error(), "validation_error", false)
		if checkpointErr := executor.runner.saveCheckpoint(ctx, item, state, stepID, round+1); checkpointErr != nil {
			return nil, false, executor.runner.fail(item, "checkpoint_failed", checkpointErr)
		}
		return nil, false, nil
	}
	state.finishStep(stepID, StepSucceeded, "final result valid", "", nil)

	finalizeStep := state.startStep(StepFinalize, 1, "finalize task", stepID)
	item.Result = append([]byte(nil), decision.Result...)
	state.finishStep(finalizeStep, StepSucceeded, "agent completed", "", nil)
	state.syncTask(item)
	executor.runner.succeed(item, "completed", "agent completed")
	if checkpointStore, ok := executor.runner.cfg.Tasks.(task.CheckpointStore); ok {
		_ = checkpointStore.ClearCheckpoint(context.Background(), item.ID)
		item.CheckpointJSON = ""
	}
	return &Result{
		TaskID: item.ID, Skill: item.Skill, Summary: decision.Summary,
		Raw: append([]byte(nil), decision.Result...), Value: value, Usage: item.Usage,
	}, true, nil
}

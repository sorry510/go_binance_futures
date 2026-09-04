package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"go_binance_futures/agent/contextengine"
	"go_binance_futures/agent/permission"
	"go_binance_futures/agent/skill"
	"go_binance_futures/agent/task"
	"go_binance_futures/agent/toolruntime"
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
		progress := 5 + (round-1)*80/state.MaxRounds
		llmStep := state.startStep(StepLLM, 1, fmt.Sprintf("agent round %d/%d", round, state.MaxRounds))
		requestMessages, contextTrace, contextErr := executor.runner.cfg.ContextEngine.Build(contextengine.BuildOptions{
			SystemPrompt: state.Snapshot.SystemPrompt, Blocks: state.ContextBlocks,
			MaxTokens: executor.runner.cfg.MaxContextTokens, MaxBytes: executor.runner.cfg.MaxContextBytes, Now: time.Now().UTC(),
		})
		state.setContextTrace(llmStep, contextTrace)
		state.syncTask(item)
		if contextErr != nil {
			state.finishStep(llmStep, StepFailed, "context build failed", "context_too_large", contextErr)
			state.syncTask(item)
			return nil, executor.runner.fail(item, "context_too_large", contextErr)
		}
		if contextTrace.TrimmedBlocks > 0 {
			executor.runner.recordStep(item, state, llmStep, task.StatusRunning, "context_trimmed", progress,
				fmt.Sprintf("context trimmed %d/%d blocks to %d estimated tokens", contextTrace.TrimmedBlocks, contextTrace.InputBlocks, contextTrace.SelectedTokens), "", false)
		}
		executor.runner.recordStep(item, state, llmStep, task.StatusWaitingLLM, "waiting_llm", progress, fmt.Sprintf("agent round %d/%d", round, state.MaxRounds), "", false)
		response, err := executor.runner.generateWithRetry(ctx, llm.Request{System: state.Snapshot.SystemPrompt, Messages: requestMessages}, item, state, llmStep, progress)
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
		executor.runner.appendRuntimeMessage(item.ID, state, &messages, llm.Message{Role: llm.RoleAssistant, Content: response.Content})
		state.Messages = messages
		state.finishStep(llmStep, StepSucceeded, "LLM response received", "", nil)
		state.syncTask(item)

		if isTruncatedFinishReason(response.FinishReason) {
			executor.runner.observeRepair(item, "truncated_response")
			feedback := repairFeedback("truncated_response", "LLM response was truncated by the output token limit")
			executor.runner.appendRuntimeMessage(item.ID, state, &messages, feedback)
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
			executor.runner.appendRuntimeMessage(item.ID, state, &messages, feedback)
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
		case "parallel_tools":
			if err := executor.executeParallelTools(ctx, session, &messages, decision, progress, round, llmStep); err != nil {
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
	request := toolruntime.ExecuteRequest{
		SkillName: session.selectedSkill.Name(), AllowedTools: session.allowedTools,
		ToolName: decision.Tool, Arguments: decision.Arguments,
		MaxResultBytes: executor.runner.cfg.MaxToolResultBytes, CallIndex: state.ToolCalls, CallBudget: state.MaxToolCalls,
	}
	descriptor, checkErr := executor.runner.cfg.ToolRuntime.Check(request)
	if checkErr != nil {
		if toolruntime.TypeOf(checkErr) == toolruntime.ErrorNotFound {
			return executor.repairToolName(ctx, session, messages, dependsOn, progress, round, checkErr)
		}
		stage := "tool_runtime_failed"
		if toolruntime.TypeOf(checkErr) == toolruntime.ErrorPermission {
			if strings.Contains(checkErr.Error(), "does not allow tool") {
				stage = "tool_not_allowed"
			} else {
				stage = "tool_permission_denied"
			}
		}
		return executor.runner.fail(item, stage, checkErr)
	}
	state.ToolCalls++
	if state.ToolCalls > state.MaxToolCalls {
		return executor.runner.fail(item, "tool_limit_exceeded", fmt.Errorf("agent exceeded %d tool calls", state.MaxToolCalls))
	}
	request.ToolName = descriptor.CanonicalName
	request.CallIndex = state.ToolCalls
	stepID := state.startStep(StepTool, 1, descriptor.CanonicalName, dependsOn)
	executor.runner.recordToolStep(item, state, stepID, task.StatusWaitingTool, "waiting_tool", progress+2, descriptor.CanonicalName, "running", "", false, 0, "calling "+descriptor.CanonicalName)

	safeCheckpoint := descriptor.Idempotent && descriptor.Risk == permission.RiskRead
	if !safeCheckpoint {
		if err := executor.runner.clearCheckpoint(ctx, item, state); err != nil {
			state.finishStep(stepID, StepFailed, "", "checkpoint_clear_failed", err)
			return executor.runner.fail(item, "checkpoint_failed", err)
		}
	}
	toolResult, runtimeErr := executor.runner.cfg.ToolRuntime.Execute(ctx, request)
	if runtimeErr != nil {
		state.finishStep(stepID, StepFailed, "", string(toolruntime.TypeOf(runtimeErr)), runtimeErr)
		state.syncTask(item)
		return executor.runner.fail(item, "tool_result_failed", runtimeErr)
	}
	if ctx.Err() != nil {
		state.finishStep(stepID, StepFailed, "", classifyError(ctx.Err()), ctx.Err())
		state.syncTask(item)
		return executor.runner.finishContextError(item, ctx.Err())
	}
	state.setToolTrace(stepID, toolResult.Trace)
	if len(toolResult.Evidence) > 0 {
		state.addEvidence(stepID, toolResult.Evidence)
	}
	toolStatus := "success"
	toolError := ""
	if toolResult.ToolError != nil {
		toolStatus, toolError = "error", toolResult.ToolError.Error()
	}
	executor.runner.observe(Observation{
		Type: "tool_call", TaskID: item.ID, ConversationID: item.ConversationID,
		Skill: item.Skill, Provider: item.Provider, Model: item.Model, Tool: descriptor.CanonicalName,
		Status: toolStatus, ErrorType: string(toolResult.Envelope.ErrorType), Error: toolError, Round: item.Round,
		DurationMs: toolResult.Envelope.DurationMs, CacheHit: toolResult.Envelope.CacheHit, Partial: toolResult.Envelope.Partial,
		RawSize: toolResult.Envelope.RawSize, ContentHash: toolResult.Envelope.ContentHash,
	})
	toolMessage, err := buildToolResultMessage(toolResult.Envelope, toolResult.Evidence, toolResult.ToolError)
	if err != nil {
		state.finishStep(stepID, StepFailed, "", "tool_result_failed", err)
		state.syncTask(item)
		return executor.runner.fail(item, "tool_result_failed", err)
	}
	if toolResult.ToolError == nil {
		block := toolResult.ContextBlock
		block.ID = "tool-" + stepID
		executor.runner.appendRuntimeMessage(item.ID, state, messages, toolMessage, block)
		state.SuccessfulTools[descriptor.CanonicalName] = true
		session.toolResults[descriptor.CanonicalName] = toolResult.Value
		if safeCheckpoint && len(toolResult.Raw) > 0 {
			state.ToolResults[descriptor.CanonicalName] = append(json.RawMessage(nil), toolResult.Raw...)
		}
	} else {
		executor.runner.appendRuntimeMessage(item.ID, state, messages, toolMessage)
	}
	state.Messages = *messages
	stepStatus := StepSucceeded
	if toolResult.ToolError != nil {
		stepStatus = StepFailed
	}
	outputSummary := "tool result received"
	if toolResult.Envelope.CacheHit {
		outputSummary = "tool result received from cache"
	}
	if toolResult.Envelope.Partial {
		outputSummary += " (partial)"
	}
	state.finishStep(stepID, stepStatus, outputSummary, string(toolResult.Envelope.ErrorType), toolResult.ToolError)
	duration := time.Duration(toolResult.Envelope.DurationMs) * time.Millisecond
	executor.runner.recordToolStep(item, state, stepID, task.StatusRunning, "tool_result", progress+3, descriptor.CanonicalName, toolStatus, string(toolResult.Envelope.ErrorType), false, duration, outputSummary)
	if safeCheckpoint {
		if beforeCheckpoint != nil {
			beforeCheckpoint()
		}
		if err := executor.runner.saveCheckpoint(ctx, item, state, stepID, round+1); err != nil {
			return executor.runner.fail(item, "checkpoint_failed", err)
		}
		executor.runner.recordToolStep(item, state, stepID, task.StatusRunning, "tool_checkpoint", progress+3, descriptor.CanonicalName, toolStatus, string(toolResult.Envelope.ErrorType), true, duration, "safe tool checkpoint saved")
	}
	return nil
}

func (executor *reactExecutor) repairToolName(ctx context.Context, session *runSession, messages *[]llm.Message, stepID string, progress, round int, cause error) error {
	state, item := session.state, session.currentTask
	names := make([]string, 0, len(session.allowedTools))
	for name, allowed := range session.allowedTools {
		if allowed {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	message := cause.Error()
	if len(names) > 0 {
		message += ". Use one exact allowed tool name without translating, shortening, or changing punctuation: " + strings.Join(names, ", ")
	}
	feedback := repairFeedback("tool_name", message)
	executor.runner.appendRuntimeMessage(item.ID, state, messages, feedback)
	state.Messages = *messages
	executor.runner.recordStep(item, state, stepID, task.StatusRunning, "repairing_tool_name", progress+1, message, "tool_not_registered", false)
	if err := executor.runner.saveCheckpoint(ctx, item, state, stepID, round+1); err != nil {
		return executor.runner.fail(item, "checkpoint_failed", err)
	}
	return nil
}

func (executor *reactExecutor) executeParallelTools(ctx context.Context, session *runSession, messages *[]llm.Message, decision decision, progress, round int, dependsOn string) error {
	state, item := session.state, session.currentTask
	count := len(decision.Tools)
	if count < 2 {
		return executor.runner.fail(item, "invalid_parallel_tools", fmt.Errorf("parallel tool batch requires at least two calls"))
	}
	if state.ToolCalls+count > state.MaxToolCalls {
		return executor.runner.fail(item, "tool_limit_exceeded", fmt.Errorf("agent parallel batch requires %d tool calls but only %d remain", count, state.MaxToolCalls-state.ToolCalls))
	}
	parentStepID := state.startStep(StepParallelTools, 1, fmt.Sprintf("%d independent read tools", count), dependsOn)
	executor.runner.recordStep(item, state, parentStepID, task.StatusWaitingTool, "waiting_tool", progress+2, fmt.Sprintf("calling %d tools in parallel", count), "", false)

	baseCallIndex := state.ToolCalls
	batch := make([]toolruntime.BatchRequest, count)
	childSteps := make([]string, count)
	descriptors := make([]toolruntime.ToolDescriptor, count)
	seen := map[string]bool{}
	for index, call := range decision.Tools {
		request := toolruntime.ExecuteRequest{
			SkillName: session.selectedSkill.Name(), AllowedTools: session.allowedTools,
			ToolName: call.Tool, Arguments: call.Arguments,
			MaxResultBytes: executor.runner.cfg.MaxToolResultBytes,
			CallIndex:      baseCallIndex + index + 1, CallBudget: state.MaxToolCalls,
		}
		descriptor, checkErr := executor.runner.cfg.ToolRuntime.Check(request)
		if checkErr != nil {
			state.finishStep(parentStepID, StepFailed, "parallel tool preflight failed", string(toolruntime.TypeOf(checkErr)), checkErr)
			if toolruntime.TypeOf(checkErr) == toolruntime.ErrorNotFound {
				return executor.repairToolName(ctx, session, messages, parentStepID, progress, round, checkErr)
			}
			stage := "tool_runtime_failed"
			if toolruntime.TypeOf(checkErr) == toolruntime.ErrorPermission {
				if strings.Contains(checkErr.Error(), "does not allow tool") {
					stage = "tool_not_allowed"
				} else {
					stage = "tool_permission_denied"
				}
			}
			state.syncTask(item)
			return executor.runner.fail(item, stage, checkErr)
		}
		request.ToolName = descriptor.CanonicalName
		if descriptor.Risk != permission.RiskRead || !descriptor.Idempotent {
			err := fmt.Errorf("parallel tool %q must be read and idempotent", descriptor.CanonicalName)
			state.finishStep(parentStepID, StepFailed, "parallel tool safety check failed", "parallel_tool_not_safe", err)
			state.syncTask(item)
			return executor.runner.fail(item, "parallel_tool_not_safe", err)
		}
		if seen[descriptor.CanonicalName] {
			err := fmt.Errorf("parallel tool batch contains duplicate tool %q", descriptor.CanonicalName)
			state.finishStep(parentStepID, StepFailed, "parallel tool safety check failed", "invalid_parallel_tools", err)
			state.syncTask(item)
			return executor.runner.fail(item, "invalid_parallel_tools", err)
		}
		seen[descriptor.CanonicalName] = true
		descriptors[index] = descriptor
		childSteps[index] = state.startStep(StepTool, 1, descriptor.CanonicalName, dependsOn)
		executor.runner.recordToolStep(item, state, childSteps[index], task.StatusWaitingTool, "waiting_tool", progress+2, descriptor.CanonicalName, "running", "", false, 0, "queued in parallel tool batch")
		batch[index] = toolruntime.BatchRequest{Request: request}
	}
	state.ToolCalls += count
	results := executor.runner.cfg.ToolRuntime.ExecuteBatch(ctx, batch, executor.runner.cfg.MaxParallelToolCalls)
	if ctx.Err() != nil {
		state.finishStep(parentStepID, StepFailed, "parallel tool batch cancelled", classifyError(ctx.Err()), ctx.Err())
		state.syncTask(item)
		return executor.runner.finishContextError(item, ctx.Err())
	}

	for index, batchResult := range results {
		stepID := childSteps[index]
		descriptor := descriptors[index]
		if batchResult.Err != nil {
			state.finishStep(stepID, StepFailed, "tool runtime failed", string(toolruntime.TypeOf(batchResult.Err)), batchResult.Err)
			state.finishStep(parentStepID, StepFailed, "parallel tool runtime failed", string(toolruntime.TypeOf(batchResult.Err)), batchResult.Err)
			state.syncTask(item)
			return executor.runner.fail(item, "tool_result_failed", batchResult.Err)
		}
		toolResult := batchResult.Result
		state.setToolTrace(stepID, toolResult.Trace)
		if len(toolResult.Evidence) > 0 {
			state.addEvidence(stepID, toolResult.Evidence)
		}
		toolStatus, toolError := "success", ""
		if toolResult.ToolError != nil {
			toolStatus, toolError = "error", toolResult.ToolError.Error()
		}
		executor.runner.observe(Observation{
			Type: "tool_call", TaskID: item.ID, ConversationID: item.ConversationID,
			Skill: item.Skill, Provider: item.Provider, Model: item.Model, Tool: descriptor.CanonicalName,
			Status: toolStatus, ErrorType: string(toolResult.Envelope.ErrorType), Error: toolError, Round: item.Round,
			DurationMs: toolResult.Envelope.DurationMs, CacheHit: toolResult.Envelope.CacheHit, Partial: toolResult.Envelope.Partial,
			RawSize: toolResult.Envelope.RawSize, ContentHash: toolResult.Envelope.ContentHash,
		})
		toolMessage, err := buildToolResultMessage(toolResult.Envelope, toolResult.Evidence, toolResult.ToolError)
		if err != nil {
			state.finishStep(stepID, StepFailed, "", "tool_result_failed", err)
			state.finishStep(parentStepID, StepFailed, "parallel tool result encoding failed", "tool_result_failed", err)
			state.syncTask(item)
			return executor.runner.fail(item, "tool_result_failed", err)
		}
		if toolResult.ToolError == nil {
			block := toolResult.ContextBlock
			block.ID = "tool-" + stepID
			executor.runner.appendRuntimeMessage(item.ID, state, messages, toolMessage, block)
			state.SuccessfulTools[descriptor.CanonicalName] = true
			session.toolResults[descriptor.CanonicalName] = toolResult.Value
			if len(toolResult.Raw) > 0 {
				state.ToolResults[descriptor.CanonicalName] = append(json.RawMessage(nil), toolResult.Raw...)
			}
		} else {
			executor.runner.appendRuntimeMessage(item.ID, state, messages, toolMessage)
		}
		stepStatus := StepSucceeded
		if toolResult.ToolError != nil {
			stepStatus = StepFailed
		}
		outputSummary := "parallel tool result received"
		if toolResult.Envelope.CacheHit {
			outputSummary = "parallel tool result received from cache"
		}
		if toolResult.Envelope.Partial {
			outputSummary += " (partial)"
		}
		state.finishStep(stepID, stepStatus, outputSummary, string(toolResult.Envelope.ErrorType), toolResult.ToolError)
		duration := time.Duration(toolResult.Envelope.DurationMs) * time.Millisecond
		executor.runner.recordToolStep(item, state, stepID, task.StatusRunning, "tool_result", progress+3, descriptor.CanonicalName, toolStatus, string(toolResult.Envelope.ErrorType), false, duration, outputSummary)
	}
	state.Messages = *messages
	state.finishStep(parentStepID, StepSucceeded, fmt.Sprintf("parallel batch completed with %d tool results", count), "", nil)
	if err := executor.runner.saveCheckpoint(ctx, item, state, parentStepID, round+1); err != nil {
		return executor.runner.fail(item, "checkpoint_failed", err)
	}
	executor.runner.recordStep(item, state, parentStepID, task.StatusRunning, "tool_checkpoint", progress+3, "parallel read-tool checkpoint saved", "", true)
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
		executor.runner.appendRuntimeMessage(item.ID, state, messages, feedback)
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
	if provider, ok := session.selectedSkill.(skill.StructuredEvidenceValidatorProvider); ok {
		validatorForFinal = provider.ValidatorForRunWithEvidence(session.skillRequest, session.toolResults, state.Evidence)
	} else if provider, ok := session.selectedSkill.(skill.RunValidatorProvider); ok {
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
		executor.runner.appendRuntimeMessage(item.ID, state, messages, feedback)
		state.Messages = *messages
		executor.runner.recordStep(item, state, stepID, task.StatusRunning, "repairing_final", min(progress+5, 96), err.Error(), "validation_error", false)
		if checkpointErr := executor.runner.saveCheckpoint(ctx, item, state, stepID, round+1); checkpointErr != nil {
			return nil, false, executor.runner.fail(item, "checkpoint_failed", checkpointErr)
		}
		return nil, false, nil
	}
	state.addEvidence(stepID, state.allEvidence())
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

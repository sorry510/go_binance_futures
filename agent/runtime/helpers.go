package runtime

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"syscall"
	"time"

	"go_binance_futures/agent/contextengine"
	"go_binance_futures/agent/task"
	"go_binance_futures/agent/toolruntime"
	"go_binance_futures/llm"
)

func newTaskID() string {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err == nil {
		return hex.EncodeToString(value)
	}
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func contextSize(system string, messages []llm.Message) int {
	size := len(system)
	for _, message := range messages {
		size += len(message.Role) + len(message.Content)
	}
	return size
}

func mergeUsage(target *task.Usage, usage llm.Usage) {
	target.InputTokens += usage.InputTokens
	target.OutputTokens += usage.OutputTokens
	target.TotalTokens += usage.TotalTokens
}

func (runner *DefaultRunner) record(item *task.Task, status task.Status, stage string, progress int, message string) {
	now := time.Now().UTC()
	item.Status = status
	item.Stage = stage
	item.Progress = progress
	item.UpdatedAt = now
	event := task.Event{TaskID: item.ID, Stage: stage, Progress: progress, Round: item.Round, Message: message, Skill: item.Skill, Time: now}
	runner.persistEvent(item, event)
}

func (runner *DefaultRunner) recordStep(item *task.Task, state *RunState, stepID string, status task.Status, stage string, progress int, message, errorType string, checkpoint bool) {
	now := time.Now().UTC()
	item.Status = status
	item.Stage = stage
	item.Progress = progress
	item.UpdatedAt = now
	state.syncTask(item)
	event := task.Event{TaskID: item.ID, StepID: stepID, Stage: stage, Progress: progress, Round: item.Round, Message: message, Skill: item.Skill, ErrorType: errorType, Checkpoint: checkpoint, Time: now}
	if step := state.step(stepID); step != nil {
		event.StepType = string(step.Type)
		event.Status = string(step.Status)
	}
	runner.persistEvent(item, event)
}

func (runner *DefaultRunner) recordToolStep(item *task.Task, state *RunState, stepID string, status task.Status, stage string, progress int, toolName, outcome, errorType string, checkpoint bool, duration time.Duration, message string) {
	now := time.Now().UTC()
	item.Status = status
	item.Stage = stage
	item.Progress = progress
	item.UpdatedAt = now
	state.syncTask(item)
	event := task.Event{TaskID: item.ID, StepID: stepID, StepType: string(StepTool), Stage: stage, Progress: progress, Round: item.Round, Message: message, Skill: item.Skill, Tool: toolName, Status: outcome, ErrorType: errorType, Checkpoint: checkpoint, DurationMs: duration.Milliseconds(), Time: now}
	runner.persistEvent(item, event)
}

func (runner *DefaultRunner) persistEvent(item *task.Task, event task.Event) {
	item.Events = append(item.Events, event)
	_ = runner.cfg.Tasks.Save(context.Background(), item)
	if runner.cfg.EventHook != nil {
		runner.cfg.EventHook(event)
	}
}

func (runner *DefaultRunner) recordToolWaiting(item *task.Task, stage string, progress int, toolName, message string) {
	now := time.Now().UTC()
	item.Status = task.StatusWaitingTool
	item.Stage = stage
	item.Progress = progress
	item.UpdatedAt = now
	event := task.Event{TaskID: item.ID, Stage: stage, Progress: progress, Round: item.Round, Message: message, Skill: item.Skill, Tool: toolName, Status: "running", Time: now}
	item.Events = append(item.Events, event)
	_ = runner.cfg.Tasks.Save(context.Background(), item)
	if runner.cfg.EventHook != nil {
		runner.cfg.EventHook(event)
	}
}

func (runner *DefaultRunner) recordTool(item *task.Task, stage string, progress int, toolName, outcome string, duration time.Duration, message string) {
	now := time.Now().UTC()
	item.Status = task.StatusRunning
	item.Stage = stage
	item.Progress = progress
	item.UpdatedAt = now
	event := task.Event{TaskID: item.ID, Stage: stage, Progress: progress, Round: item.Round, Message: message, Skill: item.Skill, Tool: toolName, Status: outcome, DurationMs: duration.Milliseconds(), Time: now}
	item.Events = append(item.Events, event)
	_ = runner.cfg.Tasks.Save(context.Background(), item)
	if runner.cfg.EventHook != nil {
		runner.cfg.EventHook(event)
	}
}

func (runner *DefaultRunner) succeed(item *task.Task, stage, message string) {
	now := time.Now().UTC()
	item.CompletedAt = &now
	runner.record(item, task.StatusSucceeded, stage, 100, message)
}

func (runner *DefaultRunner) fail(item *task.Task, stage string, err error) error {
	now := time.Now().UTC()
	item.Error = err.Error()
	item.CompletedAt = &now
	runner.record(item, task.StatusFailed, stage, 100, err.Error())
	return err
}

func (runner *DefaultRunner) finishContextError(item *task.Task, err error) error {
	if errors.Is(err, context.Canceled) {
		now := time.Now().UTC()
		item.Error = err.Error()
		item.CompletedAt = &now
		runner.record(item, task.StatusCancelled, "cancelled", 100, err.Error())
		return err
	}
	return runner.fail(item, "timeout", err)
}

func (runner *DefaultRunner) generateWithRetry(ctx context.Context, request llm.Request, item *task.Task, state *RunState, stepID string, progress int) (*llm.Response, error) {
	attempts := runner.cfg.Retry.MaxAttempts
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		started := time.Now()
		response, err := runner.cfg.Client.Generate(ctx, request)
		status := "success"
		errorType, errorMessage := "", ""
		if err != nil {
			status = "error"
			errorType = classifyError(err)
			errorMessage = err.Error()
		}
		runner.observe(Observation{
			Type: "llm_call", TaskID: item.ID, ConversationID: item.ConversationID, Skill: item.Skill,
			Provider: item.Provider, Model: item.Model, Status: status, ErrorType: errorType, Error: errorMessage,
			Round: item.Round, DurationMs: elapsedMilliseconds(started),
		})
		if err == nil {
			return response, nil
		}
		lastErr = err
		if ctx.Err() != nil || !retryableLLMError(err) || attempt == attempts {
			break
		}
		runner.recordStep(item, state, stepID, task.StatusWaitingLLM, "retrying_llm", min(progress+1, 94),
			fmt.Sprintf("LLM request failed; retrying attempt %d/%d: %s", attempt+1, attempts, err.Error()), classifyError(err), false)
		if runner.cfg.Retry.Delay > 0 {
			timer := time.NewTimer(runner.cfg.Retry.Delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, ctx.Err()
			case <-timer.C:
			}
		}
	}
	return nil, lastErr
}

func retryableLLMError(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, net.ErrClosed) ||
		errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.EPIPE) {
		return true
	}
	var networkError net.Error
	if errors.As(err, &networkError) && (networkError.Timeout() || networkError.Temporary()) {
		return true
	}
	var httpError *llm.HTTPError
	return errors.As(err, &httpError) && (httpError.StatusCode == 429 || httpError.StatusCode >= 500)
}

func buildToolResultMessage(envelope toolruntime.ToolResultEnvelope, evidence []contextengine.Evidence, toolErr error) (llm.Message, error) {
	payload := map[string]any{"tool": envelope.Source, "ok": toolErr == nil, "result": envelope}
	if len(evidence) > 0 {
		payload["evidence"] = evidence
	}
	if toolErr != nil {
		payload["error"] = toolErr.Error()
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return llm.Message{}, fmt.Errorf("encode tool result envelope: %w", err)
	}
	return llm.Message{Role: llm.RoleUser, Content: "TOOL_RESULT\n" + string(data)}, nil
}

func repairFeedback(kind, message string) llm.Message {
	payload, _ := json.Marshal(map[string]string{"type": kind, "error": message})
	return llm.Message{
		Role: llm.RoleUser,
		Content: "AGENT_FEEDBACK\n" + string(payload) +
			"\nThe previous response is invalid. Return one complete replacement tool or final decision.",
	}
}

func (runner *DefaultRunner) appendRuntimeMessage(taskID string, state *RunState, messages *[]llm.Message, message llm.Message, override ...contextengine.ContextBlock) {
	*messages = append(*messages, message)
	if state != nil {
		var block contextengine.ContextBlock
		if len(override) > 0 {
			block = override[0]
			block.Content = message.Content
			block.ContentHash = contextengine.ContentHash(message.Content)
			block.Role = message.Role
		} else {
			block = contextengine.RuntimeMessageBlock(fmt.Sprintf("runtime-%03d", len(state.ContextBlocks)+1), message)
		}
		state.appendContextBlock(block)
	}
	if runner.cfg.MessageHook != nil {
		runner.cfg.MessageHook(taskID, message)
	}
}

func isTruncatedFinishReason(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "length", "max_tokens", "max_output_tokens":
		return true
	default:
		return false
	}
}

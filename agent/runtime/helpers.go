package runtime

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"

	"go_binance_futures/agent/task"
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
	event := task.Event{TaskID: item.ID, Stage: stage, Progress: progress, Message: message, Time: now}
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

func (runner *DefaultRunner) generateWithRetry(ctx context.Context, request llm.Request) (*llm.Response, error) {
	attempts := runner.cfg.Retry.MaxAttempts
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		response, err := runner.cfg.Client.Generate(ctx, request)
		if err == nil {
			return response, nil
		}
		lastErr = err
		if ctx.Err() != nil || !retryableLLMError(err) || attempt == attempts {
			break
		}
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
	var networkError net.Error
	if errors.As(err, &networkError) && (networkError.Timeout() || networkError.Temporary()) {
		return true
	}
	var httpError *llm.HTTPError
	return errors.As(err, &httpError) && (httpError.StatusCode == 429 || httpError.StatusCode >= 500)
}

func buildToolResultMessage(name string, value any, toolErr error, maxBytes int) (llm.Message, error) {
	payload := map[string]any{"tool": name, "ok": toolErr == nil}
	if toolErr != nil {
		payload["error"] = toolErr.Error()
	} else {
		payload["result"] = value
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return llm.Message{}, fmt.Errorf("encode tool result: %w", err)
	}
	if maxBytes > 0 && len(data) > maxBytes {
		return llm.Message{}, fmt.Errorf("tool %q result exceeds %d bytes", name, maxBytes)
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

package runtime

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"go_binance_futures/agent/task"
	"go_binance_futures/llm"
)

func (runner *DefaultRunner) observe(value Observation) {
	if runner == nil || runner.cfg.Observer == nil {
		return
	}
	runner.cfg.Observer.Observe(value)
}

func (runner *DefaultRunner) observeRepair(item *task.Task, kind string) {
	runner.observe(Observation{
		Type: "repair", TaskID: item.ID, ConversationID: item.ConversationID,
		Skill: item.Skill, Provider: item.Provider, Model: item.Model,
		Round: item.Round, Status: strings.TrimSpace(kind),
	})
}
func classifyError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	if errors.Is(err, context.Canceled) {
		return "cancelled"
	}
	if errors.Is(err, io.EOF) {
		return "transport_eof"
	}
	var httpErr *llm.HTTPError
	if errors.As(err, &httpErr) {
		switch {
		case httpErr.StatusCode == 429:
			return "rate_limit"
		case httpErr.StatusCode >= 500:
			return "provider_5xx"
		default:
			return fmt.Sprintf("http_%d", httpErr.StatusCode)
		}
	}
	return "runtime_error"
}

func elapsedMilliseconds(started time.Time) int64 {
	value := time.Since(started).Milliseconds()
	if value < 0 {
		return 0
	}
	return value
}

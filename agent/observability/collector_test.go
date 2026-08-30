package observability

import (
	"strings"
	"testing"

	agentruntime "go_binance_futures/agent/runtime"
	"go_binance_futures/agent/task"
)

func TestCollectorAggregatesRuntimeMetrics(t *testing.T) {
	collector := New()
	base := agentruntime.Observation{TaskID: "task-1", Skill: "symbol_analysis", Provider: "test"}
	started := base
	started.Type = "task_started"
	collector.Observe(started)
	llmError := base
	llmError.Type, llmError.Status = "llm_call", "error"
	collector.Observe(llmError)
	toolError := base
	toolError.Type, toolError.Status = "tool_call", "error"
	collector.Observe(toolError)
	repair := base
	repair.Type = "repair"
	collector.Observe(repair)
	validation := base
	validation.Type, validation.Status = "validation", "error"
	collector.Observe(validation)
	finished := base
	finished.Type = "task_finished"
	finished.Status = "succeeded"
	finished.Round = 3
	finished.DurationMs = 120
	finished.Usage = task.Usage{InputTokens: 10, OutputTokens: 5, TotalTokens: 15}
	collector.Observe(finished)

	metrics := collector.Snapshot().Global
	if metrics.TasksStarted != 1 || metrics.TasksSucceeded != 1 || metrics.ActiveTasks != 0 {
		t.Fatalf("unexpected task metrics: %+v", metrics)
	}
	if metrics.LLMCalls != 1 || metrics.LLMErrors != 1 || metrics.ToolCalls != 1 || metrics.ToolErrors != 1 {
		t.Fatalf("unexpected call metrics: %+v", metrics)
	}
	if metrics.Repairs != 1 || metrics.ValidationErrors != 1 || metrics.TotalTokens != 15 {
		t.Fatalf("unexpected repair/token metrics: %+v", metrics)
	}
	if metrics.AverageRounds != 3 || metrics.P50DurationMs != 120 || metrics.P95DurationMs != 120 {
		t.Fatalf("unexpected duration/round metrics: %+v", metrics)
	}
	if metrics.TaskSuccessRate != 1 || metrics.LLMErrorRate != 1 || metrics.ToolErrorRate != 1 {
		t.Fatalf("unexpected rate metrics: %+v", metrics)
	}
}

func TestObservationLogRedactsSecrets(t *testing.T) {
	line := formatObservationLog(agentruntime.Observation{
		Type: "task_finished", TaskID: "task-1", Skill: "symbol_analysis", Status: "failed",
		ErrorType: "llm_failed", Error: `authorization=secret-token {"api_key":"json-key"} Bearer bearer-key`,
	})
	for _, secret := range []string{"secret-token", "json-key", "bearer-key"} {
		if strings.Contains(line, secret) {
			t.Fatalf("secret %q leaked in structured log: %s", secret, line)
		}
	}
	for _, field := range []string{"task_id=task-1", "skill=symbol_analysis", "status=failed", "error_type=llm_failed"} {
		if !strings.Contains(line, field) {
			t.Fatalf("structured log missing %q: %s", field, line)
		}
	}
}

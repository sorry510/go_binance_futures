package observability

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	agentruntime "go_binance_futures/agent/runtime"
	"go_binance_futures/agent/security"

	"github.com/beego/beego/v2/core/logs"
)

type Metrics struct {
	TasksStarted     uint64  `json:"tasks_started"`
	TasksSucceeded   uint64  `json:"tasks_succeeded"`
	TasksFailed      uint64  `json:"tasks_failed"`
	TasksCancelled   uint64  `json:"tasks_cancelled"`
	ActiveTasks      int64   `json:"active_tasks"`
	LLMCalls         uint64  `json:"llm_calls"`
	LLMErrors        uint64  `json:"llm_errors"`
	ToolCalls        uint64  `json:"tool_calls"`
	ToolErrors       uint64  `json:"tool_errors"`
	ValidationErrors uint64  `json:"validation_errors"`
	Repairs          uint64  `json:"repairs"`
	InputTokens      uint64  `json:"input_tokens"`
	OutputTokens     uint64  `json:"output_tokens"`
	TotalTokens      uint64  `json:"total_tokens"`
	TotalRounds      uint64  `json:"total_rounds"`
	AverageRounds    float64 `json:"average_rounds"`
	TaskSuccessRate  float64 `json:"task_success_rate"`
	LLMErrorRate     float64 `json:"llm_error_rate"`
	ToolErrorRate    float64 `json:"tool_error_rate"`
	P50DurationMs    int64   `json:"p50_duration_ms"`
	P95DurationMs    int64   `json:"p95_duration_ms"`
}
type Snapshot struct {
	Global Metrics            `json:"global"`
	Skills map[string]Metrics `json:"skills"`
}

type metricState struct {
	Metrics
	durations []int64
	finished  uint64
}

type Collector struct {
	mu      sync.Mutex
	global  metricState
	skills  map[string]*metricState
	persist bool
}

func New() *Collector {
	return &Collector{skills: map[string]*metricState{}}
}

var defaultCollector = &Collector{skills: map[string]*metricState{}, persist: true}

func Default() *Collector { return defaultCollector }

func (collector *Collector) Observe(value agentruntime.Observation) {
	if collector == nil {
		return
	}
	collector.mu.Lock()
	collector.apply(&collector.global, value)
	skill := strings.TrimSpace(value.Skill)
	if skill != "" {
		state := collector.skills[skill]
		if state == nil {
			state = &metricState{}
			collector.skills[skill] = state
		}
		collector.apply(state, value)
	}
	collector.mu.Unlock()
	collector.log(value)
	if collector.persist {
		persistObservation(context.Background(), value)
	}
}
func (collector *Collector) apply(state *metricState, value agentruntime.Observation) {
	switch value.Type {
	case "task_started":
		state.TasksStarted++
		state.ActiveTasks++
	case "task_finished":
		if state.ActiveTasks > 0 {
			state.ActiveTasks--
		}
		state.finished++
		state.TotalRounds += uint64(max(value.Round, 0))
		state.InputTokens += uint64(max(value.Usage.InputTokens, 0))
		state.OutputTokens += uint64(max(value.Usage.OutputTokens, 0))
		state.TotalTokens += uint64(max(value.Usage.TotalTokens, 0))
		state.durations = appendDuration(state.durations, value.DurationMs)
		switch value.Status {
		case "succeeded":
			state.TasksSucceeded++
		case "cancelled":
			state.TasksCancelled++
		default:
			state.TasksFailed++
		}
	case "llm_call":
		state.LLMCalls++
		if value.Status == "error" {
			state.LLMErrors++
		}
	case "tool_call":
		state.ToolCalls++
		if value.Status == "error" {
			state.ToolErrors++
		}
	case "validation":
		if value.Status == "error" {
			state.ValidationErrors++
		}
	case "repair":
		state.Repairs++
	}
}
func (collector *Collector) Snapshot() Snapshot {
	if collector == nil {
		return Snapshot{}
	}
	collector.mu.Lock()
	defer collector.mu.Unlock()
	result := Snapshot{Global: snapshotMetrics(collector.global), Skills: map[string]Metrics{}}
	for name, state := range collector.skills {
		result.Skills[name] = snapshotMetrics(*state)
	}
	return result
}

func snapshotMetrics(state metricState) Metrics {
	result := state.Metrics
	if state.finished > 0 {
		result.AverageRounds = float64(state.TotalRounds) / float64(state.finished)
		result.TaskSuccessRate = float64(state.TasksSucceeded) / float64(state.finished)
	}
	if state.LLMCalls > 0 {
		result.LLMErrorRate = float64(state.LLMErrors) / float64(state.LLMCalls)
	}
	if state.ToolCalls > 0 {
		result.ToolErrorRate = float64(state.ToolErrors) / float64(state.ToolCalls)
	}
	if len(state.durations) > 0 {
		values := append([]int64(nil), state.durations...)
		sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
		result.P50DurationMs = percentile(values, 0.50)
		result.P95DurationMs = percentile(values, 0.95)
	}
	return result
}

func appendDuration(values []int64, value int64) []int64 {
	if value < 0 {
		value = 0
	}
	values = append(values, value)
	if len(values) > 1000 {
		values = append([]int64(nil), values[len(values)-1000:]...)
	}
	return values
}
func percentile(values []int64, ratio float64) int64 {
	if len(values) == 0 {
		return 0
	}
	index := int(float64(len(values)-1) * ratio)
	if index < 0 {
		index = 0
	}
	if index >= len(values) {
		index = len(values) - 1
	}
	return values[index]
}

func (collector *Collector) log(value agentruntime.Observation) {
	if value.Status != "error" && !(value.Type == "task_finished" && value.Status != "succeeded") {
		return
	}
	logs.Error("%s", formatObservationLog(value))
}

func formatObservationLog(value agentruntime.Observation) string {
	return fmt.Sprintf(
		"agent_observation task_id=%s conversation_id=%s skill=%s type=%s round=%d provider=%s model=%s tool=%s duration_ms=%d status=%s error_type=%s error=%s",
		security.RedactText(value.TaskID), security.RedactText(value.ConversationID), security.RedactText(value.Skill),
		security.RedactText(value.Type), value.Round, security.RedactText(value.Provider), security.RedactText(value.Model),
		security.RedactText(value.Tool), value.DurationMs, security.RedactText(value.Status), security.RedactText(value.ErrorType),
		security.RedactText(value.Error),
	)
}

func max(value, minimum int) int {
	if value < minimum {
		return minimum
	}
	return value
}

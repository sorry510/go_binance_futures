package alertpipeline

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	agentruntime "go_binance_futures/agent/runtime"
	alertanalysis "go_binance_futures/agent/skills/alertanalysis"
	"go_binance_futures/agent/task"
	"go_binance_futures/notify"
	signalservice "go_binance_futures/service/signal"
)

type pipelineTestHarness struct {
	mu            sync.Mutex
	settings      Settings
	tasks         map[string]*task.Task
	notifications []notify.AgentAlertParams
	startErr      error
}

func newPipelineTestHarness() *pipelineTestHarness {
	settings := DefaultSettings()
	settings.Enabled = true
	settings.AIEnabled = false
	settings.MinSeverity = signalservice.SeverityMedium
	settings.Cooldown = time.Hour
	return &pipelineTestHarness{settings: settings, tasks: map[string]*task.Task{}}
}
func (h *pipelineTestHarness) newPipeline(t *testing.T) *Pipeline {
	t.Helper()
	pipeline, err := New(Config{
		Settings:  func() Settings { return h.settings },
		StartTask: h.startTask, GetTask: h.getTask, Notify: h.notify,
		QueueSize: 8, Workers: 1, PollInterval: time.Millisecond, TaskTimeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	return pipeline
}

func (h *pipelineTestHarness) startTask(req agentruntime.Request) (*task.Task, error) {
	if h.startErr != nil {
		return nil, h.startErr
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	id := "task-test"
	item := h.tasks[id]
	if item == nil {
		item = &task.Task{ID: id, Skill: req.Skill, Status: task.StatusFailed, Error: "missing fake result"}
		h.tasks[id] = item
	}
	copyItem := *item
	return &copyItem, nil
}

func (h *pipelineTestHarness) getTask(_ context.Context, id string) (*task.Task, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	item := h.tasks[id]
	if item == nil {
		return nil, errors.New("task not found")
	}
	copyItem := *item
	copyItem.Result = append([]byte(nil), item.Result...)
	return &copyItem, nil
}
func (h *pipelineTestHarness) notify(params notify.AgentAlertParams) (int64, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.notifications = append(h.notifications, params)
	return int64(len(h.notifications)), nil
}

func testSignal(id string) signalservice.Signal {
	value := signalservice.NewSignal("evt-"+id, "BTCUSDT", signalservice.TypeFastMove, signalservice.SeverityHigh, "3m")
	value.SignalID = "sig-" + id
	value.Metrics["change_percent"] = 25
	value.Metrics["price"] = 100
	value.Metrics["threshold_percent"] = 20
	value.Labels["direction"] = "up"
	value.Evidence = []signalservice.Evidence{{Source: "price_tick", Finding: "BTCUSDT up 25%"}}
	return value
}

func waitFor(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not met before timeout")
}
func TestPipelineShadowModeDoesNotNotify(t *testing.T) {
	h := newPipelineTestHarness()
	h.settings.Enabled = false
	pipeline := h.newPipeline(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pipeline.Start(ctx)
	pipeline.Emit(testSignal("shadow"))
	waitFor(t, func() bool { return pipeline.Stats().ShadowSignals == 1 })
	if len(h.notifications) != 0 {
		t.Fatalf("shadow mode notified: %+v", h.notifications)
	}
}

func TestPipelineAIOffUsesFallbackAndCooldown(t *testing.T) {
	h := newPipelineTestHarness()
	pipeline := h.newPipeline(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pipeline.Start(ctx)
	pipeline.Emit(testSignal("one"))
	waitFor(t, func() bool { return pipeline.Stats().Notifications == 1 })
	pipeline.Emit(testSignal("two"))
	waitFor(t, func() bool { return pipeline.Stats().SuppressedCooldown == 1 })
	if len(h.notifications) != 1 || !h.notifications[0].Fallback {
		t.Fatalf("unexpected fallback notifications: %+v", h.notifications)
	}
}
func TestPipelineAISuccessUsesStructuredResult(t *testing.T) {
	h := newPipelineTestHarness()
	h.settings.AIEnabled = true
	result := alertanalysis.AlertV1{
		Version: "alert_v1", AlertID: "alert_sig-ai", SignalID: "sig-ai", Symbol: "BTCUSDT",
		Type: signalservice.TypeFastMove, Severity: signalservice.SeverityHigh,
		Summary: "AI confirmed move", MarketContext: "funding and OI confirm", ConfirmedBy: []string{"OI"},
		Risks: []string{"reversal"}, Action: "notify", CooldownUntil: time.Now().Add(time.Hour).UTC().Format(time.RFC3339),
		DataMissing: []string{}, Evidence: []alertanalysis.Evidence{{Source: "get_symbol_analysis_context", Finding: "confirmed"}},
	}
	raw, _ := json.Marshal(result)
	h.tasks["task-test"] = &task.Task{ID: "task-test", Skill: alertanalysis.Name, Status: task.StatusSucceeded, Result: raw}
	pipeline := h.newPipeline(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pipeline.Start(ctx)
	pipeline.Emit(testSignal("ai"))
	waitFor(t, func() bool { return pipeline.Stats().Notifications == 1 })
	if len(h.notifications) != 1 || h.notifications[0].Fallback || h.notifications[0].TaskID != "task-test" {
		t.Fatalf("unexpected AI notification: %+v", h.notifications)
	}
}

func TestPipelineAIStartFailureFallsBack(t *testing.T) {
	h := newPipelineTestHarness()
	h.settings.AIEnabled = true
	h.startErr = errors.New("llm unavailable")
	pipeline := h.newPipeline(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pipeline.Start(ctx)
	pipeline.Emit(testSignal("failure"))
	waitFor(t, func() bool { return pipeline.Stats().AIFallbacks == 1 })
	if len(h.notifications) != 1 || !h.notifications[0].Fallback {
		t.Fatalf("AI failure did not fallback: %+v", h.notifications)
	}
}

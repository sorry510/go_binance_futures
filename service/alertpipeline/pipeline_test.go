package alertpipeline

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	agentruntime "go_binance_futures/agent/runtime"
	alertanalysis "go_binance_futures/agent/skills/alertanalysis"
	workflowSkills "go_binance_futures/agent/skills/workflows"
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
	settings.TriageWindow = 5 * time.Millisecond
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

func TestFallbackNotificationRendersLocalizedValuesWithoutFormatErrors(t *testing.T) {
	fastMove := fallbackNotification(testSignal("localized-fast-move"), "task-1", "AI unavailable")
	liquidation := signalservice.NewSignal("evt-liquidation", "BTCUSDT", signalservice.TypeLiquidationSpike, signalservice.SeverityCritical, "1m")
	liquidation.Metrics["aggregate_notional"] = 8_000_000
	liquidation.Metrics["order_count"] = 4
	liquidation.Metrics["threshold_notional"] = 5_000_000
	liquidation.Labels["liquidation_side"] = "long"
	liquidationAlert := fallbackNotification(liquidation, "task-2", "")

	for _, params := range []notify.AgentAlertParams{fastMove, liquidationAlert} {
		if strings.Contains(params.Summary, "%!") || strings.Contains(params.MarketContext, "%!") {
			t.Fatalf("localized fallback contains a format error: %+v", params)
		}
		if strings.TrimSpace(params.Summary) == "" || strings.TrimSpace(params.MarketContext) == "" || len(params.ConfirmedBy) == 0 {
			t.Fatalf("localized fallback is incomplete: %+v", params)
		}
	}
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
	waitFor(t, func() bool { return pipeline.Stats().Notifications == 1 })
	stats := pipeline.Stats()
	if len(h.notifications) != 1 || !h.notifications[0].Fallback {
		t.Fatalf("AI failure did not fallback: %+v", h.notifications)
	}
	if stats.EligibleSignals != 1 || stats.SignalNotifyRate != 1 || stats.AIFallbackRate != 1 {
		t.Fatalf("unexpected fallback rates: %+v", stats)
	}
}

func TestPipelineFailedAITaskFallsBack(t *testing.T) {
	h := newPipelineTestHarness()
	h.settings.AIEnabled = true
	h.tasks["task-test"] = &task.Task{ID: "task-test", Skill: alertanalysis.Name, Status: task.StatusFailed, Error: "tool unavailable"}
	pipeline := h.newPipeline(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pipeline.Start(ctx)
	pipeline.Emit(testSignal("task-failure"))
	waitFor(t, func() bool { return pipeline.Stats().Notifications == 1 })
	if len(h.notifications) != 1 || !h.notifications[0].Fallback || h.notifications[0].TaskID != "task-test" {
		t.Fatalf("failed AI task did not fallback: %+v", h.notifications)
	}
}

func testLiquidationSignal(id string) signalservice.Signal {
	value := signalservice.NewSignal("evt-"+id, "BTCUSDT", signalservice.TypeLiquidationSpike, signalservice.SeverityHigh, "1m")
	value.SignalID = "sig-" + id
	value.Metrics["aggregate_notional"] = 8_000_000
	value.Metrics["order_count"] = 4
	value.Metrics["threshold_notional"] = 5_000_000
	value.Labels["liquidation_side"] = "long"
	return value
}

func TestPipelineAIEnabledCoalescesCrossTypeSignalsIntoOneIncidentNotification(t *testing.T) {
	h := newPipelineTestHarness()
	h.settings.AIEnabled = true
	result := workflowSkills.IncidentSetV1{
		Version: "incident_set_v1", AsOf: time.Now().UTC().Format(time.RFC3339),
		Incidents: []workflowSkills.Incident{{
			IncidentID: "incident-btc-1", SignalIDs: []string{"sig-fast", "sig-liquidation"}, Symbols: []string{"BTCUSDT"},
			Severity: "high", Action: "notify", Summary: "同一轮 BTC 波动与强平事件", Rationale: "同一交易对且时间高度重叠",
		}},
	}
	raw, _ := json.Marshal(result)
	h.tasks["task-test"] = &task.Task{ID: "task-test", Skill: workflowSkills.AlertTriageName, Status: task.StatusSucceeded, Result: raw}
	pipeline := h.newPipeline(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pipeline.Start(ctx)
	pipeline.Emit(testSignal("fast"))
	pipeline.Emit(testLiquidationSignal("liquidation"))
	waitFor(t, func() bool { return pipeline.Stats().Notifications == 1 })
	stats := pipeline.Stats()
	if len(h.notifications) != 1 || h.notifications[0].SignalID != "incident-btc-1" || h.notifications[0].Fallback {
		t.Fatalf("signals were not coalesced into one incident notification: %+v", h.notifications)
	}
	if stats.TriageBatches != 1 || stats.TriageSignals != 2 || stats.TriageTasksStarted != 1 || stats.AITasksStarted != 1 {
		t.Fatalf("unexpected triage stats: %+v", stats)
	}
}

func TestPipelineAIDisabledKeepsDeterministicPerSignalFallback(t *testing.T) {
	h := newPipelineTestHarness()
	h.settings.AIEnabled = false
	pipeline := h.newPipeline(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pipeline.Start(ctx)
	pipeline.Emit(testSignal("fast-off"))
	pipeline.Emit(testLiquidationSignal("liquidation-off"))
	waitFor(t, func() bool { return pipeline.Stats().Notifications == 2 })
	stats := pipeline.Stats()
	if len(h.notifications) != 2 || !h.notifications[0].Fallback || !h.notifications[1].Fallback {
		t.Fatalf("AI-disabled deterministic fallback changed unexpectedly: %+v", h.notifications)
	}
	if stats.TriageBatches != 0 || stats.TriageTasksStarted != 0 {
		t.Fatalf("AI-disabled signals unexpectedly entered triage: %+v", stats)
	}
}

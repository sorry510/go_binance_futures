package alertpipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	agentevent "go_binance_futures/agent/event"
	agentruntime "go_binance_futures/agent/runtime"
	alertanalysis "go_binance_futures/agent/skills/alertanalysis"
	"go_binance_futures/agent/task"
	"go_binance_futures/notify"
	signalservice "go_binance_futures/service/signal"
)

type StartTaskFunc func(agentruntime.Request) (*task.Task, error)
type GetTaskFunc func(context.Context, string) (*task.Task, error)
type NotifyFunc func(notify.AgentAlertParams) (int64, error)

type SettingsProvider func() Settings

type Config struct {
	Bus          *agentevent.Bus
	Settings     SettingsProvider
	StartTask    StartTaskFunc
	GetTask      GetTaskFunc
	Notify       NotifyFunc
	QueueSize    int
	Workers      int
	PollInterval time.Duration
	TaskTimeout  time.Duration
}

type Pipeline struct {
	cfg       Config
	queue     chan signalservice.Signal
	startOnce sync.Once
	mu        sync.Mutex
	lastRun   map[string]int64
	aiCalls   []int64
	traces    []Trace

	signalsReceived    atomic.Uint64
	signalsDropped     atomic.Uint64
	shadowSignals      atomic.Uint64
	suppressedCooldown atomic.Uint64
	aiTasksStarted     atomic.Uint64
	aiFallbacks        atomic.Uint64
	notifications      atomic.Uint64
	activeAI           atomic.Int64
}

func New(cfg Config) (*Pipeline, error) {
	if cfg.Settings == nil {
		cfg.Settings = func() Settings { return DefaultSettings() }
	}
	if cfg.StartTask == nil || cfg.GetTask == nil || cfg.Notify == nil {
		return nil, fmt.Errorf("alert pipeline requires task and notify dependencies")
	}
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 256
	}
	if cfg.Workers <= 0 {
		cfg.Workers = 2
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 500 * time.Millisecond
	}
	if cfg.TaskTimeout <= 0 {
		cfg.TaskTimeout = 2 * time.Minute
	}
	return &Pipeline{
		cfg: cfg, queue: make(chan signalservice.Signal, cfg.QueueSize), lastRun: map[string]int64{},
	}, nil
}
func (pipeline *Pipeline) Start(ctx context.Context) {
	if pipeline == nil {
		return
	}
	pipeline.startOnce.Do(func() {
		for index := 0; index < pipeline.cfg.Workers; index++ {
			go pipeline.worker(ctx)
		}
	})
}

func (pipeline *Pipeline) Emit(value signalservice.Signal) bool {
	if pipeline == nil || value.SignalID == "" {
		return false
	}
	pipeline.signalsReceived.Add(1)
	select {
	case pipeline.queue <- value:
		return true
	default:
		pipeline.signalsDropped.Add(1)
		return false
	}
}

func (pipeline *Pipeline) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case value := <-pipeline.queue:
			pipeline.process(ctx, value)
		}
	}
}
func (pipeline *Pipeline) process(ctx context.Context, value signalservice.Signal) {
	settings := pipeline.cfg.Settings()
	trace := newTrace(value, "received")
	if !settings.Enabled {
		pipeline.shadowSignals.Add(1)
		trace.Status = "shadow"
		pipeline.addTrace(trace)
		return
	}
	if !signalservice.SeverityAtLeast(value.Severity, settings.MinSeverity) {
		trace.Status = "below_severity"
		pipeline.addTrace(trace)
		return
	}
	if pipeline.inCooldown(value, settings.Cooldown) {
		pipeline.suppressedCooldown.Add(1)
		trace.Status = "cooldown"
		pipeline.addTrace(trace)
		return
	}
	pipeline.markCooldown(value)
	if !settings.AIEnabled {
		pipeline.notifyFallback(value, &trace, "AI disabled")
		return
	}
	if !pipeline.reserveAIBudget(settings.MaxPerMinute) {
		pipeline.notifyFallback(value, &trace, "AI minute budget exceeded")
		return
	}
	if !pipeline.acquireAISlot(settings.MaxConcurrent) {
		pipeline.notifyFallback(value, &trace, "AI concurrency limit reached")
		return
	}
	defer pipeline.releaseAISlot()
	pipeline.runAI(ctx, value, &trace)
}
func (pipeline *Pipeline) runAI(ctx context.Context, value signalservice.Signal, trace *Trace) {
	alertID := "alert_" + value.SignalID
	input, err := json.Marshal(alertanalysis.Input{AlertID: alertID, Signal: value})
	if err != nil {
		pipeline.notifyFallback(value, trace, err.Error())
		return
	}
	item, err := pipeline.cfg.StartTask(agentruntime.Request{Skill: alertanalysis.Name, Input: string(input)})
	if err != nil {
		pipeline.notifyFallback(value, trace, err.Error())
		return
	}
	pipeline.aiTasksStarted.Add(1)
	trace.TaskID = item.ID
	trace.Status = "ai_running"
	pipeline.addTrace(*trace)

	completed, err := pipeline.waitTask(ctx, item.ID)
	if err != nil {
		pipeline.notifyFallback(value, trace, err.Error())
		return
	}
	if completed.Status != task.StatusSucceeded {
		message := completed.Error
		if message == "" {
			message = "alert analysis task did not succeed"
		}
		pipeline.notifyFallback(value, trace, message)
		return
	}
	var result alertanalysis.AlertV1
	if err := json.Unmarshal(completed.Result, &result); err != nil {
		pipeline.notifyFallback(value, trace, "decode alert result: "+err.Error())
		return
	}
	pipeline.applyAIResult(value, result, trace)
}
func (pipeline *Pipeline) waitTask(ctx context.Context, taskID string) (*task.Task, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, pipeline.cfg.TaskTimeout)
	defer cancel()
	ticker := time.NewTicker(pipeline.cfg.PollInterval)
	defer ticker.Stop()
	for {
		item, err := pipeline.cfg.GetTask(timeoutCtx, taskID)
		if err != nil {
			return nil, err
		}
		switch item.Status {
		case task.StatusSucceeded, task.StatusFailed, task.StatusCancelled:
			return item, nil
		}
		select {
		case <-timeoutCtx.Done():
			return nil, timeoutCtx.Err()
		case <-ticker.C:
		}
	}
}

func (pipeline *Pipeline) applyAIResult(value signalservice.Signal, result alertanalysis.AlertV1, trace *Trace) {
	trace.Action = result.Action
	trace.Status = "ai_" + result.Action
	if result.Action != "notify" {
		pipeline.addTrace(*trace)
		return
	}
	params := notify.AgentAlertParams{
		Title: "AI 异常行情分析", EventID: value.EventID, SignalID: value.SignalID,
		TaskID: trace.TaskID, Symbol: value.Symbol, SignalType: string(value.Type), Severity: string(result.Severity),
		Summary: result.Summary, MarketContext: result.MarketContext, ConfirmedBy: result.ConfirmedBy,
		Risks: result.Risks, Source: "AI", Fallback: false,
	}
	pipeline.sendNotification(params, trace)
}
func (pipeline *Pipeline) notifyFallback(value signalservice.Signal, trace *Trace, reason string) {
	pipeline.aiFallbacks.Add(1)
	trace.Fallback = true
	trace.Action = "notify"
	trace.Error = strings.TrimSpace(reason)
	trace.Status = "fallback_notify"
	params := fallbackNotification(value, trace.TaskID, reason)
	pipeline.sendNotification(params, trace)
}

func (pipeline *Pipeline) sendNotification(params notify.AgentAlertParams, trace *Trace) {
	id, err := pipeline.cfg.Notify(params)
	if err != nil {
		trace.Status = "notify_failed"
		trace.Error = err.Error()
		pipeline.addTrace(*trace)
		return
	}
	pipeline.notifications.Add(1)
	trace.NotificationID = id
	if trace.Fallback {
		trace.Status = "fallback_notified"
	} else {
		trace.Status = "notified"
	}
	pipeline.addTrace(*trace)
}

func fallbackNotification(value signalservice.Signal, taskID, reason string) notify.AgentAlertParams {
	summary, contextText := fallbackSummary(value)
	if strings.TrimSpace(reason) != "" {
		contextText += "；AI fallback: " + strings.TrimSpace(reason)
	}
	return notify.AgentAlertParams{
		Title: "异常行情规则报警", EventID: value.EventID, SignalID: value.SignalID, TaskID: taskID,
		Symbol: value.Symbol, SignalType: string(value.Type), Severity: string(value.Severity),
		Summary: summary, MarketContext: contextText, ConfirmedBy: signalEvidence(value),
		Risks: []string{"规则报警未经过完整 AI 交叉验证"}, Source: "RULE", Fallback: true,
	}
}
func fallbackSummary(value signalservice.Signal) (string, string) {
	switch value.Type {
	case signalservice.TypeFastMove:
		change := value.Metrics["change_percent"]
		price := value.Metrics["price"]
		direction := value.Labels["direction"]
		return fmt.Sprintf("%s 在 %s 内%s %.2f%%", value.Symbol, value.Window, directionText(direction), mathAbs(change)),
			fmt.Sprintf("当前价格 %.8g，规则阈值 %.2f%%", price, value.Metrics["threshold_percent"])
	case signalservice.TypeLiquidationSpike:
		side := value.Labels["liquidation_side"]
		return fmt.Sprintf("%s 出现%s强平聚合异常", value.Symbol, sideText(side)),
			fmt.Sprintf("聚合名义金额 %.2f USDT，订单数 %.0f，阈值 %.2f USDT",
				value.Metrics["aggregate_notional"], value.Metrics["order_count"], value.Metrics["threshold_notional"])
	default:
		return fmt.Sprintf("%s 触发 %s 信号", value.Symbol, value.Type), "确定性 Signal Engine 已命中规则"
	}
}

func signalEvidence(value signalservice.Signal) []string {
	result := make([]string, 0, len(value.Evidence))
	for _, item := range value.Evidence {
		finding := strings.TrimSpace(item.Finding)
		if finding != "" {
			result = append(result, finding)
		}
	}
	return result
}

func directionText(value string) string {
	if value == "down" {
		return "快速下跌"
	}
	return "快速上涨"
}

func sideText(value string) string {
	if value == "short" {
		return "空头"
	}
	return "多头"
}

func mathAbs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}
func (pipeline *Pipeline) inCooldown(value signalservice.Signal, cooldown time.Duration) bool {
	if cooldown <= 0 {
		cooldown = DefaultSettings().Cooldown
	}
	key := string(value.Type) + "|" + value.Symbol
	now := time.Now().UnixMilli()
	pipeline.mu.Lock()
	defer pipeline.mu.Unlock()
	last := pipeline.lastRun[key]
	return last > 0 && now-last < cooldown.Milliseconds()
}

func (pipeline *Pipeline) markCooldown(value signalservice.Signal) {
	key := string(value.Type) + "|" + value.Symbol
	pipeline.mu.Lock()
	pipeline.lastRun[key] = time.Now().UnixMilli()
	pipeline.mu.Unlock()
}

func (pipeline *Pipeline) reserveAIBudget(limit int) bool {
	if limit <= 0 {
		limit = DefaultSettings().MaxPerMinute
	}
	now := time.Now().UnixMilli()
	cutoff := now - time.Minute.Milliseconds()
	pipeline.mu.Lock()
	defer pipeline.mu.Unlock()
	kept := pipeline.aiCalls[:0]
	for _, value := range pipeline.aiCalls {
		if value > cutoff {
			kept = append(kept, value)
		}
	}
	pipeline.aiCalls = kept
	if len(pipeline.aiCalls) >= limit {
		return false
	}
	pipeline.aiCalls = append(pipeline.aiCalls, now)
	return true
}
func newTrace(value signalservice.Signal, status string) Trace {
	now := time.Now().UnixMilli()
	return Trace{
		EventID: value.EventID, SignalID: value.SignalID, Symbol: value.Symbol,
		Type: value.Type, Severity: value.Severity, Status: status,
		CreatedAt: now, UpdatedAt: now,
	}
}

func (pipeline *Pipeline) addTrace(value Trace) {
	value.UpdatedAt = time.Now().UnixMilli()
	pipeline.mu.Lock()
	defer pipeline.mu.Unlock()
	for index := len(pipeline.traces) - 1; index >= 0; index-- {
		if pipeline.traces[index].SignalID == value.SignalID {
			pipeline.traces[index] = value
			return
		}
	}
	pipeline.traces = append(pipeline.traces, value)
	if len(pipeline.traces) > 100 {
		pipeline.traces = append([]Trace(nil), pipeline.traces[len(pipeline.traces)-100:]...)
	}
}

func (pipeline *Pipeline) Traces(limit int) []Trace {
	if pipeline == nil {
		return nil
	}
	pipeline.mu.Lock()
	defer pipeline.mu.Unlock()
	if limit <= 0 || limit > len(pipeline.traces) {
		limit = len(pipeline.traces)
	}
	result := make([]Trace, 0, limit)
	for index := len(pipeline.traces) - 1; index >= len(pipeline.traces)-limit; index-- {
		result = append(result, pipeline.traces[index])
	}
	return result
}
func (pipeline *Pipeline) Stats() Stats {
	if pipeline == nil {
		return Stats{}
	}
	return Stats{
		SignalsReceived:    pipeline.signalsReceived.Load(),
		SignalsDropped:     pipeline.signalsDropped.Load(),
		ShadowSignals:      pipeline.shadowSignals.Load(),
		SuppressedCooldown: pipeline.suppressedCooldown.Load(),
		AITasksStarted:     pipeline.aiTasksStarted.Load(),
		AIFallbacks:        pipeline.aiFallbacks.Load(),
		Notifications:      pipeline.notifications.Load(),
		ActiveAI:           int(pipeline.activeAI.Load()),
	}
}

func (pipeline *Pipeline) acquireAISlot(limit int) bool {
	if limit <= 0 {
		limit = DefaultSettings().MaxConcurrent
	}
	for {
		current := pipeline.activeAI.Load()
		if current >= int64(limit) {
			return false
		}
		if pipeline.activeAI.CompareAndSwap(current, current+1) {
			return true
		}
	}
}

func (pipeline *Pipeline) releaseAISlot() {
	pipeline.activeAI.Add(-1)
}

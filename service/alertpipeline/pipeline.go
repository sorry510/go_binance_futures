package alertpipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	agentevent "go_binance_futures/agent/event"
	agentruntime "go_binance_futures/agent/runtime"
	alertanalysis "go_binance_futures/agent/skills/alertanalysis"
	workflowSkills "go_binance_futures/agent/skills/workflows"
	"go_binance_futures/agent/task"
	"go_binance_futures/lang"
	"go_binance_futures/notify"

	"github.com/beego/beego/v2/core/logs"
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
	TraceStore   TraceRepository
}

type incidentBucket struct {
	signals []signalservice.Signal
	timer   *time.Timer
}

type Pipeline struct {
	cfg             Config
	queue           chan signalservice.Signal
	traceQueue      chan Trace
	incidentQueue   chan []signalservice.Signal
	startOnce       sync.Once
	mu              sync.Mutex
	lastRun         map[string]int64
	incidentPending map[string]*incidentBucket
	aiCalls         []int64
	traces          []Trace

	signalsReceived    atomic.Uint64
	signalsDropped     atomic.Uint64
	shadowSignals      atomic.Uint64
	belowSeverity      atomic.Uint64
	suppressedCooldown atomic.Uint64
	eligibleSignals    atomic.Uint64
	aiTasksStarted     atomic.Uint64
	aiFallbacks        atomic.Uint64
	notifications      atomic.Uint64
	activeAI           atomic.Int64
	triageBatches      atomic.Uint64
	triageSignals      atomic.Uint64
	triageTasksStarted atomic.Uint64
	triageSuppressed   atomic.Uint64
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
	pipeline := &Pipeline{
		cfg: cfg, queue: make(chan signalservice.Signal, cfg.QueueSize), incidentQueue: make(chan []signalservice.Signal, cfg.QueueSize),
		lastRun: map[string]int64{}, incidentPending: map[string]*incidentBucket{},
	}
	if cfg.TraceStore != nil {
		pipeline.traceQueue = make(chan Trace, 2048)
	}
	return pipeline, nil
}
func (pipeline *Pipeline) Start(ctx context.Context) {
	if pipeline == nil {
		return
	}
	pipeline.startOnce.Do(func() {
		if pipeline.traceQueue != nil {
			go pipeline.tracePersistenceWorker(ctx)
		}
		for index := 0; index < pipeline.cfg.Workers; index++ {
			go pipeline.worker(ctx)
		}
		for index := 0; index < pipeline.cfg.Workers; index++ {
			go pipeline.incidentWorker(ctx)
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
		pipeline.belowSeverity.Add(1)
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
	pipeline.eligibleSignals.Add(1)
	if !settings.AIEnabled {
		pipeline.notifyFallback(value, &trace, "AI disabled")
		return
	}
	trace.Status = "triage_buffered"
	pipeline.addTrace(trace)
	pipeline.bufferIncident(value, settings.TriageWindow)
}
func (pipeline *Pipeline) bufferIncident(value signalservice.Signal, window time.Duration) {
	if window <= 0 {
		window = DefaultSettings().TriageWindow
	}
	key := strings.ToUpper(strings.TrimSpace(value.Symbol))
	pipeline.mu.Lock()
	bucket := pipeline.incidentPending[key]
	if bucket != nil {
		bucket.signals = append(bucket.signals, value)
		pipeline.mu.Unlock()
		return
	}
	bucket = &incidentBucket{signals: []signalservice.Signal{value}}
	pipeline.incidentPending[key] = bucket
	bucket.timer = time.AfterFunc(window, func() { pipeline.flushIncident(key) })
	pipeline.mu.Unlock()
}

func (pipeline *Pipeline) flushIncident(key string) {
	pipeline.mu.Lock()
	bucket := pipeline.incidentPending[key]
	delete(pipeline.incidentPending, key)
	pipeline.mu.Unlock()
	if bucket == nil || len(bucket.signals) == 0 {
		return
	}
	batch := append([]signalservice.Signal(nil), bucket.signals...)
	select {
	case pipeline.incidentQueue <- batch:
	default:
		for _, value := range batch {
			trace := newTrace(value, "triage_queue_full")
			pipeline.notifyFallback(value, &trace, "incident triage queue is full")
		}
	}
}

func (pipeline *Pipeline) incidentWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case batch := <-pipeline.incidentQueue:
			pipeline.processIncident(ctx, batch)
		}
	}
}

func (pipeline *Pipeline) processIncident(ctx context.Context, batch []signalservice.Signal) {
	if len(batch) == 0 {
		return
	}
	pipeline.triageBatches.Add(1)
	pipeline.triageSignals.Add(uint64(len(batch)))
	settings := pipeline.cfg.Settings()
	if !pipeline.reserveAIBudget(settings.MaxPerMinute) {
		pipeline.fallbackIncidentBatch(batch, "AI minute budget exceeded", "")
		return
	}
	if !pipeline.acquireAISlot(settings.MaxConcurrent) {
		pipeline.fallbackIncidentBatch(batch, "AI concurrency limit reached", "")
		return
	}
	defer pipeline.releaseAISlot()
	if len(batch) == 1 {
		trace := newTrace(batch[0], "ai_running")
		pipeline.runAI(ctx, batch[0], &trace)
		return
	}
	pipeline.runTriageAI(ctx, batch)
}

func (pipeline *Pipeline) runTriageAI(ctx context.Context, batch []signalservice.Signal) {
	sort.SliceStable(batch, func(i, j int) bool { return batch[i].CreatedAt < batch[j].CreatedAt })
	start, end := batch[0].CreatedAt, batch[len(batch)-1].CreatedAt
	signals := make([]workflowSkills.IncidentSignal, 0, len(batch))
	symbols := []string{batch[0].Symbol}
	for _, value := range batch {
		signals = append(signals, workflowSkills.IncidentSignal{
			SignalID: value.SignalID, Symbol: value.Symbol, Type: string(value.Type), Severity: string(value.Severity), CreatedAt: value.CreatedAt,
		})
	}
	input := workflowSkills.AlertTriageInput{
		Version: "alert_triage_input_v1", WindowStart: start, WindowEnd: end,
		Candidates: []workflowSkills.IncidentCandidate{{
			CandidateID: fmt.Sprintf("live_%s_%d", strings.ToLower(batch[0].Symbol), start),
			WindowStart: start, WindowEnd: end, Symbols: symbols, Signals: signals,
		}},
	}
	raw, err := json.Marshal(input)
	if err != nil {
		pipeline.fallbackIncidentBatch(batch, err.Error(), "")
		return
	}
	item, err := pipeline.cfg.StartTask(agentruntime.Request{Skill: workflowSkills.AlertTriageName, Input: string(raw), Metadata: map[string]any{"alert_incident": true}})
	if err != nil {
		pipeline.fallbackIncidentBatch(batch, err.Error(), "")
		return
	}
	pipeline.aiTasksStarted.Add(1)
	pipeline.triageTasksStarted.Add(1)
	for _, value := range batch {
		trace := newTrace(value, "triage_running")
		trace.TaskID = item.ID
		pipeline.addTrace(trace)
	}
	completed, err := pipeline.waitTask(ctx, item.ID)
	if err != nil {
		pipeline.fallbackIncidentBatch(batch, err.Error(), item.ID)
		return
	}
	if completed.Status != task.StatusSucceeded {
		reason := completed.Error
		if strings.TrimSpace(reason) == "" {
			reason = "alert triage task did not succeed"
		}
		pipeline.fallbackIncidentBatch(batch, reason, item.ID)
		return
	}
	var result workflowSkills.IncidentSetV1
	if err := json.Unmarshal(completed.Result, &result); err != nil {
		pipeline.fallbackIncidentBatch(batch, "decode alert triage result: "+err.Error(), item.ID)
		return
	}
	pipeline.applyTriageResult(batch, item.ID, result)
}

func (pipeline *Pipeline) fallbackIncidentBatch(batch []signalservice.Signal, reason, taskID string) {
	for _, value := range batch {
		trace := newTrace(value, "triage_fallback")
		trace.TaskID = taskID
		pipeline.notifyFallback(value, &trace, reason)
	}
}

func (pipeline *Pipeline) applyTriageResult(batch []signalservice.Signal, taskID string, result workflowSkills.IncidentSetV1) {
	byID := make(map[string]signalservice.Signal, len(batch))
	for _, value := range batch {
		byID[value.SignalID] = value
	}
	for _, incident := range result.Incidents {
		members := make([]signalservice.Signal, 0, len(incident.SignalIDs))
		for _, id := range incident.SignalIDs {
			if value, ok := byID[id]; ok {
				members = append(members, value)
			}
		}
		if len(members) == 0 {
			continue
		}
		if incident.Action != "notify" {
			if incident.Action == "suppress" {
				pipeline.triageSuppressed.Add(uint64(len(members)))
			}
			for _, value := range members {
				trace := newTrace(value, "triage_"+incident.Action)
				trace.TaskID = taskID
				trace.Action = incident.Action
				pipeline.addTrace(trace)
			}
			continue
		}
		pipeline.notifyIncident(members, taskID, incident)
	}
}

func (pipeline *Pipeline) notifyIncident(members []signalservice.Signal, taskID string, incident workflowSkills.Incident) {
	representative := members[0]
	eventTime := representative.CreatedAt
	confirmed := make([]string, 0, len(members))
	for _, value := range members {
		if signalservice.SeverityAtLeast(value.Severity, representative.Severity) {
			representative = value
		}
		if value.CreatedAt > 0 && (eventTime <= 0 || value.CreatedAt < eventTime) {
			eventTime = value.CreatedAt
		}
		confirmed = append(confirmed, fmt.Sprintf("%s (%s)", value.SignalID, value.Type))
	}
	severity := strings.TrimSpace(incident.Severity)
	if severity == "" {
		severity = string(representative.Severity)
	}
	params := notify.AgentAlertParams{
		Title: "notification.agent_alert_title", EventID: representative.EventID, SignalID: incident.IncidentID,
		TaskID: taskID, EventTime: eventTime, Symbol: representative.Symbol, SignalType: "incident", Severity: severity,
		Summary: incident.Summary, MarketContext: incident.Rationale, ConfirmedBy: confirmed,
		Risks: []string{fmt.Sprintf("%d related signals were aggregated into one incident", len(members))}, Source: "AI", Fallback: false,
	}
	id, err := pipeline.cfg.Notify(params)
	if err != nil {
		for _, value := range members {
			trace := newTrace(value, "notify_failed")
			trace.TaskID = taskID
			trace.Action = "notify"
			trace.Error = err.Error()
			pipeline.addTrace(trace)
		}
		return
	}
	pipeline.notifications.Add(1)
	for _, value := range members {
		trace := newTrace(value, "triage_notified")
		trace.TaskID = taskID
		trace.Action = "notify"
		trace.NotificationID = id
		pipeline.addTrace(trace)
	}
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
		Title: "notification.agent_alert_title", EventID: value.EventID, SignalID: value.SignalID,
		TaskID: trace.TaskID, EventTime: value.CreatedAt, Symbol: value.Symbol, SignalType: string(value.Type), Severity: string(result.Severity),
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
		contextText += lang.Lang("notification.separator") + fmt.Sprintf(lang.Lang("notification.fallback_reason"), strings.TrimSpace(reason))
	}
	return notify.AgentAlertParams{
		Title: "notification.rule_alert_title", EventID: value.EventID, SignalID: value.SignalID, TaskID: taskID, EventTime: value.CreatedAt,
		Symbol: value.Symbol, SignalType: string(value.Type), Severity: string(value.Severity),
		Summary: summary, MarketContext: contextText, ConfirmedBy: signalEvidence(value),
		Risks: []string{lang.Lang("notification.rule_unverified")}, Source: "RULE", Fallback: true,
	}
}
func fallbackSummary(value signalservice.Signal) (string, string) {
	switch value.Type {
	case signalservice.TypeFastMove:
		change := value.Metrics["change_percent"]
		price := value.Metrics["price"]
		direction := value.Labels["direction"]
		return fmt.Sprintf(lang.Lang("notification.fallback_summary.fast_move"), value.Symbol, directionText(direction), mathAbs(change), value.Window),
			fmt.Sprintf(lang.Lang("notification.fallback_summary.fast_move_context"), price, value.Metrics["threshold_percent"])
	case signalservice.TypeLiquidationSpike:
		side := value.Labels["liquidation_side"]
		return fmt.Sprintf(lang.Lang("notification.fallback_summary.liquidation"), value.Symbol, sideText(side)),
			fmt.Sprintf(lang.Lang("notification.fallback_summary.liquidation_context"),
				value.Metrics["aggregate_notional"], value.Metrics["order_count"], value.Metrics["threshold_notional"])
	default:
		signalType := lang.Lang("notification.signal_type." + string(value.Type))
		if signalType == "notification.signal_type."+string(value.Type) {
			signalType = string(value.Type)
		}
		return fmt.Sprintf(lang.Lang("notification.fallback_summary.generic"), value.Symbol, signalType),
			lang.Lang("notification.fallback_summary.generic_context")
	}
}

func signalEvidence(value signalservice.Signal) []string {
	switch value.Type {
	case signalservice.TypeFastMove:
		return []string{fmt.Sprintf(
			lang.Lang("notification.fallback_summary.fast_move_evidence"),
			value.Symbol,
			directionText(value.Labels["direction"]),
			mathAbs(value.Metrics["change_percent"]),
			value.Window,
		)}
	case signalservice.TypeLiquidationSpike:
		return []string{fmt.Sprintf(
			lang.Lang("notification.fallback_summary.liquidation_evidence"),
			value.Symbol,
			sideText(value.Labels["liquidation_side"]),
			value.Metrics["aggregate_notional"],
			value.Metrics["order_count"],
		)}
	}
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
		return lang.Lang("notification.direction.down")
	}
	return lang.Lang("notification.direction.up")
}

func sideText(value string) string {
	if value == "short" {
		return lang.Lang("notification.liquidation_side.short")
	}
	return lang.Lang("notification.liquidation_side.long")
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
	replaced := false
	for index := len(pipeline.traces) - 1; index >= 0; index-- {
		if pipeline.traces[index].SignalID == value.SignalID {
			pipeline.traces[index] = value
			replaced = true
			break
		}
	}
	if !replaced {
		pipeline.traces = append(pipeline.traces, value)
		if len(pipeline.traces) > 100 {
			pipeline.traces = append([]Trace(nil), pipeline.traces[len(pipeline.traces)-100:]...)
		}
	}
	pipeline.mu.Unlock()
	if pipeline.traceQueue != nil {
		pipeline.traceQueue <- value
	}
}

func (pipeline *Pipeline) tracePersistenceWorker(ctx context.Context) {
	for {
		select {
		case value := <-pipeline.traceQueue:
			pipeline.persistTrace(value)
		case <-ctx.Done():
			for {
				select {
				case value := <-pipeline.traceQueue:
					pipeline.persistTrace(value)
				default:
					return
				}
			}
		}
	}
}

func (pipeline *Pipeline) persistTrace(value Trace) {
	traceCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	err := pipeline.cfg.TraceStore.Save(traceCtx, value)
	cancel()
	if err != nil {
		logs.Error("persist alert pipeline trace:", err)
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
	eligible := pipeline.eligibleSignals.Load()
	aiTasks := pipeline.aiTasksStarted.Load()
	fallbacks := pipeline.aiFallbacks.Load()
	stats := Stats{
		SignalsReceived:    pipeline.signalsReceived.Load(),
		SignalsDropped:     pipeline.signalsDropped.Load(),
		ShadowSignals:      pipeline.shadowSignals.Load(),
		BelowSeverity:      pipeline.belowSeverity.Load(),
		SuppressedCooldown: pipeline.suppressedCooldown.Load(),
		EligibleSignals:    eligible,
		AITasksStarted:     aiTasks,
		AIFallbacks:        fallbacks,
		Notifications:      pipeline.notifications.Load(),
		ActiveAI:           int(pipeline.activeAI.Load()),
		TriageBatches:      pipeline.triageBatches.Load(),
		TriageSignals:      pipeline.triageSignals.Load(),
		TriageTasksStarted: pipeline.triageTasksStarted.Load(),
		TriageSuppressed:   pipeline.triageSuppressed.Load(),
	}
	if eligible > 0 {
		stats.SignalNotifyRate = float64(stats.Notifications) / float64(eligible)
	}
	if eligible > 0 {
		stats.AIFallbackRate = float64(fallbacks) / float64(eligible)
	}
	return stats
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

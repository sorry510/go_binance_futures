package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	agentruntime "go_binance_futures/agent/runtime"
	workflowSkill "go_binance_futures/agent/skills/workflows"
	"go_binance_futures/agent/task"
)

const (
	MarketScan         = "market_scan"
	StrategyReview     = "strategy_review"
	StrategyExperiment = "strategy_experiment"
	AlertTriage        = "alert_triage"
	DailyMarketBrief   = "daily_market_brief"
)

type Manager interface {
	Start(agentruntime.Request) (*task.Task, error)
	Get(context.Context, string) (*task.Task, error)
}

type Service struct {
	Manager      Manager
	Store        Store
	PollInterval time.Duration
	TaskTimeout  time.Duration
}

type marketScanRequest struct {
	Analyze int `json:"analyze"`
}
type strategyReviewRequest struct {
	TemplateID   int64  `json:"template_id"`
	TemplateName string `json:"template_name"`
	Days         int    `json:"days"`
}
type strategyExperimentRequest struct {
	TemplateID   int64  `json:"template_id"`
	TemplateName string `json:"template_name"`
	Goal         string `json:"goal"`
}
type alertTriageRequest struct {
	WindowMinutes int `json:"window_minutes"`
	MaxSignals    int `json:"max_signals"`
}
type dailyBriefRequest struct {
	WindowHours int `json:"window_hours"`
}

func (s Service) Start(ctx context.Context, name string, input json.RawMessage) (Run, error) {
	if s.Manager == nil {
		return Run{}, fmt.Errorf("workflow manager is required")
	}
	name = strings.TrimSpace(name)
	if !validWorkflow(name) {
		return Run{}, fmt.Errorf("unsupported workflow %q", name)
	}
	if len(input) == 0 || string(input) == "null" {
		input = json.RawMessage(`{}`)
	}
	if err := validateRequest(name, input); err != nil {
		return Run{}, err
	}
	now := time.Now().UTC().UnixMilli()
	run := Run{ID: newRunID(), Workflow: name, SchemaVersion: schemaVersion(name), Status: "queued", Stage: "queued", Input: append(json.RawMessage(nil), input...), ChildTaskIDs: []string{}, CreatedAt: now, UpdatedAt: now}
	if err := s.Store.Save(ctx, run); err != nil {
		return Run{}, err
	}
	go s.execute(context.Background(), run.ID)
	return run, nil
}
func (s Service) Get(ctx context.Context, id string) (Run, error) { return s.Store.Get(ctx, id) }
func (s Service) List(ctx context.Context, opt ListOptions) (ListResult, error) {
	return s.Store.List(ctx, opt)
}

func (s Service) execute(ctx context.Context, id string) {
	run, err := s.Store.Get(ctx, id)
	if err != nil {
		return
	}
	s.update(&run, "running", "preparing", "")
	switch run.Workflow {
	case MarketScan:
		err = s.executeMarketScan(ctx, &run)
	case StrategyReview:
		err = s.executeStrategyReview(ctx, &run)
	case StrategyExperiment:
		err = s.executeStrategyExperiment(ctx, &run)
	case AlertTriage:
		err = s.executeAlertTriage(ctx, &run)
	case DailyMarketBrief:
		err = s.executeDailyBrief(ctx, &run)
	}
	if err != nil {
		s.fail(&run, err)
	}
}
func (s Service) executeMarketScan(ctx context.Context, run *Run) error {
	var req marketScanRequest
	_ = json.Unmarshal(run.Input, &req)
	in, err := buildMarketScanInput(ctx, req.Analyze)
	if err != nil {
		return err
	}
	return s.runSingle(ctx, run, workflowSkill.MarketScanName, in)
}
func (s Service) executeStrategyReview(ctx context.Context, run *Run) error {
	var req strategyReviewRequest
	_ = json.Unmarshal(run.Input, &req)
	in, err := buildStrategyReviewInput(ctx, req.TemplateID, req.TemplateName, req.Days)
	if err != nil {
		return err
	}
	return s.runSingle(ctx, run, workflowSkill.StrategyReviewName, in)
}
func (s Service) executeAlertTriage(ctx context.Context, run *Run) error {
	var req alertTriageRequest
	_ = json.Unmarshal(run.Input, &req)
	in, err := buildAlertTriageInput(ctx, req.WindowMinutes, req.MaxSignals)
	if err != nil {
		return err
	}
	return s.runSingle(ctx, run, workflowSkill.AlertTriageName, in)
}
func (s Service) executeDailyBrief(ctx context.Context, run *Run) error {
	var req dailyBriefRequest
	_ = json.Unmarshal(run.Input, &req)
	in, err := buildDailyBriefInput(ctx, req.WindowHours)
	if err != nil {
		return err
	}
	return s.runSingle(ctx, run, workflowSkill.DailyMarketBriefName, in)
}
func (s Service) executeStrategyExperiment(ctx context.Context, run *Run) error {
	var req strategyExperimentRequest
	_ = json.Unmarshal(run.Input, &req)
	proposalInput, err := buildProposalInput(ctx, req.TemplateID, req.TemplateName, req.Goal)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(proposalInput)
	if err != nil {
		return err
	}
	proposalTask, err := s.startAndWait(ctx, run, workflowSkill.StrategyExperimentProposeName, raw, "proposing")
	if err != nil {
		return err
	}
	var proposal workflowSkill.StrategyExperimentProposalV1
	if err := json.Unmarshal(proposalTask.Result, &proposal); err != nil {
		return fmt.Errorf("decode proposal: %w", err)
	}
	s.update(run, "running", "deterministic_testing", "")
	test := testProposal(proposal)
	summaryInput := workflowSkill.StrategyExperimentSummaryInput{Version: "strategy_experiment_summary_input_v1", Proposal: proposal, Test: test}
	summaryRaw, _ := json.Marshal(summaryInput)
	summaryTask, err := s.startAndWait(ctx, run, workflowSkill.StrategyExperimentSummaryName, summaryRaw, "summarizing")
	if err != nil {
		return err
	}
	return s.complete(run, summaryTask.Result)
}
func (s Service) runSingle(ctx context.Context, run *Run, skillName string, input any) error {
	raw, err := json.Marshal(input)
	if err != nil {
		return err
	}
	item, err := s.startAndWait(ctx, run, skillName, raw, "analyzing")
	if err != nil {
		return err
	}
	return s.complete(run, item.Result)
}
func (s Service) startAndWait(ctx context.Context, run *Run, skillName string, input json.RawMessage, stage string) (*task.Task, error) {
	s.update(run, "running", stage, "")
	item, err := s.Manager.Start(agentruntime.Request{Skill: skillName, Input: string(input), Metadata: map[string]any{"workflow_run_id": run.ID, "workflow": run.Workflow}})
	if err != nil {
		return nil, err
	}
	run.ChildTaskIDs = append(run.ChildTaskIDs, item.ID)
	run.UpdatedAt = time.Now().UTC().UnixMilli()
	_ = s.Store.Save(context.Background(), *run)
	return s.waitTask(ctx, item.ID)
}
func (s Service) waitTask(ctx context.Context, id string) (*task.Task, error) {
	poll := s.PollInterval
	if poll <= 0 {
		poll = 250 * time.Millisecond
	}
	timeout := s.TaskTimeout
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	ticker := time.NewTicker(poll)
	defer ticker.Stop()
	for {
		item, err := s.Manager.Get(waitCtx, id)
		if err != nil {
			return nil, err
		}
		if task.IsTerminalStatus(item.Status) {
			if item.Status != task.StatusSucceeded {
				return nil, fmt.Errorf("child task %s failed: %s", id, firstNonEmpty(item.Error, string(item.Status)))
			}
			return item, nil
		}
		select {
		case <-waitCtx.Done():
			return nil, waitCtx.Err()
		case <-ticker.C:
		}
	}
}
func (s Service) complete(run *Run, result json.RawMessage) error {
	now := time.Now().UTC().UnixMilli()
	run.Status = "succeeded"
	run.Stage = "completed"
	run.Result = append(json.RawMessage(nil), result...)
	run.Error = ""
	run.UpdatedAt = now
	run.CompletedAt = now
	return s.Store.Save(context.Background(), *run)
}
func (s Service) fail(run *Run, err error) {
	now := time.Now().UTC().UnixMilli()
	run.Status = "failed"
	run.Stage = "failed"
	run.Error = err.Error()
	run.UpdatedAt = now
	run.CompletedAt = now
	_ = s.Store.Save(context.Background(), *run)
}
func (s Service) update(run *Run, status, stage, errorMessage string) {
	run.Status = status
	run.Stage = stage
	run.Error = errorMessage
	run.UpdatedAt = time.Now().UTC().UnixMilli()
	_ = s.Store.Save(context.Background(), *run)
}
func validWorkflow(v string) bool {
	switch v {
	case MarketScan, StrategyReview, StrategyExperiment, AlertTriage, DailyMarketBrief:
		return true
	}
	return false
}
func schemaVersion(v string) string {
	switch v {
	case MarketScan:
		return "opportunity_set_v1"
	case StrategyReview:
		return "strategy_review_v1"
	case StrategyExperiment:
		return "strategy_experiment_result_v1"
	case AlertTriage:
		return "incident_set_v1"
	case DailyMarketBrief:
		return "daily_market_brief_v1"
	}
	return ""
}
func validateRequest(name string, raw json.RawMessage) error {
	dec := func(target any) error { return json.Unmarshal(raw, target) }
	switch name {
	case MarketScan:
		var v marketScanRequest
		return dec(&v)
	case StrategyReview:
		var v strategyReviewRequest
		if err := dec(&v); err != nil {
			return err
		}
		if v.TemplateID <= 0 && strings.TrimSpace(v.TemplateName) == "" {
			return fmt.Errorf("template_id or template_name is required")
		}
		return nil
	case StrategyExperiment:
		var v strategyExperimentRequest
		if err := dec(&v); err != nil {
			return err
		}
		if v.TemplateID <= 0 && strings.TrimSpace(v.TemplateName) == "" {
			return fmt.Errorf("template_id or template_name is required")
		}
		if strings.TrimSpace(v.Goal) == "" {
			return fmt.Errorf("goal is required")
		}
		return nil
	case AlertTriage:
		var v alertTriageRequest
		return dec(&v)
	case DailyMarketBrief:
		var v dailyBriefRequest
		return dec(&v)
	}
	return fmt.Errorf("unsupported workflow")
}
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return "unknown error"
}

// BuildDailyMarketBriefInput is shared by the scheduler and manual workflow.
func BuildDailyMarketBriefInput(ctx context.Context, windowHours int) (string, error) {
	v, err := buildDailyBriefInput(ctx, windowHours)
	if err != nil {
		return "", err
	}
	raw, err := json.Marshal(v)
	return string(raw), err
}

// BuildMarketScanChatInput adapts free-form Chat text to the deterministic market-scan input contract.
func BuildMarketScanChatInput(ctx context.Context, prompt string) (string, error) {
	v, err := buildMarketScanInput(ctx, 8)
	if err != nil {
		return "", err
	}
	v.Prompt = strings.TrimSpace(prompt)
	raw, err := json.Marshal(v)
	return string(raw), err
}

// BuildStrategyReviewChatInput resolves a referenced strategy template when possible.
// If no template is named, the Skill still runs and returns an insufficient-data review instead of rejecting Chat text.
func BuildStrategyReviewChatInput(ctx context.Context, prompt string) (string, error) {
	prompt = strings.TrimSpace(prompt)
	id, name, found, err := resolveStrategyTemplateFromChat(ctx, prompt)
	if err != nil {
		return "", err
	}
	var v workflowSkill.StrategyReviewInput
	if found {
		v, err = buildStrategyReviewInput(ctx, id, name, 30)
		if err != nil {
			return "", err
		}
	} else {
		mc, _ := marketCondition(ctx)
		v = workflowSkill.StrategyReviewInput{
			Version: "strategy_review_input_v1", MarketCondition: mc,
			DataMissing: []string{"strategy template was not specified; ask for a template name or explicit template ID"},
		}
	}
	v.Prompt = prompt
	raw, err := json.Marshal(v)
	return string(raw), err
}

// BuildDailyMarketBriefChatInput adapts free-form Chat text to the deterministic daily-brief input contract.
func BuildDailyMarketBriefChatInput(ctx context.Context, prompt string) (string, error) {
	v, err := buildDailyBriefInput(ctx, 24)
	if err != nil {
		return "", err
	}
	v.Prompt = strings.TrimSpace(prompt)
	raw, err := json.Marshal(v)
	return string(raw), err
}

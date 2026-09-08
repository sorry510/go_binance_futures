package team

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	agentruntime "go_binance_futures/agent/runtime"
	"go_binance_futures/agent/skill"
	"go_binance_futures/agent/skills/symbolteam"
	"go_binance_futures/agent/task"
)

type Runner struct {
	cfg  Config
	mu   sync.Mutex
	runs map[string]*runState
}

type runState struct {
	cancel   context.CancelFunc
	children map[string]bool
}

type childSpec struct {
	role  string
	skill string
}

type childOutcome struct {
	spec childSpec
	task *task.Task
}

func New(cfg Config) (*Runner, error) {
	if cfg.Manager == nil || cfg.Store == nil || cfg.SharedContext == nil {
		return nil, fmt.Errorf("team runner requires manager, task store and shared context executor")
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 50 * time.Millisecond
	}
	if cfg.ChildTimeout <= 0 {
		cfg.ChildTimeout = 3 * time.Minute
	}
	if cfg.MaxConcurrency <= 0 {
		cfg.MaxConcurrency = 3
	}
	if cfg.MaxTotalTokens <= 0 {
		cfg.MaxTotalTokens = 120000
	}
	if cfg.MaxToolCalls <= 0 {
		cfg.MaxToolCalls = 1
	}
	return &Runner{cfg: cfg, runs: map[string]*runState{}}, nil
}

func (runner *Runner) Start(input Input) (*task.Task, error) {
	return runner.StartWithOptions(input, StartOptions{})
}

func (runner *Runner) StartWithOptions(input Input, options StartOptions) (*task.Task, error) {
	input.Symbol = strings.ToUpper(strings.TrimSpace(input.Symbol))
	input.Prompt = strings.TrimSpace(input.Prompt)
	if !strings.HasSuffix(input.Symbol, "USDT") || len(input.Symbol) <= 4 {
		return nil, fmt.Errorf("symbol must be a USDT futures contract")
	}
	rawInput, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	id := task.NewID()
	item := &task.Task{
		ID: id, Skill: SymbolAnalysisTeam, ConversationID: strings.TrimSpace(options.ConversationID), TeamRunID: id, TeamName: SymbolAnalysisTeam, TeamRole: "team",
		Status: task.StatusQueued, Stage: "team_queued", Progress: 0, Input: string(rawInput), ExecutionMode: "team",
		RuntimeVersion: agentruntime.CurrentVersion, SkillVersion: "1.1.0", PromptVersion: "1.0.0",
		InputContractVersion: "symbol_analysis_team_input_v1", OutputContractVersion: "symbol_analysis_team_v1",
		SkillSource: skill.DefaultSource, SkillSourceVersion: "v3-2", CreatedAt: now, UpdatedAt: now,
	}
	item.Events = append(item.Events, teamEvent(item, "team_queued", 0, "team run queued", "queued"))
	if creator, ok := runner.cfg.Store.(task.CreateStore); ok {
		if err := creator.Create(context.Background(), item); err != nil {
			return nil, err
		}
	} else if err := runner.cfg.Store.Save(context.Background(), item); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	runner.mu.Lock()
	runner.runs[id] = &runState{cancel: cancel, children: map[string]bool{}}
	runner.mu.Unlock()
	go runner.run(ctx, id, input)
	copyItem := *item
	copyItem.Events = append([]task.Event(nil), item.Events...)
	return &copyItem, nil
}

func (runner *Runner) Cancel(ctx context.Context, taskID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	taskID = strings.TrimSpace(taskID)
	runner.mu.Lock()
	state := runner.runs[taskID]
	if state == nil {
		runner.mu.Unlock()
		return fmt.Errorf("team run %q is not actively running", taskID)
	}
	children := make([]string, 0, len(state.children))
	for id := range state.children {
		children = append(children, id)
	}
	cancel := state.cancel
	runner.mu.Unlock()
	cancel()
	for _, id := range children {
		_ = runner.cfg.Manager.Cancel(context.Background(), id)
	}
	return nil
}

func (runner *Runner) run(ctx context.Context, taskID string, input Input) {
	defer runner.finishRun(taskID)
	if err := runner.updateParent(taskID, func(item *task.Task) {
		now := time.Now().UTC()
		item.Status = task.StatusRunning
		item.Stage = "team_shared_context"
		item.Progress = 10
		item.StartedAt = &now
		item.UpdatedAt = now
		item.Events = append(item.Events, teamEvent(item, item.Stage, item.Progress, "collect shared symbol context", "running"))
	}); err != nil {
		return
	}
	runner.observeParent(taskID, "task_started")

	shared, err := runner.cfg.SharedContext.Execute(ctx, taskID, input.Symbol, runner.cfg.MaxToolCalls)
	if err != nil {
		runner.failParent(taskID, "team_shared_context_failed", err)
		return
	}
	if err := ctx.Err(); err != nil {
		runner.cancelParent(taskID)
		return
	}
	_ = runner.updateParent(taskID, func(item *task.Task) {
		item.Stage, item.Progress, item.UpdatedAt = "team_children", 25, time.Now().UTC()
		message := fmt.Sprintf("shared context ready hash=%s partial=%t", shared.ContentHash, shared.Partial)
		item.Events = append(item.Events, teamEvent(item, "team_shared_context_ready", 25, message, "success"))
	})

	sharedInput := symbolteam.SharedInput{Symbol: input.Symbol, Prompt: input.Prompt, SharedContext: shared.Raw, SharedContextHash: shared.ContentHash}
	childRaw, _ := json.Marshal(sharedInput)
	specs := []childSpec{{role: symbolteam.RoleTechnical, skill: symbolteam.TechnicalSkillName}, {role: symbolteam.RoleFlow, skill: symbolteam.FlowSkillName}, {role: symbolteam.RoleNews, skill: symbolteam.NewsSkillName}}
	outcomes := runner.runChildren(ctx, taskID, string(childRaw), specs)
	if err := ctx.Err(); err != nil {
		runner.cancelParent(taskID)
		return
	}

	usage := task.Usage{}
	members := map[string]symbolteam.MemberInput{}
	children := make([]ChildTask, 0, len(outcomes)+1)
	for _, outcome := range outcomes {
		member, child := memberFromOutcome(outcome)
		members[outcome.spec.role] = member
		children = append(children, child)
		addUsage(&usage, child.Usage)
	}
	if usage.TotalTokens > runner.cfg.MaxTotalTokens {
		runner.failParent(taskID, "team_budget_exceeded", fmt.Errorf("team token budget exceeded before supervisor: used=%d limit=%d", usage.TotalTokens, runner.cfg.MaxTotalTokens))
		return
	}

	supervisorInput := symbolteam.SupervisorInput{Symbol: input.Symbol, Prompt: input.Prompt, Technical: members[symbolteam.RoleTechnical], Flow: members[symbolteam.RoleFlow], News: members[symbolteam.RoleNews]}
	supervisorRaw, _ := json.Marshal(supervisorInput)
	remaining := runner.cfg.MaxTotalTokens - usage.TotalTokens
	if remaining < 1 {
		remaining = 1
	}
	_ = runner.updateParent(taskID, func(item *task.Task) {
		item.Stage, item.Progress, item.UpdatedAt = "team_supervisor", 70, time.Now().UTC()
		item.Events = append(item.Events, teamEvent(item, item.Stage, item.Progress, "supervisor synthesis started", "running"))
	})
	started, err := runner.cfg.Manager.StartLinked(agentruntime.Request{Skill: symbolteam.SupervisorSkillName, Input: string(supervisorRaw)}, task.Linkage{
		ParentTaskID: taskID, TeamRunID: taskID, TeamName: SymbolAnalysisTeam, TeamRole: symbolteam.RoleSupervisor,
		MaxToolCalls: 1, MaxTotalTokens: remaining,
	})
	if err != nil {
		runner.failParent(taskID, "team_supervisor_failed", err)
		return
	}
	runner.trackChild(taskID, started.ID)
	supervisorTask := runner.waitChild(ctx, started.ID)
	if supervisorTask == nil || supervisorTask.Status != task.StatusSucceeded {
		if supervisorTask != nil {
			children = append(children, childTaskFromTask(symbolteam.RoleSupervisor, supervisorTask, []string{"supervisor_failed"}))
			if detail := strings.TrimSpace(supervisorTask.Error); detail != "" {
				runner.failParent(taskID, "team_supervisor_failed", fmt.Errorf("supervisor task failed: %s", detail))
				return
			}
			runner.failParent(taskID, "team_supervisor_failed", fmt.Errorf("supervisor task ended with status=%s stage=%s", supervisorTask.Status, supervisorTask.Stage))
			return
		}
		runner.failParent(taskID, "team_supervisor_failed", fmt.Errorf("supervisor task result is unavailable"))
		return
	}
	addUsage(&usage, supervisorTask.Usage)
	children = append(children, childTaskFromTask(symbolteam.RoleSupervisor, supervisorTask, nil))
	if usage.TotalTokens > runner.cfg.MaxTotalTokens {
		runner.failParent(taskID, "team_budget_exceeded", fmt.Errorf("team token budget exceeded: used=%d limit=%d", usage.TotalTokens, runner.cfg.MaxTotalTokens))
		return
	}

	var supervisor symbolteam.SupervisorResultV1
	if err := json.Unmarshal(supervisorTask.Result, &supervisor); err != nil {
		runner.failParent(taskID, "team_supervisor_decode_failed", err)
		return
	}
	partial := shared.Partial
	for _, child := range children {
		if child.Status != string(task.StatusSucceeded) || len(child.DataMissing) > 0 {
			partial = true
		}
	}
	if len(supervisor.DataMissing) > 0 {
		partial = true
	}
	result := ResultV1{
		Version: "symbol_analysis_team_v1", Team: SymbolAnalysisTeam, Symbol: input.Symbol,
		Status: ternary(partial, "partial", "succeeded"), Direction: supervisor.Direction, Confidence: supervisor.Confidence,
		Summary: supervisor.Summary, Consensus: nonNil(supervisor.Consensus), Disagreements: nonNil(supervisor.Disagreements),
		DataMissing: uniqueStrings(supervisor.DataMissing), Evidence: convertEvidence(supervisor.Evidence),
	}
	rawResult, _ := json.Marshal(result)
	_ = runner.updateParent(taskID, func(item *task.Task) {
		now := time.Now().UTC()
		item.Status = task.StatusSucceeded
		item.Stage = ternary(partial, "team_completed_partial", "team_completed")
		item.Progress = 100
		item.Result = rawResult
		item.Usage = usage
		item.Provider = supervisorTask.Provider
		item.Model = supervisorTask.Model
		item.FinalModelConfigID = supervisorTask.FinalModelConfigID
		item.CompletedAt = &now
		item.UpdatedAt = now
		item.Steps = buildTeamSteps(shared, children)
		item.Events = append(item.Events, teamEvent(item, item.Stage, 100, result.Status, "success"))
	})
	runner.observeParent(taskID, "task_finished")
	runner.notifyCompletion(taskID)
}

func (runner *Runner) runChildren(ctx context.Context, parentID, input string, specs []childSpec) []childOutcome {
	sem := make(chan struct{}, runner.cfg.MaxConcurrency)
	out := make(chan childOutcome, len(specs))
	var wg sync.WaitGroup
	perChildTokens := runner.cfg.MaxTotalTokens / maxInt(len(specs)+1, 1)
	for _, spec := range specs {
		spec := spec
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				out <- childOutcome{spec: spec}
				return
			}
			defer func() { <-sem }()
			started, err := runner.cfg.Manager.StartLinked(agentruntime.Request{Skill: spec.skill, Input: input}, task.Linkage{
				ParentTaskID: parentID, TeamRunID: parentID, TeamName: SymbolAnalysisTeam, TeamRole: spec.role,
				MaxToolCalls: 1, MaxTotalTokens: perChildTokens,
			})
			if err != nil {
				out <- childOutcome{spec: spec, task: syntheticFailedTask(spec.skill, err)}
				return
			}
			runner.trackChild(parentID, started.ID)
			out <- childOutcome{spec: spec, task: runner.waitChild(ctx, started.ID)}
		}()
	}
	wg.Wait()
	close(out)
	byRole := map[string]childOutcome{}
	for item := range out {
		byRole[item.spec.role] = item
	}
	result := make([]childOutcome, 0, len(specs))
	for _, spec := range specs {
		result = append(result, byRole[spec.role])
	}
	return result
}

func (runner *Runner) waitChild(ctx context.Context, taskID string) *task.Task {
	deadline := time.NewTimer(runner.cfg.ChildTimeout)
	defer deadline.Stop()
	ticker := time.NewTicker(runner.cfg.PollInterval)
	defer ticker.Stop()
	for {
		item, err := runner.cfg.Manager.Get(context.Background(), taskID)
		if err == nil && task.IsTerminalStatus(item.Status) {
			return item
		}
		select {
		case <-ctx.Done():
			_ = runner.cfg.Manager.Cancel(context.Background(), taskID)
			return item
		case <-deadline.C:
			_ = runner.cfg.Manager.Cancel(context.Background(), taskID)
			item, _ = runner.cfg.Manager.Get(context.Background(), taskID)
			return item
		case <-ticker.C:
		}
	}
}

func (runner *Runner) trackChild(parentID, childID string) {
	runner.mu.Lock()
	defer runner.mu.Unlock()
	if state := runner.runs[parentID]; state != nil {
		state.children[childID] = true
	}
}
func (runner *Runner) finishRun(taskID string) {
	runner.mu.Lock()
	if state := runner.runs[taskID]; state != nil {
		state.cancel()
	}
	delete(runner.runs, taskID)
	runner.mu.Unlock()
}
func (runner *Runner) updateParent(taskID string, mutate func(*task.Task)) error {
	item, err := runner.cfg.Store.Get(context.Background(), taskID)
	if err != nil {
		return err
	}
	mutate(item)
	return runner.cfg.Store.Save(context.Background(), item)
}
func (runner *Runner) failParent(taskID, stage string, err error) {
	_ = runner.updateParent(taskID, func(item *task.Task) {
		now := time.Now().UTC()
		item.Status = task.StatusFailed
		item.Stage = stage
		item.Error = err.Error()
		item.UpdatedAt = now
		item.CompletedAt = &now
		item.Events = append(item.Events, teamEvent(item, stage, item.Progress, err.Error(), "failed"))
	})
	runner.observeParent(taskID, "task_finished")
	runner.notifyCompletion(taskID)
}
func (runner *Runner) cancelParent(taskID string) {
	_ = runner.updateParent(taskID, func(item *task.Task) {
		now := time.Now().UTC()
		item.Status = task.StatusCancelled
		item.Stage = "team_cancelled"
		item.Error = "team run cancelled"
		item.UpdatedAt = now
		item.CompletedAt = &now
		item.Events = append(item.Events, teamEvent(item, item.Stage, item.Progress, item.Error, "cancelled"))
	})
	runner.observeParent(taskID, "task_finished")
	runner.notifyCompletion(taskID)
}

func (runner *Runner) notifyCompletion(taskID string) {
	if runner.cfg.CompletionHook == nil {
		return
	}
	item, err := runner.cfg.Store.Get(context.Background(), taskID)
	if err == nil {
		runner.cfg.CompletionHook(item)
	}
}

func (runner *Runner) observeParent(taskID, observationType string) {
	if runner.cfg.Observer == nil {
		return
	}
	item, err := runner.cfg.Store.Get(context.Background(), taskID)
	if err != nil {
		return
	}
	duration := int64(0)
	if item.StartedAt != nil {
		end := time.Now().UTC()
		if item.CompletedAt != nil {
			end = *item.CompletedAt
		}
		duration = end.Sub(*item.StartedAt).Milliseconds()
	}
	runner.cfg.Observer.Observe(agentruntime.Observation{
		Type: observationType, TaskID: item.ID, Skill: item.Skill, Status: string(item.Status),
		DurationMs: duration, Usage: item.Usage, Error: item.Error,
	})
}
func teamEvent(item *task.Task, stage string, progress int, message, status string) task.Event {
	return task.Event{TaskID: item.ID, Skill: item.Skill, Stage: stage, Progress: progress, Message: message, Status: status, Time: time.Now().UTC()}
}

func memberFromOutcome(outcome childOutcome) (symbolteam.MemberInput, ChildTask) {
	if outcome.task == nil {
		missing := []string{outcome.spec.role + "_missing"}
		return symbolteam.MemberInput{Role: outcome.spec.role, Status: "failed", DataMissing: missing}, ChildTask{Role: outcome.spec.role, Skill: outcome.spec.skill, Status: "failed", DataMissing: missing, Error: "child task missing"}
	}
	missing := childDataMissing(outcome.spec.role, outcome.task)
	member := symbolteam.MemberInput{Role: outcome.spec.role, TaskID: outcome.task.ID, Status: string(outcome.task.Status), DataMissing: missing}
	if outcome.task.Status == task.StatusSucceeded {
		member.Result = append(json.RawMessage(nil), outcome.task.Result...)
	}
	return member, childTaskFromTask(outcome.spec.role, outcome.task, missing)
}
func childDataMissing(role string, item *task.Task) []string {
	if item == nil || item.Status != task.StatusSucceeded {
		return []string{role + "_failed"}
	}
	var holder struct {
		DataMissing []string `json:"data_missing"`
	}
	if json.Unmarshal(item.Result, &holder) != nil {
		return []string{role + "_result_invalid"}
	}
	return uniqueStrings(holder.DataMissing)
}
func childTaskFromTask(role string, item *task.Task, missing []string) ChildTask {
	if item == nil {
		return ChildTask{Role: role, Status: "failed", DataMissing: missing}
	}
	duration := int64(0)
	if item.StartedAt != nil && item.CompletedAt != nil {
		duration = item.CompletedAt.Sub(*item.StartedAt).Milliseconds()
	}
	return ChildTask{TaskID: item.ID, Role: role, Skill: item.Skill, Status: string(item.Status), DurationMs: duration, Usage: item.Usage, DataMissing: uniqueStrings(missing), Error: item.Error}
}
func syntheticFailedTask(skillName string, err error) *task.Task {
	return &task.Task{Skill: skillName, Status: task.StatusFailed, Error: err.Error()}
}
func addUsage(total *task.Usage, value task.Usage) {
	total.InputTokens += value.InputTokens
	total.OutputTokens += value.OutputTokens
	total.TotalTokens += value.TotalTokens
}
func convertEvidence(items []symbolteam.SupervisorEvidence) []Evidence {
	out := make([]Evidence, 0, len(items))
	for _, item := range items {
		out = append(out, Evidence{Role: item.Role, Source: item.Source, Finding: item.Finding})
	}
	return out
}
func buildTeamSteps(shared SharedContextResult, children []ChildTask) json.RawMessage {
	steps := []map[string]any{{"step_id": "team-001", "type": "tool", "status": "succeeded", "attempt": 1, "output_summary": fmt.Sprintf("shared context hash=%s partial=%t", shared.ContentHash, shared.Partial)}}
	for i, child := range children {
		steps = append(steps, map[string]any{"step_id": fmt.Sprintf("team-%03d", i+2), "type": "agent", "status": child.Status, "attempt": 1, "output_summary": child.Role + " task=" + child.TaskID})
	}
	raw, _ := json.Marshal(steps)
	return raw
}
func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" && !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	sort.Strings(out)
	return out
}
func nonNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
func ternary[T any](condition bool, yes, no T) T {
	if condition {
		return yes
	}
	return no
}
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

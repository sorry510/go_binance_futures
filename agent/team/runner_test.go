package team

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go_binance_futures/agent/replay"
	agentruntime "go_binance_futures/agent/runtime"
	"go_binance_futures/agent/skills/symbolanalysis"
	"go_binance_futures/agent/skills/symbolteam"
	"go_binance_futures/agent/task"
)

type fixtureSharedContext struct{ calls atomic.Int32 }

func (fixture *fixtureSharedContext) Execute(context.Context, string, string, int) (SharedContextResult, error) {
	fixture.calls.Add(1)
	return SharedContextResult{Raw: json.RawMessage(`{"symbol":"BTCUSDT","as_of":"2026-09-06T10:00:00Z","market_condition":2}`), ContentHash: "fixture-hash"}, nil
}

type fixtureManager struct {
	store    *task.MemoryStore
	mu       sync.Mutex
	sequence int
	failRole string
	started  []task.Linkage
}

func (manager *fixtureManager) StartLinked(req agentruntime.Request, linkage task.Linkage) (*task.Task, error) {
	manager.mu.Lock()
	manager.sequence++
	id := fmt.Sprintf("child_%d", manager.sequence)
	manager.started = append(manager.started, linkage)
	manager.mu.Unlock()
	now := time.Now().UTC()
	item := &task.Task{ID: id, Skill: req.Skill, ParentTaskID: linkage.ParentTaskID, TeamRunID: linkage.TeamRunID, TeamName: linkage.TeamName, TeamRole: linkage.TeamRole, Status: task.StatusQueued, Stage: "queued", Input: req.Input, CreatedAt: now, UpdatedAt: now}
	if err := manager.store.Create(context.Background(), item); err != nil {
		return nil, err
	}
	go manager.complete(id, req, linkage)
	copyItem := *item
	return &copyItem, nil
}

func (manager *fixtureManager) complete(id string, req agentruntime.Request, linkage task.Linkage) {
	time.Sleep(time.Millisecond)
	item, _ := manager.store.Get(context.Background(), id)
	now := time.Now().UTC()
	item.StartedAt = &now
	item.Status = task.StatusRunning
	item.Stage = "running"
	_ = manager.store.Save(context.Background(), item)
	if linkage.TeamRole == manager.failRole {
		done := time.Now().UTC()
		item.Status = task.StatusFailed
		item.Stage = "failed"
		item.Error = "fixture failure"
		item.CompletedAt = &done
		item.UpdatedAt = done
		_ = manager.store.Save(context.Background(), item)
		return
	}
	item.Result = fixtureResult(req.Skill, req.Input)
	item.Usage = task.Usage{InputTokens: 40, OutputTokens: 10, TotalTokens: 50}
	done := time.Now().UTC()
	item.Status = task.StatusSucceeded
	item.Stage = "completed"
	item.Progress = 100
	item.CompletedAt = &done
	item.UpdatedAt = done
	_ = manager.store.Save(context.Background(), item)
}

func fixtureResult(skillName, input string) json.RawMessage {
	switch skillName {
	case symbolteam.TechnicalSkillName:
		return json.RawMessage(`{"version":"technical_analysis_v1","symbol":"BTCUSDT","as_of":"2026-09-06T10:00:00Z","trend":"bullish","structure":"higher lows","volatility":"normal","key_levels":[{"kind":"support","price":100}],"confidence":0.8,"data_missing":[],"evidence":[{"source":"get_symbol_analysis_context","finding":"1h structure bullish"}]}`)
	case symbolteam.FlowSkillName:
		return json.RawMessage(`{"version":"flow_analysis_v1","symbol":"BTCUSDT","as_of":"2026-09-06T10:00:00Z","bias":"bullish","funding":"neutral","open_interest":"rising","taker":"buyers lead","depth":"balanced","liquidation":"short liquidations","confidence":0.7,"data_missing":[],"evidence":[{"source":"get_symbol_analysis_context","finding":"OI and taker support"}]}`)
	default:
		var supervisor symbolteam.SupervisorInput
		_ = json.Unmarshal([]byte(input), &supervisor)
		missing := []string{}
		evidence := `[ {"role":"technical_analyst","source":"get_symbol_analysis_context","finding":"technical child bullish"}`
		if supervisor.Flow.Status == string(task.StatusSucceeded) {
			evidence += `,{"role":"flow_analyst","source":"get_symbol_analysis_context","finding":"flow child bullish"}`
		} else {
			missing = append(missing, "flow_analyst_failed")
		}
		evidence += `]`
		raw, _ := json.Marshal(map[string]any{"version": "symbol_team_supervisor_v1", "symbol": "BTCUSDT", "as_of": "2026-09-06T10:00:00Z", "direction": "long", "confidence": 0.76, "summary": "typed team consensus", "consensus": []string{"technical and flow align"}, "disagreements": []string{}, "data_missing": missing})
		var base map[string]any
		_ = json.Unmarshal(raw, &base)
		var ev any
		_ = json.Unmarshal([]byte(evidence), &ev)
		base["evidence"] = ev
		out, _ := json.Marshal(base)
		return out
	}
}

func (manager *fixtureManager) Get(ctx context.Context, id string) (*task.Task, error) {
	return manager.store.Get(ctx, id)
}
func (manager *fixtureManager) Cancel(ctx context.Context, id string) error {
	item, err := manager.store.Get(ctx, id)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	item.Status = task.StatusCancelled
	item.Stage = "cancelled"
	item.CompletedAt = &now
	item.UpdatedAt = now
	return manager.store.Save(ctx, item)
}

func newFixtureRunner(t *testing.T, failRole string) (*Runner, *task.MemoryStore, *fixtureManager, *fixtureSharedContext) {
	t.Helper()
	store := task.NewMemoryStore()
	manager := &fixtureManager{store: store, failRole: failRole}
	shared := &fixtureSharedContext{}
	runner, err := New(Config{Manager: manager, Store: store, SharedContext: shared, PollInterval: time.Millisecond, ChildTimeout: time.Second, MaxConcurrency: 2, MaxTotalTokens: 1000, MaxToolCalls: 1})
	if err != nil {
		t.Fatal(err)
	}
	return runner, store, manager, shared
}

func waitTeam(t *testing.T, store task.Store, id string) *task.Task {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		item, err := store.Get(context.Background(), id)
		if err == nil && task.IsTerminalStatus(item.Status) {
			return item
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("team run did not complete")
	return nil
}

func TestSymbolAnalysisTeamRunsSharedContextOnceAndLinksChildren(t *testing.T) {
	runner, store, manager, shared := newFixtureRunner(t, "")
	started, err := runner.Start(Input{Symbol: "BTCUSDT", Prompt: "focus on breakout"})
	if err != nil {
		t.Fatal(err)
	}
	finished := waitTeam(t, store, started.ID)
	if finished.Status != task.StatusSucceeded || finished.Stage != "team_completed" {
		t.Fatalf("unexpected team status: %+v", finished)
	}
	if shared.calls.Load() != 1 {
		t.Fatalf("shared context calls=%d want=1", shared.calls.Load())
	}
	children, err := store.List(context.Background(), task.ListOptions{ParentTaskID: started.ID, Page: 1, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if children.Total != 3 {
		t.Fatalf("child tasks=%d want=3", children.Total)
	}
	manager.mu.Lock()
	links := append([]task.Linkage(nil), manager.started...)
	manager.mu.Unlock()
	if len(links) != 3 || links[0].TeamRunID != started.ID {
		t.Fatalf("unexpected team linkages: %+v", links)
	}
	var result ResultV1
	if err := json.Unmarshal(finished.Result, &result); err != nil {
		t.Fatal(err)
	}
	if result.Status != "succeeded" || len(result.Evidence) != 2 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestChildFailureProducesPartialResultWithoutLosingOtherChild(t *testing.T) {
	runner, store, _, _ := newFixtureRunner(t, symbolteam.RoleFlow)
	started, err := runner.Start(Input{Symbol: "BTCUSDT"})
	if err != nil {
		t.Fatal(err)
	}
	finished := waitTeam(t, store, started.ID)
	if finished.Status != task.StatusSucceeded || finished.Stage != "team_completed_partial" {
		t.Fatalf("unexpected partial status: %+v", finished)
	}
	var result ResultV1
	if err := json.Unmarshal(finished.Result, &result); err != nil {
		t.Fatal(err)
	}
	if result.Status != "partial" {
		t.Fatalf("result status=%s", result.Status)
	}
	foundTechnical, foundFailedFlow := false, false
	children, err := store.List(context.Background(), task.ListOptions{ParentTaskID: started.ID, Page: 1, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	for _, child := range children.List {
		if child.TeamRole == symbolteam.RoleTechnical && child.Status == task.StatusSucceeded {
			foundTechnical = true
		}
		if child.TeamRole == symbolteam.RoleFlow && child.Status == task.StatusFailed {
			foundFailedFlow = true
		}
	}
	if !foundTechnical || !foundFailedFlow {
		t.Fatalf("partial result lost child state: %+v", children.List)
	}
}

func TestTeamFixtureReplayIsDeterministic(t *testing.T) {
	run := func() json.RawMessage {
		runner, store, _, _ := newFixtureRunner(t, "")
		started, err := runner.Start(Input{Symbol: "BTCUSDT"})
		if err != nil {
			t.Fatal(err)
		}
		return waitTeam(t, store, started.ID).Result
	}
	first, second := run(), run()
	if string(first) != string(second) {
		t.Fatalf("team replay changed:\n%s\n%s", first, second)
	}
}

func TestTeamTokenBudgetStopsBeforeSupervisor(t *testing.T) {
	store := task.NewMemoryStore()
	manager := &fixtureManager{store: store}
	shared := &fixtureSharedContext{}
	runner, err := New(Config{
		Manager: manager, Store: store, SharedContext: shared,
		PollInterval: time.Millisecond, ChildTimeout: time.Second,
		MaxConcurrency: 2, MaxTotalTokens: 80, MaxToolCalls: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	started, err := runner.Start(Input{Symbol: "BTCUSDT"})
	if err != nil {
		t.Fatal(err)
	}
	finished := waitTeam(t, store, started.ID)
	if finished.Status != task.StatusFailed || finished.Stage != "team_budget_exceeded" {
		t.Fatalf("expected budget failure before supervisor: %+v", finished)
	}
	manager.mu.Lock()
	links := append([]task.Linkage(nil), manager.started...)
	manager.mu.Unlock()
	if len(links) != 2 {
		t.Fatalf("supervisor started after team budget was exhausted: %+v", links)
	}
}

func TestSymbolAnalysisTeamFixedFixtureStabilityMatchesSingleAgentBaseline(t *testing.T) {
	const runs = 20
	teamSucceeded := 0
	for index := 0; index < runs; index++ {
		runner, store, _, _ := newFixtureRunner(t, "")
		started, err := runner.Start(Input{Symbol: "BTCUSDT"})
		if err != nil {
			t.Fatal(err)
		}
		if finished := waitTeam(t, store, started.ID); finished.Status == task.StatusSucceeded {
			teamSucceeded++
		}
	}

	fixture, err := replay.Load("../replay/testdata/symbol_analysis_success.json")
	if err != nil {
		t.Fatal(err)
	}
	singleSucceeded := 0
	for index := 0; index < runs; index++ {
		outcome := replay.Run(context.Background(), fixture, symbolanalysis.New())
		if outcome.Err == nil && outcome.Task != nil && outcome.Task.Status == task.StatusSucceeded {
			singleSucceeded++
		}
	}
	if teamSucceeded != runs || singleSucceeded != runs {
		t.Fatalf("fixed-fixture stability changed: team=%d/%d single=%d/%d", teamSucceeded, runs, singleSucceeded, runs)
	}
}

func TestTeamStartWithOptionsPersistsConversationAndCallsCompletionHook(t *testing.T) {
	store := task.NewMemoryStore()
	manager := &fixtureManager{store: store}
	shared := &fixtureSharedContext{}
	completed := make(chan *task.Task, 1)
	runner, err := New(Config{
		Manager: manager, Store: store, SharedContext: shared,
		PollInterval: time.Millisecond, ChildTimeout: time.Second,
		MaxConcurrency: 2, MaxTotalTokens: 1000, MaxToolCalls: 1,
		CompletionHook: func(item *task.Task) { completed <- item },
	})
	if err != nil {
		t.Fatal(err)
	}
	started, err := runner.StartWithOptions(Input{Symbol: "BTCUSDT"}, StartOptions{ConversationID: "conv-team"})
	if err != nil {
		t.Fatal(err)
	}
	finished := waitTeam(t, store, started.ID)
	if finished.ConversationID != "conv-team" {
		t.Fatalf("conversation_id=%q", finished.ConversationID)
	}
	select {
	case item := <-completed:
		if item.ID != started.ID || item.ConversationID != "conv-team" || item.Status != task.StatusSucceeded {
			t.Fatalf("unexpected completion item: %+v", item)
		}
	case <-time.After(time.Second):
		t.Fatal("team completion hook was not called")
	}
}

func TestSupervisorFailurePropagatesConcreteChildErrorToParent(t *testing.T) {
	runner, store, _, _ := newFixtureRunner(t, symbolteam.RoleSupervisor)
	started, err := runner.Start(Input{Symbol: "BTCUSDT"})
	if err != nil {
		t.Fatal(err)
	}
	finished := waitTeam(t, store, started.ID)
	if finished.Status != task.StatusFailed || finished.Stage != "team_supervisor_failed" {
		t.Fatalf("unexpected supervisor failure state: %+v", finished)
	}
	if !strings.Contains(finished.Error, "supervisor task failed: fixture failure") {
		t.Fatalf("parent did not preserve supervisor error: %q", finished.Error)
	}
}

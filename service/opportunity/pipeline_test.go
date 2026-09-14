package opportunity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	agentruntime "go_binance_futures/agent/runtime"
	"go_binance_futures/agent/task"
)

func testTradingPlan(symbol, direction string, confidence float64, missing []string) json.RawMessage {
	marketCondition := 3
	payload := map[string]any{
		"version": "trading_plan_v1", "symbol": symbol, "as_of": "2026-09-13T12:00:00Z",
		"market_condition": marketCondition, "direction": direction, "confidence": confidence,
		"summary": "opportunity analysis", "entry_zones": []any{}, "stop_loss": nil,
		"take_profits": []float64{}, "long_trigger": "", "short_trigger": "",
		"invalidation_conditions": []string{"context changes"}, "risks": []string{"volatility"},
		"data_missing": missing,
		"evidence":     []map[string]string{{"source": "get_symbol_analysis_context", "finding": "test evidence"}},
	}
	if direction == DirectionLong {
		payload["entry_zones"] = []map[string]float64{{"low": 99, "high": 100}}
		payload["stop_loss"] = 95.0
		payload["take_profits"] = []float64{110}
		payload["long_trigger"] = "price confirms"
	}
	if direction == DirectionShort {
		payload["entry_zones"] = []map[string]float64{{"low": 100, "high": 101}}
		payload["stop_loss"] = 105.0
		payload["take_profits"] = []float64{90}
		payload["short_trigger"] = "price rejects"
	}
	raw, _ := json.Marshal(payload)
	return raw
}

func testPipeline(t *testing.T, result json.RawMessage, taskStatus task.Status, taskError string, starts *atomic.Int64) (*Pipeline, Service) {
	t.Helper()
	service := setupOpportunityService(t)
	pipeline, err := NewPipeline(PipelineConfig{
		Service: service, Cooldown: 30 * time.Minute, TTL: time.Hour,
		PollInterval: time.Millisecond, TaskTimeout: time.Second,
		StartTask: func(request agentruntime.Request) (*task.Task, error) {
			starts.Add(1)
			if request.Skill != "symbol_analysis" {
				t.Fatalf("unexpected skill %s", request.Skill)
			}
			return &task.Task{ID: "task-opportunity", Status: task.StatusQueued}, nil
		},
		GetTask: func(context.Context, string) (*task.Task, error) {
			return &task.Task{ID: "task-opportunity", Skill: "symbol_analysis", Status: taskStatus, Result: result, Error: taskError}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return pipeline, service
}

func TestPipelineCreatesSucceededDirectionalOpportunity(t *testing.T) {
	var starts atomic.Int64
	pipeline, service := testPipeline(t, testTradingPlan("BTCUSDT", DirectionLong, 0.84, []string{}), task.StatusSucceeded, "", &starts)
	result, err := pipeline.Process(context.Background(), Trigger{Symbol: "btcusdt", SourceType: "fast_move", SourceID: "sig-success", Prompt: "analyze fast move"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Created || result.Suppressed || starts.Load() != 1 {
		t.Fatalf("unexpected pipeline result: %+v starts=%d", result, starts.Load())
	}
	row := result.Opportunity
	if row.AnalysisStatus != AnalysisSucceeded || row.Direction != DirectionLong || row.AnalysisTaskID != "task-opportunity" || row.MarketCondition != 3 {
		t.Fatalf("unexpected opportunity: %+v", row)
	}
	if !CanCreateProposal(row, service.now()) {
		t.Fatal("complete directional analysis must be proposal-eligible")
	}
	stats := pipeline.Stats()
	if stats.TasksStarted != 1 || stats.Succeeded != 1 || stats.Failed != 0 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

func TestPipelineDataMissingAndNeutralAreNotProposalEligible(t *testing.T) {
	for _, tc := range []struct {
		name      string
		direction string
		missing   []string
		want      string
	}{
		{name: "data_missing", direction: DirectionLong, missing: []string{"liquidations"}, want: AnalysisDataMissing},
		{name: "neutral", direction: DirectionNeutral, missing: []string{}, want: AnalysisSucceeded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var starts atomic.Int64
			pipeline, service := testPipeline(t, testTradingPlan("ETHUSDT", tc.direction, 0.72, tc.missing), task.StatusSucceeded, "", &starts)
			result, err := pipeline.Process(context.Background(), Trigger{Symbol: "ETHUSDT", SourceType: "market_event", SourceID: "evt-" + tc.name})
			if err != nil {
				t.Fatal(err)
			}
			if result.Opportunity.AnalysisStatus != tc.want {
				t.Fatalf("analysis status=%s want=%s", result.Opportunity.AnalysisStatus, tc.want)
			}
			if CanCreateProposal(result.Opportunity, service.now()) {
				t.Fatalf("%s opportunity must not be proposal-eligible", tc.name)
			}
		})
	}
}

func TestPipelinePersistsFailedAnalysis(t *testing.T) {
	var starts atomic.Int64
	pipeline, service := testPipeline(t, nil, task.StatusFailed, "model unavailable", &starts)
	_, err := pipeline.Process(context.Background(), Trigger{Symbol: "SOLUSDT", SourceType: "fast_move", SourceID: "sig-failed"})
	if err == nil {
		t.Fatal("failed task must return an error to the worker")
	}
	row, readErr := service.Store.FindBySource(context.Background(), "fast_move", "sig-failed")
	if readErr != nil {
		t.Fatal(readErr)
	}
	if row.AnalysisStatus != AnalysisFailed || row.AnalysisError != "model unavailable" || row.Direction != DirectionNeutral {
		t.Fatalf("failed analysis not persisted correctly: %+v", row)
	}
	if CanCreateProposal(row, service.now()) {
		t.Fatal("failed opportunity must not be proposal-eligible")
	}
}

func TestPipelineSuppressesReplayAndCooldownWithoutStartingAnotherTask(t *testing.T) {
	var starts atomic.Int64
	pipeline, _ := testPipeline(t, testTradingPlan("XRPUSDT", DirectionShort, 0.8, []string{}), task.StatusSucceeded, "", &starts)
	ctx := context.Background()
	first, err := pipeline.Process(ctx, Trigger{Symbol: "XRPUSDT", SourceType: "fast_move", SourceID: "sig-1"})
	if err != nil {
		t.Fatal(err)
	}
	replay, err := pipeline.Process(ctx, Trigger{Symbol: "XRPUSDT", SourceType: "fast_move", SourceID: "sig-1"})
	if err != nil || !replay.Suppressed || replay.Reason != "source_replay" {
		t.Fatalf("expected source replay suppression: %+v err=%v", replay, err)
	}
	cooldown, err := pipeline.Process(ctx, Trigger{Symbol: "XRPUSDT", SourceType: "fast_move", SourceID: "sig-2"})
	if err != nil || !cooldown.Suppressed || cooldown.Reason != "cooldown" {
		t.Fatalf("expected cooldown suppression: %+v err=%v", cooldown, err)
	}
	if cooldown.Opportunity.OpportunityID != first.Opportunity.OpportunityID || starts.Load() != 1 {
		t.Fatalf("suppressed trigger started another analysis: starts=%d first=%s cooldown=%s", starts.Load(), first.Opportunity.OpportunityID, cooldown.Opportunity.OpportunityID)
	}
}

func TestPipelineStartFailureIsPersisted(t *testing.T) {
	service := setupOpportunityService(t)
	pipeline, err := NewPipeline(PipelineConfig{
		Service: service, PollInterval: time.Millisecond, TaskTimeout: time.Second,
		StartTask: func(agentruntime.Request) (*task.Task, error) { return nil, errors.New("runtime unavailable") },
		GetTask:   func(context.Context, string) (*task.Task, error) { return nil, errors.New("must not be called") },
	})
	if err != nil {
		t.Fatal(err)
	}
	_, processErr := pipeline.Process(context.Background(), Trigger{Symbol: "BNBUSDT", SourceType: "market_scan", SourceID: "scan-1"})
	if processErr == nil {
		t.Fatal("runtime start failure must be returned")
	}
	row, readErr := service.Store.FindBySource(context.Background(), "market_scan", "scan-1")
	if readErr != nil {
		t.Fatal(readErr)
	}
	if row.AnalysisStatus != AnalysisFailed || row.AnalysisError != "runtime unavailable" {
		t.Fatalf("start failure not persisted: %+v", row)
	}
}

func TestPipelineConcurrentCooldownAdmitsOnlyOneTask(t *testing.T) {
	var starts atomic.Int64
	pipeline, _ := testPipeline(t, testTradingPlan("ADAUSDT", DirectionLong, 0.81, []string{}), task.StatusSucceeded, "", &starts)

	const count = 16
	results := make(chan ProcessResult, count)
	errorsCh := make(chan error, count)
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			result, err := pipeline.Process(context.Background(), Trigger{
				Symbol: "ADAUSDT", SourceType: "fast_move", SourceID: fmt.Sprintf("sig-concurrent-%d", index),
			})
			if err != nil {
				errorsCh <- err
				return
			}
			results <- result
		}(i)
	}
	wg.Wait()
	close(results)
	close(errorsCh)

	for err := range errorsCh {
		t.Fatalf("concurrent process failed: %v", err)
	}
	created, suppressed := 0, 0
	for result := range results {
		if result.Created {
			created++
		}
		if result.Suppressed && result.Reason == "cooldown" {
			suppressed++
		}
	}
	if created != 1 || suppressed != count-1 || starts.Load() != 1 {
		t.Fatalf("concurrent cooldown gate failed: created=%d suppressed=%d starts=%d", created, suppressed, starts.Load())
	}
}

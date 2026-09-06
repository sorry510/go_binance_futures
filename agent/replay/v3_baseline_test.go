package replay

import (
	"context"
	"encoding/json"
	"sort"
	"testing"
	"time"

	"go_binance_futures/agent/skill"
	alertanalysis "go_binance_futures/agent/skills/alertanalysis"
	symbolanalysis "go_binance_futures/agent/skills/symbolanalysis"
	workflowSkills "go_binance_futures/agent/skills/workflows"
	"go_binance_futures/agent/task"
)

type v3BaselineCase struct {
	name           string
	fixture        string
	definition     skill.Skill
	outputContract string
}

func v3BaselineCases() []v3BaselineCase {
	return []v3BaselineCase{
		{name: "symbol_analysis", fixture: "symbol_analysis_success.json", definition: symbolanalysis.New(), outputContract: "trading_plan_v1"},
		{name: "alert_analysis", fixture: "alert_analysis_success.json", definition: alertanalysis.New(), outputContract: "alert_v1"},
		{name: "market_scan", fixture: "market_scan_success.json", definition: workflowSkills.MarketScan(), outputContract: "opportunity_set_v1"},
		{name: "strategy_review", fixture: "strategy_review_success.json", definition: workflowSkills.StrategyReview(), outputContract: "strategy_review_v1"},
	}
}
func TestV3BaselineCriticalOutputContracts(t *testing.T) {
	for _, tc := range v3BaselineCases() {
		t.Run(tc.name, func(t *testing.T) {
			fixture, err := Load("testdata/" + tc.fixture)
			if err != nil {
				t.Fatal(err)
			}
			out := Run(context.Background(), fixture, tc.definition)
			if out.Err != nil {
				t.Fatalf("baseline replay failed: %v", out.Err)
			}
			if out.Task == nil || out.Task.Status != task.StatusSucceeded {
				t.Fatalf("baseline task changed: %+v", out.Task)
			}
			if out.Task.OutputContractVersion != tc.outputContract {
				t.Fatalf("output contract changed: got=%q want=%q", out.Task.OutputContractVersion, tc.outputContract)
			}
		})
	}
}

func BenchmarkV3BaselineCoreReplay(b *testing.B) {
	cases := v3BaselineCases()
	fixtures := make([]Fixture, len(cases))
	for i, tc := range cases {
		fixture, err := Load("testdata/" + tc.fixture)
		if err != nil {
			b.Fatal(err)
		}
		fixtures[i] = fixture
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for index, tc := range cases {
			out := Run(context.Background(), fixtures[index], tc.definition)
			if out.Err != nil || out.Task == nil || out.Task.Status != task.StatusSucceeded {
				b.Fatalf("%s baseline replay failed: err=%v task=%+v", tc.name, out.Err, out.Task)
			}
		}
	}
}

type v3BaselineMetrics struct {
	Cases         int     `json:"cases"`
	SuccessRate   float64 `json:"success_rate"`
	P95DurationUS int64   `json:"p95_duration_us"`
	TotalTokens   int     `json:"total_tokens"`
	ToolCalls     int     `json:"tool_calls"`
	TotalRounds   int     `json:"total_rounds"`
	AverageRounds float64 `json:"average_rounds"`
}

func TestV3BaselineReplayMetrics(t *testing.T) {
	metrics := v3BaselineMetrics{Cases: len(v3BaselineCases())}
	durations := make([]int64, 0, metrics.Cases)
	succeeded := 0
	for _, tc := range v3BaselineCases() {
		fixture, err := Load("testdata/" + tc.fixture)
		if err != nil {
			t.Fatal(err)
		}
		started := time.Now()
		out := Run(context.Background(), fixture, tc.definition)
		durations = append(durations, time.Since(started).Microseconds())
		if out.Err != nil || out.Task == nil {
			t.Fatalf("%s baseline replay failed: err=%v", tc.name, out.Err)
		}
		if out.Task.Status == task.StatusSucceeded {
			succeeded++
		}
		metrics.TotalTokens += out.Task.Usage.TotalTokens
		metrics.TotalRounds += out.Task.Round
		for _, event := range out.Task.Events {
			if event.Tool != "" {
				metrics.ToolCalls++
			}
		}
	}
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	if metrics.Cases > 0 {
		metrics.SuccessRate = float64(succeeded) / float64(metrics.Cases)
		metrics.AverageRounds = float64(metrics.TotalRounds) / float64(metrics.Cases)
		index := int(float64(len(durations)-1) * 0.95)
		metrics.P95DurationUS = durations[index]
	}
	raw, err := json.Marshal(metrics)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("V3_BASELINE_METRICS %s", raw)
}

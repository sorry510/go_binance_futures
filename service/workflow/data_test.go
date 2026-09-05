package workflow

import (
	workflowSkill "go_binance_futures/agent/skills/workflows"
	"testing"
)

func TestStrategyExperimentDeterministicTestCompilesAndRuns(t *testing.T) {
	proposal := workflowSkill.StrategyExperimentProposalV1{
		Version: "strategy_experiment_proposal_v1", BaseTemplateID: 1, CandidateName: "candidate",
		TechnologyJSON: `{"ma":[{"name":"ma_fast","enable":true,"kline_interval":"15m","period":7}],"ema":[],"macd":[],"rsi":[],"kc":[],"boll":[],"atr":[],"adx":[],"mfi":[],"obv":[],"cci":[],"roc":[],"kdj":[],"supertrend":[],"donchian":[]}`,
		StrategyJSON:   `[{"name":"entry","enable":true,"code":"NowPrice > ma_fast.Data[0] && MarketCondition == \"3\"","type":"long"}]`,
		Rationale:      []string{"test"}, Risks: []string{"test"},
	}
	report := testProposal(proposal)
	if !report.Valid || report.CompiledRules != 1 || report.ScenarioRuns != 3 || len(report.Errors) != 0 {
		t.Fatalf("unexpected deterministic report: %+v", report)
	}
}

func TestStrategyExperimentDeterministicTestRejectsInvalidExpr(t *testing.T) {
	proposal := workflowSkill.StrategyExperimentProposalV1{Version: "strategy_experiment_proposal_v1", BaseTemplateID: 1, CandidateName: "bad", TechnologyJSON: `{}`, StrategyJSON: `[{"name":"bad","enable":true,"code":"NowPrice >>> 2","type":"long"}]`, Rationale: []string{"x"}, Risks: []string{"y"}}
	report := testProposal(proposal)
	if report.Valid || len(report.Errors) == 0 {
		t.Fatalf("invalid expression unexpectedly passed: %+v", report)
	}
}

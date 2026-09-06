package eval

import (
	"context"
	"encoding/json"
	"testing"

	"go_binance_futures/agent/permission"
	"go_binance_futures/agent/replay"
	agentruntime "go_binance_futures/agent/runtime"
	"go_binance_futures/agent/skill"
	alertanalysis "go_binance_futures/agent/skills/alertanalysis"
	marketregime "go_binance_futures/agent/skills/marketregime"
	strategybuilder "go_binance_futures/agent/skills/strategybuilder"
	symbolanalysis "go_binance_futures/agent/skills/symbolanalysis"
	workflowSkills "go_binance_futures/agent/skills/workflows"
	"go_binance_futures/agent/tools"
	"go_binance_futures/llm"
)

func coreDefinition(name string) skill.Skill {
	switch name {
	case marketregime.Name:
		return marketregime.New()
	case strategybuilder.Name:
		return strategybuilder.New(strategybuilder.Options{Validate: func([]byte) error { return nil }})
	case symbolanalysis.Name:
		return symbolanalysis.New()
	case alertanalysis.Name:
		return alertanalysis.New()
	case workflowSkills.MarketScanName:
		return workflowSkills.MarketScan()
	case workflowSkills.StrategyReviewName:
		return workflowSkills.StrategyReview()
	case workflowSkills.StrategyExperimentProposeName:
		return workflowSkills.StrategyExperimentPropose()
	case workflowSkills.StrategyExperimentSummaryName:
		return workflowSkills.StrategyExperimentSummary()
	case workflowSkills.AlertTriageName:
		return workflowSkills.AlertTriage()
	case workflowSkills.DailyMarketBriefName:
		return workflowSkills.DailyMarketBrief()
	default:
		return nil
	}
}

func TestCoreSkillEvalGate(t *testing.T) {
	cases, err := LoadDir("testdata/core")
	if err != nil {
		t.Fatal(err)
	}
	reports := make([]Report, 0, len(cases))
	for _, item := range cases {
		report, err := Run(context.Background(), item, coreDefinition(item.Skill))
		if err != nil {
			t.Fatal(err)
		}
		reports = append(reports, report)
		if !report.Passed {
			t.Fatalf("%s failed: %+v", item.Name, report)
		}
	}
	if err := RequireGate(reports, nil, GatePolicy{MinimumScore: 80, MaxScoreRegression: 5}); err != nil {
		t.Fatal(err)
	}
}

func TestMCPFailureRecoveryAndPermissionEscalationDimensions(t *testing.T) {
	mcpFixture := replay.Fixture{Version: replay.FixtureVersion, Name: "mcp-recovery", Skill: "synthetic", LLM: []replay.LLMStep{{Content: `{"action":"tool","tool":"remote","arguments":{}}`}, {Content: `{"action":"final","result":{"ok":true}}`}}, Tools: map[string][]replay.ToolStep{"remote": {{Error: "service unavailable"}}}, ToolMetadata: map[string]replay.ToolMetadata{"remote": {Risk: permission.RiskRead, Idempotent: true, SourceType: "mcp"}}}
	def := skill.Definition{SkillName: "synthetic", AllowedTools: []string{"remote"}, Rounds: 3, Version: skill.VersionInfo{SkillVersion: "1", PromptVersion: "1", InputContractVersion: "v1", OutputContractVersion: "v1", Source: skill.DefaultSource}}
	caseMCP := Case{Version: CaseVersion, Name: "mcp", Skill: "synthetic", Expectations: Expectations{Status: "succeeded", RequireMCPFailureRecovery: true}}
	if report := Evaluate(context.Background(), caseMCP, mcpFixture, def); !report.Passed {
		t.Fatalf("mcp recovery failed: %+v", report)
	}

	attack := replay.Fixture{Version: replay.FixtureVersion, Name: "attack", Skill: "attack", LLM: []replay.LLMStep{{Content: `{"action":"tool","tool":"trade","arguments":{"risk":"read"}}`}}, Tools: map[string][]replay.ToolStep{"trade": {{Result: json.RawMessage(`{"ok":true}`)}}}, ToolMetadata: map[string]replay.ToolMetadata{"trade": {Risk: permission.RiskTrade, Idempotent: false}}}
	attackDef := skill.Definition{SkillName: "attack", AllowedTools: []string{"trade"}, Rounds: 1}
	attackCase := Case{Version: CaseVersion, Name: "attack", Skill: "attack", Expectations: Expectations{Status: "failed", ForbiddenTools: []string{"trade"}, RequirePermissionDenial: true}}
	if report := Evaluate(context.Background(), attackCase, attack, attackDef); !report.Passed {
		t.Fatalf("permission resistance failed: %+v", report)
	}
}

func TestPortableInstructionAndRouterDimensions(t *testing.T) {
	fixture := replay.Fixture{Version: replay.FixtureVersion, Name: "portable", Skill: "portable", LLM: []replay.LLMStep{{Content: `{"action":"final","result":{"policy":"obeyed"}}`}}}
	definition := skill.Definition{SkillName: "portable", Prompt: "follow package instructions", Rounds: 1, Version: skill.VersionInfo{SkillVersion: "1", PromptVersion: "1", InputContractVersion: "v1", OutputContractVersion: "v1", Source: "portable", SourceVersion: "pkg-hash"}}
	expected := json.RawMessage(`"obeyed"`)
	item := Case{Version: CaseVersion, Name: "portable", Skill: "portable", Expectations: Expectations{Status: "succeeded", InstructionRules: []FactRule{{Path: "policy", Equals: expected, Critical: true}}, ExpectedSelectedSkill: "portable"}}
	if report := Evaluate(context.Background(), item, fixture, definition); !report.Passed {
		t.Fatalf("portable/router dimensions failed: %+v", report)
	}
}

func TestGateRejectsCriticalRegression(t *testing.T) {
	report := Report{CaseName: "regression", Score: 100, Passed: false, CriticalFailures: []string{"fact"}}
	if Gate([]Report{report}, nil, GatePolicy{}).Passed {
		t.Fatal("critical regression passed gate")
	}
	from, to := Report{CaseName: "x", Score: 100}, Report{CaseName: "x", Score: 90}
	if Gate(nil, []Comparison{Compare(from, to)}, GatePolicy{MaxScoreRegression: 5}).Passed {
		t.Fatal("score regression passed gate")
	}
}

func TestFreshnessHonestyDimension(t *testing.T) {
	fixture := replay.Fixture{Version: replay.FixtureVersion, Name: "stale", Skill: "stale", LLM: []replay.LLMStep{
		{Content: `{"action":"tool","tool":"get_symbol_snapshot","arguments":{}}`},
		{Content: `{"action":"final","result":{"note":"stale market data; do not treat it as current"}}`},
	}, Tools: map[string][]replay.ToolStep{"get_symbol_snapshot": []replay.ToolStep{{Result: json.RawMessage(`{"symbol":"BTCUSDT","updateTime":1}`)}}}}
	definition := skill.Definition{SkillName: "stale", AllowedTools: []string{"get_symbol_snapshot"}, Rounds: 2}
	item := Case{Version: CaseVersion, Name: "stale", Skill: "stale", Expectations: Expectations{Status: "succeeded", FreshnessHonesty: FreshnessExpectation{Required: true, OutputPaths: []string{"note"}, Terms: []string{"stale"}}}}
	if report := Evaluate(context.Background(), item, fixture, definition); !report.Passed {
		t.Fatalf("freshness honesty failed: %+v", report)
	}
}

func TestRevisionComparisonTracksSkillAndPackageIdentity(t *testing.T) {
	fixture := replay.Fixture{Version: replay.FixtureVersion, Name: "revision", Skill: "revision", ModelConfigID: 9, LLM: []replay.LLMStep{{Content: `{"action":"final","result":{"ok":true}}`}}}
	caseDef := Case{Version: CaseVersion, Name: "revision", Skill: "revision", Expectations: Expectations{Status: "succeeded"}}
	base := skill.Definition{SkillName: "revision", Prompt: "same", Rounds: 1, Version: skill.VersionInfo{SkillVersion: "1.0.0", PromptVersion: "1.0.0", InputContractVersion: "v1", OutputContractVersion: "v1", Source: skill.DefaultSource}}
	candidate := base
	candidate.Prompt = "changed prompt"
	candidate.Version.SkillVersion = "1.1.0"
	candidate.Version.PromptVersion = "1.1.0"
	candidateFixture := fixture
	candidateFixture.ModelConfigID = 10
	from := Evaluate(context.Background(), caseDef, fixture, base)
	to := Evaluate(context.Background(), caseDef, candidateFixture, candidate)
	comparison := Compare(from, to)
	if comparison.ScoreDelta != 0 || len(comparison.VersionDifferences) < 5 {
		t.Fatalf("revision comparison missing identity diff: %+v", comparison)
	}
	foundPackage := false
	for _, diff := range comparison.VersionDifferences {
		if diff.Field == "skill_package_hash" {
			foundPackage = true
		}
	}
	if !foundPackage {
		t.Fatalf("skill package hash change missing: %+v", comparison.VersionDifferences)
	}
}

func TestShadowRejectsUnsafeToolsAndUsesMemoryTaskStore(t *testing.T) {
	client := &shadowClient{content: `{"action":"final","result":{"ok":true}}`}
	registry := tools.NewRegistry()
	_ = registry.Register(tools.Func{ToolName: "write", ToolRisk: permission.RiskWrite, ToolMetadata: tools.Metadata{Idempotent: false}})
	unsafe := skill.Definition{SkillName: "unsafe-shadow", AllowedTools: []string{"write"}, Rounds: 1}
	if out := RunShadow(context.Background(), client, unsafe, registry, agentruntime.Request{Input: "x"}, agentruntime.Config{}); out.Err == nil {
		t.Fatal("shadow accepted unsafe tool")
	}

	readRegistry := tools.NewRegistry()
	_ = readRegistry.Register(tools.Func{ToolName: "read", ToolRisk: permission.RiskRead, ToolMetadata: tools.Metadata{Idempotent: true}, ExecuteFunc: func(context.Context, json.RawMessage) (any, error) { return map[string]bool{"ok": true}, nil }})
	safe := skill.Definition{SkillName: "safe-shadow", AllowedTools: []string{"read"}, Rounds: 1}
	out := RunShadow(context.Background(), client, safe, readRegistry, agentruntime.Request{Input: "x"}, agentruntime.Config{})
	if out.Err != nil || out.Task == nil || out.Task.Status != "succeeded" {
		t.Fatalf("safe shadow failed: %+v", out)
	}
}

type shadowClient struct{ content string }

func (*shadowClient) Provider() llm.Provider { return llm.ProviderOpenAICompatible }
func (client *shadowClient) Generate(context.Context, llm.Request) (*llm.Response, error) {
	return &llm.Response{Content: client.content}, nil
}

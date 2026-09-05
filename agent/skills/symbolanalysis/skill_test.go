package symbolanalysis

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"go_binance_futures/agent/contextengine"
	"go_binance_futures/agent/skill"
	marketservice "go_binance_futures/service/market"
	symbolanalysisservice "go_binance_futures/service/symbolanalysis"
)

func TestSkillRequiresAggregateContext(t *testing.T) {
	definition := New()
	input, _ := json.Marshal(Input{Symbol: "ongusdt", Prompt: "分析"})
	if err := definition.ValidateInput(skill.Request{Input: string(input)}); err != nil {
		t.Fatal(err)
	}
	messages, err := definition.BuildInput(context.Background(), skill.Request{Input: string(input)})
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 1 || !strings.Contains(messages[0].Content, "ONGUSDT") {
		t.Fatalf("unexpected messages: %+v", messages)
	}
	required := definition.RequiredTools(skill.Request{Input: string(input)})
	if len(required) != 1 || required[0] != "get_symbol_analysis_context" {
		t.Fatalf("unexpected required tools: %v", required)
	}
}

func TestSkillRejectsInvalidInput(t *testing.T) {
	if err := New().ValidateInput(skill.Request{Input: `{"symbol":"BTCUSDC"}`}); err == nil {
		t.Fatal("expected non-USDT input to fail")
	}
}
func TestRunValidatorUsesToolContext(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	contextValue := testAnalysisContext(now)
	input := skill.Request{Input: `{"symbol":"ONGUSDT"}`}
	validator := New().ValidatorForRun(input, map[string]any{"get_symbol_analysis_context": contextValue})
	value, err := validator.Validate(context.Background(), validPlan(now, "ONGUSDT", 2, []string{"funding_rate"}, 99, 100, 95, []float64{105, 110}))
	if err != nil {
		t.Fatal(err)
	}
	plan, ok := value.(TradingPlanV1)
	if !ok || plan.Direction != "long" {
		t.Fatalf("unexpected plan: %#v", value)
	}
}

func TestRunValidatorRejectsContextMismatch(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	contextValue := testAnalysisContext(now)
	validator := New().ValidatorForRun(skill.Request{Input: `{"symbol":"ONGUSDT"}`}, map[string]any{"get_symbol_analysis_context": contextValue})
	cases := []struct {
		name string
		raw  json.RawMessage
		want string
	}{
		{"symbol", validPlan(now, "BTCUSDT", 2, []string{"funding_rate"}, 99, 100, 95, []float64{105}), "symbol mismatch"},
		{"condition", validPlan(now, "ONGUSDT", 4, []string{"funding_rate"}, 99, 100, 95, []float64{105}), "market_condition must match"},
		{"missing", validPlan(now, "ONGUSDT", 2, []string{}, 99, 100, 95, []float64{105}), "data_missing must preserve"},
		{"price", validPlan(now, "ONGUSDT", 2, []string{"funding_rate"}, 1000, 1100, 900, []float64{1200}), "obviously inconsistent"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := validator.Validate(context.Background(), tc.raw)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q, got %v", tc.want, err)
			}
		})
	}
}
func testAnalysisContext(now time.Time) symbolanalysisservice.Context {
	condition := marketservice.Condition{MarketCondition: 2, Name: "偏多头", Auto: true}
	return symbolanalysisservice.Context{
		Symbol:          "ONGUSDT",
		AsOf:            now.Format(time.RFC3339),
		Snapshot:        symbolanalysisservice.Snapshot{Symbol: "ONGUSDT", Price: 100, UpdatedAtMs: now.UnixMilli()},
		MarketCondition: &condition,
		Klines:          []symbolanalysisservice.KlineFeature{},
		DataMissing:     []string{"funding_rate"},
	}
}

func validPlan(now time.Time, symbol string, condition int, missing []string, entryLow, entryHigh, stop float64, targets []float64) json.RawMessage {
	plan := TradingPlanV1{
		Version: "trading_plan_v1", Symbol: symbol, AsOf: now.Format(time.RFC3339), MarketCondition: &condition,
		Direction: "long", Confidence: 0.72, Summary: "偏多，但等待回踩确认",
		EntryZones: []PriceZone{{Low: entryLow, High: entryHigh}}, StopLoss: &stop, TakeProfits: targets,
		LongTrigger: "15m 回踩企稳后重新走强", ShortTrigger: "跌破关键支撑则取消多头计划",
		InvalidationConditions: []string{"跌破止损并持续放量"}, Risks: []string{"短周期波动较高"},
		DataMissing: append([]string{}, missing...), Evidence: []Evidence{{Source: "get_symbol_analysis_context", Finding: "多周期结构与市场环境偏多"}},
	}
	data, _ := json.Marshal(plan)
	return data
}

func TestValidatorAllowsNeutralWithoutEntries(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	condition := 3
	plan := TradingPlanV1{Version: "trading_plan_v1", Symbol: "ONGUSDT", AsOf: now.Format(time.RFC3339), MarketCondition: &condition, Direction: "neutral", Confidence: 0.4, Summary: "证据混合，暂不交易", EntryZones: []PriceZone{}, TakeProfits: []float64{}, InvalidationConditions: []string{"趋势形成后重新评估"}, Risks: []string{"方向不明确"}, DataMissing: []string{}, Evidence: []Evidence{{Source: "get_symbol_analysis_context", Finding: "多周期方向不一致"}}}
	raw, _ := json.Marshal(plan)
	if _, err := New().ValidatorFor(skill.Request{Input: `{"symbol":"ONGUSDT"}`}).Validate(context.Background(), raw); err != nil {
		t.Fatalf("neutral plan should be valid: %v", err)
	}
}

func TestRunValidatorAllowsSuccessfulMCPEvidenceSource(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	contextValue := testAnalysisContext(now)
	mcpSource := "mcp.coingecko-free.get-crypto-news"
	toolResults := map[string]any{
		"get_symbol_analysis_context": contextValue,
		mcpSource:                     map[string]any{"articles": []any{}},
	}
	evidence := map[string]contextengine.Evidence{
		"native": {ID: "native", SourceType: "tool", Source: "get_symbol_analysis_context", ObservedAt: now.Format(time.RFC3339), ContentHash: "native-hash", Freshness: contextengine.FreshnessFresh},
		"mcp":    {ID: "mcp", SourceType: "tool", Source: mcpSource, ObservedAt: now.Format(time.RFC3339), ContentHash: "mcp-hash", Freshness: contextengine.FreshnessFresh},
	}
	raw := validPlan(now, "ONGUSDT", 2, []string{"funding_rate"}, 99, 100, 95, []float64{105})
	var plan TradingPlanV1
	if err := json.Unmarshal(raw, &plan); err != nil {
		t.Fatal(err)
	}
	plan.Evidence = append(plan.Evidence, Evidence{Source: mcpSource, Finding: "latest news is neutral"})
	raw, _ = json.Marshal(plan)
	if _, err := New().ValidatorForRunWithEvidence(skill.Request{Input: `{"symbol":"ONGUSDT"}`}, toolResults, evidence).Validate(context.Background(), raw); err != nil {
		t.Fatalf("successful MCP evidence should be accepted: %v", err)
	}
}

func TestValidatorRejectsUnknownNonMCPSource(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	raw := validPlan(now, "ONGUSDT", 2, []string{"funding_rate"}, 99, 100, 95, []float64{105})
	var plan TradingPlanV1
	if err := json.Unmarshal(raw, &plan); err != nil {
		t.Fatal(err)
	}
	plan.Evidence = append(plan.Evidence, Evidence{Source: "invented_external_source", Finding: "unsupported"})
	raw, _ = json.Marshal(plan)
	_, err := New().ValidatorFor(skill.Request{Input: `{"symbol":"ONGUSDT"}`}).Validate(context.Background(), raw)
	if err == nil || !strings.Contains(err.Error(), "unsupported source") {
		t.Fatalf("expected unknown source rejection, got %v", err)
	}
}

func TestRunValidatorRejectsUnusedEvidenceSource(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	contextValue := testAnalysisContext(now)
	validator := New().ValidatorForRun(skill.Request{Input: `{"symbol":"ONGUSDT"}`}, map[string]any{
		"get_symbol_analysis_context": contextValue,
	})
	raw := validPlan(now, "ONGUSDT", 2, []string{"funding_rate"}, 99, 100, 95, []float64{105})
	var plan TradingPlanV1
	if err := json.Unmarshal(raw, &plan); err != nil {
		t.Fatal(err)
	}
	plan.Evidence = append(plan.Evidence, Evidence{Source: "get_klines", Finding: "15m 结构偏强"})
	raw, _ = json.Marshal(plan)
	_, err := validator.Validate(context.Background(), raw)
	if err == nil || !strings.Contains(err.Error(), "was not successfully called") {
		t.Fatalf("expected unused evidence source rejection, got %v", err)
	}
}

func TestSymbolAnalysisMaxRounds(t *testing.T) {
	if got := New().MaxRounds(); got != 15 {
		t.Fatalf("MaxRounds() = %d, want 15", got)
	}
}
func TestBuildChatInputConvertsNaturalLanguageToExistingContract(t *testing.T) {
	definition := New()
	raw, err := definition.BuildChatInput(context.Background(), "请重新分析 ongusdt 最近的走势")
	if err != nil {
		t.Fatal(err)
	}
	var input Input
	if err := json.Unmarshal([]byte(raw), &input); err != nil {
		t.Fatal(err)
	}
	if input.Symbol != "ONGUSDT" || input.Prompt != "请重新分析 ongusdt 最近的走势" {
		t.Fatalf("unexpected chat input: %+v", input)
	}
	if err := definition.ValidateInput(skill.Request{Input: raw}); err != nil {
		t.Fatalf("chat adapter must preserve symbol_analysis_input_v1: %v", err)
	}
}

func TestBuildChatInputRequiresExplicitUSDTContract(t *testing.T) {
	_, err := New().BuildChatInput(context.Background(), "帮我分析一下比特币")
	if err == nil || !strings.Contains(err.Error(), "BTCUSDT") {
		t.Fatalf("expected explicit symbol guidance, got %v", err)
	}
}

func TestBuildChatInputWithContextReusesPreviousSuccessfulSymbol(t *testing.T) {
	raw, err := New().BuildChatInputWithContext(context.Background(), "刚才最需要注意的风险是什么？", []string{`{"symbol":"BTCUSDT","prompt":"分析 BTCUSDT"}`})
	if err != nil {
		t.Fatal(err)
	}
	var input Input
	if err := json.Unmarshal([]byte(raw), &input); err != nil {
		t.Fatal(err)
	}
	if input.Symbol != "BTCUSDT" || input.Prompt != "刚才最需要注意的风险是什么？" {
		t.Fatalf("unexpected contextual chat input: %+v", input)
	}
}

func TestBuildChatInputWithContextPrefersCurrentExplicitSymbol(t *testing.T) {
	raw, err := New().BuildChatInputWithContext(context.Background(), "改为分析 ETHUSDT", []string{`{"symbol":"BTCUSDT"}`})
	if err != nil {
		t.Fatal(err)
	}
	var input Input
	_ = json.Unmarshal([]byte(raw), &input)
	if input.Symbol != "ETHUSDT" {
		t.Fatalf("current explicit symbol must win: %+v", input)
	}
}

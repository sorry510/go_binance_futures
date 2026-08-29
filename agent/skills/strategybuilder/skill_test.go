package strategybuilder

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"go_binance_futures/agent/skill"
	"go_binance_futures/llm"
)

func TestBuilderRequiredTools(t *testing.T) {
	builder := New(Options{Validate: func([]byte) error { return nil }})
	input, _ := json.Marshal(Input{Prompt: "请查询 ONGUSDT 合约数据，并调用测试结果分析"})
	required := builder.RequiredTools(skill.Request{Input: string(input)})
	joined := strings.Join(required, ",")
	if !strings.Contains(joined, "get_features") || !strings.Contains(joined, "get_test_strategy_results") {
		t.Fatalf("unexpected required tools: %v", required)
	}
}

func TestBuilderBuildInputPreservesConversation(t *testing.T) {
	builder := New(Options{Validate: func([]byte) error { return nil }})
	input, _ := json.Marshal(Input{Prompt: "修复策略", PreviousJSON: `{"name":"old"}`, ValidationError: "bad rule"})
	history := []llm.Message{{Role: llm.RoleUser, Content: "old request"}, {Role: llm.RoleAssistant, Content: "old answer"}}
	messages, err := builder.BuildInput(context.Background(), skill.Request{Input: string(input), Metadata: map[string]any{HistoryMetadataKey: history}})
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 3 || messages[0].Content != "old request" {
		t.Fatalf("unexpected history: %+v", messages)
	}
	last := messages[len(messages)-1].Content
	if !strings.Contains(last, "修复策略") || !strings.Contains(last, "bad rule") || !strings.Contains(last, `{"name":"old"}`) {
		t.Fatalf("unexpected user prompt: %s", last)
	}
}

func TestBuilderValidatorAddsRepairGuidance(t *testing.T) {
	builder := New(Options{
		Validate: func([]byte) error { return fmt.Errorf("unknown name rsi_14_Data") },
		RepairGuidance: func(message, candidate string) string {
			if !strings.Contains(candidate, `"name":"x"`) {
				t.Fatalf("candidate missing: %s", candidate)
			}
			return "use rsi_14.Data"
		},
	})
	_, err := builder.Validator().Validate(context.Background(), json.RawMessage(`{"name":"x"}`))
	if err == nil || !strings.Contains(err.Error(), "required_fix") || !strings.Contains(err.Error(), "rsi_14.Data") {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestBuilderUsesRuntimeFinalProtocol(t *testing.T) {
	if !strings.Contains(systemPrompt, `"action":"final"`) || !strings.Contains(systemPrompt, `"result":{"name"`) {
		t.Fatal("system prompt must use runtime final result protocol")
	}
	if strings.Contains(systemPrompt, `"json":{"name"`) {
		t.Fatal("legacy final json protocol must not remain")
	}
}

func TestBuilderRequiresCurrentMarketConditionForMarketTrend(t *testing.T) {
	builder := New(Options{Validate: func([]byte) error { return nil }})
	input, _ := json.Marshal(Input{Prompt: "请根据市场趋势（当前行情）编写开仓策略"})
	required := builder.RequiredTools(skill.Request{Input: string(input)})
	if !strings.Contains(strings.Join(required, ","), "get_market_condition") {
		t.Fatalf("market condition tool should be required: %v", required)
	}
	if !strings.Contains(strings.Join(builder.Tools(), ","), "get_market_condition") {
		t.Fatalf("market condition tool should be allowed: %v", builder.Tools())
	}
}

func TestBuilderKeepsMarketConditionRequirementAcrossConversation(t *testing.T) {
	builder := New(Options{Validate: func([]byte) error { return nil }})
	input, _ := json.Marshal(Input{Prompt: "继续修复上一版策略"})
	history := []llm.Message{{Role: llm.RoleUser, Content: "User requirements:\n请结合当前行情生成策略"}}
	required := builder.RequiredTools(skill.Request{Input: string(input), Metadata: map[string]any{HistoryMetadataKey: history}})
	if !strings.Contains(strings.Join(required, ","), "get_market_condition") {
		t.Fatalf("conversation should preserve market condition requirement: %v", required)
	}
}

func TestRequiresMarketConditionDoesNotMatchIndicatorTrendOnly(t *testing.T) {
	if RequiresMarketCondition("使用 SuperTrend 趋势指标进行突破判断") {
		t.Fatal("indicator trend alone should not require market regime")
	}
}

func TestBuilderMarketConditionValidatorRequiresAllRegimes(t *testing.T) {
	builder := New(Options{Validate: func([]byte) error { return nil }, RequireMarketCondition: true})
	raw := marketConditionStrategyCandidate(10, false)
	_, err := builder.Validator().Validate(context.Background(), raw)
	if err == nil || !strings.Contains(err.Error(), "缺少: 11") {
		t.Fatalf("expected missing regime 11, got %v", err)
	}
}

func TestBuilderMarketConditionValidatorRejectsUngatedOpeningRule(t *testing.T) {
	builder := New(Options{Validate: func([]byte) error { return nil }, RequireMarketCondition: true})
	raw := marketConditionStrategyCandidate(11, true)
	_, err := builder.Validator().Validate(context.Background(), raw)
	if err == nil || !strings.Contains(err.Error(), "必须显式使用 MarketCondition") {
		t.Fatalf("expected ungated opening rule error, got %v", err)
	}
}

func TestBuilderMarketConditionValidatorAcceptsAllRegimes(t *testing.T) {
	builder := New(Options{Validate: func([]byte) error { return nil }, RequireMarketCondition: true})
	if _, err := builder.Validator().Validate(context.Background(), marketConditionStrategyCandidate(11, false)); err != nil {
		t.Fatalf("complete market condition coverage should pass: %v", err)
	}
}

func marketConditionStrategyCandidate(regimes int, addUngated bool) json.RawMessage {
	rules := make([]map[string]any, 0, regimes+1)
	for condition := 1; condition <= regimes; condition++ {
		ruleType := "long"
		if condition >= 4 && condition != 6 && condition != 8 {
			ruleType = "short"
		}
		rules = append(rules, map[string]any{
			"name": fmt.Sprintf("regime_%d", condition), "type": ruleType,
			"code":       fmt.Sprintf(`MarketCondition == "%d" && NowPrice > %d`, condition, condition),
			"fullScreen": false, "enable": true,
		})
	}
	if addUngated {
		rules = append(rules, map[string]any{
			"name": "ungated", "type": "long", "code": "NowPrice > 0", "fullScreen": false, "enable": true,
		})
	}
	payload, _ := json.Marshal(map[string]any{"name": "market-condition-test", "technology": map[string]any{}, "strategy": rules})
	return payload
}

func TestMarketConditionConversationIgnoresImportPayloads(t *testing.T) {
	history := []llm.Message{{Role: llm.RoleUser, Content: `IMPORT_ERROR
{"json":"MarketCondition == \"1\""}`}}
	if RequiresMarketConditionForConversation("继续修复", history) {
		t.Fatal("import payload content must not create a new market condition requirement")
	}
}

func TestBuilderMarketConditionValidatorAcceptsGroupedRegimes(t *testing.T) {
	builder := New(Options{Validate: func([]byte) error { return nil }, RequireMarketCondition: true})
	rules := []map[string]any{
		{"name": "bull", "type": "long", "code": `(MarketCondition == "1" || MarketCondition == "2" || MarketCondition == "6" || MarketCondition == "8") && NowPrice > 0`, "fullScreen": false, "enable": true},
		{"name": "bear", "type": "short", "code": `(MarketCondition == "4" || MarketCondition == "5" || MarketCondition == "7" || MarketCondition == "9") && NowPrice > 0`, "fullScreen": false, "enable": true},
		{"name": "range", "type": "long", "code": `(MarketCondition == "3" || MarketCondition == "10" || MarketCondition == "11") && NowPrice > 0`, "fullScreen": false, "enable": true},
	}
	payload, _ := json.Marshal(map[string]any{"name": "grouped", "technology": map[string]any{}, "strategy": rules})
	if _, err := builder.Validator().Validate(context.Background(), payload); err != nil {
		t.Fatalf("grouped MarketCondition branches should pass: %v", err)
	}
}

func TestBuilderMarketConditionValidatorStillRejectsMissingGroupedRegime(t *testing.T) {
	builder := New(Options{Validate: func([]byte) error { return nil }, RequireMarketCondition: true})
	raw := json.RawMessage(`{"name":"grouped","technology":{},"strategy":[{"name":"bull","type":"long","code":"MarketCondition == \"1\" || MarketCondition == \"2\"","fullScreen":false,"enable":true}]}`)
	_, err := builder.Validator().Validate(context.Background(), raw)
	if err == nil || !strings.Contains(err.Error(), "缺少:") {
		t.Fatalf("expected incomplete grouped coverage error, got %v", err)
	}
}
func TestBuilderRejectsProfitOnlyCloseRule(t *testing.T) {
	builder := New(Options{Validate: func([]byte) error { return nil }})
	raw := json.RawMessage(`{"name":"close","technology":{},"strategy":[{"name":"take_profit","type":"close_long","code":"ROI >= 5","fullScreen":false,"enable":true}]}`)
	_, err := builder.Validator().Validate(context.Background(), raw)
	if err == nil || !strings.Contains(err.Error(), "平仓逻辑过于简单") {
		t.Fatalf("expected profit-only close rejection, got %v", err)
	}
}

func TestBuilderAcceptsConfirmedCloseRule(t *testing.T) {
	builder := New(Options{Validate: func([]byte) error { return nil }})
	raw := json.RawMessage(`{"name":"close","technology":{},"strategy":[{"name":"confirmed_exit","type":"close_long","code":"ROI >= 3 && (MarketCondition == \"4\" || BasicTrend < 0)","fullScreen":false,"enable":true}]}`)
	if _, err := builder.Validator().Validate(context.Background(), raw); err != nil {
		t.Fatalf("confirmed close rule should pass: %v", err)
	}
}
func TestBuilderSystemPromptStaysCompactAndKeepsQualityRules(t *testing.T) {
	if len(systemPrompt) > 6000 {
		t.Fatalf("system prompt is too long: %d bytes", len(systemPrompt))
	}
	for _, required := range []string{"get_market_condition", "MAY share one rule", "MUST NOT be the sole reason to close", "descriptive let variables"} {
		if !strings.Contains(systemPrompt, required) {
			t.Fatalf("system prompt missing %q", required)
		}
	}
}
func TestBuilderRejectsPnlOnlyOrBranch(t *testing.T) {
	builder := New(Options{Validate: func([]byte) error { return nil }})
	raw := json.RawMessage(`{"name":"close","technology":{},"strategy":[{"name":"bad_or_exit","type":"close_long","code":"ROI >= 5 || MarketCondition == \"4\"","fullScreen":false,"enable":true}]}`)
	_, err := builder.Validator().Validate(context.Background(), raw)
	if err == nil || !strings.Contains(err.Error(), "可独立触发的盈亏分支") {
		t.Fatalf("expected pnl-only OR branch rejection, got %v", err)
	}
}

func TestBuilderRejectsComplexOneLineCode(t *testing.T) {
	builder := New(Options{Validate: func([]byte) error { return nil }})
	code := `MarketCondition == "1" && BasicTrend > 0 && NowPrice > NowSymbolOpen && BTCUSDT.PercentChange > 0 && ETHUSDT.PercentChange > 0 && SOLUSDT.PercentChange > 0 && BNBUSDT.PercentChange > 0`
	payload, _ := json.Marshal(map[string]any{"name": "one-line", "technology": map[string]any{}, "strategy": []map[string]any{{"name": "long", "type": "long", "code": code, "fullScreen": false, "enable": true}}})
	_, err := builder.Validator().Validate(context.Background(), payload)
	if err == nil || !strings.Contains(err.Error(), "不能全部写在一行") {
		t.Fatalf("expected complex one-line rejection, got %v", err)
	}
}

func TestBuilderAcceptsReadableLetCode(t *testing.T) {
	builder := New(Options{Validate: func([]byte) error { return nil }})
	code := "let regime_ok = MarketCondition == \"1\" && BasicTrend > 0;\nlet benchmark_ok = BTCUSDT.PercentChange > 0 && ETHUSDT.PercentChange > 0;\nlet price_ok = NowPrice > NowSymbolOpen && SOLUSDT.PercentChange > 0 && BNBUSDT.PercentChange > 0;\nregime_ok && benchmark_ok && price_ok"
	payload, _ := json.Marshal(map[string]any{"name": "readable", "technology": map[string]any{}, "strategy": []map[string]any{{"name": "long", "type": "long", "code": code, "fullScreen": false, "enable": true}}})
	if _, err := builder.Validator().Validate(context.Background(), payload); err != nil {
		t.Fatalf("readable let code should pass: %v", err)
	}
}

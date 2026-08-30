package marketregime

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"go_binance_futures/agent/skill"
	marketservice "go_binance_futures/service/market"
)

func TestSkillBuildInputContainsSnapshot(t *testing.T) {
	definition := New()
	payload, _ := json.Marshal(marketservice.RegimeSnapshot{SymbolCount: 10, AdvancingCount: 7, DecliningCount: 3})
	messages, err := definition.BuildInput(context.Background(), skill.Request{Input: string(payload)})
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 1 || !strings.Contains(messages[0].Content, `"symbol_count":10`) {
		t.Fatalf("unexpected messages: %+v", messages)
	}
	if len(definition.Tools()) != 0 {
		t.Fatalf("market regime skill must not expose tools: %+v", definition.Tools())
	}
}

func TestSkillValidatorAcceptsValidResult(t *testing.T) {
	value, err := New().Validator().Validate(context.Background(), json.RawMessage(`{"market_condition":7,"confidence":0.82,"reason":"市场分化"}`))
	if err != nil {
		t.Fatal(err)
	}
	analysis, ok := value.(Analysis)
	if !ok || analysis.MarketCondition != 7 || analysis.Confidence != 0.82 {
		t.Fatalf("unexpected analysis: %#v", value)
	}
}
func TestSkillValidatorRejectsInvalidResults(t *testing.T) {
	cases := []string{
		`{"market_condition":99,"confidence":0.5,"reason":"bad"}`,
		`{"market_condition":3,"confidence":1.2,"reason":"bad"}`,
		`{"market_condition":3,"confidence":0.5,"reason":""}`,
		`{"market_condition":3,"confidence":0.5,"reason":"ok","extra":1}`,
	}
	for _, input := range cases {
		if _, err := New().Validator().Validate(context.Background(), json.RawMessage(input)); err == nil {
			t.Fatalf("expected validator failure: %s", input)
		}
	}
}

func TestSkillBuildInputRejectsEmptySnapshot(t *testing.T) {
	payload, _ := json.Marshal(marketservice.RegimeSnapshot{})
	if _, err := New().BuildInput(context.Background(), skill.Request{Input: string(payload)}); err == nil {
		t.Fatal("expected empty snapshot to fail")
	}
}

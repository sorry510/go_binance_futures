package controllers

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestStrategyTemplateAIDecisionContract(t *testing.T) {
	toolDecision, err := parseStrategyTemplateAIAgentDecision(`{"action":"tool","summary":"need data","tool":"get_features","arguments":{"symbol":"BTCUSDT"}}`)
	if err != nil {
		t.Fatal(err)
	}
	if toolDecision.Action != "tool" || toolDecision.Tool != "get_features" {
		t.Fatalf("unexpected tool decision: %+v", toolDecision)
	}

	finalDecision, err := parseStrategyTemplateAIAgentDecision(`{"action":"final","summary":"done","json":{"name":"x","technology":{},"strategy":[]}}`)
	if err != nil {
		t.Fatal(err)
	}
	if finalDecision.Action != "final" || len(finalDecision.JSON) == 0 {
		t.Fatalf("unexpected final decision: %+v", finalDecision)
	}
}

func TestStrategyTemplateAIToolErrorContract(t *testing.T) {
	message := buildStrategyTemplateAIToolResultMessage("get_features", "", fmt.Errorf("invalid arguments"))
	if !strings.HasPrefix(message, "TOOL_RESULT\n") || !strings.Contains(message, `"ok":false`) || !strings.Contains(message, "invalid arguments") {
		t.Fatalf("unexpected tool error message: %s", message)
	}
}

func TestStrategyTemplateAIRequiredToolsContract(t *testing.T) {
	required := requiredStrategyTemplateAITools("请查询 ONGUSDT 合约数据，并调用测试结果分析")
	joined := strings.Join(required, ",")
	if !strings.Contains(joined, "get_features") || !strings.Contains(joined, "get_test_strategy_results") {
		t.Fatalf("unexpected required tools: %v", required)
	}
}

func TestStrategyTemplateAIValidationRejectsIncompleteJSON(t *testing.T) {
	payload := map[string]any{
		"name": "broken",
		"technology": map[string]any{
			"ma": []any{},
		},
		"strategy": []any{},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateGeneratedStrategyTemplateJSON(data); err == nil {
		t.Fatal("expected incomplete strategy template JSON to fail validation")
	}
}

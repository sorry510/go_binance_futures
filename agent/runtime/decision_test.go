package runtime

import (
	"strings"
	"testing"
)

func TestParseDecisionRepairsSingleExtraObjectTerminatorAfterArrayElement(t *testing.T) {
	content := `{"action":"final","summary":"repaired","result":{"name":"example","technology":{"supertrend":[{"name":"st","enable":true}}],"strategy":[]}}`

	parsed, err := parseDecision(content)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Action != "final" || parsed.Summary != "repaired" {
		t.Fatalf("unexpected decision: %+v", parsed)
	}
	if !strings.Contains(string(parsed.Result), `"supertrend"`) {
		t.Fatalf("unexpected result: %s", parsed.Result)
	}
}

func TestParseDecisionDoesNotRepairOtherMalformedJSON(t *testing.T) {
	content := `{"action":"final","result":{"strategy":[{"name":"broken",}]}}`

	_, err := parseDecision(content)
	if err == nil {
		t.Fatal("expected malformed decision to fail")
	}
	if !strings.Contains(err.Error(), "decode agent decision") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseDecisionKeepsStrictValidJSON(t *testing.T) {
	content := `{"action":"tool","tool":"get_market_condition","arguments":{}}`

	parsed, err := parseDecision(content)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Action != "tool" || parsed.Tool != "get_market_condition" {
		t.Fatalf("unexpected decision: %+v", parsed)
	}
}

func TestParseDecisionAcceptsParallelTools(t *testing.T) {
	content := `{"action":"parallel_tools","tools":[{"tool":"get_funding_rate","arguments":{"symbol":"BTCUSDT"}},{"tool":"get_market_condition","arguments":{}}]}`
	parsed, err := parseDecision(content)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Action != "parallel_tools" || len(parsed.Tools) != 2 || parsed.Tools[0].Tool != "get_funding_rate" {
		t.Fatalf("unexpected decision: %+v", parsed)
	}
}

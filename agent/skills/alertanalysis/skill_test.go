package alertanalysis

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"go_binance_futures/agent/skill"
	signalservice "go_binance_futures/service/signal"
	symbolanalysisservice "go_binance_futures/service/symbolanalysis"
)

func TestSkillRequiresAggregateContext(t *testing.T) {
	definition := New()
	if tools := definition.RequiredTools(skill.Request{}); len(tools) != 1 || tools[0] != "get_symbol_analysis_context" {
		t.Fatalf("unexpected required tools: %v", tools)
	}
}

func TestRunValidatorMatchesSignalAndToolContext(t *testing.T) {
	input := testInput()
	rawInput, _ := json.Marshal(input)
	analysisContext := symbolanalysisservice.Context{
		Symbol: "BTCUSDT", AsOf: time.Now().UTC().Format(time.RFC3339),
		Snapshot:    symbolanalysisservice.Snapshot{Symbol: "BTCUSDT", Price: 100},
		DataMissing: []string{"depth"},
	}
	validator := New().ValidatorForRun(
		skill.Request{Input: string(rawInput)},
		map[string]any{"get_symbol_analysis_context": analysisContext},
	)
	alert := AlertV1{
		Version: "alert_v1", AlertID: input.AlertID, SignalID: input.Signal.SignalID,
		Symbol: input.Signal.Symbol, Type: input.Signal.Type, Severity: signalservice.SeverityHigh,
		Summary: "快速上涨得到确认", MarketContext: "短周期动能偏强",
		ConfirmedBy: []string{"价格快速上涨"}, Risks: []string{"高波动"}, Action: "notify",
		CooldownUntil: time.Now().UTC().Add(10 * time.Minute).Format(time.RFC3339),
		DataMissing:   []string{"depth"},
		Evidence:      []Evidence{{Source: "get_symbol_analysis_context", Finding: "多周期动能偏强"}},
	}
	raw, _ := json.Marshal(alert)
	value, err := validator.Validate(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if value.(AlertV1).SignalID != input.Signal.SignalID {
		t.Fatalf("unexpected alert: %+v", value)
	}
}

func TestRunValidatorRejectsMissingContextData(t *testing.T) {
	input := testInput()
	rawInput, _ := json.Marshal(input)
	validator := New().ValidatorForRun(
		skill.Request{Input: string(rawInput)},
		map[string]any{"get_symbol_analysis_context": symbolanalysisservice.Context{Symbol: "BTCUSDT", DataMissing: []string{"funding_rate"}}},
	)
	alert := AlertV1{
		Version: "alert_v1", AlertID: input.AlertID, SignalID: input.Signal.SignalID,
		Symbol: "BTCUSDT", Type: input.Signal.Type, Severity: signalservice.SeverityMedium,
		Summary: "待观察", MarketContext: "信息不足", ConfirmedBy: []string{}, Risks: []string{"数据不足"},
		Action: "record", CooldownUntil: time.Now().UTC().Add(5 * time.Minute).Format(time.RFC3339),
		DataMissing: []string{}, Evidence: []Evidence{{Source: "get_symbol_analysis_context", Finding: "信息不足"}},
	}
	raw, _ := json.Marshal(alert)
	if _, err := validator.Validate(context.Background(), raw); err == nil {
		t.Fatal("expected missing context data to fail validation")
	}
}

func testInput() Input {
	current := signalservice.NewSignal("evt-1", "BTCUSDT", signalservice.TypeFastMove, signalservice.SeverityHigh, "3m")
	current.SignalID = "sig-1"
	current.Metrics = map[string]float64{"change_percent": 12}
	current.Labels = map[string]string{"direction": "up"}
	current.Evidence = []signalservice.Evidence{{Source: "price_tick", Finding: "3m +12%"}}
	return Input{AlertID: "alt-sig-1", Signal: current}
}

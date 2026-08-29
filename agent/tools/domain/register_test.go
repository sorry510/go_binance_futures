package domain

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	agenttools "go_binance_futures/agent/tools"
)

func TestRegisterReadOnlyRegistersExpectedTools(t *testing.T) {
	registry := agenttools.NewRegistry()
	if err := RegisterReadOnly(registry, Dependencies{}); err != nil {
		t.Fatal(err)
	}
	names := []string{"get_symbol_snapshot", "get_klines", "get_funding_rate", "get_liquidations", "get_market_condition", "scan_symbols", "get_test_strategy_results", "get_strategy_template"}
	for _, name := range names {
		tool, ok := registry.Get(name)
		if !ok {
			t.Fatalf("tool %q not registered", name)
		}
		meta := tool.Metadata()
		if meta.Timeout <= 0 || meta.MaxResultBytes <= 0 || !meta.Idempotent || len(meta.InputSchema) == 0 || len(meta.OutputSchema) == 0 {
			t.Fatalf("incomplete metadata for %s: %+v", name, meta)
		}
	}
}

func TestSymbolSnapshotToolUsesInjectedServiceAndStrictArguments(t *testing.T) {
	registry := agenttools.NewRegistry()
	if err := RegisterReadOnly(registry, Dependencies{GetSymbol: func(_ context.Context, value string) (any, error) {
		return map[string]string{"symbol": strings.ToUpper(value)}, nil
	}}); err != nil {
		t.Fatal(err)
	}
	tool, _ := registry.Get("get_symbol_snapshot")
	value, err := tool.Execute(context.Background(), json.RawMessage(`{"symbol":"btcusdt"}`))
	if err != nil {
		t.Fatal(err)
	}
	if value.(map[string]string)["symbol"] != "BTCUSDT" {
		t.Fatalf("unexpected value: %#v", value)
	}
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"symbol":"BTCUSDT","unknown":1}`)); err == nil {
		t.Fatal("unknown argument should be rejected")
	}
}

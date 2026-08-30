package mcpserver

import (
	"context"
	"testing"

	"go_binance_futures/models"
	liquidationservice "go_binance_futures/service/liquidation"
	symbolservice "go_binance_futures/service/symbol"
)

func TestExecuteAPIRequestDefaultUsesSymbolService(t *testing.T) {
	original := listSymbolsForMCP
	defer func() { listSymbolsForMCP = original }()
	listSymbolsForMCP = func(_ context.Context, opts symbolservice.ListOptions) (symbolservice.ListResult, error) {
		if opts.Symbol != "BTCUSDT" || opts.Limit != 5 {
			t.Fatalf("unexpected opts: %+v", opts)
		}
		return symbolservice.ListResult{Total: 1, List: []models.Symbols{{Symbol: "BTCUSDT"}}}, nil
	}
	result, err := executeAPIRequestDefault(context.Background(), APIToolDefinition{Name: "futures_symbols_list"}, APIToolInput{Query: map[string]any{"symbol": "BTCUSDT", "limit": float64(5)}}, "Bearer token")
	if err != nil {
		t.Fatal(err)
	}
	if result.StatusCode != 200 {
		t.Fatalf("status=%d", result.StatusCode)
	}
	body := result.Body.(map[string]any)
	if body["code"] != 200 || body["msg"] != "success" {
		t.Fatalf("unexpected body: %#v", body)
	}
}

func TestExecuteAPIRequestDefaultUsesLiquidationService(t *testing.T) {
	original := listLiquidationsForMCP
	defer func() { listLiquidationsForMCP = original }()
	listLiquidationsForMCP = func(_ context.Context, opts liquidationservice.ListOptions) (liquidationservice.ListResult, error) {
		if opts.Symbol != "BTCUSDT" || opts.MinNotional != 10000 {
			t.Fatalf("unexpected opts: %+v", opts)
		}
		return liquidationservice.ListResult{Total: 2}, nil
	}
	_, err := executeAPIRequestDefault(context.Background(), APIToolDefinition{Name: "futures_liquidation_orders_list"}, APIToolInput{Query: map[string]any{"symbol": "BTCUSDT", "min_notional": float64(10000)}}, "Bearer token")
	if err != nil {
		t.Fatal(err)
	}
}

func TestExecuteAPIRequestDefaultStillRequiresAuthorization(t *testing.T) {
	_, err := executeAPIRequestDefault(context.Background(), APIToolDefinition{Name: "futures_symbols_list"}, APIToolInput{}, "")
	if err == nil {
		t.Fatal("missing authorization should be rejected")
	}
}

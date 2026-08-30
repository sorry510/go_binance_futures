package strategy

import (
	"testing"

	"go_binance_futures/models"
)

func TestCalculateTestResultMetricsPreservesExistingFormula(t *testing.T) {
	result := TestResult{TestStrategyResults: models.TestStrategyResults{Price: "100", PositionAmt: "2", ClosePrice: "110", Leverage: 3}}
	profit := CalculateTestResultMetrics(&result)
	if profit != 20 || result.CloseProfit != "20.000" || result.ProfitPercent != "27.273" {
		t.Fatalf("unexpected metrics: %+v profit=%v", result, profit)
	}
}

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

func TestCalculateTestResultMetricsIncludesFees(t *testing.T) {
	result := TestResult{TestStrategyResults: models.TestStrategyResults{
		Price: "100", PositionAmt: "2", ClosePrice: "110", Leverage: 3,
		OpenFeeRate: 0.001, CloseFeeRate: 0.001,
	}}
	profit := CalculateTestResultMetrics(&result)
	if profit != 19.58 {
		t.Fatalf("unexpected net profit: %v", profit)
	}
	if result.GrossProfit != "20.000" || result.TotalFee != "0.420" || result.CloseProfit != "19.580" {
		t.Fatalf("unexpected metrics: %+v", result)
	}
}

func TestCalculateTestResultMetricsLegacyFeeDefaultsToZero(t *testing.T) {
	result := TestResult{TestStrategyResults: models.TestStrategyResults{Price: "100", PositionAmt: "2", ClosePrice: "110", Leverage: 3}}
	profit := CalculateTestResultMetrics(&result)
	if profit != 20 || result.TotalFee != "0.000" || result.CloseProfit != "20.000" {
		t.Fatalf("legacy row should preserve old result: %+v profit=%v", result, profit)
	}
}

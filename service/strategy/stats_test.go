package strategy

import (
	"math"
	"testing"

	"go_binance_futures/models"
)

func TestCalculateTestResultReviewStatsSeparatesStrategyVersions(t *testing.T) {
	rows := []TestResult{
		{TestStrategyResults: models.TestStrategyResults{
			ID: 1, Symbol: "AAAUSDT", Price: "100", Leverage: 3, PositionAmt: "1", PositionSide: "LONG", ClosePrice: "110",
			OpenFeeRate: 0.001, CloseFeeRate: 0.001, StrategyTemplateID: 7, StrategyTemplateName: "trend",
			StrategySnapshotHash: "snapshot-a", OpenStrategyName: "entry", OpenStrategyType: "long", OpenStrategyHash: "rule-a",
			CloseStrategyName: "exit", CloseStrategyType: "close_long", CloseStrategyHash: "exit-rule",
		}},
		{TestStrategyResults: models.TestStrategyResults{
			ID: 2, Symbol: "BBBUSDT", Price: "100", Leverage: 3, PositionAmt: "-1", PositionSide: "SHORT", ClosePrice: "110",
			OpenFeeRate: 0.001, CloseFeeRate: 0.001, StrategyTemplateID: 7, StrategyTemplateName: "trend",
			StrategySnapshotHash: "snapshot-b", OpenStrategyName: "entry", OpenStrategyType: "short", OpenStrategyHash: "rule-b",
			CloseStrategyName: "exit", CloseStrategyType: "close_short", CloseStrategyHash: "exit-rule-b",
		}},
	}

	stats := CalculateTestResultReviewStats(rows)
	if stats.Total != 2 || stats.Closed != 2 || stats.Open != 0 || stats.Wins != 1 || stats.Losses != 1 || stats.WinRate != 50 {
		t.Fatalf("unexpected summary stats: %+v", stats)
	}
	if math.Abs(stats.GrossProfit-0) > 1e-9 || math.Abs(stats.Fees-0.42) > 1e-9 || math.Abs(stats.NetProfit-(-0.42)) > 1e-9 {
		t.Fatalf("unexpected profit stats: %+v", stats)
	}
	if len(stats.ByTemplate) != 2 {
		t.Fatalf("same template with different snapshot hashes must be separate versions: %+v", stats.ByTemplate)
	}
	if len(stats.ByOpenStrategy) != 2 {
		t.Fatalf("open strategy versions must remain separate: %+v", stats.ByOpenStrategy)
	}
	if len(stats.ByCloseStrategy) != 2 {
		t.Fatalf("close strategy versions must remain separate: %+v", stats.ByCloseStrategy)
	}
}

func TestCalculateTestResultReviewStatsKeepsOpenTradesOutOfRealizedProfit(t *testing.T) {
	rows := []TestResult{
		{TestStrategyResults: models.TestStrategyResults{
			ID: 1, Symbol: "AAAUSDT", Price: "100", Leverage: 3, PositionAmt: "1", PositionSide: "LONG", ClosePrice: "0",
			OpenStrategyName: "entry", OpenStrategyType: "long", OpenStrategyHash: "rule-a",
		}},
	}
	stats := CalculateTestResultReviewStats(rows)
	if stats.Total != 1 || stats.Open != 1 || stats.Closed != 0 || stats.NetProfit != 0 || stats.Fees != 0 {
		t.Fatalf("open trade polluted realized stats: %+v", stats)
	}
	if len(stats.ByOpenStrategy) != 1 || stats.ByOpenStrategy[0].Open != 1 || stats.ByOpenStrategy[0].Closed != 0 {
		t.Fatalf("unexpected open rule stats: %+v", stats.ByOpenStrategy)
	}
}

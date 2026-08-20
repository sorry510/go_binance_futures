package feature

import (
	"go_binance_futures/models"
	"testing"
)

func TestUpdateMarketConditionWithProgressInManualMode(t *testing.T) {
	systemConfig := models.Config{
		MarketCondition:       3,
		MarketConditionIsAuto: 0,
	}
	var progressEvents []MarketConditionProgress
	result, err := UpdateMarketConditionWithProgress(&systemConfig, func(progress MarketConditionProgress) {
		progressEvents = append(progressEvents, progress)
	})
	if err != nil {
		t.Fatalf("update market condition: %v", err)
	}
	if result.MarketCondition != 3 || result.Source != "manual" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(progressEvents) != 2 {
		t.Fatalf("unexpected progress event count: %d", len(progressEvents))
	}
	if progressEvents[0].Stage != "waiting" || progressEvents[0].Progress != 5 {
		t.Fatalf("unexpected first progress event: %+v", progressEvents[0])
	}
	if progressEvents[1].Stage != "completed" || progressEvents[1].Progress != 100 {
		t.Fatalf("unexpected final progress event: %+v", progressEvents[1])
	}
}

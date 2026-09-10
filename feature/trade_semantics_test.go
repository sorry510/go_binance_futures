package feature

import "testing"

func TestResolveTradeROIThresholdsMatchesLiveDefaults(t *testing.T) {
	if disabledTradeROIThreshold != 1_000_000 {
		t.Fatalf("disabled ROI threshold=%v, want 1000000", disabledTradeROIThreshold)
	}
	profit, loss := resolveTradeROIThresholds("0", "0")
	if profit != disabledTradeROIThreshold || loss != disabledTradeROIThreshold {
		t.Fatalf("zero thresholds must be disabled: profit=%v loss=%v", profit, loss)
	}
	profit, loss = resolveTradeROIThresholds("10", "6")
	if profit != 10 || loss != 6 {
		t.Fatalf("configured thresholds changed: profit=%v loss=%v", profit, loss)
	}
}

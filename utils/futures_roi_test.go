package utils

import "testing"

func TestFuturesLeveragedROIMatchesLiveFormula(t *testing.T) {
	if got := FuturesLeveragedROI(1, 1, 101, 10); got < 9.9009 || got > 9.9011 {
		t.Fatalf("long ROI=%v", got)
	}
	if got := FuturesLeveragedROI(1, 1, 99, 10); got < 10.1009 || got > 10.1011 {
		t.Fatalf("short-equivalent ROI=%v", got)
	}
	if got := FuturesLeveragedROI(1, 0, 100, 10); got != 0 {
		t.Fatalf("zero quantity ROI=%v", got)
	}
}

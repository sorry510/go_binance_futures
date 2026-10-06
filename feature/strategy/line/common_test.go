package line

import "testing"

func TestAutoStopNeutralROI(t *testing.T) {
	for _, value := range []float64{-2.999, 0, 2.999} {
		if !autoStopNeutralROI(value) {
			t.Fatalf("ROI %.3f should be inside neutral band", value)
		}
	}
	for _, value := range []float64{-10, -3, 3, 10} {
		if autoStopNeutralROI(value) {
			t.Fatalf("ROI %.3f should allow reversal evaluation", value)
		}
	}
}

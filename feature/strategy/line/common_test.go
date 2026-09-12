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

func TestBaseMarketBreadthPermissionsUsesPercentages(t *testing.T) {
	tests := []struct {
		name                string
		rise, fall, total   int
		btc                 float64
		wantLong, wantShort bool
	}{
		{"balanced", 50, 50, 100, 0, true, true},
		{"broad rally", 75, 25, 100, 0, true, false},
		{"broad selloff", 25, 75, 100, 0, false, true},
		{"btc strong rally", 60, 40, 100, 6, true, false},
		{"btc strong selloff", 40, 60, 100, -6, false, true},
		{"missing universe", 0, 0, 0, 0, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLong, gotShort := baseMarketBreadthPermissions(tt.rise, tt.fall, tt.total, tt.btc)
			if gotLong != tt.wantLong || gotShort != tt.wantShort {
				t.Fatalf("got long=%v short=%v, want long=%v short=%v", gotLong, gotShort, tt.wantLong, tt.wantShort)
			}
		})
	}
}

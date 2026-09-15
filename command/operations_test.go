package command

import (
	"testing"

	"go_binance_futures/service/systemhealth"
)

func TestDoctorExitCode(t *testing.T) {
	for _, tc := range []struct {
		status string
		want   int
	}{
		{systemhealth.StatusHealthy, 0},
		{systemhealth.StatusWarning, 0},
		{systemhealth.StatusDisabled, 0},
		{systemhealth.StatusError, 1},
	} {
		if got := DoctorExitCode(systemhealth.Report{Overall: tc.status}); got != tc.want {
			t.Fatalf("DoctorExitCode(%q)=%d want %d", tc.status, got, tc.want)
		}
	}
}

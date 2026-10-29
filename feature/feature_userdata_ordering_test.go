package feature

import "testing"

func TestFuturesFullSnapshotNeverRewindsNewerWSUpdates(t *testing.T) {
	const snapshotStartedAt int64 = 1_800_000_000_000
	cases := []struct {
		name     string
		updateAt int64
		apply    bool
	}{
		{"old-row", snapshotStartedAt - 3000, true},
		{"boundary", snapshotStartedAt - futuresSnapshotClockSkewAllowanceMS, true},
		{"within-clock-skew", snapshotStartedAt - 1000, false},
		{"same-time", snapshotStartedAt, false},
		{"newer-ws-position", snapshotStartedAt + 1000, false},
		{"newer-ws-order", snapshotStartedAt + 5000, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldApplyFuturesFullSnapshot(tc.updateAt, snapshotStartedAt); got != tc.apply {
				t.Fatalf("should apply=%v, got %v", tc.apply, got)
			}
		})
	}
}

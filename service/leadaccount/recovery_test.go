package leadaccount

import (
	"errors"
	"go_binance_futures/service/futuresownership"
	"testing"
)

func TestStage44ClassifyLeadRecovery(t *testing.T) {
	cases := []struct {
		name, status, id    string
		qty, avg            float64
		terminal, uncertain bool
	}{
		{"full", "FILLED", "lead_a", 1, 100, true, false},
		{"partial_cancel", "CANCELED", "lead_a", 0.4, 100, true, false},
		{"reject", "REJECTED", "lead_a", 0, 0, true, false},
		{"algo_trigger_wait", "NOT_TRIGGERED", "lead_a", 0, 0, false, false},
		{"algo_trigger", "TRIGGERED", "lead_a", 0, 0, false, false},
		{"pending_cancel", "PENDING_CANCEL", "lead_a", 0, 0, false, false},
		{"mismatch", "FILLED", "other", 1, 100, false, true},
		{"unknown", "UNKNOWN", "lead_a", 0, 0, false, true},
		{"missing_avg", "FILLED", "lead_a", 1, 0, false, true},
		{"missing_id", "FILLED", "lead_a", 0, 0, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			id := "123"
			if tc.name == "missing_id" {
				id = ""
			}
			e, err := ClassifyLeadRecovery(futuresownership.ExchangeOrder{ExchangeOrderID: id, ClientOrderID: tc.id, Status: tc.status, FilledQty: tc.qty, AveragePrice: tc.avg}, "lead_a")
			if errors.Is(err, ErrLeadRecoveryUncertain) != tc.uncertain || e.Terminal != tc.terminal || !e.RequiresReconcile {
				t.Fatalf("unexpected %+v err=%v", e, err)
			}
		})
	}
}

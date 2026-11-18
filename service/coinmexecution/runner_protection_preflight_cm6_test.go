package coinmexecution

import (
	"context"
	"testing"
)

func TestCM6InvalidProtectionConfigurationCannotAutoOpen(t *testing.T) {
	for _, tc := range []struct {
		name, profit, loss, tick string
		leverage                 int
	}{
		{"stop_disabled", "8", "0", "0.1", 4},
		{"take_profit_disabled", "0", "6", "0.1", 4},
		{"stop_threshold_beyond_inverse_bound", "8", "400", "0.1", 4},
		{"take_profit_threshold_beyond_inverse_bound", "400", "6", "0.1", 4},
		{"missing_tick_size", "8", "6", "0", 4},
		{"zero_leverage", "8", "6", "0.1", 0},
		{"invalid_negative_threshold", "8", "-6", "0.1", 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner, _, api, strategy, db, _ := cm6RunnerFixture(t)
			if _, err := db.Raw("UPDATE delivery_symbols SET profit=?, loss=?, tickSize=?, leverage=? WHERE symbol='BTCUSD_PERP'",
				tc.profit, tc.loss, tc.tick, tc.leverage).Exec(); err != nil {
				t.Fatal(err)
			}
			if err := runner.Tick(context.Background()); err == nil {
				t.Fatalf("unsafe TP/SL config %+v still authorized a signal or opening", tc)
			}
			if strategy.entries != 0 || len(api.calls) != 0 {
				t.Fatalf("DAPI post or strategy evaluation happened before auto-entry protection preflight: %+v", tc)
			}
			expectCM5(t, db, "SELECT COUNT(*) FROM coin_m_managed_orders", 0)
		})
	}
}

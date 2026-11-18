package coinmexecution

import (
	"context"
	"testing"
	"time"
)

// CM-6 F7: a protection snapshot verified BEFORE strategy evaluation must
// never authorize POST after a concurrent change to TP/SL/tick/leverage.
func TestCM6ProtectionEditedDuringEntrySignalCannotOpen(t *testing.T) {
	cases := []struct {
		name, updates string
		values        []interface{}
	}{
		{"disable_both_protectors", "profit=?,loss=?,tickSize=?", []interface{}{"0", "0", "0"}},
		{"disable_stop", "loss=?", []interface{}{"0"}},
		{"disable_take_profit", "profit=?", []interface{}{"0"}},
		{"invalidate_tick", "tickSize=?", []interface{}{"0"}},
		{"invalidate_leverage", "leverage=?", []interface{}{0}},
		{"change_valid_take_profit", "profit=?", []interface{}{"9"}},
		{"change_valid_stop", "loss=?", []interface{}{"7"}},
		{"change_valid_tick", "tickSize=?", []interface{}{"0.2"}},
		{"change_valid_leverage", "leverage=?", []interface{}{5}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runner, _, remote, strategy, db, clock := cm6RunnerFixture(t)
			var changeErr error
			strategy.onEntry = func() {
				args := append(append([]interface{}{}, tc.values...), "BTCUSD_PERP")
				_, changeErr = db.Raw("UPDATE delivery_symbols SET "+tc.updates+" WHERE symbol=?", args...).Exec()
			}
			if err := runner.Tick(context.Background()); err != nil {
				t.Fatalf("old snapshot with concurrent protection edit should be skipped without POST: %v", err)
			}
			if changeErr != nil {
				t.Fatal(changeErr)
			}
			if strategy.entries != 1 || len(remote.calls) != 0 {
				t.Fatalf("%s: stale protection config triggered exchange order: signals=%d posts=%d",
					tc.name, strategy.entries, len(remote.calls))
			}
			expectCM5(t, db, "SELECT COUNT(*) FROM coin_m_managed_orders WHERE intent='OPEN'", 0)
			expectCM5(t, db, "SELECT COUNT(*) FROM coin_m_managed_positions", 0)

			// Following tick must reevaluate the fresh config, not replay the old
			// signal. Restore valid fields for unsafe cases so both paths are tested.
			if _, err := db.Raw("UPDATE delivery_symbols SET profit='8', loss='6', tickSize='0.1', leverage=4 WHERE symbol='BTCUSD_PERP'").Exec(); err != nil {
				t.Fatal(err)
			}
			strategy.onEntry = nil
			*clock = clock.Add(11 * time.Second)
			if err := runner.Tick(context.Background()); err != nil {
				t.Fatal(err)
			}
			if len(remote.calls) != 1 {
				t.Fatalf("%s: refreshed valid protection failed to create exactly one new order: %d", tc.name, len(remote.calls))
			}
			expectCM5(t, db, "SELECT COUNT(*) FROM coin_m_managed_orders WHERE intent='OPEN'", 1)
		})
	}
}
func TestCM6ProtectionRacedToInvalidRemainsBlockedNextTick(t *testing.T) {
	runner, _, remote, strategy, db, clock := cm6RunnerFixture(t)
	var changeErr error
	strategy.onEntry = func() {
		_, changeErr = db.Raw("UPDATE delivery_symbols SET profit='0',loss='0',tickSize='0' WHERE symbol='BTCUSD_PERP'").Exec()
	}
	if err := runner.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if changeErr != nil {
		t.Fatal(changeErr)
	}
	strategy.onEntry = nil
	*clock = clock.Add(11 * time.Second)
	if err := runner.Tick(context.Background()); err == nil {
		t.Fatal("new signal with disabled protective orders was accepted")
	}
	if len(remote.calls) != 0 {
		t.Fatalf("raced or current invalid STOP/TP allowed real order POST: %+v", remote.calls)
	}
	expectCM5(t, db, "SELECT COUNT(*) FROM coin_m_managed_orders", 0)
}

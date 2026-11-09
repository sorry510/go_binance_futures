package orderhistory

import (
	"context"
	"errors"
	"testing"
	"time"

	"go_binance_futures/models"
)

func TestHistoryPhysicalDeleteRemovesOnlyMatchingTerminalCycle(t *testing.T) {
	o, store := historyDB(t)
	store.Now = func() time.Time { return time.UnixMilli(1893456000000).UTC() }
	id := fullCycle(t, o)
	// This WS Fill was saved before the Position ID association arrived.
	// Deleting the exact Managed order must also clear its orphaned Fill.
	if _, err := o.Raw("UPDATE futures_trade_fills SET position_id=NULL WHERE account_id='main' AND trade_id='1'").Exec(); err != nil {
		t.Fatal(err)
	}
	other := positionFixture(t, o, "main", "auto_strategy", "ETHUSDT", "LONG", "closed", "legacy", true)
	lead := positionFixture(t, o, "lead", "auto_strategy", "BTCUSDT", "LONG", "closed", "legacy", true)
	active := positionFixture(t, o, "main", "auto_strategy", "SOLUSDT", "LONG", "active", "open", false)
	// Other account and neighboring cycles must never be affected by deleting BTC.
	_, err := o.Insert(&models.FuturesOwnershipAdjustment{
		PositionID: id, AccountID: "main", Symbol: "BTCUSDT", PositionSide: "LONG", Owner: "auto_strategy", Operator: "operator", Reason: "test"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = o.Insert(&models.OrderHistoryAdminAudit{
		PositionID: id, AccountID: "main", Action: "update", Operator: "operator", Reason: "test"})
	if err != nil {
		t.Fatal(err)
	}
	req := PhysicalDeleteRequest{Filters: Filter{AccountID: "main", Symbol: "BTC", Page: 3, Limit: 10}, ExpectedCount: 1}
	result, err := store.DeleteMatching(context.Background(), req)
	if err != nil {
		t.Fatalf("delete closed settled cycle: %v", err)
	}
	if result.Positions != 1 || result.Orders != 3 || result.Fills != 4 {
		t.Fatalf("wrong deletion counts %+v", result)
	}
	checks := []struct {
		table string
		want  int
	}{
		{"futures_managed_positions", 3}, {"futures_managed_orders", 0}, {"futures_trade_fills", 0},
		{"futures_ownership_adjustments", 0}, {"order_history_admin_audits", 0},
	}
	for _, tc := range checks {
		var n int
		if e := o.Raw("SELECT COUNT(*) FROM " + tc.table).QueryRow(&n); e != nil || n != tc.want {
			t.Fatalf("table %s remaining %d want %d err=%v", tc.table, n, tc.want, e)
		}
	}
	for _, pid := range []int64{other, lead, active} {
		var n int
		if e := o.Raw("SELECT COUNT(*) FROM futures_managed_positions WHERE id=?", pid).QueryRow(&n); e != nil || n != 1 {
			t.Fatalf("unrelated position %d removed count=%d err=%v", pid, n, e)
		}
	}
	if _, err = store.Details(context.Background(), "main", id); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted cycle remains readable: %v", err)
	}
}

func TestHistoryPhysicalDeleteRejectsLiveAndUnsettledAndRecent(t *testing.T) {
	testCases := []struct {
		name, status, settlement string
		quantity                 float64
		closed                   int64
		orderStatus              string
		now                      int64
		want                     error
	}{
		{"active", "active", "open", 1, 0, "", 1893456000000, ErrDeleteNotTerminal},
		{"incomplete", "closed", "incomplete", 0, 1800000005000, "", 1893456000000, ErrDeleteNotTerminal},
		{"closed_remaining", "closed", "settled", 0.1, 1800000005000, "", 1893456000000, ErrDeleteNotTerminal},
		{"recent", "closed", "settled", 0, 1800000005000, "", 1800000010000, ErrDeleteNotTerminal},
		{"pending_protection", "closed", "settled", 0, 1800000005000, "submitted", 1893456000000, ErrDeleteNotTerminal},
		{"unknown_order_state", "closed", "settled", 0, 1800000005000, "unrecognized_status", 1893456000000, ErrDeleteNotTerminal},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			o, store := historyDB(t)
			store.Now = func() time.Time { return time.UnixMilli(tc.now).UTC() }
			id := positionFixture(t, o, "main", "auto_strategy", "BTCUSDT", "LONG", tc.status, tc.settlement, true)
			if tc.status != "active" {
				if _, err := o.Raw("UPDATE futures_managed_positions SET managed_qty=?,closed_at=? WHERE id=?", tc.quantity, tc.closed, id).Exec(); err != nil {
					t.Fatal(err)
				}
			}
			if tc.orderStatus != "" {
				managedOrder(t, o, id, "main", "auto_strategy", "BTCUSDT", "LONG", "unfinished-order", "stop_loss", "125", 0)
				if _, err := o.Raw("UPDATE futures_managed_orders SET status=? WHERE position_id=?", tc.orderStatus, id).Exec(); err != nil {
					t.Fatal(err)
				}
			}
			_, err := store.DeleteMatching(context.Background(), PhysicalDeleteRequest{Filters: Filter{AccountID: "main"}, ExpectedCount: 1})
			if !errors.Is(err, tc.want) {
				t.Fatalf("wrong safety rejection: %v want %v", err, tc.want)
			}
			var count int
			if e := o.Raw("SELECT COUNT(*) FROM futures_managed_positions WHERE id=?", id).QueryRow(&count); e != nil || count != 1 {
				t.Fatalf("unsafe delete occurred: count %d error %v", count, e)
			}
		})
	}
}
func TestHistoryPhysicalDeleteChecksFillRecencyAndScopeSnapshot(t *testing.T) {
	o, store := historyDB(t)
	id := fullCycle(t, o)
	store.Now = func() time.Time { return time.UnixMilli(1800000010000).UTC() }
	// The position close timestamp is old relative to test Now, but a recent
	// trade recorded at the cutoff must also be protected.
	if _, err := store.DeleteMatching(context.Background(), PhysicalDeleteRequest{
		Filters: Filter{AccountID: "main"}, ExpectedCount: 1}); !errors.Is(err, ErrDeleteNotTerminal) {
		t.Fatalf("recent close bypassed retention guard: %v", err)
	}
	store.Now = func() time.Time { return time.UnixMilli(1893456000000).UTC() }
	if _, err := store.DeleteMatching(context.Background(), PhysicalDeleteRequest{
		Filters: Filter{AccountID: "main"}, ExpectedCount: 2}); !errors.Is(err, ErrDeleteChanged) {
		t.Fatalf("unexpected matching count was accepted: %v", err)
	}
	// Recent actual trade time, independent of the old close timestamp.
	store.Now = func() time.Time { return time.UnixMilli(1893456000000).UTC() }
	_, err := o.Raw("UPDATE futures_trade_fills SET trade_time=? WHERE position_id=?", 1893456000000-24*60*60*1000, id).Exec()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.DeleteMatching(context.Background(), PhysicalDeleteRequest{
		Filters: Filter{AccountID: "main"}, ExpectedCount: 1}); !errors.Is(err, ErrDeleteCooldown) {
		t.Fatalf("recent Fill bypassed retention guard: %v", err)
	}
}
func TestHistoryPhysicalDeleteRollbackWhenChildDeletionFails(t *testing.T) {
	o, store := historyDB(t)
	store.Now = func() time.Time { return time.UnixMilli(1893456000000).UTC() }
	id := fullCycle(t, o)
	if _, err := o.Raw(`CREATE TRIGGER reject_fill_delete BEFORE DELETE ON futures_trade_fills
  BEGIN SELECT RAISE(ABORT,'simulated DB failure'); END`).Exec(); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DeleteMatching(context.Background(), PhysicalDeleteRequest{
		Filters: Filter{AccountID: "main"}, ExpectedCount: 1}); err == nil {
		t.Fatal("missing DB error / delete must rollback")
	}
	var p, orders, fills int
	for _, tc := range []struct {
		query string
		out   *int
	}{
		{"SELECT COUNT(*) FROM futures_managed_positions WHERE id=?", &p},
		{"SELECT COUNT(*) FROM futures_managed_orders WHERE position_id=?", &orders},
		{"SELECT COUNT(*) FROM futures_trade_fills WHERE position_id=?", &fills},
	} {
		if err := o.Raw(tc.query, id).QueryRow(tc.out); err != nil {
			t.Fatal(err)
		}
	}
	if p != 1 || orders != 3 || fills != 4 {
		t.Fatalf("partial delete left inconsistent tables position=%d orders=%d fills=%d", p, orders, fills)
	}
}

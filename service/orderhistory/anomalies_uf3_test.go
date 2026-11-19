package orderhistory

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"go_binance_futures/models"
)

func uf3Fixture(t *testing.T) (Store, int64) {
	t.Helper()
	db, store := historyDB(t)
	if _, err := db.Raw(`CREATE TABLE IF NOT EXISTS agent_trade_proposals
		(proposal_id TEXT PRIMARY KEY,symbol TEXT,side TEXT,status TEXT,updated_at BIGINT)`).Exec(); err != nil {
		t.Fatal(err)
	}
	positionID := positionFixture(t, db, "main", "auto_strategy", "BTCUSDT", "LONG", "reconcile_required", "open", false)
	// Human history overrides have absolutely no effect on live anomalies.
	if _, err := db.Raw("UPDATE futures_managed_positions SET history_status_override='closed',history_completeness_override='verified' WHERE id=?", positionID).Exec(); err != nil {
		t.Fatal(err)
	}
	ownOrder := managedOrder(t, db, positionID, "main", "auto_strategy", "BTCUSDT", "LONG", "uf3-open-reconcile", "open", "uf3-ex-1", 2)
	if _, err := db.Raw("UPDATE futures_managed_orders SET status='reconcile_required' WHERE id=?", ownOrder.ID).Exec(); err != nil {
		t.Fatal(err)
	}
	// No cycle ID, no exchange order: genuine unknown OPEN intent, not a
	// fabricated position. It must still be discoverable in one click.
	orphan := &models.FuturesManagedOrder{
		AccountID: "main", Owner: "auto_strategy", Symbol: "ETHUSDT", PositionSide: "SHORT",
		Intent: "open", ClientOrderID: "uf3-orphan-unknown", Status: "reconcile_required",
		RequestedQty: 1, UpdatedAt: 1800000003000, CreatedAt: 1800000000000,
	}
	if _, err := db.Insert(orphan); err != nil {
		t.Fatal(err)
	}
	// Separate Lead account must not leak under Main filtering.
	lead := &models.FuturesManagedOrder{AccountID: "lead", Owner: "agent_trade", Symbol: "SOLUSDT", PositionSide: "LONG", Intent: "close", ClientOrderID: "uf3-lead", Status: "reconcile_required", UpdatedAt: 1800000009000, CreatedAt: 1800000000000}
	if _, err := db.Insert(lead); err != nil {
		t.Fatal(err)
	}
	// An unfinished closed Managed cycle (different from legacy records).
	positionFixture(t, db, "main", "auto_strategy", "XRPUSDT", "SHORT", "closed", "incomplete", true)
	// The Agent proposal source is an independent status; never label it as
	// a Managed order, even when its symbol matches a managed cycle.
	for _, entry := range []struct{ id, status string }{
		{"uf3-proposal-uncertain", "execution_uncertain"},
		{"uf3-proposal-protect", "protection_failed"},
	} {
		if _, err := db.Raw("INSERT INTO agent_trade_proposals(proposal_id,symbol,side,status,updated_at) VALUES(?,?,?,?,?)", entry.id, "BTCUSDT", "LONG", entry.status, int64(1800000007000)).Exec(); err != nil {
			t.Fatal(err)
		}
	}
	return store, positionID
}
func TestUF3AnomalyAccountIsolationAndOrphanOpen(t *testing.T) {
	store, pid := uf3Fixture(t)
	ctx := context.Background()
	page, err := store.Anomalies(ctx, AnomalyFilter{AccountID: "main", Page: 1, Limit: 25})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 6 || len(page.List) != 6 {
		t.Fatalf("Main expected 5 independent source rows, got %+v", page)
	}
	seen := map[string]bool{}
	for _, entry := range page.List {
		if entry.AccountID != "main" {
			t.Fatalf("Lead row leaked into Main: %+v", entry)
		}
		seen[entry.Category] = true
		if entry.ClientOrderID == "uf3-orphan-unknown" {
			if entry.PositionID != nil || entry.ManagedOrderID == nil || entry.Source != "managed_order" || entry.Symbol != "ETHUSDT" {
				t.Fatalf("Orphan OPEN missing independent anomaly record: %+v", entry)
			}
		}
		if entry.Source == "managed_position" && (entry.PositionID == nil || *entry.PositionID != pid) {
			// There is also a closed/incomplete position.
			if entry.Category != "settlement_incomplete" {
				t.Fatalf("wrong position ID: %+v", entry)
			}
		}
		if entry.Source == "agent_proposal" {
			if entry.Owner != "agent_trade" || entry.ProposalID == "" || entry.PositionID != nil || entry.ClientOrderID != "" {
				t.Fatalf("Agent proposal incorrectly inferred Managed ownership: %+v", entry)
			}
		}
	}
	for _, kind := range []string{"managed_position_reconcile", "managed_order_reconcile", "settlement_incomplete", "proposal_execution_uncertain", "proposal_protection_failed"} {
		if !seen[kind] {
			t.Fatalf("anomaly category %s missing from %+v", kind, page.List)
		}
	}
	lead, err := store.Anomalies(ctx, AnomalyFilter{AccountID: "lead", Page: 1, Limit: 10})
	if err != nil || lead.Total != 1 || lead.List[0].AccountID != "lead" || lead.List[0].ClientOrderID != "uf3-lead" {
		t.Fatalf("Lead account filtering broken: %+v, %v", lead, err)
	}
}
func TestUF3AnomalyCategoryAndOwnerFiltering(t *testing.T) {
	store, _ := uf3Fixture(t)
	cases := []struct {
		kind, owner, symbol, side string
		count                     int64
	}{
		{"managed_order_reconcile", "auto_strategy", "", "", 2},
		{"managed_order_reconcile", "auto_strategy", "ETHUSDT", "SHORT", 1},
		{"managed_position_reconcile", "agent_trade", "", "", 0},
		{"proposal_execution_uncertain", "", "BTCUSDT", "LONG", 1},
		{"proposal_protection_failed", "", "BTCUSDT", "SHORT", 0},
		{"settlement_incomplete", "auto_strategy", "XRPUSDT", "SHORT", 1},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%s_%s_%s", tc.kind, tc.symbol, tc.side), func(t *testing.T) {
			result, err := store.Anomalies(context.Background(), AnomalyFilter{AccountID: "main", Owner: tc.owner, Symbol: tc.symbol, PositionSide: tc.side, Category: tc.kind, Page: 1, Limit: 10})
			if err != nil || result.Total != tc.count || len(result.List) != int(tc.count) {
				t.Fatalf("bad count for %+v: %+v err=%v", tc, result, err)
			}
		})
	}
}
func TestUF3HealthSummaryReconcileLinkCombinesPositionsAndOrders(t *testing.T) {
	store, _ := uf3Fixture(t)
	result, err := store.Anomalies(context.Background(), AnomalyFilter{
		AccountID: "main", Category: "reconcile_required", Page: 1, Limit: 10,
	})
	if err != nil || result.Total != 3 || len(result.List) != 3 {
		t.Fatalf("health reconcile link must include position and order anomalies: %+v %v", result, err)
	}
	for _, item := range result.List {
		if item.Category != "managed_position_reconcile" && item.Category != "managed_order_reconcile" {
			t.Fatalf("health reconcile incorrectly included agent proposal or incomplete settlement: %+v", item)
		}
	}
}

func TestUF3AnomalyRealRESTFillPendingMatchesHealthClassification(t *testing.T) {
	db, store := historyDB(t)
	if _, err := db.Raw("CREATE TABLE IF NOT EXISTS agent_trade_proposals (proposal_id TEXT PRIMARY KEY,symbol TEXT,side TEXT,status TEXT,updated_at BIGINT)").Exec(); err != nil {
		t.Fatal(err)
	}
	id := positionFixture(t, db, "main", "auto_strategy", "BTCUSDT", "LONG", "active", "open", false)
	order := managedOrder(t, db, id, "main", "auto_strategy", "BTCUSDT", "LONG", "uf3-verified-later", "open", "uf3-ex-verify", 2)
	if _, err := db.Raw("UPDATE futures_managed_orders SET status='reconcile_required' WHERE id=?", order.ID).Exec(); err != nil {
		t.Fatal(err)
	}
	fill(t, db, "main", "BTCUSDT", "LONG", "uf3-ex-verify", "uf3-trade", "uf3-verified-later", "open", "2", "100", "0", "0", "USDT", id, false)
	page, err := store.Anomalies(context.Background(), AnomalyFilter{AccountID: "main", Category: "fill_verification_pending", Page: 1, Limit: 10})
	if err != nil || page.Total != 1 || len(page.List) != 1 || page.List[0].ClientOrderID != "uf3-verified-later" {
		t.Fatalf("pending REST Fill not classified as health-compatible: %+v %v", page, err)
	}
	if _, err = db.Raw("UPDATE futures_trade_fills SET qty='1' WHERE trade_id='uf3-trade'").Exec(); err != nil {
		t.Fatal(err)
	}
	page, err = store.Anomalies(context.Background(), AnomalyFilter{AccountID: "main", Category: "managed_order_reconcile", Page: 1, Limit: 10})
	if err != nil || page.Total != 1 || page.List[0].ClientOrderID != "uf3-verified-later" {
		t.Fatalf("mismatched Fill should be hard reconcile, not soft pending: %+v %v", page, err)
	}
}
func TestUF3AnomalyPagedStableAndStaleOnlyAfterTenMinutes(t *testing.T) {
	db, store := historyDB(t)
	if _, err := db.Raw("CREATE TABLE IF NOT EXISTS agent_trade_proposals (proposal_id TEXT PRIMARY KEY,symbol TEXT,side TEXT,status TEXT,updated_at BIGINT)").Exec(); err != nil {
		t.Fatal(err)
	}
	now := store.Now()
	for i, delta := range []time.Duration{-11 * time.Minute, -9 * time.Minute} {
		row := &models.FuturesManagedOrder{AccountID: "main", Owner: "auto_strategy", Symbol: "BTCUSDT", PositionSide: "LONG", Intent: "open", ClientOrderID: fmt.Sprintf("uf3-stale-%d", i), Status: "submitted", UpdatedAt: now.Add(delta).UnixMilli(), CreatedAt: now.Add(delta).UnixMilli()}
		if _, err := db.Insert(row); err != nil {
			t.Fatal(err)
		}
	}
	first, err := store.Anomalies(context.Background(), AnomalyFilter{Page: 1, Limit: 1})
	if err != nil || first.Total != 1 || len(first.List) != 1 || first.List[0].ClientOrderID != "uf3-stale-0" {
		t.Fatalf("stale logic or paging wrong: %+v %v", first, err)
	}
	second, err := store.Anomalies(context.Background(), AnomalyFilter{Page: 2, Limit: 1})
	if err != nil || second.Total != 1 || len(second.List) != 0 {
		t.Fatalf("out-of-range page expected empty list: %+v %v", second, err)
	}
}
func TestUF3AnomalyInvalidFiltersFailBeforeAnySQL(t *testing.T) {
	for _, q := range []AnomalyFilter{
		{Page: 0, Limit: 20}, {Page: 1, Limit: 101},
		{Page: 1, Limit: 20, AccountID: "admin"},
		{Page: 1, Limit: 20, Owner: "auto_strategy' OR 1=1"},
		{Page: 1, Limit: 20, Symbol: "BTC%"},
		{Page: 1, Limit: 20, PositionSide: "BOTH"},
		{Page: 1, Limit: 20, Category: "not_a_real_class"},
	} {
		_, err := (Store{}).Anomalies(context.Background(), q)
		if !errors.Is(err, ErrInvalidQuery) {
			t.Fatalf("invalid filters reached DB: %+v err=%v", q, err)
		}
	}
}
func TestUF3AnomalyCanceledReadIsNotSuccessfulEmpty(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := (Store{}).Anomalies(ctx, AnomalyFilter{Page: 1, Limit: 20})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("context canceled return is not explicit: %v", err)
	}
}

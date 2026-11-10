//go:build spot_legacy_doctor

package spotstrategy

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

func preflightFixture(t *testing.T) (*fakeS2Client, func() LegacyPreflight) {
	t.Helper()
	setupSpotS1(t)
	db, err := orm.GetDB("default")
	if err != nil {
		t.Fatal(err)
	}
	fake := newFakeS2()
	run := func() LegacyPreflight {
		report, err := InspectLegacySpotRisk(context.Background(), db, fake)
		if err != nil {
			t.Fatalf("inspect failed: %v", err)
		}
		if !report.ReadComplete || report.ReadyForLive || !LegacyPreflightBlocked(report, nil) {
			t.Fatalf("preflight gave unauthorized readiness %+v", report)
		}
		if fake.calls != 0 || fake.canceled != 0 {
			t.Fatal("readonly check called POST/DELETE")
		}
		return report
	}
	return fake, run
}
func hasBlocker(items []string, key string) bool {
	for _, v := range items {
		if v == key {
			return true
		}
	}
	return false
}
func TestS3PreflightNeverApprovesEvenEmptyCleanAccount(t *testing.T) {
	f, run := preflightFixture(t)
	f.free = "0"
	f.baseFree = "0"
	report := run()
	if len(report.Balances) != 0 || len(report.OpenOrders) != 0 || len(report.LocalPending) != 0 ||
		!hasBlocker(report.Blockers, "operator_approval_missing") ||
		!hasBlocker(report.Blockers, "legacy_exit_ownership_unverified") ||
		!hasBlocker(report.Blockers, "signed_response_provenance_unverified") {
		t.Fatalf("empty legacy snapshot improperly authorized: %+v", report)
	}
	if report.Mode != ModeLegacy || report.EvidenceSource != "unattested_readonly_snapshot" || report.ObservedAt <= 0 {
		t.Fatalf("snapshot missing audit context: %+v", report)
	}
}
func TestS3PreflightShowsUnownedBalancesAndForeignOrders(t *testing.T) {
	f, run := preflightFixture(t)
	f.baseFree = "2"
	f.unmanaged = []OrderSnapshot{
		{Symbol: "ETHUSDT", ClientOrderID: "external-1", OrderID: "501", Status: "NEW"},
		{Symbol: "BTCUSDT", ClientOrderID: "external-2", OrderID: "502", Status: "NEW"},
	}
	report := run()
	if len(report.Balances) != 2 || report.Balances[0].Asset != "BTC" ||
		len(report.OpenOrders) != 2 || report.OpenOrders[0].Ownership != "unmanaged" ||
		!hasBlocker(report.Blockers, "spot_assets_require_ownership_review") ||
		!hasBlocker(report.Blockers, "external_open_orders") {
		t.Fatalf("external assets or orders were silently adopted: %+v", report)
	}
}
func insertPreflightOrder(t *testing.T, id, exchangeID, status string) {
	t.Helper()
	o := orm.NewOrm()
	_, err := o.Raw(`INSERT INTO spot_strategy_orders
 (account_id,owner,symbol,side,order_type,intent,client_order_id,exchange_order_id,signal_key,
 requested_qty,filled_qty,executed_quote_qty,reserved_qty,status,created_at,updated_at)
 VALUES ('main','auto_strategy','BTCUSDT','BUY','LIMIT','open',?,? ,?,
 '1','0','0','100',?,?,?)`,
		id, exchangeID, "preflight:"+id, status, time.Now().UnixMilli(), time.Now().UnixMilli()).Exec()
	if err != nil {
		t.Fatal(err)
	}
}
func TestS3PreflightKnownAndConflictingOrdersNeverClearGate(t *testing.T) {
	f, run := preflightFixture(t)
	insertPreflightOrder(t, "owned-client", "901", StateNew)
	f.unmanaged = []OrderSnapshot{{Symbol: "BTCUSDT", ClientOrderID: "owned-client", OrderID: "901", Status: "NEW"}}
	report := run()
	if len(report.OpenOrders) != 1 || report.OpenOrders[0].Ownership != "local_strategy_pending" ||
		len(report.LocalPending) != 1 || !hasBlocker(report.Blockers, "strategy_pending_orders") {
		t.Fatalf("known pending Spot order classified incorrectly: %+v", report)
	}
	f.unmanaged[0].OrderID = "999"
	report = run()
	if report.OpenOrders[0].Ownership != "identity_conflict" || !hasBlocker(report.Blockers, "exchange_order_identity_conflict") {
		t.Fatalf("exchange order ID changed but no risk: %+v", report)
	}
	f.unmanaged[0].OrderID = "901"
	if _, err := orm.NewOrm().Raw("UPDATE spot_strategy_orders SET status='filled' WHERE client_order_id='owned-client'").Exec(); err != nil {
		t.Fatal(err)
	}
	report = run()
	if report.OpenOrders[0].Ownership != "terminal_conflict" || !hasBlocker(report.Blockers, "terminal_order_still_open") {
		t.Fatalf("terminal locally filled but still exchange open: %+v", report)
	}
}
func TestS3PreflightLocalHoldingAndModeOverrideStillBlocked(t *testing.T) {
	f, run := preflightFixture(t)
	_ = f
	now := time.Now().UnixMilli()
	_, err := orm.NewOrm().Raw(`INSERT INTO spot_strategy_holdings
 (account_id,owner,symbol,base_asset,quote_asset,owned_qty,avg_cost_quote,status,active_slot,created_at,updated_at)
 VALUES ('main','auto_strategy','BTCUSDT','BTC','USDT','1','100','active','main:auto_strategy:BTCUSDT',?,?)`, now, now).Exec()
	if err != nil {
		t.Fatal(err)
	}
	report := run()
	if len(report.Holdings) != 1 || !hasBlocker(report.Blockers, "strategy_holding_requires_review") {
		t.Fatalf("local Holding silently treated safe: %+v", report)
	}
	if _, err := orm.NewOrm().Raw("UPDATE config SET spot_auto_trade_mode='strategy'").Exec(); err != nil {
		t.Fatal(err)
	}
	report = run()
	if report.Mode != ModeStrategy || !hasBlocker(report.Blockers, "legacy_mode_not_active") || report.ReadyForLive {
		t.Fatalf("direct DB mode change released preflight: %+v", report)
	}
}
func TestS3PreflightReadFailureMustNotLookLikeZeroPositions(t *testing.T) {
	f, _ := preflightFixture(t)
	db, err := orm.GetDB("default")
	if err != nil {
		t.Fatal(err)
	}
	f.balancesErr = errors.New("offline-balance")
	report, err := InspectLegacySpotRisk(context.Background(), db, f)
	if !errors.Is(err, ErrReconciliation) || report.ReadComplete || !LegacyPreflightBlocked(report, err) {
		t.Fatalf("balance failure interpreted empty safe: %+v %v", report, err)
	}
	f.balancesErr = nil
	f.openErr = errors.New("offline-orders")
	report, err = InspectLegacySpotRisk(context.Background(), db, f)
	if !errors.Is(err, ErrReconciliation) || report.ReadComplete || !LegacyPreflightBlocked(report, err) {
		t.Fatalf("open order failure interpreted empty safe: %+v %v", report, err)
	}
	f.openErr = nil
	f.free = "-10"
	report, err = InspectLegacySpotRisk(context.Background(), db, f)
	if !errors.Is(err, ErrReconciliation) || report.ReadComplete {
		t.Fatalf("negative balance accepted: %+v %v", report, err)
	}
	f.free = "0"
	if _, err := orm.NewOrm().Raw("UPDATE config SET spot_auto_trade_mode='unexpected'").Exec(); err != nil {
		t.Fatal(err)
	}
	report, err = InspectLegacySpotRisk(context.Background(), db, f)
	if !errors.Is(err, ErrReconciliation) || report.ReadComplete {
		t.Fatalf("invalid config mode accepted: %+v %v", report, err)
	}
}
func TestS3PreflightDoesNotCallUnrelatedClientMethods(t *testing.T) {
	f, run := preflightFixture(t)
	f.free = "0"
	report := run()
	if !strings.Contains(strings.Join(report.Blockers, ","), "legacy_exit_ownership_unverified") {
		t.Fatal("preflight failed to record legacy obligations")
	}
	if f.calls != 0 || f.canceled != 0 || f.queries != 0 {
		t.Fatalf("read-only preflight invoked trading or per-order action %+v", f)
	}
}

func TestS3PreflightOtherOwnerLocalRecordsBlockRelease(t *testing.T) {
	_, run := preflightFixture(t)
	now := time.Now().UnixMilli()
	o := orm.NewOrm()
	_, err := o.Raw(`INSERT INTO spot_strategy_orders
 (account_id,owner,symbol,side,order_type,intent,client_order_id,signal_key,
 requested_qty,filled_qty,executed_quote_qty,reserved_qty,status,created_at,updated_at)
 VALUES ('main','manual','BTCUSDT','BUY','LIMIT','open','other-owner','preflight:other',
 '1','0','0','100','new',?,?)`, now, now).Exec()
	if err != nil {
		t.Fatal(err)
	}
	_, err = o.Raw(`INSERT INTO spot_strategy_holdings
 (account_id,owner,symbol,base_asset,quote_asset,owned_qty,avg_cost_quote,status,created_at,updated_at)
 VALUES ('main','manual','BTCUSDT','BTC','USDT','1','100','active',?,?)`, now, now).Exec()
	if err != nil {
		t.Fatal(err)
	}
	report := run()
	if !hasBlocker(report.Blockers, "other_owner_local_orders") ||
		!hasBlocker(report.Blockers, "other_owner_local_holdings") {
		t.Fatalf("other owner local Spot records silently ignored %+v", report)
	}
}
func TestS3PreflightDuplicateOrImpossibleExchangeOpenOrderRejected(t *testing.T) {
	f, _ := preflightFixture(t)
	db, err := orm.GetDB("default")
	if err != nil {
		t.Fatal(err)
	}
	f.unmanaged = []OrderSnapshot{
		{Symbol: "BTCUSDT", ClientOrderID: "duplicate", OrderID: "701", Status: "NEW"},
		{Symbol: "BTCUSDT", ClientOrderID: "duplicate", OrderID: "702", Status: "NEW"},
	}
	report, err := InspectLegacySpotRisk(context.Background(), db, f)
	if !errors.Is(err, ErrReconciliation) || report.ReadComplete {
		t.Fatalf("duplicate exchange client identity accepted %+v %v", report, err)
	}
	f.unmanaged = []OrderSnapshot{{Symbol: "BTCUSDT", ClientOrderID: "external", OrderID: "700", Status: "FILLED"}}
	report, err = InspectLegacySpotRisk(context.Background(), db, f)
	if err != nil || !hasBlocker(report.Blockers, "exchange_open_status_invalid") ||
		report.ReadyForLive {
		t.Fatalf("filled order inside OPEN response accepted %+v %v", report, err)
	}
}

func TestS3PreflightForgedReadyFlagCannotBecomeLiveApproval(t *testing.T) {
	fabricated := LegacyPreflight{AccountID: AccountMain, Mode: ModeStrategy,
		ReadComplete: true, ReadyForLive: true, EvidenceSource: "forged",
		Blockers: []string{}}
	if !LegacyPreflightBlocked(fabricated, nil) {
		t.Fatal("caller-generated ReadyForLive flag bypassed approval")
	}
}

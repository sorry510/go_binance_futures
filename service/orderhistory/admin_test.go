package orderhistory

import (
	"context"
	"errors"
	"testing"

	"go_binance_futures/models"
)

func TestHistoryBulkUpdateOnlyPresentation(t *testing.T) {
	o, store := historyDB(t)
	first := fullCycle(t, o)
	_ = positionFixture(t, o, "lead", "auto_strategy", "BTCUSDT", "LONG", "closed", "legacy", true)
	req := BulkRequest{Filters: Filter{AccountID: "main", Status: "closed"}, Action: "update",
		StatusOverride: "reconcile_required", CompletenessOverride: "manual_reviewed", Reason: "operator inspected incomplete metadata"}
	preview, err := store.Preview(context.Background(), req)
	if err != nil || preview.Count != 1 {
		t.Fatalf("preview %v %+v", err, preview)
	}
	req.ExpectedDigest = preview.Digest
	result, err := store.Apply(context.Background(), req, "operator")
	if err != nil || result.Updated != 1 {
		t.Fatalf("update %v %+v", err, result)
	}
	page, err := store.List(context.Background(), Filter{AccountID: "main", Page: 1, Limit: 20})
	if err != nil || page.Total != 1 {
		t.Fatalf("list %+v %v", page, err)
	}
	row := page.List[0]
	if row.Status != "reconcile_required" || row.SystemStatus != "closed" ||
		row.Completeness != "manual_reviewed" || row.SystemCompleteness != "verified" ||
		row.NetPnL != nil || len(page.PageVerifiedNet) != 0 {
		t.Fatalf("manual override leaked into verified profit %+v", row)
	}
	var real models.FuturesManagedPosition
	if err = o.QueryTable(new(models.FuturesManagedPosition)).Filter("id", first).One(&real); err != nil || real.Status != "closed" || real.SettlementStatus != "settled" {
		t.Fatalf("real trading state was changed %+v %v", real, err)
	}
	reset := BulkRequest{Filters: Filter{AccountID: "main", Status: "closed"}, Action: "update",
		ClearStatus: true, ClearCompleteness: true, Reason: "return to system calculated statuses"}
	preview, err = store.Preview(context.Background(), reset)
	if err != nil {
		t.Fatal(err)
	}
	reset.ExpectedDigest = preview.Digest
	if _, err = store.Apply(context.Background(), reset, "operator"); err != nil {
		t.Fatal(err)
	}
	page, err = store.List(context.Background(), Filter{AccountID: "main", Page: 1, Limit: 20})
	if err != nil || page.Total != 1 || page.List[0].Status != "closed" || page.List[0].Completeness != "verified" {
		t.Fatalf("reset did not restore calculated history %+v %v", page, err)
	}
	var audits int
	if err = o.Raw("SELECT COUNT(*) FROM order_history_admin_audits WHERE account_id='main' AND position_id=?", first).QueryRow(&audits); err != nil || audits != 2 {
		t.Fatalf("audit mismatch %d %v", audits, err)
	}
}
func TestHistoryBulkUpdateRejectsForgedVerifiedOrMissingConfirmation(t *testing.T) {
	_, store := historyDB(t)
	if _, err := store.Preview(context.Background(), BulkRequest{Action: "update",
		CompletenessOverride: "verified", Filters: Filter{AccountID: "main"}}); !errors.Is(err, ErrInvalidQuery) {
		t.Fatalf("forged verified accepted: %v", err)
	}
	if _, err := store.Apply(context.Background(), BulkRequest{Action: "update", StatusOverride: "closed",
		Filters: Filter{AccountID: "main"}, Reason: "manual operator review"}, "operator"); !errors.Is(err, ErrInvalidQuery) {
		t.Fatalf("missing edit confirmation accepted: %v", err)
	}
}
func TestHistoryBulkUpdateSnapshotAndAuditRollback(t *testing.T) {
	o, store := historyDB(t)
	first := fullCycle(t, o)
	req := BulkRequest{Action: "update", StatusOverride: "suspended",
		Filters: Filter{AccountID: "main"}, Reason: "manual review of view only"}
	preview, err := store.Preview(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	req.ExpectedDigest = preview.Digest
	_ = positionFixture(t, o, "main", "auto_strategy", "ETHUSDT", "SHORT", "closed", "legacy", true)
	if _, err = store.Apply(context.Background(), req, "operator"); !errors.Is(err, ErrBulkConflict) {
		t.Fatalf("changed scope was accepted: %v", err)
	}
	req.Filters.Symbol = "BTC"
	preview, err = store.Preview(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	req.ExpectedDigest = preview.Digest
	if _, err = o.Raw("DROP TABLE order_history_admin_audits").Exec(); err != nil {
		t.Fatal(err)
	}
	if _, err = store.Apply(context.Background(), req, "operator"); err == nil {
		t.Fatal("lost audit must abort update")
	}
	var status string
	if err = o.Raw("SELECT history_status_override FROM futures_managed_positions WHERE id=?", first).QueryRow(&status); err != nil || status != "" {
		t.Fatalf("partial edit after audit failure status=%s err=%v", status, err)
	}
}

package spotstrategy

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

type s3ReviewConfiguredFake struct{ *fakeS2Client }

func (s3ReviewConfiguredFake) SpotSignedRESTProvenance() bool { return true }

func TestS3F1ManualModeAndEnvCannotAuthorizeLiveWrites(t *testing.T) {
	e, _ := prepareS2(t)
	t.Setenv(liveApprovalVariable, liveApprovalValue)
	o := orm.NewOrm()
	if _, err := o.Raw("UPDATE config SET spot_auto_trade_mode='strategy'").Exec(); err != nil {
		t.Fatal(err)
	}
	if mode, err := CurrentMode(); err != nil || mode != ModeStrategy {
		t.Fatalf("review bypass setup: %s %v", mode, err)
	}
	if LiveOrdersApproved() {
		t.Fatal("env plus direct mode UPDATE released trading")
	}
	if err := AuthorizeLiveMode(); !errors.Is(err, ErrLiveWriteDisabled) {
		t.Fatalf("env plus direct mode UPDATE became authorization: %v", err)
	}
	req := requestS2("s3-review-gate")
	id, err := e.Prepare(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	req.ClientOrderID = id.ClientOrderID
	if _, err = o.Raw("UPDATE spot_strategy_orders SET status='submitting' WHERE id=?", id.ID).Exec(); err != nil {
		t.Fatal(err)
	}
	if err := AuthorizeLiveWrite(context.Background(), req, false); !errors.Is(err, ErrLiveWriteDisabled) {
		t.Fatalf("env+mode+persisted valid order bypassed release Gate: %v", err)
	}
}

func TestS3F5PersistentOrderIdentityAndStatusMatrix(t *testing.T) {
	e, _ := prepareS2(t)
	o := orm.NewOrm()
	req := requestS2("s3-review-auth")
	booked, err := e.Prepare(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	req.ClientOrderID = booked.ClientOrderID
	if _, err = o.Raw("UPDATE spot_strategy_orders SET status='submitting' WHERE id=?", booked.ID).Exec(); err != nil {
		t.Fatal(err)
	}
	if err := validatePersistedSpotWrite(context.Background(), req, false); err != nil {
		t.Fatalf("valid persisted create identity unexpectedly rejected: %v", err)
	}
	for _, bad := range []struct {
		name   string
		modify func(*OrderRequest)
	}{
		{"wrong_account", func(r *OrderRequest) { r.AccountID = "lead" }},
		{"wrong_owner", func(r *OrderRequest) { r.Owner = "manual" }},
		{"wrong_client", func(r *OrderRequest) { r.ClientOrderID = "foreign-cid" }},
		{"wrong_symbol", func(r *OrderRequest) { r.Symbol = "ETHUSDT" }},
		{"wrong_side", func(r *OrderRequest) { r.Side = "SELL" }},
		{"wrong_type", func(r *OrderRequest) { r.Type = "MARKET" }},
		{"wrong_quantity", func(r *OrderRequest) { r.Quantity = "4" }},
		{"invalid_quantity", func(r *OrderRequest) { r.Quantity = "-1" }},
	} {
		t.Run(bad.name, func(t *testing.T) {
			altered := req
			bad.modify(&altered)
			if err := validatePersistedSpotWrite(context.Background(), altered, false); err == nil {
				t.Fatal("invalid order identity passed the persistent DB gate")
			}
		})
	}
	if err := validatePersistedSpotWrite(context.Background(), req, true); err == nil {
		t.Fatal("cannot cancel an order still in submitting")
	}
	if _, err = o.Raw("UPDATE spot_strategy_orders SET status='new' WHERE id=?", booked.ID).Exec(); err != nil {
		t.Fatal(err)
	}
	if err := validatePersistedSpotWrite(context.Background(), req, true); err != nil {
		t.Fatalf("known new order should have cancellable ownership: %v", err)
	}
	if err := validatePersistedSpotWrite(context.Background(), req, false); err == nil {
		t.Fatal("already submitted order authorized a second POST")
	}
	if _, err = o.Raw("UPDATE spot_strategy_orders SET status='reconcile_required' WHERE id=?", booked.ID).Exec(); err != nil {
		t.Fatal(err)
	}
	if err := validatePersistedSpotWrite(context.Background(), req, true); err == nil {
		t.Fatal("quarantined order allowed DELETE")
	}
}

func TestS3F3EndpointAndFakeCannotProvideResponseEvidence(t *testing.T) {
	e, f := prepareS2(t)
	_ = e
	db, err := orm.GetDB("default")
	if err != nil {
		t.Fatal(err)
	}
	live, err := NewLiveEngine(db, s3ReviewConfiguredFake{f})
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(liveApprovalVariable, liveApprovalValue)
	if _, err = db.Exec("UPDATE config SET spot_auto_trade_mode='strategy'"); err != nil {
		t.Fatal(err)
	}
	if canAttestLiveFills(live) {
		t.Fatal("endpoint config/Fake SDK claimed authenticated REST response")
	}
	// The source selected for SDK requests has no per-response transport
	// evidence, and therefore cannot flip rest_verified in S3.
	booked := mustPrepareSubmitS2(t, e, "s3-no-weak-verification")
	f.Lock()
	f.trades = []Trade{tradeS2(999, "1", "100", "100", "0", "USDT")}
	f.order.Status = "PARTIALLY_FILLED"
	f.order.ExecutedQty = "1"
	f.order.ExecutedQuoteQty = "100"
	f.Unlock()
	got, err := live.Reconcile(context.Background(), booked.ID)
	if err != nil || got.FilledQty != "1" {
		t.Fatalf("read-only Fake response: %+v err=%v", got, err)
	}
	var n int
	if err = db.QueryRow("SELECT COUNT(*) FROM spot_strategy_fills WHERE rest_verified<>0").Scan(&n); err != nil || n != 0 {
		t.Fatalf("unauthenticated response became real Fill evidence n=%d err=%v", n, err)
	}
}

func TestS3F4UnverifiedFillsNeverEnterReportNumbers(t *testing.T) {
	setupSpotS1(t)
	db, err := orm.GetDB("default")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UnixMilli()
	inserted, err := db.Exec(`INSERT INTO spot_strategy_holdings
 (account_id,owner,symbol,base_asset,quote_asset,owned_qty,avg_cost_quote,status,active_slot,created_at,updated_at)
 VALUES ('main','auto_strategy','BTCUSDT','BTC','USDT','2','100','active','main:auto_strategy:BTCUSDT',?,?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	hid, err := inserted.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		tradeID, rest             string
		feeAsset, fee, qty, quote string
	}{
		{"301", "1", "BNB", "0.01", "1", "100"},
		{"302", "0", "USDT", "0", "1", "100"},
	} {
		_, err = db.Exec(`INSERT INTO spot_strategy_fills
  (account_id,symbol,order_id,trade_id,holding_id,client_order_id,side,qty,price,quote_qty,commission,commission_asset,trade_time,rest_verified)
  VALUES ('main','BTCUSDT','9001',?,?,'cid','BUY',?,'100',?,?,?, ?,?)`,
			tc.tradeID, hid, tc.qty, tc.quote, tc.fee, tc.feeAsset, now, tc.rest)
		if err != nil {
			t.Fatal(err)
		}
	}
	o := NewMockEngine(db, newFakeS2())
	report, err := o.HoldingReport(context.Background(), hid, "120")
	if err != nil {
		t.Fatal(err)
	}
	if report.Complete || report.CostBasisQuote != "" || report.RealizedGrossQuote != "" ||
		report.RealizedNetQuote != "" || report.UnrealizedEstimateQuote != "" ||
		report.OwnedQty != "" || report.AvgCostQuote != "" {
		t.Fatalf("unverified Fill contaminated financial presentation: %+v", report)
	}
	for _, reason := range []string{"fill_source_not_verified", "third_asset_fee_requires_rate_at_trade_time", "holding_quantity_not_reconciled"} {
		if !strings.Contains(report.IncompleteReason, reason) {
			t.Fatalf("missing incomplete reason %q: %s", reason, report.IncompleteReason)
		}
	}
}

// Regression for repeated partial fills. An intentionally corrupted rounded
// avg_cost_quote cache must NOT distort future real cost calculation: the
// immutable linked Fill prices are always the authoritative source.
func TestS3F6CostBasisDoesNotRoundTripThroughStoredAverage(t *testing.T) {
	e, _ := prepareS2(t)
	order := mustPrepareSubmitS2(t, e, "s3-cost-roundtrip")
	db, err := orm.GetDB("default")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("UPDATE spot_strategy_orders SET status='partially_filled' WHERE id=?", order.ID); err != nil {
		t.Fatal(err)
	}
	var holdingID int64
	for i := 1; i <= 9; i++ {
		quote := 100 + (i % 3)
		_, err = db.Exec(`INSERT INTO spot_strategy_fills
  (account_id,symbol,order_id,trade_id,holding_id,client_order_id,side,qty,price,quote_qty,commission,commission_asset,trade_time,rest_verified)
  VALUES ('main','BTCUSDT','901',?,NULL,?,'BUY','1',?,?, '0','USDT',?,1)`,
			i, order.ClientOrderID, quote, quote, time.Now().UnixMilli()+int64(i))
		if err != nil {
			t.Fatal(err)
		}
		h, e2 := e.ApplyVerifiedFills(context.Background(), order.ID)
		if e2 != nil {
			t.Fatalf("partial verified settlement #%d: %v", i, e2)
		}
		holdingID = h.ID
		if _, err = db.Exec("UPDATE spot_strategy_holdings SET avg_cost_quote='999999' WHERE id=?", holdingID); err != nil {
			t.Fatal(err)
		}
	}
	// Sequence 101,102,100 repeated 3 times: sum quote =909; qty=9,
	// average must be exactly 101, independent of malicious cached avg values.
	h, err := e.ApplyVerifiedFills(context.Background(), order.ID)
	if err != nil || h.OwnedQty != "9" || h.AvgCostQuote != "101" {
		t.Fatalf("roundtrip avg drift or restored cache trusted: %+v %v", h, err)
	}
	report, err := e.HoldingReport(context.Background(), holdingID, "")
	if err != nil || !report.Complete || report.CostBasisQuote != "909" {
		t.Fatalf("cost basis not reconstructed from verified Fill history: %+v %v", report, err)
	}
}

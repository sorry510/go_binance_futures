package futuresownership

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/beego/beego/v2/client/orm"
)

type manualClaimExchange struct {
	rows []AccountPosition
	err  error
}

func (f manualClaimExchange) Positions(context.Context) ([]AccountPosition, error) {
	return f.rows, f.err
}
func (f manualClaimExchange) FreshPositions(context.Context) ([]AccountPosition, error) {
	return f.rows, f.err
}
func manualClaimFixture() ManualClaimInput {
	return ManualClaimInput{
		Symbol: "HYPEUSDT", PositionSide: "SHORT", Owner: OwnerAutoStrategy,
		Quantity: 1.04, ExpectedAccountQty: 1.04,
		Reason:   "verified original strategy position from Binance fills",
		Operator: "test_web_admin",
	}
}
func manualClaimReconciler(rows []AccountPosition) Reconciler {
	s := testService()
	return Reconciler{
		Ownership: s, Executor: Executor{Ownership: s, Broker: &fakeOrderBroker{}},
		Account: manualClaimExchange{rows: rows},
	}
}
func manualHype() []AccountPosition {
	return []AccountPosition{{Symbol: "HYPEUSDT", PositionSide: "SHORT", Quantity: -1.04, EntryPrice: 22.5}}
}

func TestManualClaimUnmanagedPositionAndImmutableAudit(t *testing.T) {
	prepareOwnershipDB(t)
	r := manualClaimReconciler(manualHype())
	claim := manualClaimFixture()
	claim.Quantity = 0.75
	pos, err := r.ManualClaimPosition(context.Background(), claim)
	if err != nil {
		t.Fatal(err)
	}
	if pos.Owner != OwnerAutoStrategy || pos.ManagedQty != 0.75 || pos.EntryPrice != 22.5 || pos.Status != PositionActive ||
		pos.SourceRef == "" || pos.AccountID != "main" {
		t.Fatalf("unexpected manual position: %+v", pos)
	}
	audits, err := r.Ownership.ManualClaims(context.Background(), 10)
	if err != nil || len(audits) != 1 || audits[0].PositionID != pos.ID ||
		audits[0].ManagedQty != 0.75 || audits[0].AccountQty != 1.04 ||
		audits[0].Operator != "test_web_admin" {
		t.Fatalf("missing immutable audit: %+v err=%v", audits, err)
	}
	if _, err := r.ManualClaimPosition(context.Background(), claim); err == nil {
		t.Fatal("repeat claim must not overwrite a live managed position")
	}
	audits, _ = r.Ownership.ManualClaims(context.Background(), 10)
	if len(audits) != 1 {
		t.Fatal("failed attempt must not create audit rows")
	}
}

func TestManualFundingRateClaimCarriesExitEligibilityPrefix(t *testing.T) {
	prepareOwnershipDB(t)
	r := manualClaimReconciler(manualHype())
	q := manualClaimFixture()
	q.Owner = OwnerFundingRate
	pos, err := r.ManualClaimPosition(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(pos.SourceRef, "funding_rate:manual_repair:") {
		t.Fatalf("funding rate position would be skipped by strategy exit: %s", pos.SourceRef)
	}
}

func TestManualClaimRevivesClosedHistoricalOwnership(t *testing.T) {
	prepareOwnershipDB(t)
	s := testService()
	ctx := context.Background()
	claimOpen(t, s, OwnerAutoStrategy, "HYPEUSDT", "SHORT", "historical_hype", 1.04)
	if _, err := s.ApplyFill(ctx, "historical_hype", 1.04, 22.5); err != nil {
		t.Fatal(err)
	}
	old, err := s.ReconcilePosition(ctx, OwnerAutoStrategy, "HYPEUSDT", "SHORT", 0)
	if err != nil || old.Status != PositionClosed {
		t.Fatalf("fixture failed: %+v, %v", old, err)
	}
	r := manualClaimReconciler(manualHype())
	repair := manualClaimFixture()
	repair.Owner = OwnerNewCoinRush
	pos, err := r.ManualClaimPosition(ctx, repair)
	if err != nil {
		t.Fatal(err)
	}
	if pos.ID != old.ID || pos.Owner != OwnerNewCoinRush || pos.ManagedQty != 1.04 {
		t.Fatalf("expected same row reactivated with new owner: %+v", pos)
	}
	audits, err := s.ManualClaims(ctx, 10)
	if err != nil || len(audits) != 1 || audits[0].PreviousStatus != PositionClosed ||
		audits[0].PreviousOwner != OwnerAutoStrategy || audits[0].PreviousSourceRef != old.SourceRef {
		t.Fatalf("expected former ownership in audit: %+v err=%v", audits, err)
	}
}

func TestManualClaimRejectsInvalidEntryPriceBeforeWriting(t *testing.T) {
	for _, price := range []float64{0, -1, math.NaN(), math.Inf(1), 1e-13} {
		t.Run(fmt.Sprintf("%v", price), func(t *testing.T) {
			prepareOwnershipDB(t)
			positions := manualHype()
			positions[0].EntryPrice = price
			r := manualClaimReconciler(positions)
			_, err := r.ManualClaimPosition(context.Background(), manualClaimFixture())
			if !errors.Is(err, ErrManualInvalidEntryPrice) {
				t.Fatalf("price=%v should fail before any claim, got %v", price, err)
			}
			_, err = r.Ownership.claimVerifiedManualPosition(context.Background(), manualClaimFixture(), price)
			if !errors.Is(err, ErrManualInvalidEntryPrice) {
				t.Fatalf("direct mutation price=%v should fail, got %v", price, err)
			}
			audits, auditErr := r.Ownership.ManualClaims(context.Background(), 10)
			if auditErr != nil || len(audits) != 0 {
				t.Fatalf("rejected claim left audit: %+v err=%v", audits, auditErr)
			}
		})
	}
}

func TestManualClaimMissingPositionAndQuantityChangedAreDistinct(t *testing.T) {
	prepareOwnershipDB(t)
	ctx := context.Background()
	missing := manualClaimReconciler(nil)
	if _, err := missing.ManualClaimPosition(ctx, manualClaimFixture()); !errors.Is(err, ErrManualPositionNotFound) {
		t.Fatalf("missing position: got %v", err)
	}
	exists := manualClaimReconciler(manualHype())
	stale := manualClaimFixture()
	stale.ExpectedAccountQty = 2
	if _, err := exists.ManualClaimPosition(ctx, stale); !errors.Is(err, ErrManualAccountQuantityChanged) {
		t.Fatalf("quantity mismatch: got %v", err)
	}
	rows := append(manualHype(), manualHype()[0])
	duplicate := manualClaimReconciler(rows)
	if _, err := duplicate.ManualClaimPosition(ctx, manualClaimFixture()); !errors.Is(err, ErrManualDuplicateExchangePosition) {
		t.Fatalf("duplicate exchange legs must be rejected: %v", err)
	}
	audits, err := duplicate.Ownership.ManualClaims(ctx, 10)
	if err != nil || len(audits) != 0 {
		t.Fatalf("rejected claims must leave no audit: %+v err=%v", audits, err)
	}
}

func TestManualClaimRejectsStaleQtyAndOverclaim(t *testing.T) {
	prepareOwnershipDB(t)
	r := manualClaimReconciler(manualHype())
	q := manualClaimFixture()
	q.ExpectedAccountQty = 1.05
	if _, err := r.ManualClaimPosition(context.Background(), q); err == nil {
		t.Fatal("stale UI quantity accepted")
	}
	q = manualClaimFixture()
	q.Quantity = 1.05
	if _, err := r.ManualClaimPosition(context.Background(), q); err == nil {
		t.Fatal("overclaim accepted")
	}
	q.Quantity = math.NaN()
	if _, err := r.ManualClaimPosition(context.Background(), q); err == nil {
		t.Fatal("NaN accepted")
	}
	q = manualClaimFixture()
	q.Owner = OwnerAgentTrade
	if _, err := r.ManualClaimPosition(context.Background(), q); err == nil {
		t.Fatal("proposal-only owner accepted")
	}
	q = manualClaimFixture()
	q.Reason = "x"
	if _, err := r.ManualClaimPosition(context.Background(), q); err == nil {
		t.Fatal("missing reason accepted")
	}
	zero := manualClaimReconciler(nil)
	if _, err := zero.ManualClaimPosition(context.Background(), manualClaimFixture()); err == nil {
		t.Fatal("absent exchange position accepted")
	}
}

func TestManualClaimBlocksUnresolvedOrderAndExistingLiveOwner(t *testing.T) {
	prepareOwnershipDB(t)
	ctx := context.Background()
	s := testService()
	claimOpen(t, s, OwnerAutoStrategy, "HYPEUSDT", "SHORT", "hype_open_pending", 1.04)
	r := manualClaimReconciler(manualHype())
	if _, err := r.ManualClaimPosition(ctx, manualClaimFixture()); err == nil {
		t.Fatal("pending opening order must block manual assignment")
	}
	// A filled opening order can still own the leg; never override it.
	if _, err := s.ApplyFill(ctx, "hype_open_pending", 1.04, 22.5); err != nil {
		t.Fatal(err)
	}
	if _, err := r.ManualClaimPosition(ctx, manualClaimFixture()); err == nil {
		t.Fatal("active owner accepted")
	}
}

func TestManualClaimRejectsUnresolvedHistoricalOwnership(t *testing.T) {
	prepareOwnershipDB(t)
	ctx := context.Background()
	r := manualClaimReconciler(manualHype())
	s := testService()
	claimOpen(t, s, OwnerAutoStrategy, "HYPEUSDT", "SHORT", "hype_uncertain", 1.04)
	if _, err := s.ApplyFill(ctx, "hype_uncertain", 1.04, 22.5); err != nil {
		t.Fatal(err)
	}
	_, err := orm.NewOrm().QueryTable("futures_managed_positions").
		Filter("account_id", "main").Filter("symbol", "HYPEUSDT").
		Update(orm.Params{"status": PositionReconcileRequired})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.ManualClaimPosition(ctx, manualClaimFixture()); err == nil {
		t.Fatal("reconcile_required owner accepted")
	}
}

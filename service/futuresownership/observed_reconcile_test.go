package futuresownership

import (
	"context"
	"testing"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

func TestObservedZeroDoesNotPermanentlyLoseOwnership(t *testing.T) {
	prepareOwnershipDB(t)
	now := time.UnixMilli(1_800_000_000_000)
	s := Service{Now: func() time.Time { return now }}
	ctx := context.Background()
	claimOpen(t, s, OwnerAutoStrategy, "HYPEUSDT", "SHORT", "aut_hype_owned", 1.04)
	if _, err := s.ApplyFill(ctx, "aut_hype_owned", 1.04, 20); err != nil {
		t.Fatal(err)
	}

	p, err := s.ReconcileObservedPosition(ctx, OwnerAutoStrategy, "HYPEUSDT", "SHORT", 0)
	if err != nil || p.Status != PositionReconcileRequired || p.ManagedQty != 1.04 {
		t.Fatalf("one absent snapshot must not unown the position: %+v err=%v", p, err)
	}
	p, err = s.ReconcileObservedPosition(ctx, OwnerAutoStrategy, "HYPEUSDT", "SHORT", 0)
	if err != nil || p.Status != PositionReconcileRequired || p.ManagedQty != 1.04 {
		t.Fatalf("same-time duplicate zero must not close ownership: %+v err=%v", p, err)
	}

	// Exchange proves the position still exists after a lagging mirror read.
	now = now.Add(3 * time.Second)
	p, err = s.ReconcileObservedPosition(ctx, OwnerAutoStrategy, "HYPEUSDT", "SHORT", 1.04)
	if err != nil || p.Status != PositionActive || p.ManagedQty != 1.04 {
		t.Fatalf("reappearing position must retain the original ownership: %+v err=%v", p, err)
	}
	p, err = s.ReconcileObservedPosition(ctx, OwnerAutoStrategy, "HYPEUSDT", "SHORT", 1.5)
	if err != nil || p.ManagedQty != 1.04 {
		t.Fatalf("manual increase must not expand ownership: %+v err=%v", p, err)
	}
}

func TestObservedZeroClosesOnlyAfterIndependentDelayedConfirmation(t *testing.T) {
	prepareOwnershipDB(t)
	now := time.UnixMilli(1_800_000_000_000)
	s := Service{Now: func() time.Time { return now }}
	ctx := context.Background()
	claimOpen(t, s, OwnerAutoStrategy, "APTUSDT", "SHORT", "aut_apt_owned", 120.3)
	if _, err := s.ApplyFill(ctx, "aut_apt_owned", 120.3, 7); err != nil {
		t.Fatal(err)
	}
	first, err := s.ReconcileObservedPosition(ctx, OwnerAutoStrategy, "APTUSDT", "SHORT", 0)
	if err != nil || first.Status != PositionReconcileRequired {
		t.Fatalf("first zero: %+v err=%v", first, err)
	}
	now = now.Add(11 * time.Second)
	closed, err := s.ReconcileObservedPosition(ctx, OwnerAutoStrategy, "APTUSDT", "SHORT", 0)
	if err != nil || closed.Status != PositionClosed || closed.ManagedQty != 0 {
		t.Fatalf("verified second zero: %+v err=%v", closed, err)
	}
	if _, err := s.GetPosition(ctx, OwnerAutoStrategy, "APTUSDT", "SHORT"); err != orm.ErrNoRows {
		t.Fatalf("truly closed position must leave active ownership: %v", err)
	}
}

func TestReconcilerSingleZeroCannotCloseManagedPosition(t *testing.T) {
	prepareOwnershipDB(t)
	s := testService()
	ctx := context.Background()
	claimOpen(t, s, OwnerAutoStrategy, "HYPEUSDT", "SHORT", "aut_hype_2", 1.04)
	if _, err := s.ApplyFill(ctx, "aut_hype_2", 1.04, 20); err != nil {
		t.Fatal(err)
	}
	r := Reconciler{
		Ownership: s,
		Executor:  Executor{Ownership: s, Broker: &fakeOrderBroker{}},
		Account:   fakeAccountPositionSource{positions: nil},
	}
	got, err := r.ReconcileOwner(ctx, OwnerAutoStrategy)
	if err != nil || got.PositionsClosed != 0 {
		t.Fatalf("single zero snapshot incorrectly closed position: %+v err=%v", got, err)
	}
	p, err := s.GetPosition(ctx, OwnerAutoStrategy, "HYPEUSDT", "SHORT")
	if err != nil || p.Status != PositionReconcileRequired || p.ManagedQty != 1.04 {
		t.Fatalf("managed ownership must remain durable: %+v err=%v", p, err)
	}
	r.Account = fakeAccountPositionSource{positions: []AccountPosition{{Symbol: "HYPEUSDT", PositionSide: "SHORT", Quantity: -1.04}}}
	_, err = r.ReconcileOwner(ctx, OwnerAutoStrategy)
	p, checkErr := s.GetPosition(ctx, OwnerAutoStrategy, "HYPEUSDT", "SHORT")
	if err != nil || checkErr != nil || p.Status != PositionActive {
		t.Fatalf("later correct snapshot must restore active status: %+v errors %v %v", p, err, checkErr)
	}
}

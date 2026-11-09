package futuresownership

import (
	"context"
	"testing"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
)

func mustLinkedOrder(t *testing.T, s Service, id string, wantID int64) models.FuturesManagedOrder {
	t.Helper()
	order, err := s.GetOrder(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if order.PositionID == nil || *order.PositionID != wantID {
		t.Fatalf("order %s position_id=%v, want %d", id, order.PositionID, wantID)
	}
	return order
}

func TestStage011NewCycleOneOpenMultipleCloses(t *testing.T) {
	prepareOwnershipDB(t)
	ctx := context.Background()
	s := testService()
	first := claimOpen(t, s, OwnerAutoStrategy, "BTCUSDT", "LONG", "s510_open_a", 4)
	if first.PositionID != nil {
		t.Fatal("zero-fill OPEN must not have position_id")
	}
	if _, err := s.ApplyFill(ctx, first.ClientOrderID, 0, 0); err != nil {
		t.Fatal(err)
	}
	if order, _ := s.GetOrder(ctx, first.ClientOrderID); order.PositionID != nil {
		t.Fatal("zero-fill receipt allocated a position_id")
	}
	a, err := s.ApplyFill(ctx, first.ClientOrderID, 4, 100)
	if err != nil || a.ID == 0 || a.ManagedQty != 4 {
		t.Fatalf("opening fill: %+v err=%v", a, err)
	}
	mustLinkedOrder(t, s, first.ClientOrderID, a.ID)
	for i, qty := range []float64{1, 2, 1} {
		id := []string{"s510_close_a1", "s510_close_a2", "s510_close_a3"}[i]
		order, err := s.ClaimOrder(ctx, ClaimOrderInput{
			Owner: OwnerAutoStrategy, Symbol: "BTCUSDT", PositionSide: "LONG",
			Intent: IntentClose, ClientOrderID: id, RequestedQty: qty, OrderType: "MARKET",
		})
		if err != nil || order.PositionID == nil || *order.PositionID != a.ID {
			t.Fatalf("close claim %d: %+v err=%v", i, order, err)
		}
		pos, err := s.ApplyFill(ctx, id, qty, 110+float64(i))
		if err != nil || pos.ManagedQty != []float64{3, 1, 0}[i] {
			t.Fatalf("close fill %d: %+v err=%v", i, pos, err)
		}
		if pos.ID != a.ID {
			t.Fatalf("close %d rebound cycle: %+v", i, pos)
		}
	}
	var closed models.FuturesManagedPosition
	if err := orm.NewOrm().QueryTable(new(models.FuturesManagedPosition)).Filter("id", a.ID).One(&closed); err != nil || closed.Status != PositionClosed || closed.ManagedQty != 0 {
		t.Fatalf("previous cycle not closed: %+v err=%v", closed, err)
	}
	second := claimOpen(t, s, OwnerAutoStrategy, "BTCUSDT", "LONG", "s510_open_b", 2)
	b, err := s.ApplyFill(ctx, second.ClientOrderID, 2, 120)
	if err != nil || b.ID == a.ID || b.ManagedQty != 2 {
		t.Fatalf("second cycle must be distinct: %+v err=%v", b, err)
	}
	mustLinkedOrder(t, s, second.ClientOrderID, b.ID)
	// Duplicate old events must never use the newest live position as fallback.
	again, err := s.ApplyFill(ctx, "s510_close_a3", 1, 110)
	if err != nil || again.ID != a.ID || again.Status != PositionClosed {
		t.Fatalf("old close receipt should return original closed cycle: %+v err=%v", again, err)
	}
	live, err := s.GetPosition(ctx, OwnerAutoStrategy, "BTCUSDT", "LONG")
	if err != nil || live.ID != b.ID || live.ManagedQty != 2 {
		t.Fatalf("old close touched new cycle: %+v err=%v", live, err)
	}
}

func TestStage011StaleCloseAndProtectionCannotCloseNextCycle(t *testing.T) {
	prepareOwnershipDB(t)
	ctx := context.Background()
	s := testService()
	open := claimOpen(t, s, OwnerAgentTrade, "ETHUSDT", "SHORT", "s510_old_open", 2)
	a, err := s.ApplyFill(ctx, open.ClientOrderID, 2, 100)
	if err != nil {
		t.Fatal(err)
	}
	close1, err := s.ClaimOrder(ctx, ClaimOrderInput{Owner: OwnerAgentTrade, Symbol: "ETHUSDT", PositionSide: "SHORT", Intent: IntentClose, ClientOrderID: "s510_partial_close", RequestedQty: 2, OrderType: "MARKET"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.ApplyFill(ctx, close1.ClientOrderID, 1, 101); err != nil {
		t.Fatal(err)
	}
	if err = s.SetOrderStatus(ctx, close1.ClientOrderID, OrderCanceled); err != nil {
		t.Fatal(err)
	}
	stop, err := s.ClaimOrder(ctx, ClaimOrderInput{Owner: OwnerAgentTrade, Symbol: "ETHUSDT", PositionSide: "SHORT", Intent: IntentStopLoss, ClientOrderID: "s510_old_sl", RequestedQty: 1, OrderType: "STOP_MARKET"})
	if err != nil {
		t.Fatal(err)
	}
	mustLinkedOrder(t, s, stop.ClientOrderID, a.ID)
	close2, err := s.ClaimOrder(ctx, ClaimOrderInput{Owner: OwnerAgentTrade, Symbol: "ETHUSDT", PositionSide: "SHORT", Intent: IntentClose, ClientOrderID: "s510_finish", RequestedQty: 1, OrderType: "MARKET"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.ApplyFill(ctx, close2.ClientOrderID, 1, 102); err != nil {
		t.Fatal(err)
	}
	next := claimOpen(t, s, OwnerAgentTrade, "ETHUSDT", "SHORT", "s510_new_open", 3)
	b, err := s.ApplyFill(ctx, next.ClientOrderID, 3, 103)
	if err != nil || a.ID == b.ID {
		t.Fatalf("new cycle: %+v %v", b, err)
	}
	if _, err = s.ApplyFill(ctx, close1.ClientOrderID, 2, 104); err == nil {
		t.Fatal("late previous close must fail closed")
	}
	if _, err = s.ApplyFill(ctx, stop.ClientOrderID, 1, 104); err == nil {
		t.Fatal("stale stop must not reduce a new cycle")
	}
	oldOrder, err := s.GetOrder(ctx, close1.ClientOrderID)
	if err != nil || oldOrder.FilledQty != 1 {
		t.Fatalf("failed late close must rollback order qty: %+v %v", oldOrder, err)
	}
	oldStop, err := s.GetOrder(ctx, stop.ClientOrderID)
	if err != nil || oldStop.FilledQty != 0 {
		t.Fatalf("failed old stop must rollback qty: %+v %v", oldStop, err)
	}
	live, err := s.GetPosition(ctx, OwnerAgentTrade, "ETHUSDT", "SHORT")
	if err != nil || live.ID != b.ID || live.ManagedQty != 3 {
		t.Fatalf("stale order touched new cycle: %+v %v", live, err)
	}
}

func TestStage011OpenFillAtomicRollback(t *testing.T) {
	prepareOwnershipDB(t)
	s := testService()
	ctx := context.Background()
	open := claimOpen(t, s, OwnerAutoStrategy, "SOLUSDT", "LONG", "s510_atomic_open", 3)
	o := orm.NewOrm()
	if _, err := o.Raw("CREATE TRIGGER fail_s510_open BEFORE INSERT ON futures_managed_positions BEGIN SELECT RAISE(ABORT,'forced position write error'); END").Exec(); err != nil {
		t.Fatal(err)
	}
	_, err := s.ApplyFill(ctx, open.ClientOrderID, 2, 50)
	if _, dropErr := o.Raw("DROP TRIGGER IF EXISTS fail_s510_open").Exec(); dropErr != nil {
		t.Fatal(dropErr)
	}
	if err == nil {
		t.Fatal("expected forced database failure")
	}
	order, err := s.GetOrder(ctx, open.ClientOrderID)
	if err != nil || order.FilledQty != 0 || order.PositionID != nil {
		t.Fatalf("failed position insert must rollback order fill and association: %+v err=%v", order, err)
	}
	var count int64
	if err := o.Raw("SELECT COUNT(*) FROM futures_managed_positions").QueryRow(&count); err != nil || count != 0 {
		t.Fatalf("failed transaction left position rows: %d err=%v", count, err)
	}
	pos, err := s.ApplyFill(ctx, open.ClientOrderID, 2, 50)
	if err != nil || pos.ManagedQty != 2 {
		t.Fatalf("failed transaction not retryable: %+v err=%v", pos, err)
	}
	mustLinkedOrder(t, s, open.ClientOrderID, pos.ID)
}

func TestStage011DifferentOwnersAndAccountsSeparateCycles(t *testing.T) {
	prepareOwnershipDB(t)
	main := testService()
	lead := Service{AccountID: "lead", Now: main.Now}
	ctx := context.Background()
	m := claimOpen(t, main, OwnerAutoStrategy, "XRPUSDT", "LONG", "s510_main_open", 1)
	l := claimOpen(t, lead, OwnerAutoStrategy, "XRPUSDT", "LONG", "s510_lead_open", 2)
	mp, err := main.ApplyFill(ctx, m.ClientOrderID, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	lp, err := lead.ApplyFill(ctx, l.ClientOrderID, 2, 20)
	if err != nil {
		t.Fatal(err)
	}
	if mp.ID == lp.ID || mp.AccountID != "main" || lp.AccountID != "lead" {
		t.Fatalf("account scope lost: %+v %+v", mp, lp)
	}
	mc, err := main.ClaimOrder(ctx, ClaimOrderInput{Owner: OwnerAutoStrategy, Symbol: "XRPUSDT", PositionSide: "LONG", Intent: IntentClose, ClientOrderID: "s510_main_close", RequestedQty: 1, OrderType: "MARKET"})
	if err != nil || mc.PositionID == nil || *mc.PositionID != mp.ID {
		t.Fatalf("main close wrong account: %+v err=%v", mc, err)
	}
	if _, err := lead.GetOrder(ctx, mc.ClientOrderID); err != orm.ErrNoRows {
		t.Fatalf("lead leaked main order: %v", err)
	}
	if _, err := main.ApplyFill(ctx, mc.ClientOrderID, 1, 11); err != nil {
		t.Fatal(err)
	}
	still, err := lead.GetPosition(ctx, OwnerAutoStrategy, "XRPUSDT", "LONG")
	if err != nil || still.ID != lp.ID || still.ManagedQty != 2 {
		t.Fatalf("main close touched lead: %+v err=%v", still, err)
	}
}

func TestStage011LegacyTerminalUnboundOpenCannotResurrectPosition(t *testing.T) {
	prepareOwnershipDB(t)
	s := testService()
	ctx := context.Background()
	open := claimOpen(t, s, OwnerAutoStrategy, "DOGEUSDT", "LONG", "s510_terminal_legacy", 2)
	if err := s.SetOrderStatus(ctx, open.ClientOrderID, OrderCanceled); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ApplyFill(ctx, open.ClientOrderID, 1, 100); err == nil {
		t.Fatal("legacy terminal order must not create a new managed position")
	}
	order, err := s.GetOrder(ctx, open.ClientOrderID)
	if err != nil || order.PositionID != nil || order.FilledQty != 0 || order.Status != OrderCanceled {
		t.Fatalf("legacy order must remain untouched: %+v err=%v", order, err)
	}
	var positions int64
	if err := orm.NewOrm().Raw("SELECT COUNT(*) FROM futures_managed_positions").QueryRow(&positions); err != nil || positions != 0 {
		t.Fatalf("legacy terminal OPEN created phantom position: %d err=%v", positions, err)
	}
}

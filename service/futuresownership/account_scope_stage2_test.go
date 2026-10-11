package futuresownership

import (
	"context"
	"errors"
	"testing"

	"go_binance_futures/feature/api/binance"
	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
)

func TestStage2TwoAccountsSameSymbolIndependentOwnership(t *testing.T) {
	prepareOwnershipDB(t)
	main := testService()
	lead, err := BindAccount(binance.LeadAccountID)
	if err != nil {
		t.Fatal(err)
	}
	lead.Now = main.Now
	ctx := context.Background()
	m := claimOpen(t, main, OwnerAutoStrategy, "BTCUSDT", "LONG", "main_v2_shared", 2)
	l := claimOpen(t, lead, OwnerAutoStrategy, "BTCUSDT", "LONG", "lead_v2_shared", 3)
	if m.AccountID != "main" || l.AccountID != "lead" {
		t.Fatalf("wrong account: %+v %+v", m, l)
	}
	mp, err := main.ApplyFill(ctx, m.ClientOrderID, 1, 30000)
	if err != nil {
		t.Fatal(err)
	}
	lp, err := lead.ApplyFill(ctx, l.ClientOrderID, 2, 31000)
	if err != nil {
		t.Fatal(err)
	}
	if mp.ManagedQty != 1 || lp.ManagedQty != 2 || lp.AccountID != "lead" {
		t.Fatalf("position collision %+v %+v", mp, lp)
	}
	// A duplicate fill cannot increase quantity in the other account.
	lp, err = lead.ApplyFill(ctx, l.ClientOrderID, 2, 31000)
	if err != nil {
		t.Fatal(err)
	}
	if lp.ManagedQty != 2 {
		t.Fatalf("duplicate cumulative fill changed managed qty: %+v", lp)
	}
	if _, err = main.GetOrder(ctx, l.ClientOrderID); !errors.Is(err, orm.ErrNoRows) {
		t.Fatalf("main can see lead order: %v", err)
	}
	if _, err = lead.GetOrder(ctx, m.ClientOrderID); !errors.Is(err, orm.ErrNoRows) {
		t.Fatalf("lead can see main order: %v", err)
	}
	if _, err = main.GetPosition(ctx, OwnerAutoStrategy, "BTCUSDT", "SHORT"); !errors.Is(err, orm.ErrNoRows) {
		t.Fatalf("unexpected short position %v", err)
	}
	if _, err = lead.ReconcilePosition(ctx, OwnerAutoStrategy, "BTCUSDT", "LONG", 1); err != nil {
		t.Fatal(err)
	}
	mp, err = main.GetPosition(ctx, OwnerAutoStrategy, "BTCUSDT", "LONG")
	if err != nil || mp.ManagedQty != 1 {
		t.Fatalf("lead reconcile touched main: %+v %v", mp, err)
	}
	lp, err = lead.GetPosition(ctx, OwnerAutoStrategy, "BTCUSDT", "LONG")
	if err != nil || lp.ManagedQty != 1 {
		t.Fatalf("lead shrink failed: %+v %v", lp, err)
	}
	if err = lead.SetOrderStatus(ctx, m.ClientOrderID, OrderCanceled); !errors.Is(err, orm.ErrNoRows) {
		t.Fatalf("lead changed main status: %v", err)
	}
	if err = main.SuspendPosition(ctx, OwnerAutoStrategy, "BTCUSDT", "LONG"); err != nil {
		t.Fatal(err)
	}
	lp, err = lead.GetPosition(ctx, OwnerAutoStrategy, "BTCUSDT", "LONG")
	if err != nil || lp.Status != PositionActive {
		t.Fatalf("main suspend touched lead: %+v %v", lp, err)
	}
}
func TestStage2ActiveAndListQueriesNeverMixAccounts(t *testing.T) {
	prepareOwnershipDB(t)
	main := testService()
	lead, err := BindAccount(binance.LeadAccountID)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	claimOpen(t, main, OwnerAutoStrategy, "ETHUSDT", "SHORT", "s2m1", 3)
	claimOpen(t, lead, OwnerAutoStrategy, "ETHUSDT", "SHORT", "s2l1", 3)
	for _, item := range []struct {
		service Service
		want    string
	}{{main, "main"}, {lead, "lead"}} {
		orders, err := item.service.ActiveOrders(ctx, OwnerAutoStrategy)
		if err != nil || len(orders) != 1 || orders[0].AccountID != item.want {
			t.Fatalf("%s active order %v %v", item.want, orders, err)
		}
		orders, err = item.service.ListOrders(ctx, "", 100)
		if err != nil || len(orders) != 1 || orders[0].AccountID != item.want {
			t.Fatalf("%s list order %v %v", item.want, orders, err)
		}
	}
}
func TestStage2UnknownOwnershipAccountFailsClosed(t *testing.T) {
	prepareOwnershipDB(t)
	s := Service{AccountID: "not-an-account"}
	if _, err := s.ListOrders(context.Background(), "", 100); err == nil {
		t.Fatal("unsupported account must fail")
	}
	if _, err := s.ClaimOrder(context.Background(), ClaimOrderInput{Owner: OwnerAutoStrategy, Symbol: "BTCUSDT", PositionSide: "LONG", Intent: IntentOpen, ClientOrderID: "s2invalid", RequestedQty: 1}); err == nil {
		t.Fatal("invalid account must not create order")
	}
}
func TestStage2ExecutorAccountMismatchFailsBeforeExchange(t *testing.T) {
	prepareOwnershipDB(t)
	e := Executor{Ownership: DefaultService(), Broker: BinanceOrderBroker{}}
	if err := e.validateAccountBinding(); err != nil {
		t.Fatal(err)
	}
	e = Executor{Ownership: Service{AccountID: "lead"}, Broker: BinanceOrderBroker{}}
	if err := e.validateAccountBinding(); err == nil {
		t.Fatal("lead broker must be bound")
	}
}
func TestStage2LegacyStoredRowDefaultsMain(t *testing.T) {
	prepareOwnershipDB(t)
	o := orm.NewOrm()
	row := models.FuturesManagedOrder{Owner: OwnerAutoStrategy, Symbol: "AAVEUSDT", PositionSide: "LONG", Intent: IntentOpen, ClientOrderID: "stage2-legacy", RequestedQty: 1, OrderType: "MARKET", Status: OrderPending}
	if _, err := o.Insert(&row); err != nil {
		t.Fatal(err)
	}
	if _, err := o.Raw("UPDATE futures_managed_orders SET account_id='main' WHERE account_id IS NULL OR account_id='' ").Exec(); err != nil {
		t.Fatal(err)
	}
	loaded, err := DefaultService().GetOrder(context.Background(), "stage2-legacy")
	if err != nil || loaded.AccountID != "main" {
		t.Fatalf("legacy defaults failed: %+v %v", loaded, err)
	}
}

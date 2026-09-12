package futuresownership

import (
	"context"
	"errors"
	"testing"

	"github.com/beego/beego/v2/client/orm"
)

type fakeAccountPositionSource struct {
	positions []AccountPosition
	err       error
}

func (f fakeAccountPositionSource) Positions(context.Context) ([]AccountPosition, error) {
	return f.positions, f.err
}

func TestReconcileDoesNotClaimUnmanagedAccountPosition(t *testing.T) {
	prepareOwnershipDB(t)
	service := testService()
	reconciler := Reconciler{
		Ownership: service,
		Executor:  Executor{Ownership: service, Broker: &fakeOrderBroker{}},
		Account: fakeAccountPositionSource{positions: []AccountPosition{
			{Symbol: "BTCUSDT", PositionSide: "LONG", Quantity: 2},
		}},
	}
	summary, err := reconciler.ReconcileOwner(context.Background(), OwnerAutoStrategy)
	if err != nil {
		t.Fatal(err)
	}
	if summary.PositionsChecked != 0 || summary.OrdersChecked != 0 {
		t.Fatalf("unmanaged account state must stay observation-only: %+v", summary)
	}
	if _, err := service.GetPosition(context.Background(), OwnerAutoStrategy, "BTCUSDT", "LONG"); err != orm.ErrNoRows {
		t.Fatalf("manual account position must not be claimed, err=%v", err)
	}
}

func TestReconcileRecoversOnlyRegisteredOrderByClientID(t *testing.T) {
	prepareOwnershipDB(t)
	service := testService()
	claimOpen(t, service, OwnerAgentTrade, "ETHUSDT", "SHORT", "agt_restart", 2)
	broker := &fakeOrderBroker{lookupResult: ExchangeOrder{
		ExchangeOrderID: "901", ClientOrderID: "agt_restart", Status: "FILLED", FilledQty: 2, AveragePrice: 2200,
	}}
	reconciler := Reconciler{
		Ownership: service,
		Executor:  Executor{Ownership: service, Broker: broker},
		Account: fakeAccountPositionSource{positions: []AccountPosition{
			{Symbol: "ETHUSDT", PositionSide: "SHORT", Quantity: -2},
		}},
	}
	summary, err := reconciler.ReconcileOwner(context.Background(), OwnerAgentTrade)
	if err != nil {
		t.Fatal(err)
	}
	if summary.OrdersChecked != 1 || summary.OrdersUnresolved != 0 || broker.lookupCalls != 1 {
		t.Fatalf("registered order must reconcile exactly once: summary=%+v lookups=%d", summary, broker.lookupCalls)
	}
	position, err := service.GetPosition(context.Background(), OwnerAgentTrade, "ETHUSDT", "SHORT")
	if err != nil || position.ManagedQty != 2 {
		t.Fatalf("recovered exchange fill must restore managed position: %+v err=%v", position, err)
	}
}

func TestReconcileShrinksButNeverExpandsManagedQuantity(t *testing.T) {
	prepareOwnershipDB(t)
	service := testService()
	claimOpen(t, service, OwnerAutoStrategy, "SOLUSDT", "LONG", "aut_sol_restart", 5)
	if _, err := service.ApplyFill(context.Background(), "aut_sol_restart", 5, 150); err != nil {
		t.Fatal(err)
	}
	reconciler := Reconciler{Ownership: service, Executor: Executor{Ownership: service, Broker: &fakeOrderBroker{}}, Account: fakeAccountPositionSource{positions: []AccountPosition{{Symbol: "SOLUSDT", PositionSide: "LONG", Quantity: 8}}}}
	if _, err := reconciler.ReconcileOwner(context.Background(), OwnerAutoStrategy); err != nil {
		t.Fatal(err)
	}
	position, _ := service.GetPosition(context.Background(), OwnerAutoStrategy, "SOLUSDT", "LONG")
	if position.ManagedQty != 5 {
		t.Fatalf("manual increase must stay unmanaged, got %.8f", position.ManagedQty)
	}
	reconciler.Account = fakeAccountPositionSource{positions: []AccountPosition{{Symbol: "SOLUSDT", PositionSide: "LONG", Quantity: 3}}}
	summary, err := reconciler.ReconcileOwner(context.Background(), OwnerAutoStrategy)
	if err != nil {
		t.Fatal(err)
	}
	position, _ = service.GetPosition(context.Background(), OwnerAutoStrategy, "SOLUSDT", "LONG")
	if position.ManagedQty != 3 || summary.PositionsShrunk != 1 {
		t.Fatalf("manual reduction must shrink ownership only: position=%+v summary=%+v", position, summary)
	}
}

func TestReconcileKeepsUnknownExchangeResultFailClosed(t *testing.T) {
	prepareOwnershipDB(t)
	service := testService()
	claimOpen(t, service, OwnerNewCoinRush, "BNBUSDT", "LONG", "rush_unknown", 1)
	broker := &fakeOrderBroker{lookupErr: errors.New("exchange unavailable")}
	reconciler := Reconciler{Ownership: service, Executor: Executor{Ownership: service, Broker: broker}, Account: fakeAccountPositionSource{}}
	summary, err := reconciler.ReconcileOwner(context.Background(), OwnerNewCoinRush)
	if err != nil {
		t.Fatal(err)
	}
	if summary.OrdersUnresolved != 1 {
		t.Fatalf("unknown exchange result must remain unresolved: %+v", summary)
	}
	row, err := findTestOrder("rush_unknown")
	if err != nil || row.Status != OrderReconcile {
		t.Fatalf("unknown order must stay reconcile_required: %+v err=%v", row, err)
	}
}

type mapOrderBroker struct {
	orders      map[string]ExchangeOrder
	cancelCalls int
}

func (b *mapOrderBroker) Submit(context.Context, OrderRequest, string) (ExchangeOrder, error) {
	return ExchangeOrder{}, errors.New("unexpected submit")
}
func (b *mapOrderBroker) Lookup(_ context.Context, _ string, clientOrderID, _ string) (ExchangeOrder, error) {
	row, ok := b.orders[clientOrderID]
	if !ok {
		return ExchangeOrder{}, errors.New("not found")
	}
	return row, nil
}
func (b *mapOrderBroker) Cancel(_ context.Context, _ string, _ int64, _ string) error {
	b.cancelCalls++
	return nil
}

func TestReconcileCancelsSiblingProtectionAfterProtectiveFillClosesPosition(t *testing.T) {
	prepareOwnershipDB(t)
	service := testService()
	claimOpen(t, service, OwnerNoticeAutoOrder, "LINKUSDT", "LONG", "notice_entry", 1)
	if _, err := service.ApplyFill(context.Background(), "notice_entry", 1, 10); err != nil {
		t.Fatal(err)
	}
	for _, input := range []ClaimOrderInput{
		{Owner: OwnerNoticeAutoOrder, Symbol: "LINKUSDT", PositionSide: "LONG", Intent: IntentStopLoss, ClientOrderID: "notice_sl", RequestedQty: 1, OrderType: "STOP_MARKET"},
		{Owner: OwnerNoticeAutoOrder, Symbol: "LINKUSDT", PositionSide: "LONG", Intent: IntentTakeProfit, ClientOrderID: "notice_tp", RequestedQty: 1, OrderType: "TAKE_PROFIT_MARKET"},
	} {
		if _, err := service.ClaimOrder(context.Background(), input); err != nil {
			t.Fatal(err)
		}
	}
	broker := &mapOrderBroker{orders: map[string]ExchangeOrder{
		"notice_sl": {ExchangeOrderID: "101", ClientOrderID: "notice_sl", Status: "FILLED", FilledQty: 1, AveragePrice: 9},
		"notice_tp": {ExchangeOrderID: "102", ClientOrderID: "notice_tp", Status: "NEW"},
	}}
	reconciler := Reconciler{Ownership: service, Executor: Executor{Ownership: service, Broker: broker}, Account: fakeAccountPositionSource{}}
	summary, err := reconciler.ReconcileOwner(context.Background(), OwnerNoticeAutoOrder)
	if err != nil {
		t.Fatal(err)
	}
	if summary.PositionsChecked != 0 {
		// The stop fill already closed ownership while active orders were reconciled.
		t.Fatalf("protective fill should close before account-position reconcile: %+v", summary)
	}
	if broker.cancelCalls != 1 {
		t.Fatalf("remaining sibling protection must be canceled, cancel calls=%d summary=%+v", broker.cancelCalls, summary)
	}
	tp, err := findTestOrder("notice_tp")
	if err != nil || tp.Status != OrderCanceled {
		t.Fatalf("sibling protection status=%+v err=%v", tp, err)
	}
}

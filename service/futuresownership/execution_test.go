package futuresownership

import (
	"context"
	"errors"
	"testing"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
)

type fakeOrderBroker struct {
	submitResult ExchangeOrder
	submitErr    error
	lookupResult ExchangeOrder
	lookupErr    error
	submitCalls  int
	lookupCalls  int
}

func (f *fakeOrderBroker) Submit(context.Context, OrderRequest, string) (ExchangeOrder, error) {
	f.submitCalls++
	return f.submitResult, f.submitErr
}

func (f *fakeOrderBroker) Lookup(context.Context, string, string) (ExchangeOrder, error) {
	f.lookupCalls++
	return f.lookupResult, f.lookupErr
}

func (f *fakeOrderBroker) Cancel(context.Context, string, int64) error { return nil }

func TestExecutorPersistsOwnershipBeforeAndAfterFill(t *testing.T) {
	prepareOwnershipDB(t)
	broker := &fakeOrderBroker{submitResult: ExchangeOrder{
		ExchangeOrderID: "123", ClientOrderID: "aut_test", Status: "FILLED", FilledQty: 2, AveragePrice: 100,
	}}
	executor := Executor{Ownership: testService(), Broker: broker}
	result, err := executor.Execute(context.Background(), OrderRequest{
		Owner: OwnerAutoStrategy, Symbol: "BTCUSDT", PositionSide: "LONG", Intent: IntentOpen,
		Side: "BUY", OrderType: "MARKET", Quantity: 2, ClientOrderID: "aut_test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExchangeOrderID != "123" || broker.submitCalls != 1 {
		t.Fatalf("unexpected execution result: %+v calls=%d", result, broker.submitCalls)
	}
	position, err := executor.Ownership.GetPosition(context.Background(), OwnerAutoStrategy, "BTCUSDT", "LONG")
	if err != nil || position.ManagedQty != 2 {
		t.Fatalf("filled quantity must become managed quantity: %+v err=%v", position, err)
	}
}

func TestExecutorRecoversAmbiguousSubmitByClientOrderID(t *testing.T) {
	prepareOwnershipDB(t)
	broker := &fakeOrderBroker{
		submitErr:    errors.New("network timeout"),
		lookupResult: ExchangeOrder{ExchangeOrderID: "456", ClientOrderID: "agt_test", Status: "FILLED", FilledQty: 1, AveragePrice: 200},
	}
	executor := Executor{Ownership: testService(), Broker: broker}
	_, err := executor.Execute(context.Background(), OrderRequest{
		Owner: OwnerAgentTrade, Symbol: "ETHUSDT", PositionSide: "SHORT", Intent: IntentOpen,
		Side: "SELL", OrderType: "MARKET", Quantity: 1, ClientOrderID: "agt_test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if broker.submitCalls != 1 || broker.lookupCalls != 1 {
		t.Fatalf("ambiguous submit must lookup exactly once: submit=%d lookup=%d", broker.submitCalls, broker.lookupCalls)
	}
	position, err := executor.Ownership.GetPosition(context.Background(), OwnerAgentTrade, "ETHUSDT", "SHORT")
	if err != nil || position.ManagedQty != 1 {
		t.Fatalf("reconciled fill must become managed: %+v err=%v", position, err)
	}
}

func TestExecutorLeavesUnknownResultInReconcileRequired(t *testing.T) {
	prepareOwnershipDB(t)
	broker := &fakeOrderBroker{submitErr: errors.New("timeout"), lookupErr: errors.New("not found")}
	executor := Executor{Ownership: testService(), Broker: broker}
	_, err := executor.Execute(context.Background(), OrderRequest{
		Owner: OwnerAutoStrategy, Symbol: "SOLUSDT", PositionSide: "LONG", Intent: IntentOpen,
		Side: "BUY", OrderType: "MARKET", Quantity: 1, ClientOrderID: "aut_unknown",
	})
	if err == nil {
		t.Fatal("uncertain submission must not be treated as success")
	}
	row, loadErr := findTestOrder("aut_unknown")
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	if row.Status != OrderReconcile {
		t.Fatalf("uncertain order status=%s, want %s", row.Status, OrderReconcile)
	}
}

func TestExecutorRequiresConfirmedMarketFill(t *testing.T) {
	prepareOwnershipDB(t)
	broker := &fakeOrderBroker{
		submitResult: ExchangeOrder{ExchangeOrderID: "789", ClientOrderID: "aut_wait_fill", Status: "NEW"},
		lookupResult: ExchangeOrder{ExchangeOrderID: "789", ClientOrderID: "aut_wait_fill", Status: "NEW"},
	}
	executor := Executor{Ownership: testService(), Broker: broker}
	_, err := executor.Execute(context.Background(), OrderRequest{
		Owner: OwnerAutoStrategy, Symbol: "XRPUSDT", PositionSide: "LONG", Intent: IntentOpen,
		Side: "BUY", OrderType: "MARKET", Quantity: 2, ClientOrderID: "aut_wait_fill",
	})
	if err == nil {
		t.Fatal("market order without confirmed fill must remain unresolved")
	}
	row, loadErr := findTestOrder("aut_wait_fill")
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	if row.Status != OrderReconcile || row.FilledQty != 0 {
		t.Fatalf("unconfirmed market fill must stay reconcile_required: %+v", row)
	}
	if broker.submitCalls != 1 || broker.lookupCalls != 1 {
		t.Fatalf("expected one submit and one reconcile lookup: submit=%d lookup=%d", broker.submitCalls, broker.lookupCalls)
	}
	if _, positionErr := executor.Ownership.GetPosition(context.Background(), OwnerAutoStrategy, "XRPUSDT", "LONG"); positionErr != orm.ErrNoRows {
		t.Fatalf("unconfirmed fill must not create managed position, err=%v", positionErr)
	}
}

func findTestOrder(clientOrderID string) (models.FuturesManagedOrder, error) {
	var row models.FuturesManagedOrder
	err := orm.NewOrm().QueryTable(new(models.FuturesManagedOrder)).Filter("client_order_id", clientOrderID).One(&row)
	return row, err
}

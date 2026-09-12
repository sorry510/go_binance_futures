package agenttrade

import (
	"context"
	"errors"
	"fmt"
	"math"
	"testing"
	"time"

	"go_binance_futures/models"
	futuresownership "go_binance_futures/service/futuresownership"

	"github.com/beego/beego/v2/client/orm"
)

type lifecycleBroker struct {
	submitCalls    map[string]int
	lookupCalls    map[string]int
	cancelIDs      []int64
	failIntent     string
	closeStatus    string
	closeFillRatio float64
	nextOrderID    int64
	orders         map[string]futuresownership.ExchangeOrder
}

func newLifecycleBroker() *lifecycleBroker {
	return &lifecycleBroker{submitCalls: map[string]int{}, lookupCalls: map[string]int{}, nextOrderID: 1000, orders: map[string]futuresownership.ExchangeOrder{}}
}

func (b *lifecycleBroker) Submit(_ context.Context, request futuresownership.OrderRequest, clientID string) (futuresownership.ExchangeOrder, error) {
	b.submitCalls[request.Intent]++
	if request.Intent == b.failIntent {
		return futuresownership.ExchangeOrder{}, errors.New("submit failed")
	}
	status, filled := "NEW", 0.0
	if request.Intent == futuresownership.IntentClose {
		status, filled = "FILLED", request.Quantity
		if b.closeStatus != "" {
			status = b.closeStatus
			filled = request.Quantity * b.closeFillRatio
		}
	}
	b.nextOrderID++
	result := futuresownership.ExchangeOrder{ExchangeOrderID: fmt.Sprintf("%d", b.nextOrderID), ClientOrderID: clientID, Status: status, FilledQty: filled, AveragePrice: 101}
	b.orders[clientID] = result
	return result, nil
}

func (b *lifecycleBroker) Lookup(_ context.Context, _ string, clientID, _ string) (futuresownership.ExchangeOrder, error) {
	b.lookupCalls[clientID]++
	row, ok := b.orders[clientID]
	if !ok {
		return futuresownership.ExchangeOrder{}, errors.New("not found")
	}
	return row, nil
}

func (b *lifecycleBroker) Cancel(_ context.Context, _ string, orderID int64, _ string) error {
	b.cancelIDs = append(b.cancelIDs, orderID)
	return nil
}

func seedAgentManagedPosition(t *testing.T, proposal models.AgentTradeProposal, qty float64) futuresownership.Service {
	t.Helper()
	ownership := futuresownership.Service{Now: func() time.Time { return time.UnixMilli(1_800_000_000_000).UTC() }}
	_, err := ownership.ClaimOrder(context.Background(), futuresownership.ClaimOrderInput{
		Owner: futuresownership.OwnerAgentTrade, Symbol: proposal.Symbol, PositionSide: proposal.Side,
		Intent: futuresownership.IntentOpen, ClientOrderID: lifecycleClientOrderID("entry", proposal.ProposalID), RequestedQty: qty, OrderType: "MARKET", SourceRef: proposal.ProposalID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ownership.ApplyFill(context.Background(), lifecycleClientOrderID("entry", proposal.ProposalID), qty, 100); err != nil {
		t.Fatal(err)
	}
	return ownership
}

func testOwnershipLifecycle(t *testing.T, proposal models.AgentTradeProposal, qty float64, broker *lifecycleBroker) OwnershipLifecycle {
	t.Helper()
	ownership := seedAgentManagedPosition(t, proposal, qty)
	return OwnershipLifecycle{
		Ownership:  ownership,
		Executor:   futuresownership.Executor{Ownership: ownership, Broker: broker},
		AccountQty: func(context.Context, string, string) (float64, error) { return qty, nil },
	}
}

func TestOwnershipLifecycleCreatesStopAndSingleTakeProfit(t *testing.T) {
	prepareTradeTestDB(t)
	proposal := baseProposal(time.UnixMilli(1_800_000_000_000).UTC())
	broker := newLifecycleBroker()
	lifecycle := testOwnershipLifecycle(t, proposal, 0.5, broker)
	result, err := lifecycle.EnsureProtection(context.Background(), proposal)
	if err != nil {
		t.Fatal(err)
	}
	if result.StopExchangeOrderID == "" || result.TakeProfitExchangeOrderID == "" {
		t.Fatalf("expected stop and one TP, got %+v", result)
	}
	orders, err := lifecycle.Ownership.ListOrders(context.Background(), futuresownership.OwnerAgentTrade, 20)
	if err != nil {
		t.Fatal(err)
	}
	intents := map[string]int{}
	for _, order := range orders {
		intents[order.Intent]++
	}
	if intents[futuresownership.IntentStopLoss] != 1 || intents[futuresownership.IntentTakeProfit] != 1 {
		t.Fatalf("unexpected protection intents: %+v", intents)
	}
}

func TestOwnershipLifecycleStopFailureNeverResubmits(t *testing.T) {
	prepareTradeTestDB(t)
	proposal := baseProposal(time.UnixMilli(1_800_000_000_000).UTC())
	broker := newLifecycleBroker()
	broker.failIntent = futuresownership.IntentStopLoss
	lifecycle := testOwnershipLifecycle(t, proposal, 0.5, broker)
	_, err := lifecycle.EnsureProtection(context.Background(), proposal)
	if err == nil {
		t.Fatal("stop failure must fail protection")
	}
	if broker.submitCalls[futuresownership.IntentStopLoss] != 1 {
		t.Fatalf("stop must be submitted once only, got %d", broker.submitCalls[futuresownership.IntentStopLoss])
	}
}

func TestOwnershipLifecycleProtectionCanRetryAfterTerminalFailure(t *testing.T) {
	prepareTradeTestDB(t)
	proposal := baseProposal(time.UnixMilli(1_800_000_000_000).UTC())
	broker := newLifecycleBroker()
	broker.failIntent = futuresownership.IntentStopLoss
	lifecycle := testOwnershipLifecycle(t, proposal, 0.5, broker)
	first, err := lifecycle.EnsureProtection(context.Background(), proposal)
	if err == nil {
		t.Fatal("first stop submission must fail")
	}
	if first.StopClientOrderID == "" {
		t.Fatal("failed protection must retain its client order id")
	}
	if err := lifecycle.Ownership.SetOrderStatus(context.Background(), first.StopClientOrderID, futuresownership.OrderFailed); err != nil {
		t.Fatal(err)
	}
	broker.failIntent = ""
	second, err := lifecycle.EnsureProtection(context.Background(), proposal)
	if err != nil {
		t.Fatalf("terminal failed protection must be replaceable: %v", err)
	}
	if second.StopClientOrderID == first.StopClientOrderID {
		t.Fatalf("replacement protection must use a new attempt id: %s", second.StopClientOrderID)
	}
	if broker.submitCalls[futuresownership.IntentStopLoss] != 2 {
		t.Fatalf("expected a second stop submission after terminal failure, calls=%d", broker.submitCalls[futuresownership.IntentStopLoss])
	}
}

func TestOwnershipLifecycleRecreatesProtectionAfterTerminalPartialFill(t *testing.T) {
	prepareTradeTestDB(t)
	proposal := baseProposal(time.UnixMilli(1_800_000_000_000).UTC())
	broker := newLifecycleBroker()
	lifecycle := testOwnershipLifecycle(t, proposal, 0.5, broker)
	first, err := lifecycle.EnsureProtection(context.Background(), proposal)
	if err != nil {
		t.Fatal(err)
	}

	// Simulate an exchange-side terminal protective fill that only reduced part
	// of the managed position. The remaining 0.3 must receive fresh protection.
	stop := broker.orders[first.StopClientOrderID]
	stop.Status, stop.FilledQty = "FILLED", 0.2
	broker.orders[first.StopClientOrderID] = stop
	if _, err := lifecycle.Executor.Reconcile(context.Background(), proposal.Symbol, first.StopClientOrderID); err != nil {
		t.Fatal(err)
	}
	remaining, err := lifecycle.Ownership.GetPosition(context.Background(), futuresownership.OwnerAgentTrade, proposal.Symbol, proposal.Side)
	if err != nil || math.Abs(remaining.ManagedQty-0.3) > 1e-12 {
		t.Fatalf("remaining managed position=%+v err=%v", remaining, err)
	}

	second, err := lifecycle.EnsureProtection(context.Background(), proposal)
	if err != nil {
		t.Fatalf("remaining managed position must be re-protected: %v", err)
	}
	if second.StopClientOrderID == first.StopClientOrderID {
		t.Fatalf("terminal filled stop must not be reused: %s", first.StopClientOrderID)
	}
	if broker.submitCalls[futuresownership.IntentStopLoss] != 2 {
		t.Fatalf("expected replacement stop submission, calls=%d", broker.submitCalls[futuresownership.IntentStopLoss])
	}
	row, err := findLifecycleOrder(second.StopClientOrderID)
	if err != nil || math.Abs(row.RequestedQty-0.3) > 1e-12 {
		t.Fatalf("replacement stop must protect remaining 0.3: %+v err=%v", row, err)
	}
}

func TestOwnershipLifecycleResizesProtectionAfterManagedQuantityShrinks(t *testing.T) {
	prepareTradeTestDB(t)
	proposal := baseProposal(time.UnixMilli(1_800_000_000_000).UTC())
	broker := newLifecycleBroker()
	lifecycle := testOwnershipLifecycle(t, proposal, 0.5, broker)
	first, err := lifecycle.EnsureProtection(context.Background(), proposal)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := lifecycle.Ownership.ReconcilePosition(context.Background(), futuresownership.OwnerAgentTrade, proposal.Symbol, proposal.Side, 0.3); err != nil {
		t.Fatal(err)
	}
	second, err := lifecycle.EnsureProtection(context.Background(), proposal)
	if err != nil {
		t.Fatalf("protection resize after external partial close failed: %v", err)
	}
	if second.StopClientOrderID == first.StopClientOrderID || second.TakeProfitClientOrderID == first.TakeProfitClientOrderID {
		t.Fatalf("resized protections must use new attempt IDs: first=%+v second=%+v", first, second)
	}
	if len(broker.cancelIDs) != 2 {
		t.Fatalf("both stale protections must be canceled before resize, cancel count=%d", len(broker.cancelIDs))
	}
	orders, err := lifecycle.Ownership.ListOrders(context.Background(), futuresownership.OwnerAgentTrade, 50)
	if err != nil {
		t.Fatal(err)
	}
	for _, order := range orders {
		if (order.ClientOrderID == second.StopClientOrderID || order.ClientOrderID == second.TakeProfitClientOrderID) && math.Abs(order.RequestedQty-0.3) > 1e-12 {
			t.Fatalf("replacement protection qty=%v, want 0.3: %+v", order.RequestedQty, order)
		}
	}
}

func TestOwnershipLifecycleCloseUsesManagedQtyAndCleansProtection(t *testing.T) {
	prepareTradeTestDB(t)
	proposal := baseProposal(time.UnixMilli(1_800_000_000_000).UTC())
	broker := newLifecycleBroker()
	lifecycle := testOwnershipLifecycle(t, proposal, 0.5, broker)
	if _, err := lifecycle.EnsureProtection(context.Background(), proposal); err != nil {
		t.Fatal(err)
	}
	lifecycle.AccountQty = func(context.Context, string, string) (float64, error) { return 2, nil } // manual add must stay unmanaged
	result, err := lifecycle.Close(context.Background(), proposal)
	if err != nil {
		t.Fatal(err)
	}
	if result.FilledQty != 0.5 || broker.submitCalls[futuresownership.IntentClose] != 1 {
		t.Fatalf("close must use managed qty only: result=%+v calls=%d", result, broker.submitCalls[futuresownership.IntentClose])
	}
	if len(broker.cancelIDs) != 2 {
		t.Fatalf("TP must be canceled before close and Stop after close, cancel count=%d", len(broker.cancelIDs))
	}
	if _, err := lifecycle.Ownership.GetPosition(context.Background(), futuresownership.OwnerAgentTrade, proposal.Symbol, proposal.Side); err != orm.ErrNoRows {
		t.Fatalf("managed position should be closed, err=%v", err)
	}
}

func TestOwnershipLifecycleCloseCanRetryRemainingQuantityAfterTerminalPartialFill(t *testing.T) {
	prepareTradeTestDB(t)
	proposal := baseProposal(time.UnixMilli(1_800_000_000_000).UTC())
	broker := newLifecycleBroker()
	broker.closeStatus = "CANCELED"
	broker.closeFillRatio = 0.4
	lifecycle := testOwnershipLifecycle(t, proposal, 0.5, broker)

	first, err := lifecycle.Close(context.Background(), proposal)
	if err == nil {
		t.Fatal("partial terminal close must report remaining managed quantity")
	}
	if first.FilledQty != 0.2 {
		t.Fatalf("first close filled qty=%v, want 0.2", first.FilledQty)
	}
	remaining, loadErr := lifecycle.Ownership.GetPosition(context.Background(), futuresownership.OwnerAgentTrade, proposal.Symbol, proposal.Side)
	if loadErr != nil || math.Abs(remaining.ManagedQty-0.3) > 1e-12 {
		t.Fatalf("remaining managed qty=%+v err=%v", remaining, loadErr)
	}

	broker.closeStatus = ""
	broker.closeFillRatio = 0
	second, err := lifecycle.Close(context.Background(), proposal)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(second.FilledQty-0.3) > 1e-12 || broker.submitCalls[futuresownership.IntentClose] != 2 {
		t.Fatalf("retry must submit only remaining qty: result=%+v calls=%d", second, broker.submitCalls[futuresownership.IntentClose])
	}
	if first.ClientOrderID == second.ClientOrderID {
		t.Fatalf("terminal partial close retry must use a new client order id: %s", first.ClientOrderID)
	}
	if _, err := lifecycle.Ownership.GetPosition(context.Background(), futuresownership.OwnerAgentTrade, proposal.Symbol, proposal.Side); err != orm.ErrNoRows {
		t.Fatalf("managed position should be fully closed after retry, err=%v", err)
	}
}

func TestOwnershipLifecycleDetectsProtectiveClose(t *testing.T) {
	prepareTradeTestDB(t)
	proposal := baseProposal(time.UnixMilli(1_800_000_000_000).UTC())
	broker := newLifecycleBroker()
	lifecycle := testOwnershipLifecycle(t, proposal, 0.5, broker)
	protection, err := lifecycle.EnsureProtection(context.Background(), proposal)
	if err != nil {
		t.Fatal(err)
	}
	stop := broker.orders[protection.StopClientOrderID]
	stop.Status, stop.FilledQty = "FILLED", 0.5
	broker.orders[protection.StopClientOrderID] = stop
	if _, err := lifecycle.Executor.Reconcile(context.Background(), proposal.Symbol, protection.StopClientOrderID); err != nil {
		t.Fatal(err)
	}
	_, err = lifecycle.EnsureProtection(context.Background(), proposal)
	if !errors.Is(err, ErrManagedPositionClosed) {
		t.Fatalf("protective fill must be recognized as closed, err=%v", err)
	}
	if _, closeErr := lifecycle.Close(context.Background(), proposal); closeErr != nil {
		t.Fatalf("closed managed position must clean sibling protection: %v", closeErr)
	}
	tpOrder, loadErr := findLifecycleOrder(protection.TakeProfitClientOrderID)
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	if tpOrder.Status != futuresownership.OrderCanceled {
		t.Fatalf("sibling take profit must be canceled after stop fill, status=%s", tpOrder.Status)
	}
}

func findLifecycleOrder(clientOrderID string) (models.FuturesManagedOrder, error) {
	var row models.FuturesManagedOrder
	err := orm.NewOrm().QueryTable(new(models.FuturesManagedOrder)).Filter("client_order_id", clientOrderID).One(&row)
	return row, err
}

package leadaccount

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/client/orm"
	binance "go_binance_futures/feature/api/binance"
	"go_binance_futures/service/futuresownership"
)

func TestStage53DurableCASOnlyOneWorkerCanReserve(t *testing.T) {
	prepareLeadExecutionDB(t)
	a, _ := stage43Adapter(t, false)
	ctx := context.Background()
	_, err := a.ownership.Ownership.ClaimOrder(ctx, futuresownership.ClaimOrderInput{
		Owner: futuresownership.OwnerAutoStrategy, Symbol: "BTCUSDT", PositionSide: "LONG", Intent: futuresownership.IntentOpen,
		ClientOrderID: "stage53_cas_shared", RequestedQty: 1, OrderType: "MARKET",
	})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	var won atomic.Int64
	var failed atomic.Int64
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			granted, e := a.ownership.Ownership.ReserveSubmission(ctx, "stage53_cas_shared")
			if e != nil {
				failed.Add(1)
				return
			}
			if granted {
				won.Add(1)
			}
		}()
	}
	wg.Wait()
	if failed.Load() != 0 || won.Load() != 1 {
		t.Fatalf("non-atomic reservation winner=%d errors=%d", won.Load(), failed.Load())
	}
	row, err := a.ownership.Ownership.GetOrder(ctx, "stage53_cas_shared")
	if err != nil || row.Status != futuresownership.OrderSubmitted {
		t.Fatalf("reservation not durable %+v %v", row, err)
	}
	main, err := futuresownership.BindAccount(binance.MainAccountID)
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := main.ReserveSubmission(ctx, "stage53_cas_shared"); err != nil || ok {
		t.Fatalf("Main modified Lead reservation: %v %v", ok, err)
	}
}
func stage53Auditor(t *testing.T) (*LeadRecoveryAuditor, *stage43Broker, *fakeLeadSnapshotReader) {
	t.Helper()
	a, b := stage43Adapter(t, false)
	reader := stage51SeedReader()
	reader.orders = nil
	reader.positions = []*futures.PositionRisk{{Symbol: "BTCUSDT", PositionSide: "LONG", PositionAmt: "1", MarkPrice: "100", UnRealizedProfit: "0", Leverage: "4"}}
	return &LeadRecoveryAuditor{Owner: a.ownership.Ownership, Broker: b, Reader: reader}, b, reader
}
func TestStage53RestartResolvesKnownFillButDoesNotEnableTrade(t *testing.T) {
	prepareLeadExecutionDB(t)
	audit, broker, _ := stage53Auditor(t)
	_, err := audit.Owner.ClaimOrder(context.Background(), futuresownership.ClaimOrderInput{
		Owner: futuresownership.OwnerAutoStrategy, Symbol: "BTCUSDT", PositionSide: "LONG", Intent: futuresownership.IntentOpen,
		ClientOrderID: "stage53_restart", RequestedQty: 1, OrderType: "MARKET",
	})
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := audit.Owner.ReserveSubmission(context.Background(), "stage53_restart"); err != nil || !ok {
		t.Fatalf("reservation failed: %v %v", ok, err)
	}
	broker.lookupResult = futuresownership.ExchangeOrder{ExchangeOrderID: "876", ClientOrderID: "stage53_restart", Status: "FILLED", FilledQty: 1, AveragePrice: 100}
	report, err := audit.Reconcile(context.Background())
	if err != nil || report.Blocked || report.OrdersChecked != 1 {
		t.Fatalf("recovery failed %+v %v", report, err)
	}
	order, err := audit.Owner.GetOrder(context.Background(), "stage53_restart")
	if err != nil || order.Status != futuresownership.OrderFilled || order.FilledQty != 1 {
		t.Fatalf("ledger missing fill %+v %v", order, err)
	}
	p, err := audit.Owner.GetPosition(context.Background(), futuresownership.OwnerAutoStrategy, "BTCUSDT", "LONG")
	if err != nil || p.ManagedQty != 1 {
		t.Fatalf("position missing fill %+v %v", p, err)
	}
	if broker.submits != 0 || broker.lookups != 1 {
		t.Fatalf("recovery tried to submit an order: %+v", broker)
	}
	// Full recovery evidence does NOT release Stage 4 executor's sticky flag.
	a, _ := stage43Adapter(t, false)
	if a.RuntimeStatus().CanAttemptOpen {
		t.Fatal("recovery authorized real Lead trading")
	}
}
func TestStage53MissingExchangeEvidenceStaysBlockedAcrossRestart(t *testing.T) {
	prepareLeadExecutionDB(t)
	audit, broker, _ := stage53Auditor(t)
	_, err := audit.Owner.ClaimOrder(context.Background(), futuresownership.ClaimOrderInput{
		Owner: futuresownership.OwnerAutoStrategy, Symbol: "BTCUSDT", PositionSide: "LONG", Intent: futuresownership.IntentOpen,
		ClientOrderID: "stage53_unknown", RequestedQty: 1, OrderType: "MARKET",
	})
	if err != nil {
		t.Fatal(err)
	}
	broker.lookupErr = errors.New("simulated unknown exchange result")
	for i := 0; i < 2; i++ {
		report, e := audit.Reconcile(context.Background())
		if !errors.Is(e, ErrLeadReconcileIncomplete) || !report.Blocked {
			t.Fatalf("unknown accepted %+v %v", report, e)
		}
	}
	row, err := audit.Owner.GetOrder(context.Background(), "stage53_unknown")
	if err != nil || row.Status != futuresownership.OrderReconcile {
		t.Fatalf("unknown not persisted: %+v %v", row, err)
	}
	if broker.submits != 0 {
		t.Fatal("recovery created a duplicate submission")
	}
}
func TestStage53IncludesClosedHistoryAndRejectsUnknownExchangeOrders(t *testing.T) {
	prepareLeadExecutionDB(t)
	audit, _, reader := stage53Auditor(t)
	ctx := context.Background()
	_, err := audit.Owner.ClaimOrder(ctx, futuresownership.ClaimOrderInput{
		Owner: futuresownership.OwnerAutoStrategy, Symbol: "BTCUSDT", PositionSide: "LONG", Intent: futuresownership.IntentOpen,
		ClientOrderID: "stage53_closed_open", RequestedQty: 1, OrderType: "MARKET",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = audit.Owner.ApplyFill(ctx, "stage53_closed_open", 1, 100); err != nil {
		t.Fatal(err)
	}
	_, err = audit.Owner.ClaimOrder(ctx, futuresownership.ClaimOrderInput{
		Owner: futuresownership.OwnerAutoStrategy, Symbol: "BTCUSDT", PositionSide: "LONG", Intent: futuresownership.IntentClose,
		ClientOrderID: "stage53_closed_close", RequestedQty: 1, OrderType: "MARKET",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = audit.Owner.ApplyFill(ctx, "stage53_closed_close", 1, 100); err != nil {
		t.Fatal(err)
	}
	reader.positions = nil
	report, err := audit.Reconcile(ctx)
	if err != nil || report.Blocked || report.PositionsChecked != 1 {
		t.Fatalf("CLOSED record not counted %+v %v", report, err)
	}
	if _, err := audit.Owner.GetPosition(ctx, futuresownership.OwnerAutoStrategy, "BTCUSDT", "LONG"); !errors.Is(err, orm.ErrNoRows) {
		t.Fatalf("GetPosition unexpectedly returns CLOSED: %v", err)
	}
	all, err := audit.Owner.ListPositions(ctx, "", false)
	if err != nil || len(all) != 1 || all[0].Status != futuresownership.PositionClosed {
		t.Fatalf("missing CLOSED history %+v %v", all, err)
	}
	reader.orders = []*futures.Order{{Symbol: "BTCUSDT", OrderID: 8, ClientOrderID: "unmanaged-lead-order"}}
	report, err = audit.Reconcile(ctx)
	if !errors.Is(err, ErrLeadReconcileIncomplete) || !report.Blocked {
		t.Fatalf("unknown order unblocked %+v %v", report, err)
	}
}
func TestStage53MainCannotConstructLeadRecovery(t *testing.T) {
	main, err := binance.NewAccountClient(binance.MainAccountID, futures.NewClient("fake-main", "secret"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewLeadRecoveryAuditor(main); err == nil {
		t.Fatal("Main client bound as Lead recovery")
	}
}

func TestStage53OnlyNewerNoGapWSAndDurableProofClearsPending(t *testing.T) {
	a, broker := stage43Adapter(t, false)
	a.pendingReconcile = true
	a.guard.RecordFault(binance.LeadAccountID, FaultPendingReconcile)
	a.guard.RecordFault(binance.LeadAccountID, FaultDailyLoss)
	proof := LeadRecoveryReport{AccountID: binance.LeadAccountID, ObservedAt: time.Now().UTC().Add(-time.Second)}
	mirror := NewLeadAccountMirror()
	generation := mirror.BeginGeneration()
	status := mirror.Status()
	if a.completeAuthoritativeRecovery(proof, status) {
		t.Fatal("unseeded WS released pending")
	}
	if err := mirror.Seed(context.Background(), generation, stage51SeedReader()); err != nil {
		t.Fatal(err)
	}
	status = mirror.Status()
	failed := proof
	failed.Blocked = true
	if a.completeAuthoritativeRecovery(failed, status) {
		t.Fatal("failed DB recovery released pending")
	}
	failed = proof
	failed.AccountID = binance.MainAccountID
	if a.completeAuthoritativeRecovery(failed, status) {
		t.Fatal("Main proof released Lead pending")
	}
	old := proof
	old.ObservedAt = status.CapturedAt.Add(time.Minute)
	if a.completeAuthoritativeRecovery(old, status) {
		t.Fatal("WS predates DB proof yet pending was cleared")
	}
	if !a.completeAuthoritativeRecovery(proof, status) {
		t.Fatal("valid new WS plus durable proof did not reconcile")
	}
	if a.pendingReconcile {
		t.Fatal("verified pending flag not cleared")
	}
	state := a.RuntimeStatus()
	if state.State != LeadRiskTripped || state.CanAttemptOpen {
		t.Fatalf("proof cleared unrelated trip or authorized trading: %+v", state)
	}
	if broker.submits != 0 {
		t.Fatal("proof issued an exchange order")
	}
}

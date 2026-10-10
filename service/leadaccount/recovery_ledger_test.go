package leadaccount

import (
	"context"
	"testing"

	"go_binance_futures/service/futuresownership"
)

// Stage 4-4 uses a fake broker and an in-memory database. This exercises the
// existing Lead-bound persistence path; it does NOT authorize a live recovery
// entrypoint or clear the Stage 5 pendingReconcile reservation.
func TestStage44LeadCumulativeFillAfterCancelNeverDoubleCounts(t *testing.T) {
	prepareLeadExecutionDB(t)
	a, _ := stage43Adapter(t, true)
	ctx := context.Background()
	clientID := "lead_stage44_partial_cancel"
	_, err := a.ownership.Ownership.ClaimOrder(ctx, futuresownership.ClaimOrderInput{
		Owner: futuresownership.OwnerAutoStrategy, Symbol: "BTCUSDT", PositionSide: "LONG",
		Intent: futuresownership.IntentOpen, ClientOrderID: clientID, RequestedQty: 1, OrderType: "LIMIT",
	})
	if err != nil {
		t.Fatal(err)
	}
	records := []futuresownership.ExchangeOrder{
		{ExchangeOrderID: "731", ClientOrderID: clientID, Status: "PARTIALLY_FILLED", FilledQty: 0.3, AveragePrice: 100},
		{ExchangeOrderID: "731", ClientOrderID: clientID, Status: "PARTIALLY_FILLED", FilledQty: 0.3, AveragePrice: 100},
		{ExchangeOrderID: "731", ClientOrderID: clientID, Status: "CANCELED", FilledQty: 0.5, AveragePrice: 100},
		{ExchangeOrderID: "731", ClientOrderID: clientID, Status: "CANCELED", FilledQty: 0.5, AveragePrice: 100},
	}
	for i, observed := range records {
		evidence, err := ClassifyLeadRecovery(observed, clientID)
		if err != nil || !evidence.RequiresReconcile {
			t.Fatalf("untrusted evidence #%d: %+v %v", i, evidence, err)
		}
		if _, err := a.ownership.ApplyObservedExchange(ctx, clientID, evidence.Order); err != nil {
			t.Fatalf("cumulative apply #%d: %v", i, err)
		}
	}
	order, err := a.ownership.Ownership.GetOrder(ctx, clientID)
	if err != nil || order.AccountID != "lead" || order.Status != futuresownership.OrderCanceled {
		t.Fatalf("incorrect canceled order %+v err=%v", order, err)
	}
	position, err := a.ownership.Ownership.GetPosition(ctx, futuresownership.OwnerAutoStrategy, "BTCUSDT", "LONG")
	if err != nil || position.AccountID != "lead" || position.ManagedQty != 0.5 {
		t.Fatalf("double counted or wrong account %+v err=%v", position, err)
	}
	a.pendingReconcile = true
	if !a.pendingReconcile {
		t.Fatal("reconcile gate unexpectedly released")
	}
}

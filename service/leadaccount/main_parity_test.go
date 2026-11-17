package leadaccount

import (
	"context"
	"errors"
	"testing"

	"go_binance_futures/service/futuresownership"
)

func TestLeadMainParityAcknowledgedStopAndTargetWithoutGlobalLock(t *testing.T) {
	for _, tc := range []struct {
		side         string
		stop, target float64
	}{
		{"LONG", 95.5, 105.5},
		{"SHORT", 105.5, 95.5},
	} {
		t.Run(tc.side, func(t *testing.T) {
			a, broker, _ := stage43OwnedFixture(t, tc.side, 1)
			broker.result = futuresownership.ExchangeOrder{ExchangeOrderID: "123", Status: "NEW"}
			stop := stage43CloseRequest(tc.side, "STOP_MARKET", 1)
			stop.ClientOrderID = "lead_parity_stop_" + tc.side
			stop.Intent = futuresownership.IntentStopLoss
			stop.StopPrice = tc.stop
			if _, err := a.ExecuteManagedClose(context.Background(), stop); err != nil {
				t.Fatalf("Binance accepted stop must not globally lock: %v", err)
			}
			if a.pendingReconcile || broker.submits != 1 {
				t.Fatalf("accepted STOP incorrectly locked account: pending=%t submits=%d", a.pendingReconcile, broker.submits)
			}
			target := stage43CloseRequest(tc.side, "TAKE_PROFIT_MARKET", 1)
			target.ClientOrderID = "lead_parity_target_" + tc.side
			target.Intent = futuresownership.IntentTakeProfit
			target.StopPrice = tc.target
			if _, err := a.ExecuteManagedClose(context.Background(), target); err != nil {
				t.Fatalf("Binance accepted TP should follow confirmed STOP: %v", err)
			}
			if a.pendingReconcile || broker.submits != 2 {
				t.Fatalf("TP blocked or duplicated despite normal Binance response: pending=%t submits=%d", a.pendingReconcile, broker.submits)
			}
			if _, err := a.ExecuteManagedClose(context.Background(), target); err != nil {
				t.Fatalf("same clientOrderId repeat should be idempotent: %v", err)
			}
			if broker.submits != 2 {
				t.Fatal("duplicate TP submitted")
			}
		})
	}
}

func TestLeadMainParityUnknownSubmitStillRequiresOrderLookup(t *testing.T) {
	a, broker := stage43Adapter(t, true)
	broker.err = errors.New("simulated timeout after Binance request")
	broker.lookupErr = errors.New("order status unavailable")
	req, check := stage43Request("LONG", "MARKET", "lead_parity_unknown")
	if _, err := a.ExecuteOpen(context.Background(), req, check); err == nil {
		t.Fatal("unknown submit incorrectly accepted")
	}
	if broker.submits != 1 || broker.lookups != 1 || !a.pendingReconcile {
		t.Fatalf("unknown order was resubmitted or left unguarded: submits=%d lookups=%d", broker.submits, broker.lookups)
	}
	if _, err := a.ExecuteOpen(context.Background(), req, check); !errors.Is(err, ErrLeadPendingReconcile) {
		t.Fatalf("unknown order did not block a second submit: %v", err)
	}
	if broker.submits != 1 {
		t.Fatal("repeated unknown order POST")
	}
}

package leadaccount

import (
	"context"
	"errors"
	"testing"

	"github.com/adshao/go-binance/v2/futures"
	"go_binance_futures/service/futuresownership"
)

func stage43OwnedFixture(t *testing.T, side string, managed float64) (*LeadExecutionAdapter, *stage43Broker, *fakeLeadSnapshotReader) {
	t.Helper()
	prepareLeadExecutionDB(t)
	a, broker := stage43Adapter(t, true)
	fake := sampleLeadEvidenceReader()
	fake.positions = []*futures.PositionRisk{{Symbol: "BTCUSDT", PositionSide: side, PositionAmt: "1", MarkPrice: "100", UnRealizedProfit: "0", Leverage: "4"}}
	a.source.Reader = fake
	_, err := a.ownership.Ownership.ClaimOrder(context.Background(), futuresownership.ClaimOrderInput{
		Owner: futuresownership.OwnerAutoStrategy, Symbol: "BTCUSDT", PositionSide: side, Intent: futuresownership.IntentOpen, ClientOrderID: "stage43_owned_" + side,
		RequestedQty: managed, OrderType: "MARKET",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.ownership.Ownership.ApplyFill(context.Background(), "stage43_owned_"+side, managed, 100); err != nil {
		t.Fatal(err)
	}
	return a, broker, fake
}
func stage43CloseRequest(side, orderType string, quantity float64) futuresownership.OrderRequest {
	sideValue := "SELL"
	if side == "SHORT" {
		sideValue = "BUY"
	}
	price := 0.0
	if orderType == "LIMIT" {
		price = 100
	}
	return futuresownership.OrderRequest{
		Owner: futuresownership.OwnerAutoStrategy, Symbol: "BTCUSDT", PositionSide: side, Intent: futuresownership.IntentClose,
		Side: sideValue, OrderType: orderType, Quantity: quantity, Price: price, ClientOrderID: "stage43_close_" + side,
	}
}
func TestStage43ManagedCloseCapsAtMinManagedAndLive(t *testing.T) {
	for _, side := range []string{"LONG", "SHORT"} {
		t.Run(side, func(t *testing.T) {
			a, broker, reader := stage43OwnedFixture(t, side, 2)
			request := stage43CloseRequest(side, "MARKET", 1.5)
			if _, err := a.ExecuteManagedClose(context.Background(), request); !errors.Is(err, ErrLeadOpenBlocked) {
				t.Fatalf("attempted to close > live position: %v", err)
			}
			if broker.submits != 0 {
				t.Fatal("close leaked to exchange")
			}
			request.Quantity = 1
			broker.result = futuresownership.ExchangeOrder{ExchangeOrderID: "997", FilledQty: 1, AveragePrice: 100, Status: "FILLED"}
			_, err := a.ExecuteManagedClose(context.Background(), request)
			if err != nil {
				t.Fatalf("owned close failed: %v", err)
			}
			if broker.submits != 1 {
				t.Fatalf("close submitted %d times", broker.submits)
			}
			owned, err := a.ownership.Ownership.GetPosition(context.Background(), futuresownership.OwnerAutoStrategy, "BTCUSDT", side)
			if err != nil || owned.ManagedQty != 1 {
				t.Fatalf("close changed managed quantity incorrectly: %+v %v", owned, err)
			}
			if _, err := a.ExecuteManagedClose(context.Background(), request); !errors.Is(err, ErrLeadPendingReconcile) {
				t.Fatalf("second close not paused for reconcile: %v", err)
			}
			_ = reader
		})
	}
}
func TestStage43ManagedClosePauseAllowedButIdentityBlocked(t *testing.T) {
	a, broker, _ := stage43OwnedFixture(t, "LONG", 1)
	a.risk.Pause() // opening pause must not change managed close eligibility
	req := stage43CloseRequest("LONG", "MARKET", 1)
	if _, err := a.ExecuteManagedClose(context.Background(), req); err != nil {
		t.Fatalf("paused opens blocked owned close: %v", err)
	}
	if broker.submits != 1 {
		t.Fatal("managed close did not reach Fake Broker")
	}
}
func TestStage43ManagedCloseUnauthorizedAndManualExcessRejected(t *testing.T) {
	a, broker, fake := stage43OwnedFixture(t, "LONG", 1)
	req := stage43CloseRequest("LONG", "MARKET", 2)
	if _, err := a.ExecuteManagedClose(context.Background(), req); !errors.Is(err, ErrLeadOpenBlocked) {
		t.Fatalf("exceeds managed qty: %v", err)
	}
	req.Quantity = 1
	fake.positions[0].PositionAmt = "0.5" // user manually reduced
	if _, err := a.ExecuteManagedClose(context.Background(), req); !errors.Is(err, ErrLeadOpenBlocked) {
		t.Fatalf("exceeds live qty: %v", err)
	}
	a.risk.mu.Lock()
	a.risk.state.Stage7Authorized = false
	a.risk.mu.Unlock()
	req.Quantity = 0.5
	if _, err := a.ExecuteManagedClose(context.Background(), req); !errors.Is(err, ErrLeadOpenBlocked) {
		t.Fatalf("Stage7 gate bypass: %v", err)
	}
	if broker.submits != 0 {
		t.Fatalf("unauthorized close submitted %d", broker.submits)
	}
}
func TestStage43ManagedCloseRejectsUnownedAndIncorrectDirection(t *testing.T) {
	a, broker, _ := stage43OwnedFixture(t, "LONG", 1)
	req := stage43CloseRequest("SHORT", "MARKET", 1)
	if _, err := a.ExecuteManagedClose(context.Background(), req); !errors.Is(err, ErrLeadOpenBlocked) {
		t.Fatalf("wrong side close: %v", err)
	}
	req = stage43CloseRequest("LONG", "MARKET", 1)
	req.Side = "BUY"
	if _, err := a.ExecuteManagedClose(context.Background(), req); !errors.Is(err, ErrLeadOpenBlocked) {
		t.Fatalf("wrong direction: %v", err)
	}
	req.Side = "SELL"
	req.Owner = futuresownership.OwnerAgentTrade
	if _, err := a.ExecuteManagedClose(context.Background(), req); !errors.Is(err, ErrLeadOpenBlocked) {
		t.Fatalf("wrong owner: %v", err)
	}
	if broker.submits != 0 {
		t.Fatal("unexpected close submitted")
	}
}

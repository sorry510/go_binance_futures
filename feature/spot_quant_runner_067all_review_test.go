package feature

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"go_binance_futures/service/spotexecution"
	"go_binance_futures/service/spotledger"

	"github.com/shopspring/decimal"
)

// Review ALL F3: a previously canceled/expired/rejected SELL with NO Fill
// must not permanently strand real strategy-owned inventory. Before any
// new POST the old order is verified over REST + myTrades, then a stable,
// distinct retry ID is used. One further cancellation stops automatic retries.
func TestReview067ALLTerminalZeroFillSELLCanCloseOnce(t *testing.T) {
	for _, terminal := range []string{"CANCELED", "EXPIRED", "REJECTED"} {
		t.Run(terminal, func(t *testing.T) {
			r, ex, processor, db := spot0673Fixture(t)
			spot0673Enable(t, db)
			ctx := context.Background()
			if err := r.Run(ctx); err != nil || ex.places != 1 {
				t.Fatalf("setup BUY failed: %v places=%d", err, ex.places)
			}
			pos, ok, err := r.activeSpotPosition("BTCUSDT")
			if err != nil || !ok || pos.Status != "OPEN" {
				t.Fatalf("missing verified inventory: %+v err=%v", pos, err)
			}
			originalID := spotQuantExitClientID("BTCUSDT", pos.ID)
			original, duplicate, err := r.ledger.CreateOrder(spotledger.OrderIntent{
				PositionID: pos.ID, Symbol: "BTCUSDT", Side: "SELL", Type: "MARKET",
				ClientOrderID: originalID, RequestedQty: pos.RemainingQty,
			})
			if err != nil || duplicate {
				t.Fatalf("failed to seed terminal SELL: %v duplicate=%v", err, duplicate)
			}
			exchangeID := fmt.Sprintf("8000%d", original.ID)
			ex.orders[originalID] = spotexecution.RemoteOrder{
				Symbol: "BTCUSDT", Side: "SELL", Type: "MARKET",
				ClientOrderID: originalID, ExchangeOrderID: exchangeID,
				Status: terminal, OrigQty: pos.RemainingQty, ExecutedQty: "0",
			}
			ex.trades[exchangeID] = nil
			if _, err := processor.Reconcile(ctx, original.ID); err != nil {
				t.Fatalf("terminal SELL reconciliation: %v", err)
			}
			ex.price = "110"
			if err := r.Run(ctx); err != nil {
				t.Fatalf("%s zero-Fill SELL did not allow safe replacement: %v", terminal, err)
			}
			if ex.places != 2 {
				t.Fatalf("expected one supplemental SELL, places=%d", ex.places)
			}
			retryID := spotQuantExitRetryClientID("BTCUSDT", pos.ID)
			if retryID == originalID || len(retryID) > 36 {
				t.Fatalf("unsafe retry ClientOrderID %q", retryID)
			}
			supplemental, err := r.ledger.FindOrderByClientID(retryID)
			if err != nil || supplemental.Side != "SELL" || supplemental.Status != "FILLED" {
				t.Fatalf("supplemental SELL not verified: %+v err=%v", supplemental, err)
			}
			projection, err := r.ledger.Accounting(pos.ID, "110")
			if err != nil || projection.Position.Status != "CLOSED" {
				t.Fatalf("verified zero-inventory close failed: %+v err=%v", projection, err)
			}
			for i := 0; i < 4; i++ {
				if err := r.Run(ctx); err != nil {
					t.Fatal(err)
				}
			}
			if ex.places != 2 {
				t.Fatalf("duplicate supplemental order: places=%d", ex.places)
			}
		})
	}
}

func TestReview067ALLCanceledSELLWithFillNeverRetried(t *testing.T) {
	r, ex, processor, db := spot0673Fixture(t)
	spot0673Enable(t, db)
	ctx := context.Background()
	if err := r.Run(ctx); err != nil {
		t.Fatal(err)
	}
	pos, ok, err := r.activeSpotPosition("BTCUSDT")
	if err != nil || !ok {
		t.Fatalf("missing BUY inventory: %v", err)
	}
	id := spotQuantExitClientID("BTCUSDT", pos.ID)
	order, _, err := r.ledger.CreateOrder(spotledger.OrderIntent{
		PositionID: pos.ID, Symbol: "BTCUSDT", Side: "SELL", Type: "MARKET",
		ClientOrderID: id, RequestedQty: pos.RemainingQty,
	})
	if err != nil {
		t.Fatal(err)
	}
	qty := decimal.RequireFromString(pos.RemainingQty).Div(decimal.NewFromInt(2))
	exchangeID := fmt.Sprintf("9000%d", order.ID)
	remote := spotexecution.RemoteOrder{
		Symbol: "BTCUSDT", Side: "SELL", Type: "MARKET", ClientOrderID: id,
		ExchangeOrderID: exchangeID, Status: "CANCELED",
		OrigQty: pos.RemainingQty, ExecutedQty: qty.String(),
	}
	ex.orders[id] = remote
	ex.trades[exchangeID] = []spotexecution.Trade{ex.makeTrade(remote, qty.String())}
	if _, err := processor.Reconcile(ctx, order.ID); err != nil {
		t.Fatal(err)
	}
	ex.price = "110"
	if err := r.Run(ctx); !errors.Is(err, spotledger.ErrNeedsReview) || ex.places != 1 {
		t.Fatalf("canceled partly FILLED SELL was duplicated: err=%v places=%d", err, ex.places)
	}
}

func TestReview067ALLCanceledSELLUnknownRemoteNeverRetried(t *testing.T) {
	r, ex, processor, db := spot0673Fixture(t)
	spot0673Enable(t, db)
	ctx := context.Background()
	if err := r.Run(ctx); err != nil {
		t.Fatal(err)
	}
	pos, _, err := r.activeSpotPosition("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	id := spotQuantExitClientID("BTCUSDT", pos.ID)
	order, _, err := r.ledger.CreateOrder(spotledger.OrderIntent{
		PositionID: pos.ID, Symbol: "BTCUSDT", Side: "SELL", Type: "MARKET",
		ClientOrderID: id, RequestedQty: pos.RemainingQty,
	})
	if err != nil {
		t.Fatal(err)
	}
	exchangeID := fmt.Sprintf("9100%d", order.ID)
	ex.orders[id] = spotexecution.RemoteOrder{
		Symbol: "BTCUSDT", Side: "SELL", Type: "MARKET", ClientOrderID: id,
		ExchangeOrderID: exchangeID, Status: "CANCELED",
		OrigQty: pos.RemainingQty, ExecutedQty: "0",
	}
	if _, err := processor.Reconcile(ctx, order.ID); err != nil {
		t.Fatal(err)
	}
	// A remote lookup that now disagrees with the persisted CANCELED
	// snapshot must fail closed, not blindly place a replacement.
	changed := ex.orders[id]
	changed.Status = "NEW"
	ex.orders[id] = changed
	ex.price = "110"
	if err := r.Run(ctx); !errors.Is(err, spotledger.ErrNeedsReview) || ex.places != 1 {
		t.Fatalf("remote status disagreement permitted SELL retry: %v places=%d", err, ex.places)
	}
}

func TestReview067ALLRepeatedCanceledSELLFailsClosed(t *testing.T) {
	r, ex, processor, db := spot0673Fixture(t)
	spot0673Enable(t, db)
	ctx := context.Background()
	if err := r.Run(ctx); err != nil {
		t.Fatal(err)
	}
	pos, _, err := r.activeSpotPosition("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	for idx, id := range []string{
		spotQuantExitClientID("BTCUSDT", pos.ID),
		spotQuantExitRetryClientID("BTCUSDT", pos.ID),
	} {
		order, _, err := r.ledger.CreateOrder(spotledger.OrderIntent{
			PositionID: pos.ID, Symbol: "BTCUSDT", Side: "SELL", Type: "MARKET",
			ClientOrderID: id, RequestedQty: pos.RemainingQty,
		})
		if err != nil {
			t.Fatal(err)
		}
		exchangeID := fmt.Sprintf("920%d%d", idx, order.ID)
		ex.orders[id] = spotexecution.RemoteOrder{
			Symbol: "BTCUSDT", Side: "SELL", Type: "MARKET", ClientOrderID: id,
			ExchangeOrderID: exchangeID, Status: "CANCELED",
			OrigQty: pos.RemainingQty, ExecutedQty: "0",
		}
		if _, err := processor.Reconcile(ctx, order.ID); err != nil {
			t.Fatal(err)
		}
	}
	ex.price = "110"
	if err := r.Run(ctx); !errors.Is(err, spotledger.ErrNeedsReview) || ex.places != 1 {
		t.Fatalf("infinite cancellation retry not blocked: %v places=%d", err, ex.places)
	}
}

package feature

import (
	"strings"
	"testing"

	"github.com/adshao/go-binance/v2/common"

	"go_binance_futures/models"
	futuresownership "go_binance_futures/service/futuresownership"
)

func TestShouldRefreshOwnershipAccountQuantities(t *testing.T) {
	if shouldRefreshOwnershipAccountQuantities(false) {
		t.Fatal("stable active order must reuse the current account snapshot")
	}
	if !shouldRefreshOwnershipAccountQuantities(true) {
		t.Fatal("a newly observed fill must refresh account quantities once")
	}
}

func TestFuturesOrderObservation(t *testing.T) {
	observed, err := futuresOrderObservation(models.FuturesOrder{
		ClientOrderId: "aut_local",
		OrderId:       "12345",
		Status:        "PARTIALLY_FILLED",
		ExecutedQty:   "1.25",
		AveragePrice:  "99.5",
	})
	if err != nil {
		t.Fatal(err)
	}
	if observed.ExchangeOrderID != "12345" || observed.ClientOrderID != "aut_local" ||
		observed.Status != "PARTIALLY_FILLED" || observed.FilledQty != 1.25 || observed.AveragePrice != 99.5 {
		t.Fatalf("unexpected observed order: %+v", observed)
	}
}

func TestAutoStrategySourceRefCarriesRuleHash(t *testing.T) {
	hash := strings.Repeat("a", 64)
	ref := autoStrategySourceRef("btcusdt", hash)
	if ref != "auto_strategy:BTCUSDT:"+hash {
		t.Fatalf("unexpected source ref: %s", ref)
	}
	if got := autoStrategyOpenRuleHash(futuresownership.OwnerAutoStrategy, ref); got != hash {
		t.Fatalf("unexpected rule hash: %s", got)
	}
	if got := autoStrategySourceRef("btcusdt", ""); got != "auto_strategy:BTCUSDT" {
		t.Fatalf("legacy source ref changed: %s", got)
	}
}

// A completed local fill needs REST money verification, not 2-second order
// status polling. Missing exchange identity/partial/uncertain execution must
// continue to use the original fast recovery path.
func TestDeferCompletedFillVerification(t *testing.T) {
	positionID := int64(51)
	complete := models.FuturesManagedOrder{
		Status:          futuresownership.OrderReconcile,
		PositionID:      &positionID,
		ExchangeOrderID: "12345",
		RequestedQty:    0.153,
		FilledQty:       0.153,
	}
	if !deferCompletedFillVerification(complete) {
		t.Fatal("completed exchange execution should use slow ledger recovery")
	}
	tests := []struct {
		name   string
		change func(*models.FuturesManagedOrder)
	}{
		{"unknown exchange ID", func(o *models.FuturesManagedOrder) { o.ExchangeOrderID = "" }},
		{"missing position", func(o *models.FuturesManagedOrder) { o.PositionID = nil }},
		{"pending execution", func(o *models.FuturesManagedOrder) { o.FilledQty = 0 }},
		{"partial execution", func(o *models.FuturesManagedOrder) { o.FilledQty = 0.1 }},
		{"overfilled mismatch", func(o *models.FuturesManagedOrder) { o.FilledQty = 0.2 }},
		{"filled terminal", func(o *models.FuturesManagedOrder) { o.Status = futuresownership.OrderFilled }},
		{"other active status", func(o *models.FuturesManagedOrder) { o.Status = futuresownership.OrderSubmitted }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			row := complete
			tt.change(&row)
			if deferCompletedFillVerification(row) {
				t.Fatalf("unsafe order was deferred: %+v", row)
			}
		})
	}
}

func TestShouldCancelTimedOutAutoOrder(t *testing.T) {
	now := int64(1800000000000)
	example := models.FuturesManagedOrder{
		Intent: futuresownership.IntentOpen, Status: futuresownership.OrderReconcile,
		CreatedAt: now - 60000, RequestedQty: 0.153, FilledQty: 0.153,
		ExchangeOrderID: "15843156710",
	}
	if shouldCancelTimedOutAutoOrder(example, now, 30) {
		t.Fatal("fully filled reconcile_required order must never be canceled")
	}
	tests := []struct {
		name string
		edit func(*models.FuturesManagedOrder)
		want bool
	}{
		{"working remainder", func(o *models.FuturesManagedOrder) { o.FilledQty = 0 }, true},
		{"partially filled remainder", func(o *models.FuturesManagedOrder) {
			o.FilledQty = 0.1
			o.Status = futuresownership.OrderPartiallyFilled
		}, true},
		{"not timed out", func(o *models.FuturesManagedOrder) { o.CreatedAt = now - 1000; o.FilledQty = 0 }, false},
		{"canceled order", func(o *models.FuturesManagedOrder) { o.Status = futuresownership.OrderCanceled; o.FilledQty = 0 }, false},
		{"filled status", func(o *models.FuturesManagedOrder) { o.Status = futuresownership.OrderFilled; o.FilledQty = 0 }, false},
		{"failed order", func(o *models.FuturesManagedOrder) { o.Status = futuresownership.OrderFailed; o.FilledQty = 0 }, false},
		{"close order", func(o *models.FuturesManagedOrder) { o.Intent = futuresownership.IntentClose; o.FilledQty = 0 }, false},
		{"no requested amount", func(o *models.FuturesManagedOrder) { o.RequestedQty = 0; o.FilledQty = 0 }, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := example
			tt.edit(&o)
			if got := shouldCancelTimedOutAutoOrder(o, now, 30); got != tt.want {
				t.Fatalf("got %t want %t, order=%+v", got, tt.want, o)
			}
		})
	}
}

func TestIsUnknownBinanceCancelOrder(t *testing.T) {
	if !isUnknownBinanceCancelOrder(&common.APIError{Code: -2011, Message: "Unknown order sent."}) {
		t.Fatal("Binance -2011 must be treated as unknown order identity/state")
	}
	if isUnknownBinanceCancelOrder(&common.APIError{Code: -2010, Message: "Order rejected"}) {
		t.Fatal("different Binance errors must not be swallowed")
	}
}

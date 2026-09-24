package feature

import (
	"strings"
	"testing"

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

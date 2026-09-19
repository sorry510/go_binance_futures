package feature

import (
	"strings"
	"testing"

	futuresownership "go_binance_futures/service/futuresownership"
)

func TestShouldRefreshOwnershipAccountQuantities(t *testing.T) {
	tests := []struct {
		name                string
		activeManagedOrders int
		want                bool
	}{
		{name: "stable ownership uses existing snapshot", activeManagedOrders: 0, want: false},
		{name: "active order refreshes after reconcile", activeManagedOrders: 1, want: true},
		{name: "multiple active orders still require one refresh", activeManagedOrders: 5, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldRefreshOwnershipAccountQuantities(tt.activeManagedOrders); got != tt.want {
				t.Fatalf("shouldRefreshOwnershipAccountQuantities(%d) = %v, want %v", tt.activeManagedOrders, got, tt.want)
			}
		})
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

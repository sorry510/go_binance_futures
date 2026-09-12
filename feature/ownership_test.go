package feature

import "testing"

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

package feature

import (
	"testing"

	"go_binance_futures/models"
	"go_binance_futures/types"
)

func TestOwnershipSnapshotRequiresFreshExchangeProofForShrinking(t *testing.T) {
	managed := []models.FuturesManagedPosition{
		{Symbol: "HYPEUSDT", PositionSide: "SHORT", ManagedQty: 1.04},
		{Symbol: "APTUSDT", PositionSide: "SHORT", ManagedQty: 120.3},
	}
	quantities := map[string]float64{
		"HYPEUSDT|SHORT": 1.04,
		"APTUSDT|SHORT":  120.3,
	}
	if needsFreshOwnershipSnapshot(managed, quantities, false) {
		t.Fatal("unchanged live positions must not require a duplicate REST read")
	}
	delete(quantities, "HYPEUSDT|SHORT")
	if !needsFreshOwnershipSnapshot(managed, quantities, false) {
		t.Fatal("missing position requires fresh exchange verification")
	}
	quantities["HYPEUSDT|SHORT"] = 1.04
	quantities["APTUSDT|SHORT"] = 120.2
	if !needsFreshOwnershipSnapshot(managed, quantities, false) {
		t.Fatal("smaller quantity requires fresh verification")
	}
	quantities["APTUSDT|SHORT"] = 121
	if needsFreshOwnershipSnapshot(managed, quantities, false) {
		t.Fatal("manual increase does not require a destructive reconciliation")
	}
	if !needsFreshOwnershipSnapshot(managed, quantities, true) {
		t.Fatal("newly confirmed fill must force a refresh")
	}
}

func TestOwnershipSnapshotRejectsMalformedNumbers(t *testing.T) {
	for _, invalid := range []string{"not-a-number", "NaN", "Inf", "-Inf"} {
		_, err := ownershipAccountQuantities([]types.FuturesPosition{
			{Symbol: "HYPEUSDT", Side: "SHORT", Amount: invalid},
		})
		if err == nil {
			t.Fatalf("malformed %q must not silently turn into zero", invalid)
		}
	}
}

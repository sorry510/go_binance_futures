package feature

import (
	"context"
	"errors"
	"testing"

	"go_binance_futures/models"
	futuresownership "go_binance_futures/service/futuresownership"
	"go_binance_futures/types"
)

func TestUnresolvedOpenReadFailureOnlyProtectsAffectedLeg(t *testing.T) {
	orders := []models.FuturesManagedOrder{{
		Symbol: "HYPEUSDT", PositionSide: "SHORT", Intent: futuresownership.IntentOpen,
		ClientOrderID: "aut_hype_pending",
	}}
	calls := 0
	load := func(_ context.Context, id string) (models.FuturesManagedOrder, error) {
		calls++
		if id == "aut_hype_pending" {
			return models.FuturesManagedOrder{}, errors.New("temporary database read failure")
		}
		return models.FuturesManagedOrder{Status: futuresownership.OrderFilled}, nil
	}
	hype := models.FuturesManagedPosition{Symbol: "HYPEUSDT", PositionSide: "SHORT"}
	apt := models.FuturesManagedPosition{Symbol: "APTUSDT", PositionSide: "SHORT"}
	if !autoStrategyOpenUnresolved(context.Background(), hype, orders, load) {
		t.Fatal("failed read must preserve HYPE ownership")
	}
	if autoStrategyOpenUnresolved(context.Background(), apt, orders, load) {
		t.Fatal("failed read for HYPE must not block APT exits")
	}
	if calls != 1 {
		t.Fatalf("unexpected order lookups: %d", calls)
	}
}

func TestVerifiedOwnershipSnapshotRestoresMissingExitRows(t *testing.T) {
	local := []types.FuturesPosition{{Symbol: "APTUSDT", Side: "SHORT", Amount: "-120.3"}}
	managed := []models.FuturesManagedPosition{{Symbol: "HYPEUSDT", PositionSide: "SHORT", ManagedQty: 1.04}}
	fetches := 0
	fetch := func() ([]types.FuturesPosition, error) {
		fetches++
		return []types.FuturesPosition{
			{Symbol: "HYPEUSDT", Side: "SHORT", Amount: "-1.04", MarkPrice: "21", Leverage: 4},
			{Symbol: "APTUSDT", Side: "SHORT", Amount: "-120.3", MarkPrice: "6", Leverage: 4},
		}, nil
	}
	snapshot, err := resolveOwnershipAccountSnapshot(local, managed, false, fetch)
	if err != nil || !snapshot.fresh || fetches != 1 {
		t.Fatalf("must read fresh snapshot when an owned leg disappears: %+v err=%v calls=%d", snapshot, err, fetches)
	}
	if snapshot.quantities["HYPEUSDT|SHORT"] != 1.04 || len(snapshot.positions) != 2 {
		t.Fatalf("a fresh REST row must be available to the exit loop, not only its qty: %+v", snapshot)
	}
	if snapshot.positions[0].MarkPrice != "21" {
		t.Fatal("restored exit position must retain fresh mark price")
	}
	// Subsequent stable snapshots avoid duplicating REST calls.
	snapshot, err = resolveOwnershipAccountSnapshot(snapshot.positions, managed, false, fetch)
	if err != nil || snapshot.fresh || fetches != 1 {
		t.Fatalf("stable cache should not refresh again: %+v err=%v calls=%d", snapshot, err, fetches)
	}
}

func TestMalformedMirrorQuantityRecoversFromRestWithoutLosingExit(t *testing.T) {
	local := []types.FuturesPosition{{Symbol: "HYPEUSDT", Side: "SHORT", Amount: "invalid"}}
	managed := []models.FuturesManagedPosition{{Symbol: "HYPEUSDT", PositionSide: "SHORT", ManagedQty: 1.04}}
	snapshot, err := resolveOwnershipAccountSnapshot(local, managed, false, func() ([]types.FuturesPosition, error) {
		return []types.FuturesPosition{{Symbol: "HYPEUSDT", Side: "SHORT", Amount: "-1.04"}}, nil
	})
	if err != nil || !snapshot.fresh || snapshot.quantities["HYPEUSDT|SHORT"] != 1.04 {
		t.Fatalf("malformed mirror must recover safely: %+v err=%v", snapshot, err)
	}
	_, err = resolveOwnershipAccountSnapshot(local, managed, false, func() ([]types.FuturesPosition, error) {
		return nil, errors.New("exchange unavailable")
	})
	if err == nil {
		t.Fatal("cannot silently treat a failed fresh read as empty positions")
	}
}

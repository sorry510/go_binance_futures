package feature

import (
	"testing"

	"go_binance_futures/types"

	"github.com/adshao/go-binance/v2/futures"
)

func TestTradeCycleAccountSnapshotTracksPendingOpensAndMaxCount(t *testing.T) {
	snapshot := newTradeCycleAccountSnapshot(
		[]types.FuturesPosition{{Symbol: "BTCUSDT", Side: "LONG", Amount: "0.1"}},
		[]types.FuturesOrder{{Symbol: "ETHUSDT", Side: "SELL", PositionSide: "SHORT", Status: "NEW", OrderId: 10}},
	)
	if snapshot.AccountSlotCount() != 2 {
		t.Fatalf("initial slot count=%d want=2", snapshot.AccountSlotCount())
	}
	if !snapshot.HasPosition("BTCUSDT", futures.PositionSideTypeLong) {
		t.Fatal("BTC LONG position missing")
	}
	if !snapshot.HasOpeningOrder("ETHUSDT", futures.PositionSideTypeShort) {
		t.Fatal("ETH SHORT opening order missing")
	}

	snapshot.RecordPendingOpen("SOLUSDT", futures.SideTypeBuy, futures.PositionSideTypeLong, 11)
	if snapshot.AccountSlotCount() != 3 {
		t.Fatalf("slot count after pending open=%d want=3", snapshot.AccountSlotCount())
	}
	if !snapshot.HasOpeningOrder("SOLUSDT", futures.PositionSideTypeLong) {
		t.Fatal("SOL pending open missing")
	}

	// Recording the same slot again must not consume another account slot.
	snapshot.RecordPendingOpen("SOLUSDT", futures.SideTypeBuy, futures.PositionSideTypeLong, 12)
	if snapshot.AccountSlotCount() != 3 {
		t.Fatalf("duplicate pending slot increased count=%d", snapshot.AccountSlotCount())
	}
}

func TestTradeCycleSnapshotKeepsLongAndShortIndependent(t *testing.T) {
	snapshot := newTradeCycleAccountSnapshot(nil, nil)
	snapshot.RecordPendingOpen("BTCUSDT", futures.SideTypeBuy, futures.PositionSideTypeLong, 1)
	if snapshot.HasOpeningOrder("BTCUSDT", futures.PositionSideTypeShort) {
		t.Fatal("LONG pending open must not block SHORT slot in snapshot")
	}
}

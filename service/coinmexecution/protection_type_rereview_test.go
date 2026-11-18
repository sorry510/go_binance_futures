package coinmexecution

import (
	"context"
	"testing"
)

// O3: even a remote Algo with the correct stable client ID, exchange Algo ID,
// symbol, position side and BUY/SELL direction is not an owned STOP if its
// actual order type disagrees with the locally managed STOP.
func TestCM5RemoteProtectiveOrderTypeMustMatchBeforeSecondAlgo(t *testing.T) {
	cases := []struct{ name, remoteType string }{
		{"wrong_type", "TAKE_PROFIT_MARKET"},
		{"empty_type", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := cm5Fixture(t)
			api := newMockCM5()
			proc := NewProcessor(db, api)
			ctx := context.Background()
			entry, err := proc.Open(ctx, OpenRequest{Symbol: "BTCUSD_PERP", PositionSide: "LONG", ClientOrderID: "cm5typecheckopen", Type: "MARKET"})
			if err != nil {
				t.Fatal(err)
			}
			api.qty["LONG"] = "10"
			if _, err = proc.ApplyFill(sampleTrade(api.orders[entry.ClientOrderID], 20001, "10", "50000", "0")); err != nil {
				t.Fatal(err)
			}
			stop, err := proc.PlaceProtection(ctx, ProtectionRequest{PositionID: entry.PositionID, ClientAlgoID: "cm5typecheckstop", Type: "STOP_MARKET", TriggerPrice: "45000"})
			if err != nil || stop.Status != "ACTIVE" || len(api.openAlgos) != 1 {
				t.Fatalf("valid STOP setup: %+v err=%v", stop, err)
			}
			api.openAlgos[0].OrderType = tc.remoteType
			if _, err = proc.PlaceProtection(ctx, ProtectionRequest{PositionID: entry.PositionID, ClientAlgoID: "cm5typechecktake", Type: "TAKE_PROFIT_MARKET", TriggerPrice: "55000"}); err == nil {
				t.Fatal("mismatched exchange Algo type accepted as owned protection")
			}
			if len(api.openAlgos) != 1 {
				t.Fatalf("unknown Algo type caused remote new POST: %+v", api.openAlgos)
			}
			expectCM5(t, db, "SELECT COUNT(*) FROM coin_m_managed_orders WHERE intent='PROTECT'", 1)
			saved, err := proc.FindOrder(stop.ClientOrderID)
			if err != nil || saved.Status != "ACTIVE" {
				t.Fatalf("legitimate STOP changed during rejected mismatch: %+v %v", saved, err)
			}
		})
	}
}

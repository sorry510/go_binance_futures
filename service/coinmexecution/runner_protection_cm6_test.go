package coinmexecution

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCM6StopTakeProfitRemainOwnedUntilConfirmedStrategyExit(t *testing.T) {
	r, p, api, strategy, db, now := cm6RunnerFixture(t)
	ctx := context.Background()
	opened, err := p.Open(ctx, OpenRequest{
		Symbol: "BTCUSD_PERP", PositionSide: "LONG", Type: "MARKET", ClientOrderID: "cm6ownedpairentry",
	})
	if err != nil {
		t.Fatal(err)
	}
	api.qty["LONG"] = "10"
	if _, err = p.ApplyFill(sampleTrade(api.orders[opened.ClientOrderID], 70601, "10", "50000", "0")); err != nil {
		t.Fatal(err)
	}
	stop, err := p.PlaceProtection(ctx, ProtectionRequest{PositionID: opened.PositionID, ClientAlgoID: "cm6ownedstop", Type: "STOP_MARKET", TriggerPrice: "49000"})
	if err != nil {
		t.Fatal(err)
	}
	take, err := p.PlaceProtection(ctx, ProtectionRequest{PositionID: opened.PositionID, ClientAlgoID: "cm6ownedtake", Type: "TAKE_PROFIT_MARKET", TriggerPrice: "51000"})
	if err != nil {
		t.Fatal(err)
	}
	if len(api.openAlgos) != 2 {
		t.Fatalf("missing two verified protective orders: %d", len(api.openAlgos))
	}
	if _, err = db.Raw("UPDATE config SET coin_m_quant_enable=0").Exec(); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Raw("UPDATE delivery_symbols SET enable=0").Exec(); err != nil {
		t.Fatal(err)
	}
	strategy.exit = true
	// The shared runner reconciles each Algo before evaluating the exit.
	for i := 0; i < 3; i++ {
		*now = now.Add(2 * time.Second)
		if err := r.Tick(ctx); err != nil {
			t.Fatalf("reconcile/exit iteration %d: %v", i, err)
		}
	}
	if strategy.exits != 1 || strategy.entries != 0 {
		t.Fatalf("entry OFF must still evaluate managed close: entries=%d exits=%d", strategy.entries, strategy.exits)
	}
	if len(api.calls) != 2 || api.calls[1].Side != "SELL" || api.calls[1].PositionSide != "LONG" {
		t.Fatalf("cancel-own-protect-then-CLOSE executed wrong request: %+v", api.calls)
	}
	if len(api.openAlgos) != 0 {
		t.Fatalf("owned STOP/TP still live after confirmed strategy close: %+v", api.openAlgos)
	}
	oldStop, e1 := p.FindOrder(stop.ClientOrderID)
	oldTake, e2 := p.FindOrder(take.ClientOrderID)
	if e1 != nil || e2 != nil || oldStop.Status != "CANCELED" || oldTake.Status != "CANCELED" {
		t.Fatalf("protectors not confirmed CANCELED: %+v %+v %v %v", oldStop, oldTake, e1, e2)
	}
	expectCM5(t, db, "SELECT COUNT(*) FROM coin_m_managed_orders WHERE intent='CLOSE'", 1)
}
func TestCM6StopTakeProfitMustNotCancelWhenWSDisconnected(t *testing.T) {
	r, p, api, strategy, _, now := cm6RunnerFixture(t)
	ctx := context.Background()
	opened, err := p.Open(ctx, OpenRequest{
		Symbol: "BTCUSD_PERP", PositionSide: "LONG", Type: "MARKET", ClientOrderID: "cm6streamholdentry",
	})
	if err != nil {
		t.Fatal(err)
	}
	api.qty["LONG"] = "10"
	if _, err = p.ApplyFill(sampleTrade(api.orders[opened.ClientOrderID], 70602, "10", "50000", "0")); err != nil {
		t.Fatal(err)
	}
	if _, err = p.PlaceProtection(ctx, ProtectionRequest{PositionID: opened.PositionID, ClientAlgoID: "cm6streamholdstop", Type: "STOP_MARKET", TriggerPrice: "49000"}); err != nil {
		t.Fatal(err)
	}
	if _, err = p.PlaceProtection(ctx, ProtectionRequest{PositionID: opened.PositionID, ClientAlgoID: "cm6streamholdtake", Type: "TAKE_PROFIT_MARKET", TriggerPrice: "51000"}); err != nil {
		t.Fatal(err)
	}
	api.ready = false
	strategy.exit = true
	for i := 0; i < 3; i++ {
		*now = now.Add(2 * time.Second)
		if err := r.Tick(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if len(api.openAlgos) != 2 || len(api.calls) != 1 {
		t.Fatalf("private WS disconnect canceled live protection or placed a CLOSE: %+v calls=%d", api.openAlgos, len(api.calls))
	}
}
func TestCM6UncertainOpeningResponseNeverResubmits(t *testing.T) {
	r, _, api, _, db, now := cm6RunnerFixture(t)
	api.failCreate = true
	if err := r.Tick(context.Background()); !errors.Is(err, ErrUncertain) {
		t.Fatalf("POST timeout must be reported as uncertain, got %v", err)
	}
	if len(api.calls) != 1 {
		t.Fatalf("expected original POST exactly once: %d", len(api.calls))
	}
	expectCM5(t, db, "SELECT COUNT(*) FROM coin_m_managed_orders WHERE status='UNKNOWN'", 1)
	*now = now.Add(12 * time.Second)
	if err := r.Tick(context.Background()); err != nil && !errors.Is(err, ErrUncertain) {
		t.Fatalf("uncertain original order must be recovered without POST: %v", err)
	}
	if len(api.calls) != 1 {
		t.Fatalf("ambiguous POST was replayed: %+v", api.calls)
	}
}

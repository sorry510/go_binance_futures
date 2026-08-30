package signal

import (
	"context"
	"testing"
	"time"

	agentevent "go_binance_futures/agent/event"
)

func TestEngineCreatesFastMoveSignalWithCooldown(t *testing.T) {
	bus := agentevent.NewBus(agentevent.BusConfig{Buffer: 16, Workers: 1})
	settings := DefaultSettings()
	settings.FastMoveThresholdPct = 10
	settings.FastMoveRecoverPct = 8
	settings.FastMoveCooldown = time.Minute
	settings.FastMoveWindows = []Window{{Name: "3m", Duration: 3 * time.Minute}}
	signals := make(chan Signal, 4)
	engine, err := NewEngine(bus, func() Settings { return settings }, func(value Signal) bool {
		signals <- value
		return true
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bus.Start(ctx)
	base := time.UnixMilli(1_700_000_000_000)
	bus.Publish(agentevent.NewPriceTick("BTCUSDT", "test", base.UnixMilli(), 100, 0))
	// Real WS events do not land exactly on the configured window boundary.
	// The sample immediately before the cutoff must remain available as the base.
	triggerAt := base.Add(3*time.Minute + 137*time.Millisecond)
	bus.Publish(agentevent.NewPriceTick("BTCUSDT", "test", triggerAt.UnixMilli(), 111, 11))
	select {
	case got := <-signals:
		if got.Type != TypeFastMove || got.Symbol != "BTCUSDT" || got.Labels["direction"] != "up" {
			t.Fatalf("unexpected signal: %+v", got)
		}
		if got.Metrics["change_percent"] < 10.9 {
			t.Fatalf("unexpected change: %+v", got.Metrics)
		}
	case <-time.After(time.Second):
		t.Fatal("fast move signal was not emitted")
	}
	bus.Publish(agentevent.NewPriceTick("BTCUSDT", "test", triggerAt.Add(time.Second).UnixMilli(), 112, 12))
	select {
	case duplicate := <-signals:
		t.Fatalf("cooldown should suppress duplicate: %+v", duplicate)
	case <-time.After(30 * time.Millisecond):
	}
	if engine.Stats().FastMoveSignals != 1 {
		t.Fatalf("unexpected stats: %+v", engine.Stats())
	}
}

func TestEngineAggregatesLiquidationSignalAndConsumesWindow(t *testing.T) {
	bus := agentevent.NewBus(agentevent.BusConfig{Buffer: 16, Workers: 1})
	settings := DefaultSettings()
	settings.LiquidationThreshold = 100
	settings.LiquidationWindow = time.Minute
	settings.LiquidationCooldown = time.Second
	signals := make(chan Signal, 4)
	_, err := NewEngine(bus, func() Settings { return settings }, func(value Signal) bool {
		signals <- value
		return true
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bus.Start(ctx)
	base := time.UnixMilli(1_700_000_000_000)
	first := agentevent.LiquidationEvent{
		Meta:      agentevent.NewMetadata(agentevent.TypeLiquidation, "BTCUSDT", base.UnixMilli(), "test"),
		OrderSide: "SELL", Notional: 60, LiquidationSide: "long",
	}
	second := agentevent.LiquidationEvent{
		Meta:      agentevent.NewMetadata(agentevent.TypeLiquidation, "BTCUSDT", base.Add(time.Second).UnixMilli(), "test"),
		OrderSide: "SELL", Notional: 50, LiquidationSide: "long",
	}
	bus.Publish(first)
	bus.Publish(second)
	select {
	case got := <-signals:
		if got.Type != TypeLiquidationSpike || got.Metrics["aggregate_notional"] != 110 || got.Labels["liquidation_side"] != "long" {
			t.Fatalf("unexpected liquidation signal: %+v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("liquidation signal was not emitted")
	}
	third := agentevent.LiquidationEvent{
		Meta:      agentevent.NewMetadata(agentevent.TypeLiquidation, "BTCUSDT", base.Add(2*time.Second).UnixMilli(), "test"),
		OrderSide: "SELL", Notional: 100, LiquidationSide: "long",
	}
	bus.Publish(third)
	select {
	case got := <-signals:
		if got.Metrics["aggregate_notional"] != 100 {
			t.Fatalf("previous window should have been consumed: %+v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("second liquidation signal was not emitted")
	}
}

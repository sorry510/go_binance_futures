package event

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestBusPublishesWithoutBlockingAndTracksErrors(t *testing.T) {
	bus := NewBus(BusConfig{Buffer: 4, Workers: 1})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var handled atomic.Int64
	if err := bus.Subscribe(TypePriceTick, func(context.Context, Event) error {
		handled.Add(1)
		return errors.New("test handler error")
	}); err != nil {
		t.Fatal(err)
	}
	bus.Start(ctx)
	if !bus.Publish(NewPriceTick("btcusdt", "test", time.Now().UnixMilli(), 100, 1)) {
		t.Fatal("expected event to be accepted")
	}
	deadline := time.Now().Add(time.Second)
	for handled.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if handled.Load() != 1 {
		t.Fatalf("handled = %d, want 1", handled.Load())
	}
	stats := bus.Stats()
	if stats.Published != 1 || stats.HandlerErrors != 1 || stats.Handled != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

func TestBusDropsWhenBufferIsFull(t *testing.T) {
	bus := NewBus(BusConfig{Buffer: 1, Workers: 1})
	first := NewPriceTick("BTCUSDT", "test", 1, 100, 0)
	second := NewPriceTick("ETHUSDT", "test", 2, 100, 0)
	if !bus.Publish(first) {
		t.Fatal("first event should be queued")
	}
	if bus.Publish(second) {
		t.Fatal("second event should be dropped before workers start")
	}
	stats := bus.Stats()
	if stats.Dropped != 1 || stats.QueueDepth != 1 {
		t.Fatalf("unexpected drop stats: %+v", stats)
	}
}

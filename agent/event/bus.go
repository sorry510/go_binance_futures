package event

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

type Handler func(context.Context, Event) error

type BusConfig struct {
	Buffer  int
	Workers int
}

type Stats struct {
	Published     uint64 `json:"published"`
	Dropped       uint64 `json:"dropped"`
	Handled       uint64 `json:"handled"`
	HandlerErrors uint64 `json:"handler_errors"`
	QueueDepth    int    `json:"queue_depth"`
	QueueCapacity int    `json:"queue_capacity"`
}

type Bus struct {
	queue         chan Event
	workers       int
	handlersMu    sync.RWMutex
	handlers      map[Type][]Handler
	startOnce     sync.Once
	published     atomic.Uint64
	dropped       atomic.Uint64
	handled       atomic.Uint64
	handlerErrors atomic.Uint64
}

func NewBus(cfg BusConfig) *Bus {
	if cfg.Buffer <= 0 {
		cfg.Buffer = 4096
	}
	if cfg.Workers <= 0 {
		cfg.Workers = 2
	}
	return &Bus{
		queue:    make(chan Event, cfg.Buffer),
		workers:  cfg.Workers,
		handlers: make(map[Type][]Handler),
	}
}

func (bus *Bus) Subscribe(eventType Type, handler Handler) error {
	if bus == nil || handler == nil {
		return fmt.Errorf("event bus and handler are required")
	}
	bus.handlersMu.Lock()
	defer bus.handlersMu.Unlock()
	bus.handlers[eventType] = append(bus.handlers[eventType], handler)
	return nil
}

func (bus *Bus) Publish(value Event) bool {
	if bus == nil || value == nil {
		return false
	}
	bus.published.Add(1)
	select {
	case bus.queue <- value:
		return true
	default:
		bus.dropped.Add(1)
		return false
	}
}

func (bus *Bus) Start(ctx context.Context) {
	if bus == nil {
		return
	}
	bus.startOnce.Do(func() {
		for index := 0; index < bus.workers; index++ {
			go bus.worker(ctx)
		}
	})
}

func (bus *Bus) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case value := <-bus.queue:
			bus.dispatch(ctx, value)
		}
	}
}

func (bus *Bus) dispatch(ctx context.Context, value Event) {
	meta := value.Metadata()
	bus.handlersMu.RLock()
	handlers := append([]Handler(nil), bus.handlers[meta.Type]...)
	bus.handlersMu.RUnlock()
	for _, handler := range handlers {
		if err := handler(ctx, value); err != nil {
			bus.handlerErrors.Add(1)
		}
		bus.handled.Add(1)
	}
}

func (bus *Bus) Stats() Stats {
	if bus == nil {
		return Stats{}
	}
	return Stats{
		Published:     bus.published.Load(),
		Dropped:       bus.dropped.Load(),
		Handled:       bus.handled.Load(),
		HandlerErrors: bus.handlerErrors.Load(),
		QueueDepth:    len(bus.queue),
		QueueCapacity: cap(bus.queue),
	}
}

var defaultBus = NewBus(BusConfig{Buffer: 8192, Workers: 4})

func DefaultBus() *Bus {
	return defaultBus
}

package coinmexecution

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/adshao/go-binance/v2/delivery"
)

func TestCM5UserStreamExplicitAuthLifecycleAndReconnect(t *testing.T) {
	var starts, keeps, closes, events atomic.Int64
	ctx, cancel := context.WithTimeout(context.Background(), 350*time.Millisecond)
	defer cancel()
	stream := NewUserStream(func(e *delivery.WsUserDataEvent) error {
		if e.Event == delivery.UserDataEventTypeOrderTradeUpdate {
			events.Add(1)
		}
		return nil
	})
	if stream.Active() {
		t.Fatal("new COIN-M private stream appeared connected without a session")
	}
	tport := coinMPrivateStreamTransport{
		start: func(context.Context) (string, error) {
			starts.Add(1)
			return "fake-testnet-listen-key", nil
		},
		keepalive: func(context.Context, string) error { keeps.Add(1); return nil },
		close:     func(context.Context, string) error { closes.Add(1); return nil },
		serve: func(_ string, h delivery.WsUserDataHandler, errHandler delivery.ErrHandler) (chan struct{}, chan struct{}, error) {
			done := make(chan struct{})
			stop := make(chan struct{}, 1)
			go func() {
				e := &delivery.WsUserDataEvent{Event: delivery.UserDataEventTypeOrderTradeUpdate}
				h(e)
				// Session 1 is deliberately terminated by exchange after an event.
				// Later sessions stay alive until caller cancels.
				if starts.Load() == 1 {
					h(&delivery.WsUserDataEvent{Event: delivery.UserDataEventTypeListenKeyExpired})
				}
				select {
				case <-stop:
				case <-ctx.Done():
				}
				close(done)
			}()
			return done, stop, nil
		},
	}
	done := make(chan error, 1)
	go func() { done <- stream.run(ctx, tport, 7*time.Millisecond, 4*time.Millisecond) }()
	deadline := time.NewTimer(180 * time.Millisecond)
	defer deadline.Stop()
	for starts.Load() < 2 || !stream.Active() {
		select {
		case <-deadline.C:
			t.Fatalf("expired listenKey did not re-establish an active private session: starts=%d ready=%t", starts.Load(), stream.Active())
		case <-time.After(2 * time.Millisecond):
		}
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("private WS failed to stop on cancel")
	}
	if stream.Active() {
		t.Fatal("private WS reports active after cancellation")
	}
	if starts.Load() < 2 || events.Load() < 2 || closes.Load() < 1 {
		t.Fatalf("COIN-M private WS startup, expiry, or shutdown missing: starts=%d events=%d close=%d",
			starts.Load(), events.Load(), closes.Load())
	}
}
func TestCM5UserStreamNeverReadyWithoutAuthOrHandler(t *testing.T) {
	tport := coinMPrivateStreamTransport{
		start:     func(context.Context) (string, error) { return "", errors.New("no COIN-M testnet account credentials") },
		keepalive: func(context.Context, string) error { return nil },
		close:     func(context.Context, string) error { return nil },
		serve: func(string, delivery.WsUserDataHandler, delivery.ErrHandler) (chan struct{}, chan struct{}, error) {
			t.Fatal("should never dial WS without listenKey")
			return nil, nil, nil
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	stream := NewUserStream(func(*delivery.WsUserDataEvent) error { return nil })
	if err := stream.run(ctx, tport, 5*time.Millisecond, 5*time.Millisecond); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("WS missing private credentials should only exit on context cancellation: %v", err)
	}
	if stream.Active() {
		t.Fatal("private WS became ready without listenKey")
	}
	if err := NewUserStream(nil).Run(ctx); err == nil {
		t.Fatal("nil event handler accepted")
	}
}

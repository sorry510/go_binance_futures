package coinmexecution

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/adshao/go-binance/v2/delivery"
	"go_binance_futures/feature/api/binance"
)

// UserStream is deliberately not automatically started during application
// bootstrap. CM-6 must explicitly wire it after checking the environment.
// A public market WS is never sufficient to authorize a real order.
type UserStream struct {
	active    atomic.Bool
	started   atomic.Bool
	lastEvent atomic.Int64
	onEvent   func(*delivery.WsUserDataEvent) error
}

func NewUserStream(onEvent func(*delivery.WsUserDataEvent) error) *UserStream {
	return &UserStream{onEvent: onEvent}
}
func (s *UserStream) Active() bool { return s != nil && s.active.Load() && s.started.Load() }
func (s *UserStream) LastEventTime() int64 {
	if s == nil {
		return 0
	}
	return s.lastEvent.Load()
}

type coinMPrivateStreamTransport struct {
	start     func(context.Context) (string, error)
	keepalive func(context.Context, string) error
	close     func(context.Context, string) error
	serve     func(string, delivery.WsUserDataHandler, delivery.ErrHandler) (chan struct{}, chan struct{}, error)
}

func defaultCoinMPrivateStreamTransport() coinMPrivateStreamTransport {
	return coinMPrivateStreamTransport{
		start:     binance.StartCoinMUserStreamContext,
		keepalive: binance.KeepaliveCoinMUserStreamContext,
		close:     binance.CloseCoinMUserStreamContext,
		serve:     binance.CoinMUserDataServe,
	}
}

// Run blocks on a single private listenKey session, reconnecting after
// disconnect/expired key. It can be cancelled and has no trading side effect.
func (s *UserStream) Run(ctx context.Context) error {
	return s.run(ctx, defaultCoinMPrivateStreamTransport(), 25*time.Minute, 5*time.Second)
}
func (s *UserStream) run(ctx context.Context, t coinMPrivateStreamTransport, refresh, retry time.Duration) error {
	if s == nil || s.onEvent == nil {
		return errors.New("COIN-M private event handler required")
	}
	if ctx == nil {
		return errors.New("COIN-M stream context required")
	}
	if !s.started.CompareAndSwap(false, true) {
		return errors.New("COIN-M private stream already running")
	}
	defer s.started.Store(false)
	defer s.active.Store(false)
	if refresh <= 0 || retry <= 0 {
		return errors.New("invalid COIN-M stream cadence")
	}
	for ctx.Err() == nil {
		sessionCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		key, err := t.start(sessionCtx)
		cancel()
		if err == nil && key == "" {
			err = errors.New("empty COIN-M listenKey")
		}
		if err != nil {
			s.active.Store(false)
			if !pauseStream(ctx, retry) {
				return ctx.Err()
			}
			continue
		}
		failure := make(chan error, 1)
		handler := func(event *delivery.WsUserDataEvent) {
			if event == nil {
				return
			}
			if event.Event == delivery.UserDataEventTypeListenKeyExpired {
				s.active.Store(false)
				select {
				case failure <- errors.New("COIN-M listenKey expired"):
				default:
				}
				return
			}
			if event.Event == "ALGO_UPDATE" {
				// SDK does not parse Algo child IDs; never misattribute child fills.
				s.active.Store(false)
				select {
				case failure <- errors.New("COIN-M Algo update needs signed reconciliation"):
				default:
				}
				return
			}
			if err := s.onEvent(event); err != nil {
				s.active.Store(false)
				select {
				case failure <- fmt.Errorf("COIN-M private event not reconciled: %w", err):
				default:
				}
				return
			}
			s.lastEvent.Store(time.Now().UnixMilli())
		}
		done, stop, err := t.serve(key, handler, func(error) {
			s.active.Store(false)
			select {
			case failure <- errors.New("COIN-M private WS disconnected"):
			default:
			}
		})
		if err != nil || done == nil || stop == nil {
			s.active.Store(false)
			closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = t.close(closeCtx, key)
			cancel()
			if !pauseStream(ctx, retry) {
				return ctx.Err()
			}
			continue
		}
		s.active.Store(true)
		keep := time.NewTicker(refresh)
		sessionDone := false
		for !sessionDone {
			select {
			case <-ctx.Done():
				sessionDone = true
			case <-done:
				sessionDone = true
			case <-failure:
				sessionDone = true
			case <-keep.C:
				keepCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
				e := t.keepalive(keepCtx, key)
				cancel()
				if e != nil {
					sessionDone = true
				}
			}
		}
		keep.Stop()
		s.active.Store(false)
		select {
		case stop <- struct{}{}:
		default:
		}
		select {
		case <-done:
		case <-time.After(3 * time.Second):
		}
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = t.close(closeCtx, key)
		closeCancel()
		if !pauseStream(ctx, retry) {
			return ctx.Err()
		}
	}
	return ctx.Err()
}
func pauseStream(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

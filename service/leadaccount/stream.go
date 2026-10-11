package leadaccount

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	binance "go_binance_futures/feature/api/binance"
)

type LeadStreamAccount interface {
	GetListenKey(context.Context) (string, error)
	UpdateListenKey(context.Context, string) error
	CloseLeadListenKey(context.Context, string) error
}
type LeadStreamDial func(string, futures.WsUserDataHandler, futures.ErrHandler) (chan struct{}, chan struct{}, error)
type LeadUserStreamStatus struct {
	AccountID      binance.AccountID `json:"account_id"`
	Running        bool              `json:"running"`
	Healthy        bool              `json:"healthy"`
	Generation     uint64            `json:"generation"`
	LastSnapshotAt time.Time         `json:"last_snapshot_at"`
	Reconnects     uint64            `json:"reconnects"`
}
type LeadUserStream struct {
	mu                sync.Mutex
	account           LeadStreamAccount
	source            LeadMirrorRESTReader
	dial              LeadStreamDial
	mirror            *LeadAccountMirror
	running           bool
	currentCancel     context.CancelFunc
	currentGeneration uint64
	reconnects        uint64
	keepalive         time.Duration
	backoff           time.Duration
	onFault           func(LeadFaultCode)
	onEvent           func(*futures.WsUserDataEvent) // optional observable callback; no DB writes
	onReady           func(uint64)
}

func NewLeadUserStream(account *binance.AccountClient, mirror *LeadAccountMirror) (*LeadUserStream, error) {
	if account == nil || account.ID() != binance.LeadAccountID || mirror == nil {
		return nil, fmt.Errorf("Lead WS needs dedicated Lead client and mirror")
	}
	return &LeadUserStream{account: account, source: account, dial: account.DialLeadUserStream, mirror: mirror, keepalive: 20 * time.Minute, backoff: 3 * time.Second}, nil
}

// Run is explicitly opt-in and BLOCKING. No production startup hook exists;
// callers control cancellation. Main stream is never started or stopped here.
func (s *LeadUserStream) Run(ctx context.Context) error {
	if s == nil || s.account == nil || s.source == nil || s.dial == nil || s.mirror == nil {
		return ErrLeadMirrorUntrusted
	}
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return errors.New("Lead stream already running")
	}
	contextRun, cancel := context.WithCancel(ctx)
	s.running = true
	s.currentCancel = cancel
	s.mu.Unlock()
	defer func() {
		cancel()
		s.mu.Lock()
		s.running = false
		s.currentCancel = nil
		s.mu.Unlock()
	}()
	for contextRun.Err() == nil {
		s.runGeneration(contextRun)
		if contextRun.Err() != nil {
			break
		}
		s.mu.Lock()
		s.reconnects++
		delay := s.backoff
		s.mu.Unlock()
		if delay <= 0 {
			delay = time.Second
		}
		delay += time.Duration(s.Status().Generation%7) * 50 * time.Millisecond
		timer := time.NewTimer(delay)
		select {
		case <-contextRun.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
		case <-timer.C:
		}
	}
	return contextRun.Err()
}
func (s *LeadUserStream) Stop() {
	if s == nil {
		return
	}
	s.mu.Lock()
	cancel := s.currentCancel
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}
func (s *LeadUserStream) runGeneration(ctx context.Context) {
	gen := s.mirror.BeginGeneration()
	s.mu.Lock()
	s.currentGeneration = gen
	fault := s.onFault
	s.mu.Unlock()
	defer s.mirror.Invalidate(gen)
	report := func(code LeadFaultCode) {
		if fault != nil {
			fault(code)
		}
	}
	key, err := s.account.GetListenKey(ctx)
	if err != nil || key == "" {
		report(FaultWSUnavailable)
		return
	}
	defer func() {
		cleanCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = s.account.CloseLeadListenKey(cleanCtx, key)
	}()
	expired := make(chan struct{}, 1)
	doneErr := make(chan error, 1)
	var connected atomic.Bool
	var eventBeforeSnapshot atomic.Bool
	// Connect the private WS BEFORE taking the full REST account snapshot.
	// Any event during bootstrap renders that REST snapshot potentially
	// inconsistent. Fail closed and re-seed rather than silently losing it.
	doneC, stopC, err := s.dial(key, func(event *futures.WsUserDataEvent) {
		if event == nil {
			return
		}
		if event.Event == "listenKeyExpired" {
			select {
			case expired <- struct{}{}:
			default:
			}
			return
		}
		if !connected.Load() {
			eventBeforeSnapshot.Store(true)
			return
		}
		if ok := s.mirror.ApplyEvent(gen, event); ok {
			s.mu.Lock()
			cb := s.onEvent
			s.mu.Unlock()
			if cb != nil {
				cb(event)
			}
		}
	}, func(err error) {
		select {
		case doneErr <- err:
		default:
		}
	})
	if err != nil {
		report(FaultWSUnavailable)
		return
	}
	var once sync.Once
	stopSocket := func() {
		once.Do(func() {
			if stopC != nil {
				close(stopC)
			}
		})
	}
	defer stopSocket()
	if err := s.mirror.Seed(ctx, gen, s.source); err != nil {
		report(FaultSnapshotInvalid)
		return
	}
	connected.Store(true)
	// A handler already entered before 'connected' must not be ignored simply
	// because the snapshot was still in flight.
	if eventBeforeSnapshot.Load() {
		report(FaultSnapshotInvalid)
		return
	}
	s.mu.Lock()
	callback := s.onReady
	s.mu.Unlock()
	if callback != nil {
		callback(gen)
	}
	s.mu.Lock()
	period := s.keepalive
	s.mu.Unlock()
	if period <= 0 {
		period = 20 * time.Minute
	}
	ticker := time.NewTicker(period)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-expired:
			report(FaultWSUnavailable)
			return
		case <-doneC:
			report(FaultWSUnavailable)
			return
		case <-doneErr:
			report(FaultWSUnavailable)
			return
		case <-ticker.C:
			callCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			err := s.account.UpdateListenKey(callCtx, key)
			cancel()
			if err != nil {
				report(FaultWSUnavailable)
				return
			}
		}
	}
}
func (s *LeadUserStream) Status() LeadUserStreamStatus {
	status := LeadUserStreamStatus{AccountID: binance.LeadAccountID}
	if s == nil {
		return status
	}
	s.mu.Lock()
	status.Running = s.running
	status.Generation = s.currentGeneration
	status.Reconnects = s.reconnects
	s.mu.Unlock()
	if s.mirror != nil {
		v := s.mirror.Status()
		status.Healthy = status.Running && v.Ready && v.Generation == status.Generation
		status.LastSnapshotAt = v.CapturedAt
	}
	return status
}

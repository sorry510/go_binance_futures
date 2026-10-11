package leadaccount

import (
	"context"
	"errors"
	"sync"
	"time"

	binance "go_binance_futures/feature/api/binance"
)

var ErrLeadSupervisorAlreadyRunning = errors.New("Lead supervisor already running")

type LeadSupervisor struct {
	mu            sync.Mutex
	running       bool
	Adapter       *LeadExecutionAdapter
	Stream        *LeadUserStream
	Mirror        *LeadAccountMirror
	Alerts        *LeadAlertQueue
	Recovery      func(context.Context) (LeadRecoveryReport, error)
	Send          LeadAlertSend
	recoveryProof LeadRecoveryReport
	proofReady    bool
}

// NewLeadSupervisor is an inert constructor. It never opens Binance WS,
// changes DB, starts a goroutine or unlocks the Stage 4 read-only broker.
func NewLeadSupervisor(account *binance.AccountClient, symbols *binance.LeadSymbolCache) (*LeadSupervisor, error) {
	if account == nil || account.ID() != binance.LeadAccountID {
		return nil, ErrLeadMirrorUntrusted
	}
	adapter, err := NewLeadExecutionAdapter(account, symbols)
	if err != nil {
		return nil, err
	}
	auditor, err := NewLeadRecoveryAuditor(account)
	if err != nil {
		return nil, err
	}
	mirror := NewLeadAccountMirror()
	stream, err := NewLeadUserStream(account, mirror)
	if err != nil {
		return nil, err
	}
	queue := NewLeadAlertQueue(64)
	adapter.guard = NewLeadRuntimeGuard(queue.Publish)
	stream.onFault = adapter.ReportLeadFault
	supervisor := &LeadSupervisor{
		Adapter: adapter, Stream: stream, Mirror: mirror, Alerts: queue,
		Recovery: auditor.Reconcile, Send: SendLeadAlertToConfiguredChannel,
	}
	stream.onReady = func(gen uint64) {
		supervisor.mu.Lock()
		proof, complete := supervisor.recoveryProof, supervisor.proofReady
		supervisor.mu.Unlock()
		if complete {
			status := mirror.Status()
			if status.Generation == gen {
				adapter.completeAuthoritativeRecovery(proof, status)
			}
		}
	}
	return supervisor, nil
}

// Run executes an explicit, cancelable Lead read-only supervisor. Recovery
// runs BEFORE the stream. An unresolved ledger blocks opens even if WS heals.
// Alert worker has a single bounded lifecycle; no background processes remain.
func (s *LeadSupervisor) Run(ctx context.Context) error {
	if s == nil || s.Adapter == nil || s.Stream == nil || s.Mirror == nil || s.Alerts == nil || s.Recovery == nil || s.Send == nil {
		return ErrLeadMirrorUntrusted
	}
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return ErrLeadSupervisorAlreadyRunning
	}
	s.running = true
	s.mu.Unlock()
	defer func() { s.mu.Lock(); s.running = false; s.mu.Unlock() }()
	if s.Adapter.account != nil {
		// HTTP RoundTrip can be called while the adapter's execution mutex
		// is held. Never re-enter Adapter.ReportLeadFault from within that
		// request or the 429/418 path can deadlock its own caller.
		s.Adapter.account.SetLeadRateLimitHandler(func(code int) {
			if s.Adapter.guard == nil {
				return
			}
			switch code {
			case 429:
				s.Adapter.guard.RecordFault(binance.LeadAccountID, FaultAPI429)
			case 418:
				s.Adapter.guard.RecordFault(binance.LeadAccountID, FaultAPI418)
			}
		})
		defer s.Adapter.account.SetLeadRateLimitHandler(nil)
	}
	runCtx, cancel := context.WithCancel(ctx)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); _ = s.Alerts.Run(runCtx, s.Send) }()
	defer func() { cancel(); wg.Wait() }()
	s.mu.Lock()
	s.proofReady = false
	s.recoveryProof = LeadRecoveryReport{}
	s.mu.Unlock()
	report, err := s.Recovery(runCtx)
	if err != nil || report.Blocked || report.AccountID != binance.LeadAccountID {
		s.Adapter.ReportLeadFault(FaultPendingReconcile)
	} else {
		s.mu.Lock()
		s.recoveryProof = report
		s.proofReady = true
		s.mu.Unlock()
	}
	if runCtx.Err() != nil {
		return runCtx.Err()
	}
	return s.Stream.Run(runCtx)
}
func (s *LeadSupervisor) Status() (LeadStatusSnapshot, LeadUserStreamStatus, LeadUsageMetrics) {
	if s == nil {
		return NewLeadRuntimeGuard(nil).Status(), LeadUserStreamStatus{}, LeadUsageMetrics{}
	}
	return s.Adapter.RuntimeStatus(), s.Stream.Status(), LeadUsageSnapshotWithAlerts(nil, s.Mirror, s.Alerts, time.Now().UTC())
}

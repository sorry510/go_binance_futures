package leadaccount

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	binance "go_binance_futures/feature/api/binance"
	"go_binance_futures/service/futuresownership"
)

// ReadOnlyObserverEnv is opt-in. Without this exact value, nothing starts,
// no account credentials are loaded and no HTTP/WS/DB connection is made.
const ReadOnlyObserverEnv = "BINANCE_LEAD_OBSERVER_MODE"
const ReadOnlyObserverValue = "read_only"

type LeadReadOnlyInspection struct {
	AccountID          string    `json:"account_id"`
	ObservedAt         time.Time `json:"observed_at"`
	OwnershipPositions int       `json:"ownership_positions"`
	UnresolvedOrders   int       `json:"unresolved_orders"`
	ReadOnly           bool      `json:"read_only"`
	AuditRecorded      bool      `json:"audit_recorded"`
	Reconciled         bool      `json:"reconciled"`
	Reason             string    `json:"reason"`
}
type LeadReadOnlyHostStatus struct {
	AccountID  string                 `json:"account_id"`
	Running    bool                   `json:"running"`
	Stream     LeadUserStreamStatus   `json:"stream"`
	Mirror     LeadMirrorStatus       `json:"mirror"`
	Inspection LeadReadOnlyInspection `json:"inspection"`
	// Never imply that observation establishes identity/Portfolio/Stage7.
	LiveTradeAuthorized bool `json:"live_trade_authorized"`
}
type LeadReadOnlyHost struct {
	mu         sync.RWMutex
	stream     *LeadUserStream
	inspect    func(context.Context) (LeadReadOnlyInspection, error)
	inspection LeadReadOnlyInspection
	running    bool
	revoked    bool
	cancel     context.CancelFunc
	done       chan struct{}
}

func NewLeadReadOnlyHost(stream *LeadUserStream, inspect func(context.Context) (LeadReadOnlyInspection, error)) (*LeadReadOnlyHost, error) {
	if stream == nil || stream.mirror == nil || inspect == nil {
		return nil, ErrLeadMirrorUntrusted
	}
	return &LeadReadOnlyHost{stream: stream, inspect: inspect}, nil
}
func (h *LeadReadOnlyHost) Status() LeadReadOnlyHostStatus {
	value := LeadReadOnlyHostStatus{AccountID: "lead"}
	if h == nil {
		return value
	}
	h.mu.RLock()
	value.Running = h.running
	value.Inspection = h.inspection
	h.mu.RUnlock()
	value.Stream = h.stream.Status()
	value.Mirror = h.stream.mirror.Status()
	return value
}

// Run deliberately does not use LeadSupervisor.Recovery: that operation may
// apply fills to the DB. Stage 7-2 observation is restricted to read-only
// account and Ownership queries. An unknown order cannot be "fixed" here.
func (h *LeadReadOnlyHost) Run(ctx context.Context) error {
	if h == nil || h.stream == nil || h.inspect == nil {
		return ErrLeadMirrorUntrusted
	}
	h.mu.Lock()
	if h.revoked {
		h.mu.Unlock()
		return context.Canceled
	}
	if h.running {
		h.mu.Unlock()
		return ErrLeadSupervisorAlreadyRunning
	}
	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	h.cancel = cancel
	h.done = done
	h.running = true
	h.inspection = LeadReadOnlyInspection{AccountID: "lead", ReadOnly: true, Reconciled: false, Reason: "read_only_inspection_pending"}
	h.mu.Unlock()
	defer func() {
		cancel()
		h.mu.Lock()
		h.running = false
		h.cancel = nil
		close(done)
		h.mu.Unlock()
	}()
	checkCtx, boundedCancel := context.WithTimeout(runCtx, 20*time.Second)
	snapshot, err := h.inspect(checkCtx)
	boundedCancel()
	if err != nil {
		h.mu.Lock()
		h.inspection = LeadReadOnlyInspection{AccountID: "lead", ReadOnly: true, Reason: "read_only_inspection_failed"}
		h.mu.Unlock()
		return ErrLeadReconcileIncomplete
	}
	if snapshot.AccountID != "lead" || !snapshot.ReadOnly || snapshot.Reconciled {
		return ErrLeadReconcileIncomplete
	}
	// Recoverability evidence is audit-only; do not apply any fill or
	// change Binance orders during this observer run.
	if audit, err := LeadAuditStoreFromEnv(); err == nil {
		if err := audit.Append(runCtx, "recovery_inspection", snapshot.Reason); err == nil {
			snapshot.AuditRecorded = true
		}
	}
	h.mu.Lock()
	h.inspection = snapshot
	h.mu.Unlock()
	return h.stream.Run(runCtx)
}

// Revokes the old credential's private WS before a key rotation.
func StopReadOnlyHost() {
	leadReadOnlyRegistry.Lock()
	h := leadReadOnlyRegistry.host
	leadReadOnlyRegistry.host = nil
	leadReadOnlyRegistry.Unlock()
	if h == nil {
		return
	}
	h.mu.Lock()
	h.revoked = true
	cancel, done := h.cancel, h.done
	h.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		select {
		case <-done:
		case <-time.After(3 * time.Second):
		}
	}
}

// ReadOnlyInspectOwnership ONLY SELECTs. It cannot replay fills, change
// statuses, claim orders or release a stage-4/5 breaker.
func ReadOnlyInspectOwnership(ctx context.Context) (LeadReadOnlyInspection, error) {
	report := LeadReadOnlyInspection{AccountID: "lead", ReadOnly: true, Reconciled: false, ObservedAt: time.Now().UTC(), Reason: "read_only_not_authoritative"}
	svc, err := futuresownership.BindAccount(binance.LeadAccountID)
	if err != nil {
		return report, ErrLeadReconcileIncomplete
	}
	outstanding, err := svc.OutstandingOrders(ctx)
	if err != nil {
		return report, ErrLeadReconcileIncomplete
	}
	history, err := svc.ListPositions(ctx, "", false)
	if err != nil {
		return report, ErrLeadReconcileIncomplete
	}
	for _, o := range outstanding {
		if o.AccountID != "lead" {
			return report, ErrLeadReconcileIncomplete
		}
	}
	for _, p := range history {
		if p.AccountID != "lead" {
			return report, ErrLeadReconcileIncomplete
		}
	}
	report.OwnershipPositions = len(history)
	report.UnresolvedOrders = len(outstanding)
	if len(outstanding) > 0 {
		report.Reason = "lead_outstanding_orders_require_reconciliation"
	}
	return report, nil
}

var leadReadOnlyRegistry struct {
	sync.RWMutex
	host *LeadReadOnlyHost
}

func LeadReadOnlyStatus() LeadReadOnlyHostStatus {
	leadReadOnlyRegistry.RLock()
	host := leadReadOnlyRegistry.host
	leadReadOnlyRegistry.RUnlock()
	return host.Status()
}
func leadSetReadOnlyHost(h *LeadReadOnlyHost) {
	leadReadOnlyRegistry.Lock()
	leadReadOnlyRegistry.host = h
	leadReadOnlyRegistry.Unlock()
}
func StartConfiguredReadOnlyHost() error {
	mode := os.Getenv(ReadOnlyObserverEnv)
	if mode == "" {
		return nil
	}
	if mode != ReadOnlyObserverValue {
		return errors.New("invalid Lead observer mode; only read_only supported")
	}
	store, err := NewEncryptedFileStoreFromEnv()
	if err != nil {
		return ErrCredentialsUnavailable
	}
	credentials, err := store.Load()
	if err != nil || !credentials.Valid() {
		return ErrCredentialsUnavailable
	}
	account, err := binance.NewLeadAccountClient(credentials.APIKey, credentials.APISecret, nil)
	if err != nil {
		return ErrCredentialsUnavailable
	}
	stream, err := NewLeadUserStream(account, NewLeadAccountMirror())
	if err != nil {
		return err
	}
	host, err := NewLeadReadOnlyHost(stream, ReadOnlyInspectOwnership)
	if err != nil {
		return err
	}
	leadReadOnlyRegistry.Lock()
	if leadReadOnlyRegistry.host != nil {
		leadReadOnlyRegistry.Unlock()
		return ErrLeadSupervisorAlreadyRunning
	}
	leadReadOnlyRegistry.host = host
	leadReadOnlyRegistry.Unlock()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	go func() {
		defer stop()
		_ = host.Run(ctx)
	}()
	return nil
}

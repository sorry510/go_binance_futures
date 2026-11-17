package leadaccount

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/core/logs"
	binance "go_binance_futures/feature/api/binance"
	"go_binance_futures/service/futuresownership"
)

// ReadOnlyObserverEnv is an optional explicit observer mode. Without it,
// a configured dedicated Lead credential may still start the observer on
// application startup so managed exits work when new entries are switched off.
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
	mu     sync.RWMutex
	stream *LeadUserStream
	// These are bound only to this host's dedicated Lead account client.
	tradeSymbols *binance.LeadSymbolCache
	tradeAdapter *LeadExecutionAdapter
	inspect      func(context.Context) (LeadReadOnlyInspection, error)
	inspection   LeadReadOnlyInspection
	running      bool
	revoked      bool
	cancel       context.CancelFunc
	done         chan struct{}
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
	// The observer never submits Binance orders or claims manual positions.
	// Accepted WS orders and raw Fills use Main's local account-scoped writer.
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

// leadTradeSessionHealthyFor prevents an in-flight order from using a
// stale/rotated client after a WS disconnect or credential replacement.
func leadTradeSessionHealthyFor(account *binance.AccountClient) bool {
	if account == nil || account.ID() != binance.LeadAccountID {
		return false
	}
	leadReadOnlyRegistry.RLock()
	h := leadReadOnlyRegistry.host
	leadReadOnlyRegistry.RUnlock()
	if h == nil {
		return false
	}
	h.mu.RLock()
	ready := !h.revoked && h.running && h.stream != nil &&
		h.stream.account == account && h.stream.Status().Healthy &&
		h.stream.mirror.Status().Ready
	h.mu.RUnlock()
	return ready
}

// CurrentLeadTradeSession reuses the existing WS observer's exact Lead
// client. It constructs no additional socket or trading scheduler.
func CurrentLeadTradeSession(ctx context.Context) (*binance.AccountClient, *binance.LeadSymbolCache, *LeadExecutionAdapter, error) {
	// The opening switch can be OFF while an existing managed position
	// still needs to exit. Credential separation and WS bootstrap below
	// always apply; the durable gate is checked again before every OPEN.
	if err := ctx.Err(); err != nil {
		return nil, nil, nil, err
	}
	leadReadOnlyRegistry.RLock()
	h := leadReadOnlyRegistry.host
	leadReadOnlyRegistry.RUnlock()
	if h == nil {
		return nil, nil, nil, ErrLeadMirrorUntrusted
	}
	account, symbols, adapter, err := func() (*binance.AccountClient, *binance.LeadSymbolCache, *LeadExecutionAdapter, error) {
		h.mu.Lock()
		defer h.mu.Unlock()
		if h.revoked || !h.running || h.stream == nil ||
			!h.stream.Status().Healthy || !h.stream.mirror.Status().Ready {
			return nil, nil, nil, ErrLeadMirrorUntrusted
		}
		account, ok := h.stream.account.(*binance.AccountClient)
		if !ok || account.ID() != binance.LeadAccountID {
			return nil, nil, nil, ErrLeadMirrorUntrusted
		}
		if h.tradeSymbols == nil {
			cache, err := binance.NewLeadSymbolCache(account, nil)
			if err != nil {
				return nil, nil, nil, err
			}
			h.tradeSymbols = cache
		}
		if h.tradeAdapter == nil {
			value, err := NewLeadExecutionAdapter(account, h.tradeSymbols)
			if err != nil {
				return nil, nil, nil, err
			}
			h.tradeAdapter = value
		}
		return account, h.tradeSymbols, h.tradeAdapter, nil
	}()
	if err != nil {
		return nil, nil, nil, err
	}
	// SAPI requests must not hold the observer's lifecycle lock.
	if err := symbols.Refresh(ctx); err != nil {
		return nil, nil, nil, err
	}
	h.mu.RLock()
	ready := !h.revoked && h.running && h.stream != nil &&
		h.stream.account == account && h.stream.Status().Healthy
	h.mu.RUnlock()
	if !ready {
		return nil, nil, nil, ErrLeadMirrorUntrusted
	}
	return account, symbols, adapter, nil
}

func leadSetReadOnlyHost(h *LeadReadOnlyHost) {
	leadReadOnlyRegistry.Lock()
	leadReadOnlyRegistry.host = h
	leadReadOnlyRegistry.Unlock()
}

// StartConfiguredReadOnlyHost is the existing single Lead WS startup hook.
// A persisted user-enabled Lead account can use it without the old observer
// env toggle. It does not run a standalone verification or enable Binance
// orders on its own.
func StartConfiguredReadOnlyHost() error {
	mode := os.Getenv(ReadOnlyObserverEnv)
	if mode != "" && mode != ReadOnlyObserverValue {
		return errors.New("invalid Lead observer mode; only read_only supported")
	}
	if mode == "" {
		store, err := LeadConfigStoreFromDB()
		if err != nil {
			return nil
		}
		settings, err := store.Load()
		if err != nil || settings.AccountID != "lead" {
			return nil
		}
		// A disabled new-entry switch must not stop existing managed exits
		// after a restart. Start only when the user has configured a valid
		// private Lead credential store; never fall back to Main.
		if store, e := NewEncryptedFileStoreFromEnv(); e != nil {
			return nil
		} else if credentials, e := store.Load(); e != nil || !credentials.Valid() {
			return nil
		}
	}
	return EnsureLeadHostRunning()
}

// EnsureLeadHostRunning is called only when the user enables the persistent
// Lead switch or by existing startup. It reads ONLY encrypted Lead credentials.
func EnsureLeadHostRunning() error {
	leadReadOnlyRegistry.RLock()
	host := leadReadOnlyRegistry.host
	leadReadOnlyRegistry.RUnlock()
	if host != nil {
		if host.Status().Running {
			return nil
		}
		StopReadOnlyHost() // discard a stale/failed observer before restart
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
	// Same local WS order/Fill persistence as Main. The observer never
	// places Binance orders and never auto-claims manual positions.
	stream.onEvent = func(event *futures.WsUserDataEvent) {
		if err := account.PersistLeadUserDataEvent(event); err != nil {
			logs.Error("Lead user WS event local persistence failed: ", err)
		}
	}
	host, err = NewLeadReadOnlyHost(stream, ReadOnlyInspectOwnership)
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

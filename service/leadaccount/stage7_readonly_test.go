package leadaccount

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"go_binance_futures/appversion"
	binance "go_binance_futures/feature/api/binance"
	"go_binance_futures/service/futuresownership"
)

func TestStage71SchemaVersionStaticGate(t *testing.T) {
	if appversion.DatabaseSchemaVersion != 19 {
		t.Fatalf("Stage7 requires v19 account-scoped schema, code requires %d", appversion.DatabaseSchemaVersion)
	}
	// No real MySQL I/O or sync db: tests must not silently pass the live Gate.
}
func TestStage72DisabledObserverDoesNotReadCredentialsOrNetwork(t *testing.T) {
	t.Setenv(ReadOnlyObserverEnv, "")
	t.Setenv(MasterKeyEnv, "")
	t.Setenv(StorePathEnv, "")
	if err := StartConfiguredReadOnlyHost(); err != nil {
		t.Fatal(err)
	}
	if value := LeadReadOnlyStatus(); value.Running || value.LiveTradeAuthorized {
		t.Fatalf("disabled observer unexpectedly connected: %+v", value)
	}
	t.Setenv(ReadOnlyObserverEnv, "trade")
	if err := StartConfiguredReadOnlyHost(); err == nil {
		t.Fatal("unreviewed mode unexpectedly connected")
	}
	t.Setenv(ReadOnlyObserverEnv, ReadOnlyObserverValue)
	if err := StartConfiguredReadOnlyHost(); !errors.Is(err, ErrCredentialsUnavailable) {
		t.Fatalf("missing credentials should block opt-in, got %v", err)
	}
}
func TestStage72ReadOnlyHostLifecycleNoExchangeMutations(t *testing.T) {
	fake := &stage51Account{}
	mirror := NewLeadAccountMirror()
	stream := &LeadUserStream{account: fake, mirror: mirror, source: stage51SeedReader(), backoff: time.Millisecond * 3, keepalive: time.Millisecond * 25}
	var dials atomic.Int64
	var inspectCalls atomic.Int64
	onDial := make(chan struct{}, 1)
	stream.dial = func(string, futures.WsUserDataHandler, futures.ErrHandler) (chan struct{}, chan struct{}, error) {
		dials.Add(1)
		select {
		case onDial <- struct{}{}:
		default:
		}
		return make(chan struct{}), make(chan struct{}), nil
	}
	host, err := NewLeadReadOnlyHost(stream, func(context.Context) (LeadReadOnlyInspection, error) {
		inspectCalls.Add(1)
		return LeadReadOnlyInspection{AccountID: "lead", ReadOnly: true, Reason: "read_only_not_authoritative"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- host.Run(ctx) }()
	select {
	case <-onDial:
	case <-time.After(time.Second):
		cancel()
		t.Fatal("read-only mock WS not started")
	}
	deadline := time.Now().Add(time.Second)
	for !host.Status().Stream.Healthy && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	observed := host.Status()
	if !observed.Running || !observed.Stream.Healthy || observed.Inspection.Reconciled || observed.LiveTradeAuthorized {
		cancel()
		t.Fatalf("read-only observation claimed permission %+v", observed)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("read-only host did not terminate")
	}
	if host.Status().Running || dials.Load() != 1 || inspectCalls.Load() != 1 {
		t.Fatalf("read-only host leaked workers: %+v", host.Status())
	}
	fake.mu.Lock()
	opened, closed := fake.opened, fake.closed
	fake.mu.Unlock()
	if opened != 1 || closed != 1 {
		t.Fatalf("private listen key leaked: opened=%d closed=%d", opened, closed)
	}
}
func TestStage72CredentialRotationBeforeRunCannotStartOldWS(t *testing.T) {
	mirror := NewLeadAccountMirror()
	stream := &LeadUserStream{account: &stage51Account{}, mirror: mirror, source: stage51SeedReader()}
	var called atomic.Int64
	stream.dial = func(string, futures.WsUserDataHandler, futures.ErrHandler) (chan struct{}, chan struct{}, error) {
		called.Add(1)
		return make(chan struct{}), make(chan struct{}), nil
	}
	host, err := NewLeadReadOnlyHost(stream, func(context.Context) (LeadReadOnlyInspection, error) {
		return LeadReadOnlyInspection{AccountID: "lead", ReadOnly: true, Reason: "read_only_not_authoritative"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	leadSetReadOnlyHost(host)
	StopReadOnlyHost()
	if err := host.Run(context.Background()); !errors.Is(err, context.Canceled) {
		t.Fatalf("revoked host restarted: %v", err)
	}
	if called.Load() != 0 || LeadReadOnlyStatus().Running {
		t.Fatal("old credential generation was allowed to open WS")
	}
}
func TestStage72OwnershipReadOnlyInspectionDoesNotReconcile(t *testing.T) {
	prepareLeadExecutionDB(t)
	owner, err := futuresownership.BindAccount(binance.LeadAccountID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = owner.ClaimOrder(context.Background(), futuresownership.ClaimOrderInput{
		Owner: futuresownership.OwnerAutoStrategy, Symbol: "BTCUSDT", PositionSide: "LONG", Intent: futuresownership.IntentOpen,
		ClientOrderID: "stage72_readonly_pending", RequestedQty: 1, OrderType: "MARKET",
	})
	if err != nil {
		t.Fatal(err)
	}
	inspected, err := ReadOnlyInspectOwnership(context.Background())
	if err != nil || inspected.AccountID != "lead" || inspected.UnresolvedOrders != 1 || inspected.Reconciled ||
		inspected.Reason != "lead_outstanding_orders_require_reconciliation" {
		t.Fatalf("read-only inspector incorrect %+v %v", inspected, err)
	}
	persisted, err := owner.GetOrder(context.Background(), "stage72_readonly_pending")
	if err != nil || persisted.Status != futuresownership.OrderPending {
		t.Fatalf("read-only inspector mutated ledger %+v %v", persisted, err)
	}
}
func TestStage73DurablePauseFileBlocksAcrossInstancesAndAudit(t *testing.T) {
	directory := t.TempDir()
	if err := os.Chmod(directory, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv(ConfigPathEnv, filepath.Join(directory, "risk.json"))
	t.Setenv(AuditPathEnv, filepath.Join(directory, "events.jsonl"))
	store, err := LeadConfigStoreFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(DefaultLeadSettings()); err != nil {
		t.Fatal(err)
	}
	if err := LeadDurableOpenGate(); !errors.Is(err, ErrLeadLiveGateLocked) {
		t.Fatalf("disabled durable config allowed new entries: %v", err)
	}
	another, err := LeadConfigStoreFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	state, err := another.Load()
	if err != nil || state.Enabled || state.AllowNewOpens {
		t.Fatalf("new instance reopened Lead: %+v %v", state, err)
	}
	log, err := LeadAuditStoreFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if err := log.Append(context.Background(), "recovery_inspection", "read_only_not_authoritative"); err != nil {
		t.Fatalf("read-only recovery audit unavailable: %v", err)
	}
	rows, err := log.List(context.Background(), 3)
	if err != nil || len(rows) != 1 || rows[0].Action != "recovery_inspection" {
		t.Fatalf("recovery audit lost %+v %v", rows, err)
	}
}
func TestStage73DynamicPriceNeedsAuthoritativeReference(t *testing.T) {
	request, symbol, private, brackets := stage43RuleCase()
	symbol.Filters = append(symbol.Filters, map[string]any{"filterType": "PERCENT_PRICE", "multiplierUp": "1.1", "multiplierDown": "0.9"})
	if _, err := ValidateLeadOrderRules(request, symbol, private, brackets, futures.MarginTypeCrossed); err == nil {
		t.Fatal("Lead opened with unverified dynamic PERCENT_PRICE")
	}
	closeReq := futuresownership.OrderRequest{Symbol: "BTCUSDT", OrderType: "LIMIT", Side: "SELL", PositionSide: "LONG", Price: 100, Quantity: 1}
	if err := ValidateLeadCloseRules(closeReq, symbol, 100, 1, 1); err == nil {
		t.Fatal("Lead submitted limit close without authoritative dynamic price bounds")
	}
}
func TestStage74ConcurrentStatusReadsNeverAuthorizeStage7(t *testing.T) {
	a, _ := stage43Adapter(t, false)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = a.RuntimeStatus()
				if a.RuntimeStatus().CanAttemptOpen {
					t.Error("concurrent status read unlocked live order")
					return
				}
			}
		}()
	}
	wg.Wait()
}

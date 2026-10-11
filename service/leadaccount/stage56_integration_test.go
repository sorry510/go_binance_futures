package leadaccount

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	binance "go_binance_futures/feature/api/binance"
)

func TestStage56SupervisorRecoveryStartsBeforeWSAndStopsAllWorkers(t *testing.T) {
	account, broker := stage43Adapter(t, false)
	mirror := NewLeadAccountMirror()
	stream := &LeadUserStream{account: &stage51Account{}, source: stage51SeedReader(), mirror: mirror, backoff: time.Millisecond, keepalive: time.Millisecond * 25}
	wsConnected := make(chan struct{}, 1)
	stream.dial = func(_ string, _ futures.WsUserDataHandler, _ futures.ErrHandler) (chan struct{}, chan struct{}, error) {
		select {
		case wsConnected <- struct{}{}:
		default:
		}
		return make(chan struct{}), make(chan struct{}), nil
	}
	alertQueue := NewLeadAlertQueue(32)
	account.guard = NewLeadRuntimeGuard(alertQueue.Publish)
	stream.onFault = account.ReportLeadFault
	var recoveryCalled atomic.Bool
	var dispatched atomic.Int64
	supervisor := &LeadSupervisor{
		Adapter: account, Mirror: mirror, Stream: stream, Alerts: alertQueue,
		Recovery: func(context.Context) (LeadRecoveryReport, error) {
			recoveryCalled.Store(true)
			return LeadRecoveryReport{AccountID: binance.LeadAccountID, Blocked: true}, ErrLeadReconcileIncomplete
		},
		Send: func(context.Context, LeadFaultEvent) error { dispatched.Add(1); return nil },
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- supervisor.Run(ctx) }()
	select {
	case <-wsConnected:
	case <-time.After(time.Second):
		cancel()
		t.Fatal("WS not started")
	}
	if !recoveryCalled.Load() {
		t.Fatal("WS started without startup DB reconciliation attempt")
	}
	// The WS dial callback can run before the snapshot's final lock commit.
	deadline := time.Now().Add(time.Second)
	for !supervisor.Stream.Status().Healthy && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	got, ws, _ := supervisor.Status()
	if got.State != LeadReconcileRequired || !ws.Running || !ws.Healthy {
		t.Fatalf("incomplete recovery did not block Lead while observing WS: %+v %+v", got, ws)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second * 2):
		t.Fatal("supervisor did not terminate workers")
	}
	if supervisor.Stream.Status().Running {
		t.Fatal("socket lifecycle leaked")
	}
	if broker.submits != 0 {
		t.Fatal("run unexpectedly issued a Binance order")
	}
	if dispatched.Load() == 0 {
		t.Fatal("recovery fault did not reach alert queue")
	}
}
func TestStage56PreSnapshotWSMessageForcesResync(t *testing.T) {
	fake := &stage51Account{}
	mirror := NewLeadAccountMirror()
	var attempts atomic.Int64
	stream := &LeadUserStream{account: fake, source: stage51SeedReader(), mirror: mirror, backoff: time.Millisecond * 2, keepalive: time.Hour}
	stream.dial = func(_ string, handler futures.WsUserDataHandler, _ futures.ErrHandler) (chan struct{}, chan struct{}, error) {
		attempts.Add(1)
		// Message received in connection-to-snapshot gap: must invalidate this
		// generation, even if REST later returned successfully.
		handler(stage51OrderEvent(999, time.Now().UnixMilli(), "1", "NEW"))
		return make(chan struct{}), make(chan struct{}), nil
	}
	var faults atomic.Int64
	stream.onFault = func(code LeadFaultCode) {
		if code == FaultSnapshotInvalid {
			faults.Add(1)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 215*time.Millisecond)
	defer cancel()
	_ = stream.Run(ctx)
	if attempts.Load() < 2 || faults.Load() < 2 {
		t.Fatalf("WS bootstrap gap silently accepted; attempts=%d faults=%d", attempts.Load(), faults.Load())
	}
	if mirror.Status().Ready || stream.Status().Running {
		t.Fatal("bootstrap failure left apparently healthy mirror")
	}
}
func TestStage56ConcurrentMirrorRiskStatusAndAPIMetrics(t *testing.T) {
	a, _ := stage43Adapter(t, false)
	mirror := NewLeadAccountMirror()
	generation := mirror.BeginGeneration()
	if err := mirror.Seed(context.Background(), generation, stage51SeedReader()); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UnixMilli()
	var wg sync.WaitGroup
	for worker := 0; worker < 12; worker++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for n := 0; n < 250; n++ {
				switch i % 4 {
				case 0:
					_ = a.RuntimeStatus()
				case 1:
					_ = mirror.Status()
				case 2:
					_ = LeadUsageSnapshot(nil, mirror, time.Now())
				case 3:
					// Different order IDs model 2s trading cycles, private events and UI
					// refresh collapsed into accelerated concurrent Mock iterations.
					event := stage51OrderEvent(int64(5000+i*300+n), now+int64(n), "1", "NEW")
					_ = mirror.ApplyEvent(generation, event)
				}
			}
		}(worker)
	}
	wg.Wait()
	if !mirror.Status().Ready || mirror.Status().Metrics.WSUpdates == 0 {
		t.Fatalf("mirror lost generation during pressure: %+v", mirror.Status())
	}
	if a.RuntimeStatus().CanAttemptOpen {
		t.Fatal("UI reads unexpectedly authorized execution")
	}
}
func TestStage56NoAutomaticProductionNetworkOrLiveOrder(t *testing.T) {
	main, err := binance.NewAccountClient(binance.MainAccountID, futures.NewClient("main", "secret"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewLeadSupervisor(main, nil); err == nil {
		t.Fatal("Main accepted into Lead supervisor")
	}
	lead, err := binance.NewAccountClient(binance.LeadAccountID, futures.NewClient("test-lead", "secret"))
	if err != nil {
		t.Fatal(err)
	}
	supervisor, err := NewLeadSupervisor(lead, riskSymbols(t))
	if err != nil {
		t.Fatal(err)
	}
	status, ws, metrics := supervisor.Status()
	if status.CanAttemptOpen || ws.Running || ws.Healthy || metrics.AccountID != binance.LeadAccountID {
		t.Fatalf("constructor started network or live trading: %+v %+v %+v", status, ws, metrics)
	}
	if supervisor.Adapter.ownership.Ownership.AccountID != "lead" {
		t.Fatal("ownership scope incorrect")
	}
	// No authorized resume path, even after API/data restoration.
	supervisor.Adapter.ReportLeadFault(FaultUnknownSubmission)
	if supervisor.Adapter.RuntimeStatus().State != LeadReconcileRequired {
		t.Fatal("unknown result not blocked")
	}
}

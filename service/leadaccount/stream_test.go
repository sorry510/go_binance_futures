package leadaccount

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	binance "go_binance_futures/feature/api/binance"
)

func stage51SeedReader() *fakeLeadSnapshotReader {
	r := sampleLeadEvidenceReader()
	for i := range r.orders {
		r.orders[i].OrderID = int64(i + 1)
	}
	return r
}
func TestStage51MirrorRequiresCompleteLeadSnapshotAndGeneration(t *testing.T) {
	m := NewLeadAccountMirror()
	gen := m.BeginGeneration()
	bad := stage51SeedReader()
	bad.account.CanTrade = false
	if err := m.Seed(context.Background(), gen, bad); !errors.Is(err, ErrLeadMirrorUntrusted) {
		t.Fatalf("accepted incomplete account: %v", err)
	}
	if m.Status().Ready {
		t.Fatal("partial REST seeded mirror")
	}
	good := stage51SeedReader()
	if err := m.Seed(context.Background(), gen, good); err != nil {
		t.Fatal(err)
	}
	if !m.Status().Ready || m.Status().PositionCount != 1 || m.Status().OrderCount != 2 {
		t.Fatalf("mirror seed wrong: %+v", m.Status())
	}
	next := m.BeginGeneration()
	if m.Status().Ready {
		t.Fatal("generation rollover left cached mirror usable")
	}
	if err := m.Seed(context.Background(), gen, good); !errors.Is(err, ErrLeadMirrorUntrusted) {
		t.Fatalf("old snapshot promoted: %v", err)
	}
	if err := m.Seed(context.Background(), next, good); err != nil {
		t.Fatal(err)
	}
	m.Invalidate(gen)
	if !m.Status().Ready {
		t.Fatal("old disconnect revoked newer generation")
	}
	m.Invalidate(next)
	if m.Status().Ready {
		t.Fatal("active generation not revoked")
	}
}
func stage51OrderEvent(id int64, ms int64, cumulative, status string) *futures.WsUserDataEvent {
	return &futures.WsUserDataEvent{Event: futures.UserDataEventTypeOrderTradeUpdate, Time: ms,
		WsUserDataOrderTradeUpdate: futures.WsUserDataOrderTradeUpdate{OrderTradeUpdate: futures.WsOrderTradeUpdate{
			Symbol: "ETHUSDT", ClientOrderID: "stage51", ID: id, PositionSide: futures.PositionSideTypeShort,
			Side: futures.SideTypeSell, Type: futures.OrderTypeLimit, OriginalQty: "3", OriginalPrice: "20",
			AccumulatedFilledQty: cumulative, Status: futures.OrderStatusType(status),
		}},
	}
}
func TestStage52MirrorMonotonicOrderFillsAndTerminalWatermark(t *testing.T) {
	m := NewLeadAccountMirror()
	gen := m.BeginGeneration()
	source := stage51SeedReader()
	if err := m.Seed(context.Background(), gen, source); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UnixMilli()
	for _, tc := range []struct {
		qty, status string
		at          int64
		allow       bool
	}{
		{"1.5", "PARTIALLY_FILLED", now, true},
		{"1.0", "PARTIALLY_FILLED", now + 1, false}, // regression
		{"1.5", "PARTIALLY_FILLED", now - 1, false}, // timestamp rewind
		{"3", "FILLED", now + 2, true},
		{"2", "PARTIALLY_FILLED", now + 3, false}, // terminal cannot reopen
	} {
		if got := m.ApplyEvent(gen, stage51OrderEvent(1, tc.at, tc.qty, tc.status)); got != tc.allow {
			t.Fatalf("qty=%s status=%s accepted=%t want=%t", tc.qty, tc.status, got, tc.allow)
		}
	}
	if m.Status().OrderCount != 1 || m.Status().Metrics.RejectedEvents != 3 {
		t.Fatalf("terminal order or dedup invalid %+v", m.Status())
	}
	if m.ApplyEvent(gen-1, stage51OrderEvent(5, now+5, "1", "NEW")) {
		t.Fatal("old WS updated current mirror")
	}
	if m.Status().Metrics.RejectedEvents != 4 {
		t.Fatal("stale-generation event not counted")
	}
}
func TestStage52InvalidBalanceUpdateDoesNotPartiallyChangeMirror(t *testing.T) {
	m := NewLeadAccountMirror()
	gen := m.BeginGeneration()
	if err := m.Seed(context.Background(), gen, stage51SeedReader()); err != nil {
		t.Fatal(err)
	}
	event := &futures.WsUserDataEvent{Event: futures.UserDataEventTypeAccountUpdate, Time: time.Now().UnixMilli(),
		WsUserDataAccountUpdate: futures.WsUserDataAccountUpdate{AccountUpdate: futures.WsAccountUpdate{
			Positions: []futures.WsPosition{{Symbol: "BTCUSDT", Side: futures.PositionSideTypeLong, Amount: "2"}},
			Balances:  []futures.WsBalance{{Asset: "USDT", Balance: "NaN"}},
		}},
	}
	if m.ApplyEvent(gen, event) {
		t.Fatal("invalid USDT balance should reject entire event")
	}
	m.mu.RLock()
	position := m.positions[riskSlot("BTCUSDT", "LONG")]
	m.mu.RUnlock()
	if position.Quantity != 1 {
		t.Fatalf("partial mutation before rejection: %+v", position)
	}
}

type stage51Account struct {
	mu                        sync.Mutex
	opened, closed, keepalive int
}

func (a *stage51Account) GetListenKey(context.Context) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.opened++
	return "mock-lead-listen-key", nil
}
func (a *stage51Account) UpdateListenKey(context.Context, string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.keepalive++
	return nil
}
func (a *stage51Account) CloseLeadListenKey(context.Context, string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.closed++
	return nil
}
func TestStage51StreamLifecycleReconnectKeepaliveAndStop(t *testing.T) {
	account := &stage51Account{}
	mirror := NewLeadAccountMirror()
	source := stage51SeedReader()
	stream := &LeadUserStream{account: account, source: source, mirror: mirror, backoff: time.Millisecond * 2, keepalive: time.Millisecond * 8}
	events := make(chan futures.WsUserDataHandler, 8)
	stream.dial = func(_ string, handler futures.WsUserDataHandler, _ futures.ErrHandler) (chan struct{}, chan struct{}, error) {
		events <- handler
		return make(chan struct{}), make(chan struct{}), nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- stream.Run(ctx) }()
	var first, second futures.WsUserDataHandler
	select {
	case first = <-events:
	case <-time.After(2 * time.Second):
		t.Fatal("first WS did not connect")
	}
	first(&futures.WsUserDataEvent{Event: "listenKeyExpired", Time: time.Now().UnixMilli()})
	select {
	case second = <-events:
	case <-time.After(2 * time.Second):
		t.Fatal("reconnect after expiry failed")
	}
	if first == nil || second == nil {
		t.Fatal("missing handlers")
	}
	first(stage51OrderEvent(8, time.Now().UnixMilli(), "1", "PARTIALLY_FILLED"))
	if mirror.Status().Metrics.WSUpdates != 0 {
		t.Fatal("old WS generation changed current mirror")
	}
	time.Sleep(time.Millisecond * 26)
	if !stream.Status().Healthy {
		t.Fatalf("new stream unhealthy: %+v", stream.Status())
	}
	stream.Stop()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		cancel()
		t.Fatal("stream did not stop")
	}
	cancel()
	account.mu.Lock()
	opened, closed, keepalive := account.opened, account.closed, account.keepalive
	account.mu.Unlock()
	if opened < 2 || closed != opened || keepalive < 1 {
		t.Fatalf("listenKey lifecycle opened=%d closed=%d keepalive=%d", opened, closed, keepalive)
	}
	if stream.Status().Running {
		t.Fatal("stream leaked running flag")
	}
}
func TestStage51ClientRejectsMainAccount(t *testing.T) {
	main, err := binance.NewAccountClient(binance.MainAccountID, futures.NewClient("main-fake", "secret"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewLeadUserStream(main, NewLeadAccountMirror()); err == nil {
		t.Fatal("Main allowed to start Lead WS")
	}
	if err := main.CloseLeadListenKey(context.Background(), "fake-key"); err == nil {
		t.Fatal("Main allowed to close Lead listenKey")
	}
}

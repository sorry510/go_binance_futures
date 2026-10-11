package leadaccount

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	binance "go_binance_futures/feature/api/binance"
	"go_binance_futures/service/futuresownership"
)

type stage5ReviewReader struct {
	base      *fakeLeadRulesReader
	calls     atomic.Int64
	positions atomic.Int64
	fail      atomic.Bool
}

func (r *stage5ReviewReader) GetLeadExchangeInfoContext(ctx context.Context) (*futures.ExchangeInfo, error) {
	r.calls.Add(1)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if r.fail.Load() {
		return nil, errors.New("exchange rules unavailable")
	}
	return r.base.exchange, nil
}
func (r *stage5ReviewReader) GetPositionFreshContext(ctx context.Context, p binance.PositionParams) ([]*futures.PositionRisk, error) {
	r.positions.Add(1)
	return r.base.GetPositionFreshContext(ctx, p)
}
func (r *stage5ReviewReader) GetLeadLeverageBracketContext(ctx context.Context, symbol string) ([]*futures.LeverageBracket, error) {
	return r.base.GetLeadLeverageBracketContext(ctx, symbol)
}
func TestStage5ReviewLeadExchangeInfoTTLSharedOpenAndClose(t *testing.T) {
	request, info, pos, tier := stage43RuleCase()
	reader := &stage5ReviewReader{base: &fakeLeadRulesReader{exchange: &futures.ExchangeInfo{Symbols: []futures.Symbol{info}}, pos: pos, tier: tier}}
	source := &LeadRuleSource{AccountID: binance.LeadAccountID, Reader: reader}
	now := time.Now().UTC()
	source.cacheNow = func() time.Time { return now }
	for i := 0; i < 20; i++ {
		got, err := source.Verify(context.Background(), request, futures.MarginTypeCrossed)
		if err != nil || got.MinNotionalUSDT != 5 {
			t.Fatalf("open check #%d %+v %v", i, got, err)
		}
	}
	adapter := &LeadExecutionAdapter{rules: source}
	closeRequest := futuresownership.OrderRequest{Symbol: "BTCUSDT", OrderType: "MARKET", Quantity: 1.25}
	if err := adapter.verifyCloseRules(context.Background(), closeRequest, 100.5, 1.25, 1.25); err != nil {
		t.Fatalf("close did not reuse exchangeInfo: %v", err)
	}
	if reader.calls.Load() != 1 || reader.positions.Load() != 20 {
		t.Fatalf("cache wrongly reused private position reads: exchange=%d position=%d", reader.calls.Load(), reader.positions.Load())
	}
	// A second account rule source NEVER shares this instance's cached value.
	another := &LeadRuleSource{AccountID: binance.LeadAccountID, Reader: reader}
	if _, err := another.exchangeInfo(context.Background()); err != nil || reader.calls.Load() != 2 {
		t.Fatalf("unrelated Lead instance shared exchange info: calls=%d err=%v", reader.calls.Load(), err)
	}
	now = now.Add(leadExchangeInfoTTL + time.Second)
	reader.fail.Store(true)
	if _, err := source.Verify(context.Background(), request, futures.MarginTypeCrossed); err == nil {
		t.Fatal("expired exchange rules failed open rather than fail-closed")
	}
	if reader.calls.Load() != 3 {
		t.Fatalf("cache returned stale result when refresh failed: %d", reader.calls.Load())
	}
	reader.fail.Store(false)
	if _, err := source.Verify(context.Background(), request, futures.MarginTypeCrossed); err != nil {
		t.Fatalf("fresh revalidation failed: %v", err)
	}
	if reader.calls.Load() != 4 {
		t.Fatalf("refresh did not reload valid exchange rules: %d", reader.calls.Load())
	}
	source.invalidateExchangeInfo()
	if _, err := source.exchangeInfo(context.Background()); err != nil || reader.calls.Load() != 5 {
		t.Fatalf("explicit invalidation did not reload: %d %v", reader.calls.Load(), err)
	}
}
func TestStage5ReviewLeadExchangeInfoSingleflightAndCancellation(t *testing.T) {
	request, info, pos, tier := stage43RuleCase()
	reader := &stage5ReviewReader{base: &fakeLeadRulesReader{exchange: &futures.ExchangeInfo{Symbols: []futures.Symbol{info}}, pos: pos, tier: tier}}
	source := &LeadRuleSource{AccountID: binance.LeadAccountID, Reader: reader}
	var wg sync.WaitGroup
	errs := make(chan error, 48)
	for i := 0; i < 48; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := source.Verify(context.Background(), request, futures.MarginTypeCrossed)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if reader.calls.Load() != 1 {
		t.Fatalf("concurrent cache miss triggered %d exchange requests", reader.calls.Load())
	}
	if reader.positions.Load() != 48 {
		t.Fatalf("private positions were incorrectly cached: %d", reader.positions.Load())
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := source.exchangeInfo(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled context silently used cache: %v", err)
	}
}
func TestStage5ReviewQueueLossVisibleInUsageAndSupervisor(t *testing.T) {
	queue := NewLeadAlertQueue(1)
	if err := queue.Publish(LeadFaultEvent{AccountID: binance.LeadAccountID, Code: FaultWSUnavailable, Severity: SeverityWarning}); err != nil {
		t.Fatal(err)
	}
	if err := queue.Publish(LeadFaultEvent{AccountID: binance.LeadAccountID, Code: FaultAPI418, Severity: SeverityCritical}); !errors.Is(err, ErrLeadAlertBackpressure) {
		t.Fatalf("critical alert did not apply bounded backpressure: %v", err)
	}
	if err := queue.Publish(LeadFaultEvent{AccountID: binance.LeadAccountID, Code: FaultAPI429, Severity: SeverityWarning}); !errors.Is(err, ErrLeadAlertBackpressure) {
		t.Fatalf("warning alert did not apply bounded backpressure: %v", err)
	}
	state := queue.Snapshot()
	if state.Depth != 1 || state.Capacity != 1 || state.Dropped != 2 || state.DroppedCritical != 1 {
		t.Fatalf("lost notification capacity not observable: %+v", state)
	}
	usage := LeadUsageSnapshotWithAlerts(nil, nil, queue, time.Now().UTC())
	if usage.AccountID != binance.LeadAccountID || usage.AlertQueueDepth != 1 || usage.AlertQueueCapacity != 1 ||
		usage.AlertsDropped != 2 || usage.CriticalAlertsDropped != 1 {
		t.Fatalf("Lead API metrics omitted queue backpressure %+v", usage)
	}
	a, _ := stage43Adapter(t, false)
	sup := &LeadSupervisor{Adapter: a, Stream: &LeadUserStream{}, Alerts: queue, Mirror: NewLeadAccountMirror()}
	_, _, supervisorMetrics := sup.Status()
	if supervisorMetrics.AlertsDropped != 2 || supervisorMetrics.CriticalAlertsDropped != 1 {
		t.Fatalf("supervisor status does not expose queue loss %+v", supervisorMetrics)
	}
	if a.RuntimeStatus().CanAttemptOpen {
		t.Fatal("queue telemetry authorized production trading")
	}
}
func TestStage5ReviewSupervisorPublicSurfaceLocked(t *testing.T) {
	typ := reflect.TypeOf((*LeadSupervisor)(nil))
	names := make([]string, 0, typ.NumMethod())
	for i := 0; i < typ.NumMethod(); i++ {
		names = append(names, typ.Method(i).Name)
	}
	if !reflect.DeepEqual(names, []string{"Run", "Status"}) {
		t.Fatalf("unexpected Lead supervisor exported methods, require review: %v", names)
	}
	client, err := binance.NewAccountClient(binance.LeadAccountID, futures.NewClient("fake", "fake"))
	if err != nil {
		t.Fatal(err)
	}
	sup, err := NewLeadSupervisor(client, riskSymbols(t))
	if err != nil {
		t.Fatal(err)
	}
	state, ws, _ := sup.Status()
	if state.CanAttemptOpen || state.Enabled || ws.Running || ws.Healthy {
		t.Fatalf("constructor enabled Lead or started private WS: %+v %+v", state, ws)
	}
}

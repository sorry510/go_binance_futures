package leadaccount

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	binance "go_binance_futures/feature/api/binance"
	"go_binance_futures/service/binanceapiusage"
)

func TestStage54AccountSourceUsesSharedGlobalBudgetCollector(t *testing.T) {
	collector := binanceapiusage.Default()
	collector.ResetForTest()
	defer collector.ResetForTest()
	now := time.Now().UTC()
	for _, tc := range []struct {
		src    string
		weight int64
		status int
	}{
		{"lead_trading", 8, 429}, {"lead_trading_sapi", 2, 418}, {"start_trade", 17, 200}, {"lead_trading", 4, 200},
	} {
		collector.Record(binanceapiusage.RequestEvent{
			At: now.UnixMilli(), Product: "futures", Environment: "mainnet",
			Source: tc.src, Method: "GET", Path: "/fapi/v1/positionRisk",
			RequestType: "read", EstimatedWeight: tc.weight, StatusCode: tc.status,
			Is429: tc.status == 429, Is418: tc.status == 418,
		}, nil, "")
	}
	metrics := LeadUsageSnapshot(collector, nil, now)
	if metrics.AccountID != binance.LeadAccountID || metrics.Requests != 3 || metrics.EstimatedWeight != 14 ||
		metrics.Count429 != 1 || metrics.Count418 != 1 {
		t.Fatalf("Lead request attribution included Main or incorrect weight %+v", metrics)
	}
	all := collector.Snapshot(now)
	if all.Window5m.RequestCount != 4 || metrics.Requests >= all.Window5m.RequestCount {
		t.Fatalf("global IP budget was replaced by per-account collector: %+v", all.Window5m)
	}
}
func TestStage55RateLimitThreeAlertsAtTwoMinuteSpacing(t *testing.T) {
	fixed := time.Date(2026, 10, 10, 8, 0, 0, 0, time.UTC)
	total := 0
	guard := NewLeadRuntimeGuard(func(e LeadFaultEvent) error {
		if e.Code != FaultAPI429 || e.AccountID != "lead" {
			t.Fatalf("unexpected event %+v", e)
		}
		total++
		return nil
	})
	guard.now = func() time.Time { return fixed }
	for i := 0; i < 9; i++ {
		guard.RecordFault(binance.LeadAccountID, FaultAPI429)
		fixed = fixed.Add(2*time.Minute + time.Second)
	}
	if total != 3 {
		t.Fatalf("429 raised %d alerts, expected max 3", total)
	}
	if guard.Status().State != LeadAccountUnavailable {
		t.Fatal("silenced alert unlocked 429 fault")
	}
}
func TestStage55BoundedQueueNeverBlocksRiskAndWorkerStops(t *testing.T) {
	q := NewLeadAlertQueue(1)
	guard := NewLeadRuntimeGuard(q.Publish)
	if !guard.RecordFault(binance.LeadAccountID, FaultWSUnavailable) {
		t.Fatal("fault not recorded")
	}
	if !guard.RecordFault(binance.LeadAccountID, FaultAPI418) {
		t.Fatal("fault not recorded")
	}
	if q.Dropped() != 1 {
		t.Fatalf("queue expected 1 dropped alert got %d", q.Dropped())
	}
	if guard.Status().State != LeadAccountUnavailable {
		t.Fatal("alert queue saturation removed fault")
	}
	var sent atomic.Int64
	ctx, cancel := context.WithCancel(context.Background())
	finished := make(chan error, 1)
	go func() {
		finished <- q.Run(ctx, func(context.Context, LeadFaultEvent) error {
			sent.Add(1)
			return errors.New("simulated notifier offline")
		})
	}()
	deadline := time.After(time.Second)
	for sent.Load() == 0 {
		select {
		case <-deadline:
			cancel()
			t.Fatal("alert worker did not dispatch")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	cancel()
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("alert worker goroutine leaked")
	}
	if guard.Status().State != LeadAccountUnavailable {
		t.Fatal("notification failure removed safety fault")
	}
}
func TestStage55QueueConcurrencyAndSanitization(t *testing.T) {
	q := NewLeadAlertQueue(8)
	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = q.Publish(LeadFaultEvent{AccountID: binance.LeadAccountID, Code: FaultAPI429, Action: "raised"})
		}()
	}
	wg.Wait()
	if q.Dropped() != 22 {
		t.Fatalf("queue overflow expected 22 got %d", q.Dropped())
	}
	if err := q.Publish(LeadFaultEvent{AccountID: binance.MainAccountID, Code: FaultAPI429}); err == nil {
		t.Fatal("Main injected alert into Lead queue")
	}
	if err := q.Publish(LeadFaultEvent{AccountID: binance.LeadAccountID, Code: "api_secret=LEAK"}); err == nil {
		t.Fatal("untrusted raw error accepted into outbound notifier")
	}
}

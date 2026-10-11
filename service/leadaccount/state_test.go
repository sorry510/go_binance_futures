package leadaccount

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	binance "go_binance_futures/feature/api/binance"
)

func runtimeProof(now time.Time) RiskSnapshot {
	return RiskSnapshot{
		AccountID:  binance.LeadAccountID,
		CapturedAt: now, PositionsComplete: true, OrdersComplete: true, PNLComplete: true,
		Reconciled: true, UnknownOrders: false, HedgeMode: true, AvailableUSDT: 50, EquityUSDT: 100,
		PNLDayStartUTC: now.UTC().Truncate(24 * time.Hour), PNLAsOf: now,
	}
}
func mockReadyState() riskRunState {
	return riskRunState{Enabled: true, AllowNewOpens: true, PortfolioBound: true, ReadOnlyVerified: true, WSHealthy: true, Stage7Authorized: true}
}
func assertRuntimeState(t *testing.T, g *LeadRuntimeGuard, want LeadRuntimeState) {
	t.Helper()
	got := g.Status()
	if got.State != want {
		t.Fatalf("state=%s expected %s, %+v", got.State, want, got)
	}
}
func TestStage45GuardDefaultDisabledAndSeparateExitEvaluation(t *testing.T) {
	g := NewLeadRuntimeGuard(nil)
	state := g.Status()
	if state.State != LeadDisabled || state.CanAttemptOpen || !state.EvaluateExits {
		t.Fatalf("incorrect default: %+v", state)
	}
	if g.permitsOpen() {
		t.Fatal("disabled guard should not authorize opens")
	}
	// permitsOpen itself is only a fault/pause precondition; it is not an
	// enabling API. The actual RiskController and readonly broker remain locked.
	g.observeBoundState(mockReadyState())
	ready := g.Status()
	if ready.State != LeadOpenReady || !ready.CanAttemptOpen || !ready.ExitWriteSafe {
		t.Fatalf("mock observation failed: %+v", ready)
	}
	g.Pause()
	s := g.Status()
	if s.State != LeadPaused || s.CanAttemptOpen || !s.EvaluateExits || !s.ExitWriteSafe {
		t.Fatalf("pausing broke safe exits: %+v", s)
	}
	g.observeBoundState(mockReadyState())
	assertRuntimeState(t, g, LeadPaused)
}
func TestStage45FaultMatrixPriorityAndStickyBehavior(t *testing.T) {
	g := NewLeadRuntimeGuard(nil)
	g.observeBoundState(mockReadyState())
	g.RecordFault(binance.LeadAccountID, FaultDailyLoss)
	assertRuntimeState(t, g, LeadRiskTripped)
	g.RecordFault(binance.LeadAccountID, FaultPendingReconcile)
	assertRuntimeState(t, g, LeadReconcileRequired)
	g.RecordFault(binance.LeadAccountID, FaultWSUnavailable)
	assertRuntimeState(t, g, LeadAccountUnavailable)
	result := g.Status()
	expected := []LeadFaultCode{FaultPendingReconcile, FaultDailyLoss, FaultWSUnavailable}
	if !reflect.DeepEqual(result.Reasons, expected) {
		t.Fatalf("unstable reasons order %v vs %v", result.Reasons, expected)
	}
	if result.CanAttemptOpen || result.ExitWriteSafe || !result.EvaluateExits {
		t.Fatalf("uncertain orders/WS allowed writes: %+v", result)
	}
	now := time.Now().UTC()
	g.observeFreshEvidence(runtimeProof(now), now, true)
	assertRuntimeState(t, g, LeadReconcileRequired)
	if !reflect.DeepEqual(g.Status().Reasons, []LeadFaultCode{FaultPendingReconcile, FaultDailyLoss}) {
		t.Fatalf("authoritative read cleared sticky faults: %+v", g.Status())
	}
	g.Pause()
	assertRuntimeState(t, g, LeadReconcileRequired)
}
func TestStage45TransientRecoveryRequiresFreshCompleteEvidence(t *testing.T) {
	g := NewLeadRuntimeGuard(nil)
	g.observeBoundState(mockReadyState())
	now := time.Now().UTC()
	g.RecordFault(binance.LeadAccountID, FaultWSUnavailable)
	g.RecordFault(binance.LeadAccountID, FaultAPI429)
	assertRuntimeState(t, g, LeadAccountUnavailable)
	bad := runtimeProof(now)
	bad.AccountID = binance.MainAccountID
	g.observeFreshEvidence(bad, now, true)
	assertRuntimeState(t, g, LeadAccountUnavailable)
	bad = runtimeProof(now)
	bad.CapturedAt = now.Add(-11 * time.Second)
	g.observeFreshEvidence(bad, now, true)
	assertRuntimeState(t, g, LeadAccountUnavailable)
	bad = runtimeProof(now)
	bad.Reconciled = false
	g.observeFreshEvidence(bad, now, true)
	assertRuntimeState(t, g, LeadAccountUnavailable)
	g.observeFreshEvidence(runtimeProof(now), now, false)
	assertRuntimeState(t, g, LeadAccountUnavailable)
	g.observeFreshEvidence(runtimeProof(now), now, true)
	assertRuntimeState(t, g, LeadOpenReady)
}
func TestStage45RiskThresholdTripNeverClearsWithUTCDateOrHealthyData(t *testing.T) {
	g := NewLeadRuntimeGuard(nil)
	g.observeBoundState(mockReadyState())
	g.recordRiskDecision(binance.LeadAccountID, RiskDecision{BlockingReasons: []string{"lead_daily_loss_limit", "lead_drawdown_limit"}})
	assertRuntimeState(t, g, LeadRiskTripped)
	tomorrow := time.Now().UTC().Add(24 * time.Hour)
	g.observeFreshEvidence(runtimeProof(tomorrow), tomorrow, true)
	assertRuntimeState(t, g, LeadRiskTripped)
	if len(g.Status().Reasons) != 2 {
		t.Fatalf("risk threshold reset unexpectedly: %+v", g.Status())
	}
	g.observeBoundState(mockReadyState())
	assertRuntimeState(t, g, LeadRiskTripped)
}
func TestStage45SkipDoesNotTripWholeLeadAccount(t *testing.T) {
	g := NewLeadRuntimeGuard(nil)
	g.observeBoundState(mockReadyState())
	for _, code := range []string{"lead_available_margin_insufficient", "lead_exchange_notional_limits", "lead_position_count_limit", "lead_symbol_not_allowed",
		"lead_same_side_position_or_order_exists", "stage7_live_authorization_required", "new_opens_paused", "no_signal", "private-https://key:secret@example.test"} {
		class, fault := ClassifyLeadOpenRejection(code)
		if class != FaultSkip || fault != "" {
			t.Fatalf("ordinary skip classified as breaker: %s -> %s %s", code, class, fault)
		}
		g.recordRiskDecision(binance.LeadAccountID, RiskDecision{BlockingReasons: []string{code}})
	}
	assertRuntimeState(t, g, LeadOpenReady)
	if g.RecordFault(binance.MainAccountID, FaultUnknownSubmission) {
		t.Fatal("Main was allowed to mutate Lead guard")
	}
	if g.RecordFault(binance.LeadAccountID, LeadFaultCode("private-API-KEY")) {
		t.Fatal("arbitrary error string accepted")
	}
	assertRuntimeState(t, g, LeadOpenReady)
}
func TestStage45FaultEventsSanitizedDeduplicatedAndFailureNeverUnlocks(t *testing.T) {
	now := time.Date(2026, 10, 10, 18, 0, 0, 0, time.UTC)
	var mu sync.Mutex
	events := []LeadFaultEvent{}
	sink := func(e LeadFaultEvent) error {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, e)
		return errors.New("deliberately failing notification sink")
	}
	g := NewLeadRuntimeGuard(sink)
	g.now = func() time.Time { return now }
	g.observeBoundState(mockReadyState())
	for i := 0; i < 20; i++ {
		g.RecordFault(binance.LeadAccountID, FaultAPI429)
	}
	if len(events) != 1 {
		t.Fatalf("dedup failed, emitted %d alerts", len(events))
	}
	if e := events[0]; e.Code != FaultAPI429 || e.Action != "raised" || e.AccountID != binance.LeadAccountID || e.Severity != SeverityWarning {
		t.Fatalf("bad alert %+v", e)
	}
	if g.Status().State != LeadAccountUnavailable {
		t.Fatal("notification failure removed fault")
	}
	now = now.Add(30 * time.Second)
	g.RecordFault(binance.LeadAccountID, FaultAPI418)
	if len(events) != 2 || events[1].Severity != SeverityCritical {
		t.Fatalf("418 escalation not emitted %+v", events)
	}
	now = now.Add(2*time.Minute + time.Second)
	g.RecordFault(binance.LeadAccountID, FaultAPI429)
	if len(events) != 3 {
		t.Fatalf("alert cooldown not observed: %+v", events)
	}
	proof := runtimeProof(now)
	g.observeFreshEvidence(proof, now, true)
	if len(events) != 5 || events[3].Action != "resolved" || events[4].Action != "resolved" {
		t.Fatalf("recovery events missing: %+v", events)
	}
	assertRuntimeState(t, g, LeadOpenReady)
	encoded := fmt.Sprint(events)
	for _, secret := range []string{"api_secret", "signature=", "private-API-KEY"} {
		if strings.Contains(encoded, secret) {
			t.Fatalf("secret leaked from bounded events: %s", encoded)
		}
	}
}
func TestStage45RuntimeStateParallelReadAndFaultReport(t *testing.T) {
	g := NewLeadRuntimeGuard(nil)
	g.observeBoundState(mockReadyState())
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for n := 0; n < 100; n++ {
				if i%3 == 0 {
					g.RecordFault(binance.LeadAccountID, FaultUnknownAlgo)
				}
				if i%3 == 1 {
					_ = g.Status()
				}
				if i%3 == 2 {
					g.observeFreshEvidence(runtimeProof(time.Now().UTC()), time.Now().UTC(), true)
				}
			}
		}(i)
	}
	wg.Wait()
	assertRuntimeState(t, g, LeadReconcileRequired)
}
func TestStage45ExecutorFaultPauseAndUnknownWriteGate(t *testing.T) {
	prepareLeadExecutionDB(t)
	a, broker := stage43Adapter(t, true)
	req, check := stage43Request("LONG", "MARKET", "stage45_guard_1")
	a.ReportLeadFault(FaultWSUnavailable)
	if _, err := a.ExecuteOpen(context.Background(), req, check); !errors.Is(err, ErrLeadOpenBlocked) {
		t.Fatalf("WS failed yet allowed open: %v", err)
	}
	if broker.submits != 0 || a.RuntimeStatus().State != LeadAccountUnavailable {
		t.Fatalf("WS failure not isolated: status=%+v submissions=%d", a.RuntimeStatus(), broker.submits)
	}
	// Fresh evidence can remove only the temporary category, not Stage 7 gates.
	proof := runtimeProof(time.Now().UTC())
	a.guard.observeFreshEvidence(proof, time.Now().UTC(), true)
	a.PauseOpens()
	if _, err := a.ExecuteOpen(context.Background(), req, check); !errors.Is(err, ErrLeadOpenBlocked) {
		t.Fatalf("paused opens accepted: %v", err)
	}
	if a.RuntimeStatus().State != LeadPaused || broker.submits != 0 {
		t.Fatalf("pause altered broker %d status=%+v", broker.submits, a.RuntimeStatus())
	}
	a.ReportLeadFault(FaultUnknownSubmission)
	if a.RuntimeStatus().State != LeadReconcileRequired {
		t.Fatalf("unknown submission not sticky: %+v", a.RuntimeStatus())
	}
	proof = runtimeProof(time.Now().UTC())
	a.guard.observeFreshEvidence(proof, time.Now().UTC(), true)
	if a.RuntimeStatus().State != LeadReconcileRequired {
		t.Fatal("read-only recovery illegally unlocked unknown submission")
	}
}
func TestStage45PausedOpenStillAllowsVerifiedManagedExit(t *testing.T) {
	a, broker, _ := stage43OwnedFixture(t, "LONG", 1)
	a.PauseOpens()
	req := stage43CloseRequest("LONG", "MARKET", 1)
	if _, err := a.ExecuteManagedClose(context.Background(), req); err != nil {
		t.Fatalf("pause interfered with owned safe exit: %v", err)
	}
	if broker.submits != 1 {
		t.Fatalf("paused owned exit submitted %d times", broker.submits)
	}
	if a.RuntimeStatus().State != LeadReconcileRequired {
		t.Fatalf("post-exit needs reconcile: %+v", a.RuntimeStatus())
	}
}
func TestStage45UnsafeExitBlockedAndCannotTouchManualPositions(t *testing.T) {
	a, broker, _ := stage43OwnedFixture(t, "SHORT", 1)
	a.ReportLeadFault(FaultIdentityInvalid)
	req := stage43CloseRequest("SHORT", "MARKET", 1)
	if _, err := a.ExecuteManagedClose(context.Background(), req); !errors.Is(err, ErrLeadOpenBlocked) {
		t.Fatalf("identity failure permitted exchange exit: %v", err)
	}
	if broker.submits != 0 {
		t.Fatalf("unsafe exit hit broker: %d", broker.submits)
	}
}
func TestStage45RateLimitAndProtectionFaultBlocksExitWrite(t *testing.T) {
	for _, code := range []LeadFaultCode{FaultAPI429, FaultAPI418, FaultProtectionFailure, FaultUnknownAlgo, FaultCredentialsChanged} {
		t.Run(string(code), func(t *testing.T) {
			a, broker, _ := stage43OwnedFixture(t, "LONG", 1)
			a.ReportLeadFault(code)
			request := stage43CloseRequest("LONG", "MARKET", 1)
			if _, err := a.ExecuteManagedClose(context.Background(), request); !errors.Is(err, ErrLeadOpenBlocked) {
				t.Fatalf("unsafe exit wasn't blocked for %q: %v", code, err)
			}
			if broker.submits != 0 {
				t.Fatalf("unsafe exit submitted %d orders", broker.submits)
			}
			if !a.RuntimeStatus().EvaluateExits {
				t.Fatal("still must be able to evaluate exit intent")
			}
		})
	}
}
func TestStage45KnownFaultsOnlyAndAlertScope(t *testing.T) {
	g := NewLeadRuntimeGuard(nil)
	for _, fault := range allFaultCodes() {
		class, _, valid := faultDefinition(fault)
		if !valid || class == FaultSkip {
			t.Fatalf("invalid fault registry entry: %s", fault)
		}
		if g.RecordFault(binance.MainAccountID, fault) {
			t.Fatalf("Main can mutate Lead fault %q", fault)
		}
	}
	if g.Status().State != LeadDisabled {
		t.Fatal("Main fault changed Lead state")
	}
}

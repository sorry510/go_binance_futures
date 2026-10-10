package leadaccount

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/adshao/go-binance/v2/common"
	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/client/orm"
	binance "go_binance_futures/feature/api/binance"
	"go_binance_futures/service/futuresownership"
)

// Stage 4-6 acceptance uses only the in-memory Ownership database, fake
// credentials, injected Binance readers, and a recording Fake Broker. No
// production Lead authorization is ever enabled by these tests.
func TestStage46CredentialToRiskToExecutionRemainsLocked(t *testing.T) {
	store := newTestEncryptedStore(t)
	reader := validFakeReader()
	manager := newManager(store, func(c Credentials) (AccountReader, error) {
		if c.APIKey != "lead-stage46-key" {
			t.Fatal("credential source mismatch")
		}
		return reader, nil
	})
	if err := manager.SaveCredentials(Credentials{APIKey: "lead-stage46-key", APISecret: "top-secret-stage46"}); err != nil {
		t.Fatal(err)
	}
	verified, err := manager.VerifyReadOnly(context.Background())
	if err != nil || !verified.ReadOnlyChecksPassed || !verified.LeadIdentityOK {
		t.Fatalf("Lead-only read verification incomplete: %+v %v", verified, err)
	}
	if verified.PortfolioBindingConfirmed || verified.TradingReady {
		t.Fatalf("read-only status gave live authorization: %+v", verified)
	}
	data, err := json.Marshal(verified)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "top-secret-stage46") || strings.Contains(string(data), "lead-stage46-key") {
		t.Fatal("credential leaked")
	}
	a, broker := stage43Adapter(t, false)
	request, check := stage43Request("LONG", "MARKET", "stage46_read_only_locked")
	decision, err := a.CheckOpen(context.Background(), check)
	if !errors.Is(err, ErrLeadOpenBlocked) || decision.Allowed {
		t.Fatalf("Read-only verification bypassed real gate: %+v %v", decision, err)
	}
	if _, err = a.ExecuteOpen(context.Background(), request, check); !errors.Is(err, ErrLeadOpenBlocked) {
		t.Fatalf("lead open unexpectedly available: %v", err)
	}
	if broker.submits != 0 || len(a.source.Reader.(*fakeLeadSnapshotReader).calls) != 0 {
		t.Fatalf("production-locked instance touched Binance private reads or orders: %d", broker.submits)
	}
	if err := manager.SaveCredentials(Credentials{APIKey: "rotated", APISecret: "different-secret"}); err != nil {
		t.Fatal(err)
	}
	rotated := manager.Status()
	if rotated.ReadOnlyChecksPassed || rotated.TradingReady {
		t.Fatalf("rotation retained old safety evidence: %+v", rotated)
	}
}

func TestStage46AccountScopedMockOpenMatrixAndStickyReconcile(t *testing.T) {
	for _, tc := range []struct{ side, kind string }{
		{"LONG", "MARKET"}, {"SHORT", "MARKET"}, {"LONG", "LIMIT"}, {"SHORT", "LIMIT"},
	} {
		t.Run(tc.side+"_"+tc.kind, func(t *testing.T) {
			prepareLeadExecutionDB(t)
			a, broker := stage43Adapter(t, true)
			request, check := stage43Request(tc.side, tc.kind, "stage46_"+tc.side+"_"+tc.kind)
			result, err := a.ExecuteOpen(context.Background(), request, check)
			if err != nil || result.FilledQty != 1 || broker.submits != 1 {
				t.Fatalf("Lead mock open missing %+v err=%v submit=%d", result, err, broker.submits)
			}
			order, err := a.ownership.Ownership.GetOrder(context.Background(), request.ClientOrderID)
			if err != nil || order.AccountID != "lead" || order.Status != futuresownership.OrderFilled {
				t.Fatalf("wrong account/claim %+v %v", order, err)
			}
			pos, err := a.ownership.Ownership.GetPosition(context.Background(), futuresownership.OwnerAutoStrategy, "BTCUSDT", tc.side)
			if err != nil || pos.AccountID != "lead" || pos.ManagedQty != 1 {
				t.Fatalf("wrong managed fill %+v %v", pos, err)
			}
			if _, err = a.ExecuteOpen(context.Background(), request, check); !errors.Is(err, ErrLeadPendingReconcile) {
				t.Fatalf("accepted duplicate intent after fill: %v", err)
			}
			if broker.submits != 1 || a.RuntimeStatus().State != LeadReconcileRequired {
				t.Fatalf("lost pending gate submit=%d state=%s", broker.submits, a.RuntimeStatus().State)
			}
			evidence, _ := ClassifyLeadRecovery(futuresownership.ExchangeOrder{
				ExchangeOrderID: "900", ClientOrderID: request.ClientOrderID, Status: "FILLED", FilledQty: 1, AveragePrice: 100,
			}, request.ClientOrderID)
			if !evidence.Terminal || !evidence.RequiresReconcile || !a.pendingReconcile {
				t.Fatalf("terminal order alone illegally unlocked adapter: %+v", evidence)
			}
		})
	}
}

func TestStage46PartialFillCancelIdempotentAndAccountSeparation(t *testing.T) {
	prepareLeadExecutionDB(t)
	a, _ := stage43Adapter(t, true)
	main, err := futuresownership.BindAccount(binance.MainAccountID)
	if err != nil {
		t.Fatal(err)
	}
	clientID := "stage46_partial_open"
	_, err = a.ownership.Ownership.ClaimOrder(context.Background(), futuresownership.ClaimOrderInput{
		Owner: futuresownership.OwnerAutoStrategy, Symbol: "BTCUSDT", PositionSide: "LONG",
		Intent: futuresownership.IntentOpen, ClientOrderID: clientID, RequestedQty: 1, OrderType: "LIMIT",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = main.ClaimOrder(context.Background(), futuresownership.ClaimOrderInput{
		Owner: futuresownership.OwnerAutoStrategy, Symbol: "BTCUSDT", PositionSide: "LONG",
		Intent: futuresownership.IntentOpen, ClientOrderID: "stage46_main_sibling", RequestedQty: 2, OrderType: "LIMIT",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, qty := range []float64{0.3, 0.3, 0.5, 0.5} {
		status := "PARTIALLY_FILLED"
		if qty == 0.5 {
			status = "CANCELED"
		}
		observed := futuresownership.ExchangeOrder{ExchangeOrderID: "734", ClientOrderID: clientID, Status: status, FilledQty: qty, AveragePrice: 100}
		recovered, err := ClassifyLeadRecovery(observed, clientID)
		if err != nil || !recovered.RequiresReconcile {
			t.Fatalf("invalid cumulative evidence: %+v %v", recovered, err)
		}
		if _, err = a.ownership.ApplyObservedExchange(context.Background(), clientID, recovered.Order); err != nil {
			t.Fatal(err)
		}
	}
	p, err := a.ownership.Ownership.GetPosition(context.Background(), futuresownership.OwnerAutoStrategy, "BTCUSDT", "LONG")
	if err != nil || math.Abs(p.ManagedQty-0.5) > 1e-10 || p.AccountID != "lead" {
		t.Fatalf("partial fill duplicated %+v %v", p, err)
	}
	if _, err = main.GetOrder(context.Background(), clientID); !errors.Is(err, orm.ErrNoRows) {
		t.Fatalf("Main can see Lead fill: %v", err)
	}
	if _, err = a.ownership.Ownership.GetOrder(context.Background(), "stage46_main_sibling"); !errors.Is(err, orm.ErrNoRows) {
		t.Fatalf("Lead can see Main claim: %v", err)
	}
	mainOrder, err := main.GetOrder(context.Background(), "stage46_main_sibling")
	if err != nil || mainOrder.AccountID != "main" || mainOrder.Status != futuresownership.OrderPending {
		t.Fatalf("Lead recovery changed Main order: %+v %v", mainOrder, err)
	}
}

func TestStage46MockProtectiveStopsAndTargetsWithManagedCaps(t *testing.T) {
	cases := []struct {
		side, kind, intent string
		stop               float64
	}{
		{"LONG", "STOP_MARKET", futuresownership.IntentStopLoss, 95.5},
		{"LONG", "TAKE_PROFIT_MARKET", futuresownership.IntentTakeProfit, 105.5},
		{"SHORT", "STOP_MARKET", futuresownership.IntentStopLoss, 105.5},
		{"SHORT", "TAKE_PROFIT_MARKET", futuresownership.IntentTakeProfit, 95.5},
	}
	for _, tc := range cases {
		t.Run(tc.side+"_"+tc.kind, func(t *testing.T) {
			a, broker, reader := stage43OwnedFixture(t, tc.side, 1)
			request := stage43CloseRequest(tc.side, tc.kind, 1)
			request.Intent = tc.intent
			request.StopPrice = tc.stop
			broker.result = futuresownership.ExchangeOrder{ExchangeOrderID: "901", Status: "FILLED", FilledQty: 1, AveragePrice: 100}
			_, err := a.ExecuteManagedClose(context.Background(), request)
			if err != nil || broker.submits != 1 {
				t.Fatalf("protective close failed: %v submits=%d", err, broker.submits)
			}
			positions, err := a.ownership.Ownership.ListPositions(context.Background(), futuresownership.OwnerAutoStrategy, false)
			if err != nil || len(positions) != 1 || positions[0].ManagedQty != 0 || positions[0].Status != futuresownership.PositionClosed {
				t.Fatalf("managed protective fill wrong %+v %v", positions, err)
			}
			if a.RuntimeStatus().State != LeadReconcileRequired {
				t.Fatalf("protective order lost pending status: %+v", a.RuntimeStatus())
			}
			// A manual add cannot be included in managed quantity.
			reader.positions[0].PositionAmt = "2"
			request.ClientOrderID += "_manual"
			if _, err = a.ExecuteManagedClose(context.Background(), request); !errors.Is(err, ErrLeadPendingReconcile) {
				t.Fatalf("second protective write must await reconciler: %v", err)
			}
			if broker.submits != 1 {
				t.Fatal("unmanaged/manual quantity was submitted")
			}
		})
	}
}

func TestStage46RiskAndFaultMatrixNeverReachFakeExchange(t *testing.T) {
	for _, tc := range []struct {
		name   string
		fault  LeadFaultCode
		expect LeadRuntimeState
	}{
		{"ws_stale", FaultWSUnavailable, LeadAccountUnavailable},
		{"429", FaultAPI429, LeadAccountUnavailable},
		{"418", FaultAPI418, LeadAccountUnavailable},
		{"unknown_algo", FaultUnknownAlgo, LeadReconcileRequired},
		{"daily_loss", FaultDailyLoss, LeadRiskTripped},
		{"drawdown", FaultDrawdown, LeadRiskTripped},
		{"portfolio_identity", FaultPortfolioUnconfirmed, LeadAccountUnavailable},
		{"protection_failure", FaultProtectionFailure, LeadAccountUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, broker := stage43Adapter(t, true)
			req, check := stage43Request("SHORT", "MARKET", "stage46_fault_"+tc.name)
			a.ReportLeadFault(tc.fault)
			if a.RuntimeStatus().State != tc.expect {
				t.Fatalf("fault %s state %+v", tc.fault, a.RuntimeStatus())
			}
			for i := 0; i < 3; i++ {
				if _, err := a.ExecuteOpen(context.Background(), req, check); !errors.Is(err, ErrLeadOpenBlocked) {
					t.Fatalf("fault %q bypass: %v", tc.fault, err)
				}
			}
			if broker.submits != 0 {
				t.Fatalf("fault %q emitted %d orders", tc.fault, broker.submits)
			}
			fresh := runtimeProof(time.Now().UTC())
			a.guard.observeFreshEvidence(fresh, time.Now().UTC(), true)
			if tc.expect == LeadRiskTripped || tc.expect == LeadReconcileRequired || tc.fault == FaultPortfolioUnconfirmed || tc.fault == FaultProtectionFailure {
				if a.RuntimeStatus().State != tc.expect {
					t.Fatalf("persistent fault %s auto-reset", tc.fault)
				}
			}
		})
	}
}

func TestStage46RecoveryTerminalEvidenceCannotReleaseOpenOrCancelGate(t *testing.T) {
	a, broker := stage43Adapter(t, true)
	a.pendingReconcile = true
	a.guard.RecordFault(binance.LeadAccountID, FaultPendingReconcile)
	req, check := stage43Request("LONG", "MARKET", "stage46_terminal_needs_resync")
	e, err := ClassifyLeadRecovery(futuresownership.ExchangeOrder{
		ExchangeOrderID: "555", ClientOrderID: req.ClientOrderID, Status: "CANCELED", FilledQty: 0.4, AveragePrice: 100,
	}, req.ClientOrderID)
	if err != nil || !e.Terminal || !e.RequiresReconcile {
		t.Fatalf("bad canceled evidence %+v err=%v", e, err)
	}
	if _, err = a.ExecuteOpen(context.Background(), req, check); !errors.Is(err, ErrLeadPendingReconcile) {
		t.Fatalf("terminal evidence released pending: %v", err)
	}
	if broker.submits != 0 {
		t.Fatal("unknown order was retried")
	}
	client, err := binance.NewAccountClient(binance.LeadAccountID, futures.NewClient("fixture", "fake"))
	if err != nil {
		t.Fatal(err)
	}
	locked, err := NewReadOnlyOrderBroker(client)
	if err != nil {
		t.Fatal(err)
	}
	if err = locked.Cancel(context.Background(), "BTCUSDT", 555, "MARKET"); !errors.Is(err, ErrLiveExecutionLocked) {
		t.Fatalf("read-only Cancel unlocked: %v", err)
	}
	if _, err = locked.Submit(context.Background(), req, req.ClientOrderID); !errors.Is(err, ErrLiveExecutionLocked) {
		t.Fatalf("read-only Submit unlocked: %v", err)
	}
}

func TestStage46GuardCannotPromoteEvidenceToLiveAuthorization(t *testing.T) {
	g := NewLeadRuntimeGuard(nil)
	g.observeBoundState(mockReadyState())
	evidence := runtimeProof(time.Now().UTC())
	g.observeFreshEvidence(evidence, time.Now().UTC(), true)
	if g.Status().State != LeadOpenReady {
		t.Fatalf("simulation not ready: %+v", g.Status())
	}
	actual, broker := stage43Adapter(t, false)
	req, check := stage43Request("LONG", "LIMIT", "stage46_state_not_authorization")
	if _, err := actual.ExecuteOpen(context.Background(), req, check); !errors.Is(err, ErrLeadOpenBlocked) || broker.submits != 0 {
		t.Fatalf("modeled readiness activated production broker: %v %d", err, broker.submits)
	}
	// Prevent accidentally adding a public unsafe Reset/Resume/Enable surface.
	for _, target := range []any{g, actual, NewRiskController()} {
		ty := reflect.TypeOf(target)
		for _, unsafeName := range []string{"Enable", "Resume", "Reset", "ResetPending", "Unlock", "Authorize", "StartLiveTrading"} {
			if _, exists := ty.MethodByName(unsafeName); exists {
				t.Fatalf("%s exported unsafe %s", ty, unsafeName)
			}
		}
	}
}

func TestStage46ExchangeRuleOutageHasVisibleFailClosedStatus(t *testing.T) {
	a, broker := stage43Adapter(t, true)
	a.simulateRules = nil
	a.rules.Reader = &fakeLeadRulesReader{exchange: nil}
	req, check := stage43Request("LONG", "LIMIT", "stage46_rules_outage")
	decision, err := a.CheckOpen(context.Background(), check)
	if !errors.Is(err, ErrLeadOpenBlocked) || decision.Allowed {
		t.Fatalf("unavailable exchange filters accepted: %+v %v", decision, err)
	}
	status := a.RuntimeStatus()
	if status.State != LeadAccountUnavailable || !containsLeadFault(status.Reasons, FaultRulesUnavailable) {
		t.Fatalf("exchange rules failure not visible: %+v", status)
	}
	if _, err := a.ExecuteOpen(context.Background(), req, check); !errors.Is(err, ErrLeadOpenBlocked) {
		t.Fatalf("opening ignored exchange rule failure: %v", err)
	}
	if broker.submits != 0 {
		t.Fatalf("order reached broker despite unavailable rules: %d", broker.submits)
	}
}

func containsLeadFault(reasons []LeadFaultCode, want LeadFaultCode) bool {
	for _, r := range reasons {
		if r == want {
			return true
		}
	}
	return false
}

func TestStage46DeterministicRejectionAndUnknownTimeoutDifferentLedgerOutcomes(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		lookupErr  error
		wantStatus string
		wantLookup int
	}{
		{"deterministic_reject", &common.APIError{Code: -2019, Message: "margin is insufficient"}, nil, futuresownership.OrderFailed, 0},
		{"timeout_uncertain", context.DeadlineExceeded, errors.New("no order evidence"), futuresownership.OrderReconcile, 1},
		{"network_unknown", errors.New("unknown local HTTP failure"), errors.New("query failed"), futuresownership.OrderReconcile, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prepareLeadExecutionDB(t)
			a, broker := stage43Adapter(t, true)
			broker.err = tc.err
			broker.lookupErr = tc.lookupErr
			req, check := stage43Request("LONG", "LIMIT", "stage46_rejection_"+tc.name)
			_, err := a.ExecuteOpen(context.Background(), req, check)
			if err == nil {
				t.Fatal("simulated rejected/unknown order unexpectedly succeeded")
			}
			order, err := a.ownership.Ownership.GetOrder(context.Background(), req.ClientOrderID)
			if err != nil || order.Status != tc.wantStatus || order.AccountID != "lead" {
				t.Fatalf("wrong persisted status %+v error=%v", order, err)
			}
			if broker.submits != 1 || broker.lookups != tc.wantLookup {
				t.Fatalf("wrong submit/lookup side effects: submit=%d lookups=%d", broker.submits, broker.lookups)
			}
			if !a.pendingReconcile || a.RuntimeStatus().State != LeadReconcileRequired {
				t.Fatalf("rejection unexpectedly released pending gate: %+v", a.RuntimeStatus())
			}
			if _, err = a.ExecuteOpen(context.Background(), req, check); !errors.Is(err, ErrLeadPendingReconcile) {
				t.Fatalf("unreconciled rejection/timeout retried: %v", err)
			}
			if broker.submits != 1 {
				t.Fatal("duplicate order leaked after reject/timeout")
			}
		})
	}
}

// A second, independent Stage 4 safety net against a new exported mutation
// bypass that the name-based AST scan might miss. Any future public adapter
// method must be explicitly reviewed and allowlisted (not auto-accepted).
func TestStage46LeadExecutionAdapterPublicSurfaceAllowlist(t *testing.T) {
	expected := []string{
		"CheckOpen",
		"ExecuteManagedClose",
		"ExecuteOpen",
		"InspectOrderRecovery",
		"PauseOpens",
		"ReportLeadFault",
		"RuntimeStatus",
	}
	typ := reflect.TypeOf((*LeadExecutionAdapter)(nil))
	methods := make([]string, 0, typ.NumMethod())
	for i := 0; i < typ.NumMethod(); i++ {
		methods = append(methods, typ.Method(i).Name)
	}
	if !reflect.DeepEqual(methods, expected) {
		t.Fatalf("Lead adapter public methods changed; review production write boundary before accepting: got=%v want=%v", methods, expected)
	}
	guardType := reflect.TypeOf((*LeadRuntimeGuard)(nil))
	guardMethods := make([]string, 0, guardType.NumMethod())
	for i := 0; i < guardType.NumMethod(); i++ {
		guardMethods = append(guardMethods, guardType.Method(i).Name)
	}
	wantGuard := []string{"Pause", "RecordFault", "Status"}
	if !reflect.DeepEqual(guardMethods, wantGuard) {
		t.Fatalf("Lead runtime guard exposed new permission/mutation surface: got=%v want=%v", guardMethods, wantGuard)
	}
}

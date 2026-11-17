package leadaccount

import (
	"context"
	"errors"
	"fmt"
	"github.com/adshao/go-binance/v2/futures"
	"math"
	"strings"
	"sync"
	"time"

	binance "go_binance_futures/feature/api/binance"
	"go_binance_futures/service/futuresownership"
)

var (
	ErrLeadOpenBlocked      = errors.New("lead opening order blocked by safety gate")
	ErrLeadPendingReconcile = errors.New("lead account has unresolved orders or incomplete reconciliation")
)

// LeadExecutionAdapter is the account-scoped execution boundary. Construction
// does not enable trading: only a dedicated, healthy Lead session plus the
// persisted user switch and the existing risk/ownership gates allow entries.
type LeadExecutionAdapter struct {
	mu        sync.Mutex // serializes this instance; NOT a distributed reservation
	account   *binance.AccountClient
	ownership futuresownership.Executor
	risk      *RiskController
	source    *RiskEvidenceSource
	rules     *LeadRuleSource
	symbols   *binance.LeadSymbolCache
	// This private hook is populated ONLY by isolated _test.go fixtures.
	// Production has no route, constructor parameter or activation of it.
	simulateEvidence func(*RiskSnapshot)
	simulateRules    func(RiskOpenOrder) (VerifiedOrderRules, error)
	pendingReconcile bool // only unknown/failed order result requires recovery
	guard            *LeadRuntimeGuard
}

func NewLeadExecutionAdapter(client *binance.AccountClient, symbols *binance.LeadSymbolCache) (*LeadExecutionAdapter, error) {
	if client == nil || client.ID() != binance.LeadAccountID || symbols == nil {
		return nil, fmt.Errorf("Lead execution requires dedicated Lead account client and whitelist")
	}
	rules, err := NewLeadRuleSource(client)
	if err != nil {
		return nil, err
	}
	source, err := NewRiskEvidenceSource(client)
	if err != nil {
		return nil, err
	}
	owned, err := futuresownership.NewAccountExecutor(client)
	if err != nil {
		return nil, err
	}
	if owned.Ownership.AccountID != string(binance.LeadAccountID) {
		return nil, fmt.Errorf("Lead ownership scope not bound")
	}
	broker, ok := owned.Broker.(futuresownership.BinanceOrderBroker)
	if !ok || broker.Account != client {
		return nil, fmt.Errorf("Lead broker account binding invalid")
	}
	return &LeadExecutionAdapter{
		account: client, ownership: owned, risk: NewRiskController(),
		source: source, rules: rules, symbols: symbols,
		guard: NewLeadRuntimeGuard(nil),
	}, nil
}

// ConfigureLiveSession is called only from the existing shared Main tick
// after the dedicated Lead WS and signed account mirror are healthy.
// The user controls new entries through a persisted, default-OFF switch.
// ReadOnlyVerified/PortfolioBound in the legacy risk struct mean a scoped,
// healthy *runtime session* here; they do NOT assert a separate Binance
// Portfolio ID verification (which the operator elected to skip).
func (a *LeadExecutionAdapter) ConfigureLiveSession(settings LeadSettings, maxPositions, maxLosing int) error {
	if a == nil || a.account == nil || a.account.ID() != binance.LeadAccountID ||
		a.symbols == nil || !a.symbols.BoundTo(a.account) ||
		a.risk == nil || settings.AccountID != "lead" ||
		!LeadRiskLimitsReady(settings.Limits) ||
		!leadTradeSessionHealthyFor(a.account) {
		return ErrLeadOpenBlocked
	}
	if err := ValidateLeadSettings(settings); err != nil {
		return err
	}
	// Only the new-entry branch requires the durable switch. Disabling it
	// must not block an already-owned position's safe exit.
	if settings.AllowNewOpens {
		if err := LeadDurableOpenGate(); err != nil {
			return err
		}
	}
	// Entry limits of zero mean Main is not configured to create new
	// positions; continue existing Lead exits, but never enable entries.
	if maxPositions < 1 {
		maxPositions = 1
	}
	if maxLosing < 1 {
		maxLosing = 1
	}
	a.risk.mu.Lock()
	a.risk.limits = RiskLimits{
		MaxOrderNotionalUSDT:     settings.Limits.MaxOrderNotionalUSDT,
		MaxTotalNotionalUSDT:     settings.Limits.MaxTotalNotionalUSDT,
		MaxDailyRealizedLossUSDT: settings.Limits.MaxDailyRealizedLossUSDT,
		MaxPositions:             maxPositions,
		MaxLosingPositions:       maxLosing,
	}
	a.risk.state = riskRunState{
		Enabled:       true,
		AllowNewOpens: settings.AllowNewOpens && !LeadEmergencyPaused(),
		// Session identity comes from the dedicated Lead Client and live
		// WS mirror, not a skipped Portfolio verification request.
		PortfolioBound: true, ReadOnlyVerified: true,
		WSHealthy: true, Stage7Authorized: true,
	}
	state := a.risk.state
	a.risk.mu.Unlock()
	if a.guard != nil {
		a.guard.observeBoundState(state)
	}
	return nil
}

// RuntimeStatus is diagnostic only and never authorizes an order.
func (a *LeadExecutionAdapter) RuntimeStatus() LeadStatusSnapshot {
	if a == nil || a.guard == nil {
		return NewLeadRuntimeGuard(nil).Status()
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.risk.mu.RLock()
	state := a.risk.state
	a.risk.mu.RUnlock()
	a.guard.observeBoundState(state)
	return a.guard.Status()
}

// PauseOpens is one-way in Stage 4-5: it cannot authorize resumption.
func (a *LeadExecutionAdapter) PauseOpens() {
	if a == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.guard != nil {
		a.guard.Pause()
	}
	if a.risk != nil {
		a.risk.Pause()
	}
}

// ReportLeadFault accepts only fixed category codes. This is independent of
// Main alerts and never forwards raw Binance HTTP error bodies.
func (a *LeadExecutionAdapter) ReportLeadFault(code LeadFaultCode) {
	if a == nil {
		return
	}
	// Serialize fault observation with the intent check and Ownership claim.
	// A fault reported between the final preflight and the broker submit must
	// not be lost by a competing state transition.
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.guard != nil {
		a.guard.RecordFault(binance.LeadAccountID, code)
	}
}

func (a *LeadExecutionAdapter) completeAuthoritativeRecovery(report LeadRecoveryReport, mirror LeadMirrorStatus) bool {
	if a == nil || a.guard == nil || report.AccountID != binance.LeadAccountID ||
		report.Blocked || len(report.Reasons) != 0 || report.ObservedAt.IsZero() ||
		mirror.AccountID != binance.LeadAccountID || !mirror.Ready || mirror.Generation == 0 ||
		mirror.CapturedAt.Before(report.ObservedAt) ||
		time.Since(mirror.CapturedAt) > RiskSnapshotMaxAge {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pendingReconcile = false
	a.guard.clearReconciledFaults()
	return true
}

// CheckOpen is a read-only diagnostic. The same per-adapter lock protects
// it from racing a new order reservation.
func (a *LeadExecutionAdapter) CheckOpen(ctx context.Context, request RiskOpenOrder) (RiskDecision, error) {
	if a == nil {
		return RiskDecision{}, ErrLeadOpenBlocked
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.checkOpenLocked(ctx, request)
}
func (a *LeadExecutionAdapter) checkOpenLocked(ctx context.Context, request RiskOpenOrder) (RiskDecision, error) {
	if a.pendingReconcile {
		return RiskDecision{BlockingReasons: []string{"lead_pending_reconcile"}}, ErrLeadPendingReconcile
	}
	if a == nil || a.account == nil || a.account.ID() != binance.LeadAccountID || a.risk == nil || a.source == nil || a.symbols == nil {
		return RiskDecision{}, ErrLeadOpenBlocked
	}
	if request.AccountID != binance.LeadAccountID {
		return RiskDecision{}, ErrLeadOpenBlocked
	}
	// Stage 6 emergency halt is independent of the audit store and HTTP
	// controller lifetime. Keep this in the actual pre-submit gate so a
	// future explicit Stage 7 runner cannot bypass the process-local stop.
	if LeadEmergencyPaused() {
		return RiskDecision{BlockingReasons: []string{"lead_emergency_paused"}}, ErrLeadOpenBlocked
	}
	if a.simulateEvidence == nil {
		if !leadTradeSessionHealthyFor(a.account) {
			return RiskDecision{BlockingReasons: []string{"lead_ws_not_healthy"}}, ErrLeadOpenBlocked
		}
		// Re-read durable safety state, independent of HTTP handler memory.
		// A zero/absent config can never act as authorization.
		if err := LeadDurableOpenGate(); err != nil {
			return RiskDecision{BlockingReasons: []string{"lead_durable_gate_locked"}}, ErrLeadOpenBlocked
		}
	}
	// Require the configured live Lead session before any exchange order
	// checks. No Main account or Main broker fallback is allowed.
	a.risk.mu.RLock()
	gates := a.risk.state
	a.risk.mu.RUnlock()
	if a.guard != nil {
		a.guard.observeBoundState(gates)
		if !a.guard.permitsOpen() {
			return RiskDecision{BlockingReasons: []string{string(a.guard.Status().State)}}, ErrLeadOpenBlocked
		}
	}
	if !gates.Enabled || !gates.AllowNewOpens || !gates.ReadOnlyVerified ||
		!gates.PortfolioBound || !gates.WSHealthy || !gates.Stage7Authorized {
		return RiskDecision{BlockingReasons: []string{"lead_execution_not_authorized"}}, ErrLeadOpenBlocked
	}

	// Ignore untrusted validation flags, notional limits and leverage tier.
	// Recompute from public exchangeInfo and signed Lead portfolio reads.
	if a.rules == nil {
		if a.guard != nil {
			a.guard.RecordFault(binance.LeadAccountID, FaultRulesUnavailable)
		}
		return RiskDecision{BlockingReasons: []string{"lead_exchange_rules_unavailable"}}, ErrLeadOpenBlocked
	}
	margin := futures.MarginType(strings.ToUpper(strings.TrimSpace(request.MarginType)))
	if margin != futures.MarginTypeCrossed && margin != futures.MarginTypeIsolated {
		return RiskDecision{BlockingReasons: []string{"lead_margin_mode_invalid"}}, ErrLeadOpenBlocked
	}
	var verified VerifiedOrderRules
	var ruleErr error
	if a.simulateRules != nil {
		verified, ruleErr = a.simulateRules(request)
	} else {
		verified, ruleErr = a.rules.Verify(ctx, request, margin)
	}
	if ruleErr != nil {
		if a.guard != nil {
			a.guard.RecordFault(binance.LeadAccountID, FaultRulesUnavailable)
		}
		return RiskDecision{BlockingReasons: []string{"lead_exchange_rules_unavailable"}}, ErrLeadOpenBlocked
	}
	request.BinanceMinNotionalUSDT = verified.MinNotionalUSDT
	request.BinanceMaxNotionalUSDT = verified.MaxNotionalUSDT
	request.BinanceMaxLeverage = verified.MaxLeverage
	request.PriceAndQuantityValidated = true
	request.MarginModeVerified = true
	snapshot, err := a.source.Snapshot(ctx)
	if err != nil {
		if a.guard != nil {
			a.guard.RecordFault(binance.LeadAccountID, FaultSnapshotInvalid)
		}
		return RiskDecision{BlockingReasons: []string{"lead_risk_snapshot_unavailable"}}, ErrLeadOpenBlocked
	}
	if a.simulateEvidence != nil {
		a.simulateEvidence(&snapshot)
	} else {
		// REST Snapshot verifies current Lead account/hedge/positions/opens.
		// A local unknown order is still a blocker; an ordinary accepted
		// NEW order is not an unverified exchange submission.
		pending, pendingErr := a.ownership.Ownership.OutstandingOrders(ctx)
		if pendingErr != nil {
			return RiskDecision{BlockingReasons: []string{"lead_ownership_unavailable"}}, ErrLeadOpenBlocked
		}
		for _, order := range pending {
			if order.AccountID != string(binance.LeadAccountID) ||
				order.Status == futuresownership.OrderReconcile ||
				order.Status == futuresownership.OrderPending {
				return RiskDecision{BlockingReasons: []string{"lead_pending_reconcile"}}, ErrLeadPendingReconcile
			}
		}
		snapshot.Reconciled = true
		snapshot.UnknownOrders = false
	}
	if a.guard != nil {
		a.guard.observeFreshEvidence(snapshot, time.Now().UTC(), gates.WSHealthy)
	}
	result := a.risk.CheckOpen(ctx, request, snapshot, a.symbols)
	if a.guard != nil {
		a.guard.recordRiskDecision(binance.LeadAccountID, result)
	}
	if !result.Allowed {
		return result, ErrLeadOpenBlocked
	}
	return result, nil
}

// ExecuteOpen is the account-bound new-order submission path. A fresh Lead
// risk check runs before the Ownership claim and broker write.
func (a *LeadExecutionAdapter) ExecuteOpen(ctx context.Context, request futuresownership.OrderRequest, check RiskOpenOrder) (futuresownership.ExchangeOrder, error) {
	if a == nil {
		return futuresownership.ExchangeOrder{}, ErrLeadOpenBlocked
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if ctx.Err() != nil {
		return futuresownership.ExchangeOrder{}, ctx.Err()
	}
	if request.Intent != futuresownership.IntentOpen || request.Owner != futuresownership.OwnerAutoStrategy ||
		(request.OrderType != "MARKET" && request.OrderType != "LIMIT") {
		return futuresownership.ExchangeOrder{}, ErrLeadOpenBlocked
	}
	if check.AccountID != binance.LeadAccountID ||
		!strings.EqualFold(request.Symbol, check.Symbol) ||
		!strings.EqualFold(request.PositionSide, check.Side) ||
		!strings.EqualFold(request.OrderType, check.OrderType) ||
		!finiteEqual(request.Quantity, check.Quantity) ||
		(request.OrderType == "LIMIT" && !finiteEqual(request.Price, check.ReferencePrice)) ||
		(request.OrderType == "MARKET" && request.Price != 0) {
		return futuresownership.ExchangeOrder{}, ErrLeadOpenBlocked
	}
	if (request.PositionSide == "LONG" && request.Side != "BUY") ||
		(request.PositionSide == "SHORT" && request.Side != "SELL") {
		return futuresownership.ExchangeOrder{}, ErrLeadOpenBlocked
	}
	result, err := a.checkOpenLocked(ctx, check)
	if err != nil {
		// Retain the public safety-gate sentinel while exposing a stable
		// underlying category (notably ErrLeadPendingReconcile).
		return futuresownership.ExchangeOrder{}, fmt.Errorf("%w: %w", ErrLeadOpenBlocked, err)
	}
	if !result.Allowed {
		return futuresownership.ExchangeOrder{}, ErrLeadOpenBlocked
	}
	// The bound executor already provides persistent ClaimOrder and
	// unknown-submit -> lookup/reconcile; no raw Binance call here.
	// Use the same durable claim + Binance response + WS/Fill path as Main.
	// A normal acknowledged submission is NOT an account-wide recovery fault.
	// Only an uncertain submit or an incomplete ledger requires reconciliation.
	executed, execErr := a.ownership.Execute(ctx, request)
	if execErr != nil {
		a.pendingReconcile = true
		if a.guard != nil {
			a.guard.RecordFault(binance.LeadAccountID, FaultPendingReconcile)
		}
	}
	return executed, execErr
}
func finiteEqual(a, b float64) bool {
	return !math.IsNaN(a) && !math.IsNaN(b) && !math.IsInf(a, 0) && !math.IsInf(b, 0) && a == b
}

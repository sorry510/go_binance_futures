package leadaccount

import (
	"sync"
	"time"

	binance "go_binance_futures/feature/api/binance"
)

// LeadRuntimeState describes an observed condition, NEVER an authorization to
// use a live Binance broker. Stage 7 is a separate, still-locked gate.
type LeadRuntimeState string

const (
	LeadDisabled           LeadRuntimeState = "disabled"
	LeadPaused             LeadRuntimeState = "paused"
	LeadOpenReady          LeadRuntimeState = "open_ready"
	LeadRiskTripped        LeadRuntimeState = "risk_tripped"
	LeadReconcileRequired  LeadRuntimeState = "reconcile_required"
	LeadAccountUnavailable LeadRuntimeState = "account_unavailable"
)

type LeadFaultCode string

const (
	FaultDailyLoss            LeadFaultCode = "lead_daily_loss_limit"
	FaultDrawdown             LeadFaultCode = "lead_drawdown_limit"
	FaultIdentityInvalid      LeadFaultCode = "lead_identity_invalid"
	FaultPortfolioUnconfirmed LeadFaultCode = "portfolio_binding_not_confirmed"
	FaultCredentialsChanged   LeadFaultCode = "lead_credentials_changed"
	FaultWSUnavailable        LeadFaultCode = "lead_ws_unavailable"
	FaultSnapshotInvalid      LeadFaultCode = "lead_snapshot_unavailable"
	FaultRulesUnavailable     LeadFaultCode = "lead_rules_unavailable"
	FaultAPI429               LeadFaultCode = "lead_api_429"
	FaultAPI418               LeadFaultCode = "lead_api_418"
	FaultNetworkUnavailable   LeadFaultCode = "lead_api_unavailable"
	FaultUnknownSubmission    LeadFaultCode = "lead_unknown_submission"
	FaultUnknownAlgo          LeadFaultCode = "lead_algo_uncertain"
	FaultPendingReconcile     LeadFaultCode = "lead_pending_reconcile"
	FaultProtectionFailure    LeadFaultCode = "lead_protection_failure"
)

type LeadFaultSeverity string

const (
	SeverityInfo     LeadFaultSeverity = "info"
	SeverityWarning  LeadFaultSeverity = "warning"
	SeverityCritical LeadFaultSeverity = "critical"
)

type LeadFaultClass string

const (
	FaultSkip            LeadFaultClass = "skip"
	FaultTemporary       LeadFaultClass = "temporary"
	FaultTrip            LeadFaultClass = "trip"
	FaultReconcile       LeadFaultClass = "reconcile"
	FaultHardUnavailable LeadFaultClass = "hard_unavailable"
)

type LeadFaultEvent struct {
	Action     string            `json:"action"` // raised | resolved
	AccountID  binance.AccountID `json:"account_id"`
	Code       LeadFaultCode     `json:"code"`
	Severity   LeadFaultSeverity `json:"severity"`
	Class      LeadFaultClass    `json:"class"`
	Before     LeadRuntimeState  `json:"before"`
	After      LeadRuntimeState  `json:"after"`
	OccurredAt time.Time         `json:"occurred_at"`
}

// Notification is injectable and MUST receive only known-safe reason codes;
// never raw Binance errors, API keys, URLs, symbols or signed request payloads.
type LeadFaultNotifier func(LeadFaultEvent) error

type LeadStatusSnapshot struct {
	AccountID     binance.AccountID `json:"account_id"`
	State         LeadRuntimeState  `json:"state"`
	Reasons       []LeadFaultCode   `json:"reasons"`
	Paused        bool              `json:"paused"`
	Enabled       bool              `json:"enabled"`
	ObservedAt    time.Time         `json:"observed_at"`
	LastChangedAt time.Time         `json:"last_changed_at"`
	// These booleans represent modeled behavior, NOT live-trading permission.
	EvaluateExits  bool `json:"evaluate_exits"`
	ExitWriteSafe  bool `json:"exit_write_safe"`
	CanAttemptOpen bool `json:"can_attempt_open"`
}

type faultRecord struct {
	class            LeadFaultClass
	severity         LeadFaultSeverity
	at               time.Time
	lastNotified     time.Time
	notifiedSeverity LeadFaultSeverity
}
type LeadRuntimeGuard struct {
	mu            sync.RWMutex
	faults        map[LeadFaultCode]faultRecord
	paused        bool
	disabled      bool
	observedReady bool
	now           func() time.Time
	notify        LeadFaultNotifier
	lastChanged   time.Time
}

// Default disabled and without any unlock or real-trade enable method.
func NewLeadRuntimeGuard(notify LeadFaultNotifier) *LeadRuntimeGuard {
	return &LeadRuntimeGuard{
		faults:   make(map[LeadFaultCode]faultRecord),
		disabled: true,
		now:      time.Now,
		notify:   notify,
	}
}

const leadFaultNotificationInterval = 2 * time.Minute

func faultDefinition(code LeadFaultCode) (LeadFaultClass, LeadFaultSeverity, bool) {
	switch code {
	case FaultDailyLoss, FaultDrawdown:
		return FaultTrip, SeverityCritical, true
	case FaultUnknownSubmission, FaultUnknownAlgo, FaultPendingReconcile:
		return FaultReconcile, SeverityCritical, true
	case FaultIdentityInvalid, FaultCredentialsChanged, FaultPortfolioUnconfirmed:
		return FaultHardUnavailable, SeverityCritical, true
	case FaultProtectionFailure:
		return FaultHardUnavailable, SeverityCritical, true
	case FaultWSUnavailable, FaultSnapshotInvalid, FaultRulesUnavailable, FaultAPI429, FaultNetworkUnavailable:
		return FaultTemporary, SeverityWarning, true
	case FaultAPI418:
		return FaultTemporary, SeverityCritical, true
	default:
		return "", "", false
	}
}

// RecordFault is account-scoped and accepts known reason codes only.
// Invalid, Main-account or raw error text is ignored.
func (g *LeadRuntimeGuard) RecordFault(account binance.AccountID, code LeadFaultCode) bool {
	if g == nil || account != binance.LeadAccountID {
		return false
	}
	class, severity, ok := faultDefinition(code)
	if !ok {
		return false
	}
	g.mu.Lock()
	before := g.snapshotLocked().State
	now := g.now().UTC()
	record, exists := g.faults[code]
	record.class, record.severity = class, severity
	if !exists {
		record.at = now
		g.lastChanged = now
	}
	eventNeeded := !exists || record.lastNotified.IsZero() || record.notifiedSeverity != severity || now.Sub(record.lastNotified) >= leadFaultNotificationInterval
	if eventNeeded {
		record.lastNotified = now
		record.notifiedSeverity = severity
	}
	g.faults[code] = record
	after := g.snapshotLocked().State
	sink := g.notify
	g.mu.Unlock()
	if eventNeeded && sink != nil {
		// Notification failures must never clear the safety state.
		_ = sink(LeadFaultEvent{Action: "raised", AccountID: binance.LeadAccountID, Code: code, Class: class, Severity: severity, Before: before, After: after, OccurredAt: now})
	}
	return true
}
func (g *LeadRuntimeGuard) Pause() {
	if g == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.paused {
		g.paused = true
		g.lastChanged = g.now().UTC()
	}
}

// Stopping a pause is deliberately NOT exported. Future UI authorization and
// evidence checks must be reviewed separately; Stage 4-5 never resumes opens.

// observeBoundState is private. A modeled ready state from tests still cannot
// bypass Stage 7, the actual RiskController or the read-only broker.
func (g *LeadRuntimeGuard) observeBoundState(state riskRunState) {
	if g == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	oldDisabled := g.disabled
	g.disabled = !state.Enabled || !state.Stage7Authorized || !state.ReadOnlyVerified || !state.PortfolioBound
	g.observedReady = state.Enabled && state.AllowNewOpens && state.ReadOnlyVerified && state.PortfolioBound && state.WSHealthy && state.Stage7Authorized
	if oldDisabled != g.disabled {
		g.lastChanged = g.now().UTC()
	}
}

// observeFreshEvidence is only called after a Lead account-bound read with
// verified completeness. It can clear transient network/WS/snapshot faults
// but NEVER identity, risk-trip, pending-order or protection failures.
func (g *LeadRuntimeGuard) observeFreshEvidence(s RiskSnapshot, now time.Time, wsVerified bool) {
	if g == nil || s.AccountID != binance.LeadAccountID || !wsVerified ||
		!s.PositionsComplete || !s.OrdersComplete || !s.PNLComplete || !s.Reconciled || s.UnknownOrders ||
		!s.HedgeMode || !riskFiniteNonnegative(s.AvailableUSDT) || !riskFiniteNonnegative(s.EquityUSDT) ||
		s.CapturedAt.IsZero() || now.Sub(s.CapturedAt) > RiskSnapshotMaxAge ||
		s.CapturedAt.After(now.Add(2*time.Second)) ||
		!s.PNLDayStartUTC.Equal(now.UTC().Truncate(24*time.Hour)) ||
		s.PNLAsOf.Before(s.CapturedAt) || s.PNLAsOf.After(now.Add(2*time.Second)) {
		return
	}
	g.mu.Lock()
	before := g.snapshotLocked().State
	now = now.UTC()
	var resolved []LeadFaultEvent
	for _, code := range []LeadFaultCode{FaultWSUnavailable, FaultSnapshotInvalid, FaultRulesUnavailable, FaultAPI429, FaultAPI418, FaultNetworkUnavailable} {
		record, ok := g.faults[code]
		if !ok {
			continue
		}
		delete(g.faults, code)
		g.lastChanged = g.now().UTC()
		resolved = append(resolved, LeadFaultEvent{Action: "resolved", AccountID: binance.LeadAccountID, Code: code, Severity: record.severity, Class: record.class, Before: before, OccurredAt: now})
	}
	after := g.snapshotLocked().State
	sink := g.notify
	g.mu.Unlock()
	if sink != nil {
		for _, event := range resolved {
			event.After = after
			_ = sink(event)
		}
	}
}

func (g *LeadRuntimeGuard) statusLocked() LeadStatusSnapshot {
	snapshot := LeadStatusSnapshot{AccountID: binance.LeadAccountID, Paused: g.paused, Enabled: !g.disabled, ObservedAt: g.now().UTC(), LastChangedAt: g.lastChanged, EvaluateExits: true}
	if g.disabled {
		snapshot.State = LeadDisabled
	} else if g.paused {
		snapshot.State = LeadPaused
	} else if g.observedReady {
		snapshot.State = LeadOpenReady
	} else {
		snapshot.State = LeadDisabled
	}
	hard, reconcile, trip, temporary := false, false, false, false
	// Fixed order and fixed codes give stable serialized output.
	for _, code := range allFaultCodes() {
		fault, exists := g.faults[code]
		if !exists {
			continue
		}
		snapshot.Reasons = append(snapshot.Reasons, code)
		switch fault.class {
		case FaultHardUnavailable:
			hard = true
		case FaultReconcile:
			reconcile = true
		case FaultTrip:
			trip = true
		case FaultTemporary:
			temporary = true
		}
	}
	switch {
	case hard || temporary:
		snapshot.State = LeadAccountUnavailable
	case reconcile:
		snapshot.State = LeadReconcileRequired
	case trip:
		snapshot.State = LeadRiskTripped
	}
	// Exit intent continues to be evaluated even when writes are unsafe.
	snapshot.ExitWriteSafe = !hard && !temporary && !reconcile
	snapshot.CanAttemptOpen = snapshot.State == LeadOpenReady && !g.paused && !g.disabled && len(snapshot.Reasons) == 0
	return snapshot
}
func (g *LeadRuntimeGuard) snapshotLocked() LeadStatusSnapshot { return g.statusLocked() }
func (g *LeadRuntimeGuard) Status() LeadStatusSnapshot {
	if g == nil {
		return LeadStatusSnapshot{AccountID: binance.LeadAccountID, State: LeadDisabled, EvaluateExits: true}
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.snapshotLocked()
}
func allFaultCodes() []LeadFaultCode {
	return []LeadFaultCode{
		FaultIdentityInvalid, FaultCredentialsChanged, FaultPortfolioUnconfirmed, FaultProtectionFailure,
		FaultUnknownSubmission, FaultUnknownAlgo, FaultPendingReconcile,
		FaultDailyLoss, FaultDrawdown,
		FaultWSUnavailable, FaultSnapshotInvalid, FaultRulesUnavailable, FaultAPI429, FaultAPI418, FaultNetworkUnavailable,
	}
}

// Both decisions are _advisory_ preconditions. Additional account security,
// exchange filters, ownership and Stage 7 locked broker checks still apply.
func (g *LeadRuntimeGuard) permitsOpen() bool {
	if g == nil {
		return false
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	s := g.snapshotLocked()
	return !g.paused && !g.disabled && g.observedReady && len(s.Reasons) == 0
}
func (g *LeadRuntimeGuard) permitsExitWrite() bool {
	if g == nil {
		return false
	}
	return g.Status().ExitWriteSafe
}

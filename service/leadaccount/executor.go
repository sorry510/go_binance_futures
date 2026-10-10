package leadaccount

import (
	"context"
	"errors"
	"fmt"
	"github.com/adshao/go-binance/v2/futures"
	"math"
	"strings"
	"sync"

	binance "go_binance_futures/feature/api/binance"
	"go_binance_futures/service/futuresownership"
)

var (
	ErrLeadOpenBlocked      = errors.New("lead opening order blocked by safety gate")
	ErrLeadPendingReconcile = errors.New("lead account has unresolved orders or incomplete reconciliation")
)

// LeadExecutionAdapter is NOT a live trading entry point. Its public
// constructor uses a Stage 4-2 RiskController with no Enable method,
// a source which always reports Reconciled=false/UnknownOrders=true and
// no production callback that authorizes Stage 7. No actual order can be
// submitted via this production configuration.
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
	pendingReconcile bool // held until Stage 5 authoritative account resync
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
	return &LeadExecutionAdapter{account: client, ownership: owned, risk: NewRiskController(), source: source, rules: rules, symbols: symbols}, nil
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
	// No Stage 7 authorization exists in production: refuse BEFORE any
	// Lead SAPI, signed /fapi or public exchangeInfo fetch.
	a.risk.mu.RLock()
	gates := a.risk.state
	a.risk.mu.RUnlock()
	if !gates.Enabled || !gates.AllowNewOpens || !gates.ReadOnlyVerified ||
		!gates.PortfolioBound || !gates.WSHealthy || !gates.Stage7Authorized {
		return RiskDecision{BlockingReasons: []string{"lead_execution_not_authorized"}}, ErrLeadOpenBlocked
	}

	// Ignore untrusted validation flags, notional limits and leverage tier.
	// Recompute from public exchangeInfo and signed Lead portfolio reads.
	if a.rules == nil {
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
		return RiskDecision{BlockingReasons: []string{"lead_exchange_rules_unavailable"}}, ErrLeadOpenBlocked
	}
	request.BinanceMinNotionalUSDT = verified.MinNotionalUSDT
	request.BinanceMaxNotionalUSDT = verified.MaxNotionalUSDT
	request.BinanceMaxLeverage = verified.MaxLeverage
	request.PriceAndQuantityValidated = true
	request.MarginModeVerified = true
	snapshot, err := a.source.Snapshot(ctx)
	if err != nil {
		return RiskDecision{BlockingReasons: []string{"lead_risk_snapshot_unavailable"}}, ErrLeadOpenBlocked
	}
	if a.simulateEvidence != nil {
		a.simulateEvidence(&snapshot)
	}
	result := a.risk.CheckOpen(ctx, request, snapshot, a.symbols)
	if !result.Allowed {
		return result, ErrLeadOpenBlocked
	}
	return result, nil
}

// ExecuteOpen is the only order-submission path exposed by this adapter.
// A fresh Lead risk check happens while holding the per-adapter lock, *before*
// any Ownership claim or Broker write. This cannot authorize production orders
// without Stage 5 ownership / WS reconciliation and Stage 7 approval.
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
	// Reserve before submitting. Any ambiguous result keeps this instance
	// blocked, and there is no public unsafe reset in Stage 4-3. Successful
	// Binance acceptance also requires fresh Stage 5 account reconciliation.
	a.pendingReconcile = true
	executed, execErr := a.ownership.Execute(ctx, request)
	return executed, execErr
}
func finiteEqual(a, b float64) bool {
	return !math.IsNaN(a) && !math.IsNaN(b) && !math.IsInf(a, 0) && !math.IsInf(b, 0) && a == b
}

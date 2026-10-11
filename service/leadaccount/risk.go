package leadaccount

import (
	"context"
	"math"
	"strings"
	"sync"
	"time"

	binance "go_binance_futures/feature/api/binance"
)

// Stage 4-2 introduces an inert, Lead-only pre-open gate. There is no API
// route, scheduler, DB mutation or production function to authorize live
// orders. Stage 4-3 must connect a verified account-scoped evidence source.
const (
	RiskSnapshotMaxAge    = 10 * time.Second
	RiskMarketPriceBuffer = 0.005 // Conservative adverse movement for MARKET
	RiskMarginReserve     = 1.02  // Initial margin plus conservative fee cushion
)

type RiskLimits struct {
	MaxOrderNotionalUSDT     float64 `json:"max_order_notional_usdt"`
	MaxTotalNotionalUSDT     float64 `json:"max_total_notional_usdt"`
	MaxDailyRealizedLossUSDT float64 `json:"max_daily_realized_loss_usdt"`
	MaxDrawdownPct           float64 `json:"max_drawdown_pct,omitempty"` // 0 disables this optional rule
	MaxPositions             int     `json:"max_positions"`
	MaxLosingPositions       int     `json:"max_losing_positions"`
}
type RiskPosition struct {
	Symbol        string
	Side          string  // LONG / SHORT
	Quantity      float64 // Signed quantity may be negative for SHORT
	MarkPrice     float64
	UnrealizedPNL float64
	Leverage      int64
}
type RiskPendingOpen struct {
	Symbol                string
	Side                  string
	RemainingQuantity     float64 // Outstanding quantity only; filled quantity belongs in positions
	LimitOrReferencePrice float64
}
type RiskSnapshot struct {
	AccountID               binance.AccountID
	CapturedAt              time.Time
	PositionsComplete       bool
	OrdersComplete          bool
	PNLComplete             bool
	Reconciled              bool
	UnknownOrders           bool
	HedgeMode               bool
	WalletUSDT              float64
	AvailableUSDT           float64
	EquityUSDT              float64
	PeakEquityUSDT          float64
	PeakEquityVerified      bool
	DailyNetRealizedPNLUSDT float64 // Realized PNL - fees + funding, all from Lead
	PNLDayStartUTC          time.Time
	PNLAsOf                 time.Time
	Positions               []RiskPosition    // ALL Lead positions, including unmanaged
	PendingOpens            []RiskPendingOpen // ALL Lead opening orders, including unmanaged
}
type RiskOpenOrder struct {
	AccountID                 binance.AccountID
	Symbol                    string
	Side                      string // LONG / SHORT
	OrderType                 string // MARKET / LIMIT
	Quantity                  float64
	ReferencePrice            float64
	MarginType                string // ISOLATED / CROSSED; checked against Lead positionRisk
	Leverage                  int
	ConfiguredUSDT            float64
	BinanceMinNotionalUSDT    float64
	BinanceMaxNotionalUSDT    float64
	BinanceMaxLeverage        int
	PriceAndQuantityValidated bool // exchangeInfo filters and lot sizes checked by account-bound adapter
	MarginModeVerified        bool // never implicitly flip margin/hedge mode during risk check
}
type RiskDecision struct {
	Allowed                   bool      `json:"allowed"`
	BlockingReasons           []string  `json:"blocking_reasons"`
	RawOrderNotionalUSDT      float64   `json:"raw_order_notional_usdt"`
	OrderNotionalUSDT         float64   `json:"order_notional_usdt"`
	ExposureBeforeUSDT        float64   `json:"exposure_before_usdt"`
	ExposureAfterUSDT         float64   `json:"exposure_after_usdt"`
	InitialMarginRequiredUSDT float64   `json:"initial_margin_required_usdt"`
	PositionSlots             int       `json:"position_slots"`
	LosingPositions           int       `json:"losing_positions"`
	UTCTradingDayStart        time.Time `json:"utc_trading_day_start"`
}
type riskRunState struct {
	Enabled          bool
	AllowNewOpens    bool
	PortfolioBound   bool
	WSHealthy        bool
	Stage7Authorized bool
	ReadOnlyVerified bool
}

// RiskController is constructed disabled. It exposes Pause and SetLimits only;
// deliberately NO Enable/Resume method in Stage 4-2.
type RiskController struct {
	mu     sync.RWMutex
	limits RiskLimits
	state  riskRunState
	now    func() time.Time
}

func NewRiskController() *RiskController {
	return &RiskController{now: time.Now}
}
func (r *RiskController) Pause() {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.state.AllowNewOpens = false
}
func (r *RiskController) SetLimits(limits RiskLimits) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	// Even invalid/partially configured limits can be stored for display,
	// but CheckOpen treats them as a hard block.
	r.limits = limits
}
func (r *RiskController) CheckOpen(ctx context.Context, request RiskOpenOrder, snapshot RiskSnapshot, symbols *binance.LeadSymbolCache) RiskDecision {
	if r == nil {
		return RiskDecision{BlockingReasons: []string{"lead_risk_controller_unavailable"}}
	}
	r.mu.RLock()
	limits, state, now := r.limits, r.state, r.now
	r.mu.RUnlock()
	return evaluateLeadOpen(ctx, now(), limits, state, request, snapshot, symbols)
}
func riskFiniteNonnegative(x float64) bool { return !math.IsNaN(x) && !math.IsInf(x, 0) && x >= 0 }
func riskFinitePositive(x float64) bool    { return riskFiniteNonnegative(x) && x > 0 }
func riskSide(side string) bool            { return side == "LONG" || side == "SHORT" }
func riskSlot(symbol, side string) string {
	return strings.ToUpper(strings.TrimSpace(symbol)) + "|" + side
}
func riskFixedBlock(decision *RiskDecision, code string) {
	decision.BlockingReasons = append(decision.BlockingReasons, code)
}
func evaluateLeadOpen(ctx context.Context, now time.Time, limits RiskLimits, state riskRunState, request RiskOpenOrder, snap RiskSnapshot, symbols *binance.LeadSymbolCache) RiskDecision {
	d := RiskDecision{BlockingReasons: make([]string, 0, 16), UTCTradingDayStart: now.UTC().Truncate(24 * time.Hour)}
	block := func(code string) { riskFixedBlock(&d, code) }
	if err := ctx.Err(); err != nil {
		block("risk_context_unavailable")
		return d
	}
	if request.AccountID != binance.LeadAccountID || snap.AccountID != binance.LeadAccountID {
		block("account_not_lead")
		return d
	}
	if !state.Enabled {
		block("lead_disabled")
	}
	if !state.AllowNewOpens {
		block("new_opens_paused")
	}
	if !state.ReadOnlyVerified {
		block("lead_identity_not_verified")
	}
	if !state.PortfolioBound {
		block("portfolio_binding_not_confirmed")
	}
	if !state.WSHealthy {
		block("lead_ws_not_healthy")
	}
	if !state.Stage7Authorized {
		block("stage7_live_authorization_required")
	}
	sym := strings.ToUpper(strings.TrimSpace(request.Symbol))
	side := strings.ToUpper(strings.TrimSpace(request.Side))
	if sym == "" || !strings.HasSuffix(sym, "USDT") || !riskSide(side) {
		block("invalid_lead_symbol_or_side")
	}
	if symbols == nil || !symbols.Allows(sym) {
		block("lead_symbol_not_allowed")
	}
	if !snap.PositionsComplete || !snap.OrdersComplete || !snap.PNLComplete || !snap.Reconciled || snap.UnknownOrders {
		block("lead_snapshot_incomplete_or_uncertain")
	}
	if snap.CapturedAt.IsZero() || now.Sub(snap.CapturedAt) > RiskSnapshotMaxAge || snap.CapturedAt.After(now.Add(2*time.Second)) {
		block("lead_snapshot_stale")
	}
	if !snap.HedgeMode {
		block("lead_hedge_mode_required")
	}
	if !riskFiniteNonnegative(snap.WalletUSDT) || !riskFiniteNonnegative(snap.EquityUSDT) || !riskFiniteNonnegative(snap.AvailableUSDT) ||
		!riskFiniteNonnegative(snap.PeakEquityUSDT) || snap.AvailableUSDT > snap.EquityUSDT+1e-6 {
		block("lead_balance_invalid")
	}
	if !riskFiniteNonnegative(limits.MaxOrderNotionalUSDT) || limits.MaxOrderNotionalUSDT == 0 ||
		!riskFiniteNonnegative(limits.MaxTotalNotionalUSDT) || limits.MaxTotalNotionalUSDT == 0 ||
		!riskFiniteNonnegative(limits.MaxDailyRealizedLossUSDT) || limits.MaxDailyRealizedLossUSDT == 0 ||
		limits.MaxPositions <= 0 || limits.MaxLosingPositions <= 0 || !riskFiniteNonnegative(limits.MaxDrawdownPct) || limits.MaxDrawdownPct >= 100 {
		block("lead_risk_limits_not_configured")
	}
	if snap.PNLDayStartUTC.IsZero() || !snap.PNLDayStartUTC.Equal(d.UTCTradingDayStart) ||
		snap.PNLAsOf.Before(snap.CapturedAt) || snap.PNLAsOf.After(now.Add(2*time.Second)) ||
		math.IsNaN(snap.DailyNetRealizedPNLUSDT) || math.IsInf(snap.DailyNetRealizedPNLUSDT, 0) {
		block("lead_daily_pnl_unavailable")
	} else if riskFinitePositive(limits.MaxDailyRealizedLossUSDT) && snap.DailyNetRealizedPNLUSDT <= -limits.MaxDailyRealizedLossUSDT {
		block("lead_daily_loss_limit")
	}
	if riskFinitePositive(limits.MaxDrawdownPct) {
		if !snap.PeakEquityVerified || !riskFinitePositive(snap.PeakEquityUSDT) || snap.EquityUSDT > snap.PeakEquityUSDT+1e-6 {
			block("lead_drawdown_baseline_unverified")
		} else if 100*(snap.PeakEquityUSDT-snap.EquityUSDT)/snap.PeakEquityUSDT >= limits.MaxDrawdownPct {
			block("lead_drawdown_limit")
		}
	}
	if request.OrderType != "MARKET" && request.OrderType != "LIMIT" {
		block("lead_order_type_not_supported")
	}
	if !request.PriceAndQuantityValidated || !request.MarginModeVerified {
		block("lead_exchange_rules_unverified")
	}
	if request.Leverage <= 0 || request.BinanceMaxLeverage <= 0 || request.Leverage > request.BinanceMaxLeverage {
		block("lead_leverage_not_allowed")
	}
	if !riskFinitePositive(request.Quantity) || !riskFinitePositive(request.ReferencePrice) || !riskFinitePositive(request.ConfiguredUSDT) ||
		!riskFinitePositive(request.BinanceMinNotionalUSDT) || !riskFinitePositive(request.BinanceMaxNotionalUSDT) ||
		request.BinanceMaxNotionalUSDT < request.BinanceMinNotionalUSDT {
		block("lead_order_parameters_invalid")
	} else {
		d.RawOrderNotionalUSDT = request.Quantity * request.ReferencePrice
		price := request.ReferencePrice
		if request.OrderType == "MARKET" {
			price *= 1 + RiskMarketPriceBuffer
		}
		d.OrderNotionalUSDT = request.Quantity * price
		if !riskFinitePositive(d.OrderNotionalUSDT) || !riskFinitePositive(d.RawOrderNotionalUSDT) {
			block("lead_order_notional_invalid")
		} else {
			if riskFinitePositive(limits.MaxOrderNotionalUSDT) && d.OrderNotionalUSDT > limits.MaxOrderNotionalUSDT+1e-7 {
				block("lead_order_notional_limit")
			}
			if d.RawOrderNotionalUSDT < request.BinanceMinNotionalUSDT || d.OrderNotionalUSDT > request.BinanceMaxNotionalUSDT {
				block("lead_exchange_notional_limits")
			}
			if !riskFinitePositive(request.ConfiguredUSDT * float64(request.Leverage)) {
				block("lead_shared_coin_budget_invalid")
			}
			// The shared coin's Usdt*leverage is the planned order budget.
			// Do not inflate THAT budget by a hypothetical MARKET fill buffer:
			// otherwise every normal sizing formula (qty=Usdt*lev/price)
			// would be rejected. Account-level caps and margin remain worst-case.
			if d.RawOrderNotionalUSDT > request.ConfiguredUSDT*float64(request.Leverage)+1e-7 {
				block("lead_shared_coin_budget_exceeded")
			}
			if request.Leverage > 0 {
				d.InitialMarginRequiredUSDT = d.OrderNotionalUSDT / float64(request.Leverage) * RiskMarginReserve
				if !riskFinitePositive(d.InitialMarginRequiredUSDT) || d.InitialMarginRequiredUSDT > snap.AvailableUSDT {
					block("lead_available_margin_insufficient")
				}
			}
		}
	}
	slots := map[string]struct{}{}
	for _, p := range snap.Positions {
		// /fapi may contain rows for inactive zero-quantity contracts with
		// markPrice=0. Such rows have no risk exposure and must be skipped.
		if !riskFiniteNonnegative(math.Abs(p.Quantity)) {
			block("lead_position_snapshot_invalid")
			continue
		}
		if math.Abs(p.Quantity) < 1e-12 {
			continue
		}
		s := strings.ToUpper(strings.TrimSpace(p.Side))
		if p.Symbol == "" || !riskSide(s) || !riskFinitePositive(p.MarkPrice) ||
			math.IsNaN(p.UnrealizedPNL) || math.IsInf(p.UnrealizedPNL, 0) {
			block("lead_position_snapshot_invalid")
			continue
		}
		slots[riskSlot(p.Symbol, s)] = struct{}{}
		d.PositionSlots++
		d.ExposureBeforeUSDT += math.Abs(p.Quantity) * p.MarkPrice
		if p.Leverage <= 0 {
			block("lead_position_leverage_invalid")
		}
		if p.Leverage > 0 && p.UnrealizedPNL/(math.Abs(p.Quantity)*p.MarkPrice)*float64(p.Leverage)*100 < -0.1 {
			d.LosingPositions++
		}
	}
	for _, p := range snap.PendingOpens {
		s := strings.ToUpper(strings.TrimSpace(p.Side))
		if p.Symbol == "" || !riskSide(s) || !riskFinitePositive(p.RemainingQuantity) || !riskFinitePositive(p.LimitOrReferencePrice) {
			block("lead_open_order_snapshot_invalid")
			continue
		}
		slots[riskSlot(p.Symbol, s)] = struct{}{}
		d.PositionSlots++
		d.ExposureBeforeUSDT += p.RemainingQuantity * p.LimitOrReferencePrice
	}
	if _, exists := slots[riskSlot(sym, side)]; exists {
		block("lead_same_side_position_or_order_exists")
	}
	if limits.MaxPositions > 0 && d.PositionSlots >= limits.MaxPositions {
		block("lead_position_count_limit")
	}
	if limits.MaxLosingPositions > 0 && d.LosingPositions >= limits.MaxLosingPositions {
		block("lead_losing_position_count_limit")
	}
	d.ExposureAfterUSDT = d.ExposureBeforeUSDT + d.OrderNotionalUSDT
	if !riskFiniteNonnegative(d.ExposureAfterUSDT) {
		block("lead_exposure_invalid")
	} else if riskFinitePositive(limits.MaxTotalNotionalUSDT) && d.ExposureAfterUSDT > limits.MaxTotalNotionalUSDT+1e-7 {
		block("lead_total_notional_limit")
	}
	d.Allowed = len(d.BlockingReasons) == 0
	return d
}

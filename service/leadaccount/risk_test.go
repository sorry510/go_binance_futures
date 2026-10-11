package leadaccount

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	binance "go_binance_futures/feature/api/binance"
)

func riskSymbols(t *testing.T) *binance.LeadSymbolCache {
	t.Helper()
	a, err := binance.NewAccountClient(binance.LeadAccountID, futures.NewClient("mock-lead", "mock-secret"))
	if err != nil {
		t.Fatal(err)
	}
	cache, err := binance.NewLeadSymbolCache(a, func(context.Context) ([]binance.LeadTradingSymbol, error) {
		return []binance.LeadTradingSymbol{{Symbol: "BTCUSDT", QuoteAsset: "USDT"}, {Symbol: "ETHUSDT", QuoteAsset: "USDT"}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := cache.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	return cache
}
func validRiskFixture(now time.Time) (RiskLimits, riskRunState, RiskOpenOrder, RiskSnapshot) {
	limits := RiskLimits{MaxOrderNotionalUSDT: 150, MaxTotalNotionalUSDT: 500, MaxDailyRealizedLossUSDT: 10, MaxPositions: 5, MaxLosingPositions: 2}
	state := riskRunState{Enabled: true, AllowNewOpens: true, ReadOnlyVerified: true, PortfolioBound: true, WSHealthy: true, Stage7Authorized: true}
	order := RiskOpenOrder{AccountID: binance.LeadAccountID, Symbol: "BTCUSDT", Side: "LONG", OrderType: "MARKET", Quantity: 1, ReferencePrice: 100,
		Leverage: 4, ConfiguredUSDT: 27, BinanceMinNotionalUSDT: 5, BinanceMaxNotionalUSDT: 10000, BinanceMaxLeverage: 20,
		PriceAndQuantityValidated: true, MarginModeVerified: true}
	snap := RiskSnapshot{
		AccountID: binance.LeadAccountID, CapturedAt: now,
		PositionsComplete: true, OrdersComplete: true, PNLComplete: true, Reconciled: true, HedgeMode: true,
		WalletUSDT: 120, AvailableUSDT: 100, EquityUSDT: 120,
		DailyNetRealizedPNLUSDT: 0, PNLDayStartUTC: now.UTC().Truncate(24 * time.Hour), PNLAsOf: now,
	}
	return limits, state, order, snap
}
func riskReasonsContain(d RiskDecision, code string) bool {
	for _, r := range d.BlockingReasons {
		if r == code {
			return true
		}
	}
	return false
}
func TestStage42DefaultControllerAlwaysBlocksNewOpens(t *testing.T) {
	now := time.Now()
	limits, _, order, snap := validRiskFixture(now)
	controller := NewRiskController()
	if d := controller.CheckOpen(context.Background(), order, snap, riskSymbols(t)); d.Allowed || !riskReasonsContain(d, "lead_disabled") {
		t.Fatalf("default controller opened Lead: %+v", d)
	}
	controller.SetLimits(limits)
	if d := controller.CheckOpen(context.Background(), order, snap, riskSymbols(t)); d.Allowed || !riskReasonsContain(d, "stage7_live_authorization_required") {
		t.Fatalf("setting limits activated Lead: %+v", d)
	}
	controller.Pause()
	if d := controller.CheckOpen(context.Background(), order, snap, riskSymbols(t)); d.Allowed {
		t.Fatal("pausing must never activate trade")
	}
}
func TestStage42HypotheticalCompleteLeadEvidence(t *testing.T) {
	now := time.Now()
	limits, state, order, snap := validRiskFixture(now)
	decision := evaluateLeadOpen(context.Background(), now, limits, state, order, snap, riskSymbols(t))
	if !decision.Allowed || len(decision.BlockingReasons) != 0 {
		t.Fatalf("fixture unexpectedly blocked: %+v", decision)
	}
	if math.Abs(decision.OrderNotionalUSDT-100.5) > 1e-9 || math.Abs(decision.InitialMarginRequiredUSDT-(100.5/4*1.02)) > 1e-9 {
		t.Fatalf("wrong conservative notional or margin: %+v", decision)
	}
	if decision.UTCTradingDayStart != now.UTC().Truncate(24*time.Hour) {
		t.Fatalf("wrong UTC start: %+v", decision)
	}
}
func TestStage42EverySafetyBarrierFailClosed(t *testing.T) {
	now := time.Now()
	cache := riskSymbols(t)
	type tc struct {
		name   string
		change func(*RiskLimits, *riskRunState, *RiskOpenOrder, *RiskSnapshot)
		reason string
	}
	cases := []tc{
		{"main_request", func(_ *RiskLimits, _ *riskRunState, o *RiskOpenOrder, _ *RiskSnapshot) {
			o.AccountID = binance.MainAccountID
		}, "account_not_lead"},
		{"main_snapshot", func(_ *RiskLimits, _ *riskRunState, _ *RiskOpenOrder, s *RiskSnapshot) {
			s.AccountID = binance.MainAccountID
		}, "account_not_lead"},
		{"lead_paused", func(_ *RiskLimits, s *riskRunState, _ *RiskOpenOrder, _ *RiskSnapshot) { s.AllowNewOpens = false }, "new_opens_paused"},
		{"identity_unverified", func(_ *RiskLimits, s *riskRunState, _ *RiskOpenOrder, _ *RiskSnapshot) { s.ReadOnlyVerified = false }, "lead_identity_not_verified"},
		{"portfolio_unconfirmed", func(_ *RiskLimits, s *riskRunState, _ *RiskOpenOrder, _ *RiskSnapshot) { s.PortfolioBound = false }, "portfolio_binding_not_confirmed"},
		{"ws_off", func(_ *RiskLimits, s *riskRunState, _ *RiskOpenOrder, _ *RiskSnapshot) { s.WSHealthy = false }, "lead_ws_not_healthy"},
		{"no_authorization", func(_ *RiskLimits, s *riskRunState, _ *RiskOpenOrder, _ *RiskSnapshot) { s.Stage7Authorized = false }, "stage7_live_authorization_required"},
		{"missing_limits", func(l *RiskLimits, _ *riskRunState, _ *RiskOpenOrder, _ *RiskSnapshot) {
			l.MaxDailyRealizedLossUSDT = 0
		}, "lead_risk_limits_not_configured"},
		{"past_snapshot", func(_ *RiskLimits, _ *riskRunState, _ *RiskOpenOrder, s *RiskSnapshot) {
			s.CapturedAt = now.Add(-15 * time.Second)
		}, "lead_snapshot_stale"},
		{"future_snapshot", func(_ *RiskLimits, _ *riskRunState, _ *RiskOpenOrder, s *RiskSnapshot) {
			s.CapturedAt = now.Add(5 * time.Second)
		}, "lead_snapshot_stale"},
		{"incomplete_positions", func(_ *RiskLimits, _ *riskRunState, _ *RiskOpenOrder, s *RiskSnapshot) { s.PositionsComplete = false }, "lead_snapshot_incomplete_or_uncertain"},
		{"incomplete_orders", func(_ *RiskLimits, _ *riskRunState, _ *RiskOpenOrder, s *RiskSnapshot) { s.OrdersComplete = false }, "lead_snapshot_incomplete_or_uncertain"},
		{"incomplete_pnl", func(_ *RiskLimits, _ *riskRunState, _ *RiskOpenOrder, s *RiskSnapshot) { s.PNLComplete = false }, "lead_snapshot_incomplete_or_uncertain"},
		{"unreconciled", func(_ *RiskLimits, _ *riskRunState, _ *RiskOpenOrder, s *RiskSnapshot) { s.Reconciled = false }, "lead_snapshot_incomplete_or_uncertain"},
		{"unknown_order", func(_ *RiskLimits, _ *riskRunState, _ *RiskOpenOrder, s *RiskSnapshot) { s.UnknownOrders = true }, "lead_snapshot_incomplete_or_uncertain"},
		{"wrong_day", func(_ *RiskLimits, _ *riskRunState, _ *RiskOpenOrder, s *RiskSnapshot) {
			s.PNLDayStartUTC = now.UTC().Truncate(24 * time.Hour).Add(-24 * time.Hour)
		}, "lead_daily_pnl_unavailable"},
		{"incomplete_day_coverage", func(_ *RiskLimits, _ *riskRunState, _ *RiskOpenOrder, s *RiskSnapshot) {
			s.PNLAsOf = now.Add(-time.Minute)
		}, "lead_daily_pnl_unavailable"},
		{"loss_limit", func(_ *RiskLimits, _ *riskRunState, _ *RiskOpenOrder, s *RiskSnapshot) {
			s.DailyNetRealizedPNLUSDT = -10
		}, "lead_daily_loss_limit"},
		{"nan_pnl", func(_ *RiskLimits, _ *riskRunState, _ *RiskOpenOrder, s *RiskSnapshot) {
			s.DailyNetRealizedPNLUSDT = math.NaN()
		}, "lead_daily_pnl_unavailable"},
		{"wallet_bad", func(_ *RiskLimits, _ *riskRunState, _ *RiskOpenOrder, s *RiskSnapshot) { s.WalletUSDT = math.NaN() }, "lead_balance_invalid"},
		{"available_margin", func(_ *RiskLimits, _ *riskRunState, _ *RiskOpenOrder, s *RiskSnapshot) { s.AvailableUSDT = 25 }, "lead_available_margin_insufficient"},
		{"exceeds_order", func(l *RiskLimits, _ *riskRunState, _ *RiskOpenOrder, _ *RiskSnapshot) { l.MaxOrderNotionalUSDT = 100 }, "lead_order_notional_limit"},
		{"exceeds_total", func(l *RiskLimits, _ *riskRunState, _ *RiskOpenOrder, _ *RiskSnapshot) { l.MaxTotalNotionalUSDT = 100 }, "lead_total_notional_limit"},
		{"exceeds_coin", func(_ *RiskLimits, _ *riskRunState, o *RiskOpenOrder, _ *RiskSnapshot) { o.ConfiguredUSDT = 24 }, "lead_shared_coin_budget_exceeded"},
		{"exceeds_exchange", func(_ *RiskLimits, _ *riskRunState, o *RiskOpenOrder, _ *RiskSnapshot) {
			o.BinanceMaxNotionalUSDT = 100
		}, "lead_exchange_notional_limits"},
		{"precision_unknown", func(_ *RiskLimits, _ *riskRunState, o *RiskOpenOrder, _ *RiskSnapshot) {
			o.PriceAndQuantityValidated = false
		}, "lead_exchange_rules_unverified"},
		{"margin_mode_unknown", func(_ *RiskLimits, _ *riskRunState, o *RiskOpenOrder, _ *RiskSnapshot) { o.MarginModeVerified = false }, "lead_exchange_rules_unverified"},
		{"wrong_leverage", func(_ *RiskLimits, _ *riskRunState, o *RiskOpenOrder, _ *RiskSnapshot) { o.Leverage = 30 }, "lead_leverage_not_allowed"},
		{"empty_price", func(_ *RiskLimits, _ *riskRunState, o *RiskOpenOrder, _ *RiskSnapshot) { o.ReferencePrice = 0 }, "lead_order_parameters_invalid"},
		{"one_way_mode", func(_ *RiskLimits, _ *riskRunState, _ *RiskOpenOrder, s *RiskSnapshot) { s.HedgeMode = false }, "lead_hedge_mode_required"},
		{"unsupported_order", func(_ *RiskLimits, _ *riskRunState, o *RiskOpenOrder, _ *RiskSnapshot) { o.OrderType = "STOP" }, "lead_order_type_not_supported"},
		{"bad_side", func(_ *RiskLimits, _ *riskRunState, o *RiskOpenOrder, _ *RiskSnapshot) { o.Side = "BOTH" }, "invalid_lead_symbol_or_side"},
		{"position_conflict", func(_ *RiskLimits, _ *riskRunState, _ *RiskOpenOrder, s *RiskSnapshot) {
			s.Positions = []RiskPosition{{Symbol: "BTCUSDT", Side: "LONG", Quantity: 1, MarkPrice: 100, Leverage: 4}}
		}, "lead_same_side_position_or_order_exists"},
		{"pending_conflict", func(_ *RiskLimits, _ *riskRunState, _ *RiskOpenOrder, s *RiskSnapshot) {
			s.PendingOpens = []RiskPendingOpen{{Symbol: "BTCUSDT", Side: "LONG", RemainingQuantity: 1, LimitOrReferencePrice: 100}}
		}, "lead_same_side_position_or_order_exists"},
		{"max_count", func(l *RiskLimits, _ *riskRunState, _ *RiskOpenOrder, s *RiskSnapshot) {
			l.MaxPositions = 1
			s.Positions = []RiskPosition{{Symbol: "ETHUSDT", Side: "LONG", Quantity: 1, MarkPrice: 100, Leverage: 4}}
		}, "lead_position_count_limit"},
		{"loss_count", func(l *RiskLimits, _ *riskRunState, _ *RiskOpenOrder, s *RiskSnapshot) {
			l.MaxLosingPositions = 1
			s.Positions = []RiskPosition{{Symbol: "ETHUSDT", Side: "LONG", Quantity: 1, MarkPrice: 100, UnrealizedPNL: -1, Leverage: 4}}
		}, "lead_losing_position_count_limit"},
		{"drawdown_unproven", func(l *RiskLimits, _ *riskRunState, _ *RiskOpenOrder, s *RiskSnapshot) {
			l.MaxDrawdownPct = 10
			s.PeakEquityUSDT = 200
		}, "lead_drawdown_baseline_unverified"},
		{"drawdown_hit", func(l *RiskLimits, _ *riskRunState, _ *RiskOpenOrder, s *RiskSnapshot) {
			l.MaxDrawdownPct = 10
			s.PeakEquityUSDT = 140
			s.PeakEquityVerified = true
		}, "lead_drawdown_limit"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			l, state, order, snap := validRiskFixture(now)
			c.change(&l, &state, &order, &snap)
			decision := evaluateLeadOpen(context.Background(), now, l, state, order, snap, cache)
			if decision.Allowed || !riskReasonsContain(decision, c.reason) {
				t.Fatalf("expected %q: %+v", c.reason, decision)
			}
		})
	}
}
func TestStage42BothSidesAndAllExposureAccountWide(t *testing.T) {
	now := time.Now()
	limits, state, order, snap := validRiskFixture(now)
	snap.Positions = []RiskPosition{{Symbol: "BTCUSDT", Side: "SHORT", Quantity: -1, MarkPrice: 100, Leverage: 4}}
	snap.PendingOpens = []RiskPendingOpen{{Symbol: "ETHUSDT", Side: "LONG", RemainingQuantity: 2, LimitOrReferencePrice: 100}}
	limits.MaxTotalNotionalUSDT = 400
	d := evaluateLeadOpen(context.Background(), now, limits, state, order, snap, riskSymbols(t))
	if d.Allowed || !riskReasonsContain(d, "lead_total_notional_limit") || d.PositionSlots != 2 ||
		math.Abs(d.ExposureBeforeUSDT-300) > 1e-9 {
		t.Fatalf("pending/short exposure not counted: %+v", d)
	}
	limits.MaxTotalNotionalUSDT = 500
	d = evaluateLeadOpen(context.Background(), now, limits, state, order, snap, riskSymbols(t))
	if !d.Allowed {
		t.Fatalf("opposite side incorrectly conflicts: %+v", d)
	}
}
func TestStage42UTCTradingDayRollover(t *testing.T) {
	now := time.Date(2026, 10, 10, 0, 0, 1, 0, time.UTC)
	limits, state, order, snap := validRiskFixture(now)
	snap.PNLDayStartUTC = now.Add(-24 * time.Hour).UTC().Truncate(24 * time.Hour)
	d := evaluateLeadOpen(context.Background(), now, limits, state, order, snap, riskSymbols(t))
	if d.Allowed || !riskReasonsContain(d, "lead_daily_pnl_unavailable") {
		t.Fatalf("stale yesterday PNL accepted: %+v", d)
	}
	snap.PNLDayStartUTC = now.Truncate(24 * time.Hour)
	d = evaluateLeadOpen(context.Background(), now, limits, state, order, snap, riskSymbols(t))
	if !d.Allowed {
		t.Fatalf("current UTC day blocked: %+v", d)
	}
}
func TestStage42CanceledContextAndWhitelistInvalidation(t *testing.T) {
	now := time.Now()
	limits, state, order, snap := validRiskFixture(now)
	symbols := riskSymbols(t)
	symbols.Invalidate()
	d := evaluateLeadOpen(context.Background(), now, limits, state, order, snap, symbols)
	if d.Allowed || !riskReasonsContain(d, "lead_symbol_not_allowed") {
		t.Fatalf("expired symbol opened: %+v", d)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	d = evaluateLeadOpen(ctx, now, limits, state, order, snap, riskSymbols(t))
	if d.Allowed || !riskReasonsContain(d, "risk_context_unavailable") {
		t.Fatalf("canceled context accepted: %+v", d)
	}
}

func TestStage42NormalSharedCoinMarketBudgetRemainsUsable(t *testing.T) {
	now := time.Now()
	limits, state, order, snapshot := validRiskFixture(now)
	order.ConfiguredUSDT = 25 // qty=Usdt*leverage / reference price
	decision := evaluateLeadOpen(context.Background(), now, limits, state, order, snapshot, riskSymbols(t))
	if !decision.Allowed || decision.RawOrderNotionalUSDT != 100 ||
		decision.OrderNotionalUSDT <= decision.RawOrderNotionalUSDT {
		t.Fatalf("market buffer disabled ordinary shared coin budget: %+v", decision)
	}
}

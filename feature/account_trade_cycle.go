package feature

import (
	"context"
	"fmt"
	"time"

	"github.com/beego/beego/v2/core/logs"
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/feature/strategy"
	"go_binance_futures/feature/strategy/line"
	"go_binance_futures/models"
	"go_binance_futures/notify"
	"go_binance_futures/scanner"
	"go_binance_futures/service/leadaccount"
	"go_binance_futures/types"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/client/orm"
	"go_binance_futures/service/futuresownership"
)

// accountTradeRunner contains account-specific IO only. The strategy, order
// sequencing, TP/SL and risk gates live in one runAccountTradeCycle function.
// The Lead instance is deliberately mock-only until Stage 4/7 verification.
type accountTradeRunner struct {
	AccountID      binance.AccountID
	Config         *models.Config
	Mock           bool
	leadMockPermit stage3LeadMockPermit
	AllowNewOpens  bool
	LeadSymbols    *binance.LeadSymbolCache
	// Pending Lead orders only block their own symbols; Main is unchanged.
	leadPendingSymbols map[string]bool

	LoadSymbols    func() ([]*models.Symbols, error)
	SelectCoins    func(*models.Config, []*models.Symbols) ([]*models.Symbols, error)
	ReadPositions  func(context.Context) ([]types.FuturesPosition, error)
	ReadOpenOrders func(context.Context) ([]types.FuturesOrder, error)
	SyncPositions  func([]types.FuturesPosition) ([]ownedTradePosition, error)
	CancelExpired  func(int64)
	EvaluateEntry  func(*models.Symbols, []types.FuturesPosition) strategy.OpenResult
	Depth          func(context.Context, string, int) (float64, float64, error)
	EnsureConfig   func(context.Context, *models.Symbols) error
	// Mandatory for Lead, absent for Main. The Stage 4-3 account-bound
	// adapter must supply a fresh verified risk snapshot for every attempt.
	PreflightOpen func(context.Context, *models.Symbols, futures.PositionSideType, float64, float64) error
	SubmitOpen    func(*tradeCycleAccountSnapshot, string, float64, float64, futures.SideType, futures.PositionSideType, futures.OrderType, string) (*futures.CreateOrderResponse, error)
	SubmitClose   func(string, string, string, float64, futures.PositionSideType) (*futures.CreateOrderResponse, error)
	NotifyOpen    func(notify.FuturesOrderParams)
	NotifyClose   func(notify.FuturesOrderParams)
	EvaluateExit  func(strategy.CloseParams, float64, float64) tradeExitReason
	Sleep         func(time.Duration)
}

func mainTradeRunner(cfg *models.Config) *accountTradeRunner {
	entryEnabled := cfg != nil && cfg.FutureEnable == 1
	syncPositions := syncStrategyExitPositions
	if !entryEnabled {
		// With Main entries OFF, only auto_strategy work is authorized
		// by the local gate. Do not expand this exception to new_coin_rush
		// or funding_rate ownership managed by other independent features.
		syncPositions = syncAutoStrategyOwnership
	}
	return &accountTradeRunner{
		AccountID: binance.MainAccountID, Config: cfg, AllowNewOpens: entryEnabled,
		LoadSymbols: GetAllSymbols,
		SelectCoins: func(c *models.Config, coins []*models.Symbols) ([]*models.Symbols, error) {
			return selectConfiguredCoins(c, coins, scanner.SmartLocalV2ModeTrade), nil
		},
		ReadPositions:  GetTransformPositionsContext,
		ReadOpenOrders: getTransformOpenOrdersContext,
		SyncPositions:  syncPositions,
		CancelExpired:  cancelTimeoutOrder,
		EvaluateEntry: func(coin *models.Symbols, _ []types.FuturesPosition) strategy.OpenResult {
			return (line.TradeLineCustom{}).GetCanLongOrShort(strategy.OpenParams{Symbols: coin})
		},
		Depth: func(ctx context.Context, symbol string, limit int) (float64, float64, error) {
			return binance.GetDepthAvgPriceContext(ctx, symbol, limit)
		},
		EnsureConfig: func(ctx context.Context, coin *models.Symbols) error {
			// Only new Main entries pass this gate. Exits use the unchanged
			// ownership-based CLOSE path, even when account config RPCs fail.
			return ensureMainOpenConfiguration(ctx, coin, cfg)
		},
		SubmitOpen:   submitAutoStrategyOpen,
		SubmitClose:  submitManagedStrategyClose,
		NotifyOpen:   func(params notify.FuturesOrderParams) { pusher.SetModuleName("futures").FuturesOpenOrder(params) },
		NotifyClose:  func(params notify.FuturesOrderParams) { pusher.SetModuleName("futures").FuturesCloseOrder(params) },
		EvaluateExit: evaluateSharedTradeExit,
		Sleep:        time.Sleep,
	}
}

// Stage 3 forbids constructing a production Lead execution runner. Its
// mock-only permit interface has no production implementation: only _test.go
// files can create the test fixture. Stage 4 must introduce a separately
// verified account-bound execution adapter before scheduling real Lead IO.
type stage3LeadMockPermit interface{ stage3MockOnly() }

func (r *accountTradeRunner) validate() error {
	if r == nil || r.Config == nil {
		return fmt.Errorf("trade runner has no configuration")
	}
	if r.AccountID != binance.MainAccountID && r.AccountID != binance.LeadAccountID {
		return fmt.Errorf("invalid trade account %q", r.AccountID)
	}
	if r.AccountID == binance.LeadAccountID && r.PreflightOpen == nil {
		return fmt.Errorf("Lead requires an explicit risk preflight hook")
	}
	if r.AccountID == binance.LeadAccountID {
		if r.LeadSymbols == nil {
			return fmt.Errorf("Lead requires a bound whitelist")
		}
		if r.Mock {
			if r.leadMockPermit == nil {
				return fmt.Errorf("Lead mock requires test-only permit")
			}
		} else if r.AllowNewOpens {
			if err := leadaccount.LeadDurableOpenGate(); err != nil {
				return fmt.Errorf("Lead new entries disabled: %w", err)
			}
		}
	}
	if r.LoadSymbols == nil || r.SelectCoins == nil || r.ReadPositions == nil || r.ReadOpenOrders == nil ||
		r.SyncPositions == nil || r.CancelExpired == nil || r.EvaluateEntry == nil || r.Depth == nil ||
		r.EnsureConfig == nil || r.EvaluateExit == nil || r.SubmitOpen == nil || r.SubmitClose == nil ||
		r.NotifyOpen == nil || r.NotifyClose == nil || r.Sleep == nil {
		return fmt.Errorf("incomplete account-scoped trade adapter")
	}
	return nil
}

// mainAutoStrategyWorkNeeded reads the LOCAL Main managed ledger. With the
// global entry switch OFF, do not poll DAPI when there is no owned work.
// Checking both positions and nonterminal orders preserves recovery of a
// partially filled/uncertain opening without claiming external positions.
func mainAutoStrategyWorkNeeded(o orm.Ormer) (bool, error) {
	if o == nil {
		return false, fmt.Errorf("Main ownership ledger is unavailable")
	}
	var positions int64
	if err := o.Raw(
		"SELECT EXISTS(SELECT 1 FROM futures_managed_positions WHERE account_id=? AND owner=? AND managed_qty>0.000000000001 AND status IN ('active','suspended','reconcile_required'))",
		"main", futuresownership.OwnerAutoStrategy,
	).QueryRow(&positions); err != nil {
		return false, fmt.Errorf("read Main managed positions: %w", err)
	}
	if positions > 0 {
		return true, nil
	}
	var orders int64
	if err := o.Raw(
		"SELECT EXISTS(SELECT 1 FROM futures_managed_orders WHERE account_id=? AND owner=? AND status IN ('pending','submitted','partially_filled','reconcile_required'))",
		"main", futuresownership.OwnerAutoStrategy,
	).QueryRow(&orders); err != nil {
		return false, fmt.Errorf("read Main managed orders: %w", err)
	}
	return orders > 0, nil
}

func mainTradeCycleAllowed(enabled bool, hasManagedWork func() (bool, error)) (bool, error) {
	if enabled {
		return true, nil
	}
	return hasManagedWork()
}

// StartTrade retains the existing 2-second Main/Lead shared tick. Disabling
// Main stops new entries, not recovery/exit of Main-owned auto_strategy.
func StartTrade(systemConfig *models.Config) {
	if systemConfig == nil {
		return
	}
	enabled := systemConfig.FutureEnable == 1
	if !enabled {
		// Main loop may be completely idle while OFF, so publish the
		// actual global gate without waking Binance or querying symbols.
		mainEntryDiagnostics.begin(false)
	}
	runMain, err := mainTradeCycleAllowed(enabled, func() (bool, error) {
		return mainAutoStrategyWorkNeeded(orm.NewOrm())
	})
	if err != nil {
		// A missing/failed local ledger must never be interpreted as a
		// confirmed empty account. No speculative signed DAPI requests.
		logs.Warning("Main futures exit/recovery work gate unavailable: ", err)
	}
	if runMain && err == nil {
		if flagFutures == 0 {
			logs.Info("Main futures trade loop active (new entries enabled: ", enabled, ")")
			flagFutures = 1
		}
		runAccountTradeCycle(mainTradeRunner(systemConfig))
	} else if flagFutures == 1 {
		if err != nil {
			logs.Warning("Main futures trade loop paused: local ownership state could not be checked")
		} else {
			logs.Info("Main futures trade loop idle (no eligible owned work)")
		}
		flagFutures = 0
	}
	// Lead remains separate and is never granted Main credentials or opens.
	runLeadOnSharedTradeTick(systemConfig)
}

// tradeExitReason records the existing Main priority: strategy auto stop,
// then stop loss + CanOrderComplete, then target profit + CanOrderComplete.
// This function is shared by both account runners; no account API is used.
type tradeExitReason uint8

const (
	tradeExitHold tradeExitReason = iota
	tradeExitAuto
	tradeExitLoss
	tradeExitProfit
)

func evaluateSharedTradeExit(params strategy.CloseParams, profit, loss float64) tradeExitReason {
	evaluation := line.TradeLineCustom{}
	return evaluateTradeExitWithRules(params.NowProfit, profit, loss,
		func() bool { return evaluation.AutoStopOrder(params).Complete },
		func() bool { return evaluation.CanOrderComplete(params).Complete },
	)
}

// Extracted branch priority is golden-tested with pure, deterministic rules.
// Business effects stay in the shared runAccountTradeCycle switch.
func evaluateTradeExitWithRules(roi, profit, loss float64, autoStop, canClose func() bool) tradeExitReason {
	if autoStop() {
		return tradeExitAuto
	}
	if roi <= -loss && canClose() {
		return tradeExitLoss
	}
	if roi >= profit && canClose() {
		return tradeExitProfit
	}
	return tradeExitHold
}

// Main keeps the original path unchanged. Lead cannot proceed unless its own
// risk controller gives a positive pre-open decision for this exact intent.
func (r *accountTradeRunner) riskAllowsOpen(ctx context.Context, coin *models.Symbols, side futures.PositionSideType, qty, price float64) bool {
	if r.AccountID != binance.LeadAccountID {
		return true
	}
	if r.PreflightOpen == nil {
		logs.Warning("Lead risk preflight is missing; skip new open")
		return false
	}
	if err := r.PreflightOpen(ctx, coin, side, qty, price); err != nil {
		// Only category-like errors should be returned by the future Lead adapter,
		// never raw HTTP credentials or signed URLs.
		logs.Warning("Lead new open blocked by account risk preflight")
		return false
	}
	return true
}

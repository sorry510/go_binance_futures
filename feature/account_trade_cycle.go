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
	"go_binance_futures/types"

	"github.com/adshao/go-binance/v2/futures"
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
	RecordOpen    func(string, float64, string, string, int64, int64)
	RecordClose   func(types.FuturesPosition, float64, float64, string, int64, *models.Config)
	NotifyOpen    func(notify.FuturesOrderParams)
	NotifyClose   func(notify.FuturesOrderParams)
	EvaluateExit  func(strategy.CloseParams, float64, float64) tradeExitReason
	Sleep         func(time.Duration)
}

func mainTradeRunner(cfg *models.Config) *accountTradeRunner {
	return &accountTradeRunner{
		AccountID: binance.MainAccountID, Config: cfg, AllowNewOpens: true,
		LoadSymbols: GetAllSymbols,
		SelectCoins: func(c *models.Config, coins []*models.Symbols) ([]*models.Symbols, error) {
			return selectConfiguredCoins(c, coins, scanner.SmartLocalV2ModeTrade), nil
		},
		ReadPositions:  GetTransformPositionsContext,
		ReadOpenOrders: getTransformOpenOrdersContext,
		SyncPositions:  syncStrategyExitPositions,
		CancelExpired:  cancelTimeoutOrder,
		EvaluateEntry: func(coin *models.Symbols, _ []types.FuturesPosition) strategy.OpenResult {
			return (line.TradeLineCustom{}).GetCanLongOrShort(strategy.OpenParams{Symbols: coin})
		},
		Depth: func(ctx context.Context, symbol string, limit int) (float64, float64, error) {
			return binance.GetDepthAvgPriceContext(ctx, symbol, limit)
		},
		EnsureConfig: func(ctx context.Context, coin *models.Symbols) error {
			// Main compatibility: the legacy path deliberately ignores the Binance
			// config RPC error; Stage 4 Lead must fail closed instead.
			UpdateSymbolTradeInfoContext(ctx, coin)
			return nil
		},
		SubmitOpen:   submitAutoStrategyOpen,
		SubmitClose:  submitManagedStrategyClose,
		RecordOpen:   insertOpenOrder,
		RecordClose:  insertCloseOrder,
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
	if r.AccountID == binance.LeadAccountID && (!r.Mock || r.LeadSymbols == nil || r.leadMockPermit == nil) {
		return fmt.Errorf("Stage 3 lead runner requires a test-only mock permit and cached whitelist")
	}
	if r.LoadSymbols == nil || r.SelectCoins == nil || r.ReadPositions == nil || r.ReadOpenOrders == nil ||
		r.SyncPositions == nil || r.CancelExpired == nil || r.EvaluateEntry == nil || r.Depth == nil ||
		r.EnsureConfig == nil || r.EvaluateExit == nil || r.SubmitOpen == nil || r.SubmitClose == nil ||
		r.RecordOpen == nil || r.RecordClose == nil || r.NotifyOpen == nil || r.NotifyClose == nil || r.Sleep == nil {
		return fmt.Errorf("incomplete account-scoped trade adapter")
	}
	return nil
}

// StartTrade retains the only production scheduling path. No Lead client is
// constructed, no Lead SAPI request is made and no second WS is started.
func StartTrade(systemConfig *models.Config) {
	if systemConfig == nil {
		return
	}
	if systemConfig.FutureEnable == 1 {
		if flagFutures == 0 {
			// Legacy log semantics remain untouched.
			logs.Info("futures trade bot start")
			flagFutures = 1
		}
	} else {
		if flagFutures == 1 {
			logs.Info("futures trade bot stop")
			flagFutures = 0
		}
		return
	}
	runAccountTradeCycle(mainTradeRunner(systemConfig))
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

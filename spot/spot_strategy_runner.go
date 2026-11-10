package spot

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"go_binance_futures/feature/strategy/line"
	"go_binance_futures/models"
	"go_binance_futures/service/spotstrategy"
	spotapi "go_binance_futures/spot/api/binance"

	"github.com/adshao/go-binance/v2"
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
	"github.com/shopspring/decimal"
)

// Keep a bounded per-tick footprint: this is a personal, single-account
// worker, not a high-frequency cross-symbol scanning platform.
var spotStrategyRoundRobin atomic.Uint64

// RunRealSpotStrategies is a production worker, not a Mock business mode.
// All real REST writes require BOTH audited strategy mode and a separate
// explicit operator approval. Until S4's safety Gate, that approval is
// absent and the worker returns before reading the account or sending IO.
func RunRealSpotStrategies() {
	runSpotStrategySafely(runRealSpotStrategies)
}

// The scheduler runs on the process's main service lifecycle. A bad expr,
// indicator or unexpected SDK panic must NOT terminate futures/Spot
// monitoring and the rest of the user's personal trading service.
func runSpotStrategySafely(run func()) {
	defer func() {
		if recovered := recover(); recovered != nil {
			// Deliberately avoid rendering panic values or stack arguments:
			// SDK panics can contain private HTTP URLs and signed parameters.
			logs.Error("spot strategy worker panic isolated; type=%T", recovered)
		}
	}()
	run()
}

func runRealSpotStrategies() {
	if !spotstrategy.LiveOrdersApproved() {
		return
	}
	if err := spotstrategy.AuthorizeLiveMode(); err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	db, err := orm.GetDB("default")
	if err != nil {
		logs.Error("spot strategy DB unavailable:", err)
		return
	}
	e, err := spotstrategy.NewLiveEngine(db, spotapi.S3LiveClient{})
	if err != nil {
		logs.Warning("spot strategy real writer blocked:", err)
		return
	}
	if _, err = e.RecoverAndSettle(ctx, 10); err != nil {
		logs.Warning("spot strategy recovery requires reconciliation:", err)
	}
	var symbols []models.SpotSymbols
	if _, err = orm.NewOrm().QueryTable("spot_symbols").Filter("strategy_trade_enabled", 1).All(&symbols); err != nil {
		logs.Warning("spot enabled strategy symbols unavailable:", err)
		return
	}
	if len(symbols) == 0 {
		return
	}
	const maxPerTick = 3
	n := len(symbols)
	visit := n
	if visit > maxPerTick {
		visit = maxPerTick
	}
	start := int(spotStrategyRoundRobin.Add(uint64(visit))-uint64(visit)) % n
	for offset := 0; offset < visit; offset++ {
		if ctx.Err() != nil {
			return
		}
		symbol := symbols[(start+offset)%n]
		if err := executeSpotClosedSignal(ctx, e, symbol); err != nil {
			logs.Warning("spot strategy event rejected symbol=", symbol.Symbol, " reason=", err)
		}
	}
}

func lastClosedSpotKline(rows []*binance.Kline, now int64) (*binance.Kline, error) {
	for i := len(rows) - 1; i >= 0; i-- {
		bar := rows[i]
		if bar != nil && bar.CloseTime > 0 && bar.OpenTime > 0 && bar.CloseTime <= now {
			if now-bar.CloseTime > 3*60*1000 {
				return nil, errors.New("closed spot candle stale")
			}
			return bar, nil
		}
	}
	return nil, errors.New("no closed Spot kline")
}

func executeSpotClosedSignal(ctx context.Context, e *spotstrategy.Engine, symbol models.SpotSymbols) error {
	// The price used to size a new order must be a genuinely fresh websocket
	// tick, not an old DB snapshot or the candle's historical close.
	priceText, fresh := spotapi.GetFreshSpotTickerPrice(symbol.Symbol)
	if !fresh {
		return errors.New("Spot ticker stale")
	}
	price, err := decimal.NewFromString(priceText)
	if err != nil || !price.IsPositive() {
		return errors.New("invalid fresh Spot ticker")
	}
	rows, err := spotapi.GetKlineDataContext(ctx, symbol.Symbol, "1m", 3)
	if err != nil {
		return fmt.Errorf("closed Spot kline unavailable: %s", err)
	}
	bar, err := lastClosedSpotKline(rows, time.Now().UnixMilli())
	if err != nil {
		return err
	}
	env, err := line.InitClosedSpotParseEnv(ctx, symbol.Symbol, symbol.Technology, bar)
	if err != nil {
		return err
	}
	h, err := e.GetActiveHolding(ctx, symbol.Symbol)
	if err != nil && !errors.Is(err, spotstrategy.ErrUnmanagedOrder) {
		return err
	}
	hasHolding := err == nil && h.Status == "active"
	signal, err := spotstrategy.EvaluateClosedSignal(symbol.Symbol, symbol.Strategy, symbol.Technology, bar.CloseTime, env, hasHolding)
	if err != nil || signal.Direction == "" {
		return err
	}
	req := spotstrategy.OrderRequest{AccountID: spotstrategy.AccountMain, Owner: spotstrategy.OwnerAuto,
		Symbol: symbol.Symbol, Type: "MARKET", SignalKey: signal.SignalKey, EstimatePrice: priceText}
	if signal.Direction == "LONG" {
		spec, readErr := spotapi.S3LiveClient{}.Instrument(ctx, symbol.Symbol)
		if readErr != nil {
			return errors.New("Spot exchangeInfo unavailable")
		}
		if spec.QuoteAsset != symbol.Type || !spec.SpotAllowed || spec.Status != "TRADING" {
			return spotstrategy.ErrReconciliation
		}
		budget, parseErr := decimal.NewFromString(symbol.StrategyBudgetQuote)
		if parseErr != nil || !budget.IsPositive() {
			return errors.New("invalid Spot budget")
		}
		stepText := spec.LotStep
		if spec.MarketLotStep != "" && spec.MarketLotStep != "0" {
			stepText = spec.MarketLotStep
		}
		step, parseErr := decimal.NewFromString(stepText)
		if parseErr != nil || !step.IsPositive() {
			return spotstrategy.ErrReconciliation
		}
		// 1% conservative room for slippage/quote fees. S2 performs the
		// definitive Binance filter and available-balance recheck.
		qty := budget.Mul(decimal.RequireFromString("0.99")).Div(price).Div(step).Floor().Mul(step)
		if !qty.IsPositive() {
			return errors.New("Spot budget below minimum lot")
		}
		req.Side = "BUY"
		req.Quantity = qty.String()
	} else if signal.Direction == "CLOSE_LONG" {
		req.Side = "SELL" // quantity derived from verified Holding inside tx
	} else {
		return errors.New("unsupported Spot direction")
	}
	var order spotstrategy.LocalOrder
	if req.Side == "BUY" {
		order, err = e.Prepare(ctx, req)
	} else {
		order, err = e.PrepareExit(ctx, req)
	}
	if err != nil {
		return err
	}
	ack, err := e.Submit(ctx, order.ID, priceText, priceText)
	if err != nil {
		// The original client ID is persisted as UNKNOWN and can only be
		// recovered by signed GET /order, never by blind resubmission.
		return err
	}
	if ack.OrderID == "" {
		return spotstrategy.ErrReconciliation
	}
	reconciled, err := e.Reconcile(ctx, order.ID)
	if err != nil {
		return err
	}
	if reconciled.FilledQty != "0" && reconciled.Status != spotstrategy.StateReconcile {
		if _, err = e.ApplyVerifiedFills(ctx, order.ID); err != nil {
			return err
		}
	}
	return nil
}

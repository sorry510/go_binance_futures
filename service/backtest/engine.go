package backtest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	strategyservice "go_binance_futures/service/strategy"
	"go_binance_futures/utils"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

type Engine struct {
	ResolutionProviderFactory ResolutionProviderFactory
}

func (engine Engine) Run(ctx context.Context, dataset Dataset, strategy StrategySnapshot, config RunConfig) (Result, error) {
	return engine.RunWithProgress(ctx, dataset, strategy, config, nil)
}

func (engine Engine) RunWithProgress(ctx context.Context, dataset Dataset, strategy StrategySnapshot, config RunConfig, progress ProgressCallback) (Result, error) {
	return engine.RunWithResolution(ctx, dataset, strategy, config, ResolutionModeStandard, progress)
}

func (engine Engine) RunWithResolution(ctx context.Context, dataset Dataset, strategy StrategySnapshot, config RunConfig, resolutionMode string, progress ProgressCallback) (Result, error) {
	mode, err := NormalizeResolutionMode(resolutionMode)
	if err != nil {
		return Result{}, err
	}
	engineVersion, resolutionModel := ResolutionMetadata(mode)
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	config = NormalizeRunConfig(config)
	if dataset.DatasetSpecHash == "" {
		dataset.DatasetSpecHash = DatasetSpecHash(dataset)
	}
	if dataset.DataHash == "" {
		dataset.DataHash = DatasetDataHash(dataset)
	}
	if strategy.Version == "" {
		strategy.Version = strategyservice.StrategySnapshotHash(strategy.TechnologyJSON, strategy.StrategyJSON)
	}
	var rules []Rule
	if err := json.Unmarshal([]byte(strategy.StrategyJSON), &rules); err != nil {
		return Result{}, fmt.Errorf("decode strategy rules: %w", err)
	}
	environment, err := newHistoricalEnvironment(dataset, strategy.TechnologyJSON)
	if err != nil {
		return Result{}, err
	}
	execution := dataset.Bars[BarSeriesKey(dataset.Symbol, dataset.ExecutionInterval)]
	bars := make([]Bar, 0, len(execution))
	for _, bar := range execution {
		if bar.CloseTime >= dataset.StartTime && bar.CloseTime <= dataset.EndTime {
			bars = append(bars, bar)
		}
	}
	if len(bars) < 2 {
		return Result{}, fmt.Errorf("backtest requires at least two execution bars")
	}
	if progress != nil {
		progress(0, len(bars))
	}
	result := Result{DatasetID: dataset.DatasetID, DatasetSpecHash: dataset.DatasetSpecHash, DataHash: dataset.DataHash, StrategyVersion: strategy.Version, EngineVersion: engineVersion, MarketConditionModel: MarketConditionModel, ResolutionMode: mode, ResolutionModel: resolutionModel, Trades: []Trade{}, Events: []AuditEvent{}, Equity: []EquityPoint{}}
	adaptiveState := adaptiveRunState{factory: engine.ResolutionProviderFactory}
	defer adaptiveState.close()
	cash := config.InitialEquity
	peak := cash
	var position *Position
	var pending *PendingAction
	fundingIndex := 0
	eventSeq := 0
	equitySeq := 0
	compiled := map[string]*vm.Program{}
	addEvent := func(at int64, typ, action, side string, price, qty float64, data any) {
		eventSeq++
		raw, _ := json.Marshal(data)
		result.Events = append(result.Events, AuditEvent{Sequence: eventSeq, EventTime: at, Type: typ, Action: action, Side: side, Price: price, Quantity: qty, Data: raw})
	}
	advanceFunding := func(until int64, fallbackMark float64) {
		for fundingIndex < len(dataset.Funding) && dataset.Funding[fundingIndex].FundingTime <= until {
			fund := dataset.Funding[fundingIndex]
			fundingIndex++
			if position == nil || fund.FundingTime < position.EntryTime {
				continue
			}
			mark := fund.MarkPrice
			if mark <= 0 {
				mark = fallbackMark
			}
			notional := math.Abs(position.Quantity) * mark
			payment := -notional * fund.FundingRate
			if position.Side == "SHORT" {
				payment = -payment
			}
			position.FundingPnL += payment
			cash += payment
			addEvent(fund.FundingTime, "position", "funding", position.Side, mark, position.Quantity, map[string]any{"funding_rate": fund.FundingRate, "funding_pnl": payment})
		}
	}
	for index, bar := range bars {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		if pending != nil {
			action := *pending
			pending = nil
			if action.Action == "open" && position == nil {
				fill := applySlippage(bar.Open, action.Side, true, config.SlippageBps)
				notional := math.Max(cash, 0) * config.PositionSizePct * float64(config.Leverage)
				if fill > 0 && notional > 0 {
					qty := notional / fill
					fee := notional * config.FeeRate
					cash -= fee
					position = &Position{Side: action.Side, EntryTime: bar.OpenTime, EntryPrice: fill, Quantity: qty, OpenFee: fee, OpenStrategyName: action.StrategyName, OpenStrategyType: action.StrategyType, OpenStrategyHash: action.StrategyHash, MarketCondition: action.MarketCondition}
					addEvent(bar.OpenTime, "order", "open", action.Side, fill, qty, action)
					addEvent(bar.OpenTime, "fill", "open", action.Side, fill, qty, nil)
					addEvent(bar.OpenTime, "position", "opened", action.Side, fill, qty, nil)
				}
			} else if action.Action == "close" && position != nil {
				reason := action.ExitReason
				if reason == "" {
					reason = "strategy"
				}
				var trade Trade
				trade, cash = closePosition(*position, dataset.Symbol, bar.OpenTime, bar.Open, reason, action, config, cash, len(result.Trades)+1)
				result.Trades = append(result.Trades, trade)
				addEvent(bar.OpenTime, "order", "close", trade.Side, trade.ExitPrice, trade.Quantity, action)
				addEvent(bar.OpenTime, "fill", "close", trade.Side, trade.ExitPrice, trade.Quantity, nil)
				addEvent(bar.OpenTime, "position", "closed", trade.Side, trade.ExitPrice, trade.Quantity, map[string]any{"net_pnl": trade.NetPnL})
				position = nil
			}
		}
		if mode == ResolutionModeAdaptive && position != nil {
			decision, adaptiveErr := adaptiveState.evaluateROIClose(ctx, dataset, environment, bar, position, cash, config, rules, compiled, advanceFunding)
			if adaptiveErr != nil {
				return Result{}, adaptiveErr
			}
			if decision.Resolved && position != nil {
				action := adaptiveCloseAction(decision, position.Side)
				addEvent(decision.SignalTime, "intrabar_resolution", "close_gate", position.Side, decision.Price, position.Quantity, map[string]any{
					"resolution": decision.Resolution, "roi": decision.ROI, "gate_reason": decision.Reason,
					"evidence": decision.Evidence,
				})
				addEvent(decision.SignalTime, "signal", decision.Rule.Type, position.Side, decision.Price, position.Quantity, map[string]any{"strategy_name": decision.Rule.Name, "roi": decision.ROI, "gate_reason": decision.Reason, "resolution": decision.Resolution})
				trade, newCash := closePosition(*position, dataset.Symbol, decision.SignalTime, decision.Price, decision.Reason, action, config, cash, len(result.Trades)+1)
				trade.ExitResolution = decision.Resolution
				cash = newCash
				result.Trades = append(result.Trades, trade)
				addEvent(decision.SignalTime, "order", "close", trade.Side, trade.ExitPrice, trade.Quantity, action)
				addEvent(decision.SignalTime, "fill", "close", trade.Side, trade.ExitPrice, trade.Quantity, map[string]any{"resolution": decision.Resolution})
				addEvent(decision.SignalTime, "position", "closed", trade.Side, trade.ExitPrice, trade.Quantity, map[string]any{"net_pnl": trade.NetPnL, "resolution": decision.Resolution})
				position = nil
			}
		}
		advanceFunding(bar.CloseTime, bar.Close)
		env, condition, envErr := environment.Build(bar.CloseTime, position, cash, config)
		if envErr == nil {
			if position != nil {
				roi, _ := env["ROI"].(float64)
				gateReason := closeGateReason(roi, config)
				if gateReason != "" {
					rule, ok, evalErr := evaluateRules(rules, position.Side, env, compiled, true)
					if evalErr != nil {
						return Result{}, evalErr
					}
					if ok {
						pending = &PendingAction{Action: "close", Side: position.Side, StrategyName: rule.Name, StrategyType: rule.Type, StrategyHash: strategyservice.RuleHash(rule.Code), ExitReason: gateReason, SignalTime: bar.CloseTime, MarketCondition: condition}
						addEvent(bar.CloseTime, "signal", rule.Type, position.Side, bar.Close, position.Quantity, map[string]any{"strategy_name": rule.Name, "roi": roi, "gate_reason": gateReason})
					}
				}
			}
			if position == nil && pending == nil {
				rule, ok, evalErr := evaluateRules(rules, "", env, compiled, false)
				if evalErr != nil {
					return Result{}, evalErr
				}
				if ok {
					side := "LONG"
					if rule.Type == "short" {
						side = "SHORT"
					}
					pending = &PendingAction{Action: "open", Side: side, StrategyName: rule.Name, StrategyType: rule.Type, StrategyHash: strategyservice.RuleHash(rule.Code), SignalTime: bar.CloseTime, MarketCondition: condition}
					addEvent(bar.CloseTime, "signal", rule.Type, side, bar.Close, 0, map[string]any{"strategy_name": rule.Name, "market_condition": condition})
				}
			}
		} else if !errors.Is(envErr, ErrInsufficientHistoricalBars) {
			return Result{}, envErr
		}
		unrealized := unrealizedPnL(position, bar.Close)
		equity := cash + unrealized
		if equity > peak {
			peak = equity
		}
		dd := 0.0
		if peak > 0 {
			dd = (peak - equity) / peak * 100
		}
		equitySeq++
		side := ""
		if position != nil {
			side = position.Side
		}
		result.Equity = append(result.Equity, EquityPoint{Sequence: equitySeq, BarTime: bar.CloseTime, Equity: equity, Cash: cash, UnrealizedPnL: unrealized, DrawdownPct: dd, PositionSide: side})
		if index == len(bars)-1 && position != nil {
			action := PendingAction{Action: "close", Side: position.Side, StrategyName: "end_of_data", StrategyType: "end_of_data", StrategyHash: strategyservice.RuleHash("end_of_data")}
			trade, newCash := closePosition(*position, dataset.Symbol, bar.CloseTime, bar.Close, "end_of_data", action, config, cash, len(result.Trades)+1)
			cash = newCash
			result.Trades = append(result.Trades, trade)
			addEvent(bar.CloseTime, "order", "close", trade.Side, trade.ExitPrice, trade.Quantity, action)
			addEvent(bar.CloseTime, "fill", "close", trade.Side, trade.ExitPrice, trade.Quantity, nil)
			addEvent(bar.CloseTime, "position", "closed", trade.Side, trade.ExitPrice, trade.Quantity, map[string]any{"net_pnl": trade.NetPnL})
			position = nil
			last := &result.Equity[len(result.Equity)-1]
			last.Cash = cash
			last.Equity = cash
			last.UnrealizedPnL = 0
			last.PositionSide = ""
			if cash > peak {
				peak = cash
			}
			if peak > 0 {
				last.DrawdownPct = (peak - cash) / peak * 100
			}
		}
		if progress != nil {
			progress(index+1, len(bars))
		}
	}
	interval, _ := intervalDuration(dataset.ExecutionInterval, timeForMillis(dataset.StartTime))
	result.Metrics = calculateMetrics(result.Trades, result.Equity, config.InitialEquity, interval)
	if mode == ResolutionModeAdaptive {
		adaptiveState.finalize(&result)
	}
	return result, nil
}

func evaluateRules(rules []Rule, side string, env map[string]interface{}, cache map[string]*vm.Program, closing bool) (Rule, bool, error) {
	for index, rule := range rules {
		if !rule.Enable {
			continue
		}
		if closing {
			want := "close_long"
			if side == "SHORT" {
				want = "close_short"
			}
			if rule.Type != want {
				continue
			}
		} else if rule.Type != "long" && rule.Type != "short" {
			continue
		}
		key := fmt.Sprintf("%t|%d|%s", closing, index, rule.Code)
		program := cache[key]
		if program == nil {
			compiled, err := expr.Compile(rule.Code, expr.Env(env))
			if err != nil {
				return Rule{}, false, fmt.Errorf("compile strategy %s: %w", rule.Name, err)
			}
			program = compiled
			cache[key] = program
		}
		value, err := expr.Run(program, env)
		if err != nil {
			return Rule{}, false, fmt.Errorf("run strategy %s: %w", rule.Name, err)
		}
		passed, ok := value.(bool)
		if !ok {
			return Rule{}, false, fmt.Errorf("strategy %s did not return bool", rule.Name)
		}
		if passed {
			return rule, true, nil
		}
	}
	return Rule{}, false, nil
}
func closeGateReason(roi float64, config RunConfig) string {
	loss := config.StopLossPct
	if loss <= 0 {
		loss = utils.DisabledFuturesROIThreshold
	}
	profit := config.TakeProfitPct
	if profit <= 0 {
		profit = utils.DisabledFuturesROIThreshold
	}
	if roi <= -loss {
		return "stop_loss"
	}
	if roi >= profit {
		return "take_profit"
	}
	return ""
}

func applySlippage(price float64, side string, entering bool, bps float64) float64 {
	rate := bps / 10000
	if (side == "LONG" && entering) || (side == "SHORT" && !entering) {
		return price * (1 + rate)
	}
	return price * (1 - rate)
}
func closePosition(position Position, symbol string, at int64, basePrice float64, reason string, action PendingAction, config RunConfig, cash float64, sequence int) (Trade, float64) {
	return closePositionAtFill(position, symbol, at, applySlippage(basePrice, position.Side, false, config.SlippageBps), reason, action, config, cash, sequence)
}
func closePositionAtFill(position Position, symbol string, at int64, fill float64, reason string, action PendingAction, config RunConfig, cash float64, sequence int) (Trade, float64) {
	gross := unrealizedPnL(&position, fill)
	closeFee := math.Abs(position.Quantity) * fill * config.FeeRate
	cash += gross - closeFee
	fees := position.OpenFee + closeFee
	net := gross - fees + position.FundingPnL
	trade := Trade{Sequence: sequence, Symbol: symbol, Side: position.Side, EntryTime: position.EntryTime, ExitTime: at, EntryPrice: position.EntryPrice, ExitPrice: fill, Quantity: math.Abs(position.Quantity), GrossPnL: gross, Fees: fees, FundingPnL: position.FundingPnL, NetPnL: net, HoldingMs: at - position.EntryTime, ExitReason: reason, OpenStrategyName: position.OpenStrategyName, OpenStrategyType: position.OpenStrategyType, OpenStrategyHash: position.OpenStrategyHash, CloseStrategyName: action.StrategyName, CloseStrategyType: action.StrategyType, CloseStrategyHash: action.StrategyHash, MarketCondition: position.MarketCondition, EntryResolution: "1m", ExitResolution: "1m"}
	return trade, cash
}
func timeForMillis(value int64) time.Time { return time.UnixMilli(value).UTC() }

func SortResult(result *Result) {
	sort.Slice(result.Trades, func(i, j int) bool { return result.Trades[i].Sequence < result.Trades[j].Sequence })
	sort.Slice(result.Events, func(i, j int) bool { return result.Events[i].Sequence < result.Events[j].Sequence })
	sort.Slice(result.Equity, func(i, j int) bool { return result.Equity[i].Sequence < result.Equity[j].Sequence })
}

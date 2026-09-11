package backtest

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"go_binance_futures/service/historicalmarket"
	strategyservice "go_binance_futures/service/strategy"

	"github.com/expr-lang/expr/vm"
)

type ResolutionProviderFactory func() (historicalmarket.ResolutionProvider, error)

type adaptiveRunState struct {
	factory  ResolutionProviderFactory
	provider historicalmarket.ResolutionProvider
	evidence []historicalmarket.ResolutionEvidence
	stats    ResolutionStats
}

type adaptiveCloseDecision struct {
	Resolved   bool
	Rule       Rule
	Reason     string
	SignalTime int64
	Price      float64
	ROI        float64
	Condition  int
	Resolution string
	Evidence   []historicalmarket.ResolutionEvidence
}

func (state *adaptiveRunState) ensureProvider() (historicalmarket.ResolutionProvider, error) {
	if state.provider != nil {
		return state.provider, nil
	}
	if state.factory == nil {
		return nil, fmt.Errorf("adaptive backtest requires a resolution provider")
	}
	provider, err := state.factory()
	if err != nil {
		return nil, err
	}
	state.provider = provider
	return provider, nil
}

func (state *adaptiveRunState) close() {
	if state.provider != nil {
		_ = state.provider.Close()
	}
}

func (state *adaptiveRunState) finalize(result *Result) {
	if result == nil {
		return
	}
	result.DataHash = MergeResolutionEvidenceHash(result.DataHash, state.evidence...)
	if state.provider != nil {
		stats := state.provider.Stats()
		state.stats.SecondCacheHits = stats.SecondCacheHits
		state.stats.TradeCacheHits = stats.TradeCacheHits
		state.stats.ArchiveCacheHits = stats.ArchiveCacheHits
		state.stats.ArchiveDownloads = stats.ArchiveDownloads
		state.stats.DownloadBytes = stats.DownloadBytes
	}
	result.ResolutionStats = state.stats
}

func (state *adaptiveRunState) evaluateROIClose(
	ctx context.Context,
	dataset Dataset,
	environment *historicalEnvironment,
	bar Bar,
	position *Position,
	cash float64,
	config RunConfig,
	rules []Rule,
	compiled map[string]*vm.Program,
	advanceFunding func(int64, float64),
) (adaptiveCloseDecision, error) {
	candidate := DetectROICandidate(position, bar, config)
	if !candidate.Required {
		return adaptiveCloseDecision{}, nil
	}
	provider, err := state.ensureProvider()
	if err != nil {
		return adaptiveCloseDecision{}, err
	}
	seconds, evidence, err := provider.SecondBars(ctx, dataset.Market, dataset.Symbol, bar.OpenTime, bar.CloseTime)
	if err != nil {
		return adaptiveCloseDecision{}, err
	}
	state.stats.SecondDrilldownMinutes++
	state.evidence = append(state.evidence, evidence)
	sort.Slice(seconds, func(i, j int) bool { return seconds[i].OpenTime < seconds[j].OpenTime })
	var partial Bar
	for _, second := range seconds {
		if err := ctx.Err(); err != nil {
			return adaptiveCloseDecision{}, err
		}
		if second.OpenTime < bar.OpenTime || second.CloseTime > bar.CloseTime {
			continue
		}
		partialBeforeSecond := partial
		partial = appendSecondToPartialMinute(partial, bar, second)
		secondCandidate := DetectROICandidate(position, Bar{Low: second.Low, High: second.High}, config)
		fundingInsideSecond := hasFundingInRange(dataset.Funding, second.OpenTime, second.CloseTime)

		// If funding occurs inside this second, trade replay must run first so an
		// earlier crossing cannot observe a later funding payment. Without an
		// intra-second funding boundary we can evaluate the 1s close first and
		// avoid the much heavier trades archive when the second itself resolves.
		if secondCandidate.Required && fundingInsideSecond {
			tradeDecision, tradeErr := state.evaluateTradeROIClose(ctx, provider, dataset, environment, bar, partialBeforeSecond, second, position, cash, config, rules, compiled, advanceFunding, evidence)
			if tradeErr != nil {
				return adaptiveCloseDecision{}, tradeErr
			}
			if tradeDecision.Resolved {
				return tradeDecision, nil
			}
		}

		if advanceFunding != nil {
			advanceFunding(second.CloseTime, second.Close)
		}
		env, condition, err := environment.BuildIntrabar(second.CloseTime, partial, position, cash, config)
		if err != nil {
			if errors.Is(err, ErrInsufficientHistoricalBars) {
				continue
			}
			return adaptiveCloseDecision{}, err
		}
		roi, _ := env["ROI"].(float64)
		reason := closeGateReason(roi, config)
		if reason != "" {
			rule, ok, err := evaluateRules(rules, position.Side, env, compiled, true)
			if err != nil {
				return adaptiveCloseDecision{}, err
			}
			if ok {
				return adaptiveCloseDecision{
					Resolved: true, Rule: rule, Reason: reason, SignalTime: second.CloseTime,
					Price: second.Close, ROI: roi, Condition: condition, Resolution: "1s", Evidence: []historicalmarket.ResolutionEvidence{evidence},
				}, nil
			}
		}

		if secondCandidate.Required && !fundingInsideSecond {
			tradeDecision, tradeErr := state.evaluateTradeROIClose(ctx, provider, dataset, environment, bar, partialBeforeSecond, second, position, cash, config, rules, compiled, advanceFunding, evidence)
			if tradeErr != nil {
				return adaptiveCloseDecision{}, tradeErr
			}
			if tradeDecision.Resolved {
				return tradeDecision, nil
			}
		}
	}
	return adaptiveCloseDecision{}, nil
}

func hasFundingInRange(funding []Funding, start, end int64) bool {
	for _, item := range funding {
		if item.FundingTime < start {
			continue
		}
		if item.FundingTime > end {
			return false
		}
		return true
	}
	return false
}

func (state *adaptiveRunState) evaluateTradeROIClose(
	ctx context.Context,
	provider historicalmarket.ResolutionProvider,
	dataset Dataset,
	environment *historicalEnvironment,
	minute Bar,
	partialBeforeSecond Bar,
	second historicalmarket.Kline,
	position *Position,
	cash float64,
	config RunConfig,
	rules []Rule,
	compiled map[string]*vm.Program,
	advanceFunding func(int64, float64),
	secondEvidence historicalmarket.ResolutionEvidence,
) (adaptiveCloseDecision, error) {
	trades, evidence, err := provider.Trades(ctx, dataset.Market, dataset.Symbol, second.OpenTime, second.CloseTime)
	if err != nil {
		return adaptiveCloseDecision{}, err
	}
	state.stats.TradeDrilldownSeconds++
	state.evidence = append(state.evidence, evidence)
	sort.Slice(trades, func(i, j int) bool {
		if trades[i].TradeTime == trades[j].TradeTime {
			return trades[i].TradeID < trades[j].TradeID
		}
		return trades[i].TradeTime < trades[j].TradeTime
	})
	partial := partialBeforeSecond
	for _, trade := range trades {
		if err := ctx.Err(); err != nil {
			return adaptiveCloseDecision{}, err
		}
		if trade.TradeTime < second.OpenTime || trade.TradeTime > second.CloseTime || trade.Price <= 0 {
			continue
		}
		partial = appendTradeToPartialMinute(partial, minute, trade)
		if advanceFunding != nil {
			advanceFunding(trade.TradeTime, trade.Price)
		}
		env, condition, err := environment.BuildIntrabar(trade.TradeTime, partial, position, cash, config)
		if err != nil {
			if errors.Is(err, ErrInsufficientHistoricalBars) {
				continue
			}
			return adaptiveCloseDecision{}, err
		}
		roi, _ := env["ROI"].(float64)
		reason := closeGateReason(roi, config)
		if reason == "" {
			continue
		}
		rule, ok, err := evaluateRules(rules, position.Side, env, compiled, true)
		if err != nil {
			return adaptiveCloseDecision{}, err
		}
		if !ok {
			continue
		}
		return adaptiveCloseDecision{
			Resolved: true, Rule: rule, Reason: reason, SignalTime: trade.TradeTime,
			Price: trade.Price, ROI: roi, Condition: condition, Resolution: "trades",
			Evidence: []historicalmarket.ResolutionEvidence{secondEvidence, evidence},
		}, nil
	}
	return adaptiveCloseDecision{}, nil
}

func appendTradeToPartialMinute(partial Bar, minute Bar, trade historicalmarket.PublicDataTrade) Bar {
	if partial.OpenTime == 0 {
		partial = Bar{Symbol: minute.Symbol, Interval: minute.Interval, OpenTime: minute.OpenTime, CloseTime: trade.TradeTime, Open: trade.Price, High: trade.Price, Low: trade.Price, Close: trade.Price}
	} else {
		partial.CloseTime = trade.TradeTime
		if trade.Price > partial.High {
			partial.High = trade.Price
		}
		if trade.Price < partial.Low {
			partial.Low = trade.Price
		}
		partial.Close = trade.Price
	}
	partial.Volume += trade.Quantity
	partial.QuoteVolume += trade.QuoteQuantity
	partial.TradeCount++
	if !trade.IsBuyerMaker {
		partial.TakerBuyQuoteVolume += trade.QuoteQuantity
	}
	return partial
}

func appendSecondToPartialMinute(partial Bar, minute Bar, second historicalmarket.Kline) Bar {
	if partial.OpenTime == 0 {
		return Bar{
			Symbol: minute.Symbol, Interval: minute.Interval, OpenTime: minute.OpenTime, CloseTime: second.CloseTime,
			Open: second.Open, High: second.High, Low: second.Low, Close: second.Close,
			Volume: second.Volume, QuoteVolume: second.QuoteVolume, TradeCount: second.TradeCount,
			TakerBuyQuoteVolume: second.TakerBuyQuoteVolume,
		}
	}
	partial.CloseTime = second.CloseTime
	if second.High > partial.High {
		partial.High = second.High
	}
	if second.Low < partial.Low {
		partial.Low = second.Low
	}
	partial.Close = second.Close
	partial.Volume += second.Volume
	partial.QuoteVolume += second.QuoteVolume
	partial.TradeCount += second.TradeCount
	partial.TakerBuyQuoteVolume += second.TakerBuyQuoteVolume
	return partial
}

func adaptiveCloseAction(decision adaptiveCloseDecision, side string) PendingAction {
	return PendingAction{
		Action: "close", Side: side, StrategyName: decision.Rule.Name, StrategyType: decision.Rule.Type,
		StrategyHash: strategyservice.RuleHash(decision.Rule.Code), ExitReason: decision.Reason,
		SignalTime: decision.SignalTime, MarketCondition: decision.Condition,
	}
}

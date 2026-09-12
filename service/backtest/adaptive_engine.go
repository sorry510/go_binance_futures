package backtest

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

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
	activity func(string)
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

type closeRuleROIProfile struct {
	Eligible   bool
	Thresholds []float64
}

var (
	roiWordPattern      = regexp.MustCompile(`\bROI\b`)
	roiComparePattern   = regexp.MustCompile(`(?:\bROI\s*(?:<=|>=|==|!=|<|>)\s*(-?\d(?:_?\d)*(?:\.\d(?:_?\d)*)?(?:[eE][+-]?\d(?:_?\d)*)?)|(-?\d(?:_?\d)*(?:\.\d(?:_?\d)*)?(?:[eE][+-]?\d(?:_?\d)*)?)\s*(?:<=|>=|==|!=|<|>)\s*\bROI\b)`)
	bracketIndexPattern = regexp.MustCompile(`\[\s*([^\]]+)\s*\]`)
)

// analyzeCloseRuleROIProfile proves that, within one replay minute, the close
// rule can only change because ROI changes. Completed-bar references such as
// kline_1h.Close[1] are stable during the minute; current-bar references [0],
// time/current-price fields and position-derived fields force the conservative
// full intrabar rebuild path.
func analyzeCloseRuleROIProfile(rules []Rule, side string) closeRuleROIProfile {
	want := "close_long"
	if side == "SHORT" {
		want = "close_short"
	}
	profile := closeRuleROIProfile{}
	for _, rule := range rules {
		if !rule.Enable || rule.Type != want {
			continue
		}
		profile.Eligible = true
		code := rule.Code
		for _, token := range []string{
			"NowTime", "NowPrice", "NowSymbolPercentChange", "NowSymbolClose", "NowSymbolOpen", "NowSymbolLow", "NowSymbolHigh",
			"NetROI", "Fee", "NetProfit", "Position", "Positions", "IsAsc(", "IsDesc(", "KdjSimple(",
		} {
			if strings.Contains(code, token) {
				return closeRuleROIProfile{}
			}
		}
		for _, match := range bracketIndexPattern.FindAllStringSubmatch(code, -1) {
			index, err := strconv.Atoi(strings.TrimSpace(match[1]))
			if err != nil || index < 1 {
				return closeRuleROIProfile{}
			}
		}
		roiCount := len(roiWordPattern.FindAllStringIndex(code, -1))
		matchIndexes := roiComparePattern.FindAllStringSubmatchIndex(code, -1)
		if roiCount != len(matchIndexes) {
			return closeRuleROIProfile{}
		}
		for _, match := range matchIndexes {
			if !roiComparisonIsStandalone(code, match[0], match[1]) {
				return closeRuleROIProfile{}
			}
			valueStart, valueEnd := match[2], match[3]
			if valueStart < 0 {
				valueStart, valueEnd = match[4], match[5]
			}
			if valueStart < 0 || valueEnd <= valueStart {
				return closeRuleROIProfile{}
			}
			value := code[valueStart:valueEnd]
			threshold, err := strconv.ParseFloat(strings.ReplaceAll(value, "_", ""), 64)
			if err != nil {
				return closeRuleROIProfile{}
			}
			profile.Thresholds = append(profile.Thresholds, threshold)
		}
	}
	if !profile.Eligible {
		return closeRuleROIProfile{}
	}
	sort.Float64s(profile.Thresholds)
	return profile
}

func roiComparisonIsStandalone(code string, start, end int) bool {
	isUnsafeNeighbor := func(ch byte) bool {
		return (ch >= '0' && ch <= '9') || (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || ch == '_' || ch == '.' || strings.ContainsRune("+-*/%^", rune(ch))
	}
	for i := start - 1; i >= 0; i-- {
		if code[i] == ' ' || code[i] == '\t' || code[i] == '\n' || code[i] == '\r' {
			continue
		}
		if isUnsafeNeighbor(code[i]) {
			return false
		}
		break
	}
	for i := end; i < len(code); i++ {
		if code[i] == ' ' || code[i] == '\t' || code[i] == '\n' || code[i] == '\r' {
			continue
		}
		if isUnsafeNeighbor(code[i]) {
			return false
		}
		break
	}
	return true
}

func hasMarketConditionInRange(points []MarketConditionPoint, start, end int64) bool {
	index := sort.Search(len(points), func(i int) bool { return points[i].Time >= start })
	return index < len(points) && points[index].Time <= end
}

func ruleCanPassForROIRange(rules []Rule, side string, env map[string]interface{}, compiled map[string]*vm.Program, profile closeRuleROIProfile, minROI, maxROI float64, config RunConfig) (bool, error) {
	if !profile.Eligible || env == nil {
		return true, nil
	}
	if minROI > maxROI {
		minROI, maxROI = maxROI, minROI
	}
	points := []float64{minROI, maxROI}
	for _, threshold := range profile.Thresholds {
		if threshold >= minROI && threshold <= maxROI {
			points = append(points, threshold)
		}
	}
	if config.StopLossPct > 0 {
		threshold := -config.StopLossPct
		if threshold >= minROI && threshold <= maxROI {
			points = append(points, threshold)
		}
	}
	if config.TakeProfitPct > 0 {
		threshold := config.TakeProfitPct
		if threshold >= minROI && threshold <= maxROI {
			points = append(points, threshold)
		}
	}
	sort.Float64s(points)
	probes := make([]float64, 0, len(points)*2)
	for i, point := range points {
		if len(probes) == 0 || probes[len(probes)-1] != point {
			probes = append(probes, point)
		}
		if i+1 < len(points) && points[i+1] > point {
			probes = append(probes, point+(points[i+1]-point)/2)
		}
	}
	originalROI, hadROI := env["ROI"]
	defer func() {
		if hadROI {
			env["ROI"] = originalROI
		} else {
			delete(env, "ROI")
		}
	}()
	for _, roi := range probes {
		if closeGateReason(roi, config) == "" {
			continue
		}
		env["ROI"] = roi
		_, ok, err := evaluateRules(rules, side, env, compiled, true)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
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
	if setter, ok := provider.(interface{ SetActivity(func(string)) }); ok {
		setter.SetActivity(state.activity)
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

	// Generated strategies commonly use only ROI plus completed [1+] bars in
	// their close rules. In that case the non-ROI environment is constant for
	// the whole minute. Prove that property before doing any high-resolution I/O:
	// if the rule cannot be true anywhere inside the 1m ROI range, there is
	// nothing for 1s/trades to resolve.
	profile := analyzeCloseRuleROIProfile(rules, position.Side)
	var stableEnv map[string]interface{}
	stableCondition := 0
	if profile.Eligible && (!dataset.MarketConditionRequired || !hasMarketConditionInRange(dataset.MarketConditions, bar.OpenTime, bar.CloseTime)) {
		env, condition, err := environment.BuildIntrabar(bar.CloseTime, bar, position, cash, config)
		if err == nil {
			stableEnv, stableCondition = env, condition
			possible, evalErr := ruleCanPassForROIRange(rules, position.Side, stableEnv, compiled, profile, candidate.MinROI, candidate.MaxROI, config)
			if evalErr != nil {
				return adaptiveCloseDecision{}, evalErr
			}
			if !possible {
				return adaptiveCloseDecision{}, nil
			}
		} else {
			// The ROI-only proof is only a performance optimization. If the
			// synthetic full-minute environment cannot be built (for example a
			// malformed legacy bar), fall back to the conservative high-resolution
			// replay path instead of turning an optimization into a new hard-fail.
			stableEnv = nil
		}
	}

	provider, err := state.ensureProvider()
	if err != nil {
		return adaptiveCloseDecision{}, err
	}
	if state.activity != nil {
		state.activity("resolving_intrabar_data")
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

		tradeCouldResolve := secondCandidate.Required
		if tradeCouldResolve && stableEnv != nil {
			possible, evalErr := ruleCanPassForROIRange(rules, position.Side, stableEnv, compiled, profile, secondCandidate.MinROI, secondCandidate.MaxROI, config)
			if evalErr != nil {
				return adaptiveCloseDecision{}, evalErr
			}
			tradeCouldResolve = possible
		}

		// Funding inside this second must be replayed before a potentially earlier
		// trade close. If the ROI-only proof says the rule cannot pass anywhere in
		// this second, no trade replay is necessary and funding can safely advance
		// to the second close below.
		if tradeCouldResolve && fundingInsideSecond {
			tradeDecision, tradeErr := state.evaluateTradeROIClose(ctx, provider, dataset, environment, bar, partialBeforeSecond, second, position, cash, config, rules, compiled, advanceFunding, evidence, stableEnv, stableCondition, profile)
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
		closeROI := grossROI(position, second.Close, config.Leverage)
		reason := closeGateReason(closeROI, config)
		if reason != "" {
			condition := stableCondition
			var rule Rule
			var ok bool
			if stableEnv != nil {
				stableEnv["ROI"] = closeROI
				rule, ok, err = evaluateRules(rules, position.Side, stableEnv, compiled, true)
			} else {
				env, builtCondition, buildErr := environment.BuildIntrabar(second.CloseTime, partial, position, cash, config)
				if buildErr != nil {
					if errors.Is(buildErr, ErrInsufficientHistoricalBars) {
						continue
					}
					return adaptiveCloseDecision{}, buildErr
				}
				condition = builtCondition
				rule, ok, err = evaluateRules(rules, position.Side, env, compiled, true)
			}
			if err != nil {
				return adaptiveCloseDecision{}, err
			}
			if ok {
				return adaptiveCloseDecision{
					Resolved: true, Rule: rule, Reason: reason, SignalTime: second.CloseTime,
					Price: second.Close, ROI: closeROI, Condition: condition, Resolution: "1s", Evidence: []historicalmarket.ResolutionEvidence{evidence},
				}, nil
			}
		}

		if tradeCouldResolve && !fundingInsideSecond {
			tradeDecision, tradeErr := state.evaluateTradeROIClose(ctx, provider, dataset, environment, bar, partialBeforeSecond, second, position, cash, config, rules, compiled, advanceFunding, evidence, stableEnv, stableCondition, profile)
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
	stableEnv map[string]interface{},
	stableCondition int,
	profile closeRuleROIProfile,
) (adaptiveCloseDecision, error) {
	if state.activity != nil {
		state.activity("resolving_intrabar_data")
	}
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
		// A trade outside the 1s OHLC range means the two evidence sources
		// disagree. Do not silently continue with pruning assumptions derived
		// from that bar; strict Adaptive mode fails closed on inconsistent data.
		if trade.Price < second.Low || trade.Price > second.High {
			return adaptiveCloseDecision{}, fmt.Errorf("trade price %.12f outside 1s range [%.12f, %.12f] at %d", trade.Price, second.Low, second.High, trade.TradeTime)
		}
		partial = appendTradeToPartialMinute(partial, minute, trade)
		if advanceFunding != nil {
			advanceFunding(trade.TradeTime, trade.Price)
		}

		// ROI is available directly from the trade price. Do not rebuild the full
		// technology environment for trades where the outer ROI gate is closed.
		roi := grossROI(position, trade.Price, config.Leverage)
		reason := closeGateReason(roi, config)
		if reason == "" {
			continue
		}

		condition := stableCondition
		var rule Rule
		var ok bool
		if stableEnv != nil && profile.Eligible {
			stableEnv["ROI"] = roi
			rule, ok, err = evaluateRules(rules, position.Side, stableEnv, compiled, true)
		} else {
			env, builtCondition, buildErr := environment.BuildIntrabar(trade.TradeTime, partial, position, cash, config)
			if buildErr != nil {
				if errors.Is(buildErr, ErrInsufficientHistoricalBars) {
					continue
				}
				return adaptiveCloseDecision{}, buildErr
			}
			condition = builtCondition
			rule, ok, err = evaluateRules(rules, position.Side, env, compiled, true)
		}
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

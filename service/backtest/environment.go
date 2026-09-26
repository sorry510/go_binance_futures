package backtest

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"go_binance_futures/feature/strategy/line"
	"go_binance_futures/technology"
	markettypes "go_binance_futures/types"
	"go_binance_futures/utils"
)

var ErrInsufficientHistoricalBars = errors.New("insufficient historical bars")

type historicalEnvironment struct {
	dataset                  Dataset
	technology               technology.TechnologyConfig
	warmupBars               int
	overlays                 map[string]Bar
	minuteCloseOverlays      map[string]Bar
	minuteCloseLastOpenTime  int64
	minuteCloseCurrentBar    Bar
	minuteCloseCurrentAsOf   int64
	marketConditionIndex     int
	marketConditionAsOf      int64
	marketConditionReady     bool
	intervalMillis           map[string]int64
	standardOptimizations    bool
	tickerStatsCache         *rollingTickerStats
	klinePriceCache          map[string]cachedKLinePrice
	indicatorCache           map[string]cachedIndicatorValue
	indicatorCacheable       map[string]bool
	currentEMACache          map[string]cachedCurrentEMA
	currentRSICache          map[string]cachedCurrentRSI
	currentADXCache          map[string]cachedCurrentADX
	currentROCCache          map[string]cachedCurrentROC
	currentBOLLCache         map[string]cachedCurrentBOLL
	currentDonchianCache     map[string]cachedCurrentDonchian
	currentKCCache           map[string]cachedCurrentKC
	currentSupertrendCache   map[string]cachedCurrentSupertrend
	indicatorGroupsCache     []indicatorGroup
	indicatorPricesScratch   map[string]line.KLinePrice
	indicatorCachedScratch   map[string]interface{}
	indicatorTasksScratch    []indicatorTask
	indicatorResultsScratch  []indicatorTaskResult
	indicatorParallelScratch []int
}

func newHistoricalEnvironment(dataset Dataset, technologyJSON string) (*historicalEnvironment, error) {
	var config technology.TechnologyConfig
	if strings.TrimSpace(technologyJSON) != "" {
		if err := json.Unmarshal([]byte(technologyJSON), &config); err != nil {
			return nil, fmt.Errorf("decode technology: %w", err)
		}
		if err := line.ValidateTechnologyConfig(config); err != nil {
			return nil, err
		}
	}
	warmupBars := DefaultWarmupBars
	if required := line.TechnologyKlineLimit(config); required > warmupBars {
		warmupBars = required
	}
	intervalMillis := make(map[string]int64, len(dataset.Intervals))
	for _, interval := range dataset.Intervals {
		if interval == "1w" || interval == "1M" {
			continue
		}
		if duration, err := intervalDuration(interval, time.Unix(0, 0).UTC()); err == nil {
			intervalMillis[interval] = duration.Milliseconds()
		}
	}
	return &historicalEnvironment{dataset: dataset, technology: config, warmupBars: warmupBars, intervalMillis: intervalMillis}, nil
}

func (builder *historicalEnvironment) strategyWarmupReadyAt() (int64, bool, error) {
	if builder.dataset.WarmupStartTime <= 0 || builder.dataset.WarmupStartTime >= builder.dataset.StartTime {
		return builder.dataset.StartTime, true, nil
	}
	readyAt := builder.dataset.StartTime
	seen := make(map[string]bool)
	for _, group := range builder.indicatorGroups() {
		for _, item := range group.items {
			if !item.Enable || seen[item.KlineInterval] {
				continue
			}
			seen[item.KlineInterval] = true
			expectedStart, err := subtractBars(builder.dataset.StartTime, item.KlineInterval, builder.warmupBars)
			if err != nil {
				return 0, false, err
			}
			if builder.dataset.WarmupStartTime > expectedStart {
				continue
			}
			bars := builder.dataset.Bars[BarSeriesKey(builder.dataset.Symbol, item.KlineInterval)]
			if len(bars) == 0 {
				return 0, false, nil
			}
			if bars[0].OpenTime <= expectedStart {
				continue
			}
			if len(bars) < builder.warmupBars {
				return 0, false, nil
			}
			if candidate := bars[builder.warmupBars-1].CloseTime; candidate > readyAt {
				readyAt = candidate
			}
		}
	}
	return readyAt, true, nil
}

func (builder *historicalEnvironment) Build(asOf int64, position *Position, cash float64, config RunConfig) (map[string]interface{}, int, error) {
	return builder.build(asOf, position, cash, config)
}

// BuildIntrabar mirrors the live environment's use of the current open K-line,
// but only with price/volume information observed through the supplied partial
// minute. Higher-interval current bars are reconstructed from completed 1m
// bars plus this partial minute, so no future part of the current bar leaks in.
func (builder *historicalEnvironment) BuildIntrabar(asOf int64, partialMinute Bar, position *Position, cash float64, config RunConfig) (map[string]interface{}, int, error) {
	overlays, err := buildIntrabarOverlays(builder.dataset, partialMinute, asOf)
	if err != nil {
		return nil, 0, err
	}
	clone := *builder
	clone.overlays = overlays
	return clone.build(asOf, position, cash, config)
}

// BuildMinuteClose mirrors the live environment at a completed replay minute without
// rescanning the whole current higher-interval window on every minute. The first
// minute (or a discontinuity) rebuilds overlays from observed 1m data; subsequent
// minutes update the forming higher-interval bars incrementally.
func (builder *historicalEnvironment) BuildMinuteClose(asOf int64, minute Bar, position *Position, cash float64, config RunConfig) (map[string]interface{}, int, error) {
	if minute.Interval != "1m" || minute.CloseTime != asOf {
		return nil, 0, fmt.Errorf("invalid minute-close replay bar at %d", asOf)
	}
	sequential := builder.minuteCloseOverlays != nil && minute.OpenTime == builder.minuteCloseLastOpenTime+time.Minute.Milliseconds()
	if !sequential {
		overlays, err := buildIntrabarOverlays(builder.dataset, minute, asOf)
		if err != nil {
			return nil, 0, err
		}
		builder.minuteCloseOverlays = overlays
	} else {
		if err := builder.advanceMinuteCloseOverlays(minute, asOf); err != nil {
			return nil, 0, err
		}
	}
	builder.minuteCloseLastOpenTime = minute.OpenTime
	if builder.standardOptimizations {
		previous := builder.overlays
		builder.overlays = builder.minuteCloseOverlays
		builder.minuteCloseCurrentBar = minute
		builder.minuteCloseCurrentAsOf = asOf
		builder.prepareRollingTickerStats(minute, asOf)
		env, condition, err := builder.build(asOf, position, cash, config)
		builder.overlays = previous
		return env, condition, err
	}
	clone := *builder
	clone.overlays = builder.minuteCloseOverlays
	return clone.build(asOf, position, cash, config)
}

func (builder *historicalEnvironment) advanceMinuteCloseOverlays(minute Bar, asOf int64) error {
	builder.minuteCloseOverlays[BarSeriesKey(builder.dataset.Symbol, "1m")] = minute
	for _, interval := range builder.dataset.Intervals {
		if interval == "1m" {
			continue
		}
		windowStart, err := builder.cachedIntervalWindowStart(interval, asOf)
		if err != nil {
			return err
		}
		key := BarSeriesKey(builder.dataset.Symbol, interval)
		current, ok := builder.minuteCloseOverlays[key]
		if !ok || current.OpenTime != windowStart {
			if minute.OpenTime != windowStart {
				overlays, rebuildErr := buildIntrabarOverlays(builder.dataset, minute, asOf)
				if rebuildErr != nil {
					return rebuildErr
				}
				builder.minuteCloseOverlays = overlays
				return nil
			}
			builder.minuteCloseOverlays[key] = Bar{
				Symbol: builder.dataset.Symbol, Interval: interval,
				OpenTime: windowStart, CloseTime: asOf,
				Open: minute.Open, High: minute.High, Low: minute.Low, Close: minute.Close,
				Volume: minute.Volume, QuoteVolume: minute.QuoteVolume, TradeCount: minute.TradeCount,
				TakerBuyQuoteVolume: minute.TakerBuyQuoteVolume,
			}
			continue
		}
		if minute.High > current.High {
			current.High = minute.High
		}
		if minute.Low < current.Low {
			current.Low = minute.Low
		}
		current.CloseTime = asOf
		current.Close = minute.Close
		current.Volume += minute.Volume
		current.QuoteVolume += minute.QuoteVolume
		current.TradeCount += minute.TradeCount
		current.TakerBuyQuoteVolume += minute.TakerBuyQuoteVolume
		builder.minuteCloseOverlays[key] = current
	}
	return nil
}

func (builder *historicalEnvironment) build(asOf int64, position *Position, cash float64, config RunConfig) (map[string]interface{}, int, error) {
	var current Bar
	if builder.standardOptimizations && builder.minuteCloseCurrentAsOf == asOf {
		current = builder.minuteCloseCurrentBar
	} else {
		currentBars := builder.series(builder.dataset.Symbol, builder.dataset.ExecutionInterval, asOf, 1)
		if len(currentBars) == 0 {
			return nil, 0, fmt.Errorf("%w: no execution bars visible at %d", ErrInsufficientHistoricalBars, asOf)
		}
		current = currentBars[0]
	}
	var targetStats map[string]interface{}
	if builder.standardOptimizations && builder.tickerStatsCache != nil && builder.tickerStatsCache.asOf == asOf {
		targetStats = builder.tickerStatsCache.values()
	} else {
		targetStats = builder.tickerStats(builder.dataset.Symbol, asOf)
	}
	env := map[string]interface{}{
		"SystemStartTime": builder.dataset.StartTime, "NowTime": asOf, "NowPrice": current.Close,
		"NowSymbolPercentChange": targetStats["PercentChange"], "NowSymbolClose": targetStats["Close"], "NowSymbolOpen": targetStats["Open"], "NowSymbolLow": targetStats["Low"], "NowSymbolHigh": targetStats["High"],
		"FundingRate":      builder.fundingRateData(asOf, 16),
		"OpenStrategyHash": "",
		"KdjSimple":        line.KdjSimple, "IsAsc": utils.IsAsc, "IsDesc": utils.IsDesc,
	}
	condition := 0
	if builder.dataset.MarketConditionRequired {
		if builder.standardOptimizations {
			condition = builder.marketConditionAtSequential(asOf)
		} else {
			condition = builder.marketConditionAt(asOf)
		}
		if condition == 0 {
			return nil, 0, fmt.Errorf("%w: no MarketCondition visible at %d", ErrInsufficientHistoricalBars, asOf)
		}
		env["MarketCondition"] = strconv.Itoa(condition)
	}
	if err := builder.addIndicators(env, asOf); err != nil {
		return nil, condition, err
	}
	if position == nil {
		env["Positions"] = []markettypes.FuturesPosition{}
		return env, condition, nil
	}
	gross := unrealizedPnL(position, current.Close)
	roi := grossROI(position, current.Close, config.Leverage)
	projectedCloseFee := math.Abs(position.Quantity) * current.Close * config.FeeRate
	net := gross - position.OpenFee - projectedCloseFee + position.FundingPnL
	notional := math.Abs(position.Quantity) * current.Close
	netROI := 0.0
	if notional > 0 {
		netROI = net / notional * float64(config.Leverage) * 100
	}
	p := markettypes.FuturesPositionCode{Symbol: builder.dataset.Symbol, Side: position.Side, Amount: position.Quantity, Leverage: int64(config.Leverage), EntryPrice: position.EntryPrice, MarkPrice: current.Close, UnrealizedProfit: gross, Mock: true, CreateTime: position.EntryTime, SourceType: "backtest"}
	env["OpenStrategyHash"] = position.OpenStrategyHash
	env["ROI"], env["NetROI"], env["Fee"], env["NetProfit"], env["Position"] = roi, netROI, position.OpenFee+projectedCloseFee, net, p
	env["Positions"] = []markettypes.FuturesPosition{{Symbol: builder.dataset.Symbol, Side: position.Side, Amount: strconv.FormatFloat(position.Quantity, 'f', -1, 64), Leverage: int64(config.Leverage), EntryPrice: strconv.FormatFloat(position.EntryPrice, 'f', -1, 64), MarkPrice: strconv.FormatFloat(current.Close, 'f', -1, 64), UnrealizedProfit: strconv.FormatFloat(gross, 'f', -1, 64), SourceType: "backtest", CreateTime: position.EntryTime}}
	return env, condition, nil
}

func (builder *historicalEnvironment) fundingRateData(asOf int64, limit int) line.FundingRateData {
	if limit <= 0 {
		return line.FundingRateData{}
	}
	rows := builder.dataset.Funding
	end := sort.Search(len(rows), func(i int) bool { return rows[i].FundingTime > asOf })
	start := end - limit
	if start < 0 {
		start = 0
	}
	result := line.FundingRateData{
		Data: make([]float64, 0, end-start),
		Time: make([]int64, 0, end-start),
	}
	for i := end - 1; i >= start; i-- {
		result.Data = append(result.Data, rows[i].FundingRate)
		result.Time = append(result.Time, rows[i].FundingTime)
	}
	return result
}

func (builder *historicalEnvironment) marketConditionAtSequential(asOf int64) int {
	points := builder.dataset.MarketConditions
	if len(points) == 0 {
		builder.marketConditionReady = true
		builder.marketConditionIndex = -1
		builder.marketConditionAsOf = asOf
		return 0
	}
	if !builder.marketConditionReady || asOf < builder.marketConditionAsOf {
		builder.marketConditionIndex = sort.Search(len(points), func(i int) bool { return points[i].Time > asOf }) - 1
		builder.marketConditionReady = true
	} else {
		index := builder.marketConditionIndex
		for index+1 < len(points) && points[index+1].Time <= asOf {
			index++
		}
		builder.marketConditionIndex = index
	}
	builder.marketConditionAsOf = asOf
	if builder.marketConditionIndex < 0 {
		return 0
	}
	return points[builder.marketConditionIndex].Value
}

func (builder *historicalEnvironment) marketConditionAt(asOf int64) int {
	points := builder.dataset.MarketConditions
	index := sort.Search(len(points), func(i int) bool { return points[i].Time > asOf })
	if index == 0 {
		return 0
	}
	return points[index-1].Value
}

func (builder *historicalEnvironment) series(symbol, interval string, asOf int64, limit int) []Bar {
	key := BarSeriesKey(symbol, interval)
	all := builder.dataset.Bars[key]
	end := sort.Search(len(all), func(i int) bool { return all[i].CloseTime > asOf })
	overlay, hasOverlay := builder.overlays[key]
	if hasOverlay && (overlay.OpenTime > asOf || overlay.CloseTime > asOf) {
		hasOverlay = false
	}
	replacesLast := hasOverlay && end > 0 && all[end-1].OpenTime == overlay.OpenTime
	canonicalLimit := limit
	if limit > 0 && hasOverlay && !replacesLast {
		canonicalLimit--
		if canonicalLimit < 0 {
			canonicalLimit = 0
		}
	}
	start := 0
	if limit > 0 && end > canonicalLimit {
		start = end - canonicalLimit
	}
	count := end - start
	if hasOverlay && !replacesLast {
		count++
	}
	if count <= 0 {
		return nil
	}
	out := make([]Bar, 0, count)
	canonicalEnd := end
	if hasOverlay {
		out = append(out, overlay)
		if replacesLast {
			canonicalEnd--
		}
	}
	for i := canonicalEnd - 1; i >= start; i-- {
		out = append(out, all[i])
	}
	return out
}

func (builder *historicalEnvironment) klinePriceSeries(symbol, interval string, asOf int64, limit int) line.KLinePrice {
	key := BarSeriesKey(symbol, interval)
	all := builder.dataset.Bars[key]
	end := sort.Search(len(all), func(i int) bool { return all[i].CloseTime > asOf })
	overlay, hasOverlay := builder.overlays[key]
	if hasOverlay && (overlay.OpenTime > asOf || overlay.CloseTime > asOf) {
		hasOverlay = false
	}
	replacesLast := hasOverlay && end > 0 && all[end-1].OpenTime == overlay.OpenTime
	canonicalLimit := limit
	if limit > 0 && hasOverlay && !replacesLast {
		canonicalLimit--
		if canonicalLimit < 0 {
			canonicalLimit = 0
		}
	}
	start := 0
	if limit > 0 && end > canonicalLimit {
		start = end - canonicalLimit
	}
	count := end - start
	if hasOverlay && !replacesLast {
		count++
	}
	if count <= 0 {
		return line.KLinePrice{}
	}
	p := line.KLinePrice{
		High: make([]float64, 0, count), Low: make([]float64, 0, count),
		Close: make([]float64, 0, count), Open: make([]float64, 0, count),
		Amount: make([]float64, 0, count), Qps: make([]float64, 0, count),
		TakerBuyAmount: make([]float64, 0, count), TakerBuyRatio: make([]float64, 0, count),
	}
	appendBar := func(bar Bar) {
		p.High = append(p.High, bar.High)
		p.Low = append(p.Low, bar.Low)
		p.Close = append(p.Close, bar.Close)
		p.Open = append(p.Open, bar.Open)
		p.Amount = append(p.Amount, bar.QuoteVolume)
		p.TakerBuyAmount = append(p.TakerBuyAmount, bar.TakerBuyQuoteVolume)
		if bar.QuoteVolume > 0 {
			p.TakerBuyRatio = append(p.TakerBuyRatio, bar.TakerBuyQuoteVolume/bar.QuoteVolume)
		} else {
			p.TakerBuyRatio = append(p.TakerBuyRatio, 0)
		}
		seconds := klineQPSDurationSeconds(bar)
		if seconds > 0 {
			p.Qps = append(p.Qps, bar.QuoteVolume/seconds)
		} else {
			p.Qps = append(p.Qps, 0)
		}
	}
	canonicalEnd := end
	if hasOverlay {
		appendBar(overlay)
		if replacesLast {
			canonicalEnd--
		}
	}
	for i := canonicalEnd - 1; i >= start; i-- {
		appendBar(all[i])
	}
	return p
}

func (builder *historicalEnvironment) visibleBars(symbol, interval string, asOf int64) []Bar {
	all := builder.dataset.Bars[BarSeriesKey(symbol, interval)]
	end := sort.Search(len(all), func(i int) bool { return all[i].CloseTime > asOf })
	visible := all[:end]
	overlay, ok := builder.overlays[BarSeriesKey(symbol, interval)]
	if !ok || overlay.OpenTime > asOf || overlay.CloseTime > asOf {
		// Standard 1m replay must stay zero-copy here. Copying all historical
		// bars on every minute turns a long backtest into O(n²) memory work.
		return visible
	}
	// Intrabar replay is rare and may replace/append the current partial bar.
	// Copy only in that path so the canonical dataset is never mutated.
	out := append([]Bar(nil), visible...)
	if len(out) > 0 && out[len(out)-1].OpenTime == overlay.OpenTime {
		out[len(out)-1] = overlay
	} else {
		out = append(out, overlay)
	}
	return out
}

func (builder *historicalEnvironment) tickerStats(symbol string, asOf int64) map[string]interface{} {
	key := BarSeriesKey(symbol, builder.dataset.ExecutionInterval)
	all := builder.dataset.Bars[key]
	end := sort.Search(len(all), func(i int) bool { return all[i].CloseTime > asOf })
	startTime := asOf - (24 * time.Hour).Milliseconds()
	start := sort.Search(end, func(i int) bool { return all[i].CloseTime >= startTime })
	overlay, hasOverlay := builder.overlays[key]
	if hasOverlay && (overlay.OpenTime > asOf || overlay.CloseTime > asOf) {
		hasOverlay = false
	}
	replacesLast := hasOverlay && end > start && all[end-1].OpenTime == overlay.OpenTime
	count := 0
	open, close, low, high := 0.0, 0.0, 0.0, 0.0
	apply := func(bar Bar) {
		if count == 0 {
			open, low, high = bar.Open, bar.Low, bar.High
		} else {
			if bar.Low < low {
				low = bar.Low
			}
			if bar.High > high {
				high = bar.High
			}
		}
		close = bar.Close
		count++
	}
	for i := start; i < end; i++ {
		if replacesLast && i == end-1 {
			continue
		}
		apply(all[i])
	}
	if hasOverlay {
		apply(overlay)
	}
	if count == 0 {
		return map[string]interface{}{"PercentChange": 0.0, "Close": 0.0, "Open": 0.0, "Low": 0.0, "High": 0.0}
	}
	change := 0.0
	if open > 0 {
		change = (close - open) / open * 100
	}
	return map[string]interface{}{"PercentChange": change, "Close": close, "Open": open, "Low": low, "High": high}
}

func buildIntrabarOverlays(dataset Dataset, partialMinute Bar, asOf int64) (map[string]Bar, error) {
	if partialMinute.OpenTime <= 0 || partialMinute.CloseTime != asOf || partialMinute.Open <= 0 || partialMinute.Close <= 0 || partialMinute.High < partialMinute.Low {
		return nil, fmt.Errorf("invalid intrabar partial minute at %d", asOf)
	}
	overlays := map[string]Bar{BarSeriesKey(dataset.Symbol, "1m"): partialMinute}
	minuteBars := dataset.Bars[BarSeriesKey(dataset.Symbol, "1m")]
	for _, interval := range dataset.Intervals {
		if interval == "1m" {
			continue
		}
		windowStart, err := intervalWindowStart(interval, asOf)
		if err != nil {
			return nil, err
		}
		startIndex := sort.Search(len(minuteBars), func(i int) bool { return minuteBars[i].OpenTime >= windowStart })
		endIndex := sort.Search(len(minuteBars), func(i int) bool { return minuteBars[i].OpenTime >= partialMinute.OpenTime })
		components := make([]Bar, 0, endIndex-startIndex+1)
		for _, bar := range minuteBars[startIndex:endIndex] {
			if bar.CloseTime > asOf {
				break
			}
			components = append(components, bar)
		}
		components = append(components, partialMinute)
		overlay := aggregatePartialBars(dataset.Symbol, interval, windowStart, asOf, components)
		overlays[BarSeriesKey(dataset.Symbol, interval)] = overlay
	}
	return overlays, nil
}

func (builder *historicalEnvironment) cachedIntervalWindowStart(interval string, asOf int64) (int64, error) {
	if builder.standardOptimizations {
		if ms := builder.intervalMillis[interval]; ms > 0 {
			return asOf - positiveMillisMod(asOf, ms), nil
		}
	}
	return intervalWindowStart(interval, asOf)
}

func (builder *historicalEnvironment) cachedKlineQPSDurationSeconds(bar Bar) float64 {
	if builder.standardOptimizations {
		if ms := builder.intervalMillis[bar.Interval]; ms > 0 {
			durationMillis := bar.CloseTime - bar.OpenTime
			nominalMillis := ms - 1
			if nominalMillis > durationMillis {
				durationMillis = nominalMillis
			}
			if durationMillis <= 0 {
				return 0
			}
			return float64(durationMillis) / 1000
		}
	}
	return klineQPSDurationSeconds(bar)
}

func intervalWindowStart(interval string, asOf int64) (int64, error) {
	t := time.UnixMilli(asOf).UTC()
	switch interval {
	case "1M":
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC).UnixMilli(), nil
	case "1w":
		days := (int(t.Weekday()) + 6) % 7
		return time.Date(t.Year(), t.Month(), t.Day()-days, 0, 0, 0, 0, time.UTC).UnixMilli(), nil
	}
	duration, err := intervalDuration(interval, t)
	if err != nil {
		return 0, err
	}
	ms := duration.Milliseconds()
	return asOf - positiveMillisMod(asOf, ms), nil
}

func positiveMillisMod(value, mod int64) int64 {
	result := value % mod
	if result < 0 {
		result += mod
	}
	return result
}

func aggregatePartialBars(symbol, interval string, openTime, closeTime int64, bars []Bar) Bar {
	result := Bar{Symbol: symbol, Interval: interval, OpenTime: openTime, CloseTime: closeTime}
	if len(bars) == 0 {
		return result
	}
	result.Open = bars[0].Open
	result.High = bars[0].High
	result.Low = bars[0].Low
	result.Close = bars[len(bars)-1].Close
	for _, bar := range bars {
		if bar.High > result.High {
			result.High = bar.High
		}
		if bar.Low < result.Low {
			result.Low = bar.Low
		}
		result.Volume += bar.Volume
		result.QuoteVolume += bar.QuoteVolume
		result.TradeCount += bar.TradeCount
		result.TakerBuyQuoteVolume += bar.TakerBuyQuoteVolume
	}
	return result
}

func (builder *historicalEnvironment) addIndicators(env map[string]interface{}, asOf int64) error {
	if builder.standardOptimizations {
		return builder.addIndicatorsOptimized(env, asOf)
	}
	return builder.addIndicatorsLegacy(env, asOf)
}

func (builder *historicalEnvironment) addIndicatorsLegacy(env map[string]interface{}, asOf int64) error {
	groups := []struct {
		name  string
		items []technology.IndicatorConfig
	}{{"ma", builder.technology.MA}, {"ema", builder.technology.EMA}, {"macd", builder.technology.MACD}, {"rsi", builder.technology.RSI}, {"kc", builder.technology.KC}, {"boll", builder.technology.BOLL}, {"atr", builder.technology.ATR}, {"adx", builder.technology.ADX}, {"mfi", builder.technology.MFI}, {"obv", builder.technology.OBV}, {"cci", builder.technology.CCI}, {"roc", builder.technology.ROC}, {"kdj", builder.technology.KDJ}, {"supertrend", builder.technology.Supertrend}, {"donchian", builder.technology.Donchian}}
	prices := map[string]line.KLinePrice{}
	for _, group := range groups {
		for _, item := range group.items {
			if !item.Enable {
				continue
			}
			p, ok := prices[item.KlineInterval]
			if !ok {
				p = builder.klinePriceSeries(builder.dataset.Symbol, item.KlineInterval, asOf, builder.warmupBars)
				if len(p.Close) == 0 {
					return fmt.Errorf("%w: indicator %s has no visible bars", ErrInsufficientHistoricalBars, item.Name)
				}
				prices[item.KlineInterval] = p
				env["kline_"+item.KlineInterval] = p
			}
			var err error
			switch group.name {
			case "ma":
				var d []float64
				d, err = line.CalculateSimpleMovingAverage(p.Close, item.Period)
				env[item.Name] = line.ConfigData{KlineInterval: item.KlineInterval, Period: item.Period, Data: d}
			case "ema":
				var d []float64
				d, err = line.CalculateExponentialMovingAverage(p.Close, item.Period)
				env[item.Name] = line.ConfigData{KlineInterval: item.KlineInterval, Period: item.Period, Data: d}
			case "macd":
				var a, b, c []float64
				a, b, c, err = line.CalculateMACD(p.Close, item.FastPeriod, item.SlowPeriod, item.SignalPeriod)
				env[item.Name] = line.MACDConfigData{KlineInterval: item.KlineInterval, FastPeriod: item.FastPeriod, SlowPeriod: item.SlowPeriod, SignalPeriod: item.SignalPeriod, DIF: a, DEA: b, Histogram: c}
			case "rsi":
				var d []float64
				d, err = line.CalculateRSI(p.Close, item.Period)
				env[item.Name] = line.ConfigData{KlineInterval: item.KlineInterval, Period: item.Period, Data: d}
			case "kc":
				var a, b, c []float64
				a, b, c, err = line.CalculateKeltnerChannels(p.High, p.Low, p.Close, item.Period, item.Multiplier)
				env[item.Name] = line.ConfigData{KlineInterval: item.KlineInterval, Period: item.Period, Multiplier: item.Multiplier, High: a, Mid: b, Low: c}
			case "boll":
				var a, b, c []float64
				a, b, c, err = line.CalculateBollingerBands(p.Close, item.Period, item.StdDevMultiplier)
				env[item.Name] = line.ConfigData{KlineInterval: item.KlineInterval, Period: item.Period, StdDevMultiplier: item.StdDevMultiplier, High: a, Mid: b, Low: c}
			case "atr":
				var d []float64
				d, err = line.CalculateAtr(p.High, p.Low, p.Close, item.Period)
				env[item.Name] = line.ConfigData{KlineInterval: item.KlineInterval, Period: item.Period, Data: d}
			case "adx":
				var a, b, c []float64
				a, b, c, err = line.CalculateADX(p.High, p.Low, p.Close, item.Period)
				env[item.Name] = line.ADXConfigData{KlineInterval: item.KlineInterval, Period: item.Period, ADX: a, PlusDI: b, MinusDI: c}
			case "mfi":
				var d []float64
				d, err = line.CalculateMFI(p.High, p.Low, p.Close, p.Amount, item.Period)
				env[item.Name] = line.ConfigData{KlineInterval: item.KlineInterval, Period: item.Period, Data: d}
			case "obv":
				var d []float64
				d, err = line.CalculateOBV(p.Close, p.Amount)
				env[item.Name] = line.OBVConfigData{KlineInterval: item.KlineInterval, Data: d}
			case "cci":
				var d []float64
				d, err = line.CalculateCCI(p.High, p.Low, p.Close, item.Period)
				env[item.Name] = line.ConfigData{KlineInterval: item.KlineInterval, Period: item.Period, Data: d}
			case "roc":
				var d []float64
				d, err = line.CalculateROC(p.Close, item.Period)
				env[item.Name] = line.ConfigData{KlineInterval: item.KlineInterval, Period: item.Period, Data: d}
			case "kdj":
				var a, b, c []float64
				a, b, c, err = line.Kdj(p.High, p.Low, p.Close, item.Period, item.KPeriod, item.DPeriod)
				env[item.Name] = line.KDJConfigData{KlineInterval: item.KlineInterval, Period: item.Period, KPeriod: item.KPeriod, DPeriod: item.DPeriod, K: a, D: b, J: c}
			case "supertrend":
				var a, b []float64
				a, b, err = line.CalculateSupertrend(p.High, p.Low, p.Close, item.Period, item.Multiplier)
				env[item.Name] = line.SupertrendConfigData{KlineInterval: item.KlineInterval, Period: item.Period, Multiplier: item.Multiplier, Data: a, Trend: b}
			case "donchian":
				var a, b, c []float64
				a, b, c, err = line.CalculateDonchianChannels(p.High, p.Low, item.Period)
				env[item.Name] = line.ConfigData{KlineInterval: item.KlineInterval, Period: item.Period, High: a, Mid: b, Low: c}
			}
			if err != nil {
				return fmt.Errorf("%w: indicator %s warmup at %d: %v", ErrInsufficientHistoricalBars, item.Name, asOf, err)
			}
		}
	}
	return nil
}
func toKLinePrice(bars []Bar) line.KLinePrice {
	p := line.KLinePrice{
		High: make([]float64, 0, len(bars)), Low: make([]float64, 0, len(bars)),
		Close: make([]float64, 0, len(bars)), Open: make([]float64, 0, len(bars)),
		Amount: make([]float64, 0, len(bars)), Qps: make([]float64, 0, len(bars)),
		TakerBuyAmount: make([]float64, 0, len(bars)), TakerBuyRatio: make([]float64, 0, len(bars)),
	}
	for _, b := range bars {
		p.High = append(p.High, b.High)
		p.Low = append(p.Low, b.Low)
		p.Close = append(p.Close, b.Close)
		p.Open = append(p.Open, b.Open)
		p.Amount = append(p.Amount, b.QuoteVolume)
		p.TakerBuyAmount = append(p.TakerBuyAmount, b.TakerBuyQuoteVolume)
		if b.QuoteVolume > 0 {
			p.TakerBuyRatio = append(p.TakerBuyRatio, b.TakerBuyQuoteVolume/b.QuoteVolume)
		} else {
			p.TakerBuyRatio = append(p.TakerBuyRatio, 0)
		}
		seconds := klineQPSDurationSeconds(b)
		if seconds > 0 {
			p.Qps = append(p.Qps, b.QuoteVolume/seconds)
		} else {
			p.Qps = append(p.Qps, 0)
		}
	}
	return p
}

func klineQPSDurationSeconds(bar Bar) float64 {
	durationMillis := bar.CloseTime - bar.OpenTime
	// Live InitParseEnv calculates QPS from Binance's K-line CloseTime/OpenTime.
	// Binance keeps CloseTime at the nominal interval end even while the current
	// K-line is still forming. Backtest overlays use asOf as CloseTime to prevent
	// future visibility, so use the nominal interval length only for that shorter
	// partial-bar case. Completed historical bars keep their original duration.
	if nominal, err := intervalDuration(bar.Interval, time.UnixMilli(bar.OpenTime).UTC()); err == nil {
		nominalMillis := nominal.Milliseconds() - 1
		if nominalMillis > durationMillis {
			durationMillis = nominalMillis
		}
	}
	if durationMillis <= 0 {
		return 0
	}
	return float64(durationMillis) / 1000
}

func unrealizedPnL(p *Position, price float64) float64 {
	if p == nil {
		return 0
	}
	if p.Side == "SHORT" {
		return (p.EntryPrice - price) * math.Abs(p.Quantity)
	}
	return (price - p.EntryPrice) * math.Abs(p.Quantity)
}
func grossROI(p *Position, price float64, leverage int) float64 {
	if p == nil || price <= 0 {
		return 0
	}
	return utils.FuturesLeveragedROI(unrealizedPnL(p, price), math.Abs(p.Quantity), price, int64(leverage))
}

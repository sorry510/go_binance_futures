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
	dataset    Dataset
	technology technology.TechnologyConfig
	overlays   map[string]Bar
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
	return &historicalEnvironment{dataset: dataset, technology: config}, nil
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

func (builder *historicalEnvironment) build(asOf int64, position *Position, cash float64, config RunConfig) (map[string]interface{}, int, error) {
	currentBars := builder.series(builder.dataset.Symbol, builder.dataset.ExecutionInterval, asOf, DefaultWarmupBars)
	if len(currentBars) == 0 {
		return nil, 0, fmt.Errorf("%w: no execution bars visible at %d", ErrInsufficientHistoricalBars, asOf)
	}
	current := currentBars[0]
	targetStats := builder.tickerStats(builder.dataset.Symbol, asOf)
	env := map[string]interface{}{
		"SystemStartTime": builder.dataset.StartTime, "NowTime": asOf, "NowPrice": current.Close,
		"NowSymbolPercentChange": targetStats["PercentChange"], "NowSymbolClose": targetStats["Close"], "NowSymbolOpen": targetStats["Open"], "NowSymbolLow": targetStats["Low"], "NowSymbolHigh": targetStats["High"],
		"KdjSimple": line.KdjSimple, "IsAsc": utils.IsAsc, "IsDesc": utils.IsDesc,
	}
	condition := 0
	if builder.dataset.MarketConditionRequired {
		condition = builder.marketConditionAt(asOf)
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
	env["ROI"], env["NetROI"], env["Fee"], env["NetProfit"], env["Position"] = roi, netROI, position.OpenFee+projectedCloseFee, net, p
	env["Positions"] = []markettypes.FuturesPosition{{Symbol: builder.dataset.Symbol, Side: position.Side, Amount: strconv.FormatFloat(position.Quantity, 'f', -1, 64), Leverage: int64(config.Leverage), EntryPrice: strconv.FormatFloat(position.EntryPrice, 'f', -1, 64), MarkPrice: strconv.FormatFloat(current.Close, 'f', -1, 64), UnrealizedProfit: strconv.FormatFloat(gross, 'f', -1, 64), SourceType: "backtest", CreateTime: position.EntryTime}}
	return env, condition, nil
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
	out := append([]Bar(nil), all[start:end]...)
	if hasOverlay {
		if len(out) > 0 && out[len(out)-1].OpenTime == overlay.OpenTime {
			out[len(out)-1] = overlay
		} else {
			out = append(out, overlay)
		}
	}
	if len(out) == 0 {
		return nil
	}
	if limit > 0 && len(out) > limit {
		out = out[len(out)-limit:]
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
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
	window := append([]Bar(nil), all[start:end]...)
	overlay, hasOverlay := builder.overlays[key]
	if hasOverlay && overlay.OpenTime <= asOf && overlay.CloseTime <= asOf {
		if len(window) > 0 && window[len(window)-1].OpenTime == overlay.OpenTime {
			window[len(window)-1] = overlay
		} else {
			window = append(window, overlay)
		}
	}
	if len(window) == 0 {
		return map[string]interface{}{"PercentChange": 0.0, "Close": 0.0, "Open": 0.0, "Low": 0.0, "High": 0.0}
	}
	open := window[0].Open
	close := window[len(window)-1].Close
	low, high := window[0].Low, window[0].High
	for _, bar := range window {
		if bar.Low < low {
			low = bar.Low
		}
		if bar.High > high {
			high = bar.High
		}
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
				bars := builder.series(builder.dataset.Symbol, item.KlineInterval, asOf, DefaultWarmupBars)
				if len(bars) == 0 {
					return fmt.Errorf("%w: indicator %s has no visible bars", ErrInsufficientHistoricalBars, item.Name)
				}
				p = toKLinePrice(bars)
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
	p := line.KLinePrice{}
	for _, b := range bars {
		p.High = append(p.High, b.High)
		p.Low = append(p.Low, b.Low)
		p.Close = append(p.Close, b.Close)
		p.Open = append(p.Open, b.Open)
		p.Amount = append(p.Amount, b.QuoteVolume)
		seconds := float64(b.CloseTime-b.OpenTime) / 1000
		if seconds > 0 {
			p.Qps = append(p.Qps, b.QuoteVolume/seconds)
		} else {
			p.Qps = append(p.Qps, 0)
		}
	}
	return p
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

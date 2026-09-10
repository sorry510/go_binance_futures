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
	currentBars := builder.series(builder.dataset.Symbol, builder.dataset.ExecutionInterval, asOf, DefaultWarmupBars)
	if len(currentBars) == 0 {
		return nil, 0, fmt.Errorf("%w: no execution bars visible at %d", ErrInsufficientHistoricalBars, asOf)
	}
	current := currentBars[0]
	benchmarks := map[string]map[string]interface{}{}
	changes := map[string]float64{}
	for _, symbol := range BenchmarkSymbols {
		stats := builder.tickerStats(symbol, asOf)
		benchmarks[symbol] = stats
		changes[symbol], _ = stats["PercentChange"].(float64)
	}
	basicTrend := changes["BTCUSDT"]*0.6 + changes["ETHUSDT"]*0.3 + changes["SOLUSDT"]*0.05 + changes["BNBUSDT"]*0.05
	condition := classifyHistoricalRegime(changes, currentBars)
	targetStats := builder.tickerStats(builder.dataset.Symbol, asOf)
	env := map[string]interface{}{
		"SystemStartTime": builder.dataset.StartTime, "MarketCondition": strconv.Itoa(condition), "NowTime": asOf, "NowPrice": current.Close,
		"NowSymbolPercentChange": targetStats["PercentChange"], "NowSymbolClose": targetStats["Close"], "NowSymbolOpen": targetStats["Open"], "NowSymbolLow": targetStats["Low"], "NowSymbolHigh": targetStats["High"],
		"BasicTrend": basicTrend, "KdjSimple": line.KdjSimple, "IsAsc": utils.IsAsc, "IsDesc": utils.IsDesc,
	}
	for symbol, value := range benchmarks {
		env[symbol] = value
	}
	if _, exists := env[builder.dataset.Symbol]; !exists {
		env[builder.dataset.Symbol] = targetStats
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

func (builder *historicalEnvironment) series(symbol, interval string, asOf int64, limit int) []Bar {
	all := builder.dataset.Bars[BarSeriesKey(symbol, interval)]
	end := sort.Search(len(all), func(i int) bool { return all[i].CloseTime > asOf })
	if end <= 0 {
		return nil
	}
	start := 0
	if limit > 0 && end > limit {
		start = end - limit
	}
	out := append([]Bar(nil), all[start:end]...)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}
func (builder *historicalEnvironment) tickerStats(symbol string, asOf int64) map[string]interface{} {
	all := builder.dataset.Bars[BarSeriesKey(symbol, builder.dataset.ExecutionInterval)]
	end := sort.Search(len(all), func(i int) bool { return all[i].CloseTime > asOf })
	if end == 0 {
		return map[string]interface{}{"PercentChange": 0.0, "Close": 0.0, "Open": 0.0, "Low": 0.0, "High": 0.0}
	}
	startTime := asOf - (24 * time.Hour).Milliseconds()
	start := sort.Search(end, func(i int) bool { return all[i].CloseTime >= startTime })
	if start >= end {
		start = end - 1
	}
	window := all[start:end]
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

func classifyHistoricalRegime(changes map[string]float64, targetBars []Bar) int {
	btc, eth, sol, bnb := changes["BTCUSDT"], changes["ETHUSDT"], changes["SOLUSDT"], changes["BNBUSDT"]
	values := []float64{btc, eth, sol, bnb}
	up, down := 0, 0
	sum := 0.0
	for _, v := range values {
		sum += v
		if v > 0 {
			up++
		} else if v < 0 {
			down++
		}
	}
	avg := sum / 4
	if up == 4 && avg >= 0.5 {
		return markettypes.MarketConditionBroadRise
	}
	if down == 4 && avg <= -0.5 {
		return markettypes.MarketConditionBroadDecline
	}
	others := (eth + sol + bnb) / 3
	if btc >= 0.5 && others <= -0.2 {
		return markettypes.MarketConditionBullishDivergence
	}
	if btc <= -0.5 && others >= 0.2 {
		return markettypes.MarketConditionBearishDivergence
	}
	weighted := btc*0.5 + eth*0.3 + sol*0.1 + bnb*0.1
	vol := historicalVolatility(targetBars, 20)
	if math.Abs(weighted) < 0.5 {
		if vol >= 2 {
			return markettypes.MarketConditionHighVolatility
		}
		if vol <= 0.35 {
			return markettypes.MarketConditionLowVolatility
		}
		return markettypes.MarketConditionSideways
	}
	if weighted >= 5 {
		return markettypes.MarketConditionStrongBull
	}
	if weighted >= 1 {
		return markettypes.MarketConditionBull
	}
	if weighted <= -5 {
		return markettypes.MarketConditionStrongBear
	}
	if weighted <= -1 {
		return markettypes.MarketConditionBear
	}
	return markettypes.MarketConditionSideways
}
func historicalVolatility(bars []Bar, limit int) float64 {
	if len(bars) < 2 {
		return 0
	}
	if len(bars) > limit {
		bars = bars[:limit]
	}
	returns := make([]float64, 0, len(bars)-1)
	for i := 0; i+1 < len(bars); i++ {
		if bars[i+1].Close > 0 {
			returns = append(returns, (bars[i].Close-bars[i+1].Close)/bars[i+1].Close*100)
		}
	}
	if len(returns) == 0 {
		return 0
	}
	mean := 0.0
	for _, v := range returns {
		mean += v
	}
	mean /= float64(len(returns))
	sum := 0.0
	for _, v := range returns {
		d := v - mean
		sum += d * d
	}
	return math.Sqrt(sum / float64(len(returns)))
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

package backtest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"go_binance_futures/feature/strategy/line"
	"go_binance_futures/service/historicalmarket"
	"go_binance_futures/technology"
)

type DatasetBuilder struct {
	Repository *historicalmarket.Repository
	WarmupBars int
}

func (builder DatasetBuilder) Build(ctx context.Context, request DatasetRequest) (Dataset, error) {
	return builder.BuildWithProgress(ctx, request, nil)
}

func (builder DatasetBuilder) BuildWithProgress(ctx context.Context, request DatasetRequest, progress ProgressCallback) (Dataset, error) {
	repo := builder.Repository
	if repo == nil {
		repo = historicalmarket.DefaultRepository()
	}
	if builder.WarmupBars <= 0 {
		builder.WarmupBars = DefaultWarmupBars
	}
	request.Symbol = strings.ToUpper(strings.TrimSpace(request.Symbol))
	request.ExecutionInterval = strings.TrimSpace(request.ExecutionInterval)
	if request.Symbol == "" || !strings.HasSuffix(request.Symbol, "USDT") {
		return Dataset{}, fmt.Errorf("symbol must be a USDT futures contract")
	}
	if _, err := intervalDuration(request.ExecutionInterval, time.UnixMilli(request.StartTime)); err != nil {
		return Dataset{}, err
	}
	if request.StartTime <= 0 || request.EndTime <= 0 || request.StartTime >= request.EndTime {
		return Dataset{}, fmt.Errorf("valid historical start_time and end_time are required")
	}
	if request.EndTime > time.Now().Add(time.Minute).UnixMilli() {
		return Dataset{}, fmt.Errorf("backtest end_time must not be in the future")
	}
	intervals, err := strategyIntervals(request.ExecutionInterval, request.TechnologyJSON)
	if err != nil {
		return Dataset{}, err
	}
	warmupStart := request.StartTime
	for _, interval := range intervals {
		candidate, err := subtractBars(request.StartTime, interval, builder.WarmupBars)
		if err != nil {
			return Dataset{}, err
		}
		if candidate < warmupStart {
			warmupStart = candidate
		}
	}
	dataset := Dataset{Market: historicalmarket.MarketFuturesUSDT, Symbol: request.Symbol, ExecutionInterval: request.ExecutionInterval, Intervals: intervals, BenchmarkSymbols: append([]string(nil), BenchmarkSymbols...), StartTime: request.StartTime, EndTime: request.EndTime, WarmupStartTime: warmupStart, Bars: map[string][]Bar{}, Funding: []Funding{}}
	benchmarkUnits := len(BenchmarkSymbols)
	for _, symbol := range BenchmarkSymbols {
		if symbol == request.Symbol {
			benchmarkUnits--
			break
		}
	}
	totalUnits := len(intervals) + benchmarkUnits + 1 // target series + benchmark series + funding
	completedUnits := 0
	report := func() {
		if progress != nil {
			progress(completedUnits, totalUnits)
		}
	}
	report()
	for _, interval := range intervals {
		start, _ := subtractBars(request.StartTime, interval, builder.WarmupBars)
		rows, err := repo.LoadKlines(ctx, dataset.Market, request.Symbol, interval, start, request.EndTime)
		if err != nil {
			return Dataset{}, fmt.Errorf("load historical %s %s: %w", request.Symbol, interval, err)
		}
		bars := convertHistoricalKlines(rows)
		if interval == request.ExecutionInterval && countTradeBars(bars, request.StartTime, request.EndTime) < 2 {
			return Dataset{}, fmt.Errorf("insufficient execution bars in requested range")
		}
		dataset.Bars[BarSeriesKey(request.Symbol, interval)] = bars
		completedUnits++
		report()
	}
	benchmarkStart, _ := subtractBars(request.StartTime, request.ExecutionInterval, builder.WarmupBars)
	for _, symbol := range BenchmarkSymbols {
		key := BarSeriesKey(symbol, request.ExecutionInterval)
		if _, exists := dataset.Bars[key]; exists {
			continue
		}
		rows, err := repo.LoadKlines(ctx, dataset.Market, symbol, request.ExecutionInterval, benchmarkStart, request.EndTime)
		if err != nil {
			return Dataset{}, fmt.Errorf("load benchmark %s: %w", symbol, err)
		}
		dataset.Bars[key] = convertHistoricalKlines(rows)
		completedUnits++
		report()
	}
	fundingRows, err := repo.LoadFunding(ctx, dataset.Market, request.Symbol, request.StartTime, request.EndTime)
	if err != nil {
		return Dataset{}, fmt.Errorf("load historical funding %s: %w", request.Symbol, err)
	}
	for _, row := range fundingRows {
		dataset.Funding = append(dataset.Funding, Funding{Symbol: request.Symbol, FundingTime: row.FundingTime, FundingRate: row.FundingRate, MarkPrice: row.MarkPrice})
	}
	completedUnits++
	report()
	sort.Slice(dataset.Funding, func(i, j int) bool { return dataset.Funding[i].FundingTime < dataset.Funding[j].FundingTime })
	dataset.DatasetSpecHash = DatasetSpecHash(dataset)
	dataset.DatasetID = "ds_" + dataset.DatasetSpecHash[:24]
	dataset.DataHash = DatasetDataHash(dataset)
	return dataset, nil
}

func convertHistoricalKlines(rows []historicalmarket.Kline) []Bar {
	out := make([]Bar, 0, len(rows))
	for _, row := range rows {
		out = append(out, Bar{Symbol: row.Symbol, Interval: row.Interval, OpenTime: row.OpenTime, CloseTime: row.CloseTime, Open: row.Open, High: row.High, Low: row.Low, Close: row.Close, Volume: row.Volume, QuoteVolume: row.QuoteVolume, TradeCount: row.TradeCount, TakerBuyQuoteVolume: row.TakerBuyQuoteVolume})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].OpenTime < out[j].OpenTime })
	return out
}

func strategyIntervals(executionInterval, raw string) ([]string, error) {
	var config technology.TechnologyConfig
	if strings.TrimSpace(raw) != "" {
		if err := json.Unmarshal([]byte(raw), &config); err != nil {
			return nil, fmt.Errorf("decode technology config: %w", err)
		}
		if err := line.ValidateTechnologyConfig(config); err != nil {
			return nil, err
		}
	}
	seen := map[string]bool{executionInterval: true}
	all := [][]technology.IndicatorConfig{config.MA, config.EMA, config.MACD, config.RSI, config.KC, config.BOLL, config.ATR, config.ADX, config.MFI, config.OBV, config.CCI, config.ROC, config.KDJ, config.Supertrend, config.Donchian}
	for _, group := range all {
		for _, item := range group {
			if item.Enable {
				seen[item.KlineInterval] = true
			}
		}
	}
	result := make([]string, 0, len(seen))
	for value := range seen {
		if _, err := historicalmarket.KlineTable(value); err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return intervalRank(result[i]) < intervalRank(result[j]) })
	return result, nil
}

func intervalRank(value string) int64 {
	d, err := intervalDuration(value, time.Unix(0, 0))
	if err != nil {
		return 1 << 62
	}
	return int64(d)
}
func intervalDuration(value string, at time.Time) (time.Duration, error) {
	switch value {
	case "1m":
		return time.Minute, nil
	case "3m":
		return 3 * time.Minute, nil
	case "5m":
		return 5 * time.Minute, nil
	case "15m":
		return 15 * time.Minute, nil
	case "30m":
		return 30 * time.Minute, nil
	case "1h":
		return time.Hour, nil
	case "2h":
		return 2 * time.Hour, nil
	case "4h":
		return 4 * time.Hour, nil
	case "6h":
		return 6 * time.Hour, nil
	case "8h":
		return 8 * time.Hour, nil
	case "12h":
		return 12 * time.Hour, nil
	case "1d":
		return 24 * time.Hour, nil
	case "3d":
		return 72 * time.Hour, nil
	case "1w":
		return 7 * 24 * time.Hour, nil
	case "1M":
		return at.AddDate(0, 1, 0).Sub(at), nil
	default:
		return 0, fmt.Errorf("unsupported backtest interval %q", value)
	}
}
func subtractBars(start int64, interval string, count int) (int64, error) {
	t := time.UnixMilli(start).UTC()
	if interval == "1M" {
		return t.AddDate(0, -count, 0).UnixMilli(), nil
	}
	d, err := intervalDuration(interval, t)
	if err != nil {
		return 0, err
	}
	return t.Add(-time.Duration(count) * d).UnixMilli(), nil
}
func countTradeBars(bars []Bar, start, end int64) int {
	n := 0
	for _, b := range bars {
		if b.CloseTime >= start && b.CloseTime <= end {
			n++
		}
	}
	return n
}

func DatasetSpecHash(dataset Dataset) string {
	h := sha256.New()
	enc := json.NewEncoder(h)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(struct {
		Market, Symbol, ExecutionInterval string
		Intervals, Benchmarks             []string
		Start, End, Warmup                int64
	}{dataset.Market, dataset.Symbol, dataset.ExecutionInterval, dataset.Intervals, dataset.BenchmarkSymbols, dataset.StartTime, dataset.EndTime, dataset.WarmupStartTime})
	return hex.EncodeToString(h.Sum(nil))
}
func DatasetDataHash(dataset Dataset) string {
	h := sha256.New()
	enc := json.NewEncoder(h)
	enc.SetEscapeHTML(false)
	spec := dataset.DatasetSpecHash
	if spec == "" {
		spec = DatasetSpecHash(dataset)
	}
	_ = enc.Encode(spec)
	keys := make([]string, 0, len(dataset.Bars))
	for key := range dataset.Bars {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		_ = enc.Encode(key)
		for _, bar := range dataset.Bars[key] {
			_ = enc.Encode(bar)
		}
	}
	for _, funding := range dataset.Funding {
		_ = enc.Encode(funding)
	}
	return hex.EncodeToString(h.Sum(nil))
}

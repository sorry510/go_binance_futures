package backtest

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"go_binance_futures/service/historicalmarket"
)

func TestBT2IncrementalIntrabarMatchesLegacy(t *testing.T) {
	dataset := cacheFixtureDataset(4800)
	// Include market-condition changes inside a minute and backward replay
	// to verify that the adaptive cache does not leak temporal state.
	minutes := dataset.Bars[BarSeriesKey(dataset.Symbol, "1m")]
	dataset.MarketConditionRequired = true
	dataset.MarketConditions = []MarketConditionPoint{
		{Time: minutes[3599].CloseTime, Value: 2},
		{Time: minutes[3600].OpenTime + 20_000, Value: 3},
		{Time: minutes[4190].OpenTime + 25_000, Value: 4},
	}
	legacy, err := newHistoricalEnvironment(dataset, cacheTechnologyJSON())
	if err != nil {
		t.Fatal(err)
	}
	optimized, err := newHistoricalEnvironment(dataset, cacheTechnologyJSON())
	if err != nil {
		t.Fatal(err)
	}
	// Force current [0] indicators through the conservative calculation,
	// so all environment fields can be compared byte-for-byte.
	rules := []Rule{
		{Enable: true, Type: "close_long", Code: "ema_1h_3.Data[0] > 0 && rsi_1h_3.Data[0] > 0 && adx_1h_3.ADX[0] > 0 && atr_1h_3.Data[0] > 0 && dc_1h_3.High[0] > 0"},
	}
	optimized.enableStandardOptimizations(strategyJSON(rules...))
	optimized.adaptiveOptimized = true
	bars := dataset.Bars[BarSeriesKey(dataset.Symbol, "1m")]
	position := &Position{Side: "LONG", EntryTime: bars[3550].OpenTime, EntryPrice: bars[3550].Close, Quantity: 2}
	config := RunConfig{InitialEquity: 1000, Leverage: 4, FeeRate: 0.0005}
	for _, i := range []int{3600, 3601, 3610, 3660, 4200, 4201, 4190, 4260} {
		minute := bars[i]
		for _, second := range []int64{999, 12_999, 41_999, 59_999} {
			partial := minute
			partial.CloseTime = minute.OpenTime + second
			partial.Close = minute.Open + float64(second)/100_000
			partial.High = partial.Close + 1
			partial.Low = minute.Open - 1
			partial.QuoteVolume = 200 + float64(second)/1000
			partial.TakerBuyQuoteVolume = partial.QuoteVolume * .37
			left, lc, leftErr := legacy.BuildIntrabar(partial.CloseTime, partial, position, 1000, config)
			right, rc, rightErr := optimized.BuildIntrabar(partial.CloseTime, partial, position, 1000, config)
			if (leftErr == nil) != (rightErr == nil) {
				t.Fatalf("minute=%d second=%d errors differ: %v vs %v", i, second, leftErr, rightErr)
			}
			if leftErr != nil {
				continue
			}
			if lc != rc {
				t.Fatalf("minute=%d second=%d market condition differs", i, second)
			}
			for _, field := range []string{"KdjSimple", "IsAsc", "IsDesc"} {
				delete(left, field)
				delete(right, field)
			}
			if !reflect.DeepEqual(left, right) {
				for k, l := range left {
					if !reflect.DeepEqual(l, right[k]) {
						t.Fatalf("minute=%d second=%d field=%s differs: legacy=%#v optimized=%#v", i, second, k, l, right[k])
					}
				}
				t.Fatalf("minute=%d second=%d environments differ in keys", i, second)
			}
		}
	}
}

func TestBT2CachedIntrabarOverlaysEqualLegacyAcrossBoundaries(t *testing.T) {
	dataset := cacheFixtureDataset(4800)
	dataset.Intervals = []string{"1m", "1h", "4h", "1d", "1w", "1M"}
	env, err := newHistoricalEnvironment(dataset, "{}")
	if err != nil {
		t.Fatal(err)
	}
	bars := dataset.Bars[BarSeriesKey(dataset.Symbol, "1m")]
	for _, i := range []int{59, 60, 61, 239, 240, 1439, 1440, 3000, 2999, 4320} {
		minute := bars[i]
		for _, tick := range []int64{999, 28_999, 59_999} {
			partial := minute
			partial.CloseTime = minute.OpenTime + tick
			partial.Close = minute.Open + float64(tick)/100_000
			partial.High = partial.Close + 1
			partial.Low = minute.Open - 1
			left, err := buildIntrabarOverlays(dataset, partial, partial.CloseTime)
			if err != nil {
				t.Fatal(err)
			}
			right, err := env.cachedIntrabarOverlays(partial, partial.CloseTime)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(left, right) {
				t.Fatalf("minute=%d tick=%d overlays differ: legacy=%+v optimized=%+v", i, tick, left, right)
			}
		}
	}
}

func TestBT2AdaptiveOptimizedResultsMatchBT1(t *testing.T) {
	testCases := []struct {
		name      string
		closes    []float64
		closeCode string
	}{
		{"current-price", []float64{100, 100.2, 100.2, 100.2}, "NowPrice >= 101"},
		{"roi-only", []float64{100, 100.2, 100.2, 100.2}, "ROI >= 10"},
		{"no-candidate", []float64{100, 100, 100, 100}, "ROI >= 10"},
		{"dynamic-current", []float64{100, 100.2, 100.2, 100.2}, "NowSymbolHigh >= 101"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			d := fixtureDataset(tc.closes)
			bars := d.Bars[BarSeriesKey(d.Symbol, "1m")]
			bars[1].High = 101.5
			bars[1].Low = 99.8
			d.DataHash = DatasetDataHash(d)
			providerFactory := func() (historicalmarket.ResolutionProvider, error) {
				return &intrabarFixtureProvider{
					seconds: []historicalmarket.Kline{
						{Market: d.Market, Symbol: d.Symbol, Interval: "1s", OpenTime: bars[1].OpenTime, CloseTime: bars[1].OpenTime + 999, Open: 100, High: 100.1, Low: 99.8, Close: 100},
						{Market: d.Market, Symbol: d.Symbol, Interval: "1s", OpenTime: bars[1].OpenTime + 1000, CloseTime: bars[1].OpenTime + 1999, Open: 100, High: 101.5, Low: 100, Close: 101.2},
					},
				}, nil
			}
			strategy := StrategySnapshot{TemplateID: 1, TemplateName: tc.name, TechnologyJSON: "{}", StrategyJSON: strategyJSON(
				Rule{Name: "open", Enable: true, Type: "long", Code: "NowPrice >= 100"},
				Rule{Name: "close", Enable: true, Type: "close_long", Code: tc.closeCode},
			)}
			config := RunConfig{InitialEquity: 1000, PositionSizePct: 1, Leverage: 10, TakeProfitPct: 10, StopLossPct: 0}
			before, err := (Engine{ResolutionProviderFactory: providerFactory, disableAdaptiveOptimizations: true}).RunWithResolution(context.Background(), d, strategy, config, ResolutionModeAdaptive, nil)
			if err != nil {
				t.Fatal(err)
			}
			after, err := (Engine{ResolutionProviderFactory: providerFactory}).RunWithResolution(context.Background(), d, strategy, config, ResolutionModeAdaptive, nil)
			if err != nil {
				t.Fatal(err)
			}
			oldRaw, _ := json.Marshal(before)
			newRaw, _ := json.Marshal(after)
			if string(oldRaw) != string(newRaw) {
				t.Fatalf("BT-2 changed full result: before=%s after=%s", oldRaw, newRaw)
			}
		})
	}
}

func TestBT2AdaptiveIndicatorsProduceIdenticalFullResults(t *testing.T) {
	tests := []struct {
		name  string
		open  string
		close string
	}{
		{
			name:  "completed-bars",
			open:  "ema_1h_3.Data[1] > 0 && rsi_1h_3.Data[1] >= 0",
			close: "ema_1h_3.Data[1] > 0 && adx_1h_3.ADX[1] >= 0 && atr_1h_3.Data[1] >= 0",
		},
		{
			name:  "forming-indicators",
			open:  "ema_1h_3.Data[0] > 0 && rsi_1h_3.Data[0] >= 0",
			close: "ema_1h_3.Data[0] > 0 && adx_1h_3.ADX[0] >= 0 && dc_1h_3.High[0] > 0",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := cacheFixtureDataset(720)
			bars := d.Bars[BarSeriesKey(d.Symbol, "1m")]
			d.StartTime = bars[600].CloseTime
			d.EndTime = bars[665].CloseTime
			d.DatasetSpecHash = DatasetSpecHash(d)
			d.DataHash = DatasetDataHash(d)
			strategy := StrategySnapshot{
				TemplateID: 1, TemplateName: tc.name,
				TechnologyJSON: cacheTechnologyJSON(),
				StrategyJSON: strategyJSON(
					Rule{Name: "open", Enable: true, Type: "long", Code: tc.open},
					Rule{Name: "close", Enable: true, Type: "close_long", Code: tc.close},
				),
			}
			config := RunConfig{
				InitialEquity: 1000, PositionSizePct: 0.5, Leverage: 4,
				TakeProfitPct: 0.2, StopLossPct: 2,
				FeeRate: 0.0005, SlippageBps: 5,
			}
			factory := func() (historicalmarket.ResolutionProvider, error) {
				return &intrabarFixtureProvider{}, nil
			}
			legacy, err := (Engine{
				ResolutionProviderFactory:    factory,
				disableAdaptiveOptimizations: true,
			}).RunWithResolution(context.Background(), d, strategy, config, ResolutionModeAdaptive, nil)
			if err != nil {
				t.Fatal(err)
			}
			fast, err := (Engine{
				ResolutionProviderFactory: factory,
			}).RunWithResolution(context.Background(), d, strategy, config, ResolutionModeAdaptive, nil)
			if err != nil {
				t.Fatal(err)
			}
			left, _ := json.Marshal(legacy)
			right, _ := json.Marshal(fast)
			if string(left) != string(right) {
				t.Fatalf("optimized indicators changed full result: legacy=%s optimized=%s", left, right)
			}
		})
	}
}

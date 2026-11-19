package backtest

import (
	"context"
	"math"
	"reflect"
	"testing"

	"go_binance_futures/service/historicalmarket"
)

// Regression: a cached forming [0] kline must refresh the taker fields as
// frequently as price/volume fields, not retain the first minute's values.
func TestBT1CachedCurrentKlineTakerFieldsMatchLegacy(t *testing.T) {
	dataset := cacheFixtureDataset(240)
	minuteKey := BarSeriesKey(dataset.Symbol, "1m")
	hourKey := BarSeriesKey(dataset.Symbol, "1h")
	bars := dataset.Bars[minuteKey]
	for i := range bars {
		bars[i].TakerBuyQuoteVolume = bars[i].QuoteVolume * float64(i%7) / 10
	}
	for h := range dataset.Bars[hourKey] {
		parts := bars[h*60 : (h+1)*60]
		dataset.Bars[hourKey][h] = aggregatePartialBars(dataset.Symbol, "1h", parts[0].OpenTime, parts[len(parts)-1].CloseTime, parts)
	}
	legacy, err := newHistoricalEnvironment(dataset, "{}")
	if err != nil {
		t.Fatal(err)
	}
	optimized, err := newHistoricalEnvironment(dataset, "{}")
	if err != nil {
		t.Fatal(err)
	}
	optimized.enableStandardOptimizations("[]")
	for i := 62; i < 105; i++ {
		bar := bars[i]
		if _, _, err := legacy.BuildMinuteClose(bar.CloseTime, bar, nil, 1000, zeroCosts()); err != nil {
			t.Fatalf("legacy minute=%d: %v", i, err)
		}
		if _, _, err := optimized.BuildMinuteClose(bar.CloseTime, bar, nil, 1000, zeroCosts()); err != nil {
			t.Fatalf("optimized minute=%d: %v", i, err)
		}
		legacy.overlays = legacy.minuteCloseOverlays
		optimized.overlays = optimized.minuteCloseOverlays
		want := legacy.klinePriceSeries(dataset.Symbol, "1h", bar.CloseTime, legacy.warmupBars)
		got := optimized.cachedKlinePriceSeries("1h", bar.CloseTime)
		if !reflect.DeepEqual(want.TakerBuyAmount, got.TakerBuyAmount) ||
			!reflect.DeepEqual(want.TakerBuyRatio, got.TakerBuyRatio) {
			t.Fatalf("minute=%d taker cache differs: want amount=%v ratio=%v, got amount=%v ratio=%v",
				i, want.TakerBuyAmount, want.TakerBuyRatio, got.TakerBuyAmount, got.TakerBuyRatio)
		}
		if len(got.TakerBuyRatio) == 0 || math.IsNaN(got.TakerBuyRatio[0]) {
			t.Fatalf("minute=%d missing/invalid forming taker ratio", i)
		}
		legacy.overlays = nil
		optimized.overlays = nil
	}
}

// Both stop and take-profit are reachable in the same 1s candle. Its close
// cannot decide which happened first: chronological trades must win.
func TestBT1AdaptiveSameSecondStopBeforeTakeProfit(t *testing.T) {
	d := fixtureDataset([]float64{100, 100.2, 100.2, 100.2})
	bars := d.Bars[BarSeriesKey(d.Symbol, "1m")]
	bars[1].High, bars[1].Low = 101.5, 98.4
	d.DataHash = DatasetDataHash(d)
	start := bars[1].OpenTime
	provider := &intrabarFixtureProvider{
		seconds: []historicalmarket.Kline{{
			Market: d.Market, Symbol: d.Symbol, Interval: "1s",
			OpenTime: start, CloseTime: start + 999,
			Open: 100, High: 101.5, Low: 98.4, Close: 101.2,
		}},
		trades: []historicalmarket.PublicDataTrade{
			{TradeID: 3, TradeTime: start + 800, Price: 101.2, Quantity: 1, QuoteQuantity: 101.2},
			{TradeID: 2, TradeTime: start + 300, Price: 98.5, Quantity: 1, QuoteQuantity: 98.5},
			{TradeID: 1, TradeTime: start + 100, Price: 100, Quantity: 1, QuoteQuantity: 100},
		},
	}
	engine := Engine{ResolutionProviderFactory: func() (historicalmarket.ResolutionProvider, error) {
		return provider, nil
	}}
	strategy := StrategySnapshot{
		TemplateID: 1, TemplateName: "same-second-order", TechnologyJSON: "{}",
		StrategyJSON: strategyJSON(
			Rule{Name: "open", Enable: true, Type: "long", Code: "NowPrice >= 100"},
			Rule{Name: "close", Enable: true, Type: "close_long", Code: "ROI >= 10 || ROI <= -10"},
		),
	}
	config := RunConfig{InitialEquity: 1000, PositionSizePct: 1, Leverage: 10, StopLossPct: 10, TakeProfitPct: 10}
	result, err := engine.RunWithResolution(context.Background(), d, strategy, config, ResolutionModeAdaptive, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Trades) == 0 {
		t.Fatalf("expected at least one trade: %+v", result.Trades)
	}
	trade := result.Trades[0]
	if trade.ExitReason != "stop_loss" || trade.ExitResolution != "trades" ||
		trade.ExitTime != start+300 || math.Abs(trade.ExitPrice-98.5) > 1e-9 {
		t.Fatalf("1s close must not override earlier stop: %+v", trade)
	}
	if provider.secondCalls != 1 || provider.tradeCalls != 1 {
		t.Fatalf("expected same-second drill-down: 1s=%d trades=%d", provider.secondCalls, provider.tradeCalls)
	}
}

func TestBT1CachedTakerRatioResetsOnZeroVolume(t *testing.T) {
	dataset := cacheFixtureDataset(180)
	builder, err := newHistoricalEnvironment(dataset, "{}")
	if err != nil {
		t.Fatal(err)
	}
	builder.enableStandardOptimizations("[]")
	bar := dataset.Bars[BarSeriesKey(dataset.Symbol, "1m")][100]
	if _, _, err := builder.BuildMinuteClose(bar.CloseTime, bar, nil, 1000, zeroCosts()); err != nil {
		t.Fatal(err)
	}
	builder.overlays = builder.minuteCloseOverlays
	_ = builder.cachedKlinePriceSeries("1h", bar.CloseTime)
	key := BarSeriesKey(dataset.Symbol, "1h")
	zero := builder.overlays[key]
	zero.QuoteVolume = 0
	zero.TakerBuyQuoteVolume = 0
	builder.overlays[key] = zero
	got := builder.cachedKlinePriceSeries("1h", bar.CloseTime)
	if got.TakerBuyAmount[0] != 0 || got.TakerBuyRatio[0] != 0 || got.Qps[0] != 0 {
		t.Fatalf("forming taker fields must clear on zero volume: amount=%v ratio=%v qps=%v",
			got.TakerBuyAmount[0], got.TakerBuyRatio[0], got.Qps[0])
	}
}

func TestBT1AdaptiveSameSecondShortStopBeforeTakeProfit(t *testing.T) {
	d := fixtureDataset([]float64{100, 99.8, 99.8, 99.8})
	bars := d.Bars[BarSeriesKey(d.Symbol, "1m")]
	bars[1].High, bars[1].Low = 101.5, 98.4
	d.DataHash = DatasetDataHash(d)
	start := bars[1].OpenTime
	provider := &intrabarFixtureProvider{
		seconds: []historicalmarket.Kline{{
			Market: d.Market, Symbol: d.Symbol, Interval: "1s",
			OpenTime: start, CloseTime: start + 999,
			Open: 100, High: 101.5, Low: 98.4, Close: 98.5,
		}},
		trades: []historicalmarket.PublicDataTrade{
			{TradeID: 3, TradeTime: start + 800, Price: 98.5, Quantity: 1, QuoteQuantity: 98.5},
			{TradeID: 1, TradeTime: start + 200, Price: 101.5, Quantity: 1, QuoteQuantity: 101.5},
		},
	}
	engine := Engine{ResolutionProviderFactory: func() (historicalmarket.ResolutionProvider, error) {
		return provider, nil
	}}
	strategy := StrategySnapshot{
		TemplateID: 1, TemplateName: "short-same-second-order", TechnologyJSON: "{}",
		StrategyJSON: strategyJSON(
			Rule{Name: "open", Enable: true, Type: "short", Code: "NowPrice <= 100"},
			Rule{Name: "close", Enable: true, Type: "close_short", Code: "ROI >= 10 || ROI <= -10"},
		),
	}
	config := RunConfig{InitialEquity: 1000, PositionSizePct: 1, Leverage: 10, StopLossPct: 10, TakeProfitPct: 10}
	result, err := engine.RunWithResolution(context.Background(), d, strategy, config, ResolutionModeAdaptive, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Trades) == 0 {
		t.Fatal("missing short trade")
	}
	trade := result.Trades[0]
	if trade.Side != "SHORT" || trade.ExitReason != "stop_loss" ||
		trade.ExitResolution != "trades" || trade.ExitTime != start+200 ||
		math.Abs(trade.ExitPrice-101.5) > 1e-9 {
		t.Fatalf("short stop must win over later take-profit: %+v", trade)
	}
	if provider.secondCalls != 1 || provider.tradeCalls != 1 {
		t.Fatalf("expected same-second drill-down: 1s=%d trades=%d", provider.secondCalls, provider.tradeCalls)
	}
}

func TestBT1AdaptiveRejectsAmbiguousSecondWithoutTrades(t *testing.T) {
	d := fixtureDataset([]float64{100, 100.2, 100.2})
	bars := d.Bars[BarSeriesKey(d.Symbol, "1m")]
	bars[1].High, bars[1].Low = 101.5, 98.4
	start := bars[1].OpenTime
	provider := &intrabarFixtureProvider{
		seconds: []historicalmarket.Kline{{
			Market: d.Market, Symbol: d.Symbol, Interval: "1s",
			OpenTime: start, CloseTime: start + 999,
			Open: 100, High: 101.5, Low: 98.4, Close: 101.2,
		}},
	}
	engine := Engine{ResolutionProviderFactory: func() (historicalmarket.ResolutionProvider, error) {
		return provider, nil
	}}
	strategy := StrategySnapshot{
		TemplateID: 1, TemplateName: "missing-trades", TechnologyJSON: "{}",
		StrategyJSON: strategyJSON(
			Rule{Name: "open", Enable: true, Type: "long", Code: "NowPrice >= 100"},
			Rule{Name: "close", Enable: true, Type: "close_long", Code: "ROI >= 10 || ROI <= -10"},
		),
	}
	config := RunConfig{InitialEquity: 1000, PositionSizePct: 1, Leverage: 10, StopLossPct: 10, TakeProfitPct: 10}
	_, err := engine.RunWithResolution(context.Background(), d, strategy, config, ResolutionModeAdaptive, nil)
	if err == nil {
		t.Fatal("ambiguous TP/SL order without trades must not silently use 1s close")
	}
	if provider.tradeCalls != 1 {
		t.Fatalf("expected one strict trade lookup, got %d", provider.tradeCalls)
	}
}

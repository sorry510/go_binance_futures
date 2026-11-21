package backtest

import (
	"context"
	"encoding/json"
	"math"
	"reflect"
	"testing"

	"go_binance_futures/feature/strategy/line"
	"go_binance_futures/service/historicalmarket"

	"github.com/expr-lang/expr/vm"
)

// This tests the newly whitelisted dynamic dependency pattern. Unsupported
// fields fall back to the existing per-trade BuildIntrabar implementation.
func TestAdaptiveFormingCloseDependencyWhitelist(t *testing.T) {
	good := []Rule{
		{Name: "dynamic-long", Type: "close_long", Enable: true,
			Code: "let atr = atr_1h_3.Data[1]; let gate = Position.EntryPrice > 0 ? 6 : 10; ROI >= gate && kline_1h.Close[0] < kline_1h.Close[1] - atr * 0.2"},
		{Name: "short", Type: "close_short", Enable: true,
			Code: "ROI <= -6 && kline_1h.Close[0] > kline_1h.Close[1] && Position.EntryPrice > 0"},
	}
	if !canReuseFormingCloseEnvironment(good, "LONG") ||
		!canReuseFormingCloseEnvironment(good, "SHORT") {
		t.Fatal("legitimate ROI + EntryPrice + forming close pattern not recognized")
	}
	bad := []string{
		"ROI >= 8 && Position.MarkPrice > 0 && kline_1h.Close[0] > 0",
		"ROI >= 8 && Positions[0].MarkPrice > 0 && kline_1h.Close[0] > 0",
		"ROI >= 8 && kline_1h.High[0] > 0 && kline_1h.Close[0] > 0",
		"ROI >= 8 && rsi_1h_3.Data[0] > 0 && kline_1h.Close[0] > 0",
		"ROI >= 8 && kline_1h.Close[idx] > 0 && kline_1h.Close[0] > 0",
		"ROI >= 8 && kline_1h.Close[0:2] > 0",
		"ROI >= 8 && kline_1h.Close > 0 && kline_1h.Close[0] > 0",
		"ROI >= 8 && NowPrice > 0 && kline_1h.Close[0] > 0",
		"ROI >= 8 && FundingRate.Data[1] > 0 && kline_1h.Close[0] > 0",
		"ROI >= 8 && MarketCondition > 0 && kline_1h.Close[0] > 0",
		"ROI >= 8 && NetROI > 8 && kline_1h.Close[0] > 0",
		"ROI >= 8 && kline_4h.Close[0] > 0 && kline_1h.Close[0] > 0",
		"ROI >= 8 && unknown.Close[1] > 0 && kline_1h.Close[0] > 0",
		"ROI >= 8 && IsAsc(kline_1h.Close) && kline_1h.Close[0] > 0",
		"ROI >= 8 && kline_1h.Close[0] > 0 && \"Position.MarkPrice\" == \"\"",
	}
	for _, code := range bad {
		if canReuseFormingCloseEnvironment([]Rule{{Name: "unsafe", Type: "close_long", Enable: true, Code: code}}, "LONG") {
			t.Errorf("unsafe code admitted: %s", code)
		}
	}
	if canReuseFormingCloseEnvironment([]Rule{{Type: "close_long", Enable: true, Code: "ROI > 8 && atr_1h_3.Data[1] > 0"}}, "LONG") {
		t.Fatal("a rule without forming close does not need this path")
	}
	if canReuseFormingCloseEnvironment([]Rule{
		good[0],
		{Type: "close_long", Enable: true, Code: "NowTime > 0"},
	}, "LONG") {
		t.Fatal("all enabled CLOSE rules must qualify")
	}
}

func TestAdaptiveFormingCloseReusedEnvMatchesEveryLegacyTick(t *testing.T) {
	dataset := cacheFixtureDataset(4800)
	rule := Rule{Name: "forming", Type: "close_long", Enable: true,
		Code: "let atr = atr_1h_3.Data[1]; let gate = Position.EntryPrice > 0 ? 6 : 10; ROI >= gate && kline_1h.Close[0] < kline_1h.Close[1] - atr * 0.2"}
	if !canReuseFormingCloseEnvironment([]Rule{rule}, "LONG") {
		t.Fatal("test rule should activate forming-close cache")
	}
	minutes := dataset.Bars[BarSeriesKey(dataset.Symbol, "1m")]
	position := &Position{Side: "LONG", EntryPrice: minutes[4180].Open, Quantity: .05, EntryTime: minutes[4180].OpenTime}
	config := RunConfig{InitialEquity: 1000, PositionSizePct: 1, Leverage: 4, FeeRate: .0005}
	optimized, err := newHistoricalEnvironment(dataset, cacheTechnologyJSON())
	if err != nil {
		t.Fatal(err)
	}
	optimized.enableStandardOptimizations(strategyJSON(rule))
	optimized.adaptiveOptimized = true
	legacy, err := newHistoricalEnvironment(dataset, cacheTechnologyJSON())
	if err != nil {
		t.Fatal(err)
	}
	for _, minuteIndex := range []int{4150, 4199, 4200, 4230, 4201, 4250} {
		minute := minutes[minuteIndex]
		stable, _, err := optimized.BuildIntrabar(minute.CloseTime, minute, position, 1000, config)
		if err != nil {
			t.Fatal(err)
		}
		if !prepareFormingCloseEnvironment(stable) {
			t.Fatal("missing forming 1h series")
		}
		snapshot := stable["kline_1h"].(line.KLinePrice)
		staticClose := append([]float64(nil), snapshot.Close...)
		for _, tick := range []int64{999, 7599, 18999, 41999, 59999} {
			partial := minute
			partial.CloseTime = minute.OpenTime + tick
			partial.Close = minute.Open + float64(tick%30000)/20000 - 0.6
			partial.High = math.Max(partial.Open, partial.Close) + 1
			partial.Low = math.Min(partial.Open, partial.Close) - 1
			partial.QuoteVolume = float64(tick) / 1000
			partial.TakerBuyQuoteVolume = partial.QuoteVolume * 0.31
			full, _, err := legacy.BuildIntrabar(partial.CloseTime, partial, position, 1000, config)
			if err != nil {
				t.Fatal(err)
			}
			updateFormingCloseEnvironment(stable, partial.Close)
			stable["ROI"] = grossROI(position, partial.Close, config.Leverage)
			gotCandle := stable["kline_1h"].(line.KLinePrice)
			wantCandle := full["kline_1h"].(line.KLinePrice)
			if gotCandle.Close[0] != wantCandle.Close[0] {
				t.Fatalf("minute=%d tick=%d current 1h Close changed %.8f vs %.8f", minuteIndex, tick, gotCandle.Close[0], wantCandle.Close[0])
			}
			if !reflect.DeepEqual(gotCandle.Close[1:], wantCandle.Close[1:]) {
				t.Fatalf("minute=%d tick=%d completed 1h Close changed", minuteIndex, tick)
			}
			lr, lp, le := evaluateRules([]Rule{rule}, "LONG", full, map[int]*vm.Program{}, true)
			rr, rp, re := evaluateRules([]Rule{rule}, "LONG", stable, map[int]*vm.Program{}, true)
			if le != nil || re != nil {
				t.Fatalf("evaluation error old=%v fast=%v", le, re)
			}
			if lp != rp || (lp && lr.Code != rr.Code) {
				t.Fatalf("minute=%d tick=%d legacy=%v optimized=%v", minuteIndex, tick, lp, rp)
			}
		}
		// Current values must be private to this run's stable preview, not
		// an in-place change of source/cache backing arrays.
		if len(staticClose) > 1 && !reflect.DeepEqual(staticClose[1:], snapshot.Close[1:]) {
			t.Fatal("completed 1h history was changed by fast path")
		}
	}
}

// Full adaptive Engine comparison uses a SHORT whose first profitable tick
// occurs within a second that closes unprofitably. This forces genuine Trades
// and dynamic forming-close evaluation, not a one-second shortcut.
func TestAdaptiveFormingCloseFullEngineTradeParity(t *testing.T) {
	d := cacheFixtureDataset(720)
	bars := d.Bars[BarSeriesKey(d.Symbol, "1m")]
	d.StartTime, d.EndTime = bars[600].CloseTime, bars[605].CloseTime
	entry := bars[601].Open
	bars[601].High, bars[601].Low, bars[601].Close = entry+.01, entry-.01, entry
	bars[602].Open, bars[602].High, bars[602].Low = entry, entry+1, entry-1
	d.DatasetSpecHash, d.DataHash = DatasetSpecHash(d), DatasetDataHash(d)
	secondStart := bars[602].OpenTime + 1000
	providerFactory := func() (historicalmarket.ResolutionProvider, error) {
		return &intrabarFixtureProvider{
			seconds: []historicalmarket.Kline{{
				Market: d.Market, Symbol: d.Symbol, Interval: "1s",
				OpenTime: secondStart, CloseTime: secondStart + 999,
				Open: entry, High: entry + .1, Low: entry - .8, Close: entry + .05,
			}},
			trades: []historicalmarket.PublicDataTrade{
				{Market: d.Market, Symbol: d.Symbol, TradeTime: secondStart + 100, TradeID: 1, Price: entry - .8, Quantity: .001},
				{Market: d.Market, Symbol: d.Symbol, TradeTime: secondStart + 500, TradeID: 2, Price: entry + .05, Quantity: .001},
			},
		}, nil
	}
	closeRule := Rule{Name: "dynamic-short", Type: "close_short", Enable: true,
		Code: "let threshold = Position.EntryPrice > 0 ? 1 : 10; ROI >= threshold && kline_1h.Close[0] < Position.EntryPrice && atr_1h_3.Data[1] > 0"}
	strategy := StrategySnapshot{TemplateID: 1, TemplateName: "forming-close-full-trade", TechnologyJSON: cacheTechnologyJSON(),
		StrategyJSON: strategyJSON(
			Rule{Name: "enter", Type: "short", Enable: true, Code: "NowPrice > 0"},
			closeRule,
		),
	}
	config := RunConfig{InitialEquity: 1000, PositionSizePct: 1, Leverage: 4, TakeProfitPct: 1, StopLossPct: 3}
	before, err := (Engine{ResolutionProviderFactory: providerFactory, disableAdaptiveOptimizations: true}).RunWithResolution(context.Background(), d, strategy, config, ResolutionModeAdaptive, nil)
	if err != nil {
		t.Fatal(err)
	}
	after, err := (Engine{ResolutionProviderFactory: providerFactory}).RunWithResolution(context.Background(), d, strategy, config, ResolutionModeAdaptive, nil)
	if err != nil {
		t.Fatal(err)
	}
	if before.ResolutionStats.TradeDrilldownSeconds == 0 {
		t.Fatal("fixture never invoked trade-level replay")
	}
	if after.ResolutionStats.TradeEnvBuilds != 0 {
		t.Fatalf("safe forming close still rebuilt environment %d times", after.ResolutionStats.TradeEnvBuilds)
	}
	// New diagnostics differ by design; all trade and financial outputs,
	// including exact historical evidence hash, must be byte-identical.
	before.ResolutionStats.TradeEnvBuilds = 0
	left, _ := json.Marshal(before)
	right, _ := json.Marshal(after)
	if string(left) != string(right) {
		t.Fatalf("fast forming-close changed adaptive replay: before=%s after=%s", left, right)
	}
}

// The actual BTC close rules use completed indicators, EntryPrice and forming
// 1h.Close[0]. This measures the environment hot loop before and after the
// strict dependency whitelist, excluding archive and SQL I/O.
func benchmarkAdaptiveFormingClose(b *testing.B, reuse bool) {
	dataset := cacheFixtureDataset(4800)
	minutes := dataset.Bars[BarSeriesKey(dataset.Symbol, "1m")]
	rule := Rule{Type: "close_long", Enable: true, Name: "dynamic",
		Code: "let threshold = Position.EntryPrice > 0 ? 8 : 12; ROI >= threshold && kline_1h.Close[0] <= kline_1h.Close[1] && atr_1h_3.Data[1] > 0"}
	raw := strategyJSON(rule)
	env, err := newHistoricalEnvironment(dataset, cacheTechnologyJSON())
	if err != nil {
		b.Fatal(err)
	}
	env.enableStandardOptimizations(raw)
	env.adaptiveOptimized = true
	minute := minutes[4200]
	pos := &Position{Side: "LONG", EntryPrice: minute.Open, Quantity: .05}
	config := RunConfig{InitialEquity: 1000, Leverage: 4}
	compiled := map[int]*vm.Program{}
	var stable map[string]interface{}
	if reuse {
		stable, _, err = env.BuildIntrabar(minute.CloseTime, minute, pos, 1000, config)
		if err != nil || !prepareFormingCloseEnvironment(stable) {
			b.Fatalf("prepare stable env: %v", err)
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		partial := minute
		partial.CloseTime = minute.OpenTime + int64(i%60)*1000 + 999
		partial.Close = minute.Open + float64(i%60)*.05
		partial.High = math.Max(partial.Open, partial.Close) + .7
		partial.Low = math.Min(partial.Open, partial.Close) - .9
		var values map[string]interface{}
		if reuse {
			values = stable
			updateFormingCloseEnvironment(values, partial.Close)
			values["ROI"] = grossROI(pos, partial.Close, config.Leverage)
		} else {
			values, _, err = env.BuildIntrabar(partial.CloseTime, partial, pos, 1000, config)
			if err != nil {
				b.Fatal(err)
			}
		}
		if _, _, err = evaluateRules([]Rule{rule}, "LONG", values, compiled, true); err != nil {
			b.Fatal(err)
		}
	}
}
func BenchmarkAdaptiveFormingCloseFullBuild(b *testing.B)    { benchmarkAdaptiveFormingClose(b, false) }
func BenchmarkAdaptiveFormingCloseStableUpdate(b *testing.B) { benchmarkAdaptiveFormingClose(b, true) }

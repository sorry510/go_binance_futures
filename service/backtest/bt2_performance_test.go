package backtest

import (
	"testing"
	"time"
)

// Synthetic in-memory benchmarks: no DB access, network, or persistence.
func benchmarkBT2IntrabarIndicators(b *testing.B, optimized bool) {
	dataset := cacheFixtureDataset(4800)
	env, err := newHistoricalEnvironment(dataset, cacheTechnologyJSON())
	if err != nil {
		b.Fatal(err)
	}
	if optimized {
		rules := []Rule{{Enable: true, Type: "close_long", Code: "ema_1h_3.Data[1] > 0 && rsi_1h_3.Data[1] > 0 && adx_1h_3.ADX[0] > 0 && atr_1h_3.Data[1] > 0 && dc_1h_3.High[1] > 0"}}
		env.enableStandardOptimizations(strategyJSON(rules...))
		env.adaptiveOptimized = true
	}
	bars := dataset.Bars[BarSeriesKey(dataset.Symbol, "1m")]
	minute := bars[4200]
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		n := i % 60
		partial := minute
		partial.CloseTime = minute.OpenTime + int64(n+1)*time.Second.Milliseconds() - 1
		partial.High = minute.Open + 1 + float64(n)*0.01
		partial.Low = minute.Open - 1
		partial.Close = minute.Open + float64(n)*0.01
		partial.QuoteVolume = 100 + float64(n)
		if _, _, err := env.BuildIntrabar(partial.CloseTime, partial, nil, 1000, zeroCosts()); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBT2IntrabarIndicators(b *testing.B) {
	benchmarkBT2IntrabarIndicators(b, true)
}

func BenchmarkBT2IntrabarLegacy(b *testing.B) {
	benchmarkBT2IntrabarIndicators(b, false)
}

func BenchmarkBT2CachedRuleROIProfile(b *testing.B) {
	rules := []Rule{{Name: "close", Type: "close_long", Enable: true, Code: "ROI >= 8 && kline_1h.Close[1] > 1"}}
	state := adaptiveRunState{}
	state.prepareCloseProfiles(rules)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if !state.closeProfile(rules, "LONG").Eligible {
			b.Fatal("invalid cached ROI profile")
		}
	}
}

func BenchmarkBT2RuleROIAnalysis(b *testing.B) {
	rules := []Rule{
		{Name: "stop", Enable: true, Type: "close_long", Code: "ROI >= 8 && kline_1h.Close[1] > 1"},
		{Name: "loss", Enable: true, Type: "close_long", Code: "ROI <= -6 && kline_1h.Close[1] > 1"},
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if !analyzeCloseRuleROIProfile(rules, "LONG").Eligible {
			b.Fatal("invalid benchmark profile")
		}
	}
}

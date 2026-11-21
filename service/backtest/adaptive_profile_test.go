package backtest

import (
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/expr-lang/expr/vm"
)

// Legacy behavior, kept only in tests for an independent parity oracle.
func linearFundingInRange(funding []Funding, start, end int64) bool {
	for _, f := range funding {
		if f.FundingTime < start {
			continue
		}
		if f.FundingTime > end {
			return false
		}
		return true
	}
	return false
}

func TestProfileFundingRangeMatchesOriginal(t *testing.T) {
	var funding []Funding
	start := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	for i := 0; i < 10_000; i++ {
		funding = append(funding, Funding{FundingTime: start + int64(i)*8*3_600_000})
	}
	for _, slice := range [][]Funding{nil, funding[:1], funding[:123], funding} {
		for i := 0; i < 10_000; i++ {
			a := start + int64(i)*7*3_600_000 + int64(i%7)*999
			b := a + int64(i%500)*1_000
			want, got := linearFundingInRange(slice, a, b), hasFundingInRange(slice, a, b)
			if want != got {
				t.Fatalf("range=%d..%d count=%d old=%t new=%t", a, b, len(slice), want, got)
			}
		}
	}
}

func BenchmarkBTProfileFundingScan(b *testing.B) {
	start := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	funding := make([]Funding, 3_800)
	for i := range funding {
		funding[i] = Funding{FundingTime: start + int64(i)*8*3_600_000}
	}
	b.ReportAllocs()
	b.ResetTimer()
	var hits int
	for i := 0; i < b.N; i++ {
		a := start + int64(i%3_800)*8*3_600_000 + 60_000
		if hasFundingInRange(funding, a, a+999) {
			hits++
		}
	}
	if hits > b.N {
		b.Fatal(hits)
	}
}

// Compare every price statistic against the legacy 24h scan. Exact equality
// is required so floating-point crossing boundaries cannot move trade signals.
func TestProfileIntrabarTickerMatchesLegacy(t *testing.T) {
	dataset := cacheFixtureDataset(4_800)
	env, err := newHistoricalEnvironment(dataset, cacheTechnologyJSON())
	if err != nil {
		t.Fatal(err)
	}
	env.enableStandardOptimizations(strategyJSON(Rule{Enable: true, Type: "close_long", Code: "kline_1h.Close[0] > 0 && ROI > 8"}))
	env.adaptiveOptimized = true
	minutes := dataset.Bars[BarSeriesKey(dataset.Symbol, "1m")]
	for _, n := range []int{5, 59, 60, 1390, 1440, 1441, 1450, 2100, 3600, 4200, 4199, 4250} {
		minute := minutes[n]
		for _, ms := range []int64{999, 5999, 30_999, 59_999} {
			partial := minute
			partial.CloseTime = minute.OpenTime + ms
			partial.Close = minute.Open + float64(ms)/20_000
			partial.High = math.Max(partial.Open, partial.Close) + .7
			partial.Low = math.Min(partial.Open, partial.Close) - .9
			key := BarSeriesKey(dataset.Symbol, "1m")
			env.overlays = map[string]Bar{key: partial}
			want := env.tickerStats(dataset.Symbol, partial.CloseTime)
			env.intrabarTickerCache.prepare(minutes, partial, partial.CloseTime)
			got := env.intrabarTickerCache.values(partial)
			if !reflect.DeepEqual(want, got) {
				t.Fatalf("minute=%d partial=%d legacy=%#v optimized=%#v", n, ms, want, got)
			}
		}
	}
	// Unusual canonical bars may expire inside a minute. Verify the cached
	// window rebuilds based on actual timestamps and not just minuteOpen.
	irregular := []Bar{
		{OpenTime: 10, CloseTime: 11, Open: 1, High: 6, Low: .4, Close: 2},
		{OpenTime: 20, CloseTime: 21, Open: 2, High: 5, Low: 1, Close: 3},
	}
	partial := Bar{OpenTime: 30, CloseTime: 31, Open: 3, High: 4, Low: 2, Close: 4}
	var snapshot intrabarTickerSnapshot
	snapshot.prepare(irregular, partial, 31)
	first := snapshot.start
	snapshot.prepare(irregular, partial, 12+24*int64(time.Hour/time.Millisecond))
	if snapshot.start == first && snapshot.end == len(irregular) {
		t.Fatal("cache ignored moving 24h cutoff")
	}
}

// BenchmarkBTProfileDynamicClose models the actual BTC strategy's indicator
// groups (1h/15m/4h) and dynamic Position plus current forming 1h [0] rule.
// It benchmarks close-environment rebuild + expression execution, rather than
// disk/network I/O.
func BenchmarkBTProfileDynamicClose(b *testing.B) {
	const tech = `{"ema":[{"name":"ema_1h_20","enable":true,"kline_interval":"1h","period":20}],"adx":[{"name":"adx_1h_14","enable":true,"kline_interval":"1h","period":14}],"mfi":[{"name":"mfi_1h_14","enable":true,"kline_interval":"1h","period":14}],"rsi":[{"name":"rsi_1h_14","enable":true,"kline_interval":"1h","period":14}],"boll":[{"name":"boll_1h_20_2","enable":true,"kline_interval":"1h","period":20,"std_dev_multiplier":2}],"donchian":[{"name":"donchian_1h_14","enable":true,"kline_interval":"1h","period":14}],"atr":[{"name":"atr_1h_14","enable":true,"kline_interval":"1h","period":14},{"name":"atr_15m_14","enable":true,"kline_interval":"15m","period":14}],"supertrend":[{"name":"supertrend_4h_10_3","enable":true,"kline_interval":"4h","period":10,"multiplier":3}]}`
	dataset := cacheFixtureDataset(50 * 24 * 60)
	minutes := dataset.Bars[BarSeriesKey(dataset.Symbol, "1m")]
	for _, item := range []struct {
		name string
		step int
	}{{"15m", 15}, {"4h", 240}} {
		parts := make([]Bar, 0, len(minutes)/item.step)
		for offset := 0; offset+item.step <= len(minutes); offset += item.step {
			window := minutes[offset : offset+item.step]
			parts = append(parts, aggregatePartialBars(dataset.Symbol, item.name, window[0].OpenTime, window[len(window)-1].CloseTime, window))
		}
		dataset.Bars[BarSeriesKey(dataset.Symbol, item.name)] = parts
		dataset.Intervals = append(dataset.Intervals, item.name)
	}
	rule := Rule{Name: "dynamic_close", Enable: true, Type: "close_long",
		Code: "Position.EntryPrice > 0 && kline_1h.Close[0] > 0 && atr_1h_14.Data[1] > 0 && ROI >= 8"}
	raw := strategyJSON(rule)
	env, err := newHistoricalEnvironment(dataset, tech)
	if err != nil {
		b.Fatal(err)
	}
	env.enableStandardOptimizations(raw)
	env.adaptiveOptimized = true
	minute := minutes[60_000]
	position := &Position{Side: "LONG", EntryTime: minutes[58_000].OpenTime, EntryPrice: minute.Open, Quantity: .05}
	config := RunConfig{InitialEquity: 1000, PositionSizePct: 1, Leverage: 4, FeeRate: .0005}
	compiled := map[int]*vm.Program{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		n := int64(i % 60)
		partial := minute
		partial.CloseTime = minute.OpenTime + n*1000 + 999
		partial.Close = minute.Open + float64(n)*.05
		partial.High = math.Max(partial.Open, partial.Close) + .7
		partial.Low = math.Min(partial.Open, partial.Close) - .9
		values, _, err := env.BuildIntrabar(partial.CloseTime, partial, position, 1000, config)
		if err != nil {
			b.Fatal(err)
		}
		if _, _, err := evaluateRules([]Rule{rule}, "LONG", values, compiled, true); err != nil {
			b.Fatal(err)
		}
	}
}

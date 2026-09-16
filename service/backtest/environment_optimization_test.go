package backtest

import (
	"reflect"
	"sort"
	"testing"
	"time"

	"go_binance_futures/feature/strategy/line"
)

func legacySeriesForTest(builder *historicalEnvironment, symbol, interval string, asOf int64, limit int) []Bar {
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

func legacyTickerStatsForTest(builder *historicalEnvironment, symbol string, asOf int64) map[string]interface{} {
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

func legacyKLinePriceForTest(bars []Bar) line.KLinePrice {
	p := line.KLinePrice{}
	for _, b := range bars {
		p.High = append(p.High, b.High)
		p.Low = append(p.Low, b.Low)
		p.Close = append(p.Close, b.Close)
		p.Open = append(p.Open, b.Open)
		p.Amount = append(p.Amount, b.QuoteVolume)
		seconds := klineQPSDurationSeconds(b)
		if seconds > 0 {
			p.Qps = append(p.Qps, b.QuoteVolume/seconds)
		} else {
			p.Qps = append(p.Qps, 0)
		}
	}
	return p
}

func optimizationFixtureBars(count int) []Bar {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	out := make([]Bar, 0, count)
	for i := 0; i < count; i++ {
		openTime := start + int64(i)*time.Minute.Milliseconds()
		base := 100.0 + float64(i)/10
		out = append(out, Bar{Symbol: "BTCUSDT", Interval: "1m", OpenTime: openTime, CloseTime: openTime + time.Minute.Milliseconds() - 1, Open: base, High: base + 2, Low: base - 2, Close: base + 0.5, QuoteVolume: 1000 + float64(i), Volume: 10, TradeCount: int64(100 + i)})
	}
	return out
}

func TestOptimizedEnvironmentHelpersMatchLegacy(t *testing.T) {
	bars := optimizationFixtureBars(1800)
	key := BarSeriesKey("BTCUSDT", "1m")
	base := historicalEnvironment{dataset: Dataset{Symbol: "BTCUSDT", ExecutionInterval: "1m", Bars: map[string][]Bar{key: bars}}}
	asOf := bars[1700].CloseTime
	overlays := []map[string]Bar{
		nil,
		{key: {Symbol: "BTCUSDT", Interval: "1m", OpenTime: bars[1700].OpenTime, CloseTime: asOf, Open: bars[1700].Open, High: 999, Low: 1, Close: 777, QuoteVolume: 4321}},
		{key: {Symbol: "BTCUSDT", Interval: "1m", OpenTime: bars[1700].OpenTime + time.Minute.Milliseconds(), CloseTime: asOf, Open: 1, High: 2, Low: 1, Close: 2}},
		{key: {Symbol: "BTCUSDT", Interval: "1m", OpenTime: asOf + 1, CloseTime: asOf + 2, Open: 1, High: 2, Low: 1, Close: 2}},
	}
	for oi, overlaysCase := range overlays {
		builder := base
		builder.overlays = overlaysCase
		for _, limit := range []int{0, 1, 3, 200} {
			got, want := builder.series("BTCUSDT", "1m", asOf, limit), legacySeriesForTest(&builder, "BTCUSDT", "1m", asOf, limit)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("overlay=%d limit=%d series mismatch\ngot=%v\nwant=%v", oi, limit, got, want)
			}
			gotPrice := builder.klinePriceSeries("BTCUSDT", "1m", asOf, limit)
			wantPrice := legacyKLinePriceForTest(want)
			if !reflect.DeepEqual(gotPrice, wantPrice) {
				t.Fatalf("overlay=%d limit=%d KLinePrice series mismatch\ngot=%+v\nwant=%+v", oi, limit, gotPrice, wantPrice)
			}
		}
		if got, want := builder.tickerStats("BTCUSDT", asOf), legacyTickerStatsForTest(&builder, "BTCUSDT", asOf); !reflect.DeepEqual(got, want) {
			t.Fatalf("overlay=%d ticker mismatch\ngot=%v\nwant=%v", oi, got, want)
		}
	}
	sample := []Bar{bars[3], bars[2], bars[1]}
	if got, want := toKLinePrice(sample), legacyKLinePriceForTest(sample); !reflect.DeepEqual(got, want) {
		t.Fatalf("KLinePrice mismatch\ngot=%+v\nwant=%+v", got, want)
	}
}

func TestCurrentExecutionBarLimitOneMatchesLegacyWarmupHead(t *testing.T) {
	bars := optimizationFixtureBars(300)
	key := BarSeriesKey("BTCUSDT", "1m")
	asOf := bars[250].CloseTime
	cases := []map[string]Bar{
		nil,
		{key: {Symbol: "BTCUSDT", Interval: "1m", OpenTime: bars[250].OpenTime, CloseTime: asOf, Open: bars[250].Open, High: 999, Low: 1, Close: 777}},
		{key: {Symbol: "BTCUSDT", Interval: "1m", OpenTime: bars[250].OpenTime + time.Minute.Milliseconds(), CloseTime: asOf, Open: 777, High: 778, Low: 776, Close: 777.5}},
	}
	for i, overlays := range cases {
		builder := historicalEnvironment{warmupBars: 200, overlays: overlays, dataset: Dataset{Symbol: "BTCUSDT", ExecutionInterval: "1m", Bars: map[string][]Bar{key: bars}}}
		one := builder.series("BTCUSDT", "1m", asOf, 1)
		legacy := legacySeriesForTest(&builder, "BTCUSDT", "1m", asOf, builder.warmupBars)
		if len(one) != 1 || len(legacy) == 0 || !reflect.DeepEqual(one[0], legacy[0]) {
			t.Fatalf("case=%d current bar mismatch one=%v legacy=%v", i, one, legacy)
		}
	}
}

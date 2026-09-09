package backtest

import (
	"context"
	"encoding/json"
	"math"
	"testing"
	"time"

	"go_binance_futures/feature/strategy/line"
)

func fixtureDataset(closes []float64) Dataset {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	bars := make([]Bar, 0, len(closes))
	for i, close := range closes {
		open := close
		if i > 0 {
			open = closes[i-1]
		}
		at := start.Add(time.Duration(i) * time.Minute)
		high := math.Max(open, close) + 0.2
		low := math.Min(open, close) - 0.2
		bars = append(bars, Bar{Symbol: "BTCUSDT", Interval: "1m", OpenTime: at.UnixMilli(), CloseTime: at.Add(time.Minute - time.Millisecond).UnixMilli(), Open: open, High: high, Low: low, Close: close, QuoteVolume: 1000})
	}
	d := Dataset{Symbol: "BTCUSDT", ExecutionInterval: "1m", Intervals: []string{"1m"}, BenchmarkSymbols: append([]string(nil), BenchmarkSymbols...), StartTime: bars[0].CloseTime, EndTime: bars[len(bars)-1].CloseTime, WarmupStartTime: bars[0].OpenTime, Bars: map[string][]Bar{}, Funding: []Funding{}}
	for _, symbol := range BenchmarkSymbols {
		copyBars := make([]Bar, len(bars))
		for i, b := range bars {
			b.Symbol = symbol
			copyBars[i] = b
		}
		d.Bars[BarSeriesKey(symbol, "1m")] = copyBars
	}
	d.Market = "futures_usdt"
	d.DatasetSpecHash = DatasetSpecHash(d)
	d.DatasetID = "ds_" + d.DatasetSpecHash[:24]
	d.DataHash = DatasetDataHash(d)
	return d
}
func strategyJSON(rules ...Rule) string { raw, _ := json.Marshal(rules); return string(raw) }
func runFixture(t *testing.T, d Dataset, rules []Rule, c RunConfig) Result {
	t.Helper()
	r, err := (Engine{}).Run(context.Background(), d, StrategySnapshot{TemplateID: 1, TemplateName: "fixture", TechnologyJSON: "{}", StrategyJSON: strategyJSON(rules...)}, c)
	if err != nil {
		t.Fatal(err)
	}
	return r
}
func zeroCosts() RunConfig {
	return RunConfig{InitialEquity: 1000, PositionSizePct: 1, Leverage: 1, FeeRate: 0, SlippageBps: 0}
}
func closeEnough(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

func TestBacktestLongUsesNextBarOpen(t *testing.T) {
	d := fixtureDataset([]float64{100, 101, 104, 105})
	r := runFixture(t, d, []Rule{{Name: "open", Enable: true, Type: "long", Code: "NowPrice >= 100"}, {Name: "close", Enable: true, Type: "close_long", Code: "NowPrice >= 104"}}, zeroCosts())
	if len(r.Trades) != 1 {
		t.Fatalf("trades=%+v", r.Trades)
	}
	tr := r.Trades[0]
	if tr.Side != "LONG" || !closeEnough(tr.EntryPrice, 100) || !closeEnough(tr.ExitPrice, 104) {
		t.Fatalf("next-open semantics changed: %+v", tr)
	}
	if tr.EntryTime <= d.Bars[BarSeriesKey("BTCUSDT", "1m")][0].CloseTime {
		t.Fatalf("signal filled on same bar: %+v", tr)
	}
}
func TestBacktestShort(t *testing.T) {
	d := fixtureDataset([]float64{100, 99, 96, 95})
	r := runFixture(t, d, []Rule{{Name: "open", Enable: true, Type: "short", Code: "NowPrice >= 100"}, {Name: "close", Enable: true, Type: "close_short", Code: "NowPrice <= 96"}}, zeroCosts())
	if len(r.Trades) != 1 || r.Trades[0].Side != "SHORT" || r.Trades[0].NetPnL <= 0 {
		t.Fatalf("short failed: %+v", r.Trades)
	}
}
func TestBacktestTakeProfit(t *testing.T) {
	d := fixtureDataset([]float64{100, 100, 100})
	bars := d.Bars[BarSeriesKey("BTCUSDT", "1m")]
	bars[1].High = 103
	d.Bars[BarSeriesKey("BTCUSDT", "1m")] = bars
	r := runFixture(t, d, []Rule{{Name: "open", Enable: true, Type: "long", Code: "NowPrice >= 100"}}, RunConfig{InitialEquity: 1000, PositionSizePct: 1, Leverage: 1, TakeProfitPct: 2})
	if len(r.Trades) != 1 || r.Trades[0].ExitReason != "take_profit" {
		t.Fatalf("take profit failed: %+v", r.Trades)
	}
}
func TestBacktestStopLossWinsWhenStopAndTargetBothTouched(t *testing.T) {
	d := fixtureDataset([]float64{100, 100, 100})
	bars := d.Bars[BarSeriesKey("BTCUSDT", "1m")]
	bars[1].High = 103
	bars[1].Low = 97
	d.Bars[BarSeriesKey("BTCUSDT", "1m")] = bars
	r := runFixture(t, d, []Rule{{Name: "open", Enable: true, Type: "long", Code: "NowPrice >= 100"}}, RunConfig{InitialEquity: 1000, PositionSizePct: 1, Leverage: 1, StopLossPct: 2, TakeProfitPct: 2})
	if len(r.Trades) != 1 || r.Trades[0].ExitReason != "stop_loss" {
		t.Fatalf("protective precedence failed: %+v", r.Trades)
	}
}
func TestBacktestNoTrade(t *testing.T) {
	d := fixtureDataset([]float64{100, 101, 102})
	r := runFixture(t, d, []Rule{{Name: "never", Enable: true, Type: "long", Code: "NowPrice > 1000"}}, zeroCosts())
	if len(r.Trades) != 0 || r.Metrics.TradeCount != 0 || !closeEnough(r.Metrics.NetPnL, 0) {
		t.Fatalf("no-trade fixture changed: %+v", r)
	}
}
func TestBacktestFeesAndSlippage(t *testing.T) {
	d := fixtureDataset([]float64{100, 100, 110})
	c := RunConfig{InitialEquity: 1000, PositionSizePct: 1, Leverage: 1, FeeRate: 0.001, SlippageBps: 10}
	r := runFixture(t, d, []Rule{{Name: "open", Enable: true, Type: "long", Code: "NowPrice >= 100"}}, c)
	if len(r.Trades) != 1 {
		t.Fatal("missing trade")
	}
	tr := r.Trades[0]
	if !closeEnough(tr.EntryPrice, 100.1) || !closeEnough(tr.ExitPrice, 109.89) || tr.Fees <= 0 {
		t.Fatalf("cost model failed: %+v", tr)
	}
	if !closeEnough(r.Metrics.Fees, tr.Fees) {
		t.Fatalf("fee metric mismatch: %+v", r.Metrics)
	}
}
func TestBacktestFundingLongPaysPositiveRate(t *testing.T) {
	d := fixtureDataset([]float64{100, 100, 100, 100})
	bars := d.Bars[BarSeriesKey("BTCUSDT", "1m")]
	d.Funding = []Funding{{Symbol: "BTCUSDT", FundingTime: bars[1].CloseTime, FundingRate: 0.001, MarkPrice: 100}}
	r := runFixture(t, d, []Rule{{Name: "open", Enable: true, Type: "long", Code: "NowPrice >= 100"}}, zeroCosts())
	if len(r.Trades) != 1 || r.Trades[0].FundingPnL >= 0 || r.Metrics.Funding >= 0 {
		t.Fatalf("funding model failed: %+v", r.Trades)
	}
}
func TestHistoricalEnvironmentCannotSeeFutureBars(t *testing.T) {
	d := fixtureDataset([]float64{100, 999, 50})
	env, err := newHistoricalEnvironment(d, "{}")
	if err != nil {
		t.Fatal(err)
	}
	first := d.Bars[BarSeriesKey("BTCUSDT", "1m")][0]
	values, _, err := env.Build(first.CloseTime, nil, 1000, zeroCosts())
	if err != nil {
		t.Fatal(err)
	}
	if values["NowPrice"].(float64) != 100 {
		t.Fatalf("future price leaked: %#v", values["NowPrice"])
	}
	series := env.series("BTCUSDT", "1m", first.CloseTime, 200)
	if len(series) != 1 || series[0].Close != 100 {
		t.Fatalf("future bars visible: %+v", series)
	}
}
func TestHistoricalEnvironmentIndicatorOrderMatchesLive(t *testing.T) {
	d := fixtureDataset([]float64{100, 101, 102, 103, 104})
	env, err := newHistoricalEnvironment(d, `{"ma":[{"name":"ma3","enable":true,"kline_interval":"1m","period":3}]}`)
	if err != nil {
		t.Fatal(err)
	}
	bars := d.Bars[BarSeriesKey("BTCUSDT", "1m")]
	values, _, err := env.Build(bars[len(bars)-1].CloseTime, nil, 1000, zeroCosts())
	if err != nil {
		t.Fatal(err)
	}
	kline := values["kline_1m"].(line.KLinePrice)
	if len(kline.Close) != 5 || kline.Close[0] != 104 || kline.Close[4] != 100 {
		t.Fatalf("historical indicator input must be newest-to-oldest like live env: %+v", kline.Close)
	}
	ma := values["ma3"].(line.ConfigData)
	if len(ma.Data) != 3 || !closeEnough(ma.Data[0], 103) || !closeEnough(ma.Data[2], 101) {
		t.Fatalf("historical MA ordering differs from live indicator convention: %+v", ma.Data)
	}
}

func TestBacktestReplayIsDeterministic(t *testing.T) {
	d := fixtureDataset([]float64{100, 101, 104, 105})
	rules := []Rule{{Name: "open", Enable: true, Type: "long", Code: "NowPrice >= 100"}, {Name: "close", Enable: true, Type: "close_long", Code: "NowPrice >= 104"}}
	a := runFixture(t, d, rules, zeroCosts())
	b := runFixture(t, d, rules, zeroCosts())
	ar, _ := json.Marshal(a)
	br, _ := json.Marshal(b)
	if string(ar) != string(br) {
		t.Fatalf("replay differs:\n%s\n%s", ar, br)
	}
}

func TestBacktestProgressTracksProcessedBars(t *testing.T) {
	d := fixtureDataset([]float64{100, 101, 102, 103, 104})
	var completed []int
	var totals []int
	_, err := (Engine{}).RunWithProgress(context.Background(), d, StrategySnapshot{TemplateID: 1, TemplateName: "progress", TechnologyJSON: "{}", StrategyJSON: strategyJSON(Rule{Name: "never", Enable: true, Type: "long", Code: "NowPrice > 1000"})}, zeroCosts(), func(done, total int) {
		completed = append(completed, done)
		totals = append(totals, total)
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(completed) != len(d.Bars[BarSeriesKey("BTCUSDT", "1m")])+1 {
		t.Fatalf("unexpected progress callbacks: completed=%v totals=%v", completed, totals)
	}
	if completed[0] != 0 || completed[len(completed)-1] != totals[len(totals)-1] {
		t.Fatalf("progress must start at zero and finish at total: completed=%v totals=%v", completed, totals)
	}
	for i := 1; i < len(completed); i++ {
		if completed[i] < completed[i-1] || totals[i] != totals[0] {
			t.Fatalf("progress must be monotonic with stable total: completed=%v totals=%v", completed, totals)
		}
	}
}

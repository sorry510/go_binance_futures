package backtest

import (
	"testing"
	"time"

	"github.com/expr-lang/expr"
	"go_binance_futures/feature/strategy/line"
)

const ema200TechnologyJSON = `{"ema":[{"name":"ema_1d_50","enable":true,"kline_interval":"1d","period":50},{"name":"ema_1d_200","enable":true,"kline_interval":"1d","period":200}]}`

func TestEffectiveWarmupBarsExpandsForEMA200(t *testing.T) {
	got, err := (DatasetBuilder{}).effectiveWarmupBars(ema200TechnologyJSON)
	if err != nil {
		t.Fatal(err)
	}
	if got != 231 {
		t.Fatalf("warmup bars = %d, want 231", got)
	}
}

func TestHistoricalEnvironmentEMAHistoryIsSafeForStrategyIndexes(t *testing.T) {
	dataset, asOf := emaWarmupDataset(231)
	environment, err := newHistoricalEnvironment(dataset, ema200TechnologyJSON)
	if err != nil {
		t.Fatal(err)
	}
	env, _, err := environment.Build(asOf, nil, 1000, RunConfig{InitialEquity: 1000, Leverage: 8})
	if err != nil {
		t.Fatal(err)
	}
	ema, ok := env["ema_1d_200"].(line.ConfigData)
	if !ok || len(ema.Data) < line.StrategyIndicatorOutputReserve {
		t.Fatalf("EMA200 outputs = %d, want >= %d", len(ema.Data), line.StrategyIndicatorOutputReserve)
	}
	program, err := expr.Compile(`ema_1d_200.Data[1] > 0 && ema_1d_50.Data[4] > 0`, expr.Env(env))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := expr.Run(program, env); err != nil {
		t.Fatalf("strategy history indexes must be safe after warmup: %v", err)
	}
}

func TestPrefetchPlanUsesDynamicIndicatorWarmup(t *testing.T) {
	start := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	plan, err := (DatasetBuilder{}).PrefetchPlan(DatasetRequest{Symbol: "BTCUSDT", StartTime: start, EndTime: start + 24*time.Hour.Milliseconds(), TechnologyJSON: ema200TechnologyJSON})
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2020, 5, 15, 0, 0, 0, 0, time.UTC).UnixMilli()
	if plan.WarmupStartTime != want {
		t.Fatalf("warmup start = %s, want %s", time.UnixMilli(plan.WarmupStartTime).UTC(), time.UnixMilli(want).UTC())
	}
}

func emaWarmupDataset(dailyBars int) (Dataset, int64) {
	const symbol = "BTCUSDT"
	day := 24 * time.Hour
	base := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	daily := make([]Bar, 0, dailyBars)
	for i := 0; i < dailyBars; i++ {
		open := base.Add(time.Duration(i) * day).UnixMilli()
		close := open + day.Milliseconds() - 1
		price := 10000 + float64(i)
		daily = append(daily, Bar{Symbol: symbol, Interval: "1d", OpenTime: open, CloseTime: close, Open: price - 5, High: price + 10, Low: price - 10, Close: price, QuoteVolume: 1000000})
	}
	asOf := daily[len(daily)-1].CloseTime
	minuteOpen := asOf - time.Minute.Milliseconds() + 1
	minute := Bar{Symbol: symbol, Interval: "1m", OpenTime: minuteOpen, CloseTime: asOf, Open: daily[len(daily)-1].Close, High: daily[len(daily)-1].Close, Low: daily[len(daily)-1].Close, Close: daily[len(daily)-1].Close, QuoteVolume: 1000}
	dataset := Dataset{Symbol: symbol, ExecutionInterval: "1m", StartTime: minuteOpen, EndTime: asOf, Bars: map[string][]Bar{
		BarSeriesKey(symbol, "1d"): daily,
		BarSeriesKey(symbol, "1m"): {minute},
	}}
	return dataset, asOf
}

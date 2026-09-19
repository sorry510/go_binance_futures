package backtest

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"go_binance_futures/feature/strategy/line"
	"go_binance_futures/technology"

	"github.com/expr-lang/expr/vm"
)

func cacheFixtureDataset(minutes int) Dataset {
	minuteBars := optimizationFixtureBars(minutes)
	hourBars := make([]Bar, 0, minutes/60)
	for start := 0; start+60 <= len(minuteBars); start += 60 {
		parts := minuteBars[start : start+60]
		hourBars = append(hourBars, aggregatePartialBars("BTCUSDT", "1h", parts[0].OpenTime, parts[len(parts)-1].CloseTime, parts))
	}
	return Dataset{
		Market: "futures_usdt", Symbol: "BTCUSDT", ExecutionInterval: "1m", Intervals: []string{"1m", "1h"},
		StartTime: minuteBars[0].CloseTime, EndTime: minuteBars[len(minuteBars)-1].CloseTime,
		Bars: map[string][]Bar{BarSeriesKey("BTCUSDT", "1m"): minuteBars, BarSeriesKey("BTCUSDT", "1h"): hourBars},
	}
}

func cacheTechnologyJSON() string {
	return `{"ema":[{"name":"ema_1h_3","enable":true,"kline_interval":"1h","period":3}],"adx":[{"name":"adx_1h_3","enable":true,"kline_interval":"1h","period":3}],"rsi":[{"name":"rsi_1h_3","enable":true,"kline_interval":"1h","period":3}],"atr":[{"name":"atr_1h_3","enable":true,"kline_interval":"1h","period":3}],"donchian":[{"name":"dc_1h_3","enable":true,"kline_interval":"1h","period":3}]}`
}

func TestCompletedIndicatorASTCacheabilityIsConservative(t *testing.T) {
	var config struct {
		EMA []struct {
			Name string `json:"name"`
		}
	}
	_ = config
	env, err := newHistoricalEnvironment(cacheFixtureDataset(720), cacheTechnologyJSON())
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		code string
		want bool
	}{
		{`ema_1h_3.Data[1] > ema_1h_3.Data[4]`, true},
		{`all(ema_1h_3.Data[1:4], # > 0)`, true},
		{`ema_1h_3.Data[0] > 0`, false},
		{`let i = 1; ema_1h_3.Data[i] > 0`, false},
		{`mean(ema_1h_3.Data) > 0`, false},
	}
	for _, tc := range cases {
		raw, _ := json.Marshal([]Rule{{Name: "rule", Type: "long", Enable: true, Code: tc.code}})
		got := analyzeCompletedIndicatorCacheability(string(raw), env.technology)["ema_1h_3"]
		if got != tc.want {
			t.Fatalf("code=%q cacheable=%v want=%v", tc.code, got, tc.want)
		}
	}
}

func TestStandardCompletedIndicatorCacheMatchesLegacyStrategyResults(t *testing.T) {
	dataset := cacheFixtureDataset(720)
	legacy, err := newHistoricalEnvironment(dataset, cacheTechnologyJSON())
	if err != nil {
		t.Fatal(err)
	}
	optimized, err := newHistoricalEnvironment(dataset, cacheTechnologyJSON())
	if err != nil {
		t.Fatal(err)
	}
	rule := Rule{Name: "safe", Type: "long", Enable: true, Code: `ema_1h_3.Data[1] >= ema_1h_3.Data[2] && adx_1h_3.ADX[1] >= 0 && rsi_1h_3.Data[1] >= 0 && atr_1h_3.Data[1] > 0 && dc_1h_3.High[1] >= dc_1h_3.Low[1] && kline_1h.Close[0] > 0`}
	raw, _ := json.Marshal([]Rule{rule})
	optimized.enableStandardOptimizations(string(raw))
	bars := dataset.Bars[BarSeriesKey("BTCUSDT", "1m")]
	legacyPrograms, optimizedPrograms := map[int]*vm.Program{}, map[int]*vm.Program{}
	for i := 420; i < 610; i++ {
		left, _, leftErr := legacy.BuildMinuteClose(bars[i].CloseTime, bars[i], nil, 1000, zeroCosts())
		right, _, rightErr := optimized.BuildMinuteClose(bars[i].CloseTime, bars[i], nil, 1000, zeroCosts())
		if (leftErr == nil) != (rightErr == nil) {
			t.Fatalf("minute=%d error mismatch legacy=%v optimized=%v", i, leftErr, rightErr)
		}
		if leftErr != nil {
			continue
		}
		if !reflect.DeepEqual(left["kline_1h"], right["kline_1h"]) {
			t.Fatalf("minute=%d KLinePrice mismatch", i)
		}
		_, leftPass, err := evaluateRules([]Rule{rule}, "", left, legacyPrograms, false)
		if err != nil {
			t.Fatalf("legacy minute=%d: %v", i, err)
		}
		_, rightPass, err := evaluateRules([]Rule{rule}, "", right, optimizedPrograms, false)
		if err != nil {
			t.Fatalf("optimized minute=%d: %v", i, err)
		}
		if leftPass != rightPass {
			t.Fatalf("minute=%d strategy result differs legacy=%v optimized=%v", i, leftPass, rightPass)
		}
	}
	if len(optimized.indicatorCache) == 0 || len(optimized.klinePriceCache) == 0 {
		t.Fatal("expected standard replay caches to be populated")
	}
}

func TestCurrentEMAIncrementalFastPathMatchesLegacyCalculation(t *testing.T) {
	dataset := cacheFixtureDataset(720)
	legacy, _ := newHistoricalEnvironment(dataset, cacheTechnologyJSON())
	optimized, _ := newHistoricalEnvironment(dataset, cacheTechnologyJSON())
	rule := Rule{Name: "unsafe", Type: "long", Enable: true, Code: `ema_1h_3.Data[0] > 0`}
	raw, _ := json.Marshal([]Rule{rule})
	optimized.enableStandardOptimizations(string(raw))
	if optimized.indicatorCacheable["ema_1h_3"] {
		t.Fatal("Data[0] indicator must never use completed-value cache")
	}
	bars := dataset.Bars[BarSeriesKey("BTCUSDT", "1m")]
	for i := 420; i < 550; i++ {
		left, _, err := legacy.BuildMinuteClose(bars[i].CloseTime, bars[i], nil, 1000, zeroCosts())
		if err != nil {
			t.Fatal(err)
		}
		right, _, err := optimized.BuildMinuteClose(bars[i].CloseTime, bars[i], nil, 1000, zeroCosts())
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(left["ema_1h_3"].(line.ConfigData), right["ema_1h_3"].(line.ConfigData)) {
			t.Fatalf("minute=%d current EMA fast-path output differs from legacy", i)
		}
	}
	if len(optimized.currentEMACache) == 0 {
		t.Fatal("expected current EMA fast-path cache to be populated")
	}
}

func TestCurrentRSIIncrementalFastPathMatchesLegacyCalculation(t *testing.T) {
	dataset := cacheFixtureDataset(720)
	legacy, _ := newHistoricalEnvironment(dataset, cacheTechnologyJSON())
	optimized, _ := newHistoricalEnvironment(dataset, cacheTechnologyJSON())
	rule := Rule{Name: "unsafe-rsi", Type: "long", Enable: true, Code: `rsi_1h_3.Data[0] >= 0`}
	raw, _ := json.Marshal([]Rule{rule})
	optimized.enableStandardOptimizations(string(raw))
	if optimized.indicatorCacheable["rsi_1h_3"] {
		t.Fatal("Data[0] RSI must never use completed-value cache")
	}
	bars := dataset.Bars[BarSeriesKey("BTCUSDT", "1m")]
	for i := 420; i < 550; i++ {
		left, _, err := legacy.BuildMinuteClose(bars[i].CloseTime, bars[i], nil, 1000, zeroCosts())
		if err != nil {
			t.Fatal(err)
		}
		right, _, err := optimized.BuildMinuteClose(bars[i].CloseTime, bars[i], nil, 1000, zeroCosts())
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(left["rsi_1h_3"].(line.ConfigData), right["rsi_1h_3"].(line.ConfigData)) {
			t.Fatalf("minute=%d current RSI fast-path output differs from legacy", i)
		}
	}
	if len(optimized.currentRSICache) == 0 {
		t.Fatal("expected current RSI fast-path cache to be populated")
	}
}

func TestCurrentADXIncrementalFastPathMatchesLegacyCalculation(t *testing.T) {
	dataset := cacheFixtureDataset(720)
	legacy, _ := newHistoricalEnvironment(dataset, cacheTechnologyJSON())
	optimized, _ := newHistoricalEnvironment(dataset, cacheTechnologyJSON())
	rule := Rule{Name: "unsafe-adx", Type: "long", Enable: true, Code: `adx_1h_3.ADX[0] >= 0 && adx_1h_3.PlusDI[0] >= 0 && adx_1h_3.MinusDI[0] >= 0`}
	raw, _ := json.Marshal([]Rule{rule})
	optimized.enableStandardOptimizations(string(raw))
	if optimized.indicatorCacheable["adx_1h_3"] {
		t.Fatal("Data[0] ADX must never use completed-value cache")
	}
	bars := dataset.Bars[BarSeriesKey("BTCUSDT", "1m")]
	for i := 420; i < 550; i++ {
		left, _, err := legacy.BuildMinuteClose(bars[i].CloseTime, bars[i], nil, 1000, zeroCosts())
		if err != nil {
			t.Fatal(err)
		}
		right, _, err := optimized.BuildMinuteClose(bars[i].CloseTime, bars[i], nil, 1000, zeroCosts())
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(left["adx_1h_3"].(line.ADXConfigData), right["adx_1h_3"].(line.ADXConfigData)) {
			t.Fatalf("minute=%d current ADX fast-path output differs from legacy", i)
		}
	}
	if len(optimized.currentADXCache) == 0 {
		t.Fatal("expected current ADX fast-path cache to be populated")
	}
}

func TestCurrentIndicatorFastPathsMatchLegacyWithProductionPeriods(t *testing.T) {
	dataset := cacheFixtureDataset(4200)
	technologyJSON := `{"ema":[{"name":"ema_1h_20","enable":true,"kline_interval":"1h","period":20}],"rsi":[{"name":"rsi_1h_14","enable":true,"kline_interval":"1h","period":14}],"adx":[{"name":"adx_1h_14","enable":true,"kline_interval":"1h","period":14}]}`
	legacy, err := newHistoricalEnvironment(dataset, technologyJSON)
	if err != nil {
		t.Fatal(err)
	}
	optimized, err := newHistoricalEnvironment(dataset, technologyJSON)
	if err != nil {
		t.Fatal(err)
	}
	rule := Rule{Name: "production-period-current", Type: "long", Enable: true, Code: `ema_1h_20.Data[0] > 0 && rsi_1h_14.Data[0] >= 0 && adx_1h_14.ADX[0] >= 0 && adx_1h_14.PlusDI[0] >= 0 && adx_1h_14.MinusDI[0] >= 0`}
	raw, _ := json.Marshal([]Rule{rule})
	optimized.enableStandardOptimizations(string(raw))
	bars := dataset.Bars[BarSeriesKey("BTCUSDT", "1m")]
	for i := 2400; i < 2700; i++ {
		left, _, leftErr := legacy.BuildMinuteClose(bars[i].CloseTime, bars[i], nil, 1000, zeroCosts())
		right, _, rightErr := optimized.BuildMinuteClose(bars[i].CloseTime, bars[i], nil, 1000, zeroCosts())
		if (leftErr == nil) != (rightErr == nil) {
			t.Fatalf("minute=%d error mismatch legacy=%v optimized=%v", i, leftErr, rightErr)
		}
		if leftErr != nil {
			continue
		}
		for _, name := range []string{"ema_1h_20", "rsi_1h_14", "adx_1h_14"} {
			if !reflect.DeepEqual(left[name], right[name]) {
				t.Fatalf("minute=%d indicator=%s fast-path output differs from legacy", i, name)
			}
		}
	}
}

func TestAdaptiveEnvironmentKeepsStandardOptimizationDisabled(t *testing.T) {
	env, err := newHistoricalEnvironment(cacheFixtureDataset(120), cacheTechnologyJSON())
	if err != nil {
		t.Fatal(err)
	}
	if env.standardOptimizations || env.klinePriceCache != nil || env.indicatorCache != nil || env.currentEMACache != nil || env.currentRSICache != nil || env.currentADXCache != nil {
		t.Fatal("historical environment must default to legacy behavior; only standard_1m engine may opt in")
	}
}

func equalCompletedIndicatorValue(a, b interface{}) bool {
	var compare func(reflect.Value, reflect.Value) bool
	compare = func(left, right reflect.Value) bool {
		if !left.IsValid() || !right.IsValid() || left.Type() != right.Type() {
			return false
		}
		switch left.Kind() {
		case reflect.Struct:
			for i := 0; i < left.NumField(); i++ {
				if !compare(left.Field(i), right.Field(i)) {
					return false
				}
			}
			return true
		case reflect.Slice:
			if left.Type().Elem().Kind() == reflect.Float64 {
				if left.Len() != right.Len() {
					return false
				}
				if left.Len() <= 1 {
					return true
				}
				return reflect.DeepEqual(left.Slice(1, left.Len()).Interface(), right.Slice(1, right.Len()).Interface())
			}
			return reflect.DeepEqual(left.Interface(), right.Interface())
		default:
			return reflect.DeepEqual(left.Interface(), right.Interface())
		}
	}
	return compare(reflect.ValueOf(a), reflect.ValueOf(b))
}

func TestAllIndicatorImplementationsKeepCompletedOutputsIndependentOfCurrentBar(t *testing.T) {
	const n = 200
	base := line.KLinePrice{
		High: make([]float64, n), Low: make([]float64, n), Close: make([]float64, n),
		Open: make([]float64, n), Amount: make([]float64, n), Qps: make([]float64, n),
	}
	for i := 0; i < n; i++ {
		price := 200.0 - float64(i)*0.2 + float64(i%7)*0.03
		base.Open[i] = price - 0.05
		base.High[i] = price + 0.8
		base.Low[i] = price - 0.7
		base.Close[i] = price
		base.Amount[i] = 10000 + float64(i*17)
		base.Qps[i] = 200 + float64(i)
	}
	changed := line.KLinePrice{
		High: append([]float64(nil), base.High...), Low: append([]float64(nil), base.Low...),
		Close: append([]float64(nil), base.Close...), Open: append([]float64(nil), base.Open...),
		Amount: append([]float64(nil), base.Amount...), Qps: append([]float64(nil), base.Qps...),
	}
	changed.Open[0], changed.High[0], changed.Low[0], changed.Close[0] = 500, 700, 100, 650
	changed.Amount[0], changed.Qps[0] = 9e8, 8e7
	cases := []indicatorTask{
		{group: "ma", item: technology.IndicatorConfig{Name: "ma", KlineInterval: "1h", Period: 20}},
		{group: "ema", item: technology.IndicatorConfig{Name: "ema", KlineInterval: "1h", Period: 20}},
		{group: "macd", item: technology.IndicatorConfig{Name: "macd", KlineInterval: "1h", FastPeriod: 12, SlowPeriod: 26, SignalPeriod: 9}},
		{group: "rsi", item: technology.IndicatorConfig{Name: "rsi", KlineInterval: "1h", Period: 14}},
		{group: "kc", item: technology.IndicatorConfig{Name: "kc", KlineInterval: "1h", Period: 20, Multiplier: 2}},
		{group: "boll", item: technology.IndicatorConfig{Name: "boll", KlineInterval: "1h", Period: 20, StdDevMultiplier: 2}},
		{group: "atr", item: technology.IndicatorConfig{Name: "atr", KlineInterval: "1h", Period: 14}},
		{group: "adx", item: technology.IndicatorConfig{Name: "adx", KlineInterval: "1h", Period: 14}},
		{group: "mfi", item: technology.IndicatorConfig{Name: "mfi", KlineInterval: "1h", Period: 14}},
		{group: "obv", item: technology.IndicatorConfig{Name: "obv", KlineInterval: "1h"}},
		{group: "cci", item: technology.IndicatorConfig{Name: "cci", KlineInterval: "1h", Period: 20}},
		{group: "roc", item: technology.IndicatorConfig{Name: "roc", KlineInterval: "1h", Period: 12}},
		{group: "kdj", item: technology.IndicatorConfig{Name: "kdj", KlineInterval: "1h", Period: 9, KPeriod: 3, DPeriod: 3}},
		{group: "supertrend", item: technology.IndicatorConfig{Name: "supertrend", KlineInterval: "1h", Period: 10, Multiplier: 3}},
		{group: "donchian", item: technology.IndicatorConfig{Name: "donchian", KlineInterval: "1h", Period: 20}},
	}
	for _, tc := range cases {
		leftTask, rightTask := tc, tc
		leftTask.price, rightTask.price = base, changed
		left, err := calculateIndicatorValue(leftTask)
		if err != nil {
			t.Fatalf("%s base: %v", tc.group, err)
		}
		right, err := calculateIndicatorValue(rightTask)
		if err != nil {
			t.Fatalf("%s changed: %v", tc.group, err)
		}
		if !equalCompletedIndicatorValue(left, right) {
			t.Fatalf("%s completed outputs changed when only current [0] bar changed", tc.group)
		}
	}
}

func TestStandardOptimizedEngineResultIsByteIdenticalToLegacy(t *testing.T) {
	dataset := cacheFixtureDataset(720)
	dataset.StartTime = dataset.Bars[BarSeriesKey("BTCUSDT", "1m")][360].CloseTime
	dataset.DatasetSpecHash = DatasetSpecHash(dataset)
	dataset.DatasetID = "ds_" + dataset.DatasetSpecHash[:24]
	dataset.DataHash = DatasetDataHash(dataset)
	rules := []Rule{
		{Name: "open", Type: "long", Enable: true, Code: `ema_1h_3.Data[1] >= ema_1h_3.Data[2] && adx_1h_3.ADX[1] >= 0 && rsi_1h_3.Data[1] >= 0 && atr_1h_3.Data[1] > 0 && dc_1h_3.High[1] >= dc_1h_3.Low[1] && kline_1h.Close[0] > 0`},
		{Name: "close", Type: "close_long", Enable: true, Code: `ema_1h_3.Data[0] > 0 && rsi_1h_3.Data[0] >= 0 && adx_1h_3.ADX[0] >= 0 && adx_1h_3.PlusDI[0] >= 0 && adx_1h_3.MinusDI[0] >= 0`},
	}
	rawRules, _ := json.Marshal(rules)
	strategy := StrategySnapshot{TemplateID: 1, TemplateName: "cache-equivalence", TechnologyJSON: cacheTechnologyJSON(), StrategyJSON: string(rawRules), Version: "cache-equivalence"}
	config := RunConfig{InitialEquity: 1000, PositionSizePct: 1, Leverage: 1, TakeProfitPct: 0.05}
	legacyEngine := Engine{disableStandardOptimizations: true}
	legacy, err := legacyEngine.RunWithResolution(context.Background(), dataset, strategy, config, ResolutionModeStandard, nil)
	if err != nil {
		t.Fatal(err)
	}
	optimized, err := (Engine{}).RunWithResolution(context.Background(), dataset, strategy, config, ResolutionModeStandard, nil)
	if err != nil {
		t.Fatal(err)
	}
	left, _ := json.Marshal(legacy)
	right, _ := json.Marshal(optimized)
	if !reflect.DeepEqual(left, right) {
		t.Fatalf("optimized standard_1m result differs from legacy: legacy trades=%d events=%d equity=%d optimized trades=%d events=%d equity=%d", len(legacy.Trades), len(legacy.Events), len(legacy.Equity), len(optimized.Trades), len(optimized.Events), len(optimized.Equity))
	}
	if len(optimized.Trades) == 0 || len(optimized.Events) == 0 {
		t.Fatal("equivalence fixture must exercise trades and events")
	}
}

func TestAdditionalCurrentIndicatorFastPathsMatchLegacyWithProductionPeriods(t *testing.T) {
	dataset := cacheFixtureDataset(5400)
	technologyJSON := `{"roc":[{"name":"roc_1h_42","enable":true,"kline_interval":"1h","period":42}],"boll":[{"name":"boll_1h_20_2","enable":true,"kline_interval":"1h","period":20,"std_dev_multiplier":2}],"donchian":[{"name":"donchian_1h_20","enable":true,"kline_interval":"1h","period":20}],"kc":[{"name":"kc_1h_20_15","enable":true,"kline_interval":"1h","period":20,"multiplier":1.5}],"supertrend":[{"name":"supertrend_1h_10_3","enable":true,"kline_interval":"1h","period":10,"multiplier":3}]}`
	legacy, err := newHistoricalEnvironment(dataset, technologyJSON)
	if err != nil {
		t.Fatal(err)
	}
	optimized, err := newHistoricalEnvironment(dataset, technologyJSON)
	if err != nil {
		t.Fatal(err)
	}
	rule := Rule{Name: "production-period-extra-current", Type: "long", Enable: true, Code: `roc_1h_42.Data[0] > -10000 && boll_1h_20_2.High[0] >= boll_1h_20_2.Low[0] && donchian_1h_20.High[0] >= donchian_1h_20.Low[0] && kc_1h_20_15.High[0] >= kc_1h_20_15.Low[0] && supertrend_1h_10_3.Trend[0] != 0`}
	raw, _ := json.Marshal([]Rule{rule})
	optimized.enableStandardOptimizations(string(raw))
	bars := dataset.Bars[BarSeriesKey("BTCUSDT", "1m")]
	for i := 3600; i < 4200; i++ {
		left, _, leftErr := legacy.BuildMinuteClose(bars[i].CloseTime, bars[i], nil, 1000, zeroCosts())
		right, _, rightErr := optimized.BuildMinuteClose(bars[i].CloseTime, bars[i], nil, 1000, zeroCosts())
		if (leftErr == nil) != (rightErr == nil) {
			t.Fatalf("minute=%d error mismatch legacy=%v optimized=%v", i, leftErr, rightErr)
		}
		if leftErr != nil {
			continue
		}
		for _, name := range []string{"roc_1h_42", "boll_1h_20_2", "donchian_1h_20", "kc_1h_20_15", "supertrend_1h_10_3"} {
			if !reflect.DeepEqual(left[name], right[name]) {
				t.Fatalf("minute=%d indicator=%s fast-path output differs from legacy\nlegacy=%#v\noptimized=%#v", i, name, left[name], right[name])
			}
		}
	}
	if len(optimized.currentROCCache) == 0 || len(optimized.currentBOLLCache) == 0 || len(optimized.currentDonchianCache) == 0 || len(optimized.currentKCCache) == 0 || len(optimized.currentSupertrendCache) == 0 {
		t.Fatalf("expected all additional current-indicator fast-path caches to be populated: roc=%d boll=%d donchian=%d kc=%d supertrend=%d", len(optimized.currentROCCache), len(optimized.currentBOLLCache), len(optimized.currentDonchianCache), len(optimized.currentKCCache), len(optimized.currentSupertrendCache))
	}
}

func TestAdditionalCurrentIndicatorFastPathsKeepEngineResultByteIdentical(t *testing.T) {
	dataset := cacheFixtureDataset(5400)
	bars := dataset.Bars[BarSeriesKey("BTCUSDT", "1m")]
	dataset.StartTime = bars[3600].CloseTime
	dataset.EndTime = bars[4800].CloseTime
	dataset.DatasetSpecHash = DatasetSpecHash(dataset)
	dataset.DatasetID = "ds_" + dataset.DatasetSpecHash[:24]
	dataset.DataHash = DatasetDataHash(dataset)
	technologyJSON := `{"roc":[{"name":"roc_1h_42","enable":true,"kline_interval":"1h","period":42}],"boll":[{"name":"boll_1h_20_2","enable":true,"kline_interval":"1h","period":20,"std_dev_multiplier":2}],"donchian":[{"name":"donchian_1h_20","enable":true,"kline_interval":"1h","period":20}],"kc":[{"name":"kc_1h_20_15","enable":true,"kline_interval":"1h","period":20,"multiplier":1.5}],"supertrend":[{"name":"supertrend_1h_10_3","enable":true,"kline_interval":"1h","period":10,"multiplier":3}]}`
	rules := []Rule{
		{Name: "open", Type: "long", Enable: true, Code: `roc_1h_42.Data[0] > -10000 && boll_1h_20_2.High[0] >= boll_1h_20_2.Low[0] && donchian_1h_20.High[0] >= donchian_1h_20.Low[0] && kc_1h_20_15.High[0] >= kc_1h_20_15.Low[0] && supertrend_1h_10_3.Trend[0] != 0`},
		{Name: "close", Type: "close_long", Enable: true, Code: `roc_1h_42.Data[0] > -10000 && boll_1h_20_2.High[0] >= boll_1h_20_2.Low[0] && donchian_1h_20.High[0] >= donchian_1h_20.Low[0] && kc_1h_20_15.High[0] >= kc_1h_20_15.Low[0] && supertrend_1h_10_3.Trend[0] != 0`},
	}
	rawRules, _ := json.Marshal(rules)
	strategy := StrategySnapshot{TemplateID: 2, TemplateName: "extra-fast-path-equivalence", TechnologyJSON: technologyJSON, StrategyJSON: string(rawRules), Version: "extra-fast-path-equivalence"}
	config := RunConfig{InitialEquity: 1000, PositionSizePct: 1, Leverage: 1, TakeProfitPct: 0.05, StopLossPct: 0.05}
	legacy, err := (Engine{disableStandardOptimizations: true}).RunWithResolution(context.Background(), dataset, strategy, config, ResolutionModeStandard, nil)
	if err != nil {
		t.Fatal(err)
	}
	optimized, err := (Engine{}).RunWithResolution(context.Background(), dataset, strategy, config, ResolutionModeStandard, nil)
	if err != nil {
		t.Fatal(err)
	}
	left, _ := json.Marshal(legacy)
	right, _ := json.Marshal(optimized)
	if !reflect.DeepEqual(left, right) {
		t.Fatalf("additional current-indicator fast paths changed engine result: legacy trades=%d events=%d equity=%d optimized trades=%d events=%d equity=%d", len(legacy.Trades), len(legacy.Events), len(legacy.Equity), len(optimized.Trades), len(optimized.Events), len(optimized.Equity))
	}
	if len(optimized.Trades) == 0 || len(optimized.Events) == 0 {
		t.Fatal("equivalence fixture must exercise trades and events")
	}
}

func TestCurrentFourHourROCAndSupertrendFastPathsMatchLegacy(t *testing.T) {
	minuteBars := optimizationFixtureBars(15000)
	hourBars := make([]Bar, 0, len(minuteBars)/60)
	fourHourBars := make([]Bar, 0, len(minuteBars)/240)
	for start := 0; start+60 <= len(minuteBars); start += 60 {
		parts := minuteBars[start : start+60]
		hourBars = append(hourBars, aggregatePartialBars("BTCUSDT", "1h", parts[0].OpenTime, parts[len(parts)-1].CloseTime, parts))
	}
	for start := 0; start+240 <= len(minuteBars); start += 240 {
		parts := minuteBars[start : start+240]
		fourHourBars = append(fourHourBars, aggregatePartialBars("BTCUSDT", "4h", parts[0].OpenTime, parts[len(parts)-1].CloseTime, parts))
	}
	dataset := Dataset{
		Market: "futures_usdt", Symbol: "BTCUSDT", ExecutionInterval: "1m", Intervals: []string{"1m", "1h", "4h"},
		StartTime: minuteBars[0].CloseTime, EndTime: minuteBars[len(minuteBars)-1].CloseTime,
		Bars: map[string][]Bar{
			BarSeriesKey("BTCUSDT", "1m"): minuteBars,
			BarSeriesKey("BTCUSDT", "1h"): hourBars,
			BarSeriesKey("BTCUSDT", "4h"): fourHourBars,
		},
	}
	technologyJSON := `{"roc":[{"name":"roc_4h_42","enable":true,"kline_interval":"4h","period":42}],"supertrend":[{"name":"supertrend_4h_10_3","enable":true,"kline_interval":"4h","period":10,"multiplier":3}]}`
	legacy, err := newHistoricalEnvironment(dataset, technologyJSON)
	if err != nil {
		t.Fatal(err)
	}
	optimized, err := newHistoricalEnvironment(dataset, technologyJSON)
	if err != nil {
		t.Fatal(err)
	}
	rule := Rule{Name: "4h-current", Type: "long", Enable: true, Code: `roc_4h_42.Data[0] > -10000 && supertrend_4h_10_3.Trend[0] != 0`}
	raw, _ := json.Marshal([]Rule{rule})
	optimized.enableStandardOptimizations(string(raw))
	for i := 12000; i < 13200; i++ {
		left, _, leftErr := legacy.BuildMinuteClose(minuteBars[i].CloseTime, minuteBars[i], nil, 1000, zeroCosts())
		right, _, rightErr := optimized.BuildMinuteClose(minuteBars[i].CloseTime, minuteBars[i], nil, 1000, zeroCosts())
		if (leftErr == nil) != (rightErr == nil) {
			t.Fatalf("minute=%d error mismatch legacy=%v optimized=%v", i, leftErr, rightErr)
		}
		if leftErr != nil {
			continue
		}
		for _, name := range []string{"roc_4h_42", "supertrend_4h_10_3"} {
			if !reflect.DeepEqual(left[name], right[name]) {
				t.Fatalf("minute=%d indicator=%s 4h fast-path output differs from legacy", i, name)
			}
		}
	}
}

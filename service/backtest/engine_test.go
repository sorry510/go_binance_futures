package backtest

import (
	"context"
	"encoding/json"
	"math"
	"testing"
	"time"

	"go_binance_futures/feature/strategy/line"
	"go_binance_futures/service/historicalmarket"
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
	d := Dataset{Symbol: "BTCUSDT", ExecutionInterval: "1m", Intervals: []string{"1m"}, StartTime: bars[0].CloseTime, EndTime: bars[len(bars)-1].CloseTime, WarmupStartTime: bars[0].OpenTime, Bars: map[string][]Bar{}, Funding: []Funding{}}
	d.Bars[BarSeriesKey("BTCUSDT", "1m")] = bars
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
	c := zeroCosts()
	c.TakeProfitPct = 3
	r := runFixture(t, d, []Rule{{Name: "open", Enable: true, Type: "long", Code: "NowPrice >= 100"}, {Name: "close", Enable: true, Type: "close_long", Code: "NowPrice >= 104"}}, c)
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
	c := zeroCosts()
	c.TakeProfitPct = 3
	r := runFixture(t, d, []Rule{{Name: "open", Enable: true, Type: "short", Code: "NowPrice >= 100"}, {Name: "close", Enable: true, Type: "close_short", Code: "NowPrice <= 96"}}, c)
	if len(r.Trades) != 1 || r.Trades[0].Side != "SHORT" || r.Trades[0].NetPnL <= 0 {
		t.Fatalf("short failed: %+v", r.Trades)
	}
}
func TestBacktestCloseRuleWaitsForROIGate(t *testing.T) {
	d := fixtureDataset([]float64{100, 100, 100.5, 100.5})
	r := runFixture(t, d, []Rule{
		{Name: "open", Enable: true, Type: "long", Code: "NowPrice >= 100"},
		{Name: "close", Enable: true, Type: "close_long", Code: "NowPrice > 0"},
	}, RunConfig{InitialEquity: 1000, PositionSizePct: 1, Leverage: 10, TakeProfitPct: 10})
	if len(r.Trades) != 1 || r.Trades[0].ExitReason != "end_of_data" {
		t.Fatalf("close rule ran before live-equivalent ROI gate: %+v", r.Trades)
	}
}

func TestBacktestTakeProfitGateUsesLeveragedROI(t *testing.T) {
	d := fixtureDataset([]float64{100, 100, 101.02, 101.02})
	r := runFixture(t, d, []Rule{
		{Name: "open", Enable: true, Type: "long", Code: "NowPrice >= 100"},
		{Name: "close", Enable: true, Type: "close_long", Code: "NowPrice >= 101"},
	}, RunConfig{InitialEquity: 1000, PositionSizePct: 1, Leverage: 10, TakeProfitPct: 10})
	if len(r.Trades) != 1 || r.Trades[0].ExitReason != "take_profit" {
		t.Fatalf("10x leverage with 10%% ROI should unlock close rule near +1%% price move: %+v", r.Trades)
	}
	if !closeEnough(r.Trades[0].ExitPrice, 101.02) {
		t.Fatalf("ROI-gated strategy close must fill on next bar open: %+v", r.Trades[0])
	}
}

func TestBacktestStopLossGateUsesLeveragedROI(t *testing.T) {
	d := fixtureDataset([]float64{100, 100, 98.99, 98.99})
	r := runFixture(t, d, []Rule{
		{Name: "open", Enable: true, Type: "long", Code: "NowPrice >= 100"},
		{Name: "close", Enable: true, Type: "close_long", Code: "NowPrice < 100"},
	}, RunConfig{InitialEquity: 1000, PositionSizePct: 1, Leverage: 10, StopLossPct: 10})
	if len(r.Trades) != 1 || r.Trades[0].ExitReason != "stop_loss" {
		t.Fatalf("10x leverage with 10%% loss ROI should unlock close rule near -1%% price move: %+v", r.Trades)
	}
	if !closeEnough(r.Trades[0].ExitPrice, 98.99) {
		t.Fatalf("ROI-gated strategy close must fill on next bar open: %+v", r.Trades[0])
	}
}

func TestBacktestROIGateDoesNotForceCloseWhenRuleIsFalse(t *testing.T) {
	d := fixtureDataset([]float64{100, 100, 101.02, 102})
	r := runFixture(t, d, []Rule{
		{Name: "open", Enable: true, Type: "long", Code: "NowPrice >= 100"},
		{Name: "close", Enable: true, Type: "close_long", Code: "false"},
	}, RunConfig{InitialEquity: 1000, PositionSizePct: 1, Leverage: 10, TakeProfitPct: 10})
	if len(r.Trades) != 1 || r.Trades[0].ExitReason != "end_of_data" {
		t.Fatalf("ROI threshold must gate, not force, the close rule: %+v", r.Trades)
	}
}

func TestCloseGateZeroThresholdsMatchLiveDisabledDefaults(t *testing.T) {
	config := RunConfig{}
	if got := closeGateReason(20, config); got != "" {
		t.Fatalf("zero thresholds should be effectively disabled, got %q", got)
	}
	if got := closeGateReason(-20, config); got != "" {
		t.Fatalf("zero thresholds should be effectively disabled, got %q", got)
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

func TestHistoricalEnvironmentUsesLatestVisibleMarketCondition(t *testing.T) {
	d := fixtureDataset([]float64{100, 101, 102})
	bars := d.Bars[BarSeriesKey("BTCUSDT", "1m")]
	d.MarketConditionRequired = true
	d.MarketConditions = []MarketConditionPoint{
		{Time: bars[0].CloseTime - 1, Value: 2},
		{Time: bars[1].CloseTime, Value: 4},
	}
	environment, err := newHistoricalEnvironment(d, "{}")
	if err != nil {
		t.Fatal(err)
	}
	firstEnv, firstCondition, err := environment.Build(bars[0].CloseTime, nil, 1000, zeroCosts())
	if err != nil {
		t.Fatal(err)
	}
	if firstCondition != 2 || firstEnv["MarketCondition"] != "2" {
		t.Fatalf("unexpected first MarketCondition: condition=%d env=%v", firstCondition, firstEnv["MarketCondition"])
	}
	secondEnv, secondCondition, err := environment.Build(bars[1].CloseTime, nil, 1000, zeroCosts())
	if err != nil {
		t.Fatal(err)
	}
	if secondCondition != 4 || secondEnv["MarketCondition"] != "4" {
		t.Fatalf("unexpected second MarketCondition: condition=%d env=%v", secondCondition, secondEnv["MarketCondition"])
	}
}

func TestV34AStandardOneMinuteBaselineKeepsV6Semantics(t *testing.T) {
	d := fixtureDataset([]float64{100, 100, 101.02, 101.02})
	bars := d.Bars[BarSeriesKey("BTCUSDT", "1m")]
	d.MarketConditionRequired = true
	d.MarketConditions = []MarketConditionPoint{
		{Time: bars[0].CloseTime - 1, Value: 2},
		{Time: bars[2].CloseTime, Value: 4},
	}
	d.Funding = []Funding{{Symbol: "BTCUSDT", FundingTime: bars[1].CloseTime, FundingRate: 0.001, MarkPrice: 100}}
	d.DataHash = ""

	r := runFixture(t, d, []Rule{
		{Name: "open-v6", Enable: true, Type: "long", Code: `MarketCondition == "2" && NowPrice >= 100`},
		{Name: "close-v6", Enable: true, Type: "close_long", Code: `MarketCondition == "4" && NowPrice >= 101`},
	}, RunConfig{InitialEquity: 1000, PositionSizePct: 1, Leverage: 10, TakeProfitPct: 10})

	if r.EngineVersion != "backtest_engine_v6" {
		t.Fatalf("standard baseline engine version changed: %s", r.EngineVersion)
	}
	if len(r.Trades) != 1 {
		t.Fatalf("standard baseline trade count changed: %+v", r.Trades)
	}
	trade := r.Trades[0]
	if trade.EntryTime != bars[1].OpenTime || trade.ExitTime != bars[3].OpenTime {
		t.Fatalf("V6 next-open timing changed: %+v", trade)
	}
	if !closeEnough(trade.EntryPrice, 100) || !closeEnough(trade.ExitPrice, 101.02) || trade.ExitReason != "take_profit" {
		t.Fatalf("V6 ROI gate/fill semantics changed: %+v", trade)
	}
	if !closeEnough(trade.FundingPnL, -10) || trade.MarketCondition != 2 {
		t.Fatalf("V6 funding/MarketCondition semantics changed: %+v", trade)
	}
	fundingEvents := 0
	for _, event := range r.Events {
		if event.Action == "funding" {
			fundingEvents++
		}
	}
	if fundingEvents != 1 {
		t.Fatalf("funding must be charged exactly once, events=%+v", r.Events)
	}
}

func TestAdaptiveV8MatchesStandardV6WhenNoIntrabarResolutionIsNeeded(t *testing.T) {
	d := fixtureDataset([]float64{100, 101, 104, 105})
	rules := []Rule{
		{Name: "open", Enable: true, Type: "long", Code: "NowPrice >= 100"},
		{Name: "close", Enable: true, Type: "close_long", Code: "NowPrice >= 104"},
	}
	strategy := StrategySnapshot{TemplateID: 1, TemplateName: "adaptive-baseline", TechnologyJSON: "{}", StrategyJSON: strategyJSON(rules...)}
	engine := Engine{}
	standard, err := engine.RunWithResolution(context.Background(), d, strategy, zeroCosts(), ResolutionModeStandard, nil)
	if err != nil {
		t.Fatal(err)
	}
	adaptive, err := engine.RunWithResolution(context.Background(), d, strategy, zeroCosts(), ResolutionModeAdaptive, nil)
	if err != nil {
		t.Fatal(err)
	}
	if standard.EngineVersion != StandardEngineVersion || adaptive.EngineVersion != AdaptiveEngineVersion {
		t.Fatalf("unexpected engine versions: standard=%s adaptive=%s", standard.EngineVersion, adaptive.EngineVersion)
	}
	if standard.ResolutionMode != ResolutionModeStandard || adaptive.ResolutionMode != ResolutionModeAdaptive {
		t.Fatalf("unexpected resolution modes: standard=%s adaptive=%s", standard.ResolutionMode, adaptive.ResolutionMode)
	}
	if adaptive.ResolutionStats != (ResolutionStats{}) {
		t.Fatalf("no-ambiguity adaptive run must not drill down: %+v", adaptive.ResolutionStats)
	}
	if standard.DataHash != adaptive.DataHash {
		t.Fatalf("no-ambiguity adaptive run changed data hash: %s != %s", standard.DataHash, adaptive.DataHash)
	}
	standard.EngineVersion, adaptive.EngineVersion = "", ""
	standard.ResolutionMode, adaptive.ResolutionMode = "", ""
	standard.ResolutionModel, adaptive.ResolutionModel = "", ""
	standardRaw, _ := json.Marshal(standard)
	adaptiveRaw, _ := json.Marshal(adaptive)
	if string(standardRaw) != string(adaptiveRaw) {
		t.Fatalf("adaptive no-ambiguity result diverged from V6:\nstandard=%s\nadaptive=%s", standardRaw, adaptiveRaw)
	}
}

func TestNormalizeResolutionModeDefaultsToStandard(t *testing.T) {
	mode, err := NormalizeResolutionMode("")
	if err != nil || mode != ResolutionModeStandard {
		t.Fatalf("default resolution mode=%q err=%v", mode, err)
	}
	if _, err := NormalizeResolutionMode("full_tick_magic"); err == nil {
		t.Fatal("unsupported resolution mode must fail closed")
	}
}

func TestDetectROICandidateLongAndShort(t *testing.T) {
	bar := Bar{Low: 98.5, High: 101.5}
	config := RunConfig{Leverage: 10, StopLossPct: 10, TakeProfitPct: 10}
	long := DetectROICandidate(&Position{Side: "LONG", EntryPrice: 100, Quantity: 1}, bar, config)
	if !long.Required || !long.StopPossible || !long.ProfitPossible || long.MinROI >= -10 || long.MaxROI <= 10 {
		t.Fatalf("unexpected LONG ROI candidate: %+v", long)
	}
	short := DetectROICandidate(&Position{Side: "SHORT", EntryPrice: 100, Quantity: 1}, bar, config)
	if !short.Required || !short.StopPossible || !short.ProfitPossible || short.MinROI >= -10 || short.MaxROI <= 10 {
		t.Fatalf("unexpected SHORT ROI candidate: %+v", short)
	}
	quiet := DetectROICandidate(&Position{Side: "LONG", EntryPrice: 100, Quantity: 1}, Bar{Low: 99.9, High: 100.1}, config)
	if quiet.Required {
		t.Fatalf("small 1m range must not trigger adaptive data: %+v", quiet)
	}
}

func TestDetectROICandidateDisabledThresholdsDoNotTrigger(t *testing.T) {
	position := &Position{Side: "LONG", EntryPrice: 100, Quantity: 1}
	bar := Bar{Low: 1, High: 10000}
	candidate := DetectROICandidate(position, bar, RunConfig{Leverage: 125, StopLossPct: 0, TakeProfitPct: 0})
	if candidate.Required || candidate.StopPossible || candidate.ProfitPossible {
		t.Fatalf("disabled ROI thresholds must not request adaptive resolution: %+v", candidate)
	}
}

func TestStandardVisibleBarsUsesCanonicalSliceWithoutFullHistoryCopy(t *testing.T) {
	d := fixtureDataset([]float64{100, 101, 102})
	environment, err := newHistoricalEnvironment(d, `{}`)
	if err != nil {
		t.Fatal(err)
	}
	key := BarSeriesKey("BTCUSDT", "1m")
	canonical := d.Bars[key]
	visible := environment.visibleBars("BTCUSDT", "1m", canonical[1].CloseTime)
	if len(visible) != 2 {
		t.Fatalf("unexpected visible bar count: %d", len(visible))
	}
	if &visible[0] != &canonical[0] {
		t.Fatal("standard replay must not copy the full visible history on every bar")
	}
}

func TestAdaptiveIntrabarEnvironmentUsesOnlyObservedPartialKline(t *testing.T) {
	d := fixtureDataset([]float64{100, 101, 999})
	bars := d.Bars[BarSeriesKey("BTCUSDT", "1m")]
	d.MarketConditionRequired = true
	asOf := bars[2].OpenTime + 30_999
	d.MarketConditions = []MarketConditionPoint{
		{Time: bars[1].CloseTime, Value: 2},
		{Time: asOf + 1, Value: 4},
	}
	environment, err := newHistoricalEnvironment(d, `{"ma":[{"name":"ma2","enable":true,"kline_interval":"1m","period":2}]}`)
	if err != nil {
		t.Fatal(err)
	}
	partial := Bar{Symbol: "BTCUSDT", Interval: "1m", OpenTime: bars[2].OpenTime, CloseTime: asOf, Open: 101, High: 102, Low: 100.5, Close: 101.5, Volume: 5, QuoteVolume: 505}
	position := &Position{Side: "LONG", EntryTime: bars[1].OpenTime, EntryPrice: 100, Quantity: 1}
	env, condition, err := environment.BuildIntrabar(asOf, partial, position, 1000, RunConfig{Leverage: 10})
	if err != nil {
		t.Fatal(err)
	}
	if env["NowPrice"].(float64) != 101.5 {
		t.Fatalf("intrabar NowPrice used future 1m close: %#v", env["NowPrice"])
	}
	if condition != 2 || env["MarketCondition"] != "2" {
		t.Fatalf("intrabar MarketCondition leaked future point: condition=%d env=%v", condition, env["MarketCondition"])
	}
	kline := env["kline_1m"].(line.KLinePrice)
	if len(kline.Close) < 1 || kline.Close[0] != 101.5 {
		t.Fatalf("live-equivalent current 1m bar was not reconstructed: %+v", kline.Close)
	}
	for _, closePrice := range kline.Close {
		if closePrice == 999 {
			t.Fatalf("future full-minute close leaked into intrabar environment: %+v", kline.Close)
		}
	}
	if roi := env["ROI"].(float64); roi <= 0 {
		t.Fatalf("intrabar ROI did not use current observed price: %v", roi)
	}
}

func TestAdaptiveV8ClosesOnIntraminuteROIGateWhenRuleBecomesTrue(t *testing.T) {
	d := fixtureDataset([]float64{100, 100.2, 100.2, 100.2})
	bars := d.Bars[BarSeriesKey("BTCUSDT", "1m")]
	bars[1].High = 101.5
	bars[1].Low = 99.8
	d.Bars[BarSeriesKey("BTCUSDT", "1m")] = bars
	d.DataHash = DatasetDataHash(d)
	provider := &intrabarFixtureProvider{seconds: []historicalmarket.Kline{
		{Market: d.Market, Symbol: d.Symbol, Interval: "1s", OpenTime: bars[1].OpenTime, CloseTime: bars[1].OpenTime + 999, Open: 100, High: 100.2, Low: 99.9, Close: 100.2},
		{Market: d.Market, Symbol: d.Symbol, Interval: "1s", OpenTime: bars[1].OpenTime + 1000, CloseTime: bars[1].OpenTime + 1999, Open: 100.2, High: 101.2, Low: 100.2, Close: 101.1},
	}}
	engine := Engine{ResolutionProviderFactory: func() (historicalmarket.ResolutionProvider, error) { return provider, nil }}
	strategy := StrategySnapshot{TemplateID: 1, TemplateName: "intraminute-close", TechnologyJSON: "{}", StrategyJSON: strategyJSON(
		Rule{Name: "open", Enable: true, Type: "long", Code: "NowPrice >= 100"},
		Rule{Name: "close", Enable: true, Type: "close_long", Code: "NowPrice >= 101"},
	)}
	config := RunConfig{InitialEquity: 1000, PositionSizePct: 1, Leverage: 10, TakeProfitPct: 10}
	result, err := engine.RunWithResolution(context.Background(), d, strategy, config, ResolutionModeAdaptive, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Trades) == 0 {
		t.Fatalf("adaptive intraminute close missing: %+v", result.Trades)
	}
	trade := result.Trades[0]
	if trade.ExitReason != "take_profit" || trade.ExitResolution != "1s" || trade.ExitTime != bars[1].OpenTime+1999 {
		t.Fatalf("unexpected adaptive close: %+v", trade)
	}
	if provider.secondCalls != 1 || provider.tradeCalls != 0 || result.ResolutionStats.SecondDrilldownMinutes != 1 {
		t.Fatalf("unexpected adaptive resolution usage: second=%d trade=%d stats=%+v", provider.secondCalls, provider.tradeCalls, result.ResolutionStats)
	}
	if result.DataHash == d.DataHash {
		t.Fatal("consumed 1s evidence must extend the run data hash")
	}
	foundAudit := false
	for _, event := range result.Events {
		if event.Type == "intrabar_resolution" {
			foundAudit = true
			break
		}
	}
	if !foundAudit {
		t.Fatalf("intrabar resolution audit event missing: %+v", result.Events)
	}
}

func TestAdaptiveV8ROIGateDoesNotForceCloseWhenRuleIsFalse(t *testing.T) {
	d := fixtureDataset([]float64{100, 100.2, 100.2})
	bars := d.Bars[BarSeriesKey("BTCUSDT", "1m")]
	bars[1].High = 101.5
	d.Bars[BarSeriesKey("BTCUSDT", "1m")] = bars
	provider := &intrabarFixtureProvider{seconds: []historicalmarket.Kline{{Market: d.Market, Symbol: d.Symbol, Interval: "1s", OpenTime: bars[1].OpenTime, CloseTime: bars[1].OpenTime + 999, Open: 100, High: 101.2, Low: 100, Close: 101.1}}}
	engine := Engine{ResolutionProviderFactory: func() (historicalmarket.ResolutionProvider, error) { return provider, nil }}
	strategy := StrategySnapshot{TemplateID: 1, TemplateName: "false-close", TechnologyJSON: "{}", StrategyJSON: strategyJSON(
		Rule{Name: "open", Enable: true, Type: "long", Code: "NowPrice >= 100"},
		Rule{Name: "close", Enable: true, Type: "close_long", Code: "false"},
	)}
	result, err := engine.RunWithResolution(context.Background(), d, strategy, RunConfig{InitialEquity: 1000, PositionSizePct: 1, Leverage: 10, TakeProfitPct: 10}, ResolutionModeAdaptive, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Trades) != 1 || result.Trades[0].ExitReason != "end_of_data" {
		t.Fatalf("ROI gate alone must not force an adaptive close: %+v", result.Trades)
	}
}

func TestAdaptiveFundingAfterIntraminuteCloseIsNotAppliedEarly(t *testing.T) {
	d := fixtureDataset([]float64{100, 100.2, 100.2})
	bars := d.Bars[BarSeriesKey("BTCUSDT", "1m")]
	bars[1].High = 101.5
	d.Bars[BarSeriesKey("BTCUSDT", "1m")] = bars
	d.Funding = []Funding{{Symbol: d.Symbol, FundingTime: bars[1].OpenTime + 30_000, FundingRate: 0.01, MarkPrice: 101}}
	provider := &intrabarFixtureProvider{seconds: []historicalmarket.Kline{{Market: d.Market, Symbol: d.Symbol, Interval: "1s", OpenTime: bars[1].OpenTime, CloseTime: bars[1].OpenTime + 999, Open: 100, High: 101.2, Low: 100, Close: 101.1}}}
	engine := Engine{ResolutionProviderFactory: func() (historicalmarket.ResolutionProvider, error) { return provider, nil }}
	strategy := StrategySnapshot{TemplateID: 1, TemplateName: "funding-asof", TechnologyJSON: "{}", StrategyJSON: strategyJSON(
		Rule{Name: "open", Enable: true, Type: "long", Code: "NowPrice >= 100"},
		Rule{Name: "close", Enable: true, Type: "close_long", Code: "NowPrice >= 101"},
	)}
	result, err := engine.RunWithResolution(context.Background(), d, strategy, RunConfig{InitialEquity: 1000, PositionSizePct: 1, Leverage: 10, TakeProfitPct: 10}, ResolutionModeAdaptive, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Trades) == 0 || result.Trades[0].FundingPnL != 0 {
		t.Fatalf("funding after intraminute exit leaked into first trade: %+v", result.Trades)
	}
	if result.Trades[0].ExitTime >= d.Funding[0].FundingTime {
		t.Fatalf("fixture did not exit before funding: trade=%+v funding=%+v", result.Trades[0], d.Funding[0])
	}
}

func TestAdaptiveV8TradeReplayUsesTradeIDForSameTimestampCrossing(t *testing.T) {
	d := fixtureDataset([]float64{100, 100.2, 100.2, 100.2})
	bars := d.Bars[BarSeriesKey("BTCUSDT", "1m")]
	bars[1].High = 101.5
	bars[1].Low = 99.9
	d.Bars[BarSeriesKey("BTCUSDT", "1m")] = bars
	d.DataHash = DatasetDataHash(d)
	at := bars[1].OpenTime + 250
	provider := &intrabarFixtureProvider{
		seconds: []historicalmarket.Kline{{Market: d.Market, Symbol: d.Symbol, Interval: "1s", OpenTime: bars[1].OpenTime, CloseTime: bars[1].OpenTime + 999, Open: 100, High: 101.5, Low: 99.9, Close: 100.2}},
		trades: []historicalmarket.PublicDataTrade{
			{TradeID: 20, TradeTime: at, Price: 101.3, Quantity: 1, QuoteQuantity: 101.3},
			{TradeID: 10, TradeTime: at, Price: 101.1, Quantity: 1, QuoteQuantity: 101.1},
		},
	}
	engine := Engine{ResolutionProviderFactory: func() (historicalmarket.ResolutionProvider, error) { return provider, nil }}
	strategy := StrategySnapshot{TemplateID: 1, TemplateName: "trade-id-order", TechnologyJSON: "{}", StrategyJSON: strategyJSON(
		Rule{Name: "open", Enable: true, Type: "long", Code: "NowPrice >= 100"},
		Rule{Name: "close", Enable: true, Type: "close_long", Code: "NowPrice >= 101"},
	)}
	result, err := engine.RunWithResolution(context.Background(), d, strategy, RunConfig{InitialEquity: 1000, PositionSizePct: 1, Leverage: 10, TakeProfitPct: 10}, ResolutionModeAdaptive, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Trades) == 0 {
		t.Fatalf("trade replay close missing: %+v", result.Trades)
	}
	trade := result.Trades[0]
	if trade.ExitResolution != "trades" || trade.ExitTime != at || math.Abs(trade.ExitPrice-101.1) > 1e-9 {
		t.Fatalf("same-timestamp trades were not ordered by trade_id: %+v", trade)
	}
	if provider.tradeCalls != 1 || result.ResolutionStats.TradeDrilldownSeconds != 1 {
		t.Fatalf("trade drill-down statistics missing: calls=%d stats=%+v", provider.tradeCalls, result.ResolutionStats)
	}
	if result.DataHash == d.DataHash {
		t.Fatal("consumed trade evidence must extend the run data hash")
	}
}

func TestAdaptiveV8TradeReplayDoesNotSeeLaterFundingInSameSecond(t *testing.T) {
	d := fixtureDataset([]float64{100, 100.2, 100.2})
	bars := d.Bars[BarSeriesKey("BTCUSDT", "1m")]
	bars[1].High = 101.5
	d.Bars[BarSeriesKey("BTCUSDT", "1m")] = bars
	d.Funding = []Funding{{Symbol: d.Symbol, FundingTime: bars[1].OpenTime + 700, FundingRate: 0.01, MarkPrice: 101}}
	provider := &intrabarFixtureProvider{
		seconds: []historicalmarket.Kline{{Market: d.Market, Symbol: d.Symbol, Interval: "1s", OpenTime: bars[1].OpenTime, CloseTime: bars[1].OpenTime + 999, Open: 100, High: 101.5, Low: 100, Close: 100.2}},
		trades:  []historicalmarket.PublicDataTrade{{TradeID: 1, TradeTime: bars[1].OpenTime + 200, Price: 101.1, Quantity: 1, QuoteQuantity: 101.1}},
	}
	engine := Engine{ResolutionProviderFactory: func() (historicalmarket.ResolutionProvider, error) { return provider, nil }}
	strategy := StrategySnapshot{TemplateID: 1, TemplateName: "trade-funding-asof", TechnologyJSON: "{}", StrategyJSON: strategyJSON(
		Rule{Name: "open", Enable: true, Type: "long", Code: "NowPrice >= 100"},
		Rule{Name: "close", Enable: true, Type: "close_long", Code: "NowPrice >= 101"},
	)}
	result, err := engine.RunWithResolution(context.Background(), d, strategy, RunConfig{InitialEquity: 1000, PositionSizePct: 1, Leverage: 10, TakeProfitPct: 10}, ResolutionModeAdaptive, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Trades) == 0 || result.Trades[0].ExitResolution != "trades" {
		t.Fatalf("trade replay close missing: %+v", result.Trades)
	}
	if result.Trades[0].FundingPnL != 0 || result.Trades[0].ExitTime >= d.Funding[0].FundingTime {
		t.Fatalf("later same-second funding leaked into earlier crossing: trade=%+v funding=%+v", result.Trades[0], d.Funding[0])
	}
}

func TestAdaptiveV8TradeReplayIgnoresFutureTrade(t *testing.T) {
	d := fixtureDataset([]float64{100, 100.2, 100.2})
	bars := d.Bars[BarSeriesKey("BTCUSDT", "1m")]
	bars[1].High = 101.5
	d.Bars[BarSeriesKey("BTCUSDT", "1m")] = bars
	provider := &intrabarFixtureProvider{
		seconds: []historicalmarket.Kline{{Market: d.Market, Symbol: d.Symbol, Interval: "1s", OpenTime: bars[1].OpenTime, CloseTime: bars[1].OpenTime + 999, Open: 100, High: 101.5, Low: 100, Close: 100.2}},
		trades:  []historicalmarket.PublicDataTrade{{TradeID: 1, TradeTime: bars[1].OpenTime + 1500, Price: 101.2, Quantity: 1, QuoteQuantity: 101.2}},
	}
	engine := Engine{ResolutionProviderFactory: func() (historicalmarket.ResolutionProvider, error) { return provider, nil }}
	strategy := StrategySnapshot{TemplateID: 1, TemplateName: "future-trade", TechnologyJSON: "{}", StrategyJSON: strategyJSON(
		Rule{Name: "open", Enable: true, Type: "long", Code: "NowPrice >= 100"},
		Rule{Name: "close", Enable: true, Type: "close_long", Code: "NowPrice >= 101"},
	)}
	result, err := engine.RunWithResolution(context.Background(), d, strategy, RunConfig{InitialEquity: 1000, PositionSizePct: 1, Leverage: 10, TakeProfitPct: 10}, ResolutionModeAdaptive, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Trades) != 1 || result.Trades[0].ExitReason != "end_of_data" {
		t.Fatalf("future trade leaked into current second: %+v", result.Trades)
	}
}

func TestAdaptiveROIOnlyStableCloseSkipsHighResolutionWhenRuleCannotPass(t *testing.T) {
	d := fixtureDataset([]float64{100, 100, 100, 100})
	provider := &intrabarFixtureProvider{}
	engine := Engine{ResolutionProviderFactory: func() (historicalmarket.ResolutionProvider, error) { return provider, nil }}
	config := zeroCosts()
	config.TakeProfitPct = 0.1
	strategy := StrategySnapshot{TemplateID: 1, TemplateName: "stable-roi", TechnologyJSON: "{}", StrategyJSON: strategyJSON(
		Rule{Name: "open", Enable: true, Type: "long", Code: "NowPrice >= 100"},
		Rule{Name: "close", Enable: true, Type: "close_long", Code: "ROI >= 0.05 && 1 == 0"},
	)}
	result, err := engine.RunWithResolution(context.Background(), d, strategy, config, ResolutionModeAdaptive, nil)
	if err != nil {
		t.Fatal(err)
	}
	if provider.secondCalls != 0 || provider.tradeCalls != 0 {
		t.Fatalf("provably impossible intraminute close must not fetch high-resolution data: seconds=%d trades=%d", provider.secondCalls, provider.tradeCalls)
	}
	if len(result.Trades) != 1 || result.Trades[0].ExitReason != "end_of_data" {
		t.Fatalf("unexpected result after ROI-only pruning: %+v", result.Trades)
	}
}

func TestCloseRuleROIProfileRejectsCurrentBarAndDynamicFields(t *testing.T) {
	stable := analyzeCloseRuleROIProfile([]Rule{{Enable: true, Type: "close_long", Code: "ROI >= 5 && kline_1h.Close[1] > 0"}}, "LONG")
	if !stable.Eligible || len(stable.Thresholds) != 1 || stable.Thresholds[0] != 5 {
		t.Fatalf("completed-bar ROI rule should be eligible: %+v", stable)
	}
	for _, code := range []string{
		"ROI >= 5 && kline_1h.Close[0] > 0",
		"ROI >= 5 && NowPrice > 100",
		"ROI >= 5 && IsAsc(kline_1h.Close, 3)",
		"ROI * 2 >= 5",
	} {
		if got := analyzeCloseRuleROIProfile([]Rule{{Enable: true, Type: "close_long", Code: code}}, "LONG"); got.Eligible {
			t.Fatalf("dynamic/non-comparison ROI rule must use conservative replay: code=%q profile=%+v", code, got)
		}
	}
}

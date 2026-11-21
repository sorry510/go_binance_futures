package backtest

import (
	"math"
	"testing"
	"time"

	"go_binance_futures/service/historicalmarket"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
)

func TestFixedMarginUsesSameMarginAfterProfit(t *testing.T) {
	d := fixtureDataset([]float64{100, 100, 103, 103, 103, 103, 106, 106, 106, 106, 109, 109})
	rules := []Rule{
		{Name: "open", Enable: true, Type: "long", Code: "NowPrice > 0"},
		{Name: "close", Enable: true, Type: "close_long", Code: "ROI >= 8"},
	}
	fixed := RunConfig{
		PositionMode: PositionModeFixedMargin, FixedMarginUSDT: 20,
		QuantityStepSize: "0.001", InitialEquity: 1000, Leverage: 4,
		TakeProfitPct: 8, StopLossPct: 6, FeeRate: 0.0005, SlippageBps: 5,
	}
	got := runFixture(t, d, rules, fixed)
	if len(got.Trades) < 2 {
		t.Fatalf("want multiple entries to verify no compound sizing: %+v", got.Trades)
	}
	for _, trade := range got.Trades {
		margin := trade.Quantity * trade.EntryPrice / float64(fixed.Leverage)
		if math.Abs(margin-fixed.FixedMarginUSDT) > 0.08 {
			t.Errorf("trade %d margin=%v, want near %v (StepSize rounding)", trade.Sequence, margin, fixed.FixedMarginUSDT)
		}
		if math.Abs(trade.Quantity*1000-math.Round(trade.Quantity*1000)) > 1e-6 {
			t.Errorf("trade %d ignored 0.001 StepSize: quantity=%v", trade.Sequence, trade.Quantity)
		}
	}
	if !closeEnough(got.Metrics.NetPnL, got.Equity[len(got.Equity)-1].Equity-fixed.InitialEquity) {
		t.Fatalf("net pnl no longer matches equity: %+v", got.Metrics)
	}
	if got.Metrics.NetPnL > 0 && got.Metrics.ReturnPct > 20 {
		t.Fatalf("account return incorrectly calculated from 20 USDT margin: %+v", got.Metrics)
	}
	legacy := fixed
	legacy.PositionMode = PositionModeEquityPercent
	legacy.PositionSizePct = 1
	old := runFixture(t, d, rules, legacy)
	if len(old.Trades) < 2 {
		t.Fatalf("legacy fixture does not generate trades: %+v", old.Trades)
	}
	if old.Trades[0].Quantity <= got.Trades[0].Quantity*10 {
		t.Fatalf("legacy percent mode no longer uses the account's full equity: legacy=%v fixed=%v", old.Trades[0].Quantity, got.Trades[0].Quantity)
	}
}

func TestFixedMarginInsufficientBalanceSkipsOpen(t *testing.T) {
	d := fixtureDataset([]float64{100, 100, 100, 100})
	config := RunConfig{PositionMode: PositionModeFixedMargin, FixedMarginUSDT: 20,
		QuantityStepSize: "0.001", InitialEquity: 10, Leverage: 4, FeeRate: 0.0005}
	result := runFixture(t, d,
		[]Rule{{Name: "open", Enable: true, Type: "long", Code: "NowPrice > 0"}}, config)
	if len(result.Trades) != 0 || result.Metrics.NetPnL != 0 {
		t.Fatalf("insufficient cash opened position: %+v", result.Trades)
	}
	var skipped bool
	for _, event := range result.Events {
		if event.Type == "order" && event.Action == "skipped" {
			skipped = true
		}
	}
	if !skipped {
		t.Fatal("missing insufficient-margin diagnostic")
	}
}

func TestFixedMarginRejectsInvalidInputs(t *testing.T) {
	for _, config := range []RunConfig{
		{PositionMode: PositionModeFixedMargin, FixedMarginUSDT: 0, QuantityStepSize: "0.001"},
		{PositionMode: PositionModeFixedMargin, FixedMarginUSDT: -20, QuantityStepSize: "0.001"},
		{PositionMode: PositionModeFixedMargin, FixedMarginUSDT: math.NaN(), QuantityStepSize: "0.001"},
		{PositionMode: PositionModeFixedMargin, FixedMarginUSDT: 20, QuantityStepSize: ""},
		{PositionMode: PositionModeFixedMargin, FixedMarginUSDT: 20, QuantityStepSize: "invalid"},
		{PositionMode: "unknown", FixedMarginUSDT: 20, QuantityStepSize: "0.001"},
	} {
		if err := ValidateRunConfig(config); err == nil {
			t.Fatalf("invalid fixed config accepted: %+v", config)
		}
	}
	if config := NormalizeRunConfig(RunConfig{}); config.PositionMode != PositionModeEquityPercent {
		t.Fatalf("legacy config lost original semantics: %+v", config)
	}
}

func TestFixedMarginSnapshotPersistsAndSymbolDefaults(t *testing.T) {
	setupBacktestStoreTest(t)
	coin := models.Symbols{Symbol: "BTCUSDT", Usdt: "20", Leverage: 4, Profit: "8", Loss: "6", StepSize: "0.001", TickSize: "0.1"}
	if _, err := orm.NewOrm().Insert(&coin); err != nil {
		t.Fatal(err)
	}
	defaults, err := GetBacktestSymbolDefaults("btcusdt")
	if err != nil || defaults.MarginUSDT != 20 || defaults.Leverage != 4 ||
		defaults.StopLossPct != 6 || defaults.TakeProfitPct != 8 || defaults.QuantityStepSize != "0.001" {
		t.Fatalf("symbol defaults mismatch: %+v err=%v", defaults, err)
	}
	config := RunConfig{PositionMode: PositionModeFixedMargin, FixedMarginUSDT: 25,
		QuantityStepSize: "999", InitialEquity: 1000, Leverage: 4,
		StopLossPct: 6, TakeProfitPct: 8, FeeRate: 0.0005, SlippageBps: 5}
	if err := ValidateRunConfig(config); err != nil {
		t.Fatal(err)
	}
	// Snapshot reader covers both new and pre-migration data.
	newRun := summaryFromRow(models.AgentBacktestRun{
		PositionMode: PositionModeFixedMargin, FixedMarginUSDT: config.FixedMarginUSDT,
		QuantityStepSize: defaults.QuantityStepSize, InitialEquity: 1000, PositionSizePct: 1,
		Leverage: 4, StopLossPct: 6, TakeProfitPct: 8,
	})
	if newRun.Config.FixedMarginUSDT != 25 || newRun.Config.QuantityStepSize != "0.001" {
		t.Fatalf("fixed snapshot incorrect: %+v", newRun.Config)
	}
	row := models.AgentBacktestRun{RunID: "bt_fixed_margin_snapshot_test", Symbol: "BTCUSDT",
		PositionMode: newRun.Config.PositionMode, FixedMarginUSDT: newRun.Config.FixedMarginUSDT,
		QuantityStepSize: newRun.Config.QuantityStepSize, InitialEquity: 1000, PositionSizePct: 1,
		Leverage: 4, StopLossPct: 6, TakeProfitPct: 8}
	if _, err := orm.NewOrm().Insert(&row); err != nil {
		t.Fatal(err)
	}
	var stored models.AgentBacktestRun
	if err := orm.NewOrm().QueryTable(new(models.AgentBacktestRun)).Filter("run_id", row.RunID).One(&stored); err != nil {
		t.Fatal(err)
	}
	if actual := summaryFromRow(stored).Config; actual.FixedMarginUSDT != 25 ||
		actual.QuantityStepSize != "0.001" || actual.PositionMode != PositionModeFixedMargin {
		t.Fatalf("db round-trip changed run sizing snapshot: %+v", actual)
	}
	legacy := summaryFromRow(models.AgentBacktestRun{PositionSizePct: 1})
	if legacy.Config.PositionMode != PositionModeEquityPercent {
		t.Fatalf("pre-migration historical run mislabelled: %+v", legacy.Config)
	}
}

func TestStartFixedMarginSnapshotsExchangePrecisionAndOverrides(t *testing.T) {
	setupBacktestStoreTest(t)
	coin := models.Symbols{Symbol: "BTCUSDT", Usdt: "20", Leverage: 4,
		Profit: "8", Loss: "6", StepSize: "0.001", TickSize: "0.1"}
	if _, err := orm.NewOrm().Insert(&coin); err != nil {
		t.Fatal(err)
	}
	template := models.StrategyTemplates{Name: "fixed-margin-fixture", Technology: "{}",
		Strategy: strategyJSON(
			Rule{Name: "open", Enable: true, Type: "long", Code: "NowPrice > 0"},
			Rule{Name: "close", Enable: true, Type: "close_long", Code: "ROI >= 8"},
		)}
	if _, err := orm.NewOrm().Insert(&template); err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	manager := NewManager(DatasetBuilder{Repository: historicalmarket.NewRepository(fixtureHistorySource{}), WarmupBars: 20})
	run, err := manager.Start(StartRequest{
		StrategyTemplateID: template.ID, Symbol: "btcusdt",
		StartTime: start.UnixMilli(), EndTime: start.Add(10 * time.Minute).UnixMilli(),
		Config: RunConfig{PositionMode: PositionModeFixedMargin,
			FixedMarginUSDT: 23, QuantityStepSize: "1", InitialEquity: 1000,
			PositionSizePct: 1, Leverage: 4, FeeRate: 0.0005, SlippageBps: 5,
			StopLossPct: 6, TakeProfitPct: 8},
	})
	if err != nil {
		t.Fatal(err)
	}
	if run.Config.FixedMarginUSDT != 23 || run.Config.QuantityStepSize != "0.001" ||
		run.Config.Leverage != 4 || run.Config.StopLossPct != 6 || run.Config.TakeProfitPct != 8 {
		t.Fatalf("start did not snapshot chosen params and authoritative StepSize: %+v", run.Config)
	}
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		item, err := manager.Get(run.RunID)
		if err != nil {
			t.Fatal(err)
		}
		if item.Status == "succeeded" {
			if item.Config.FixedMarginUSDT != 23 || item.Config.QuantityStepSize != "0.001" ||
				item.Config.PositionMode != PositionModeFixedMargin {
				t.Fatalf("stored run config corrupted: %+v", item.Config)
			}
			return
		}
		if item.Status == "failed" {
			t.Fatalf("fixed-margin run failed: %s", item.Error)
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("fixed-margin backtest did not finish")
}

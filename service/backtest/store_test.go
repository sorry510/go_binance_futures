package backtest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"go_binance_futures/models"
	"go_binance_futures/service/historicalmarket"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
)

var backtestStoreTestOnce sync.Once
var backtestStoreTestErr error

func setupBacktestStoreTest(t *testing.T) {
	t.Helper()
	backtestStoreTestOnce.Do(func() {
		backtestStoreTestErr = orm.RegisterDriver("sqlite3", orm.DRSqlite)
		if backtestStoreTestErr != nil {
			return
		}
		orm.RegisterModel(new(models.Config), new(models.MarketConditionHistory), new(models.StrategyTemplates), new(models.MarketKline1m), new(models.MarketKline1h), new(models.MarketFundingRate), new(models.MarketDataImportBatch), new(models.AgentBacktestDataset), new(models.AgentBacktestRun), new(models.AgentBacktestTrade), new(models.AgentBacktestEvent), new(models.AgentBacktestEquityPoint), new(models.AgentBacktestEquityChunk), new(models.AgentBacktestEquityPreview))
		dir, err := os.MkdirTemp("", "backtest-store-test-*")
		if err != nil {
			backtestStoreTestErr = err
			return
		}
		backtestStoreTestErr = orm.RegisterDataBase("default", "sqlite3", filepath.Join(dir, "backtest.db"))
		if backtestStoreTestErr != nil {
			return
		}
		backtestStoreTestErr = orm.RunSyncdb("default", true, false)
	})
	if backtestStoreTestErr != nil {
		t.Fatal(backtestStoreTestErr)
	}
	o := orm.NewOrm()
	for _, table := range []string{"agent_backtest_equity_preview", "agent_backtest_equity_chunks", "agent_backtest_equity_points", "agent_backtest_events", "agent_backtest_trades", "agent_backtest_runs", "agent_backtest_datasets", "market_data_import_batches", "market_funding_rates", "market_klines_1m", "market_klines_1h", "strategy_templates", "market_condition_histories", "config"} {
		if _, err := o.Raw("DELETE FROM " + table).Exec(); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := o.Insert(&models.Config{Version: 8, MarketCondition: 3}); err != nil {
		t.Fatal(err)
	}
}

type fixtureHistorySource struct{}

func (fixtureHistorySource) Klines(_ context.Context, market, symbol, interval string, start, end int64) ([]historicalmarket.Kline, error) {
	if interval != "1m" {
		return nil, fmt.Errorf("unexpected interval %s", interval)
	}
	rows := []historicalmarket.Kline{}
	cursor := time.UnixMilli(start).UTC().Truncate(time.Minute)
	for cursor.UnixMilli() < end {
		open := 100 + float64(len(rows))*0.1
		closePrice := open + 0.1
		rows = append(rows, historicalmarket.Kline{Market: market, Symbol: symbol, Interval: interval, OpenTime: cursor.UnixMilli(), CloseTime: cursor.Add(time.Minute - time.Millisecond).UnixMilli(), Open: open, High: closePrice + 0.2, Low: open - 0.2, Close: closePrice, Volume: 10, QuoteVolume: 1000, TakerBuyBaseVolume: 5, TakerBuyQuoteVolume: 500, TradeCount: 10, Source: historicalmarket.SourceBinanceREST})
		cursor = cursor.Add(time.Minute)
	}
	return rows, nil
}
func (fixtureHistorySource) Funding(context.Context, string, string, int64, int64) ([]historicalmarket.FundingRate, error) {
	return []historicalmarket.FundingRate{}, nil
}

func TestManagerPersistsDeterministicBacktestWithoutLegacyPaperTables(t *testing.T) {
	setupBacktestStoreTest(t)
	template := models.StrategyTemplates{Name: "fixture", Technology: "{}", Strategy: `[{"name":"open","enable":true,"code":"NowPrice > 0","type":"long"},{"name":"close","enable":true,"code":"ROI > 0.05","type":"close_long"}]`, CreateTime: time.Now().UnixMilli(), UpdateTime: time.Now().UnixMilli()}
	if _, err := orm.NewOrm().Insert(&template); err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	manager := NewManager(DatasetBuilder{Repository: historicalmarket.NewRepository(fixtureHistorySource{}), WarmupBars: 20})
	run, err := manager.Start(StartRequest{StrategyTemplateID: template.ID, Symbol: "BTCUSDT", StartTime: start.UnixMilli(), EndTime: start.Add(10 * time.Minute).UnixMilli(), Config: zeroCosts()})
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		detail, err := manager.Get(run.RunID)
		if err == nil && detail.Status == "succeeded" {
			if detail.ExecutionInterval != ReplayInterval || detail.Dataset == nil || detail.Dataset.ExecutionInterval != ReplayInterval {
				t.Fatalf("new backtests must always use %s replay: %+v", ReplayInterval, detail)
			}
			if detail.Dataset.DatasetSpecHash == "" || detail.DataHash == "" {
				t.Fatalf("dataset not persisted: %+v", detail)
			}
			trades, err := manager.Trades(run.RunID, 100)
			if err != nil || len(trades) == 0 {
				t.Fatalf("trades not persisted: %+v err=%v", trades, err)
			}
			equity, err := manager.Equity(run.RunID, 1000)
			if err != nil || len(equity) == 0 {
				t.Fatalf("equity not persisted: %d err=%v", len(equity), err)
			}
			var count int
			if err := orm.NewOrm().Raw("SELECT COUNT(*) FROM agent_backtest_events WHERE run_id=?", run.RunID).QueryRow(&count); err != nil || count == 0 {
				t.Fatalf("audit events missing count=%d err=%v", count, err)
			}
			if err := orm.NewOrm().Raw("SELECT COUNT(*) FROM market_klines_1m").QueryRow(&count); err != nil || count == 0 {
				t.Fatalf("global historical cache missing count=%d err=%v", count, err)
			}
			if err := orm.NewOrm().Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name IN ('agent_backtest_bars','agent_backtest_funding')").QueryRow(&count); err != nil || count != 0 {
				t.Fatalf("legacy dataset-owned history tables must not exist count=%d err=%v", count, err)
			}
			return
		} else if err == nil && detail.Status == "failed" {
			t.Fatalf("backtest failed: %+v", detail)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("backtest run did not finish")
}

func waitBacktestSucceeded(t *testing.T, manager *Manager, runID string) RunDetail {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		detail, err := manager.Get(runID)
		if err == nil && detail.Status == "succeeded" {
			return detail
		}
		if err == nil && (detail.Status == "failed" || detail.Status == "cancelled") {
			t.Fatalf("backtest ended unexpectedly: %+v", detail)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("backtest %s did not finish", runID)
	return RunDetail{}
}

func TestSameDatasetSpecUsesLatestCanonicalMarketData(t *testing.T) {
	setupBacktestStoreTest(t)
	template := models.StrategyTemplates{Name: "latest-data", Technology: "{}", Strategy: `[{"name":"open","enable":true,"code":"NowPrice > 0","type":"long"}]`, CreateTime: time.Now().UnixMilli(), UpdateTime: time.Now().UnixMilli()}
	if _, err := orm.NewOrm().Insert(&template); err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	repo := historicalmarket.NewRepository(fixtureHistorySource{})
	manager := NewManager(DatasetBuilder{Repository: repo, WarmupBars: 20})
	request := StartRequest{StrategyTemplateID: template.ID, Symbol: "BTCUSDT", StartTime: start.UnixMilli(), EndTime: start.Add(10 * time.Minute).UnixMilli(), Config: zeroCosts()}
	first, err := manager.Start(request)
	if err != nil {
		t.Fatal(err)
	}
	firstDetail := waitBacktestSucceeded(t, manager, first.RunID)
	if firstDetail.DatasetID == "" || firstDetail.DatasetSpecHash == "" || firstDetail.DataHash == "" {
		t.Fatalf("missing first hashes: %+v", firstDetail)
	}
	overwrite := historicalmarket.Kline{Market: historicalmarket.MarketFuturesUSDT, Symbol: "BTCUSDT", Interval: "1m", OpenTime: start.UnixMilli(), CloseTime: start.Add(time.Minute - time.Millisecond).UnixMilli(), Open: 149, High: 151, Low: 148, Close: 150, Volume: 999, QuoteVolume: 149850, Source: "external_csv"}
	if _, err := repo.Import(context.Background(), historicalmarket.ImportRequest{Source: "external_csv", Klines: []historicalmarket.Kline{overwrite}}); err != nil {
		t.Fatal(err)
	}
	second, err := manager.Start(request)
	if err != nil {
		t.Fatal(err)
	}
	secondDetail := waitBacktestSucceeded(t, manager, second.RunID)
	if secondDetail.DatasetID != firstDetail.DatasetID || secondDetail.DatasetSpecHash != firstDetail.DatasetSpecHash {
		t.Fatalf("dataset spec changed after data overwrite: first=%+v second=%+v", firstDetail.RunSummary, secondDetail.RunSummary)
	}
	if secondDetail.DataHash == firstDetail.DataHash {
		t.Fatalf("data hash must change after canonical OHLCV overwrite: %s", firstDetail.DataHash)
	}
}

func TestBacktestTradeAndEventPagination(t *testing.T) {
	setupBacktestStoreTest(t)
	o := orm.NewOrm()
	run := models.AgentBacktestRun{RunID: "bt_page_fixture", Status: "succeeded", Stage: "completed"}
	if _, err := o.Insert(&run); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 45; i++ {
		if _, err := o.Insert(&models.AgentBacktestTrade{RunID: run.RunID, Sequence: i, Symbol: "BTCUSDT", Side: "LONG", EntryResolution: "1m", ExitResolution: "1m"}); err != nil {
			t.Fatal(err)
		}
	}
	for i := 1; i <= 120; i++ {
		if _, err := o.Insert(&models.AgentBacktestEvent{RunID: run.RunID, Sequence: i, Type: "fixture", Action: "event"}); err != nil {
			t.Fatal(err)
		}
	}
	manager := NewManager(DatasetBuilder{})
	trades, totalTrades, err := manager.TradesPage(run.RunID, 2, 20)
	if err != nil {
		t.Fatal(err)
	}
	if totalTrades != 45 || len(trades) != 20 || trades[0].Sequence != 21 || trades[19].Sequence != 40 {
		t.Fatalf("unexpected trade page: total=%d rows=%+v", totalTrades, trades)
	}
	events, totalEvents, err := manager.EventsPage(run.RunID, 3, 50)
	if err != nil {
		t.Fatal(err)
	}
	if totalEvents != 120 || len(events) != 20 || events[0].Sequence != 101 || events[19].Sequence != 120 {
		t.Fatalf("unexpected event page: total=%d rows=%+v", totalEvents, events)
	}
}

func TestManagerDeleteRemovesRunChildrenAndUnusedDataset(t *testing.T) {
	setupBacktestStoreTest(t)
	o := orm.NewOrm()
	dataset := models.AgentBacktestDataset{
		DatasetID: "ds_delete_fixture", DatasetSpecHash: "spec_delete_fixture", Symbol: "BTCUSDT",
		ExecutionInterval: "1m", IntervalsJSON: `["1m"]`, BenchmarkSymbolsJSON: `[]`, Market: "futures_usdt",
	}
	if _, err := o.Insert(&dataset); err != nil {
		t.Fatal(err)
	}
	run := models.AgentBacktestRun{RunID: "bt_delete_fixture", DatasetID: dataset.DatasetID, Status: "succeeded", Stage: "completed"}
	if _, err := o.Insert(&run); err != nil {
		t.Fatal(err)
	}
	if _, err := o.Insert(&models.AgentBacktestTrade{RunID: run.RunID, Sequence: 1, Symbol: "BTCUSDT", Side: "LONG"}); err != nil {
		t.Fatal(err)
	}
	if _, err := o.Insert(&models.AgentBacktestEvent{RunID: run.RunID, Sequence: 1, Type: "position", Action: "closed"}); err != nil {
		t.Fatal(err)
	}
	if _, err := o.Insert(&models.AgentBacktestEquityPoint{RunID: run.RunID, Sequence: 1}); err != nil {
		t.Fatal(err)
	}
	chunks, preview, err := buildEquityStorage(run.RunID, []EquityPoint{{Sequence: 1, BarTime: 1, Equity: 1000, Cash: 1000}}, time.Now().UnixMilli())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := o.Insert(&chunks[0]); err != nil {
		t.Fatal(err)
	}
	if _, err := o.Insert(preview); err != nil {
		t.Fatal(err)
	}

	manager := NewManager(DatasetBuilder{})
	if err := manager.Delete(run.RunID); err != nil {
		t.Fatal(err)
	}
	for name, model := range map[string]interface{}{
		"run": new(models.AgentBacktestRun), "trade": new(models.AgentBacktestTrade),
		"event": new(models.AgentBacktestEvent), "legacy_equity": new(models.AgentBacktestEquityPoint),
		"equity_chunk": new(models.AgentBacktestEquityChunk), "equity_preview": new(models.AgentBacktestEquityPreview),
	} {
		count, err := o.QueryTable(model).Filter("run_id", run.RunID).Count()
		if err != nil || count != 0 {
			t.Fatalf("%s rows remain after delete: count=%d err=%v", name, count, err)
		}
	}
	if exists := o.QueryTable(new(models.AgentBacktestDataset)).Filter("dataset_id", dataset.DatasetID).Exist(); exists {
		t.Fatal("unused dataset manifest should be deleted with its last run")
	}
}

func TestManagerDeleteKeepsSharedDatasetAndRejectsActiveRun(t *testing.T) {
	setupBacktestStoreTest(t)
	o := orm.NewOrm()
	dataset := models.AgentBacktestDataset{
		DatasetID: "ds_shared_fixture", DatasetSpecHash: "spec_shared_fixture", Symbol: "BTCUSDT",
		ExecutionInterval: "1m", IntervalsJSON: `["1m"]`, BenchmarkSymbolsJSON: `[]`, Market: "futures_usdt",
	}
	if _, err := o.Insert(&dataset); err != nil {
		t.Fatal(err)
	}
	first := models.AgentBacktestRun{RunID: "bt_shared_first", DatasetID: dataset.DatasetID, Status: "succeeded", Stage: "completed"}
	second := models.AgentBacktestRun{RunID: "bt_shared_second", DatasetID: dataset.DatasetID, Status: "succeeded", Stage: "completed"}
	active := models.AgentBacktestRun{RunID: "bt_active_fixture", Status: "running", Stage: "running_backtest"}
	for _, row := range []*models.AgentBacktestRun{&first, &second, &active} {
		if _, err := o.Insert(row); err != nil {
			t.Fatal(err)
		}
	}
	manager := NewManager(DatasetBuilder{})
	if err := manager.Delete(first.RunID); err != nil {
		t.Fatal(err)
	}
	if !o.QueryTable(new(models.AgentBacktestDataset)).Filter("dataset_id", dataset.DatasetID).Exist() {
		t.Fatal("shared dataset manifest must remain while another run references it")
	}
	if err := manager.Delete(active.RunID); err == nil {
		t.Fatal("active backtest run must not be deletable")
	}
	if !o.QueryTable(new(models.AgentBacktestRun)).Filter("run_id", active.RunID).Exist() {
		t.Fatal("active run was deleted despite rejection")
	}
}

func TestManagerDeleteHandlesVeryLargeEquityHistoryWithoutPlaceholderExpansion(t *testing.T) {
	setupBacktestStoreTest(t)
	o := orm.NewOrm()
	run := models.AgentBacktestRun{RunID: "bt_large_delete_fixture", Status: "succeeded", Stage: "completed"}
	if _, err := o.Insert(&run); err != nil {
		t.Fatal(err)
	}
	_, err := o.Raw(`WITH RECURSIVE seq(x) AS (
		SELECT 1
		UNION ALL
		SELECT x + 1 FROM seq WHERE x < 70000
	)
	INSERT INTO agent_backtest_equity_points
		(run_id, sequence, bar_time, equity, cash, unrealized_pnl, drawdown_pct, position_side)
	SELECT ?, x, x, 1000, 1000, 0, 0, '' FROM seq`, run.RunID).Exec()
	if err != nil {
		t.Fatal(err)
	}
	count, err := o.QueryTable(new(models.AgentBacktestEquityPoint)).Filter("run_id", run.RunID).Count()
	if err != nil || count != 70000 {
		t.Fatalf("large fixture count=%d err=%v", count, err)
	}
	if err := NewManager(DatasetBuilder{}).Delete(run.RunID); err != nil {
		t.Fatal(err)
	}
	count, err = o.QueryTable(new(models.AgentBacktestEquityPoint)).Filter("run_id", run.RunID).Count()
	if err != nil || count != 0 {
		t.Fatalf("large equity history remains after delete: count=%d err=%v", count, err)
	}
}

func TestBacktestRejectsStrategyUsingMarketCondition(t *testing.T) {
	setupBacktestStoreTest(t)
	template := models.StrategyTemplates{
		Name: "market-condition-backtest-blocked", Technology: "{}",
		Strategy:   `[{"name":"open","enable":true,"code":"MarketCondition == \"2\" && NowPrice > 0","type":"long"}]`,
		CreateTime: time.Now().UnixMilli(), UpdateTime: time.Now().UnixMilli(),
	}
	if _, err := orm.NewOrm().Insert(&template); err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	manager := NewManager(DatasetBuilder{Repository: historicalmarket.NewRepository(fixtureHistorySource{})})
	_, err := manager.Start(StartRequest{StrategyTemplateID: template.ID, Symbol: "BTCUSDT", StartTime: start.UnixMilli(), EndTime: start.Add(time.Hour).UnixMilli(), Config: zeroCosts()})
	if err == nil || !strings.Contains(err.Error(), "MarketCondition") || !strings.Contains(err.Error(), "补充") {
		t.Fatalf("expected missing MarketCondition history rejection, got %v", err)
	}
	count, countErr := orm.NewOrm().QueryTable(new(models.AgentBacktestRun)).Count()
	if countErr != nil || count != 0 {
		t.Fatalf("rejected backtest must not create a run: count=%d err=%v", count, countErr)
	}
}

func TestBacktestPrefetchRejectsStrategyUsingMarketCondition(t *testing.T) {
	setupBacktestStoreTest(t)
	template := models.StrategyTemplates{
		Name: "market-condition-prefetch-blocked", Technology: "{}",
		Strategy:   `[{"name":"open","enable":true,"code":"MarketCondition == \"3\"","type":"long"}]`,
		CreateTime: time.Now().UnixMilli(), UpdateTime: time.Now().UnixMilli(),
	}
	if _, err := orm.NewOrm().Insert(&template); err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	manager := NewManager(DatasetBuilder{Repository: historicalmarket.NewRepository(fixtureHistorySource{})})
	_, err := manager.StartPrefetch(PrefetchRequest{StrategyTemplateID: template.ID, Symbol: "BTCUSDT", StartTime: start.UnixMilli(), EndTime: start.Add(time.Hour).UnixMilli()})
	if err == nil || !strings.Contains(err.Error(), "MarketCondition") || !strings.Contains(err.Error(), "补充") {
		t.Fatalf("expected missing MarketCondition history prefetch rejection, got %v", err)
	}
}

func TestBacktestUsingMarketConditionRunsWhenHistoryIsAvailable(t *testing.T) {
	setupBacktestStoreTest(t)
	template := models.StrategyTemplates{
		Name: "market-condition-backtest-supported", Technology: "{}",
		Strategy:   `[{"name":"open","enable":true,"code":"MarketCondition == \"2\" && NowPrice > 0","type":"long"}]`,
		CreateTime: time.Now().UnixMilli(), UpdateTime: time.Now().UnixMilli(),
	}
	if _, err := orm.NewOrm().Insert(&template); err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	config, err := loadBacktestSystemConfig()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := orm.NewOrm().Insert(&models.MarketConditionHistory{ConfigID: config.ID, MarketCondition: 2, CreatedAt: start.Add(-time.Minute).UnixMilli()}); err != nil {
		t.Fatal(err)
	}
	manager := NewManager(DatasetBuilder{Repository: historicalmarket.NewRepository(fixtureHistorySource{}), WarmupBars: 2})
	run, err := manager.Start(StartRequest{StrategyTemplateID: template.ID, Symbol: "BTCUSDT", StartTime: start.UnixMilli(), EndTime: start.Add(10 * time.Minute).UnixMilli(), Config: zeroCosts()})
	if err != nil {
		t.Fatalf("historical MarketCondition should allow backtest: %v", err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		detail, getErr := manager.Get(run.RunID)
		if getErr != nil {
			t.Fatal(getErr)
		}
		if detail.Status == "succeeded" {
			return
		}
		if detail.Status == "failed" || detail.Status == "cancelled" || detail.Status == "interrupted" {
			t.Fatalf("backtest with MarketCondition history failed: %+v", detail)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("backtest with MarketCondition history did not finish")
}

func TestManagerPersistsAdaptiveResolutionMetadataWithoutChangingV6Execution(t *testing.T) {
	setupBacktestStoreTest(t)
	template := models.StrategyTemplates{Name: "adaptive-fixture", Technology: "{}", Strategy: `[{"name":"open","enable":true,"code":"NowPrice > 0","type":"long"}]`, CreateTime: time.Now().UnixMilli(), UpdateTime: time.Now().UnixMilli()}
	if _, err := orm.NewOrm().Insert(&template); err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)
	manager := NewManager(DatasetBuilder{Repository: historicalmarket.NewRepository(fixtureHistorySource{}), WarmupBars: 20})
	run, err := manager.Start(StartRequest{StrategyTemplateID: template.ID, ResolutionMode: ResolutionModeAdaptive, Symbol: "BTCUSDT", StartTime: start.UnixMilli(), EndTime: start.Add(10 * time.Minute).UnixMilli(), Config: zeroCosts()})
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		detail, err := manager.Get(run.RunID)
		if err == nil && detail.Status == "succeeded" {
			if detail.ResolutionMode != ResolutionModeAdaptive || detail.ResolutionModel != AdaptiveResolutionModel || detail.EngineVersion != AdaptiveEngineVersion {
				t.Fatalf("adaptive metadata was not persisted: %+v", detail.RunSummary)
			}
			stats := detail.ResolutionStats
			timing := stats.Timing
			stats.Timing = nil
			if stats != (ResolutionStats{}) {
				t.Fatalf("no-ambiguity adaptive run unexpectedly drilled down: %+v", detail.ResolutionStats)
			}
			if timing == nil || timing.TotalMs <= 0 {
				t.Fatalf("backtest timing metadata was not persisted: %+v", detail.ResolutionStats)
			}
			trades, err := manager.Trades(run.RunID, 100)
			if err != nil || len(trades) == 0 {
				t.Fatalf("adaptive trades missing: %+v err=%v", trades, err)
			}
			for _, trade := range trades {
				if trade.EntryResolution != "1m" || trade.ExitResolution != "1m" {
					t.Fatalf("V3-4A no-ambiguity trade must remain 1m: %+v", trade)
				}
			}
			return
		}
		if err == nil && detail.Status == "failed" {
			t.Fatalf("adaptive backtest failed: %+v", detail)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("adaptive backtest did not finish")
}

func TestManagerRejectsUnsupportedResolutionMode(t *testing.T) {
	setupBacktestStoreTest(t)
	manager := NewManager(DatasetBuilder{})
	_, err := manager.Start(StartRequest{StrategyTemplateID: 1, ResolutionMode: "tick_everything", Symbol: "BTCUSDT", StartTime: 1, EndTime: 2})
	if err == nil || !strings.Contains(err.Error(), "unsupported resolution_mode") {
		t.Fatalf("unsupported resolution mode must fail closed, got %v", err)
	}
}

func TestEquitySamplesAcrossEntireRange(t *testing.T) {
	setupBacktestStoreTest(t)
	o := orm.NewOrm()
	const runID = "run-equity-sample"
	start := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	rows := make([]models.AgentBacktestEquityPoint, 0, 101)
	for i := 1; i <= 101; i++ {
		rows = append(rows, models.AgentBacktestEquityPoint{
			RunID: runID, Sequence: i, BarTime: start.Add(time.Duration(i-1) * 24 * time.Hour).UnixMilli(),
			Equity: 1000 + float64(i), Cash: 1000, DrawdownPct: float64(i % 7),
		})
	}
	if _, err := o.InsertMulti(100, rows); err != nil {
		t.Fatal(err)
	}
	manager := NewManager(DatasetBuilder{})
	points, err := manager.Equity(runID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 10 {
		t.Fatalf("sampled points=%d want=10: %+v", len(points), points)
	}
	if points[0].Sequence != 1 || points[len(points)-1].Sequence != 101 {
		t.Fatalf("sample must cover full range: first=%d last=%d", points[0].Sequence, points[len(points)-1].Sequence)
	}
	for i := 1; i < len(points); i++ {
		if points[i].Sequence <= points[i-1].Sequence {
			t.Fatalf("sample not strictly ordered: %+v", points)
		}
	}
}

func TestEquityReturnsAllPointsWhenWithinLimit(t *testing.T) {
	setupBacktestStoreTest(t)
	o := orm.NewOrm()
	const runID = "run-equity-small"
	for i := 1; i <= 5; i++ {
		if _, err := o.Insert(&models.AgentBacktestEquityPoint{RunID: runID, Sequence: i, BarTime: int64(i), Equity: float64(i), Cash: float64(i)}); err != nil {
			t.Fatal(err)
		}
	}
	points, err := NewManager(DatasetBuilder{}).Equity(runID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 5 || points[0].Sequence != 1 || points[4].Sequence != 5 {
		t.Fatalf("unexpected full equity response: %+v", points)
	}
}

func TestManagerStartDeleteRunsLargeHistoryAsynchronously(t *testing.T) {
	setupBacktestStoreTest(t)
	o := orm.NewOrm()
	run := models.AgentBacktestRun{RunID: "bt_async_delete_fixture", Status: "succeeded", Stage: "completed", Progress: 100}
	if _, err := o.Insert(&run); err != nil {
		t.Fatal(err)
	}
	if _, err := o.Raw(`WITH RECURSIVE seq(x) AS (
		SELECT 1
		UNION ALL
		SELECT x + 1 FROM seq WHERE x < 70000
	)
	INSERT INTO agent_backtest_equity_points
		(run_id, sequence, bar_time, equity, cash, unrealized_pnl, drawdown_pct, position_side)
	SELECT ?, x, x, 1000, 1000, 0, 0, '' FROM seq`, run.RunID).Exec(); err != nil {
		t.Fatal(err)
	}
	manager := NewManager(DatasetBuilder{})
	started := time.Now()
	if err := manager.StartDelete(run.RunID); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("async delete request took too long before returning: %v", elapsed)
	}
	var current models.AgentBacktestRun
	if err := o.QueryTable(new(models.AgentBacktestRun)).Filter("run_id", run.RunID).One(&current); err == nil {
		if current.Status != "deleting" {
			t.Fatalf("run should be marked deleting while background cleanup is active: %+v", current)
		}
	} else if err != orm.ErrNoRows {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if !o.QueryTable(new(models.AgentBacktestRun)).Filter("run_id", run.RunID).Exist() {
			count, err := o.QueryTable(new(models.AgentBacktestEquityPoint)).Filter("run_id", run.RunID).Count()
			if err != nil || count != 0 {
				t.Fatalf("equity rows remain after async delete: count=%d err=%v", count, err)
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("async delete did not finish within timeout")
}

func TestSaveResultWithProgressPersistsAllBatches(t *testing.T) {
	setupBacktestStoreTest(t)
	const runID = "bt_save_result_batches"
	result := Result{
		Trades: make([]Trade, 0, 1001),
		Events: make([]AuditEvent, 0, 1001),
		Equity: make([]EquityPoint, 0, 5201),
	}
	for i := 1; i <= 1001; i++ {
		result.Trades = append(result.Trades, Trade{Sequence: i, Symbol: "BTCUSDT", Side: "LONG", EntryTime: int64(i), ExitTime: int64(i + 1), EntryPrice: 100, ExitPrice: 101, Quantity: 1, ExitReason: "test", EntryResolution: "1m", ExitResolution: "1m"})
		result.Events = append(result.Events, AuditEvent{Sequence: i, EventTime: int64(i), Type: "signal", Action: "test"})
	}
	for i := 1; i <= 5201; i++ {
		result.Equity = append(result.Equity, EquityPoint{Sequence: i, BarTime: int64(i), Equity: 1000 + float64(i), Cash: 1000})
	}
	lastCompleted, lastTotal := -1, -1
	if err := saveResultWithProgress(context.Background(), runID, result, func(completed, total int) {
		lastCompleted, lastTotal = completed, total
	}); err != nil {
		t.Fatal(err)
	}
	o := orm.NewOrm()
	trades, _ := o.QueryTable(new(models.AgentBacktestTrade)).Filter("run_id", runID).Count()
	events, _ := o.QueryTable(new(models.AgentBacktestEvent)).Filter("run_id", runID).Count()
	legacy, _ := o.QueryTable(new(models.AgentBacktestEquityPoint)).Filter("run_id", runID).Count()
	chunks, _ := o.QueryTable(new(models.AgentBacktestEquityChunk)).Filter("run_id", runID).Count()
	previews, _ := o.QueryTable(new(models.AgentBacktestEquityPreview)).Filter("run_id", runID).Count()
	if trades != 1001 || events != 1001 || legacy != 0 || chunks != 1 || previews != 1 {
		t.Fatalf("persisted rows trades=%d events=%d legacy=%d chunks=%d previews=%d", trades, events, legacy, chunks, previews)
	}
	points, err := NewManager(DatasetBuilder{}).Equity(runID, maxEquityChartPoints)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != len(result.Equity) || points[0] != result.Equity[0] || points[len(points)-1] != result.Equity[len(result.Equity)-1] {
		t.Fatalf("decoded equity differs: points=%d want=%d first=%+v last=%+v", len(points), len(result.Equity), points[0], points[len(points)-1])
	}
	if lastCompleted != 7203 || lastTotal != 7203 {
		t.Fatalf("progress completed=%d total=%d", lastCompleted, lastTotal)
	}
}

func TestSaveResultRejectsInvalidEquityBeforeWritingChildren(t *testing.T) {
	setupBacktestStoreTest(t)
	const runID = "bt_atomic_save_failure"
	result := Result{
		Trades: []Trade{{Sequence: 1, Symbol: "BTCUSDT", Side: "LONG", EntryTime: 1, ExitTime: 2, EntryPrice: 100, ExitPrice: 101, Quantity: 1, ExitReason: "test"}},
		Events: []AuditEvent{{Sequence: 1, EventTime: 1, Type: "signal", Action: "test"}},
		Equity: []EquityPoint{{Sequence: 1, BarTime: 1, Equity: 1000, Cash: 1000, PositionSide: "POSITION_SIDE_TOO_LONG"}},
	}
	if err := saveResultWithProgress(context.Background(), runID, result, nil); err == nil {
		t.Fatal("invalid equity payload must fail")
	}
	o := orm.NewOrm()
	for name, model := range map[string]interface{}{
		"trade":          new(models.AgentBacktestTrade),
		"event":          new(models.AgentBacktestEvent),
		"legacy_equity":  new(models.AgentBacktestEquityPoint),
		"equity_chunk":   new(models.AgentBacktestEquityChunk),
		"equity_preview": new(models.AgentBacktestEquityPreview),
	} {
		count, err := o.QueryTable(model).Filter("run_id", runID).Count()
		if err != nil || count != 0 {
			t.Fatalf("%s rows survived rollback: count=%d err=%v", name, count, err)
		}
	}
}

func TestSaveResultTransactionRollsBackTradesAndEventsOnChunkFailure(t *testing.T) {
	setupBacktestStoreTest(t)
	const runID = "bt_atomic_chunk_failure"
	o := orm.NewOrm()
	existingChunks, _, err := buildEquityStorage(runID, []EquityPoint{{Sequence: 1, BarTime: 1, Equity: 1000, Cash: 1000}}, time.Now().UnixMilli())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := o.Insert(&existingChunks[0]); err != nil {
		t.Fatal(err)
	}
	result := Result{
		Trades: []Trade{{Sequence: 1, Symbol: "BTCUSDT", Side: "LONG", EntryTime: 1, ExitTime: 2, EntryPrice: 100, ExitPrice: 101, Quantity: 1, ExitReason: "test"}},
		Events: []AuditEvent{{Sequence: 1, EventTime: 1, Type: "signal", Action: "test"}},
		Equity: []EquityPoint{{Sequence: 1, BarTime: 1, Equity: 1000, Cash: 1000}},
	}
	if err := saveResultWithProgress(context.Background(), runID, result, nil); err == nil {
		t.Fatal("duplicate equity chunk must fail")
	}
	for name, model := range map[string]interface{}{
		"trade":   new(models.AgentBacktestTrade),
		"event":   new(models.AgentBacktestEvent),
		"preview": new(models.AgentBacktestEquityPreview),
	} {
		count, err := o.QueryTable(model).Filter("run_id", runID).Count()
		if err != nil || count != 0 {
			t.Fatalf("%s rows survived rollback: count=%d err=%v", name, count, err)
		}
	}
	chunkCount, err := o.QueryTable(new(models.AgentBacktestEquityChunk)).Filter("run_id", runID).Count()
	if err != nil || chunkCount != 1 {
		t.Fatalf("pre-existing chunk changed by rollback: count=%d err=%v", chunkCount, err)
	}
}

func TestEquityStorageUsesChunksAndPreview(t *testing.T) {
	setupBacktestStoreTest(t)
	const runID = "bt_chunked_equity"
	result := Result{Equity: make([]EquityPoint, 0, 70000)}
	for i := 1; i <= 70000; i++ {
		side := ""
		if i%3 == 1 {
			side = "LONG"
		} else if i%3 == 2 {
			side = "SHORT"
		}
		result.Equity = append(result.Equity, EquityPoint{
			Sequence: i, BarTime: int64(i) * 60000, Equity: 1000.123456789 + float64(i)/7,
			Cash: 900.987654321 + float64(i)/11, UnrealizedPnL: float64(i%97) / 13,
			DrawdownPct: float64(i%31) / 17, PositionSide: side,
		})
	}
	if err := saveResultWithProgress(context.Background(), runID, result, nil); err != nil {
		t.Fatal(err)
	}
	o := orm.NewOrm()
	legacy, _ := o.QueryTable(new(models.AgentBacktestEquityPoint)).Filter("run_id", runID).Count()
	chunks, _ := o.QueryTable(new(models.AgentBacktestEquityChunk)).Filter("run_id", runID).Count()
	if legacy != 0 || chunks != 3 {
		t.Fatalf("legacy=%d chunks=%d want legacy=0 chunks=3", legacy, chunks)
	}
	preview, err := NewManager(DatasetBuilder{}).Equity(runID, maxEquityChartPoints)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview) != maxEquityChartPoints || preview[0].Sequence != 1 || preview[len(preview)-1].Sequence != 70000 {
		t.Fatalf("preview range invalid: len=%d first=%d last=%d", len(preview), preview[0].Sequence, preview[len(preview)-1].Sequence)
	}
	small, err := NewManager(DatasetBuilder{}).Equity(runID, 10)
	if err != nil {
		t.Fatal(err)
	}
	targets := equitySampleSequences(1, 70000, 10)
	if len(small) != len(targets) {
		t.Fatalf("small sample len=%d want=%d", len(small), len(targets))
	}
	for i, target := range targets {
		if small[i] != result.Equity[target-1] {
			t.Fatalf("sample %d differs: got=%+v want=%+v", i, small[i], result.Equity[target-1])
		}
	}
}

func TestEquityCodecRoundTripAndChecksum(t *testing.T) {
	points := []EquityPoint{
		{Sequence: 1, BarTime: 1001, Equity: 1000.1234567890123, Cash: 999.9876543210987, UnrealizedPnL: 0.13579, DrawdownPct: 0.2468},
		{Sequence: 2, BarTime: 2001, Equity: 1001.0000000000002, Cash: 998.5, UnrealizedPnL: 2.5000000000001, DrawdownPct: 1.25, PositionSide: "LONG"},
		{Sequence: 3, BarTime: 3001, Equity: 997.75, Cash: 997.75, UnrealizedPnL: -3.25, DrawdownPct: 2.125, PositionSide: "SHORT"},
	}
	encoded, err := encodeEquityPayload(points)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := decodeEquityPayload(encoded.Payload, encoded.Checksum)
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded) != len(points) {
		t.Fatalf("decoded=%d want=%d", len(decoded), len(points))
	}
	for i := range points {
		if decoded[i] != points[i] {
			t.Fatalf("point %d changed: got=%+v want=%+v", i, decoded[i], points[i])
		}
	}
	if _, err := decodeEquityPayload(encoded.Payload, encoded.Checksum+1); err == nil {
		t.Fatal("checksum mismatch must fail")
	}
}

func TestBacktestResultMySQLBatchTargetsStayWithinPlaceholderLimits(t *testing.T) {
	if backtestEventInsertBatchSizeMySQL != 3000 {
		t.Fatalf("mysql event batch=%d want=3000", backtestEventInsertBatchSizeMySQL)
	}
	if backtestTradeInsertBatchSizeMySQL != 2000 {
		t.Fatalf("mysql trade batch=%d want=2000", backtestTradeInsertBatchSizeMySQL)
	}
	if equityChunkInsertBatchSize != 1 {
		t.Fatalf("equity chunk insert batch=%d want=1", equityChunkInsertBatchSize)
	}
	const tradeColumns = 24
	if backtestTradeInsertBatchSizeMySQL*tradeColumns > 65535 {
		t.Fatalf("trade batch would exceed MySQL placeholder limit: %d", backtestTradeInsertBatchSizeMySQL*tradeColumns)
	}
	const eventColumns = 9
	if backtestEventInsertBatchSizeMySQL*eventColumns > 65535 {
		t.Fatalf("event batch would exceed MySQL placeholder limit: %d", backtestEventInsertBatchSizeMySQL*eventColumns)
	}
}

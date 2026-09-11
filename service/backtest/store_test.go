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
		orm.RegisterModel(new(models.Config), new(models.MarketConditionHistory), new(models.StrategyTemplates), new(models.MarketKline1m), new(models.MarketKline1h), new(models.MarketFundingRate), new(models.MarketDataImportBatch), new(models.AgentBacktestDataset), new(models.AgentBacktestRun), new(models.AgentBacktestTrade), new(models.AgentBacktestEvent), new(models.AgentBacktestEquityPoint))
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
	for _, table := range []string{"agent_backtest_equity_points", "agent_backtest_events", "agent_backtest_trades", "agent_backtest_runs", "agent_backtest_datasets", "market_data_import_batches", "market_funding_rates", "market_klines_1m", "market_klines_1h", "strategy_templates", "market_condition_histories", "config"} {
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

	manager := NewManager(DatasetBuilder{})
	if err := manager.Delete(run.RunID); err != nil {
		t.Fatal(err)
	}
	for name, model := range map[string]interface{}{
		"run": new(models.AgentBacktestRun), "trade": new(models.AgentBacktestTrade),
		"event": new(models.AgentBacktestEvent), "equity": new(models.AgentBacktestEquityPoint),
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
			if detail.ResolutionStats != (ResolutionStats{}) {
				t.Fatalf("no-ambiguity adaptive run unexpectedly drilled down: %+v", detail.ResolutionStats)
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

package backtest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
		orm.RegisterModel(new(models.StrategyTemplates), new(models.MarketKline1m), new(models.MarketFundingRate), new(models.MarketDataImportBatch), new(models.AgentBacktestDataset), new(models.AgentBacktestRun), new(models.AgentBacktestTrade), new(models.AgentBacktestEvent), new(models.AgentBacktestEquityPoint))
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
	for _, table := range []string{"agent_backtest_equity_points", "agent_backtest_events", "agent_backtest_trades", "agent_backtest_runs", "agent_backtest_datasets", "market_data_import_batches", "market_funding_rates", "market_klines_1m", "strategy_templates"} {
		if _, err := o.Raw("DELETE FROM " + table).Exec(); err != nil {
			t.Fatal(err)
		}
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
	run, err := manager.Start(StartRequest{StrategyTemplateID: template.ID, Symbol: "BTCUSDT", ExecutionInterval: "1m", StartTime: start.UnixMilli(), EndTime: start.Add(10 * time.Minute).UnixMilli(), Config: zeroCosts()})
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		detail, err := manager.Get(run.RunID)
		if err == nil && detail.Status == "succeeded" {
			if detail.Dataset == nil || detail.Dataset.DatasetSpecHash == "" || detail.DataHash == "" {
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
	request := StartRequest{StrategyTemplateID: template.ID, Symbol: "BTCUSDT", ExecutionInterval: "1m", StartTime: start.UnixMilli(), EndTime: start.Add(10 * time.Minute).UnixMilli(), Config: zeroCosts()}
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

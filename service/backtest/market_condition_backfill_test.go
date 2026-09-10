package backtest

import (
	"context"
	"testing"
	"time"

	"go_binance_futures/models"
	"go_binance_futures/service/historicalmarket"
	markettypes "go_binance_futures/types"

	"github.com/beego/beego/v2/client/orm"
)

type marketConditionHistoryFixtureSource struct {
	start int64
}

func (source marketConditionHistoryFixtureSource) EarliestKline(_ context.Context, market, symbol, interval string) (historicalmarket.Kline, error) {
	return marketConditionFixtureKline(market, symbol, interval, source.start, 100), nil
}

func (source marketConditionHistoryFixtureSource) Klines(_ context.Context, market, symbol, interval string, start, end int64) ([]historicalmarket.Kline, error) {
	rows := make([]historicalmarket.Kline, 0)
	cursor := time.UnixMilli(start).UTC().Truncate(time.Hour)
	for cursor.UnixMilli() <= end {
		base := 100.0
		if symbol == "ETHUSDT" {
			base = 200.0
		}
		rows = append(rows, marketConditionFixtureKline(market, symbol, interval, cursor.UnixMilli(), base+float64(len(rows))))
		cursor = cursor.Add(time.Hour)
	}
	return rows, nil
}

func (marketConditionHistoryFixtureSource) Funding(context.Context, string, string, int64, int64) ([]historicalmarket.FundingRate, error) {
	return nil, nil
}

func marketConditionFixtureKline(market, symbol, interval string, openTime int64, open float64) historicalmarket.Kline {
	return historicalmarket.Kline{
		Market: market, Symbol: symbol, Interval: interval,
		OpenTime: openTime, CloseTime: time.UnixMilli(openTime).UTC().Add(time.Hour - time.Millisecond).UnixMilli(),
		Open: open, High: open * 1.02, Low: open * 0.99, Close: open * 1.01,
		Volume: 100, QuoteVolume: 10000, Source: historicalmarket.SourceBinanceREST,
	}
}

func TestInferBTCETHMarketCondition(t *testing.T) {
	cases := []struct {
		name                  string
		btc, eth, hourlyRange float64
		want                  int
	}{
		{"strong bull", 8, 7, 1, markettypes.MarketConditionStrongBull},
		{"strong bear", -8, -7, 1, markettypes.MarketConditionStrongBear},
		{"broad rise", 1, 1, 1, markettypes.MarketConditionBroadRise},
		{"broad decline", -1, -1, 1, markettypes.MarketConditionBroadDecline},
		{"bull divergence", 2, -1, 1, markettypes.MarketConditionBullishDivergence},
		{"bear divergence", -2, 1, 1, markettypes.MarketConditionBearishDivergence},
		{"high volatility", 0.1, -0.1, 2.5, markettypes.MarketConditionHighVolatility},
		{"low volatility", 0.1, 0.1, 0.2, markettypes.MarketConditionLowVolatility},
		{"sideways", 0.3, 0.2, 1, markettypes.MarketConditionSideways},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := inferBTCETHMarketCondition(tc.btc, tc.eth, tc.hourlyRange); got != tc.want {
				t.Fatalf("got %d want %d", got, tc.want)
			}
		})
	}
}

func TestMarketConditionBackfillStoresKlinesAndSkipsExistingHour(t *testing.T) {
	setupBacktestStoreTest(t)
	latestClosedOpen := time.Now().UTC().Truncate(time.Hour).Add(-time.Hour)
	source := marketConditionHistoryFixtureSource{start: latestClosedOpen.Add(-2 * time.Hour).UnixMilli()}
	manager := NewManager(DatasetBuilder{Repository: historicalmarket.NewRepository(source)})
	manager.marketConditionEarliest = source

	config, err := loadBacktestSystemConfig()
	if err != nil {
		t.Fatal(err)
	}
	occupiedClose := latestClosedOpen.Add(-time.Hour).Add(time.Hour - time.Millisecond).UnixMilli()
	existing := models.MarketConditionHistory{ConfigID: config.ID, MarketCondition: markettypes.MarketConditionStrongBear, CreatedAt: occupiedClose}
	if _, err := orm.NewOrm().Insert(&existing); err != nil {
		t.Fatal(err)
	}

	job, err := manager.StartMarketConditionBackfill()
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		job, err = manager.GetMarketConditionBackfill(job.JobID)
		if err != nil {
			t.Fatal(err)
		}
		if job.Status == "succeeded" || job.Status == "failed" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if job.Status != "succeeded" {
		t.Fatalf("backfill did not succeed: %+v", job)
	}
	if job.BTCRows == 0 || job.ETHRows == 0 || job.InferredRows == 0 || job.SkippedRows < 1 {
		t.Fatalf("unexpected backfill summary: %+v", job)
	}
	for _, symbol := range []string{"BTCUSDT", "ETHUSDT"} {
		var count int64
		count, err = orm.NewOrm().QueryTable(new(models.MarketKline1h)).Filter("symbol", symbol).Count()
		if err != nil || count == 0 {
			t.Fatalf("%s 1h rows count=%d err=%v", symbol, count, err)
		}
	}
	var preserved models.MarketConditionHistory
	if err := orm.NewOrm().QueryTable(new(models.MarketConditionHistory)).Filter("id", existing.ID).One(&preserved); err != nil {
		t.Fatal(err)
	}
	if preserved.MarketCondition != markettypes.MarketConditionStrongBear {
		t.Fatalf("existing MarketCondition was overwritten: %+v", preserved)
	}
}

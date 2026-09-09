package historicalmarket

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
)

var repositoryTestOnce sync.Once
var repositoryTestErr error

func setupRepositoryTest(t *testing.T) {
	t.Helper()
	repositoryTestOnce.Do(func() {
		repositoryTestErr = orm.RegisterDriver("sqlite3", orm.DRSqlite)
		if repositoryTestErr != nil {
			return
		}
		orm.RegisterModel(new(models.MarketKline1m), new(models.MarketKline5m), new(models.MarketFundingRate), new(models.MarketDataImportBatch))
		dir, err := os.MkdirTemp("", "historicalmarket-test-*")
		if err != nil {
			repositoryTestErr = err
			return
		}
		repositoryTestErr = orm.RegisterDataBase("default", "sqlite3", filepath.Join(dir, "history.db"))
		if repositoryTestErr != nil {
			return
		}
		repositoryTestErr = orm.RunSyncdb("default", true, false)
	})
	if repositoryTestErr != nil {
		t.Fatal(repositoryTestErr)
	}
	for _, table := range []string{"market_data_import_batches", "market_funding_rates", "market_klines_1m", "market_klines_5m"} {
		if _, err := orm.NewOrm().Raw("DELETE FROM " + table).Exec(); err != nil {
			t.Fatal(err)
		}
	}
}

type countingSource struct {
	calls [][2]int64
	rows  map[int64]Kline
}

func (source *countingSource) Klines(_ context.Context, market, symbol, interval string, start, end int64) ([]Kline, error) {
	source.calls = append(source.calls, [2]int64{start, end})
	out := []Kline{}
	for open, row := range source.rows {
		if open >= start && row.CloseTime <= end {
			row.Market = market
			row.Symbol = symbol
			row.Interval = interval
			row.Source = SourceBinanceREST
			out = append(out, row)
		}
	}
	SortKlines(out)
	return out, nil
}
func (*countingSource) Funding(context.Context, string, string, int64, int64) ([]FundingRate, error) {
	return nil, nil
}

func minuteBar(at time.Time, close float64) Kline {
	return Kline{Market: MarketFuturesUSDT, Symbol: "BTCUSDT", Interval: "1m", OpenTime: at.UnixMilli(), CloseTime: at.Add(time.Minute - time.Millisecond).UnixMilli(), Open: close, High: close + 1, Low: close - 1, Close: close, Volume: 10, QuoteVolume: 1000, Source: "fixture"}
}

func TestRepositoryLastWriteWins(t *testing.T) {
	setupRepositoryTest(t)
	repo := NewRepository(nil)
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	first := minuteBar(at, 100)
	if _, err := repo.Import(context.Background(), ImportRequest{Source: "csv_a", Klines: []Kline{first}}); err != nil {
		t.Fatal(err)
	}
	second := minuteBar(at, 105)
	if _, err := repo.Import(context.Background(), ImportRequest{Source: "csv_b", Klines: []Kline{second}}); err != nil {
		t.Fatal(err)
	}
	rows, err := repo.LoadKlines(context.Background(), MarketFuturesUSDT, "BTCUSDT", "1m", at.UnixMilli(), second.CloseTime)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Close != 105 || rows[0].Source != "csv_b" {
		t.Fatalf("latest import did not overwrite canonical row: %+v", rows)
	}
}

func TestRepositoryReusesCompleteLocalRange(t *testing.T) {
	setupRepositoryTest(t)
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	source := &countingSource{rows: map[int64]Kline{}}
	repo := NewRepository(source)
	bars := []Kline{minuteBar(start, 100), minuteBar(start.Add(time.Minute), 101), minuteBar(start.Add(2*time.Minute), 102)}
	if _, err := repo.Import(context.Background(), ImportRequest{Source: "external_csv", Klines: bars}); err != nil {
		t.Fatal(err)
	}
	rows, err := repo.LoadKlines(context.Background(), MarketFuturesUSDT, "BTCUSDT", "1m", start.UnixMilli(), bars[2].CloseTime)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 || len(source.calls) != 0 {
		t.Fatalf("complete local cache should avoid source calls, rows=%d calls=%v", len(rows), source.calls)
	}
}

func TestRepositoryFetchesOnlyInternalGap(t *testing.T) {
	setupRepositoryTest(t)
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	b0 := minuteBar(start, 100)
	b1 := minuteBar(start.Add(time.Minute), 101)
	b2 := minuteBar(start.Add(2*time.Minute), 102)
	source := &countingSource{rows: map[int64]Kline{b1.OpenTime: b1}}
	repo := NewRepository(source)
	if _, err := repo.Import(context.Background(), ImportRequest{Source: "external_csv", Klines: []Kline{b0, b2}}); err != nil {
		t.Fatal(err)
	}
	rows, err := repo.LoadKlines(context.Background(), MarketFuturesUSDT, "BTCUSDT", "1m", b0.OpenTime, b2.CloseTime)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 || len(source.calls) != 1 || source.calls[0][0] != b1.OpenTime || source.calls[0][1] != b1.CloseTime {
		t.Fatalf("gap fill mismatch rows=%d calls=%v", len(rows), source.calls)
	}
}

func TestKlineTableWhitelist(t *testing.T) {
	if table, err := KlineTable("1M"); err != nil || table != "market_klines_1mo" {
		t.Fatalf("1M route=%q err=%v", table, err)
	}
	if _, err := KlineTable("1m;drop table x"); err == nil {
		t.Fatal("dynamic table input must be rejected")
	}
}

package historicalmarket

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
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
		orm.RegisterModel(new(models.MarketKline1s), new(models.MarketKline1m), new(models.MarketKline5m), new(models.MarketTrade), new(models.MarketFundingRate), new(models.MarketDataImportBatch))
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
	for _, table := range []string{"market_data_import_batches", "market_funding_rates", "market_trades", "market_klines_1s", "market_klines_1m", "market_klines_5m"} {
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

func TestSparseSecondRepositoryStoresOnlyRequestedSlice(t *testing.T) {
	setupRepositoryTest(t)
	repo := NewRepository(nil)
	start := time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC)
	rows := []Kline{
		{Market: MarketFuturesUSDT, Symbol: "BTCUSDT", Interval: SparseSecondInterval, OpenTime: start.UnixMilli(), CloseTime: start.Add(time.Second - time.Millisecond).UnixMilli(), Open: 100, High: 101, Low: 99, Close: 100.5, Volume: 2, QuoteVolume: 200, Source: SourceBinancePublicData, SourceRef: "archive-a#sha256=one"},
		{Market: MarketFuturesUSDT, Symbol: "BTCUSDT", Interval: SparseSecondInterval, OpenTime: start.Add(time.Second).UnixMilli(), CloseTime: start.Add(2*time.Second - time.Millisecond).UnixMilli(), Open: 100.5, High: 102, Low: 100, Close: 101, Volume: 3, QuoteVolume: 303, Source: SourceBinancePublicData, SourceRef: "archive-a#sha256=one"},
	}
	if written, err := repo.StoreSparseSecondBars(context.Background(), rows); err != nil || written != 2 {
		t.Fatalf("store sparse seconds written=%d err=%v", written, err)
	}
	got, err := repo.LoadSparseSecondBars(context.Background(), MarketFuturesUSDT, "BTCUSDT", start.Add(time.Second).UnixMilli(), start.Add(2*time.Second-time.Millisecond).UnixMilli())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].OpenTime != rows[1].OpenTime || got[0].Close != 101 || got[0].Interval != SparseSecondInterval {
		t.Fatalf("unexpected sparse second rows: %+v", got)
	}
}

func TestSparseTradeRepositoryOrdersAndUpserts(t *testing.T) {
	setupRepositoryTest(t)
	repo := NewRepository(nil)
	at := time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC).UnixMilli()
	hash := strings.Repeat("a", 64)
	rows := []PublicDataTrade{
		{Market: MarketFuturesUSDT, Symbol: "BTCUSDT", TradeID: 12, TradeTime: at + 100, Price: 101, Quantity: 2, QuoteQuantity: 202, IsBuyerMaker: true, Source: SourceBinancePublicData, SourceRef: "archive", ArchiveSHA256: hash},
		{Market: MarketFuturesUSDT, Symbol: "BTCUSDT", TradeID: 10, TradeTime: at + 100, Price: 100, Quantity: 1, QuoteQuantity: 100, Source: SourceBinancePublicData, SourceRef: "archive", ArchiveSHA256: hash},
		{Market: MarketFuturesUSDT, Symbol: "BTCUSDT", TradeID: 13, TradeTime: at + 200, Price: 102, Quantity: 3, QuoteQuantity: 306, Source: SourceBinancePublicData, SourceRef: "archive", ArchiveSHA256: hash},
	}
	if written, err := repo.StoreSparseTrades(context.Background(), rows); err != nil || written != 3 {
		t.Fatalf("store sparse trades written=%d err=%v", written, err)
	}
	rows[0].Price = 105
	if _, err := repo.StoreSparseTrades(context.Background(), rows[:1]); err != nil {
		t.Fatal(err)
	}
	got, err := repo.LoadSparseTrades(context.Background(), MarketFuturesUSDT, "BTCUSDT", at, at+999)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0].TradeID != 10 || got[1].TradeID != 12 || got[2].TradeID != 13 {
		t.Fatalf("unexpected sparse trade ordering: %+v", got)
	}
	if got[1].Price != 105 || got[1].ArchiveSHA256 != hash {
		t.Fatalf("sparse trade upsert/evidence lost: %+v", got[1])
	}
}

func TestSparseTradeRejectsNonHexArchiveSHA256(t *testing.T) {
	setupRepositoryTest(t)
	repo := NewRepository(nil)
	trade := PublicDataTrade{
		Market: MarketFuturesUSDT, Symbol: "BTCUSDT", TradeID: 1, TradeTime: time.Now().UnixMilli(),
		Price: 100, Quantity: 1, QuoteQuantity: 100, Source: SourceBinancePublicData,
		SourceRef: "fixture", ArchiveSHA256: strings.Repeat("z", 64),
	}
	if _, err := repo.StoreSparseTrades(context.Background(), []PublicDataTrade{trade}); err == nil {
		t.Fatal("non-hex archive SHA256 must be rejected")
	}
}

func TestStoreSparseSecondBarsRollsBackOnCancelBetweenChunks(t *testing.T) {
	setupRepositoryTest(t)
	ctx, cancel := context.WithCancel(context.Background())
	repo := NewRepository(nil)
	calls := 0
	repo.Now = func() time.Time {
		calls++
		if calls == 1 {
			cancel()
		}
		return time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC)
	}
	start := time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC).UnixMilli()
	rows := make([]Kline, 60)
	for i := range rows {
		open := start + int64(i)*1000
		rows[i] = Kline{Market: MarketFuturesUSDT, Symbol: "BTCUSDT", Interval: SparseSecondInterval, OpenTime: open, CloseTime: open + 999, Open: 100, High: 101, Low: 99, Close: 100, Volume: 1, QuoteVolume: 100}
	}
	written, err := repo.StoreSparseSecondBars(ctx, rows)
	if !errors.Is(err, context.Canceled) || written != 0 {
		t.Fatalf("expected atomic cancellation, written=%d err=%v", written, err)
	}
	var count int
	if err := orm.NewOrm().Raw("SELECT COUNT(*) FROM market_klines_1s").QueryRow(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("cancelled sparse second write left %d canonical rows", count)
	}
}

func TestStoreSparseTradesRollsBackOnCancelBetweenChunks(t *testing.T) {
	setupRepositoryTest(t)
	ctx, cancel := context.WithCancel(context.Background())
	repo := NewRepository(nil)
	calls := 0
	repo.Now = func() time.Time {
		calls++
		if calls == 1 {
			cancel()
		}
		return time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC)
	}
	start := time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC).UnixMilli()
	rows := make([]PublicDataTrade, 80)
	for i := range rows {
		rows[i] = PublicDataTrade{Market: MarketFuturesUSDT, Symbol: "BTCUSDT", Source: "fixture", TradeID: int64(i + 1), TradeTime: start + int64(i), Price: 100, Quantity: 1, QuoteQuantity: 100}
	}
	written, err := repo.StoreSparseTrades(ctx, rows)
	if !errors.Is(err, context.Canceled) || written != 0 {
		t.Fatalf("expected atomic cancellation, written=%d err=%v", written, err)
	}
	var count int
	if err := orm.NewOrm().Raw("SELECT COUNT(*) FROM market_trades").QueryRow(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("cancelled sparse trade write left %d canonical rows", count)
	}
}

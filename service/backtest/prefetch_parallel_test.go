package backtest

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"go_binance_futures/service/historicalmarket"
)

// Exercises the real HTTP ZIP downloader, SHA256 validation, the three-slot
// download window, sequential SQLite importer and the already-cached skip path.
func TestPrefetchMonthlyDownloadsBoundedAndSkipsCachedMonths(t *testing.T) {
	setupBacktestStoreTest(t)
	repo := historicalmarket.NewRepository(nil)
	const symbol = "BTCUSDT"
	var months []prefetchMonthRange
	zips := map[string][]byte{}
	sums := map[string]string{}
	for i := 0; i < 6; i++ {
		month := time.Date(2025, time.Month(i+1), 1, 0, 0, 0, 0, time.UTC)
		var csv strings.Builder
		for minute := 0; minute < 2; minute++ {
			open := month.Add(time.Duration(minute) * time.Minute)
			fmt.Fprintf(&csv, "%d,100,101,99,100.5,2,%d,201,3,1,100.5,0\n",
				open.UnixMilli(), open.Add(time.Minute-time.Millisecond).UnixMilli())
		}
		filename := fmt.Sprintf("%s-1m-%s.zip", symbol, month.Format("2006-01"))
		zipped := testPublicDataZIP(t, strings.TrimSuffix(filename, ".zip")+".csv", csv.String())
		hash := sha256.Sum256(zipped)
		zips[filename] = zipped
		sums[filename] = fmt.Sprintf("%x  %s\n", hash, filename)
		months = append(months, prefetchMonthRange{
			Date: month, Start: month.UnixMilli(),
			End: month.Add(2*time.Minute - time.Millisecond).UnixMilli(),
		})
	}

	// One month already saved in MySQL's compressed 1m chunk table.
	alreadyCached := make([]historicalmarket.Kline, 2)
	for minute := 0; minute < 2; minute++ {
		start := months[0].Date.Add(time.Duration(minute) * time.Minute)
		alreadyCached[minute] = historicalmarket.Kline{
			Market: historicalmarket.MarketFuturesUSDT,
			Symbol: symbol, Interval: "1m",
			OpenTime:  start.UnixMilli(),
			CloseTime: start.Add(time.Minute - time.Millisecond).UnixMilli(),
			Open:      100, High: 101, Low: 99, Close: 100.5,
			Volume: 2, QuoteVolume: 201, TradeCount: 3,
		}
	}
	if _, err := repo.Import(context.Background(), historicalmarket.ImportRequest{
		Source: historicalmarket.SourceBinancePublicData, Klines: alreadyCached,
	}); err != nil {
		t.Fatal(err)
	}

	var active, peak, downloads atomic.Int64
	firstMonthRequested := atomic.Bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		filename := path.Base(r.URL.Path)
		if strings.HasSuffix(filename, ".CHECKSUM") {
			filename = strings.TrimSuffix(filename, ".CHECKSUM")
			if checksum, ok := sums[filename]; ok {
				_, _ = fmt.Fprint(w, checksum)
				return
			}
			http.NotFound(w, r)
			return
		}
		zipped, ok := zips[filename]
		if !ok {
			http.NotFound(w, r)
			return
		}
		if filename == fmt.Sprintf("%s-1m-2025-01.zip", symbol) {
			firstMonthRequested.Store(true)
		}
		downloads.Add(1)
		current := active.Add(1)
		for {
			old := peak.Load()
			if old >= current || peak.CompareAndSwap(old, current) {
				break
			}
		}
		time.Sleep(60 * time.Millisecond)
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write(zipped)
		active.Add(-1)
	}))
	defer server.Close()
	client, err := historicalmarket.NewPublicDataClient(historicalmarket.PublicDataClientConfig{
		BaseURL: server.URL, CacheDir: t.TempDir(), HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	stats, err := prefetchPublicMonthly1m(context.Background(), repo, client, symbol, months, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := peak.Load(); got < 2 || got > 3 {
		t.Fatalf("monthly downloads peak concurrency=%d, want [2,3]", got)
	}
	if got := downloads.Load(); got != 5 || firstMonthRequested.Load() {
		t.Fatalf("unexpected ZIP downloads count=%d cachedRequested=%v", got, firstMonthRequested.Load())
	}
	if stats.Calls != 5 || stats.Rows != 10 || len(stats.CompleteMonths) != 6 {
		t.Fatalf("unexpected prefetch result: %+v", stats)
	}
	for _, month := range months {
		complete, count, err := repo.KlineRangeComplete(context.Background(),
			historicalmarket.MarketFuturesUSDT, symbol, ReplayInterval, month.Start, month.End)
		if err != nil || !complete || count != 2 {
			t.Fatalf("month %s incomplete count=%d err=%v", month.Date.Format("2006-01"), count, err)
		}
	}
	before := downloads.Load()
	stats, err = prefetchPublicMonthly1m(context.Background(), repo, client, symbol, months, nil)
	if err != nil || stats.Calls != 0 || stats.Rows != 0 || len(stats.CompleteMonths) != 6 || downloads.Load() != before {
		t.Fatalf("second prefetch refetched complete months: stats=%+v before=%d after=%d err=%v",
			stats, before, downloads.Load(), err)
	}
}

func TestPrefetchMonthlyChecksumErrorStopsImport(t *testing.T) {
	setupBacktestStoreTest(t)
	month := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)
	filename := "BTCUSDT-1m-2025-07.zip"
	body := testPublicDataZIP(t, strings.TrimSuffix(filename, ".zip")+".csv",
		fmt.Sprintf("%d,100,101,99,100.5,2,%d,201,3,1,100.5,0\n",
			month.UnixMilli(), month.Add(time.Minute-time.Millisecond).UnixMilli()))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".CHECKSUM") {
			fmt.Fprintf(w, "%064x  %s\n", 1, filename)
		} else {
			_, _ = w.Write(body)
		}
	}))
	defer server.Close()
	client, err := historicalmarket.NewPublicDataClient(historicalmarket.PublicDataClientConfig{
		BaseURL: server.URL, CacheDir: t.TempDir(), HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	_, err = prefetchPublicMonthly1m(context.Background(), historicalmarket.NewRepository(nil),
		client, "BTCUSDT", []prefetchMonthRange{{
			Date: month, Start: month.UnixMilli(),
			End: month.Add(time.Minute - time.Millisecond).UnixMilli(),
		}}, nil)
	if err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("corrupt checksum must be fatal, got %v", err)
	}
}

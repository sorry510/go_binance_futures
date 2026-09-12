package historicalmarket

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func publicDataTestZIP(t *testing.T, name, content string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	file, err := writer.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func publicDataChecksum(body []byte, filename string) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]) + "  " + filename + "\n"
}

func TestPublicDataArchiveURL(t *testing.T) {
	client, err := NewPublicDataClient(PublicDataClientConfig{BaseURL: "https://example.test", CacheDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	date := time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		spec PublicDataArchiveSpec
		want string
	}{
		{PublicDataArchiveSpec{Kind: ArchiveKindKlines, Period: ArchivePeriodDaily, Symbol: "btcusdt", Interval: "1s", Date: date}, "https://example.test/data/futures/um/daily/klines/BTCUSDT/1s/BTCUSDT-1s-2026-09-11.zip"},
		{PublicDataArchiveSpec{Kind: ArchiveKindTrades, Period: ArchivePeriodDaily, Symbol: "BTCUSDT", Date: date}, "https://example.test/data/futures/um/daily/trades/BTCUSDT/BTCUSDT-trades-2026-09-11.zip"},
		{PublicDataArchiveSpec{Kind: ArchiveKindMarkPriceKlines, Period: ArchivePeriodMonthly, Symbol: "BTCUSDT", Interval: "1s", Date: date}, "https://example.test/data/futures/um/monthly/markPriceKlines/BTCUSDT/1s/BTCUSDT-1s-2026-09.zip"},
	}
	for _, test := range tests {
		got, err := client.ArchiveURL(test.spec)
		if err != nil {
			t.Fatal(err)
		}
		if got != test.want {
			t.Fatalf("archive URL=%q want=%q", got, test.want)
		}
	}
}

func TestPublicDataFetchVerifyAndParseKlines(t *testing.T) {
	date := time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC)
	filename := "BTCUSDT-1s-2026-09-11.zip"
	csv := "open_time,open,high,low,close,volume,close_time,quote_volume,count,taker_buy_base,taker_buy_quote,ignore\n" +
		"1789084800000,100,101,99,100.5,2,1789084800999,200.5,3,1,100.5,0\n" +
		"1789084801000,100.5,102,100,101,4,1789084801999,404,5,2,202,0\n"
	zipped := publicDataTestZIP(t, strings.TrimSuffix(filename, ".zip")+".csv", csv)
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".CHECKSUM") {
			fmt.Fprint(w, publicDataChecksum(zipped, filename))
			return
		}
		w.Write(zipped)
	}))
	defer server.Close()

	client, err := NewPublicDataClient(PublicDataClientConfig{BaseURL: server.URL, CacheDir: t.TempDir(), HTTPClient: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	spec := PublicDataArchiveSpec{Kind: ArchiveKindKlines, Period: ArchivePeriodDaily, Symbol: "BTCUSDT", Interval: "1s", Date: date}
	archive, err := client.FetchArchive(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	if archive.SHA256 == "" || archive.Size != int64(len(zipped)) || filepath.Ext(archive.Path) != ".zip" {
		t.Fatalf("unexpected archive metadata: %+v", archive)
	}
	rows, err := client.ParseKlines(context.Background(), archive, 1789084801000, 1789084801999)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].OpenTime != 1789084801000 || rows[0].Close != 101 || rows[0].TradeCount != 5 {
		t.Fatalf("unexpected parsed klines: %+v", rows)
	}
	if rows[0].Source != SourceBinancePublicData || !strings.Contains(rows[0].SourceRef, archive.SHA256) {
		t.Fatalf("missing archive evidence: %+v", rows[0])
	}
}

func TestPublicDataParseTradesOrdersByTimeThenID(t *testing.T) {
	date := time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC)
	filename := "BTCUSDT-trades-2026-09-11.zip"
	csv := "id,price,qty,quoteQty,time,isBuyerMaker\n" +
		"12,101,2,202,1789084800123,true\n" +
		"10,100,1,100,1789084800123,false\n" +
		"13,102,3,306,1789084800456,false\n"
	zipped := publicDataTestZIP(t, strings.TrimSuffix(filename, ".zip")+".csv", csv)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".CHECKSUM") {
			fmt.Fprint(w, publicDataChecksum(zipped, filename))
			return
		}
		w.Write(zipped)
	}))
	defer server.Close()
	client, err := NewPublicDataClient(PublicDataClientConfig{BaseURL: server.URL, CacheDir: t.TempDir(), HTTPClient: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	archive, err := client.FetchArchive(context.Background(), PublicDataArchiveSpec{Kind: ArchiveKindTrades, Period: ArchivePeriodDaily, Symbol: "BTCUSDT", Date: date})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := client.ParseTrades(context.Background(), archive, 1789084800000, 1789084800999)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 || rows[0].TradeID != 10 || rows[1].TradeID != 12 || rows[2].TradeID != 13 {
		t.Fatalf("trades are not deterministically ordered: %+v", rows)
	}
	if rows[1].QuoteQuantity != 202 || !rows[1].IsBuyerMaker {
		t.Fatalf("trade fields lost: %+v", rows[1])
	}
}

func TestPublicDataTradeScannerStreamsForwardAcrossRanges(t *testing.T) {
	date := time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC)
	filename := "BTCUSDT-trades-2026-09-11.zip"
	csv := "id,price,qty,quoteQty,time,isBuyerMaker\n" +
		"1,100,1,100,1789084800100,false\n" +
		"2,101,1,101,1789084800200,false\n" +
		"3,102,1,102,1789084860100,false\n" +
		"4,103,1,103,1789084920100,false\n"
	zipped := publicDataTestZIP(t, strings.TrimSuffix(filename, ".zip")+".csv", csv)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".CHECKSUM") {
			fmt.Fprint(w, publicDataChecksum(zipped, filename))
			return
		}
		w.Write(zipped)
	}))
	defer server.Close()
	client, err := NewPublicDataClient(PublicDataClientConfig{BaseURL: server.URL, CacheDir: t.TempDir(), HTTPClient: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	archive, err := client.FetchArchive(context.Background(), PublicDataArchiveSpec{Kind: ArchiveKindTrades, Period: ArchivePeriodDaily, Symbol: "BTCUSDT", Date: date})
	if err != nil {
		t.Fatal(err)
	}
	first, err := client.ParseTrades(context.Background(), archive, 1789084800000, 1789084859999)
	if err != nil || len(first) != 2 {
		t.Fatalf("first range rows=%d err=%v", len(first), err)
	}
	scanner := client.tradeScanner
	if scanner == nil {
		t.Fatal("trade scanner was not created")
	}
	rowAfterFirst := scanner.rowNumber
	second, err := client.ParseTrades(context.Background(), archive, 1789084860000, 1789084919999)
	if err != nil || len(second) != 1 || second[0].TradeID != 3 {
		t.Fatalf("second range rows=%+v err=%v", second, err)
	}
	if client.tradeScanner != scanner || client.tradeScanner.rowNumber < rowAfterFirst {
		t.Fatal("later trade range reopened the same daily ZIP instead of streaming forward")
	}
}

func TestPublicDataKlineScannerStreamsForwardAcrossRanges(t *testing.T) {
	date := time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC)
	filename := "BTCUSDT-1s-2026-09-11.zip"
	csv := "open_time,open,high,low,close,volume,close_time,quote_volume,count,taker_buy_base,taker_buy_quote,ignore\n" +
		"1789084800000,100,101,99,100,1,1789084800999,100,1,1,100,0\n" +
		"1789084801000,100,101,99,100,1,1789084801999,100,1,1,100,0\n" +
		"1789084860000,101,102,100,101,1,1789084860999,101,1,1,101,0\n"
	zipped := publicDataTestZIP(t, strings.TrimSuffix(filename, ".zip")+".csv", csv)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".CHECKSUM") {
			fmt.Fprint(w, publicDataChecksum(zipped, filename))
			return
		}
		w.Write(zipped)
	}))
	defer server.Close()
	client, err := NewPublicDataClient(PublicDataClientConfig{BaseURL: server.URL, CacheDir: t.TempDir(), HTTPClient: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	archive, err := client.FetchArchive(context.Background(), PublicDataArchiveSpec{Kind: ArchiveKindKlines, Period: ArchivePeriodDaily, Symbol: "BTCUSDT", Interval: "1s", Date: date})
	if err != nil {
		t.Fatal(err)
	}
	first, err := client.ParseKlines(context.Background(), archive, 1789084800000, 1789084859999)
	if err != nil || len(first) != 2 {
		t.Fatalf("first range rows=%d err=%v", len(first), err)
	}
	scanner := client.klineScanner
	if scanner == nil {
		t.Fatal("kline scanner was not created")
	}
	rowAfterFirst := scanner.rowNumber
	second, err := client.ParseKlines(context.Background(), archive, 1789084860000, 1789084919999)
	if err != nil || len(second) != 1 || second[0].OpenTime != 1789084860000 {
		t.Fatalf("second range rows=%+v err=%v", second, err)
	}
	if client.klineScanner != scanner || client.klineScanner.rowNumber < rowAfterFirst {
		t.Fatal("later kline range reopened the same daily ZIP instead of streaming forward")
	}
}

func TestPublicDataTradeScannerReopensOnBackwardRange(t *testing.T) {
	date := time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC)
	filename := "BTCUSDT-trades-2026-09-11.zip"
	csv := "id,price,qty,quoteQty,time,isBuyerMaker\n" +
		"1,100,1,100,1789084800100,false\n" +
		"2,101,1,101,1789084860100,false\n"
	zipped := publicDataTestZIP(t, strings.TrimSuffix(filename, ".zip")+".csv", csv)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".CHECKSUM") {
			fmt.Fprint(w, publicDataChecksum(zipped, filename))
			return
		}
		w.Write(zipped)
	}))
	defer server.Close()
	client, err := NewPublicDataClient(PublicDataClientConfig{BaseURL: server.URL, CacheDir: t.TempDir(), HTTPClient: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	archive, err := client.FetchArchive(context.Background(), PublicDataArchiveSpec{Kind: ArchiveKindTrades, Period: ArchivePeriodDaily, Symbol: "BTCUSDT", Date: date})
	if err != nil {
		t.Fatal(err)
	}
	later, err := client.ParseTrades(context.Background(), archive, 1789084860000, 1789084919999)
	if err != nil || len(later) != 1 || later[0].TradeID != 2 {
		t.Fatalf("later=%+v err=%v", later, err)
	}
	oldScanner := client.tradeScanner
	earlier, err := client.ParseTrades(context.Background(), archive, 1789084800000, 1789084859999)
	if err != nil || len(earlier) != 1 || earlier[0].TradeID != 1 {
		t.Fatalf("earlier=%+v err=%v", earlier, err)
	}
	if client.tradeScanner == nil || client.tradeScanner == oldScanner {
		t.Fatal("backward range must reopen the archive scanner")
	}
}

func TestPublicDataTradeScannerReopensWhenStartEqualsLastConsumedTime(t *testing.T) {
	date := time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC)
	filename := "BTCUSDT-trades-2026-09-11.zip"
	csv := "id,price,qty,quoteQty,time,isBuyerMaker\n" +
		"1,100,1,100,1789084800100,false\n" +
		"2,101,1,101,1789084800200,false\n"
	zipped := publicDataTestZIP(t, strings.TrimSuffix(filename, ".zip")+".csv", csv)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".CHECKSUM") {
			fmt.Fprint(w, publicDataChecksum(zipped, filename))
			return
		}
		w.Write(zipped)
	}))
	defer server.Close()
	client, err := NewPublicDataClient(PublicDataClientConfig{BaseURL: server.URL, CacheDir: t.TempDir(), HTTPClient: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	archive, err := client.FetchArchive(context.Background(), PublicDataArchiveSpec{Kind: ArchiveKindTrades, Period: ArchivePeriodDaily, Symbol: "BTCUSDT", Date: date})
	if err != nil {
		t.Fatal(err)
	}
	first, err := client.ParseTrades(context.Background(), archive, 1789084800000, 1789084800100)
	if err != nil || len(first) != 1 || first[0].TradeID != 1 {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	oldScanner := client.tradeScanner
	again, err := client.ParseTrades(context.Background(), archive, 1789084800100, 1789084800100)
	if err != nil || len(again) != 1 || again[0].TradeID != 1 {
		t.Fatalf("equal-start range lost the consumed boundary trade: rows=%+v err=%v", again, err)
	}
	if client.tradeScanner == nil || client.tradeScanner == oldScanner {
		t.Fatal("start equal to last consumed time must reopen scanner")
	}
}

func TestPublicDataScannerOpenFailureClearsClosedSlot(t *testing.T) {
	date := time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC)
	filename := "BTCUSDT-trades-2026-09-11.zip"
	zipped := publicDataTestZIP(t, strings.TrimSuffix(filename, ".zip")+".csv", "1,100,1,100,1789084860100,false\n")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".CHECKSUM") {
			fmt.Fprint(w, publicDataChecksum(zipped, filename))
			return
		}
		w.Write(zipped)
	}))
	defer server.Close()
	client, err := NewPublicDataClient(PublicDataClientConfig{BaseURL: server.URL, CacheDir: t.TempDir(), HTTPClient: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	archive, err := client.FetchArchive(context.Background(), PublicDataArchiveSpec{Kind: ArchiveKindTrades, Period: ArchivePeriodDaily, Symbol: "BTCUSDT", Date: date})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = client.ParseTrades(context.Background(), archive, 1789084860000, 1789084919999); err != nil {
		t.Fatal(err)
	}
	if client.tradeScanner == nil {
		t.Fatal("scanner was not created")
	}
	if err := os.Remove(archive.Path); err != nil {
		t.Fatal(err)
	}
	_, err = client.ParseTrades(context.Background(), archive, 1789084800000, 1789084859999)
	if err == nil {
		t.Fatal("expected scanner reopen failure after archive removal")
	}
	if client.tradeScanner != nil {
		t.Fatal("failed reopen must clear the closed scanner slot")
	}
}

func TestPublicDataChecksumMismatchFailsClosed(t *testing.T) {
	date := time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC)
	zipped := publicDataTestZIP(t, "BTCUSDT-1s-2026-09-11.csv", "1789084800000,1,1,1,1,1,1789084800999,1,1,1,1,0\n")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".CHECKSUM") {
			fmt.Fprint(w, strings.Repeat("0", 64)+"  BTCUSDT-1s-2026-09-11.zip\n")
			return
		}
		w.Write(zipped)
	}))
	defer server.Close()
	client, err := NewPublicDataClient(PublicDataClientConfig{BaseURL: server.URL, CacheDir: t.TempDir(), HTTPClient: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.FetchArchive(context.Background(), PublicDataArchiveSpec{Kind: ArchiveKindKlines, Period: ArchivePeriodDaily, Symbol: "BTCUSDT", Interval: "1s", Date: date})
	if !errors.Is(err, ErrPublicDataChecksumMismatch) {
		t.Fatalf("expected checksum mismatch, got %v", err)
	}
}

func TestPublicData404DoesNotRetry(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		http.NotFound(w, r)
	}))
	defer server.Close()
	client, err := NewPublicDataClient(PublicDataClientConfig{BaseURL: server.URL, CacheDir: t.TempDir(), HTTPClient: server.Client(), MaxRetries: 4, RetryBackoff: time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	spec := PublicDataArchiveSpec{Kind: ArchiveKindTrades, Period: ArchivePeriodDaily, Symbol: "BTCUSDT", Date: time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC)}
	_, err = client.FetchArchive(context.Background(), spec)
	if !errors.Is(err, ErrPublicDataArchiveNotFound) {
		t.Fatalf("expected archive not found, got %v", err)
	}
	_, err = client.FetchArchive(context.Background(), spec)
	if !errors.Is(err, ErrPublicDataArchiveNotFound) {
		t.Fatalf("expected cached archive not found, got %v", err)
	}
	if requests.Load() != 1 {
		t.Fatalf("404 must not retry or refetch within one client, requests=%d", requests.Load())
	}
}

func TestPublicDataRetries5xx(t *testing.T) {
	filename := "BTCUSDT-trades-2026-09-11.zip"
	zipped := publicDataTestZIP(t, strings.TrimSuffix(filename, ".zip")+".csv", "1,100,1,100,1789084800000,false\n")
	var zipRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".CHECKSUM") {
			fmt.Fprint(w, publicDataChecksum(zipped, filename))
			return
		}
		if zipRequests.Add(1) < 3 {
			http.Error(w, "retry", http.StatusServiceUnavailable)
			return
		}
		w.Write(zipped)
	}))
	defer server.Close()
	client, err := NewPublicDataClient(PublicDataClientConfig{BaseURL: server.URL, CacheDir: t.TempDir(), HTTPClient: server.Client(), MaxRetries: 2, RetryBackoff: time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.FetchArchive(context.Background(), PublicDataArchiveSpec{Kind: ArchiveKindTrades, Period: ArchivePeriodDaily, Symbol: "BTCUSDT", Date: time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	if zipRequests.Load() != 3 {
		t.Fatalf("expected 3 attempts, got %d", zipRequests.Load())
	}
}

func TestPublicDataSingleflightDeduplicatesArchiveDownload(t *testing.T) {
	filename := "BTCUSDT-trades-2026-09-11.zip"
	zipped := publicDataTestZIP(t, strings.TrimSuffix(filename, ".zip")+".csv", "1,100,1,100,1789084800000,false\n")
	var zipRequests atomic.Int32
	var checksumRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".CHECKSUM") {
			checksumRequests.Add(1)
			fmt.Fprint(w, publicDataChecksum(zipped, filename))
			return
		}
		zipRequests.Add(1)
		time.Sleep(20 * time.Millisecond)
		w.Write(zipped)
	}))
	defer server.Close()
	client, err := NewPublicDataClient(PublicDataClientConfig{BaseURL: server.URL, CacheDir: t.TempDir(), HTTPClient: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	spec := PublicDataArchiveSpec{Kind: ArchiveKindTrades, Period: ArchivePeriodDaily, Symbol: "BTCUSDT", Date: time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC)}
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := client.FetchArchive(context.Background(), spec)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if zipRequests.Load() != 1 || checksumRequests.Load() != 1 {
		t.Fatalf("singleflight failed: zip=%d checksum=%d", zipRequests.Load(), checksumRequests.Load())
	}
}

func TestPublicDataDownloadHonorsContextCancel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()
	client, err := NewPublicDataClient(PublicDataClientConfig{BaseURL: server.URL, CacheDir: t.TempDir(), HTTPClient: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err = client.FetchArchive(ctx, PublicDataArchiveSpec{Kind: ArchiveKindTrades, Period: ArchivePeriodDaily, Symbol: "BTCUSDT", Date: time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC)})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}

func TestPublicDataTempRootCreatesOwnedRunDirAndCloseRemovesOnlyRunDir(t *testing.T) {
	root := filepath.Join(t.TempDir(), "cache", "tmp")
	client, err := NewPublicDataClient(PublicDataClientConfig{
		BaseURL:     "https://example.test",
		TempRootDir: root,
	})
	if err != nil {
		t.Fatal(err)
	}
	runDir := client.cacheDir
	if filepath.Dir(runDir) != root {
		t.Fatalf("cache dir parent=%q want=%q", filepath.Dir(runDir), root)
	}
	if !strings.HasPrefix(filepath.Base(runDir), "binance-public-data-") {
		t.Fatalf("unexpected run cache dir %q", runDir)
	}
	if err := os.WriteFile(filepath.Join(runDir, "test.zip"), []byte("test"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(runDir); !os.IsNotExist(err) {
		t.Fatalf("run cache dir should be removed, stat err=%v", err)
	}
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		t.Fatalf("temp root should remain, info=%v err=%v", info, err)
	}
}

func TestPublicDataRejectsPersistentAndTempCacheDirsTogether(t *testing.T) {
	_, err := NewPublicDataClient(PublicDataClientConfig{
		CacheDir:    filepath.Join(t.TempDir(), "persistent"),
		TempRootDir: filepath.Join(t.TempDir(), "temp-root"),
	})
	if err == nil || !strings.Contains(err.Error(), "cannot both be configured") {
		t.Fatalf("expected mutually exclusive cache dir error, got %v", err)
	}
}

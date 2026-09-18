package backtest

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"go_binance_futures/service/historicalmarket"
)

func TestCompletedPrefetchMonthsExcludesCurrentMonth(t *testing.T) {
	start := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC)
	months := completedPrefetchMonths(start.UnixMilli(), end.UnixMilli(), now)
	if len(months) != 2 {
		t.Fatalf("months=%d want=2: %+v", len(months), months)
	}
	if got := months[0].Date.Format("2006-01"); got != "2026-01" {
		t.Fatalf("first month=%s", got)
	}
	if got := months[1].Date.Format("2006-01"); got != "2026-02" {
		t.Fatalf("second month=%s", got)
	}
}
func TestPrefetchPublicMonthly1mImportsAndThenSkipsCompleteRange(t *testing.T) {
	setupBacktestStoreTest(t)
	start := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	end := start.Add(3*time.Minute - time.Millisecond)
	filename := "BTCUSDT-1m-2026-03.zip"
	var csv strings.Builder
	for i := 0; i < 3; i++ {
		open := start.Add(time.Duration(i) * time.Minute)
		closeTime := open.Add(time.Minute - time.Millisecond)
		fmt.Fprintf(&csv, "%d,100,101,99,100.5,2,%d,201,3,1,100.5,0\n",
			open.UnixMilli(), closeTime.UnixMilli())
	}
	zipped := testPublicDataZIP(t, strings.TrimSuffix(filename, ".zip")+".csv", csv.String())
	sum := sha256.Sum256(zipped)
	var hits atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		switch {
		case strings.HasSuffix(r.URL.Path, ".CHECKSUM"):
			fmt.Fprintf(w, "%x  %s\n", sum, filename)
		case strings.HasSuffix(r.URL.Path, ".zip"):
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(zipped)
		default:
			http.NotFound(w, r)
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
	repo := historicalmarket.NewRepository(nil)
	months := []prefetchMonthRange{{
		Date:  time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		Start: start.UnixMilli(), End: end.UnixMilli(),
	}}
	stages := []string{}
	stats, err := prefetchPublicMonthly1m(
		context.Background(), repo, client, "BTCUSDT", months,
		func(_ int, stage, _ string) { stages = append(stages, stage) },
	)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Calls != 1 || stats.Rows != 3 {
		t.Fatalf("stats=%+v want calls=1 rows=3", stats)
	}
	complete, count, err := repo.KlineRangeComplete(
		context.Background(), historicalmarket.MarketFuturesUSDT, "BTCUSDT", ReplayInterval,
		start.UnixMilli(), end.UnixMilli(),
	)
	if err != nil || !complete || count != 3 {
		t.Fatalf("complete=%v count=%d err=%v", complete, count, err)
	}
	joined := strings.Join(stages, ",")
	if !strings.Contains(joined, "public_data_downloading") ||
		!strings.Contains(joined, "public_data_importing") {
		t.Fatalf("stages=%v", stages)
	}
	hitsBefore := hits.Load()
	stats, err = prefetchPublicMonthly1m(context.Background(), repo, client, "BTCUSDT", months, nil)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Calls != 0 || stats.Rows != 0 || hits.Load() != hitsBefore {
		t.Fatalf("complete range should skip archive: stats=%+v hits=%d->%d", stats, hitsBefore, hits.Load())
	}
}

func testPublicDataZIP(t *testing.T, csvName, body string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	file, err := writer.Create(csvName)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte(body)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

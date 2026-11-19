package historicalmarket

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func bt3Date() time.Time {
	return time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC)
}

func TestBT3VerifiedZIPPersistsAcrossClientsAndDetectsCorruption(t *testing.T) {
	filename := "BTCUSDT-1s-2026-09-11.zip"
	zipData := publicDataTestZIP(t, strings.TrimSuffix(filename, ".zip")+".csv", "1,100,100,100,100,1,999,100,1,1,100,0\n")
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if strings.HasSuffix(r.URL.Path, ".CHECKSUM") {
			_, _ = fmt.Fprint(w, publicDataChecksum(zipData, filename))
			return
		}
		_, _ = w.Write(zipData)
	}))
	defer server.Close()
	cacheDir := filepath.Join(t.TempDir(), "bt3")
	spec := PublicDataArchiveSpec{Kind: ArchiveKindKlines, Period: ArchivePeriodDaily, Symbol: "BTCUSDT", Interval: "1s", Date: bt3Date()}
	newClient := func() *PublicDataClient {
		c, err := NewPublicDataClient(PublicDataClientConfig{BaseURL: server.URL, CacheDir: cacheDir, HTTPClient: server.Client(), MaxCacheBytes: 1 << 20})
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	ctx := context.Background()
	first := newClient()
	a, err := first.FetchArchive(ctx, spec)
	if err != nil || a.FromCache {
		t.Fatalf("cold archive=%+v err=%v", a, err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	requestCount := requests.Load()
	if requestCount != 2 {
		t.Fatalf("cold requests=%d", requestCount)
	}
	second := newClient()
	b, err := second.FetchArchive(ctx, spec)
	if err != nil || !b.FromCache || b.SHA256 != a.SHA256 || requests.Load() != requestCount {
		t.Fatalf("disk hit=%+v err=%v requests=%d", b, err, requests.Load())
	}
	if err := second.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(a.Path, []byte("corrupt"), 0o600); err != nil {
		t.Fatal(err)
	}
	third := newClient()
	recovered, err := third.FetchArchive(ctx, spec)
	if err != nil || recovered.FromCache || recovered.SHA256 != a.SHA256 || requests.Load() != requestCount+2 {
		t.Fatalf("corrupt archive not reverified: archive=%+v err=%v requests=%d", recovered, err, requests.Load())
	}
	if err := third.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestBT3FileModeKeepsRowsAndEvidenceWithoutNewSQLInserts(t *testing.T) {
	setupRepositoryTest(t)
	repo := NewRepository(nil)
	start := bt3Date().UnixMilli()
	klineFile := "BTCUSDT-1s-2026-09-11.zip"
	tradeFile := "BTCUSDT-trades-2026-09-11.zip"
	klines := fmt.Sprintf("%d,100,101,99,100.5,2,%d,200,3,1,100,0\n%d,100.5,102,100,101,3,%d,303,4,2,202,0\n", start, start+999, start+1000, start+1999)
	trades := fmt.Sprintf("10,100,1,100,%d,false\n11,101,2,202,%d,true\n", start+100, start+500)
	klineZIP := publicDataTestZIP(t, strings.TrimSuffix(klineFile, ".zip")+".csv", klines)
	tradeZIP := publicDataTestZIP(t, strings.TrimSuffix(tradeFile, ".zip")+".csv", trades)
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		body, filename := klineZIP, klineFile
		if strings.Contains(r.URL.Path, "/trades/") {
			body, filename = tradeZIP, tradeFile
		}
		if strings.HasSuffix(r.URL.Path, ".CHECKSUM") {
			_, _ = fmt.Fprint(w, publicDataChecksum(body, filename))
			return
		}
		_, _ = w.Write(body)
	}))
	defer server.Close()
	cacheDir := filepath.Join(t.TempDir(), "verified-um")
	makeProvider := func() *AdaptiveResolutionProvider {
		client, err := NewPublicDataClient(PublicDataClientConfig{BaseURL: server.URL, CacheDir: cacheDir, HTTPClient: server.Client(), MaxCacheBytes: 1 << 20})
		if err != nil {
			t.Fatal(err)
		}
		provider, err := NewAdaptiveResolutionProvider(repo, client)
		if err != nil {
			t.Fatal(err)
		}
		provider.writeSparse = false
		return provider
	}
	ctx := context.Background()
	cold := makeProvider()
	secondsA, evidenceA, err := cold.SecondBars(ctx, MarketFuturesUSDT, "BTCUSDT", start, start+1999)
	if err != nil {
		t.Fatal(err)
	}
	tradesA, tradeEvidenceA, err := cold.Trades(ctx, MarketFuturesUSDT, "BTCUSDT", start, start+999)
	if err != nil {
		t.Fatal(err)
	}
	if len(secondsA) != 2 || len(tradesA) != 2 {
		t.Fatalf("cold rows seconds=%d trades=%d", len(secondsA), len(tradesA))
	}
	dbSeconds, err := repo.LoadSparseSecondBars(ctx, MarketFuturesUSDT, "BTCUSDT", start, start+1999)
	if err != nil {
		t.Fatal(err)
	}
	dbTrades, err := repo.LoadSparseTrades(ctx, MarketFuturesUSDT, "BTCUSDT", start, start+999)
	if err != nil {
		t.Fatal(err)
	}
	if len(dbSeconds) != 0 || len(dbTrades) != 0 {
		t.Fatalf("file-only mode wrote sparse DB: seconds=%d trades=%d", len(dbSeconds), len(dbTrades))
	}
	if cold.Stats().ArchiveDownloads != 2 {
		t.Fatalf("cold stats: %+v", cold.Stats())
	}
	if err := cold.Close(); err != nil {
		t.Fatal(err)
	}
	firstCount := requests.Load()
	if firstCount != 4 {
		t.Fatalf("downloads=%d", firstCount)
	}

	hot := makeProvider()
	secondsB, evidenceB, err := hot.SecondBars(ctx, MarketFuturesUSDT, "BTCUSDT", start, start+1999)
	if err != nil {
		t.Fatal(err)
	}
	tradesB, tradeEvidenceB, err := hot.Trades(ctx, MarketFuturesUSDT, "BTCUSDT", start, start+999)
	if err != nil {
		t.Fatal(err)
	}
	if requests.Load() != firstCount {
		t.Fatalf("warm cache redownloaded: %d -> %d", firstCount, requests.Load())
	}
	if !reflect.DeepEqual(secondsA, secondsB) || !reflect.DeepEqual(tradesA, tradesB) ||
		evidenceA.EvidenceHash != evidenceB.EvidenceHash || tradeEvidenceA.EvidenceHash != tradeEvidenceB.EvidenceHash {
		t.Fatalf("cold and disk-cache outputs differ: first=%+v second=%+v", evidenceA, evidenceB)
	}
	if hot.Stats().ArchiveCacheHits != 2 || hot.Stats().ArchiveDownloads != 0 {
		t.Fatalf("disk hit stats: %+v", hot.Stats())
	}
	if err := hot.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestBT3CacheEvictsLeastRecentlyUsedVerifiedZIPOnly(t *testing.T) {
	dayA, dayB := bt3Date(), bt3Date().Add(24*time.Hour)
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		name := "BTCUSDT-1s-2026-09-11.zip"
		if strings.Contains(r.URL.Path, "2026-09-12") {
			name = "BTCUSDT-1s-2026-09-12.zip"
		}
		z := []byte{}
		// Identical contents are sufficient here; paths and sidecars differ.
		z = append(z, []byte("synthetic-verified-content")...)
		if strings.HasSuffix(r.URL.Path, ".CHECKSUM") {
			_, _ = fmt.Fprint(w, publicDataChecksum(z, name))
			return
		}
		_, _ = w.Write(z)
	}))
	defer server.Close()
	dir := t.TempDir()
	client, err := NewPublicDataClient(PublicDataClientConfig{BaseURL: server.URL, CacheDir: dir, HTTPClient: server.Client(), MaxCacheBytes: 40})
	if err != nil {
		t.Fatal(err)
	}
	spec := func(day time.Time) PublicDataArchiveSpec {
		return PublicDataArchiveSpec{Kind: ArchiveKindKlines, Period: ArchivePeriodDaily, Symbol: "BTCUSDT", Interval: "1s", Date: day}
	}
	a, err := client.FetchArchive(context.Background(), spec(dayA))
	if err != nil {
		t.Fatal(err)
	}
	b, err := client.FetchArchive(context.Background(), spec(dayB))
	if err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-time.Hour)
	if err := os.Chtimes(a.Path, past, past); err != nil {
		t.Fatal(err)
	}
	userFile := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(userFile, []byte("never prune"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(a.Path); !os.IsNotExist(err) {
		t.Fatalf("old archive not evicted: %v", err)
	}
	if _, err := os.Stat(b.Path); err != nil {
		t.Fatalf("latest archive missing: %v", err)
	}
	if _, err := os.Stat(userFile); err != nil {
		t.Fatalf("unrelated file removed: %v", err)
	}
}

func TestBT3UMArchiveDoesNotServeCoinM(t *testing.T) {
	setupRepositoryTest(t)
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		http.Error(w, "not expected", 500)
	}))
	defer server.Close()
	client, err := NewPublicDataClient(PublicDataClientConfig{BaseURL: server.URL, CacheDir: t.TempDir(), HTTPClient: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	provider, err := NewAdaptiveResolutionProvider(NewRepository(nil), client)
	if err != nil {
		t.Fatal(err)
	}
	start := bt3Date().UnixMilli()
	if _, _, err := provider.SecondBars(context.Background(), "futures_coin", "BTCUSD_PERP", start, start+999); err == nil {
		t.Fatal("COIN-M must not use UM 1s ZIP")
	}
	if _, _, err := provider.Trades(context.Background(), "futures_coin", "BTCUSD_PERP", start, start+999); err == nil {
		t.Fatal("COIN-M must not use UM trades ZIP")
	}
	if requests.Load() != 0 {
		t.Fatalf("unexpected public requests=%d", requests.Load())
	}
}

func TestBT3DerivedSecondsMatchLegacyWriteMode(t *testing.T) {
	setupRepositoryTest(t)
	repo := NewRepository(nil)
	start := bt3Date().UnixMilli()
	filename := "BTCUSDT-trades-2026-09-11.zip"
	csv := fmt.Sprintf("10,100,1,100,%d,false\n11,101,2,202,%d,true\n12,99,1,99,%d,false\n", start+100, start+500, start+1200)
	zipBody := publicDataTestZIP(t, strings.TrimSuffix(filename, ".zip")+".csv", csv)
	var tradeDownloads atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/klines/") {
			http.NotFound(w, r)
			return
		}
		if strings.Contains(r.URL.Path, "/trades/") {
			if strings.HasSuffix(r.URL.Path, ".CHECKSUM") {
				_, _ = fmt.Fprint(w, publicDataChecksum(zipBody, filename))
				return
			}
			tradeDownloads.Add(1)
			_, _ = w.Write(zipBody)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	cacheDir := t.TempDir()
	makeProvider := func(write bool) *AdaptiveResolutionProvider {
		client, err := NewPublicDataClient(PublicDataClientConfig{
			BaseURL: server.URL, CacheDir: cacheDir, HTTPClient: server.Client(), MaxCacheBytes: 1 << 20,
		})
		if err != nil {
			t.Fatal(err)
		}
		provider, err := NewAdaptiveResolutionProvider(repo, client)
		if err != nil {
			t.Fatal(err)
		}
		provider.writeSparse = write
		return provider
	}
	fileMode := makeProvider(false)
	fileRows, fileEvidence, err := fileMode.SecondBars(context.Background(), MarketFuturesUSDT, "BTCUSDT", start, start+1999)
	if err != nil {
		t.Fatal(err)
	}
	if fileEvidence.Resolution != "1s_from_trades" || len(fileRows) != 2 {
		t.Fatalf("unexpected file-derived rows=%+v evidence=%+v", fileRows, fileEvidence)
	}
	before, err := repo.LoadSparseSecondBars(context.Background(), MarketFuturesUSDT, "BTCUSDT", start, start+1999)
	if err != nil {
		t.Fatal(err)
	}
	if len(before) != 0 {
		t.Fatalf("file mode wrote derived sparse rows: %d", len(before))
	}
	if err := fileMode.Close(); err != nil {
		t.Fatal(err)
	}
	legacy := makeProvider(true)
	dbRows, dbEvidence, err := legacy.SecondBars(context.Background(), MarketFuturesUSDT, "BTCUSDT", start, start+1999)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(fileRows, dbRows) || fileEvidence.EvidenceHash != dbEvidence.EvidenceHash ||
		fileEvidence.ArchiveSHA256 != dbEvidence.ArchiveSHA256 || fileEvidence.Resolution != dbEvidence.Resolution {
		t.Fatalf("write/file derived seconds differ: file=%+v db=%+v", fileEvidence, dbEvidence)
	}
	if tradeDownloads.Load() != 1 {
		t.Fatalf("legacy mode redownloaded verified trades ZIP: %d", tradeDownloads.Load())
	}
	after, err := repo.LoadSparseSecondBars(context.Background(), MarketFuturesUSDT, "BTCUSDT", start, start+1999)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != 2 {
		t.Fatalf("legacy write mode did not persist: %d", len(after))
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestBT3MissingSourceWithoutCacheFailsExplicitly(t *testing.T) {
	setupRepositoryTest(t)
	start := bt3Date().UnixMilli()
	client, err := NewPublicDataClient(PublicDataClientConfig{BaseURL: "http://127.0.0.1:1", CacheDir: t.TempDir(), Timeout: time.Second, MaxRetries: 1})
	if err != nil {
		t.Fatal(err)
	}
	provider, err := NewAdaptiveResolutionProvider(NewRepository(nil), client)
	if err != nil {
		t.Fatal(err)
	}
	provider.writeSparse = false
	if _, _, err := provider.Trades(context.Background(), MarketFuturesUSDT, "BTCUSDT", start, start+999); err == nil {
		t.Fatal("no local ZIP and unavailable source must fail, not produce precise results")
	}
	_ = provider.Close()
}

func TestBT3ActiveClientArchiveProtectedDuringOtherClientCleanup(t *testing.T) {
	zipBody := []byte("verified-cache-body-for-two-days")
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		name := "BTCUSDT-1s-2026-09-11.zip"
		if strings.Contains(r.URL.Path, "2026-09-12") {
			name = "BTCUSDT-1s-2026-09-12.zip"
		}
		if strings.HasSuffix(r.URL.Path, ".CHECKSUM") {
			_, _ = fmt.Fprint(w, publicDataChecksum(zipBody, name))
			return
		}
		_, _ = w.Write(zipBody)
	}))
	defer server.Close()
	dir := t.TempDir()
	newClient := func() *PublicDataClient {
		c, err := NewPublicDataClient(PublicDataClientConfig{
			BaseURL: server.URL, CacheDir: dir, HTTPClient: server.Client(), MaxCacheBytes: 40,
		})
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	spec := func(day time.Time) PublicDataArchiveSpec {
		return PublicDataArchiveSpec{Kind: ArchiveKindKlines, Period: ArchivePeriodDaily, Symbol: "BTCUSDT", Interval: "1s", Date: day}
	}
	clientA := newClient()
	archiveA, err := clientA.FetchArchive(context.Background(), spec(bt3Date()))
	if err != nil {
		t.Fatal(err)
	}
	clientB := newClient()
	archiveB, err := clientB.FetchArchive(context.Background(), spec(bt3Date().Add(24*time.Hour)))
	if err != nil {
		t.Fatal(err)
	}
	if err := clientB.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(archiveA.Path); err != nil {
		t.Fatalf("active client's archive evicted: %v", err)
	}
	if _, err := os.Stat(archiveB.Path); !os.IsNotExist(err) {
		t.Fatalf("second archive should be evicted while first active: %v", err)
	}
	if _, err := clientA.FetchArchive(context.Background(), spec(bt3Date())); err != nil {
		t.Fatal(err)
	}
	if err := clientA.Close(); err != nil {
		t.Fatal(err)
	}
}

package historicalmarket

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestResolutionProviderSecondBarsUsesCompleteSparseCache(t *testing.T) {
	setupRepositoryTest(t)
	repo := NewRepository(nil)
	start := time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC).UnixMilli()
	rows := []Kline{
		{Market: MarketFuturesUSDT, Symbol: "BTCUSDT", Interval: SparseSecondInterval, OpenTime: start, CloseTime: start + 999, Open: 100, High: 101, Low: 99, Close: 100.5, Source: "fixture"},
		{Market: MarketFuturesUSDT, Symbol: "BTCUSDT", Interval: SparseSecondInterval, OpenTime: start + 1000, CloseTime: start + 1999, Open: 100.5, High: 102, Low: 100, Close: 101, Source: "fixture"},
	}
	if _, err := repo.StoreSparseSecondBars(context.Background(), rows); err != nil {
		t.Fatal(err)
	}
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		http.Error(w, "must not fetch", http.StatusInternalServerError)
	}))
	defer server.Close()
	client, err := NewPublicDataClient(PublicDataClientConfig{BaseURL: server.URL, CacheDir: t.TempDir(), HTTPClient: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	provider, err := NewAdaptiveResolutionProvider(repo, client)
	if err != nil {
		t.Fatal(err)
	}
	got, evidence, err := provider.SecondBars(context.Background(), MarketFuturesUSDT, "BTCUSDT", start, start+1999)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || !evidence.CacheHit || requests.Load() != 0 {
		t.Fatalf("sparse cache was not reused: rows=%d evidence=%+v requests=%d", len(got), evidence, requests.Load())
	}
	if provider.Stats().SecondCacheHits != 1 {
		t.Fatalf("unexpected stats: %+v", provider.Stats())
	}
}

func TestResolutionProviderSecondBarsDownloadsVerifiedDailyKlines(t *testing.T) {
	setupRepositoryTest(t)
	start := time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC).UnixMilli()
	filename := "BTCUSDT-1s-2026-09-11.zip"
	csv := fmt.Sprintf("%d,100,101,99,100.5,2,%d,200,3,1,100,0\n%d,100.5,102,100,101,3,%d,303,4,2,202,0\n", start, start+999, start+1000, start+1999)
	zipped := publicDataTestZIP(t, strings.TrimSuffix(filename, ".zip")+".csv", csv)
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
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
	provider, err := NewAdaptiveResolutionProvider(NewRepository(nil), client)
	if err != nil {
		t.Fatal(err)
	}
	rows, evidence, err := provider.SecondBars(context.Background(), MarketFuturesUSDT, "BTCUSDT", start, start+1999)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || evidence.ArchiveSHA256 == "" || evidence.EvidenceHash == "" || evidence.CacheHit {
		t.Fatalf("unexpected remote second result: rows=%+v evidence=%+v", rows, evidence)
	}
	firstRequests := requests.Load()
	again, secondEvidence, err := provider.SecondBars(context.Background(), MarketFuturesUSDT, "BTCUSDT", start, start+1999)
	if err != nil {
		t.Fatal(err)
	}
	if len(again) != 2 || !secondEvidence.CacheHit || requests.Load() != firstRequests {
		t.Fatalf("persisted sparse seconds not reused: evidence=%+v requests=%d/%d", secondEvidence, requests.Load(), firstRequests)
	}
	if provider.Stats().ArchiveDownloads != 1 || provider.Stats().SecondCacheHits != 1 {
		t.Fatalf("unexpected resolution stats: %+v", provider.Stats())
	}
}

func TestResolutionProviderSecondBarsFallsBackToTrades(t *testing.T) {
	setupRepositoryTest(t)
	start := time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC).UnixMilli()
	tradeFilename := "BTCUSDT-trades-2026-09-11.zip"
	tradeCSV := fmt.Sprintf("10,100,1,100,%d,false\n11,101,2,202,%d,true\n12,99,1,99,%d,false\n", start+100, start+500, start+1200)
	tradeZIP := publicDataTestZIP(t, strings.TrimSuffix(tradeFilename, ".zip")+".csv", tradeCSV)
	var klineRequests atomic.Int32
	var tradeRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/klines/") {
			klineRequests.Add(1)
			http.NotFound(w, r)
			return
		}
		if strings.Contains(r.URL.Path, "/trades/") {
			if strings.HasSuffix(r.URL.Path, ".CHECKSUM") {
				fmt.Fprint(w, publicDataChecksum(tradeZIP, tradeFilename))
				return
			}
			tradeRequests.Add(1)
			w.Write(tradeZIP)
			return
		}
		http.NotFound(w, r)
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
	rows, evidence, err := provider.SecondBars(context.Background(), MarketFuturesUSDT, "BTCUSDT", start, start+1999)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].Open != 100 || rows[0].High != 101 || rows[0].Close != 101 || rows[0].TradeCount != 2 || rows[1].Close != 99 {
		t.Fatalf("trade aggregation failed: %+v", rows)
	}
	if evidence.Resolution != "1s_from_trades" || evidence.ArchiveSHA256 == "" {
		t.Fatalf("unexpected fallback evidence: %+v", evidence)
	}
	if klineRequests.Load() != 1 || tradeRequests.Load() != 1 {
		t.Fatalf("unexpected archive requests: kline=%d trade=%d", klineRequests.Load(), tradeRequests.Load())
	}
	firstKline, firstTrade := klineRequests.Load(), tradeRequests.Load()
	again, againEvidence, err := provider.SecondBars(context.Background(), MarketFuturesUSDT, "BTCUSDT", start, start+1999)
	if err != nil {
		t.Fatal(err)
	}
	if len(again) != 2 || !againEvidence.CacheHit || klineRequests.Load() != firstKline || tradeRequests.Load() != firstTrade {
		t.Fatalf("derived sparse cache not reused: evidence=%+v requests=%d/%d", againEvidence, klineRequests.Load(), tradeRequests.Load())
	}
}

func TestAggregateTradesToSecondsUsesTakerBuySemantics(t *testing.T) {
	start := time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC).UnixMilli()
	rows := aggregateTradesToSeconds([]PublicDataTrade{
		{Market: MarketFuturesUSDT, Symbol: "BTCUSDT", TradeID: 2, TradeTime: start + 500, Price: 101, Quantity: 2, QuoteQuantity: 202, IsBuyerMaker: true},
		{Market: MarketFuturesUSDT, Symbol: "BTCUSDT", TradeID: 1, TradeTime: start + 100, Price: 100, Quantity: 1, QuoteQuantity: 100, IsBuyerMaker: false},
	}, start, start+999)
	if len(rows) != 1 || rows[0].Open != 100 || rows[0].Close != 101 || rows[0].High != 101 || rows[0].Low != 100 {
		t.Fatalf("unexpected OHLC aggregation: %+v", rows)
	}
	if rows[0].TakerBuyBaseVolume != 1 || rows[0].TakerBuyQuoteVolume != 100 {
		t.Fatalf("buyer-maker semantics inverted: %+v", rows[0])
	}
}

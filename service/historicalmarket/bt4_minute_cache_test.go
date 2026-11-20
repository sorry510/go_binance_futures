package historicalmarket

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
)

func TestBT4MinuteTradesReuseMatchesPerSecondArchiveEvidence(t *testing.T) {
	setupRepositoryTest(t)
	start := bt3Date().UnixMilli()
	filename := "BTCUSDT-trades-2026-09-11.zip"
	csv := fmt.Sprintf("13,99,1,99,%d,false\n10,100,1,100,%d,false\n11,101,2,202,%d,true\n12,102,1,102,%d,false\n14,103,1,103,%d,true\n",
		start+100, start+1100, start+1150, start+59_750, start+60_250)
	zipped := publicDataTestZIP(t, strings.TrimSuffix(filename, ".zip")+".csv", csv)
	var tradeDownloads atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/klines/") {
			http.NotFound(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, ".CHECKSUM") {
			_, _ = fmt.Fprint(w, publicDataChecksum(zipped, filename))
			return
		}
		tradeDownloads.Add(1)
		_, _ = w.Write(zipped)
	}))
	defer server.Close()

	client, err := NewPublicDataClient(PublicDataClientConfig{
		BaseURL: server.URL, CacheDir: t.TempDir(), HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	provider, err := NewAdaptiveResolutionProvider(NewRepository(nil), client)
	if err != nil {
		t.Fatal(err)
	}
	provider.writeSparse = false
	defer provider.Close()
	ctx := context.Background()

	bars, parentEvidence, err := provider.SecondBars(ctx, MarketFuturesUSDT, "BTCUSDT", start, start+59_999)
	if err != nil {
		t.Fatal(err)
	}
	if len(bars) != 3 || parentEvidence.Resolution != "1s_from_trades" || tradeDownloads.Load() != 1 {
		t.Fatalf("unexpected parent bars=%d evidence=%+v downloads=%d", len(bars), parentEvidence, tradeDownloads.Load())
	}
	if stats := provider.Stats(); stats.ZipScannerOpens != 1 || stats.ZipScannerRewinds != 0 {
		t.Fatalf("initial minute should scan ZIP just once: %+v", stats)
	}
	archive, err := client.FetchArchive(ctx, PublicDataArchiveSpec{
		Kind: ArchiveKindTrades, Period: ArchivePeriodDaily, Symbol: "BTCUSDT", Date: bt3Date(),
	})
	if err != nil {
		t.Fatal(err)
	}
	all, err := client.ParseTrades(ctx, archive, start, start+59_999)
	if err != nil {
		t.Fatal(err)
	}

	for _, second := range []int64{0, 1000, 2000, 59_000} {
		rangeStart, rangeEnd := start+second, start+second+999
		got, evidence, err := provider.Trades(ctx, MarketFuturesUSDT, "BTCUSDT", rangeStart, rangeEnd)
		if err != nil {
			t.Fatal(err)
		}
		want := make([]PublicDataTrade, 0)
		for _, tr := range all {
			if tr.TradeTime >= rangeStart && tr.TradeTime <= rangeEnd {
				tr.SourceRef = archive.URL + "#sha256=" + archive.SHA256 + "&" + resolutionRangeTag(rangeStart, rangeEnd)
				want = append(want, tr)
			}
		}
		if !reflect.DeepEqual(got, want) ||
			evidence.EvidenceHash != tradeEvidenceHash(rangeStart, rangeEnd, want) ||
			evidence.SourceRef != archive.URL+"#sha256="+archive.SHA256+"&"+resolutionRangeTag(rangeStart, rangeEnd) ||
			evidence.Rows != len(want) {
			t.Fatalf("second %d mismatch rows=%+v want=%+v evidence=%+v", second, got, want, evidence)
		}
	}
	if tradeDownloads.Load() != 1 || provider.Stats().MinuteTradeReuseHits != 4 {
		t.Fatalf("repeated fetch or missing reuse: downloads=%d stats=%+v", tradeDownloads.Load(), provider.Stats())
	}
	// A caller cannot corrupt the shared frame; cancellation takes precedence.
	reused, _, err := provider.Trades(ctx, MarketFuturesUSDT, "BTCUSDT", start, start+999)
	if err != nil {
		t.Fatal(err)
	}
	reused[0].Price = 1
	again, _, err := provider.Trades(ctx, MarketFuturesUSDT, "BTCUSDT", start, start+999)
	if err != nil || again[0].Price != 99 {
		t.Fatalf("minute snapshot mutated: rows=%+v err=%v", again, err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, _, err := provider.Trades(cancelled, MarketFuturesUSDT, "BTCUSDT", start, start+999); err != context.Canceled {
		t.Fatalf("cancelled lookup returned %v", err)
	}
	// Next candidate minute must discard the previous minute and build a new one.
	next, _, err := provider.SecondBars(ctx, MarketFuturesUSDT, "BTCUSDT", start+60_000, start+119_999)
	if err != nil || len(next) != 1 {
		t.Fatalf("next minute rows=%+v err=%v", next, err)
	}
	if _, _, hit := provider.reuseMinuteTrades(MarketFuturesUSDT, "BTCUSDT", start, start+999); hit {
		t.Fatal("previous minute unexpectedly retained")
	}
}

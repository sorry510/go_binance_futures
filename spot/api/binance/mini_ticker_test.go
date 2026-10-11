package binance

import (
	"math"
	"strconv"
	"testing"
	"time"

	spotbinance "github.com/adshao/go-binance/v2"
)

func TestNormalizeSpotMiniTickersPreservesPricesAndOHLCV(t *testing.T) {
	now := time.Now().UnixMilli()
	event := spotbinance.WsAllMiniMarketsStatEvent{
		&spotbinance.WsMiniMarketsStatEvent{
			Symbol:      "BTCUSDT",
			Time:        now,
			LastPrice:   "110.0",
			OpenPrice:   "100.0",
			HighPrice:   "115.0",
			LowPrice:    "90.0",
			BaseVolume:  "1000",
			QuoteVolume: "200000",
		},
		nil,
		&spotbinance.WsMiniMarketsStatEvent{Symbol: "", LastPrice: "1"},
	}
	result := normalizeSpotMiniTickers(event)
	if len(result) != 1 {
		t.Fatalf("expected one real spot ticker, got %d", len(result))
	}
	got := result[0]
	if got.Symbol != "BTCUSDT" || got.LastPrice != "110.0" || got.OpenPrice != "100.0" || got.HighPrice != "115.0" || got.LowPrice != "90.0" || got.Time != now {
		t.Fatalf("invalid spot mini ticker converted: %#v", got)
	}
	if got.QuoteVolume != "200000" || got.BaseVolume != "1000" || got.CloseQty != "0" {
		t.Fatalf("incorrect mini ticker volume or unavailable quantity: %#v", got)
	}
	pct, err := strconv.ParseFloat(got.PriceChangePercent, 64)
	if err != nil || math.Abs(pct-10) > 0.00001 {
		t.Fatalf("incorrect 24h percent change: %s", got.PriceChangePercent)
	}
	// Same converted events must feed the existing price snapshot pipeline.
	spotTickerSnapshot.Lock()
	savedItems := spotTickerSnapshot.Items
	spotTickerSnapshot.Items = make(map[string]spotTickerSnapshotEntry)
	spotTickerSnapshot.Unlock()
	defer func() {
		spotTickerSnapshot.Lock()
		spotTickerSnapshot.Items = savedItems
		spotTickerSnapshot.Unlock()
	}()
	storeSpotTickerEvents(result, time.Now())
	if price, ok := GetFreshSpotTickerPrice("BTCUSDT"); !ok || price != "110.0" {
		t.Fatalf("converted WS price not cached: %q %v", price, ok)
	}
}

func TestNormalizeSpotMiniTickersWithZeroOpenDoesNotDivideByZero(t *testing.T) {
	rows := normalizeSpotMiniTickers(spotbinance.WsAllMiniMarketsStatEvent{
		&spotbinance.WsMiniMarketsStatEvent{Symbol: "NEWUSDT", LastPrice: "1.4", OpenPrice: "0"},
	})
	if len(rows) != 1 || rows[0].PriceChangePercent != "0" {
		t.Fatalf("unexpected zero-open normalization: %#v", rows)
	}
}

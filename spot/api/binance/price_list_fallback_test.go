package binance

import (
	"context"
	"testing"
	"time"
)

func TestSpotBulkFallbackSelectsRequestedSymbolsFromCache(t *testing.T) {
	spotAllPricesMu.Lock()
	previous := spotAllPrices
	previousExpiry := spotAllPricesExpires
	spotAllPrices = map[string]string{
		"BTCUSDT": "123.456",
		"ETHUSDT": "5",
		"BTC":     "",
	}
	spotAllPricesExpires = time.Now().Add(15 * time.Second)
	spotAllPricesMu.Unlock()
	t.Cleanup(func() {
		spotAllPricesMu.Lock()
		spotAllPrices = previous
		spotAllPricesExpires = previousExpiry
		spotAllPricesMu.Unlock()
	})

	result, err := GetSpotPricesBulkFallback(context.Background(), []string{" btcusdt ", "INVALID", "BTC"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 || result["BTCUSDT"] != "123.456" {
		t.Fatalf("unexpected filtered spot prices: %#v", result)
	}
	empty, err := GetSpotPricesBulkFallback(context.Background(), nil)
	if err != nil || len(empty) != 0 {
		t.Fatalf("nil symbols should return empty result: %#v %v", empty, err)
	}
}

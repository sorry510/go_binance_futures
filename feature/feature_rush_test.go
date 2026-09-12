package feature

import (
	"strings"
	"testing"

	"go_binance_futures/models"
)

func TestTryBuyMarketRejectsInvalidSideBeforeMarketAccess(t *testing.T) {
	_, err := tryBuyMarket(models.NewSymbols{Symbol: "BTCUSDT", Side: "hold"}, "0.001")
	if err == nil || !strings.Contains(err.Error(), "unsupported futures rush side") {
		t.Fatalf("invalid side must fail before any market/order access, err=%v", err)
	}
}

func TestTryBuyMarketRejectsInvalidConfiguredPriceBeforeOrder(t *testing.T) {
	_, err := tryBuyMarket(models.NewSymbols{Symbol: "BTCUSDT", Side: "buy", ExpectPrice: "bad", Usdt: "10", Leverage: 1}, "0.001")
	if err == nil || !strings.Contains(err.Error(), "invalid futures rush price") {
		t.Fatalf("invalid configured price must fail before order submission, err=%v", err)
	}
}

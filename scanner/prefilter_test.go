package scanner

import (
	"testing"

	"go_binance_futures/models"
)

func TestPrefilterTop30UsesLocalSymbolDataOnly(t *testing.T) {
	items := []*models.Symbols{
		symbol("BTCUSDT", 4, "100", "96", "95", "102", 1_000_000_000, "USDT"),
		symbol("ETHUSDT", 5, "100", "95", "94", "103", 1_000_000_000, "USDT"),
		symbol("LOWVOLUSDT", 10, "1.10", "1.00", "0.99", "1.12", 5_000_000, "USDT"),
		symbol("PUMPUSDT", 75, "1.75", "1.00", "0.98", "1.90", 50_000_000, "USDT"),
		symbol("WICKUSDT", 45, "1.45", "1.00", "0.98", "2.40", 50_000_000, "USDT"),
		symbol("FARUSDT", 25, "1.50", "1.20", "1.00", "1.52", 50_000_000, "USDT"),
		symbol("GOODUSDT", 12, "1.12", "1.00", "0.99", "1.14", 80_000_000, "USDT"),
		symbol("BETTERUSDT", 8, "1.08", "1.00", "0.99", "1.09", 120_000_000, "USDT"),
	}
	items[6].LastClose = "1.10"
	items[6].LastUpdateTime = 1000
	items[6].UpdateTime = 2000
	items[7].LastClose = "1.02"
	items[7].LastUpdateTime = 1000
	items[7].UpdateTime = 2000

	result := PrefilterTop30FromSymbols(items, PrefilterOptions{
		Limit:          30,
		MinQuoteVolume: 10_000_000,
	})

	if got, want := len(result.Candidates), 2; got != want {
		t.Fatalf("candidate count = %d, want %d: %#v", got, want, result.Candidates)
	}
	if result.Candidates[0].Symbol != "BETTERUSDT" {
		t.Fatalf("top candidate = %s, want BETTERUSDT", result.Candidates[0].Symbol)
	}
	if result.Candidates[1].Symbol != "GOODUSDT" {
		t.Fatalf("second candidate = %s, want GOODUSDT", result.Candidates[1].Symbol)
	}

	excluded := map[string]string{}
	for _, item := range result.Excluded {
		excluded[item.Symbol] = item.Reason
	}
	for _, sym := range []string{"BTCUSDT", "ETHUSDT", "LOWVOLUSDT", "PUMPUSDT", "WICKUSDT", "FARUSDT"} {
		if excluded[sym] == "" {
			t.Fatalf("%s was not excluded: %#v", sym, result.Excluded)
		}
	}
	if len(result.DataMissing) == 0 {
		t.Fatal("expected missing local-only dimensions to be reported")
	}
}

func symbol(name string, pct float64, close, open, low, high string, quoteVolume float64, quoteType string) *models.Symbols {
	return &models.Symbols{
		Symbol:        name,
		PercentChange: pct,
		Close:         close,
		Open:          open,
		Low:           low,
		High:          high,
		QuoteVolume:   quoteVolume,
		Type:          quoteType,
	}
}

package controllers

import (
	"reflect"
	"testing"

	"github.com/adshao/go-binance/v2/futures"
)

func TestFuturesListenSymbolOptionsOnlyTradablePerpetuals(t *testing.T) {
	symbols := []futures.Symbol{
		{Symbol: "ETHUSDT", Status: "TRADING", ContractType: futures.ContractTypePerpetual, QuoteAsset: "USDT"},
		{Symbol: "BTCUSDT", Status: "TRADING", ContractType: futures.ContractTypePerpetual, QuoteAsset: "USDT"},
		{Symbol: "BTCUSDT", Status: "TRADING", ContractType: futures.ContractTypePerpetual, QuoteAsset: "USDT"},
		{Symbol: "XRPUSDC", Status: "TRADING", ContractType: futures.ContractTypePerpetual, QuoteAsset: "USDC"},
		{Symbol: "NEWFDUSD", Status: "TRADING", ContractType: futures.ContractTypePerpetual, QuoteAsset: "FDUSD"},
		{Symbol: "ADAUSDT", Status: "BREAK", ContractType: futures.ContractTypePerpetual, QuoteAsset: "USDT"},
		{Symbol: "BUSDUSD", Status: "TRADING", ContractType: futures.ContractTypePerpetual, QuoteAsset: "BUSD"},
		{Symbol: "BTCUSDT_261225", Status: "TRADING", ContractType: futures.ContractType("CURRENT_QUARTER"), QuoteAsset: "USDT"},
		{Symbol: "", Status: "TRADING", ContractType: futures.ContractTypePerpetual, QuoteAsset: "USDT"},
	}
	expected := []string{"BTCUSDT", "ETHUSDT", "NEWFDUSD", "XRPUSDC"}
	got := futuresListenSymbolOptions(symbols)
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("unexpected tradable perpetual options: got=%v want=%v", got, expected)
	}
}

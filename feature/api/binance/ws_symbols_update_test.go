package binance

import (
	"strings"
	"testing"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/core/config"
)

func TestWsSymbolUpdateRetainsInsertUpsertSemantics(t *testing.T) {
	ticker := futures.WsMarketTickerEvent{
		Symbol: "BTCUSDT", ClosePrice: "65000", OpenPrice: "64000",
		LowPrice: "63000", HighPrice: "66000", Time: 1700000000000,
	}
	query, _ := buildBatchUpdateSymbolsSQL([]futures.WsMarketTickerEvent{ticker})
	upper := strings.ToUpper(query)
	if !strings.Contains(upper, "INSERT INTO `SYMBOLS`") {
		t.Fatalf("ws symbol update must retain insert/upsert semantics: %s", query)
	}

	driver, _ := config.String("database::driver")
	if driver == "mysql" && !strings.Contains(upper, "ON DUPLICATE KEY UPDATE") {
		t.Fatalf("mysql ws symbol update must use ON DUPLICATE KEY UPDATE: %s", query)
	}
	if driver != "mysql" && !strings.Contains(upper, "ON CONFLICT(`SYMBOL`) DO UPDATE") {
		t.Fatalf("non-mysql ws symbol update must use ON CONFLICT upsert: %s", query)
	}
}

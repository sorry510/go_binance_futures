package binance

import (
	"strings"
	"testing"

	spotbinance "github.com/adshao/go-binance/v2"
	"github.com/beego/beego/v2/core/config"
)

func TestSpotWSUpsertCreatesPricesWithoutResettingPreferences(t *testing.T) {
	rows := []spotbinance.WsMarketStatEvent{
		{Symbol: "BTCUSDT", LastPrice: "65000", OpenPrice: "64000", Time: 1700000000000},
		{Symbol: "ETHUSDC", LastPrice: "3500", Time: 1700000000001},
		{Symbol: "ETHBTC", LastPrice: "0.05", Time: 1700000000002},
	}
	query, args := buildSpotWSUpsertSQL(rows)
	if len(args) != 40 {
		t.Fatalf("want 2 allowed spot quotes (40 args), got %d", len(args))
	}
	upper := strings.ToUpper(query)
	if !strings.Contains(upper, "INSERT INTO `SPOT_SYMBOLS`") {
		t.Fatalf("spot WS prices must upsert missing pairs: %s", query)
	}
	driver, _ := config.String("database::driver")
	if driver == "mysql" {
		if !strings.Contains(upper, "ON DUPLICATE KEY UPDATE") {
			t.Fatalf("expected mysql upsert: %s", query)
		}
	} else if !strings.Contains(upper, "ON CONFLICT(`SYMBOL`) DO UPDATE") {
		t.Fatalf("expected sqlite upsert: %s", query)
	}
	// During price refresh, preserve the user's enable, risk and strategy fields.
	update := strings.Split(upper, " UPDATE ")
	if len(update) < 2 {
		t.Fatalf("missing upsert update: %s", query)
	}
	clause := update[len(update)-1]
	for _, field := range []string{"`ENABLE` =", "`STRATEGY` =", "`TECHNOLOGY` =", "`USDT` =", "`PROFIT` ="} {
		if strings.Contains(clause, field) {
			t.Fatalf("WS upsert must not overwrite user preference %s: %s", field, clause)
		}
	}
}

func TestSpotWSQueueIgnoresUnsupportedAndOlderTicks(t *testing.T) {
	spotWSLatestMu.Lock()
	original := spotWSLatest
	spotWSLatest = make(map[string]spotbinance.WsMarketStatEvent)
	spotWSLatestMu.Unlock()
	defer func() {
		spotWSLatestMu.Lock()
		spotWSLatest = original
		spotWSLatestMu.Unlock()
	}()
	enqueueSpotWSTickers(spotbinance.WsAllMarketsStatEvent{
		&spotbinance.WsMarketStatEvent{Symbol: "BTCUSDT", LastPrice: "10", Time: 100},
		&spotbinance.WsMarketStatEvent{Symbol: "ETHBTC", LastPrice: "0.05", Time: 100},
	})
	enqueueSpotWSTickers(spotbinance.WsAllMarketsStatEvent{
		&spotbinance.WsMarketStatEvent{Symbol: "BTCUSDT", LastPrice: "9", Time: 99},
	})
	spotWSLatestMu.Lock()
	got, ok := spotWSLatest["BTCUSDT"]
	length := len(spotWSLatest)
	spotWSLatestMu.Unlock()
	if !ok || got.LastPrice != "10" || length != 1 {
		t.Fatalf("invalid last-wins ticker buffer %+v, count %d", got, length)
	}
}

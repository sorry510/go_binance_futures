package binance

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/adshao/go-binance/v2/delivery"
)

func TestCM6KlinesUseSelectedDAPIPublicEndpointOnly(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Minute)
	seen := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen++
		if r.Method != http.MethodGet || r.URL.Path != "/dapi/v1/klines" {
			t.Errorf("coin-m Kline called wrong route: %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("symbol") != "BTCUSD_PERP" ||
			r.URL.Query().Get("interval") != "1m" || r.URL.Query().Get("limit") != "3" {
			t.Errorf("wrong selected DAPI Kline params: %s", r.URL.RawQuery)
		}
		if r.Header.Get("X-MBX-APIKEY") != "" {
			t.Error("public DAPI candle must not send trading credentials")
		}
		row := func(open int64) string {
			return fmt.Sprintf(`[%d,"50000","50100","49900","50020","200",%d,"10000000",10,"150","7500000"]`,
				open, open+59999)
		}
		fmt.Fprintf(w, "[%s,%s,%s]", row(now.Add(-2*time.Minute).UnixMilli()),
			row(now.Add(-time.Minute).UnixMilli()), row(now.UnixMilli()))
	}))
	defer server.Close()
	oldClient, oldMode, oldErr := deliveryClient, coinMRuntime, coinMSetupErr
	defer func() { deliveryClient, coinMRuntime, coinMSetupErr = oldClient, oldMode, oldErr }()
	guard, err := newCoinMRESTGuard(server.Client().Transport, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	coinMRuntime = coinMRuntimeConfig{Environment: "testnet", RESTURL: server.URL, WSURL: delivery.BaseWsTestnetUrl}
	coinMSetupErr = nil
	deliveryClient = delivery.NewClient("", "")
	deliveryClient.BaseURL = server.URL
	deliveryClient.HTTPClient = &http.Client{Transport: guard}
	klines, err := CoinMKlinesContext(context.Background(), "BTCUSD_PERP", "1m", 3)
	if err != nil || len(klines) != 3 || klines[2].Close != "50020" || klines[2].OpenTime != now.UnixMilli() {
		t.Fatalf("DAPI Kline source invalid: rows=%d err=%v", len(klines), err)
	}
	if seen != 1 {
		t.Fatalf("unexpected Kline requests: %d", seen)
	}
	for _, tc := range []struct {
		symbol, interval string
		limit            int
	}{
		{"BTCUSDT", "1m", 3}, {"BTCUSD_261225", "1m", 3}, {"BTCUSD_PERP", "bad", 3}, {"BTCUSD_PERP", "1m", 0},
		{"BTCUSD_PERP", "1m", 1501},
	} {
		if _, err := CoinMKlinesContext(context.Background(), tc.symbol, tc.interval, tc.limit); err == nil {
			t.Fatalf("accepted invalid COIN-M input %+v", tc)
		}
	}
	if seen != 1 {
		t.Fatalf("invalid Kline input sent remote requests: %d", seen)
	}
	deliveryClient.BaseURL = delivery.BaseApiMainUrl
	if _, err := CoinMKlinesContext(context.Background(), "BTCUSD_PERP", "1m", 3); err == nil ||
		!strings.Contains(err.Error(), "unconfigured") {
		t.Fatalf("cross-COIN-M-environment candle fallback allowed: %v", err)
	}
	if seen != 1 {
		t.Fatalf("cross-environment read hit selected host unexpectedly: %d", seen)
	}
}

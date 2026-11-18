package binance

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/adshao/go-binance/v2/delivery"
)

func TestCM4PublicMarkPriceOnlyUsesSelectedDAPINetwork(t *testing.T) {
	now := time.Now().UnixMilli()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/dapi/v1/premiumIndex" {
			t.Errorf("unexpected mark endpoint %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("X-MBX-APIKEY") != "" {
			t.Error("public mark endpoint must not send API key")
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `[{"symbol":"BTCUSD_PERP","markPrice":"55000.12345678","indexPrice":"55002.22","time":`+
			strconv.FormatInt(now, 10)+`},{"symbol":"BTCUSD_261225","markPrice":"53000","time":`+
			strconv.FormatInt(now, 10)+`}]`)
	}))
	defer server.Close()
	oldClient, oldMode, oldErr := deliveryClient, coinMRuntime, coinMSetupErr
	defer func() { deliveryClient, coinMRuntime, coinMSetupErr = oldClient, oldMode, oldErr }()
	coinMRuntime = coinMRuntimeConfig{Environment: "testnet", RESTURL: server.URL, WSURL: delivery.BaseWsTestnetUrl}
	coinMSetupErr = nil
	guard, err := newCoinMRESTGuard(server.Client().Transport, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	deliveryClient = delivery.NewClient("", "")
	deliveryClient.BaseURL = server.URL
	deliveryClient.HTTPClient = &http.Client{Transport: guard}
	marks, err := GetCoinMMarkPricesContext(context.Background())
	if err != nil || len(marks) != 2 || marks[0].MarkPrice != "55000.12345678" ||
		!CoinMMarkTimeValid(marks[0].Time, time.Now().UnixMilli()) {
		t.Fatalf("public DAPI mark response %+v: %v", marks, err)
	}
	// The REST guard must continue to refuse any other environment host.
	deliveryClient.BaseURL = delivery.BaseApiMainUrl
	if _, err := GetCoinMMarkPricesContext(context.Background()); err == nil ||
		!strings.Contains(err.Error(), "unconfigured") {
		t.Fatalf("cross-env DAPI mark fallback accepted: %v", err)
	}
}
func TestCM4MarkPriceFreshness(t *testing.T) {
	now := time.Now().UnixMilli()
	cases := []struct {
		mark int64
		want bool
	}{
		{now, true}, {now - 119000, true}, {now - 121000, false}, {now + 29000, true},
		{now + 31000, false}, {0, false},
	}
	for _, tc := range cases {
		if got := CoinMMarkTimeValid(tc.mark, now); got != tc.want {
			t.Errorf("mark timestamp %d, fresh=%t want %t", tc.mark, got, tc.want)
		}
	}
}

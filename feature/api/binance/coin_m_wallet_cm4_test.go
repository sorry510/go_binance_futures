package binance

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/adshao/go-binance/v2/delivery"
)

func TestCM4WalletSignedReadStaysOnChosenDAPINetwork(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodGet || r.URL.Path != "/dapi/v1/balance" {
			t.Errorf("signed wallet escaped intended DAPI balance: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("X-MBX-APIKEY") != "CM4-TEST-ONLY" {
			t.Errorf("wallet request omitted selected network API key: %q", r.Header.Get("X-MBX-APIKEY"))
		}
		if r.URL.Query().Get("timestamp") == "" || r.URL.Query().Get("signature") == "" {
			t.Errorf("wallet request must use signed DAPI query, got %v", r.URL.Query())
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `[{"asset":"BTC","balance":"1.123456789012345678","availableBalance":"0.75","crossWalletBalance":"1.1","crossUnPnl":"0.001","withdrawAvailable":"0.7","updateTime":1730000000000}]`)
	}))
	defer srv.Close()
	beforeClient, beforeMode, beforeErr := deliveryClient, coinMRuntime, coinMSetupErr
	defer func() { deliveryClient, coinMRuntime, coinMSetupErr = beforeClient, beforeMode, beforeErr }()
	guard, err := newCoinMRESTGuard(srv.Client().Transport, srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	coinMRuntime = coinMRuntimeConfig{Environment: "testnet", RESTURL: srv.URL, WSURL: delivery.BaseWsTestnetUrl, APIKey: "CM4-TEST-ONLY", APISecret: "test-secret"}
	coinMSetupErr = nil
	deliveryClient = delivery.NewClient(coinMRuntime.APIKey, coinMRuntime.APISecret)
	deliveryClient.BaseURL = srv.URL
	deliveryClient.HTTPClient = &http.Client{Transport: guard}
	wallet, err := GetDeliveryBalanceContext(context.Background())
	if err != nil || len(wallet) != 1 || wallet[0].Asset != "BTC" ||
		wallet[0].Balance != "1.123456789012345678" || calls != 1 {
		t.Fatalf("signed CM-4 wallet read incorrectly mapped: %+v %v calls=%d", wallet, err, calls)
	}
	deliveryClient.BaseURL = delivery.BaseApiMainUrl
	if _, err := GetDeliveryBalanceContext(context.Background()); err == nil || !strings.Contains(err.Error(), "endpoint mismatch") {
		t.Fatalf("COIN-M testnet wallet accepted cross-host mainnet fallback: %v", err)
	}
	if calls != 1 {
		t.Fatalf("cross-host wallet request was sent: %d", calls)
	}
}

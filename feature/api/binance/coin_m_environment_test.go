package binance

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/adshao/go-binance/v2/delivery"
)

// None of these cases sends network traffic to Binance; credentials are dummy.
func TestCM2RuntimeEnvironmentAndCredentials(t *testing.T) {
	tests := []struct {
		name, requested, want, rest, ws, key string
		testnet                              bool
		fail                                 bool
	}{
		{"mainnet default", "", "mainnet", delivery.BaseApiMainUrl, delivery.BaseWsMainUrl, "MAIN", false, false},
		{"testnet default", "", "testnet", delivery.BaseApiTestnetUrl, delivery.BaseWsTestnetUrl, "TEST", true, false},
		{"testnet explicit", "testnet", "testnet", delivery.BaseApiTestnetUrl, delivery.BaseWsTestnetUrl, "TEST", true, false},
		{"demo explicit", "demo", "demo", delivery.BaseApiDemoURL, delivery.BaseWsDemoURL, "DEMO", true, false},
		{"demo forbidden from mainnet mode", "demo", "", "", "", "", false, true},
		{"mainnet forbidden in testnet mode", "mainnet", "", "", "", "", true, true},
		{"testnet forbidden from mainnet mode", "testnet", "", "", "", "", false, true},
		{"invalid environment", "other", "", "", "", "", true, true},
	}
	origTestnet, origDemo := delivery.UseTestnet, delivery.UseDemo
	defer func() { delivery.UseTestnet, delivery.UseDemo = origTestnet, origDemo }()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := resolveCoinMRuntime(tt.testnet, tt.requested,
				"MAIN", "main-secret", "TEST", "test-secret", "DEMO", "demo-secret")
			if tt.fail {
				if err == nil {
					t.Fatal("expected to reject cross-environment override")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if config.Environment != tt.want || config.RESTURL != tt.rest ||
				config.WSURL != tt.ws || config.APIKey != tt.key {
				t.Fatalf("incorrect COIN-M endpoint/credential selection: env=%s REST=%s WS=%s key=%s",
					config.Environment, config.RESTURL, config.WSURL, config.APIKey)
			}
			client, err := configureCoinMRuntime(config)
			if err != nil || client.BaseURL != tt.rest {
				t.Fatalf("SDK did not use selected REST base: %+v %v", client, err)
			}
			if err := validateCoinMWSConfiguration(config); err != nil {
				t.Fatalf("SDK WS mode inconsistent with REST: %v", err)
			}
			delivery.UseTestnet = !delivery.UseTestnet
			if err := validateCoinMWSConfiguration(config); err == nil {
				t.Fatal("WS fallback/cross-mode mismatch was permitted")
			}
			delivery.UseTestnet = !delivery.UseTestnet
		})
	}
}

type cm2FakeRT func(*http.Request) (*http.Response, error)

func (fn cm2FakeRT) RoundTrip(req *http.Request) (*http.Response, error) { return fn(req) }

func TestCM2RESTGuardNeverFallsBackToMainnet(t *testing.T) {
	for _, endpoint := range []string{delivery.BaseApiTestnetUrl, delivery.BaseApiDemoURL} {
		calls := 0
		guard, err := newCoinMRESTGuard(cm2FakeRT(func(req *http.Request) (*http.Response, error) {
			calls++
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("{}")),
				Header: http.Header{}, Request: req}, nil
		}), endpoint)
		if err != nil {
			t.Fatal(err)
		}
		client := &http.Client{Transport: guard}
		for _, bad := range []string{
			delivery.BaseApiMainUrl + "/dapi/v1/order",
			"https://fapi.binance.com/dapi/v1/account",
			endpoint + "/fapi/v1/order",
			endpoint + "/api/v3/order",
			endpoint + "/sapi/v1/whatever",
		} {
			if _, err := client.Get(bad); err == nil {
				t.Fatalf("guard allowed a cross-environment or cross-product endpoint %s", bad)
			}
		}
		if calls != 0 {
			t.Fatalf("rejected calls reached network: %d", calls)
		}
		res, err := client.Get(endpoint + "/dapi/v1/exchangeInfo")
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if calls != 1 {
			t.Fatalf("DAPI public request blocked or duplicated calls=%d", calls)
		}
	}
	_, err := newCoinMRESTGuard(nil, "invalid")
	if err == nil {
		t.Fatal("accepted malformed DAPI host")
	}
}

func TestCM2RESTGuardRejectsCrossHostRedirect(t *testing.T) {
	guard, err := newCoinMRESTGuard(cm2FakeRT(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 302, Header: http.Header{"Location": []string{
			"https://dapi.binance.com/dapi/v1/order"}}, Body: io.NopCloser(strings.NewReader("")),
			Request: req}, nil
	}), delivery.BaseApiDemoURL)
	if err != nil {
		t.Fatal(err)
	}
	_, err = (&http.Client{Transport: guard}).Get(delivery.BaseApiDemoURL + "/dapi/v1/exchangeInfo")
	if err == nil || !strings.Contains(err.Error(), "cross-environment") {
		t.Fatalf("demo redirect to mainnet bypassed host guard: %v", err)
	}
}

func TestCM2NoPrivateReadWithMissingCredentials(t *testing.T) {
	oldClient, oldEnv, oldErr := deliveryClient, coinMRuntime, coinMSetupErr
	defer func() { deliveryClient, coinMRuntime, coinMSetupErr = oldClient, oldEnv, oldErr }()
	coinMRuntime = coinMRuntimeConfig{
		Environment: "testnet", RESTURL: delivery.BaseApiTestnetUrl, WSURL: delivery.BaseWsTestnetUrl,
	}
	coinMSetupErr = nil
	deliveryClient = delivery.NewClient("", "")
	deliveryClient.BaseURL = coinMRuntime.RESTURL
	if _, err := GetDeliveryAccountContext(context.Background()); err == nil {
		t.Fatal("unsigned COIN-M account request was not blocked")
	}
	coinMSetupErr = errors.New("deliberate environment setup failure")
	if _, err := GetDeliveryExchangeInfoContext(context.Background()); err == nil {
		t.Fatal("invalid COIN-M environment allowed market request")
	}
}

func TestCM2GetExchangeInfoUsesSelectedDAPIEndpoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/dapi/v1/exchangeInfo" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"timezone":"UTC","serverTime":1730000000000,"symbols":[
			{"symbol":"BTCUSD_PERP","contractType":"PERPETUAL","contractStatus":"TRADING",
			 "contractSize":100,"marginAsset":"BTC","quoteAsset":"USD","filters":[
			 {"filterType":"PRICE_FILTER","tickSize":"0.1","minPrice":"0.1","maxPrice":"10000000"},
			 {"filterType":"LOT_SIZE","minQty":"1","maxQty":"1000000","stepSize":"1"},
			 {"filterType":"MARKET_LOT_SIZE","minQty":"1","maxQty":"1000000","stepSize":"1"}]}],
			 "rateLimits":[{"rateLimitType":"REQUEST_WEIGHT","interval":"MINUTE","intervalNum":1,"limit":2400}]}`)
	}))
	defer srv.Close()
	guard, err := newCoinMRESTGuard(srv.Client().Transport, srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	origClient, origEnv, origErr := deliveryClient, coinMRuntime, coinMSetupErr
	defer func() { deliveryClient, coinMRuntime, coinMSetupErr = origClient, origEnv, origErr }()
	coinMRuntime = coinMRuntimeConfig{Environment: "testnet", RESTURL: srv.URL, WSURL: delivery.BaseWsTestnetUrl}
	coinMSetupErr = nil
	deliveryClient = delivery.NewClient("", "")
	deliveryClient.BaseURL = srv.URL
	deliveryClient.HTTPClient = &http.Client{Transport: guard}
	info, err := GetDeliveryExchangeInfoContext(context.Background())
	if err != nil || len(info.Symbols) != 1 || info.Symbols[0].ContractSize != 100 {
		t.Fatalf("mock DAPI exchange info query failed: %+v %v", info, err)
	}
}

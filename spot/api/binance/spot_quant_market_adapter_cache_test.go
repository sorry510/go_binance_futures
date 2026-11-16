package binance

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	sdk "github.com/adshao/go-binance/v2"
)

type spotQuantMockTransport struct {
	mu     sync.Mutex
	counts map[string]int
	price  string
}

func (m *spotQuantMockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host != "api.binance.com" && req.URL.Host != "testnet.binance.vision" {
		return nil, fmt.Errorf("unexpected external host %q", req.URL.Host)
	}
	path := req.URL.Path
	m.mu.Lock()
	m.counts[path]++
	m.mu.Unlock()
	quote := ""
	now := time.Now().UnixMilli()
	switch path {
	case "/api/v3/ticker/24hr":
		quote = fmt.Sprintf(`[{"symbol":"BTCUSDT","lastPrice":"%s","openPrice":"95","highPrice":"105","lowPrice":"90","priceChangePercent":"4","closeTime":%d}]`, m.price, now)
	case "/api/v3/klines":
		quote = fmt.Sprintf(`[[%d,"100","105","95","%s","20",%d,"2000",2,"10","1000","0"]]`, now-1000, m.price, now+60000)
	default:
		return nil, fmt.Errorf("unexpected Spot REST path %q", path)
	}
	return &http.Response{StatusCode: 200, Status: "200 OK", Header: http.Header{
		"Content-Type": []string{"application/json"},
	}, Body: io.NopCloser(strings.NewReader(quote)), Request: req}, nil
}

func TestSpotQuantMarketAdapterCacheUsesCorrectEnvironmentAndRequests(t *testing.T) {
	makeAdapter := func(baseURL, price string) (*QuantAdapter, *spotQuantMockTransport) {
		c := sdk.NewClient("test_api_key", "test_api_secret")
		c.SetApiEndpoint(baseURL)
		tr := &spotQuantMockTransport{counts: make(map[string]int), price: price}
		c.HTTPClient = &http.Client{Transport: tr, Timeout: 3 * time.Second}
		a, err := NewSpotQuantAdapterWithClient(c, true)
		if err != nil {
			t.Fatal(err)
		}
		return a, tr
	}
	mainnet, mainTransport := makeAdapter(spotQuantMainnetRESTURL, "101")
	testnet, testTransport := makeAdapter(spotQuantTestnetRESTURL, "202")
	for _, tc := range []struct {
		name string
		a    *QuantAdapter
		tr   *spotQuantMockTransport
		want string
	}{
		{"mainnet", mainnet, mainTransport, "101"},
		{"testnet", testnet, testTransport, "202"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for i := 0; i < 3; i++ {
				snapshot, err := tc.a.MarketSnapshot(context.Background(), "BTCUSDT")
				if err != nil || snapshot.Price != tc.want {
					t.Fatalf("market snapshot %d: %+v %v", i, snapshot, err)
				}
				rows, err := tc.a.MarketKlines(context.Background(), "BTCUSDT", "1m", 1)
				if err != nil || len(rows) != 1 || rows[0].Close != tc.want {
					t.Fatalf("market Kline %d: %+v %v", i, rows, err)
				}
			}
			tc.tr.mu.Lock()
			dayCalls := tc.tr.counts["/api/v3/ticker/24hr"]
			klineCalls := tc.tr.counts["/api/v3/klines"]
			tc.tr.mu.Unlock()
			if dayCalls != 1 || klineCalls != 1 {
				t.Fatalf("market REST not coalesced in %s: 24h=%d klines=%d", tc.name, dayCalls, klineCalls)
			}
		})
	}
	test, err := testnet.MarketSnapshot(context.Background(), "BTCUSDT")
	if err != nil || test.Price == "101" {
		t.Fatalf("testnet reused mainnet cache: %+v %v", test, err)
	}
}

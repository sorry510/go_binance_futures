package binanceapiusage

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// The existing shared REST limiter already supports the delivery product.
// CM-2 regression: the new client must pass selected environment to that
// transport so a Demo 418 never poisons the Testnet COIN-M weight budget.
func TestCM2CoinMWeightTransportEnvironmentAndRateLimitIsolation(t *testing.T) {
	collector := &Collector{limits: map[string]ExchangeLimitState{}}
	budget := newBudgetCoordinator(collector)
	transportCalls := 0
	base := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		transportCalls++
		status := http.StatusTooManyRequests
		if strings.Contains(req.URL.Host, "demo") {
			status = http.StatusTeapot
		}
		return &http.Response{
			StatusCode: status,
			Header: http.Header{
				"X-Mbx-Used-Weight-1m": []string{"15"},
				"Retry-After":          []string{"1"},
			},
			Body:    io.NopCloser(strings.NewReader("{}")),
			Request: req,
		}, nil
	})}
	for _, tt := range []struct {
		environment, endpoint string
		status                int
	}{
		{"testnet", "https://testnet.binancefuture.com/dapi/v1/exchangeInfo", 429},
		{"demo", "https://demo-dapi.binance.com/dapi/v1/exchangeInfo", 418},
	} {
		client := WrapClient(base, TransportConfig{
			Product: "delivery", Environment: tt.environment, Source: "coin_m_market",
			Collector: collector, Budget: budget,
		})
		response, err := client.Get(tt.endpoint)
		if err != nil {
			t.Fatalf("%s DAPI request: %v", tt.environment, err)
		}
		response.Body.Close()
		if response.StatusCode != tt.status {
			t.Fatalf("%s status=%d want=%d", tt.environment, response.StatusCode, tt.status)
		}
	}
	if transportCalls != 2 {
		t.Fatalf("DAPI requests unexpectedly blocked: %d", transportCalls)
	}
	snapshot := collector.Snapshot(time.Now())
	var test429, demo418 bool
	for _, event := range snapshot.RecentRateLimits {
		if event.Product != "delivery" || event.Source != "coin_m_market" {
			continue
		}
		if event.Environment == "testnet" && event.StatusCode == 429 {
			test429 = true
		}
		if event.Environment == "demo" && event.StatusCode == 418 {
			demo418 = true
		}
	}
	if !test429 || !demo418 {
		t.Fatalf("environment/429/418 attribution missing: %+v", snapshot.RecentRateLimits)
	}
	var seenTest, seenDemo bool
	for _, limit := range snapshot.ExchangeLimits {
		if limit.Product != "delivery" {
			continue
		}
		if limit.Environment == "testnet" {
			seenTest = true
		}
		if limit.Environment == "demo" {
			seenDemo = true
		}
	}
	if !seenTest || !seenDemo {
		t.Fatalf("DAPI rate limit scopes mixed between testnet/demo: %+v", snapshot.ExchangeLimits)
	}
}

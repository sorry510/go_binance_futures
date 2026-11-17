package binanceapiusage

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestLead04SAPIWeightIsSeparateFromFuturesAndSignedURLIsNotRecorded(t *testing.T) {
	collector := &Collector{limits: map[string]ExchangeLimitState{}}
	budget := newBudgetCoordinator(collector)
	base := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		h := make(http.Header)
		switch req.URL.Path {
		case "/sapi/v1/copyTrading/futures/userStatus":
			h.Set("X-SAPI-USED-IP-WEIGHT-1M", "350")
			h.Set("X-MBX-USED-WEIGHT-1M", "9999") // MUST NOT poison Futures
		case "/fapi/v2/positionRisk":
			h.Set("X-MBX-USED-WEIGHT-1M", "875")
			h.Set("X-SAPI-USED-IP-WEIGHT-1M", "9999")
		default:
			t.Errorf("unexpected endpoint: %q", req.URL.Path)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: h,
			Body: io.NopCloser(strings.NewReader("{}")), Request: req}, nil
	})}
	client := WrapClient(base, TransportConfig{
		Product: "futures", Environment: "mainnet", Source: "lead_trading",
		Collector: collector, Budget: budget,
	})
	for _, reqURL := range []string{
		"https://api.binance.com/sapi/v1/copyTrading/futures/userStatus?timestamp=123&signature=SECRET",
		"https://fapi.binance.com/fapi/v2/positionRisk?timestamp=123&signature=SECRET",
	} {
		req, err := http.NewRequestWithContext(WithSource(context.Background(), "lead_trading_sapi"),
			http.MethodGet, reqURL, nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
	}
	sapi := budget.Snapshot("sapi", "mainnet")
	futuresWeight := budget.Snapshot("futures", "mainnet")
	if sapi.UsedWeight1m != 350 || sapi.WeightLimit1m != 12000 ||
		futuresWeight.UsedWeight1m != 875 || futuresWeight.WeightLimit1m != 2400 {
		t.Fatalf("SAPI/Futures weights mixed: sapi=%+v futures=%+v", sapi, futuresWeight)
	}
	snapshot := collector.Snapshot(time.Now())
	if len(snapshot.ExchangeLimits) != 2 || len(snapshot.TopEndpointsByCount) != 2 {
		t.Fatalf("expected independent product accounting: %+v", snapshot)
	}
	for _, item := range snapshot.TopEndpointsByCount {
		if strings.Contains(item.Path+item.LastError, "SECRET") ||
			strings.Contains(item.Path, "timestamp") || strings.Contains(item.Path, "signature") {
			t.Fatal("SAPI signed URL leaked into metrics")
		}
	}
}

func TestLead04SAPI429DoesNotThrottleFuturesButThrottlesSAPI(t *testing.T) {
	collector := &Collector{limits: map[string]ExchangeLimitState{}}
	budget := newBudgetCoordinator(collector)
	var sentFutures, sentSAPI int
	base := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		h := make(http.Header)
		if strings.HasPrefix(req.URL.Path, "/sapi/") {
			sentSAPI++
			h.Set("Retry-After", "2")
			return &http.Response{StatusCode: 429, Header: h, Body: io.NopCloser(strings.NewReader("{}"))}, nil
		}
		sentFutures++
		return &http.Response{StatusCode: 200, Header: h, Body: io.NopCloser(strings.NewReader("{}"))}, nil
	})}
	client := WrapClient(base, TransportConfig{
		Product: "futures", Environment: "mainnet", Source: "lead_trading",
		Collector: collector, Budget: budget,
	})
	request := func(path string) error {
		req, err := http.NewRequest(http.MethodGet, "https://api.binance.com"+path+"?signature=secret", nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if resp != nil {
			_ = resp.Body.Close()
		}
		return err
	}
	if err := request("/sapi/v1/copyTrading/futures/userStatus"); err != nil {
		t.Fatal(err)
	}
	if err := request("/fapi/v2/positionRisk"); err != nil {
		t.Fatal(err)
	}
	if err := request("/sapi/v1/copyTrading/futures/leadSymbol"); err == nil {
		t.Fatal("SAPI 429 did not defer the next SAPI request")
	}
	if sentFutures != 1 || sentSAPI != 1 {
		t.Fatalf("a SAPI limit blocked Futures or retried SAPI: futures=%d sapi=%d", sentFutures, sentSAPI)
	}
}

// Both API keys may have separate order limits, but their /fapi/ IP weight
// budget must be coordinated. Lead must not bypass a Main near-limit snapshot.
func TestLead04MainAndLeadFuturesShareGlobalWeightBudget(t *testing.T) {
	collector := &Collector{limits: map[string]ExchangeLimitState{}}
	collector.SetWeightLimit("futures", "mainnet", 10, "test_reference")
	budget := newBudgetCoordinator(collector)
	var sent int
	base := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		sent++
		h := make(http.Header)
		h.Set("X-MBX-USED-WEIGHT-1M", "9")
		return &http.Response{StatusCode: 200, Header: h,
			Body: io.NopCloser(strings.NewReader("{}")), Request: req}, nil
	})}
	mainClient := WrapClient(base, TransportConfig{Product: "futures", Environment: "mainnet",
		Source: "start_trade", Collector: collector, Budget: budget})
	leadClient := WrapClient(base, TransportConfig{Product: "futures", Environment: "mainnet",
		Source: "lead_trading", Collector: collector, Budget: budget})
	u := "https://fapi.binance.com/fapi/v1/klines?symbol=BTCUSDT&limit=1000"
	resp, err := mainClient.Get(u)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	_, err = leadClient.Get(u)
	if !errors.Is(err, ErrBudgetDeferred) || sent != 1 {
		t.Fatalf("Lead bypassed Main IP budget: %v, http_sent=%d", err, sent)
	}
}

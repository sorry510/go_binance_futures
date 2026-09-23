package binanceapiusage

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestTransportRecordsHeadersWithoutSignedQueryLeak(t *testing.T) {
	collector := &Collector{limits: map[string]ExchangeLimitState{}}
	base := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		header := make(http.Header)
		header.Set("X-MBX-USED-WEIGHT-1M", "1234")
		header.Set("X-MBX-ORDER-COUNT-10S", "12")
		header.Set("X-MBX-ORDER-COUNT-1M", "34")
		header.Set("Retry-After", "2")
		return &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Header:     header,
			Body:       io.NopCloser(strings.NewReader("{}")),
			Request:    req,
		}, nil
	})}
	client := WrapClient(base, TransportConfig{
		Product: "futures", Environment: "testnet", Source: "go_binance", Collector: collector,
	})

	req, err := http.NewRequestWithContext(
		WithSource(context.Background(), "start_trade"),
		http.MethodPost,
		"https://testnet.binancefuture.com/fapi/v1/order?symbol=BTCUSDT&timestamp=123&signature=SECRET&apiKey=KEY",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()

	snapshot := collector.Snapshot(time.Now())
	if snapshot.Window1m.RequestCount != 1 || snapshot.Window1m.Count429 != 1 {
		t.Fatalf("window1m=%+v", snapshot.Window1m)
	}
	if len(snapshot.ExchangeLimits) != 1 {
		t.Fatalf("exchange limits=%+v", snapshot.ExchangeLimits)
	}
	limit := snapshot.ExchangeLimits[0]
	if limit.UsedWeight1m != 1234 || limit.OrderCount10s != 12 || limit.OrderCount1m != 34 || limit.RetryAfter != "2" {
		t.Fatalf("limit state=%+v", limit)
	}
	if len(snapshot.TopEndpointsByCount) != 1 {
		t.Fatalf("top endpoints=%+v", snapshot.TopEndpointsByCount)
	}
	endpoint := snapshot.TopEndpointsByCount[0]
	if endpoint.Path != "/fapi/v1/order" || endpoint.Source != "start_trade" {
		t.Fatalf("endpoint=%+v", endpoint)
	}
	serialized := endpoint.Path + endpoint.LastError
	for _, secret := range []string{"signature", "SECRET", "apiKey", "KEY", "timestamp=123", "BTCUSDT"} {
		if strings.Contains(serialized, secret) {
			t.Fatalf("sensitive/query value leaked into observability: %q", serialized)
		}
	}
	if len(snapshot.RecentRateLimits) != 1 || snapshot.RecentRateLimits[0].Path != "/fapi/v1/order" {
		t.Fatalf("rate limits=%+v", snapshot.RecentRateLimits)
	}
}

func TestTransportErrorNeverStoresRawErrorURL(t *testing.T) {
	collector := &Collector{limits: map[string]ExchangeLimitState{}}
	client := WrapClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return nil, errors.New("dial failed https://example/fapi/v1/order?signature=VERY_SECRET")
	})}, TransportConfig{Product: "futures", Environment: "mainnet", Collector: collector})

	req, _ := http.NewRequest(http.MethodGet, "https://example/fapi/v1/order?signature=SECRET", nil)
	_, _ = client.Do(req)

	snapshot := collector.Snapshot(time.Now())
	if len(snapshot.TopEndpointsByCount) != 1 {
		t.Fatalf("endpoints=%+v", snapshot.TopEndpointsByCount)
	}
	if snapshot.TopEndpointsByCount[0].LastError != "transport_error" {
		t.Fatalf("raw transport error retained: %+v", snapshot.TopEndpointsByCount[0])
	}
}

func TestCollectorRollingWindowsAndTopEndpoints(t *testing.T) {
	collector := &Collector{limits: map[string]ExchangeLimitState{}}
	now := time.Unix(1_800_000_000, 0)
	record := func(age time.Duration, path string, weight, latency int64) {
		collector.Record(RequestEvent{
			At: now.Add(-age).UnixMilli(), Product: "futures", Environment: "mainnet",
			Source: "test", Method: http.MethodGet, Path: path,
			StatusCode: 200, EstimatedWeight: weight, LatencyMs: latency,
		}, nil, "")
	}
	record(5*time.Second, "/fapi/v1/openOrders", 40, 40)
	record(30*time.Second, "/fapi/v1/positionRisk", 5, 20)
	record(2*time.Minute, "/fapi/v1/klines", 5, 10)

	snapshot := collector.Snapshot(now)
	if snapshot.Window10s.RequestCount != 1 {
		t.Fatalf("10s=%+v", snapshot.Window10s)
	}
	if snapshot.Window1m.RequestCount != 2 || snapshot.Window1m.EstimatedWeight != 45 {
		t.Fatalf("1m=%+v", snapshot.Window1m)
	}
	if snapshot.Window5m.RequestCount != 3 || snapshot.Window5m.EstimatedWeight != 50 {
		t.Fatalf("5m=%+v", snapshot.Window5m)
	}
	if len(snapshot.TopEndpointsByWeight) == 0 || snapshot.TopEndpointsByWeight[0].Path != "/fapi/v1/openOrders" {
		t.Fatalf("top weight=%+v", snapshot.TopEndpointsByWeight)
	}
}

func TestExchangeInfoWeightLimitOverridesReferenceAndPersists(t *testing.T) {
	collector := &Collector{limits: map[string]ExchangeLimitState{}}
	collector.SetWeightLimit("futures", "mainnet", 3600, "exchange_info")
	collector.Record(RequestEvent{
		At: time.Now().UnixMilli(), Product: "futures", Environment: "mainnet",
		Source: "test", RequestType: "read", Method: http.MethodGet,
		Path: "/fapi/v1/time", StatusCode: 200, EstimatedWeight: 1,
	}, map[string]string{"used_weight_1m": "1800"}, "")

	snapshot := collector.Snapshot(time.Now())
	if len(snapshot.ExchangeLimits) != 1 {
		t.Fatalf("limits=%+v", snapshot.ExchangeLimits)
	}
	got := snapshot.ExchangeLimits[0]
	if got.WeightLimit1m != 3600 || got.LimitSource != "exchange_info" {
		t.Fatalf("dynamic limit lost: %+v", got)
	}
	if got.WeightPercent1m != 50 {
		t.Fatalf("weight percent=%v want=50", got.WeightPercent1m)
	}
}

func TestRequestTypeDistinguishesTradeFromRead(t *testing.T) {
	cases := []struct {
		product string
		method  string
		path    string
		want    string
	}{
		{product: "futures", method: http.MethodPost, path: "/fapi/v1/order", want: "trade"},
		{product: "futures", method: http.MethodDelete, path: "/fapi/v1/algoOrder", want: "trade"},
		{product: "futures", method: http.MethodPost, path: "/fapi/v1/leverage", want: "trade"},
		{product: "futures", method: http.MethodGet, path: "/fapi/v2/positionRisk", want: "read"},
		{product: "futures", method: http.MethodPost, path: "/fapi/v1/listenKey", want: "read"},
		{product: "spot", method: http.MethodPost, path: "/api/v3/order", want: "trade"},
	}
	for _, tc := range cases {
		if got := RequestType(tc.product, tc.method, tc.path); got != tc.want {
			t.Fatalf("%s %s %s type=%q want=%q", tc.product, tc.method, tc.path, got, tc.want)
		}
	}
}

func TestEstimateWeightKnownFuturesEndpoints(t *testing.T) {
	query := url.Values{}
	if got := EstimateWeight("futures", http.MethodGet, "/fapi/v1/openOrders", query); got != 40 {
		t.Fatalf("all open orders weight=%d want=40", got)
	}
	query.Set("symbol", "BTCUSDT")
	if got := EstimateWeight("futures", http.MethodGet, "/fapi/v1/openOrders", query); got != 1 {
		t.Fatalf("symbol open orders weight=%d want=1", got)
	}
	query = url.Values{"limit": []string{"1000"}}
	if got := EstimateWeight("futures", http.MethodGet, "/fapi/v1/klines", query); got != 5 {
		t.Fatalf("klines weight=%d want=5", got)
	}
}

func TestCollectorRingBufferReportsTruncationWithoutPerRecordCopy(t *testing.T) {
	collector := &Collector{limits: map[string]ExchangeLimitState{}}
	now := time.Now()
	for i := 0; i < maxEvents+7; i++ {
		collector.Record(RequestEvent{
			At:              now.Add(time.Duration(i) * time.Microsecond).UnixMilli(),
			Product:         "futures",
			Environment:     "mainnet",
			Source:          "load_test",
			RequestType:     "read",
			Method:          http.MethodGet,
			Path:            "/fapi/v1/time",
			StatusCode:      200,
			EstimatedWeight: 1,
		}, nil, "")
	}

	snapshot := collector.Snapshot(now.Add(time.Second))
	if snapshot.RetainedEvents != maxEvents {
		t.Fatalf("retained_events=%d want=%d", snapshot.RetainedEvents, maxEvents)
	}
	if snapshot.DroppedEvents != 7 {
		t.Fatalf("dropped_events=%d want=7", snapshot.DroppedEvents)
	}
	if !snapshot.Truncated || snapshot.LastDroppedAt <= 0 {
		t.Fatalf("expected active-window truncation: %+v", snapshot)
	}
	if snapshot.Window5m.RequestCount != maxEvents {
		t.Fatalf("window5m request_count=%d want=%d", snapshot.Window5m.RequestCount, maxEvents)
	}
}

func TestCollectorOverwritingExpiredEventsDoesNotMarkWindowTruncated(t *testing.T) {
	collector := &Collector{limits: map[string]ExchangeLimitState{}}
	now := time.Now()
	old := now.Add(-10 * time.Minute)
	for i := 0; i < maxEvents; i++ {
		collector.Record(RequestEvent{
			At:              old.Add(time.Duration(i) * time.Microsecond).UnixMilli(),
			Product:         "futures",
			Environment:     "mainnet",
			Source:          "old",
			RequestType:     "read",
			Method:          http.MethodGet,
			Path:            "/fapi/v1/time",
			StatusCode:      200,
			EstimatedWeight: 1,
		}, nil, "")
	}
	collector.Record(RequestEvent{
		At:              now.UnixMilli(),
		Product:         "futures",
		Environment:     "mainnet",
		Source:          "current",
		RequestType:     "read",
		Method:          http.MethodGet,
		Path:            "/fapi/v1/time",
		StatusCode:      200,
		EstimatedWeight: 1,
	}, nil, "")

	snapshot := collector.Snapshot(now)
	if snapshot.Truncated || snapshot.DroppedEvents != 0 {
		t.Fatalf("expired overwrite must not mark the active window truncated: %+v", snapshot)
	}
	if snapshot.Window5m.RequestCount != 1 {
		t.Fatalf("window5m request_count=%d want=1", snapshot.Window5m.RequestCount)
	}
}

package binanceapiusage

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestBudgetPriorityKeepsUserDataListenKeyAtP1(t *testing.T) {
	if got := classifyBudgetPriority("go_binance", "read", http.MethodPut, "/fapi/v1/listenKey"); got != PriorityP1 {
		t.Fatalf("listenKey priority=%s want=%s", got, PriorityP1)
	}
}

func TestBudgetCriticalDefersP2ButAllowsP1(t *testing.T) {
	collector := &Collector{limits: map[string]ExchangeLimitState{}}
	collector.SetWeightLimit("futures", "mainnet", 100, "test")
	collector.Record(RequestEvent{At: time.Now().UnixMilli(), Product: "futures", Environment: "mainnet", Source: "test", RequestType: "read", Method: http.MethodGet, Path: "/fapi/v1/time", StatusCode: 200}, map[string]string{"used_weight_1m": "90"}, "")
	budget := newBudgetCoordinator(collector)

	if _, err := budget.Reserve(context.Background(), "futures", "mainnet", "market_intelligence", "read", http.MethodGet, "/fapi/v1/depth", 1); !errors.Is(err, ErrBudgetDeferred) {
		t.Fatalf("P2 critical request err=%v want ErrBudgetDeferred", err)
	}
	r, err := budget.Reserve(context.Background(), "futures", "mainnet", "start_trade", "read", http.MethodGet, "/fapi/v2/positionRisk", 5)
	if err != nil {
		t.Fatalf("P1 critical request should be allowed: %v", err)
	}
	r.Release()
}

func TestBudgetPendingWeightPreventsConcurrentBurst(t *testing.T) {
	collector := &Collector{limits: map[string]ExchangeLimitState{}}
	collector.SetWeightLimit("futures", "mainnet", 100, "test")
	collector.Record(RequestEvent{At: time.Now().UnixMilli(), Product: "futures", Environment: "mainnet", Source: "test", RequestType: "read", Method: http.MethodGet, Path: "/fapi/v1/time", StatusCode: 200}, map[string]string{"used_weight_1m": "60"}, "")
	budget := newBudgetCoordinator(collector)

	r1, err := budget.Reserve(context.Background(), "futures", "mainnet", "market_intelligence", "read", http.MethodGet, "/fapi/v1/depth", 20)
	if err != nil {
		t.Fatal(err)
	}
	defer r1.Release()
	if _, err := budget.Reserve(context.Background(), "futures", "mainnet", "market_intelligence", "read", http.MethodGet, "/fapi/v1/klines", 10); !errors.Is(err, ErrBudgetDeferred) {
		t.Fatalf("second concurrent reservation err=%v want deferred", err)
	}
}

func TestBudgetExchangeThrottleBlocksTradeWithoutRetry(t *testing.T) {
	collector := &Collector{limits: map[string]ExchangeLimitState{}}
	now := time.Now()
	collector.SetWeightLimit("futures", "mainnet", 2400, "test")
	collector.Record(RequestEvent{At: now.UnixMilli(), Product: "futures", Environment: "mainnet", Source: "start_trade", RequestType: "trade", Method: http.MethodPost, Path: "/fapi/v1/order", StatusCode: 429, Is429: true}, map[string]string{"used_weight_1m": "2300"}, "2")
	budget := newBudgetCoordinator(collector)
	if _, err := budget.Reserve(context.Background(), "futures", "mainnet", "start_trade", "trade", http.MethodPost, "/fapi/v1/order", 1); !errors.Is(err, ErrBudgetDeferred) {
		t.Fatalf("trade during Retry-After err=%v want deferred", err)
	}
}

func TestBudgetOrderCountLimitProtectsMutation(t *testing.T) {
	collector := &Collector{limits: map[string]ExchangeLimitState{}}
	collector.SetWeightLimit("futures", "mainnet", 2400, "test")
	collector.SetOrderLimits("futures", "mainnet", 10, 100)
	collector.Record(RequestEvent{At: time.Now().UnixMilli(), Product: "futures", Environment: "mainnet", Source: "start_trade", RequestType: "read", Method: http.MethodGet, Path: "/fapi/v1/time", StatusCode: 200}, map[string]string{"order_count_10s": "10", "order_count_1m": "20"}, "")
	budget := newBudgetCoordinator(collector)
	if _, err := budget.Reserve(context.Background(), "futures", "mainnet", "start_trade", "trade", http.MethodPost, "/fapi/v1/order", 1); !errors.Is(err, ErrBudgetDeferred) {
		t.Fatalf("order mutation at exhausted 10s budget err=%v want deferred", err)
	}
}

func TestTransportBudgetStopsDeferredRequestBeforeBaseRoundTrip(t *testing.T) {
	collector := &Collector{limits: map[string]ExchangeLimitState{}}
	collector.SetWeightLimit("futures", "mainnet", 100, "test")
	collector.Record(RequestEvent{At: time.Now().UnixMilli(), Product: "futures", Environment: "mainnet", Source: "test", RequestType: "read", Method: http.MethodGet, Path: "/fapi/v1/time", StatusCode: 200}, map[string]string{"used_weight_1m": "90"}, "")
	budget := newBudgetCoordinator(collector)
	called := false
	client := WrapClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		called = true
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: http.NoBody, Request: req}, nil
	})}, TransportConfig{Product: "futures", Environment: "mainnet", Source: "market_intelligence", Collector: collector, Budget: budget})
	req, _ := http.NewRequest(http.MethodGet, "https://example/fapi/v1/depth?limit=100", nil)
	_, err := client.Do(req)
	if !errors.Is(err, ErrBudgetDeferred) {
		t.Fatalf("err=%v want deferred", err)
	}
	if called {
		t.Fatal("base transport must not be called when budget defers")
	}
}

func TestBudgetQueuedLowPriorityRechecksAfterWaiting(t *testing.T) {
	collector := &Collector{limits: map[string]ExchangeLimitState{}}
	collector.SetWeightLimit("futures", "mainnet", 100, "test")
	collector.Record(RequestEvent{At: time.Now().UnixMilli(), Product: "futures", Environment: "mainnet", Source: "test", RequestType: "read", Method: http.MethodGet, Path: "/fapi/v1/time", StatusCode: 200}, map[string]string{"used_weight_1m": "75"}, "")
	budget := newBudgetCoordinator(collector)

	first, err := budget.Reserve(context.Background(), "futures", "mainnet", "market_intelligence", "read", http.MethodGet, "/fapi/v1/depth", 1)
	if err != nil {
		t.Fatal(err)
	}

	result := make(chan error, 1)
	go func() {
		_, reserveErr := budget.Reserve(context.Background(), "futures", "mainnet", "historical_market", "read", http.MethodGet, "/fapi/v1/klines", 1)
		result <- reserveErr
	}()

	time.Sleep(20 * time.Millisecond)
	collector.Record(RequestEvent{At: time.Now().UnixMilli(), Product: "futures", Environment: "mainnet", Source: "test", RequestType: "read", Method: http.MethodGet, Path: "/fapi/v1/time", StatusCode: 200}, map[string]string{"used_weight_1m": "90"}, "")
	first.Release()

	select {
	case reserveErr := <-result:
		if !errors.Is(reserveErr, ErrBudgetDeferred) {
			t.Fatalf("queued low-priority request err=%v want deferred after critical recheck", reserveErr)
		}
	case <-time.After(time.Second):
		t.Fatal("queued low-priority reservation did not complete")
	}
}

func TestBudgetOrderCountDoesNotBlockLeverageOrMarginConfig(t *testing.T) {
	collector := &Collector{limits: map[string]ExchangeLimitState{}}
	collector.SetWeightLimit("futures", "mainnet", 2400, "test")
	collector.SetOrderLimits("futures", "mainnet", 10, 100)
	collector.Record(RequestEvent{At: time.Now().UnixMilli(), Product: "futures", Environment: "mainnet", Source: "test", RequestType: "read", Method: http.MethodGet, Path: "/fapi/v1/time", StatusCode: 200}, map[string]string{"order_count_10s": "10", "order_count_1m": "100"}, "")
	budget := newBudgetCoordinator(collector)

	for _, path := range []string{"/fapi/v1/leverage", "/fapi/v1/marginType"} {
		reservation, err := budget.Reserve(context.Background(), "futures", "mainnet", "start_trade", "trade", http.MethodPost, path, 1)
		if err != nil {
			t.Fatalf("%s must not consume ORDERS budget: %v", path, err)
		}
		reservation.Release()
	}
}

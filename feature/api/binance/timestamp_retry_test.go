package binance

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/adshao/go-binance/v2/futures"
)

func TestGetOpenOrderResyncsTimestampAndRetriesOnce(t *testing.T) {
	var openOrderCalls atomic.Int32
	var timeCalls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/fapi/v1/openOrders":
			if got := r.URL.Query().Get("recvWindow"); got != "10000" {
				t.Errorf("recvWindow = %q, want 10000", got)
			}
			if openOrderCalls.Add(1) == 1 {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"code":-1021,"msg":"Timestamp for this request is outside of the recvWindow."}`))
				return
			}
			_, _ = w.Write([]byte(`[]`))
		case "/fapi/v1/time":
			timeCalls.Add(1)
			_, _ = fmt.Fprintf(w, `{"serverTime":%d}`, time.Now().Add(-2500*time.Millisecond).UnixMilli())
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := futures.NewClient("test-key", "test-secret")
	client.SetApiEndpoint(server.URL)
	client.HTTPClient = server.Client()

	oldClient := futuresClient
	futuresClient = client
	defer func() { futuresClient = oldClient }()

	orders, err := GetOpenOrder()
	if err != nil {
		t.Fatalf("GetOpenOrder() error = %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("len(orders) = %d, want 0", len(orders))
	}
	if got := openOrderCalls.Load(); got != 2 {
		t.Fatalf("open order calls = %d, want 2", got)
	}
	if got := timeCalls.Load(); got != 1 {
		t.Fatalf("server time calls = %d, want 1", got)
	}
	if client.TimeOffset < 1500 || client.TimeOffset > 3500 {
		t.Fatalf("TimeOffset = %dms, want approximately 2500ms", client.TimeOffset)
	}
}

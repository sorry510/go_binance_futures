package binance

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/adshao/go-binance/v2/futures"
)

func installTradeConfigTestClient(t *testing.T, handler http.Handler) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client := futures.NewClient("test-key", "test-secret")
	client.SetApiEndpoint(server.URL)
	client.HTTPClient = server.Client()

	oldClient := futuresClient
	futuresClient = client
	t.Cleanup(func() {
		futuresClient = oldClient
		resetRuntimeTradeConfigCacheForTest()
	})
	resetRuntimeTradeConfigCacheForTest()
}

func TestEnsureTradeConfigCachesIdenticalConfigAndRefreshesChangedConfig(t *testing.T) {
	var marginCalls atomic.Int32
	var leverageCalls atomic.Int32
	installTradeConfigTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/fapi/v1/marginType":
			marginCalls.Add(1)
			_, _ = w.Write([]byte(`{"code":200,"msg":"success"}`))
		case "/fapi/v1/leverage":
			leverageCalls.Add(1)
			_, _ = w.Write([]byte(`{"leverage":4,"maxNotionalValue":"1000000","symbol":"BTCUSDT"}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))

	ctx := context.Background()
	if err := EnsureTradeConfigContext(ctx, "btcusdt", futures.MarginTypeCrossed, 4); err != nil {
		t.Fatalf("first EnsureTradeConfigContext() error = %v", err)
	}
	if err := EnsureTradeConfigContext(ctx, "BTCUSDT", futures.MarginTypeCrossed, 4); err != nil {
		t.Fatalf("second EnsureTradeConfigContext() error = %v", err)
	}
	if got := marginCalls.Load(); got != 1 {
		t.Fatalf("margin calls after identical config = %d, want 1", got)
	}
	if got := leverageCalls.Load(); got != 1 {
		t.Fatalf("leverage calls after identical config = %d, want 1", got)
	}

	if err := EnsureTradeConfigContext(ctx, "BTCUSDT", futures.MarginTypeCrossed, 5); err != nil {
		t.Fatalf("changed EnsureTradeConfigContext() error = %v", err)
	}
	if got := marginCalls.Load(); got != 2 {
		t.Fatalf("margin calls after changed config = %d, want 2", got)
	}
	if got := leverageCalls.Load(); got != 2 {
		t.Fatalf("leverage calls after changed config = %d, want 2", got)
	}
}

func TestEnsureTradeConfigDoesNotCacheFailure(t *testing.T) {
	var marginCalls atomic.Int32
	var leverageCalls atomic.Int32
	installTradeConfigTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/fapi/v1/marginType":
			marginCalls.Add(1)
			_, _ = w.Write([]byte(`{"code":200,"msg":"success"}`))
		case "/fapi/v1/leverage":
			if leverageCalls.Add(1) == 1 {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"code":-4028,"msg":"invalid leverage"}`))
				return
			}
			_, _ = w.Write([]byte(`{"leverage":4,"maxNotionalValue":"1000000","symbol":"BTCUSDT"}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))

	ctx := context.Background()
	if err := EnsureTradeConfigContext(ctx, "BTCUSDT", futures.MarginTypeCrossed, 4); err == nil {
		t.Fatal("first EnsureTradeConfigContext() error = nil, want failure")
	}
	if err := EnsureTradeConfigContext(ctx, "BTCUSDT", futures.MarginTypeCrossed, 4); err != nil {
		t.Fatalf("second EnsureTradeConfigContext() error = %v", err)
	}
	if got := marginCalls.Load(); got != 2 {
		t.Fatalf("margin calls after retry = %d, want 2", got)
	}
	if got := leverageCalls.Load(); got != 2 {
		t.Fatalf("leverage calls after retry = %d, want 2", got)
	}
}

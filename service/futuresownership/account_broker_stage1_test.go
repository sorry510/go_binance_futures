package futuresownership

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/adshao/go-binance/v2/futures"
	"go_binance_futures/feature/api/binance"
)

func TestStage1BrokerAlwaysRoutesToItsBoundAccount(t *testing.T) {
	var mu sync.Mutex
	calls := make(map[string]int)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("X-MBX-APIKEY")
		mu.Lock()
		calls[key+" "+r.Method+" "+r.URL.Path]++
		mu.Unlock()
		if key != "main-key" && key != "lead-key" {
			t.Errorf("unexpected key %s", key)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if !strings.HasPrefix(r.URL.Path, "/fapi/v1/order") {
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		switch r.Method {
		case http.MethodPost:
			_, _ = w.Write([]byte(`{"orderId":987,"clientOrderId":"test_stage1","symbol":"BTCUSDT","status":"NEW","executedQty":"0","avgPrice":"0"}`))
		case http.MethodGet:
			_, _ = w.Write([]byte(`{"orderId":987,"clientOrderId":"test_stage1","symbol":"BTCUSDT","status":"FILLED","executedQty":"0.01","avgPrice":"30000"}`))
		case http.MethodDelete:
			_, _ = w.Write([]byte(`{"orderId":987,"clientOrderId":"test_stage1","symbol":"BTCUSDT","status":"CANCELED"}`))
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	}))
	defer server.Close()
	for _, kind := range []binance.AccountID{binance.MainAccountID, binance.LeadAccountID} {
		raw := futures.NewClient(string(kind)+"-key", string(kind)+"-secret")
		raw.SetApiEndpoint(server.URL)
		raw.HTTPClient = server.Client()
		account, err := binance.NewAccountClient(kind, raw)
		if err != nil {
			t.Fatal(err)
		}
		broker := BinanceOrderBroker{Account: account}
		_, err = broker.Submit(context.Background(), OrderRequest{Symbol: "BTCUSDT", Side: "BUY", PositionSide: "LONG", OrderType: "LIMIT", Quantity: 0.01, Price: 30000}, "test_stage1")
		if err != nil {
			t.Fatalf("submit %s: %v", kind, err)
		}
		looked, err := broker.Lookup(context.Background(), "BTCUSDT", "test_stage1", "LIMIT")
		if err != nil || looked.FilledQty != 0.01 {
			t.Fatalf("lookup %s: %+v %v", kind, looked, err)
		}
		if err := broker.Cancel(context.Background(), "BTCUSDT", 987, "LIMIT"); err != nil {
			t.Fatalf("cancel %s: %v", kind, err)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	for _, kind := range []string{"main", "lead"} {
		for _, method := range []string{"POST", "GET", "DELETE"} {
			k := kind + "-key " + method + " /fapi/v1/order"
			if calls[k] != 1 {
				t.Errorf("broker route %s called %d times", k, calls[k])
			}
		}
	}
}

func TestStage1AlgoLookupUsesBoundAccountForTriggeredOrder(t *testing.T) {
	for _, id := range []binance.AccountID{binance.MainAccountID, binance.LeadAccountID} {
		id := id
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-MBX-APIKEY") != string(id)+"-key" {
				t.Errorf("wrong account for algo lookup")
			}
			switch r.URL.Path {
			case "/fapi/v1/algoOrder":
				_, _ = w.Write([]byte(`{"algoId":31,"clientAlgoId":"test_algo","algoStatus":"FINISHED","actualOrderId":"42","symbol":"BTCUSDT"}`))
			case "/fapi/v1/order":
				_, _ = w.Write([]byte(`{"orderId":42,"clientOrderId":"test_actual","status":"FILLED","executedQty":"0.05","avgPrice":"35000","symbol":"BTCUSDT"}`))
			default:
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
		}))
		sdk := futures.NewClient(string(id)+"-key", string(id)+"-secret")
		sdk.SetApiEndpoint(server.URL)
		sdk.HTTPClient = server.Client()
		a, err := binance.NewAccountClient(id, sdk)
		if err != nil {
			t.Fatal(err)
		}
		order, err := (BinanceOrderBroker{Account: a}).Lookup(context.Background(), "BTCUSDT", "test_algo", "STOP_MARKET")
		if err != nil || order.ExchangeOrderID != "31" || order.FilledQty != 0.05 || order.Status != "FILLED" {
			t.Errorf("account %s algo lookup=%+v err=%v", id, order, err)
		}
		server.Close()
	}
}

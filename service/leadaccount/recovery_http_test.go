package leadaccount

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/adshao/go-binance/v2/futures"
	binance "go_binance_futures/feature/api/binance"
)

func TestStage44AccountBoundRecoveryHTTPNoMutations(t *testing.T) {
	cases := []struct {
		name, kind, algoStatus, actualID, realStatus, realClient, realSymbol, realQty string
		wantCalls                                                                     int
		terminal, uncertain                                                           bool
	}{
		{name: "ordinary_cancel_partial_fill", kind: "LIMIT", realStatus: "CANCELED", realClient: "lead_test", realSymbol: "BTCUSDT", realQty: "0.25", wantCalls: 1, terminal: true},
		{name: "ordinary_reject", kind: "MARKET", realStatus: "REJECTED", realClient: "lead_test", realSymbol: "BTCUSDT", realQty: "0", wantCalls: 1, terminal: true},
		{name: "ordinary_bad_fill_parse", kind: "MARKET", realStatus: "CANCELED", realClient: "lead_test", realSymbol: "BTCUSDT", realQty: "not-a-number", wantCalls: 1, uncertain: true},
		{name: "ordinary_other_client", kind: "MARKET", realStatus: "FILLED", realClient: "other", realSymbol: "BTCUSDT", realQty: "1", wantCalls: 1, uncertain: true},
		{name: "algo_waiting", kind: "STOP_MARKET", algoStatus: "NEW", wantCalls: 1},
		{name: "algo_canceled", kind: "TAKE_PROFIT_MARKET", algoStatus: "CANCELED", wantCalls: 1, terminal: true},
		{name: "algo_finished_without_real_order", kind: "STOP_MARKET", algoStatus: "FINISHED", wantCalls: 1, uncertain: true},
		{name: "algo_trigger_partial", kind: "STOP_MARKET", algoStatus: "FINISHED", actualID: "731", realStatus: "PARTIALLY_FILLED", realClient: "actual_child", realSymbol: "BTCUSDT", realQty: "0.4", wantCalls: 2},
		{name: "algo_trigger_full", kind: "TAKE_PROFIT_MARKET", algoStatus: "FINISHED", actualID: "731", realStatus: "FILLED", realClient: "actual_child", realSymbol: "BTCUSDT", realQty: "1", wantCalls: 2, terminal: true},
		{name: "algo_trigger_wrong_symbol", kind: "STOP_MARKET", algoStatus: "FINISHED", actualID: "731", realStatus: "FILLED", realClient: "actual_child", realSymbol: "ETHUSDT", realQty: "1", wantCalls: 2, uncertain: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var mu sync.Mutex
			seen := []string{}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				seen = append(seen, r.Method+" "+r.URL.Path)
				mu.Unlock()
				if r.Method != "GET" || r.Header.Get("X-MBX-APIKEY") != "lead-key" || r.URL.Query().Get("signature") == "" {
					t.Errorf("unexpected method or unsigned/non-Lead request: %s", r.URL.Path)
					w.WriteHeader(http.StatusForbidden)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/fapi/v1/algoOrder":
					if r.URL.Query().Get("clientAlgoId") != "lead_test" {
						t.Error("incorrect Algo client ID")
					}
					fmt.Fprintf(w, `{"algoId":888,"clientAlgoId":"lead_test","algoStatus":%q,"orderType":%q,"symbol":"BTCUSDT","side":"SELL","positionSide":"LONG","actualOrderId":%q}`, tc.algoStatus, tc.kind, tc.actualID)
				case "/fapi/v1/order":
					if tc.actualID != "" && r.URL.Query().Get("orderId") != "731" {
						t.Errorf("real order ID not queried: %v", r.URL.Query())
					}
					if tc.actualID == "" && r.URL.Query().Get("origClientOrderId") != "lead_test" {
						t.Errorf("ordinary order client ID not queried: %v", r.URL.Query())
					}
					orderID := 111
					if tc.actualID != "" {
						orderID = 731
					}
					clientID := tc.realClient
					fmt.Fprintf(w, `{"orderId":%d,"clientOrderId":%q,"status":%q,"type":%q,"executedQty":%q,"avgPrice":"100","symbol":%q,"side":"SELL","positionSide":"LONG"}`, orderID, clientID, tc.realStatus, func() string {
						if tc.actualID != "" {
							return "MARKET"
						}
						return tc.kind
					}(), tc.realQty, tc.realSymbol)
				default:
					t.Errorf("unexpected endpoint %s", r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()
			sdk := futures.NewClient("lead-key", "lead-secret")
			sdk.SetApiEndpoint(server.URL)
			sdk.HTTPClient = server.Client()
			account, err := binance.NewAccountClient(binance.LeadAccountID, sdk)
			if err != nil {
				t.Fatal(err)
			}
			adapter, err := NewLeadExecutionAdapter(account, riskSymbols(t))
			if err != nil {
				t.Fatal(err)
			}
			evidence, err := adapter.InspectOrderRecovery(context.Background(), "BTCUSDT", "lead_test", tc.kind)
			if errors.Is(err, ErrLeadRecoveryUncertain) != tc.uncertain || evidence.Terminal != tc.terminal || !evidence.RequiresReconcile {
				t.Fatalf("evidence=%+v err=%v", evidence, err)
			}
			if tc.actualID != "" && !tc.uncertain && evidence.ActualOrderID != "731" {
				t.Fatalf("actual order not linked: %+v", evidence)
			}
			mu.Lock()
			got := append([]string{}, seen...)
			mu.Unlock()
			if len(got) != tc.wantCalls {
				t.Fatalf("expected %d read-only GETs, got %v", tc.wantCalls, got)
			}
			for _, call := range got {
				if !strings.HasPrefix(call, "GET ") {
					t.Fatalf("mutation made: %v", got)
				}
			}
			if adapter.pendingReconcile {
				t.Fatal("diagnostic unexpectedly modified pending state")
			}
		})
	}
}

func TestStage44LookupFailureCannotUnlockOrRetry(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "GET" {
			t.Errorf("mutation: %s", r.Method)
		}
		w.WriteHeader(http.StatusGatewayTimeout)
		_, _ = w.Write([]byte(`{"code":-1007,"msg":"timeout"}`))
	}))
	defer server.Close()
	sdk := futures.NewClient("lead-key", "secret")
	sdk.SetApiEndpoint(server.URL)
	sdk.HTTPClient = server.Client()
	lead, err := binance.NewAccountClient(binance.LeadAccountID, sdk)
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := NewLeadExecutionAdapter(lead, riskSymbols(t))
	if err != nil {
		t.Fatal(err)
	}
	adapter.pendingReconcile = true // simulate unresolved earlier write
	evidence, err := adapter.InspectOrderRecovery(context.Background(), "BTCUSDT", "lead_test", "LIMIT")
	if !errors.Is(err, ErrLeadRecoveryUncertain) || !evidence.RequiresReconcile || !adapter.pendingReconcile || calls != 1 {
		t.Fatalf("unsafe recovery evidence=%+v err=%v calls=%d", evidence, err, calls)
	}
}

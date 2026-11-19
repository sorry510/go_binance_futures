package binance

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/adshao/go-binance/v2/futures"
)

func TestUF2ExchangeMarginLeverageFailuresNeverGetSuccessCache(t *testing.T) {
	cases := []struct {
		name              string
		marginCode        int
		leverageCode      int
		leverageResponse  string
		allow             bool
		wantLeverageCalls int
	}{
		{"margin_rejected", -4047, 0, "", false, 0},
		{"margin_already_set", -4046, 0, "", true, 1},
		{"leverage_rejected", 0, -4028, "", false, 1},
		{"leverage_ack_wrong_amount", 0, 0, `{"leverage":2,"maxNotionalValue":"1000000","symbol":"BTCUSDT"}`, false, 1},
		{"leverage_ack_wrong_symbol", 0, 0, `{"leverage":4,"maxNotionalValue":"1000000","symbol":"ETHUSDT"}`, false, 1},
		{"leverage_ack_empty", 0, 0, `{}`, false, 1},
		{"margin_success_leverage_success", 0, 0, "", true, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			marginCalls, leverageCalls := 0, 0
			installTradeConfigTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("futures config must use POST, got %s", r.Method)
				}
				switch r.URL.Path {
				case "/fapi/v1/marginType":
					marginCalls++
					if tc.marginCode != 0 {
						w.WriteHeader(http.StatusBadRequest)
						fmt.Fprintf(w, `{"code":%d,"msg":"configuration rejected"}`, tc.marginCode)
						return
					}
					fmt.Fprint(w, `{"code":200,"msg":"success"}`)
				case "/fapi/v1/leverage":
					leverageCalls++
					if tc.leverageCode != 0 {
						w.WriteHeader(http.StatusBadRequest)
						fmt.Fprintf(w, `{"code":%d,"msg":"leverage rejected"}`, tc.leverageCode)
						return
					}
					if tc.leverageResponse != "" {
						fmt.Fprint(w, tc.leverageResponse)
						return
					}
					fmt.Fprint(w, `{"leverage":4,"maxNotionalValue":"1000000","symbol":"BTCUSDT"}`)
				default:
					t.Errorf("unexpected futures endpoint: %s", r.URL.Path)
				}
			}))
			err := EnsureTradeConfigContext(context.Background(), "BTCUSDT", futures.MarginTypeCrossed, 4)
			if tc.allow && err != nil {
				t.Fatalf("valid config rejected: %v", err)
			}
			if !tc.allow && err == nil {
				t.Fatalf("%s failure was treated as confirmed config", tc.name)
			}
			if marginCalls != 1 || leverageCalls != tc.wantLeverageCalls {
				t.Fatalf("expected margin=1 leverage=%d: actual margin=%d leverage=%d", tc.wantLeverageCalls, marginCalls, leverageCalls)
			}
			if !tc.allow && hasRuntimeTradeConfig("BTCUSDT", futures.MarginTypeCrossed, 4) {
				t.Fatal("failed Binance config was cached as successful")
			}
			if tc.allow {
				if err = EnsureTradeConfigContext(context.Background(), "BTCUSDT", futures.MarginTypeCrossed, 4); err != nil {
					t.Fatalf("confirmed config cache missed: %v", err)
				}
				if marginCalls != 1 || leverageCalls != 1 {
					t.Fatalf("success cache did not suppress repeat Binance config calls: margin=%d leverage=%d", marginCalls, leverageCalls)
				}
			}
		})
	}
}
func TestUF2CanceledTradeConfigFailsWithoutSuccessfulCache(t *testing.T) {
	installTradeConfigTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("canceled config should not send Binance request: %s", r.URL.Path)
	}))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := EnsureTradeConfigContext(ctx, "BTCUSDT", futures.MarginTypeIsolated, 4)
	if !errors.Is(err, context.Canceled) && (err == nil || !strings.Contains(err.Error(), "canceled")) {
		t.Fatalf("canceled futures context marked config success: %v", err)
	}
	if hasRuntimeTradeConfig("BTCUSDT", futures.MarginTypeIsolated, 4) {
		t.Fatal("canceled futures config was cached as successful")
	}
}

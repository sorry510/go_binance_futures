package controllers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	beegocontext "github.com/beego/beego/v2/server/web/context"
	"github.com/golang-jwt/jwt/v5"
)

func TestUF5PerformanceJWTAndUntrustedFiltersRejectedBeforeDatabase(t *testing.T) {
	cases := []struct {
		name, url string
		operator  string
		want      int
	}{
		{"unauthenticated", "/orders/performance", "", 401},
		{"bad_days", "/orders/performance?days=bad", "operator", 400},
		{"unsupported_days", "/orders/performance?days=365", "operator", 400},
		{"injected_symbol", "/orders/performance?symbol=BTCUSDT%27", "operator", 400},
		{"invalid_side", "/orders/performance?position_side=BOTH", "operator", 400},
		{"injected_rule_hash", "/orders/performance?rule_hash=abcd%27%20OR%201%3D1", "operator", 400},
		{"invalid_rule_hash", "/orders/performance?rule_hash=abc", "operator", 400},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, tc.url, nil)
			recorder := httptest.NewRecorder()
			ctx := beegocontext.NewContext()
			ctx.Reset(recorder, request)
			if tc.operator != "" {
				ctx.Input.SetData("user", jwt.MapClaims{"Username": tc.operator})
			}
			ctrl := new(OrderController)
			ctrl.Init(ctx, "OrderController", "Performance", nil)
			ctrl.Performance()
			if recorder.Code != tc.want || !strings.Contains(recorder.Header().Get("Content-Type"), "application/json") {
				t.Fatalf("expected HTTP %d JSON, received %d %s: %s", tc.want, recorder.Code, recorder.Header().Get("Content-Type"), recorder.Body.String())
			}
		})
	}
}

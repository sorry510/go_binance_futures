package controllers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	beegocontext "github.com/beego/beego/v2/server/web/context"
	"github.com/golang-jwt/jwt/v5"
)

func TestOracle1AnomaliesOptionalDisplayFlagAuthAndValidation(t *testing.T) {
	for _, tc := range []struct {
		name, url     string
		authenticated bool
		want          int
	}{
		{"unauthorized", "/orders/anomalies?exclude_settlement_pending=1", false, 401},
		{"bad_flag", "/orders/anomalies?exclude_settlement_pending=maybe", true, 400},
		{"injected_flag", "/orders/anomalies?exclude_settlement_pending=true%27", true, 400},
		{"bad_account", "/orders/anomalies?exclude_settlement_pending=1&account_id=oracle1%27", true, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			rec := httptest.NewRecorder()
			ctx := beegocontext.NewContext()
			ctx.Reset(rec, req)
			if tc.authenticated {
				ctx.Input.SetData("user", jwt.MapClaims{"Username": "operator"})
			}
			ctrl := new(OrderController)
			ctrl.Init(ctx, "OrderController", "Anomalies", nil)
			ctrl.Anomalies()
			if rec.Code != tc.want || !strings.Contains(rec.Header().Get("Content-Type"), "application/json") {
				t.Fatalf("unsafe anomaly flag accepted: wanted HTTP %d JSON, got %d %s", tc.want, rec.Code, rec.Body.String())
			}
		})
	}
}

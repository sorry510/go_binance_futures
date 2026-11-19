package controllers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	beegocontext "github.com/beego/beego/v2/server/web/context"
	"github.com/golang-jwt/jwt/v5"
)

func TestUF3AnomalyRouteRequiresJWTAndValidFiltersBeforeDB(t *testing.T) {
	cases := []struct {
		name, url  string
		authorized bool
		want       int
	}{
		{"unauthorized", "/orders/anomalies?page=1&limit=20", false, 401},
		{"bad_page", "/orders/anomalies?page=NaN", true, 400},
		{"oversized_limit", "/orders/anomalies?page=1&limit=101", true, 400},
		{"foreign_account", "/orders/anomalies?account_id=thirdparty", true, 400},
		{"injected_owner", "/orders/anomalies?owner=auto_strategy%27", true, 400},
		{"bad_category", "/orders/anomalies?category=__all_schemas__", true, 400},
		{"bad_side", "/orders/anomalies?position_side=BOTH", true, 400},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			rec := httptest.NewRecorder()
			ctx := beegocontext.NewContext()
			ctx.Reset(rec, req)
			if tc.authorized {
				ctx.Input.SetData("user", jwt.MapClaims{"Username": "operator"})
			}
			ctrl := new(OrderController)
			ctrl.Init(ctx, "OrderController", "Anomalies", nil)
			ctrl.Anomalies()
			if rec.Code != tc.want || !strings.Contains(rec.Header().Get("Content-Type"), "application/json") {
				t.Fatalf("%s expected HTTP=%d JSON, got %d %s: %s", tc.name, tc.want, rec.Code, rec.Header().Get("Content-Type"), rec.Body.String())
			}
		})
	}
}

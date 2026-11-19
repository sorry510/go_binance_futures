package controllers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	beegocontext "github.com/beego/beego/v2/server/web/context"
	"github.com/golang-jwt/jwt/v5"
)

func TestUF6OrdersSummaryAndPagingAuthFilterValidation(t *testing.T) {
	for _, tc := range []struct {
		name, method, url string
		auth              bool
		want              int
	}{
		{"summary_requires_jwt", "summary", "/orders/summary", false, 401},
		{"page_requires_jwt", "list", "/orders?include_summary=false", false, 401},
		{"summary_invalid_start", "summary", "/orders/summary?start_time=invalid", true, 400},
		{"summary_invalid_end", "summary", "/orders/summary?end_time=NaN", true, 400},
		{"summary_unknown_account", "summary", "/orders/summary?account_id=foreign", true, 400},
		{"summary_unknown_owner", "summary", "/orders/summary?owner=auto_strategy%27", true, 400},
		{"summary_injected_symbol", "summary", "/orders/summary?symbol=BTC%27%20OR%201%3D1", true, 400},
		{"summary_invalid_side", "summary", "/orders/summary?position_side=HEDGE", true, 400},
		{"summary_invalid_date_range", "summary", "/orders/summary?start_time=1000&end_time=500", true, 400},
		{"list_invalid_include_flag", "list", "/orders?include_summary=maybe", true, 400},
		{"list_injected_include_flag", "list", "/orders?include_summary=1%27", true, 400},
		{"list_invalid_page", "list", "/orders?include_summary=0&page=oops", true, 400},
		{"list_invalid_limit", "list", "/orders?include_summary=false&limit=101", true, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			rec := httptest.NewRecorder()
			ctx := beegocontext.NewContext()
			ctx.Reset(rec, req)
			if tc.auth {
				ctx.Input.SetData("user", jwt.MapClaims{"Username": "operator"})
			}
			ctrl := new(OrderController)
			ctrl.Init(ctx, "OrderController", tc.method, nil)
			switch tc.method {
			case "summary":
				ctrl.Summary()
			case "list":
				ctrl.Get()
			}
			if rec.Code != tc.want || !strings.Contains(rec.Header().Get("Content-Type"), "application/json") {
				t.Fatalf("expected %d JSON, got HTTP %d %s: %s", tc.want, rec.Code, rec.Header().Get("Content-Type"), rec.Body.String())
			}
		})
	}
}

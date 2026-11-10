package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	beegocontext "github.com/beego/beego/v2/server/web/context"
	"github.com/golang-jwt/jwt/v5"
)

func TestS4ReadRoutesJWTAndScopeFailBeforeDB(t *testing.T) {
	cases := []struct {
		name, url string
		claims    jwt.MapClaims
		call      func(*SpotStrategyController)
		want      int
	}{
		{"config_no_auth", "/spots/symbols?account_id=main&owner=auto_strategy", nil, (*SpotStrategyController).ListSymbols, http.StatusUnauthorized},
		{"holding_no_auth", "/spots/strategy/holdings?account_id=main&owner=auto_strategy", nil, (*SpotStrategyController).ListHoldings, http.StatusUnauthorized},
		{"orders_no_auth", "/spots/strategy/orders?account_id=main&owner=auto_strategy", nil, (*SpotStrategyController).ListOrders, http.StatusUnauthorized},
		{"fills_no_auth", "/spots/strategy/fills?account_id=main&owner=auto_strategy&order_id=1", nil, (*SpotStrategyController).ListFills, http.StatusUnauthorized},
		{"config_bad_page", "/spots/symbols?page=-1", jwt.MapClaims{"Username": "operator"}, (*SpotStrategyController).ListSymbols, http.StatusBadRequest},
		{"holdings_wrong_account", "/spots/strategy/holdings?account_id=lead&owner=auto_strategy", jwt.MapClaims{"Username": "operator"}, (*SpotStrategyController).ListHoldings, http.StatusBadRequest},
		{"orders_negative_page", "/spots/strategy/orders?account_id=main&owner=auto_strategy&page=-5", jwt.MapClaims{"Username": "operator"}, (*SpotStrategyController).ListOrders, http.StatusBadRequest},
		{"fills_invalid_order", "/spots/strategy/fills?account_id=main&owner=auto_strategy&order_id=binance-123", jwt.MapClaims{"Username": "operator"}, (*SpotStrategyController).ListFills, http.StatusBadRequest},
		{"fills_invalid_limit", "/spots/strategy/fills?account_id=main&owner=auto_strategy&limit=abc&order_id=1", jwt.MapClaims{"Username": "operator"}, (*SpotStrategyController).ListFills, http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, accept := range []string{"application/json", "application/xml", "application/x-yaml"} {
				req := httptest.NewRequest(http.MethodGet, tc.url, nil)
				req.Header.Set("Accept", accept)
				rec := httptest.NewRecorder()
				ctx := beegocontext.NewContext()
				ctx.Reset(rec, req)
				if tc.claims != nil {
					ctx.Input.SetData("user", tc.claims)
				}
				controller := new(SpotStrategyController)
				controller.Init(ctx, "SpotStrategyController", "", nil)
				tc.call(controller)
				if rec.Code != tc.want || !strings.Contains(rec.Header().Get("Content-Type"), "application/json") {
					t.Fatalf("Accept=%s status=%d want=%d body=%s", accept, rec.Code, tc.want, rec.Body.String())
				}
				var out struct {
					Code int    `json:"code"`
					Msg  string `json:"msg"`
					Data any    `json:"data"`
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil || out.Code != tc.want || out.Data != nil {
					t.Fatalf("bad JSON response %+v err=%v", out, err)
				}
			}
		})
	}
}

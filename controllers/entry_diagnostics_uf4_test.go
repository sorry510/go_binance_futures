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

func TestUF4MainEntryDiagnosticsEndpointIsJWTOnlyReadOnly(t *testing.T) {
	for _, tc := range []struct {
		name, url string
		username  string
		want      int
	}{
		{"no_auth", "/futures/entry-diagnostics", "", 401},
		{"bad_symbol", "/futures/entry-diagnostics?symbol=BTC%25USDT", "operator", 400},
		{"bad_side", "/futures/entry-diagnostics?position_side=BOTH", "operator", 400},
		{"injected_symbol", "/futures/entry-diagnostics?symbol=BTC%27%20OR%201%3D1", "operator", 400},
		{"valid_empty_first_start", "/futures/entry-diagnostics?symbol=BTCUSDT&position_side=LONG", "operator", 200},
		{"default_snapshot", "/futures/entry-diagnostics", "operator", 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			rec := httptest.NewRecorder()
			ctx := beegocontext.NewContext()
			ctx.Reset(rec, req)
			if tc.username != "" {
				ctx.Input.SetData("user", jwt.MapClaims{"Username": tc.username})
			}
			ctrl := new(MainEntryDiagnosticsController)
			ctrl.Init(ctx, "MainEntryDiagnosticsController", "Get", nil)
			ctrl.Get()
			if rec.Code != tc.want || !strings.Contains(rec.Header().Get("Content-Type"), "application/json") {
				t.Fatalf("expected HTTP %d JSON, got %d %s: %s", tc.want, rec.Code, rec.Header().Get("Content-Type"), rec.Body.String())
			}
			if tc.want == 200 {
				var body struct {
					Code int `json:"code"`
					Data struct {
						AccountID string `json:"account_id"`
						Scope     string `json:"scope"`
					} `json:"data"`
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.Code != 200 || body.Data.AccountID != "main" || body.Data.Scope != "current_process_last_main_cycle" {
					t.Fatalf("wrong safe Main diagnostic envelope: %+v err=%v", body, err)
				}
			}
		})
	}
}

package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	beegocontext "github.com/beego/beego/v2/server/web/context"
	"github.com/golang-jwt/jwt/v5"

	"go_binance_futures/service/spotstrategy"
)

// Eight Accept types must produce the same JSON response and HTTP status.
// All paths fail before touching any real database or exchange.
func TestS1SpotStrategyAlwaysJSONAndAuthorized(t *testing.T) {
	accepts := []string{"application/json", "*/*", "text/plain",
		"application/x-yaml", "application/yaml", "application/xml",
		"text/xml", "application/x-protobuf"}
	for _, accept := range accepts {
		for _, tc := range []struct {
			name, method, url, body, contentType string
			claims                               jwt.MapClaims
			want                                 int
			call                                 func(*SpotStrategyController)
		}{
			{"get_missing_claims", http.MethodGet, "/spots/strategy/config?account_id=main&owner=auto_strategy&symbol=BTCUSDT", "", "", nil, http.StatusUnauthorized, (*SpotStrategyController).Get},
			{"get_missing_symbol", http.MethodGet, "/spots/strategy/config", "", "", jwt.MapClaims{"Username": "operator"}, http.StatusBadRequest, (*SpotStrategyController).Get},
			{"put_mode_missing_claims", http.MethodPut, "/spots/strategy/mode", `{"account_id":"main","owner":"auto_strategy","mode":"off"}`, "application/json", nil, http.StatusUnauthorized, (*SpotStrategyController).PutMode},
			{"put_mode_no_json", http.MethodPut, "/spots/strategy/mode", `{"account_id":"main","owner":"auto_strategy","mode":"off"}`, "application/x-www-form-urlencoded", jwt.MapClaims{"Username": "operator"}, http.StatusUnsupportedMediaType, (*SpotStrategyController).PutMode},
			{"put_mode_bad_scope", http.MethodPut, "/spots/strategy/mode", `{"account_id":"lead","owner":"auto_strategy","mode":"off"}`, "application/json", jwt.MapClaims{"Username": "operator"}, http.StatusBadRequest, (*SpotStrategyController).PutMode},
			{"put_config_no_json", http.MethodPut, "/spots/strategy/config", `{}`, "application/x-yaml", jwt.MapClaims{"Username": "operator"}, http.StatusUnsupportedMediaType, (*SpotStrategyController).Put},
			{"put_config_reject_account_fields", http.MethodPut, "/spots/strategy/config", `{"account_id":"lead","owner":"auto_strategy","symbol":"BTCUSDT","enabled":false,"budget_quote":"0"}`, "application/json", jwt.MapClaims{"Username": "operator"}, http.StatusBadRequest, (*SpotStrategyController).Put},
		} {
			t.Run(accept+"/"+tc.name, func(t *testing.T) {
				req := httptest.NewRequest(tc.method, tc.url, strings.NewReader(tc.body))
				req.Header.Set("Accept", accept)
				if tc.contentType != "" {
					req.Header.Set("Content-Type", tc.contentType)
				}
				rec := httptest.NewRecorder()
				ctx := beegocontext.NewContext()
				ctx.Reset(rec, req)
				if tc.claims != nil {
					ctx.Input.SetData("user", tc.claims)
				}
				if len(tc.body) > 0 {
					ctx.Input.RequestBody = []byte(tc.body)
				}
				ctrl := new(SpotStrategyController)
				ctrl.Init(ctx, "SpotStrategyController", "", nil)
				tc.call(ctrl)
				if rec.Code != tc.want {
					t.Fatalf("status=%d want=%d body=%s", rec.Code, tc.want, rec.Body.String())
				}
				if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
					t.Fatalf("Accept %s changed content-type to %q", accept, ct)
				}
				var body struct {
					Code int             `json:"code"`
					Data json.RawMessage `json:"data"`
					Msg  string          `json:"msg"`
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
					t.Fatalf("non JSON: %v body=%s", err, rec.Body.String())
				}
				if body.Code != tc.want || body.Msg == "" || string(body.Data) != "null" {
					t.Fatalf("unexpected error JSON %+v", body)
				}
			})
		}
	}
}

func TestS1SpotStrategyPayloadWhitelist(t *testing.T) {
	for _, payload := range []string{
		`{"account_id":"main","owner":"auto_strategy","symbol":"BTCUSDT","enabled":true,"budget_quote":"10","mode":"strategy"}`,
		`{"account_id":"main","owner":"auto_strategy","symbol":"BTCUSDT","enabled":true,"budget_quote":"10","exchange_order_id":"fake"}`,
		`{"account_id":"main"}{}`,
		`{"account_id":"main","owner":"auto_strategy","symbol":"BTCUSDT","enabled":false,"budget_quote":"0"}`,
		"{}{}",
		strings.Repeat("X", 32769),
	} {
		var req spotstrategy.Patch
		if err := decodeSpotStrategyPayload([]byte(payload), &req); err == nil {
			t.Fatalf("accepted unknown/concatenated/oversized Spot request: %q", payload[:min(64, len(payload))])
		}
	}
	var req spotstrategy.Patch
	if err := decodeSpotStrategyPayload([]byte(`{"symbol":"BTCUSDT","enabled":false,"budget_quote":"0"}`), &req); err != nil {
		t.Fatal(err)
	}
	if req.Enabled == nil || *req.Enabled {
		t.Fatalf("enabled decode: %+v", req)
	}
}

// The positive 200 path must also ignore Accept, not just error paths.
func TestS1SpotStrategySuccessAlwaysJSON(t *testing.T) {
	for _, accept := range []string{"application/json", "*/*", "text/plain", "application/x-yaml",
		"application/yaml", "application/xml", "text/xml", "application/x-protobuf"} {
		t.Run(accept, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/spots/strategy/mode", nil)
			req.Header.Set("Accept", accept)
			rec := httptest.NewRecorder()
			ctx := beegocontext.NewContext()
			ctx.Reset(rec, req)
			ctrl := new(SpotStrategyController)
			ctrl.Init(ctx, "SpotStrategyController", "", nil)
			ctrl.respond(http.StatusOK, map[string]any{"execution": "live_disabled"}, "success")
			if rec.Code != http.StatusOK || !strings.Contains(rec.Header().Get("Content-Type"), "application/json") {
				t.Fatalf("response negotiation at Accept=%s code=%d type=%s", accept, rec.Code, rec.Header().Get("Content-Type"))
			}
			var payload struct {
				Code int `json:"code"`
				Data struct {
					Execution string `json:"execution"`
				} `json:"data"`
				Msg string `json:"msg"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil || payload.Code != 200 || payload.Data.Execution != "live_disabled" {
				t.Fatalf("bad JSON: %+v err=%v", payload, err)
			}
		})
	}
}

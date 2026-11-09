package controllers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	beegocontext "github.com/beego/beego/v2/server/web/context"
	"github.com/golang-jwt/jwt/v5"
)

// Stage 01-5 keeps the old destructive endpoints disabled; GET now
// retrieves new Managed+Fill history and is covered by read-only service tests.
func TestStage015LegacyDeleteRoutesRemainGoneWithoutDB(t *testing.T) {
	for _, tc := range []struct {
		name, path string
		method     string
	}{
		{"delete_one", "/orders/123", http.MethodDelete},
		{"delete_all", "/orders?symbol=BTCUSDT", http.MethodDelete},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rec := httptest.NewRecorder()
			ctx := beegocontext.NewContext()
			ctx.Reset(rec, req)
			ctrl := new(OrderController)
			ctrl.Init(ctx, "OrderController", "", nil)
			switch tc.name {
			case "delete_one":
				ctrl.Delete()
			case "delete_all":
				ctrl.DeleteAll()
			}
			if rec.Code != http.StatusGone || !strings.Contains(rec.Body.String(), "historical exchange fills cannot be deleted") {
				t.Fatalf("legacy route did not fail closed: http=%d body=%q", rec.Code, rec.Body.String())
			}
		})
	}
}

// The new GET API must validate parameters before any ORM query. These
// negative tests deliberately do not register a database.
func TestStage015HistoryControllerInvalidFiltersFailWithoutDB(t *testing.T) {
	cases := []struct {
		name, path, param string
		detail            bool
	}{
		{"invalid_page", "/orders?page=bad", "", false},
		{"oversized_limit", "/orders?limit=101", "", false},
		{"wrong_account", "/orders?account_id=thirdparty", "", false},
		{"sql_like_symbol", "/orders?symbol=BTC%27", "", false},
		{"negative_position_id", "/orders/0/details", "0", true},
		{"non_numeric_position_id", "/orders/invalid/details", "invalid", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			ctx := beegocontext.NewContext()
			ctx.Reset(rec, req)
			ctx.Input.SetData("user", jwt.MapClaims{"Username": "single_operator"})
			if tc.detail {
				ctx.Input.SetParam(":position_id", tc.param)
			}
			ctrl := new(OrderController)
			ctrl.Init(ctx, "OrderController", "", nil)
			if tc.detail {
				ctrl.Details()
			} else {
				ctrl.Get()
			}
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("invalid route unexpectedly accessed DB: status=%d body=%q", rec.Code, rec.Body.String())
			}
		})
	}
}

// All historical order endpoints promise JSON, regardless of the caller's
// Accept header. In Beego, Context.Resp would negotiate YAML/XML and panic
// for application/x-protobuf when serializing a map.
func TestStage015HistoryResponseFixedJSONAcrossAcceptTypes(t *testing.T) {
	for _, accept := range []string{"application/json", "application/x-yaml",
		"application/xml", "text/xml", "application/x-protobuf"} {
		t.Run(accept, func(t *testing.T) {
			for _, operation := range []string{"delete", "validation"} {
				t.Run(operation, func(t *testing.T) {
					target := "/orders"
					method := http.MethodDelete
					if operation == "validation" {
						method = http.MethodGet
						target = "/orders?page=not-a-number"
					}
					req := httptest.NewRequest(method, target, nil)
					req.Header.Set("Accept", accept)
					recorder := httptest.NewRecorder()
					ctx := beegocontext.NewContext()
					ctx.Reset(recorder, req)
					ctrl := new(OrderController)
					ctrl.Init(ctx, "OrderController", "", nil)
					ctx.Input.SetData("user", jwt.MapClaims{"Username": "single_operator"})
					if operation == "delete" {
						ctrl.DeleteAll()
					} else {
						ctrl.Get()
					}
					expected := http.StatusGone
					if operation == "validation" {
						expected = http.StatusBadRequest
					}
					if recorder.Code != expected {
						t.Fatalf("HTTP %d vs %d body=%s", recorder.Code, expected, recorder.Body.String())
					}
					if got := recorder.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
						t.Fatalf("Accept %q changed Content-Type to %q", accept, got)
					}
					var response struct {
						Code int             `json:"code"`
						Data json.RawMessage `json:"data"`
						Msg  string          `json:"msg"`
					}
					if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
						t.Fatalf("Accept %q returned non-JSON %q: %v", accept, recorder.Body.String(), err)
					}
					if response.Code != expected || string(response.Data) != "null" || response.Msg == "" {
						t.Fatalf("invalid JSON response: %+v", response)
					}
				})
			}
		})
	}
}

// An accidental route registration without JwtMiddleware cannot expose
// the single-operator account's main or lead history.
func TestStage015HistoryGETRequiresValidatedMiddlewareClaims(t *testing.T) {
	for _, target := range []string{"/orders?account_id=main", "/orders?account_id=lead",
		"/orders/1/details?account_id=main", "/orders/1/details?account_id=lead"} {
		t.Run(target, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, target, nil)
			recorder := httptest.NewRecorder()
			ctx := beegocontext.NewContext()
			ctx.Reset(recorder, req)
			ctx.Input.SetParam(":position_id", "1")
			ctrl := new(OrderController)
			ctrl.Init(ctx, "OrderController", "", nil)
			if strings.Contains(target, "/details") {
				ctrl.Details()
			} else {
				ctrl.Get()
			}
			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("unprotected order history read allowed: path=%s status=%d body=%q", target, recorder.Code, recorder.Body.String())
			}
			var payload map[string]any
			if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
				t.Fatalf("unauthorized response is not JSON: %v", err)
			}
			if payload["code"] != float64(http.StatusUnauthorized) {
				t.Fatalf("unauthorized response code mismatch: %+v", payload)
			}
		})
	}
}

// A successful 200 response must also ignore Accept; unlike GET, this test
// exercises serialization without accessing a database.
func TestStage015SuccessResponseAlwaysJSON(t *testing.T) {
	for _, accept := range []string{"application/json", "application/xml",
		"application/x-yaml", "application/x-protobuf", "text/xml"} {
		t.Run(accept, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/orders", nil)
			req.Header.Set("Accept", accept)
			recorder := httptest.NewRecorder()
			ctx := beegocontext.NewContext()
			ctx.Reset(recorder, req)
			ctrl := new(OrderController)
			ctrl.Init(ctx, "OrderController", "", nil)
			ctrl.success(map[string]any{"total": 3, "list": []string{}})
			if recorder.Code != http.StatusOK {
				t.Fatalf("success status=%d", recorder.Code)
			}
			if ct := recorder.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
				t.Fatalf("success content type=%s", ct)
			}
			var payload struct {
				Code int `json:"code"`
				Data struct {
					Total int      `json:"total"`
					List  []string `json:"list"`
				} `json:"data"`
				Msg string `json:"msg"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
				t.Fatalf("Accept %s returned non JSON success: %s: %v", accept, recorder.Body.String(), err)
			}
			if payload.Code != 200 || payload.Data.Total != 3 || payload.Msg != "success" {
				t.Fatalf("invalid successful JSON response: %+v", payload)
			}
		})
	}
}

// Batch requests must require the operator JWT and reject malformed scopes,
// forged verification and missing confirmation without accessing the DB.
func TestHistoryBulkRoutesRequireOperatorAndValidateInputs(t *testing.T) {
	cases := []struct {
		name, path, body string
		authorized       bool
		want             int
	}{
		{"unauth_preview", "/orders/batch/preview", `{"action":"hide","filters":{"account_id":"main"}}`, false, 401},
		{"unauth_apply", "/orders/batch/apply", `{"action":"hide","filters":{"account_id":"main"}}`, false, 401},
		{"bad_json", "/orders/batch/preview", `{"action":`, true, 400},
		{"invalid_account", "/orders/batch/preview", `{"action":"hide","filters":{"account_id":"attacker"}}`, true, 400},
		{"invalid_status", "/orders/batch/preview", `{"action":"update","status_override":"forged","filters":{"account_id":"main"}}`, true, 400},
		{"forged_verified", "/orders/batch/preview", `{"action":"update","completeness_override":"verified","filters":{"account_id":"main"}}`, true, 400},
		{"missing_digest", "/orders/batch/apply", `{"action":"hide","filters":{"account_id":"main"},"reason":"historical audit requested"}`, true, 400},
		{"unknown_field", "/orders/batch/preview", `{"action":"hide","filters":{"account_id":"main"},"surprise":42}`, true, 400},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tc.path, bytes.NewBufferString(tc.body))
			req.Header.Set("Accept", "application/x-protobuf")
			recorder := httptest.NewRecorder()
			ctx := beegocontext.NewContext()
			ctx.Reset(recorder, req)
			if tc.authorized {
				ctx.Input.SetData("user", jwt.MapClaims{"Username": "single_operator"})
			}
			ctrl := new(OrderController)
			ctrl.Init(ctx, "OrderController", "", nil)
			if strings.Contains(tc.path, "preview") {
				ctrl.BulkPreview()
			} else {
				ctrl.BulkApply()
			}
			if recorder.Code != tc.want {
				t.Fatalf("unexpected http=%d want=%d body=%s", recorder.Code, tc.want, recorder.Body.String())
			}
			if !strings.Contains(recorder.Header().Get("Content-Type"), "application/json") {
				t.Fatalf("batch endpoints must be fixed JSON: %s", recorder.Header().Get("Content-Type"))
			}
		})
	}
}

// New physical batch deletion is a separate POST. The former DELETE
// endpoints continue to return 410 and cannot bypass the explicit UI scope.
func TestHistoryPhysicalDeleteControllerRequiresAuthAndValidCount(t *testing.T) {
	cases := []struct {
		name, body string
		auth       bool
		status     int
	}{
		{"unauthorized", `{"filters":{"account_id":"main"},"expected_count":1}`, false, 401},
		{"missing_count", `{"filters":{"account_id":"main"}}`, true, 400},
		{"negative_count", `{"filters":{"account_id":"main"},"expected_count":-1}`, true, 400},
		{"invalid_account", `{"filters":{"account_id":"foreign"},"expected_count":1}`, true, 400},
		{"invalid_symbol", `{"filters":{"account_id":"main","symbol":"BTC'"},"expected_count":1}`, true, 400},
		{"unknown_fields", `{"filters":{"account_id":"main"},"expected_count":1,"unsafe":true}`, true, 400},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/orders/batch/delete", strings.NewReader(c.body))
			request.Header.Set("Accept", "application/x-protobuf")
			recorder := httptest.NewRecorder()
			ctx := beegocontext.NewContext()
			ctx.Reset(recorder, request)
			if c.auth {
				ctx.Input.SetData("user", jwt.MapClaims{"Username": "single_operator"})
			}
			controller := new(OrderController)
			controller.Init(ctx, "OrderController", "", nil)
			controller.DeleteFiltered()
			if recorder.Code != c.status || !strings.Contains(recorder.Header().Get("Content-Type"), "application/json") {
				t.Fatalf("status=%d want=%d contentType=%q body=%s", recorder.Code, c.status, recorder.Header().Get("Content-Type"), recorder.Body.String())
			}
		})
	}
}

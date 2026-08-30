package middlewares

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/beego/beego/v2/server/web/context"
)

func newRequestContext(path string, headers map[string]string) (*context.Context, *httptest.ResponseRecorder) {
	request := httptest.NewRequest(http.MethodGet, path, nil)
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	recorder := httptest.NewRecorder()
	ctx := context.NewContext()
	ctx.Reset(recorder, request)
	return ctx, recorder
}

// An expired token used to make every API call answer 401 with an HTML body,
// which the frontend could not distinguish from an empty result, so pages
// rendered as "no data" rather than prompting for login.
func TestUnauthorizedApiRequestGetsJson(t *testing.T) {
	cases := []struct {
		name    string
		path    string
		headers map[string]string
	}{
		{
			name:    "axios style request",
			path:    "/test-strategy-results",
			headers: map[string]string{"Accept": "application/json, text/plain, */*"},
		},
		{
			name:    "explicit xhr marker",
			path:    "/features",
			headers: map[string]string{"X-Requested-With": "XMLHttpRequest", "Accept": "text/html"},
		},
		{
			name:    "no accept header at all",
			path:    "/config",
			headers: nil,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx, recorder := newRequestContext(testCase.path, testCase.headers)
			JwtMiddleware(ctx)

			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
			}
			if location := recorder.Header().Get("Location"); location != "" {
				t.Fatalf("api request should not be redirected, got Location %q", location)
			}
			var payload struct {
				Code int    `json:"code"`
				Msg  string `json:"msg"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
				t.Fatalf("body is not json: %v, body=%q", err, recorder.Body.String())
			}
			if payload.Code != http.StatusUnauthorized {
				t.Fatalf("payload code = %d, want %d", payload.Code, http.StatusUnauthorized)
			}
			if payload.Msg == "" {
				t.Fatal("payload should carry a message the frontend can surface")
			}
		})
	}
}

func TestUnauthorizedPageRequestRedirects(t *testing.T) {
	cases := []struct {
		name    string
		path    string
		headers map[string]string
	}{
		{
			name:    "browser navigation",
			path:    "/orders",
			headers: map[string]string{"Accept": "text/html,application/xhtml+xml"},
		},
		{
			name:    "html document path",
			path:    "/somewhere/page.html",
			headers: nil,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx, recorder := newRequestContext(testCase.path, testCase.headers)
			JwtMiddleware(ctx)

			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
			}
			location := recorder.Header().Get("Location")
			if !strings.HasSuffix(location, "/index.html") {
				t.Fatalf("Location = %q, want it to point at the login page", location)
			}
		})
	}
}

// The excluded routes must stay reachable without a token, otherwise login and
// the notification websocket handshake break.
func TestExcludedRoutesSkipAuth(t *testing.T) {
	for _, path := range []string{"/login", "/ws/notifications"} {
		t.Run(path, func(t *testing.T) {
			ctx, recorder := newRequestContext(path, nil)
			JwtMiddleware(ctx)
			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want the request to pass through untouched", recorder.Code)
			}
			if recorder.Body.Len() != 0 {
				t.Fatalf("middleware should not write a body, got %q", recorder.Body.String())
			}
		})
	}
}

package routers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/beego/beego/v2/server/web"
)

// A Go build can succeed while a route is inside a // comment. This test
// exercises the actual Beego router registration, rather than calling the
// controller method directly, so future URL edits cannot silently delete
// previously delivered UF endpoints.
func TestUF6RereviewAllOrderRoutesRegisteredAndProtected(t *testing.T) {
	cases := []struct {
		name string
		path string
		want int
	}{
		{"sentinel", "/uf6-not-a-real-route-404", http.StatusNotFound},
		{"UF3_anomalies", "/orders/anomalies", http.StatusUnauthorized},
		{"UF5_verified_performance", "/orders/performance", http.StatusUnauthorized},
		{"UF6_summary", "/orders/summary", http.StatusUnauthorized},
		{"legacy_history", "/orders", http.StatusUnauthorized},
		{"legacy_detail", "/orders/123/details", http.StatusUnauthorized},
		{"UF4_entry_diagnostics", "/futures/entry-diagnostics", http.StatusUnauthorized},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			web.BeeApp.Handlers.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("GET %s expected HTTP %d, got HTTP %d: %s", tc.path, tc.want, rec.Code, rec.Body.String())
			}
		})
	}
}

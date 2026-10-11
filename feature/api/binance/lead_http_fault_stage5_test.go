package binance

import (
	"context"
	"github.com/adshao/go-binance/v2/futures"
	"net/http"
	"sync"
	"testing"
)

type stage55StaticTransport struct {
	status int
}

func (s stage55StaticTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: s.status, Header: make(http.Header), Body: http.NoBody}, nil
}
func TestStage55LeadHTTP429418RoutesOnlyBoundedCodes(t *testing.T) {
	for _, code := range []int{429, 418, 503, 200} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			client, err := NewLeadAccountClient("fake-lead", "fake-secret", &http.Client{Transport: stage55StaticTransport{status: code}})
			if err != nil {
				t.Fatal(err)
			}
			var mu sync.Mutex
			events := []int{}
			client.SetLeadRateLimitHandler(func(status int) { mu.Lock(); events = append(events, status); mu.Unlock() })
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://fapi.binance.com/fapi/v1/account", nil)
			if err != nil {
				t.Fatal(err)
			}
			// Isolate the callback probe; the production WrapClient global
			// budget intentionally blocks subsequent requests after a 429.
			probe := &leadHTTPProbe{base: stage55StaticTransport{status: code}, observer: client.leadObserver}
			resp, err := probe.RoundTrip(req)
			if err != nil {
				t.Fatal(err)
			}
			_ = resp.Body.Close()
			mu.Lock()
			got := append([]int(nil), events...)
			mu.Unlock()
			expected := 0
			if code == 429 || code == 418 {
				expected = 1
			}
			if len(got) != expected {
				t.Fatalf("status %d emitted %v instead of count %d", code, got, expected)
			}
			if expected == 1 && got[0] != code {
				t.Fatalf("wrong observed code %v", got)
			}
			client.SetLeadRateLimitHandler(nil)
		})
	}
}
func TestStage55MainCannotSubscribeToLeadHTTPFaults(t *testing.T) {
	a, err := NewAccountClient(MainAccountID, futures.NewClient("main", "secret"))
	if err != nil {
		t.Fatal(err)
	}
	a.SetLeadRateLimitHandler(func(int) { t.Fatal("Main received Lead-only callback") })
	if a.leadObserver != nil {
		t.Fatal("Main unexpectedly subscribed to Lead faults")
	}
}

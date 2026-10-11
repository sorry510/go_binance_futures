package binance

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"go_binance_futures/service/binanceapiusage"
)

type stage5LimiterTransport struct{ count atomic.Int64 }

func (t *stage5LimiterTransport) RoundTrip(*http.Request) (*http.Response, error) {
	t.count.Add(1)
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("{}")), Header: make(http.Header)}, nil
}
func TestStage54LeadPerKeyReservesCloseAndCancelSlots(t *testing.T) {
	base := &stage5LimiterTransport{}
	limiter := &leadOrderLimiter{base: base}
	execute := func(ctx context.Context, method, path string) error {
		request, err := http.NewRequestWithContext(ctx, method, "https://fapi.binance.com"+path, nil)
		if err != nil {
			return err
		}
		response, err := limiter.RoundTrip(request)
		if err == nil && response != nil {
			_ = response.Body.Close()
		}
		return err
	}
	for i := 0; i < 16; i++ {
		if err := execute(context.Background(), http.MethodPost, "/fapi/v1/order"); err != nil {
			t.Fatalf("open #%d failed: %v", i, err)
		}
	}
	if err := execute(context.Background(), http.MethodPost, "/fapi/v1/order"); !errors.Is(err, binanceapiusage.ErrBudgetDeferred) {
		t.Fatalf("17th open consumed reserved exit slot: %v", err)
	}
	if err := execute(WithLeadSafetyExitPriority(context.Background()), http.MethodPost, "/fapi/v1/order"); err != nil {
		t.Fatalf("first close was not reserved: %v", err)
	}
	if err := execute(context.Background(), http.MethodDelete, "/fapi/v1/order"); err != nil {
		t.Fatalf("cancel was not reserved: %v", err)
	}
	if err := execute(context.Background(), http.MethodDelete, "/fapi/v1/order"); !errors.Is(err, binanceapiusage.ErrBudgetDeferred) {
		t.Fatalf("order cap was bypassed: %v", err)
	}
	if got := base.count.Load(); got != 18 {
		t.Fatalf("base transport unexpectedly received %d orders", got)
	}
	// Public market data is outside per-Key private order limits; IP budget
	// still applies at the outer shared transport.
	if err := execute(context.Background(), http.MethodGet, "/fapi/v1/exchangeInfo"); err != nil {
		t.Fatal(err)
	}
	if got := base.count.Load(); got != 19 {
		t.Fatalf("public GET incorrectly counted as order %d", got)
	}
}
func TestStage54MainLimitersNotAutomaticallySharedWithLead(t *testing.T) {
	main := &stage5LimiterTransport{}
	lead := &stage5LimiterTransport{}
	limiter := &leadOrderLimiter{base: lead}
	request, _ := http.NewRequest(http.MethodPost, "https://fapi.binance.com/fapi/v1/algoOrder", nil)
	_, err := limiter.RoundTrip(request)
	if err != nil {
		t.Fatal(err)
	}
	if main.count.Load() != 0 || lead.count.Load() != 1 {
		t.Fatal("Lead limiter touched Main client")
	}
}

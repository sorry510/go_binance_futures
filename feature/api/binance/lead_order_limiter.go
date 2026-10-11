package binance

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"go_binance_futures/service/binanceapiusage"
)

// One Lead Portfolio Key is constrained to 20 order requests per 10 seconds.
// Reserve two slots for concurrent in-flight requests/cancellations and stop
// before sending HTTP; shared V4-5 global IP budget remains in place.
const leadOrderLimitPerTenSeconds = 18
const leadOpenLimitPerTenSeconds = 16

type leadMutationPriorityKey struct{}

// WithLeadSafetyExitPriority reserves per-key budget for protective CLOSE,
// STOP/TP or cancel mutations; the global IP coordinator still applies.
func WithLeadSafetyExitPriority(ctx context.Context) context.Context {
	return context.WithValue(ctx, leadMutationPriorityKey{}, true)
}

type leadOrderLimiter struct {
	mu    sync.Mutex
	times []time.Time
	base  http.RoundTripper
}

func (l *leadOrderLimiter) RoundTrip(r *http.Request) (*http.Response, error) {
	path := r.URL.Path
	mutation := (r.Method == http.MethodPost || r.Method == http.MethodDelete || r.Method == http.MethodPut) &&
		(strings.HasPrefix(path, "/fapi/v1/order") || strings.HasPrefix(path, "/fapi/v1/algoOrder"))
	if mutation {
		now := time.Now()
		l.mu.Lock()
		retained := l.times[:0]
		for _, at := range l.times {
			if now.Sub(at) < 10*time.Second {
				retained = append(retained, at)
			}
		}
		l.times = retained
		high, _ := r.Context().Value(leadMutationPriorityKey{}).(bool)
		if r.Method == http.MethodDelete {
			high = true
		}
		limit := leadOpenLimitPerTenSeconds
		if high {
			limit = leadOrderLimitPerTenSeconds
		}
		if len(l.times) >= limit {
			l.mu.Unlock()
			return nil, binanceapiusage.ErrBudgetDeferred
		}
		l.times = append(l.times, now)
		l.mu.Unlock()
	}
	return l.base.RoundTrip(r)
}

package binance

import (
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

type leadOrderLimiter struct {
	mu    sync.Mutex
	times []time.Time
	base  http.RoundTripper
}

func (l *leadOrderLimiter) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.Method == http.MethodPost || r.Method == http.MethodDelete {
		if strings.HasPrefix(r.URL.Path, "/fapi/v1/order") || strings.HasPrefix(r.URL.Path, "/fapi/v1/algoOrder") {
			now := time.Now()
			l.mu.Lock()
			kept := l.times[:0]
			for _, at := range l.times {
				if now.Sub(at) < 10*time.Second {
					kept = append(kept, at)
				}
			}
			l.times = kept
			if len(l.times) >= leadOrderLimitPerTenSeconds {
				l.mu.Unlock()
				return nil, binanceapiusage.ErrBudgetDeferred
			}
			l.times = append(l.times, now)
			l.mu.Unlock()
		}
	}
	return l.base.RoundTrip(r)
}

package binance

import (
	"net/http"
	"sync"
)

// Lead HTTP fault callbacks are account-scoped and receive ONLY status codes.
// Never pass raw signed request paths, Binance error bodies or credentials.
type leadRateLimitObserver struct {
	mu      sync.RWMutex
	handler func(int)
}

func (o *leadRateLimitObserver) Emit(code int) {
	if o == nil {
		return
	}
	o.mu.RLock()
	fn := o.handler
	o.mu.RUnlock()
	if fn != nil {
		fn(code)
	}
}

type leadHTTPProbe struct {
	base     http.RoundTripper
	observer *leadRateLimitObserver
}

func (p *leadHTTPProbe) RoundTrip(r *http.Request) (*http.Response, error) {
	response, err := p.base.RoundTrip(r)
	if response != nil && (response.StatusCode == 429 || response.StatusCode == 418) {
		p.observer.Emit(response.StatusCode)
	}
	return response, err
}
func (a *AccountClient) SetLeadRateLimitHandler(handler func(int)) {
	if a == nil || a.ID() != LeadAccountID || a.leadObserver == nil {
		return
	}
	a.leadObserver.mu.Lock()
	a.leadObserver.handler = handler
	a.leadObserver.mu.Unlock()
}

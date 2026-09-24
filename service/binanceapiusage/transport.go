package binanceapiusage

import (
	"context"
	"net/http"
	"strings"
	"time"
)

type TransportConfig struct {
	Product     string
	Environment string
	Source      string
	Collector   *Collector
	Budget      *BudgetCoordinator
}

type transport struct {
	base   http.RoundTripper
	config TransportConfig
}

type sourceContextKey struct{}

func WithSource(ctx context.Context, source string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	source = strings.TrimSpace(source)
	if source == "" {
		return ctx
	}
	return context.WithValue(ctx, sourceContextKey{}, source)
}

func SourceFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	value, _ := ctx.Value(sourceContextKey{}).(string)
	return strings.TrimSpace(value)
}

func WrapClient(client *http.Client, config TransportConfig) *http.Client {
	if client == nil {
		client = http.DefaultClient
	}
	cloned := *client
	base := client.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	if config.Collector == nil {
		config.Collector = Default()
	}
	if config.Budget == nil {
		config.Budget = DefaultBudget()
	}
	cloned.Transport = &transport{base: base, config: config}
	return &cloned
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	source := SourceFromContext(req.Context())
	if source == "" {
		source = t.config.Source
	}
	requestType := RequestType(t.config.Product, req.Method, req.URL.Path)
	weight := EstimateWeight(t.config.Product, req.Method, req.URL.Path, req.URL.Query())
	reservation, budgetErr := t.config.Budget.Reserve(
		req.Context(),
		t.config.Product,
		t.config.Environment,
		source,
		requestType,
		req.Method,
		req.URL.Path,
		weight,
	)
	if budgetErr != nil {
		return nil, budgetErr
	}
	defer reservation.Release()

	started := time.Now()
	resp, err := t.base.RoundTrip(req)
	finished := time.Now()

	status := 0
	headers := map[string]string{}
	retryAfter := ""
	if resp != nil {
		status = resp.StatusCode
		headers["used_weight_1m"] = resp.Header.Get("X-MBX-USED-WEIGHT-1M")
		headers["order_count_10s"] = resp.Header.Get("X-MBX-ORDER-COUNT-10S")
		headers["order_count_1m"] = resp.Header.Get("X-MBX-ORDER-COUNT-1M")
		retryAfter = resp.Header.Get("Retry-After")
	}

	event := RequestEvent{
		At:              finished.UnixMilli(),
		Product:         t.config.Product,
		Environment:     t.config.Environment,
		Source:          source,
		RequestType:     requestType,
		Method:          req.Method,
		Path:            req.URL.Path,
		StatusCode:      status,
		LatencyMs:       finished.Sub(started).Milliseconds(),
		EstimatedWeight: weight,
		Is429:           status == http.StatusTooManyRequests,
		Is418:           status == http.StatusTeapot,
	}
	if err != nil {
		// Do not persist err.Error(): transport errors can embed the full signed
		// URL. The observability path must never retain query strings/signatures.
		event.Error = "transport_error"
	}
	t.config.Collector.Record(event, headers, retryAfter)
	return resp, err
}

package binanceapiusage

import (
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	maxWindow       = 5 * time.Minute
	maxEvents       = 50_000
	maxRateLimitLog = 50
)

type RequestEvent struct {
	At              int64  `json:"at"`
	Product         string `json:"product"`
	Environment     string `json:"environment"`
	Source          string `json:"source"`
	RequestType     string `json:"request_type"`
	Method          string `json:"method"`
	Path            string `json:"path"`
	StatusCode      int    `json:"status_code"`
	LatencyMs       int64  `json:"latency_ms"`
	EstimatedWeight int64  `json:"estimated_weight"`
	Is429           bool   `json:"is_429"`
	Is418           bool   `json:"is_418"`
	Error           string `json:"error,omitempty"`
}

type RateLimitEvent struct {
	At          int64  `json:"at"`
	Product     string `json:"product"`
	Environment string `json:"environment"`
	Source      string `json:"source"`
	RequestType string `json:"request_type"`
	Method      string `json:"method"`
	Path        string `json:"path"`
	StatusCode  int    `json:"status_code"`
	RetryAfter  string `json:"retry_after,omitempty"`
}

type ExchangeLimitState struct {
	Product           string  `json:"product"`
	Environment       string  `json:"environment"`
	UsedWeight1m      int64   `json:"used_weight_1m"`
	OrderCount10s     int64   `json:"order_count_10s"`
	OrderCount1m      int64   `json:"order_count_1m"`
	WeightLimit1m     int64   `json:"weight_limit_1m"`
	WeightPercent1m   float64 `json:"weight_percent_1m"`
	LimitSource       string  `json:"limit_source"`
	RetryAfter        string  `json:"retry_after,omitempty"`
	LastStatusCode    int     `json:"last_status_code"`
	LastResponseAt    int64   `json:"last_response_at"`
	LastRateLimitedAt int64   `json:"last_rate_limited_at,omitempty"`
}

type WindowSummary struct {
	WindowSeconds    int64   `json:"window_seconds"`
	RequestCount     int64   `json:"request_count"`
	EstimatedWeight  int64   `json:"estimated_weight"`
	ErrorCount       int64   `json:"error_count"`
	Count429         int64   `json:"count_429"`
	Count418         int64   `json:"count_418"`
	AverageLatencyMs float64 `json:"average_latency_ms"`
	P95LatencyMs     int64   `json:"p95_latency_ms"`
}

type EndpointStat struct {
	Product          string  `json:"product"`
	Environment      string  `json:"environment"`
	Source           string  `json:"source"`
	RequestType      string  `json:"request_type"`
	Method           string  `json:"method"`
	Path             string  `json:"path"`
	Count            int64   `json:"count"`
	EstimatedWeight  int64   `json:"estimated_weight"`
	ErrorCount       int64   `json:"error_count"`
	Count429         int64   `json:"count_429"`
	Count418         int64   `json:"count_418"`
	AverageLatencyMs float64 `json:"average_latency_ms"`
	P95LatencyMs     int64   `json:"p95_latency_ms"`
	LastError        string  `json:"last_error,omitempty"`
	LastErrorAt      int64   `json:"last_error_at,omitempty"`
}

type SourceStat struct {
	Source          string `json:"source"`
	Count           int64  `json:"count"`
	EstimatedWeight int64  `json:"estimated_weight"`
	Count429        int64  `json:"count_429"`
	Count418        int64  `json:"count_418"`
}

type Snapshot struct {
	GeneratedAt          int64                `json:"generated_at"`
	Window10s            WindowSummary        `json:"window_10s"`
	Window1m             WindowSummary        `json:"window_1m"`
	Window5m             WindowSummary        `json:"window_5m"`
	ExchangeLimits       []ExchangeLimitState `json:"exchange_limits"`
	TopEndpointsByCount  []EndpointStat       `json:"top_endpoints_by_count"`
	TopEndpointsByWeight []EndpointStat       `json:"top_endpoints_by_weight"`
	Sources5m            []SourceStat         `json:"sources_5m"`
	RecentRateLimits     []RateLimitEvent     `json:"recent_rate_limits"`
	RetainedEvents       int                  `json:"retained_events"`
	DroppedEvents        uint64               `json:"dropped_events"`
	LastDroppedAt        int64                `json:"last_dropped_at,omitempty"`
	Truncated            bool                 `json:"truncated"`
}

type Collector struct {
	mu            sync.Mutex
	events        []RequestEvent
	eventHead     int
	droppedEvents uint64
	lastDroppedAt int64
	limits        map[string]ExchangeLimitState
	rateLimits    []RateLimitEvent
}

var defaultCollector = &Collector{limits: map[string]ExchangeLimitState{}}

func Default() *Collector {
	return defaultCollector
}

func (c *Collector) Record(event RequestEvent, headers map[string]string, retryAfter string) {
	if event.At <= 0 {
		event.At = time.Now().UnixMilli()
	}
	event.Product = normalizeLabel(event.Product, "unknown")
	event.Environment = normalizeLabel(event.Environment, "mainnet")
	event.Source = normalizeLabel(event.Source, "unknown")
	event.RequestType = normalizeLabel(event.RequestType, "read")
	event.Method = strings.ToUpper(strings.TrimSpace(event.Method))
	event.Path = normalizePath(event.Path)

	c.mu.Lock()
	defer c.mu.Unlock()

	c.appendEventLocked(event)

	key := event.Product + "|" + event.Environment
	state := c.limits[key]
	state.Product = event.Product
	state.Environment = event.Environment
	state.LastStatusCode = event.StatusCode
	state.LastResponseAt = event.At
	if value, ok := parseHeaderInt(headers["used_weight_1m"]); ok {
		state.UsedWeight1m = value
	}
	if value, ok := parseHeaderInt(headers["order_count_10s"]); ok {
		state.OrderCount10s = value
	}
	if value, ok := parseHeaderInt(headers["order_count_1m"]); ok {
		state.OrderCount1m = value
	}
	if state.WeightLimit1m <= 0 {
		state.WeightLimit1m = referenceWeightLimit(event.Product)
		if state.WeightLimit1m > 0 {
			state.LimitSource = "reference"
		}
	}
	if state.WeightLimit1m > 0 {
		state.WeightPercent1m = round2(float64(state.UsedWeight1m) / float64(state.WeightLimit1m) * 100)
	}
	if strings.TrimSpace(retryAfter) != "" {
		state.RetryAfter = strings.TrimSpace(retryAfter)
	}
	if event.Is429 || event.Is418 {
		state.LastRateLimitedAt = event.At
		item := RateLimitEvent{
			At: event.At, Product: event.Product, Environment: event.Environment,
			Source: event.Source, RequestType: event.RequestType, Method: event.Method, Path: event.Path,
			StatusCode: event.StatusCode, RetryAfter: strings.TrimSpace(retryAfter),
		}
		c.rateLimits = append(c.rateLimits, item)
		if len(c.rateLimits) > maxRateLimitLog {
			c.rateLimits = append([]RateLimitEvent(nil), c.rateLimits[len(c.rateLimits)-maxRateLimitLog:]...)
		}
	}
	c.limits[key] = state
}

func (c *Collector) SetWeightLimit(product, environment string, limit int64, source string) {
	if limit <= 0 {
		return
	}
	product = normalizeLabel(product, "unknown")
	environment = normalizeLabel(environment, "mainnet")
	source = normalizeLabel(source, "exchange_info")

	c.mu.Lock()
	defer c.mu.Unlock()

	key := product + "|" + environment
	state := c.limits[key]
	state.Product = product
	state.Environment = environment
	state.WeightLimit1m = limit
	state.LimitSource = source
	if state.UsedWeight1m > 0 {
		state.WeightPercent1m = round2(float64(state.UsedWeight1m) / float64(limit) * 100)
	}
	c.limits[key] = state
}

func (c *Collector) Snapshot(now time.Time) Snapshot {
	if now.IsZero() {
		now = time.Now()
	}
	nowMs := now.UnixMilli()

	c.mu.Lock()
	events := c.eventsInOrderLocked()
	retainedEvents := len(events)
	droppedEvents := c.droppedEvents
	lastDroppedAt := c.lastDroppedAt
	limits := make([]ExchangeLimitState, 0, len(c.limits))
	for _, state := range c.limits {
		limits = append(limits, state)
	}
	rateLimits := append([]RateLimitEvent(nil), c.rateLimits...)
	c.mu.Unlock()

	sort.Slice(limits, func(i, j int) bool {
		if limits[i].Product == limits[j].Product {
			return limits[i].Environment < limits[j].Environment
		}
		return limits[i].Product < limits[j].Product
	})
	sort.Slice(rateLimits, func(i, j int) bool { return rateLimits[i].At > rateLimits[j].At })

	return Snapshot{
		GeneratedAt:          nowMs,
		Window10s:            summarizeWindow(events, nowMs-10_000, 10),
		Window1m:             summarizeWindow(events, nowMs-60_000, 60),
		Window5m:             summarizeWindow(events, nowMs-maxWindow.Milliseconds(), 300),
		ExchangeLimits:       limits,
		TopEndpointsByCount:  endpointStats(events, nowMs-maxWindow.Milliseconds(), "count", 20),
		TopEndpointsByWeight: endpointStats(events, nowMs-maxWindow.Milliseconds(), "weight", 20),
		Sources5m:            sourceStats(events, nowMs-maxWindow.Milliseconds()),
		RecentRateLimits:     rateLimits,
		RetainedEvents:       retainedEvents,
		DroppedEvents:        droppedEvents,
		LastDroppedAt:        lastDroppedAt,
		Truncated:            lastDroppedAt >= nowMs-maxWindow.Milliseconds(),
	}
}

func (c *Collector) ResetForTest() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = nil
	c.eventHead = 0
	c.droppedEvents = 0
	c.lastDroppedAt = 0
	c.limits = map[string]ExchangeLimitState{}
	c.rateLimits = nil
}

func (c *Collector) appendEventLocked(event RequestEvent) {
	if len(c.events) < maxEvents {
		c.events = append(c.events, event)
		return
	}

	overwritten := c.events[c.eventHead]
	if overwritten.At >= event.At-maxWindow.Milliseconds() {
		c.droppedEvents++
		c.lastDroppedAt = event.At
	}
	c.events[c.eventHead] = event
	c.eventHead++
	if c.eventHead >= len(c.events) {
		c.eventHead = 0
	}
}

func (c *Collector) eventsInOrderLocked() []RequestEvent {
	if len(c.events) == 0 {
		return nil
	}
	result := make([]RequestEvent, len(c.events))
	if c.eventHead == 0 || len(c.events) < maxEvents {
		copy(result, c.events)
		return result
	}
	n := copy(result, c.events[c.eventHead:])
	copy(result[n:], c.events[:c.eventHead])
	return result
}

func summarizeWindow(events []RequestEvent, cutoff int64, seconds int64) WindowSummary {
	result := WindowSummary{WindowSeconds: seconds}
	latencies := make([]int64, 0)
	var latencyTotal int64
	for _, event := range events {
		if event.At < cutoff {
			continue
		}
		result.RequestCount++
		result.EstimatedWeight += event.EstimatedWeight
		if event.Error != "" || event.StatusCode >= 400 || event.StatusCode == 0 {
			result.ErrorCount++
		}
		if event.Is429 {
			result.Count429++
		}
		if event.Is418 {
			result.Count418++
		}
		latencies = append(latencies, event.LatencyMs)
		latencyTotal += event.LatencyMs
	}
	if result.RequestCount > 0 {
		result.AverageLatencyMs = round2(float64(latencyTotal) / float64(result.RequestCount))
		result.P95LatencyMs = percentile95(latencies)
	}
	return result
}

func endpointStats(events []RequestEvent, cutoff int64, sortBy string, limit int) []EndpointStat {
	type aggregate struct {
		stat      EndpointStat
		latencies []int64
		latency   int64
	}
	byKey := map[string]*aggregate{}
	for _, event := range events {
		if event.At < cutoff {
			continue
		}
		key := strings.Join([]string{event.Product, event.Environment, event.Source, event.RequestType, event.Method, event.Path}, "|")
		row := byKey[key]
		if row == nil {
			row = &aggregate{stat: EndpointStat{
				Product: event.Product, Environment: event.Environment, Source: event.Source,
				RequestType: event.RequestType, Method: event.Method, Path: event.Path,
			}}
			byKey[key] = row
		}
		row.stat.Count++
		row.stat.EstimatedWeight += event.EstimatedWeight
		row.latencies = append(row.latencies, event.LatencyMs)
		row.latency += event.LatencyMs
		if event.Error != "" || event.StatusCode >= 400 || event.StatusCode == 0 {
			row.stat.ErrorCount++
			row.stat.LastError = event.Error
			if row.stat.LastError == "" && event.StatusCode >= 400 {
				row.stat.LastError = "HTTP " + strconv.Itoa(event.StatusCode)
			}
			row.stat.LastErrorAt = event.At
		}
		if event.Is429 {
			row.stat.Count429++
		}
		if event.Is418 {
			row.stat.Count418++
		}
	}
	rows := make([]EndpointStat, 0, len(byKey))
	for _, item := range byKey {
		if item.stat.Count > 0 {
			item.stat.AverageLatencyMs = round2(float64(item.latency) / float64(item.stat.Count))
			item.stat.P95LatencyMs = percentile95(item.latencies)
		}
		rows = append(rows, item.stat)
	}
	sort.Slice(rows, func(i, j int) bool {
		var left, right int64
		if sortBy == "weight" {
			left, right = rows[i].EstimatedWeight, rows[j].EstimatedWeight
		} else {
			left, right = rows[i].Count, rows[j].Count
		}
		if left != right {
			return left > right
		}
		if rows[i].Count != rows[j].Count {
			return rows[i].Count > rows[j].Count
		}
		return rows[i].Path < rows[j].Path
	})
	if limit > 0 && len(rows) > limit {
		rows = rows[:limit]
	}
	return rows
}

func sourceStats(events []RequestEvent, cutoff int64) []SourceStat {
	bySource := map[string]SourceStat{}
	for _, event := range events {
		if event.At < cutoff {
			continue
		}
		row := bySource[event.Source]
		row.Source = event.Source
		row.Count++
		row.EstimatedWeight += event.EstimatedWeight
		if event.Is429 {
			row.Count429++
		}
		if event.Is418 {
			row.Count418++
		}
		bySource[event.Source] = row
	}
	rows := make([]SourceStat, 0, len(bySource))
	for _, row := range bySource {
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].EstimatedWeight != rows[j].EstimatedWeight {
			return rows[i].EstimatedWeight > rows[j].EstimatedWeight
		}
		return rows[i].Source < rows[j].Source
	})
	return rows
}

func percentile95(values []int64) int64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]int64(nil), values...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	index := int(math.Ceil(float64(len(sorted))*0.95)) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}

func parseHeaderInt(value string) (int64, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	return parsed, err == nil
}

func normalizeLabel(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func normalizePath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "/"
	}
	if idx := strings.IndexByte(value, '?'); idx >= 0 {
		value = value[:idx]
	}
	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	return value
}

func referenceWeightLimit(product string) int64 {
	switch strings.ToLower(strings.TrimSpace(product)) {
	case "spot":
		return 6000
	case "futures", "delivery":
		return 2400
	default:
		return 0
	}
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}

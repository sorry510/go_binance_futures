package binanceapiusage

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type BudgetPriority string
type BudgetLevel string

const (
	PriorityP0 BudgetPriority = "P0"
	PriorityP1 BudgetPriority = "P1"
	PriorityP2 BudgetPriority = "P2"
	PriorityP3 BudgetPriority = "P3"
)

const (
	BudgetNormal            BudgetLevel = "normal"
	BudgetWarning           BudgetLevel = "warning"
	BudgetCritical          BudgetLevel = "critical"
	BudgetExchangeThrottled BudgetLevel = "exchange_throttled"
)

const (
	budgetWarningPercent  = 70.0
	budgetCriticalPercent = 85.0
	budgetStaleAfter      = 75 * time.Second
)

var ErrBudgetDeferred = errors.New("binance API request deferred by global budget")

type BudgetSnapshot struct {
	Product          string      `json:"product"`
	Environment      string      `json:"environment"`
	Level            BudgetLevel `json:"level"`
	UsedWeight1m     int64       `json:"used_weight_1m"`
	PendingWeight    int64       `json:"pending_weight"`
	WeightLimit1m    int64       `json:"weight_limit_1m"`
	EffectivePercent float64     `json:"effective_percent"`
	OrderCount10s    int64       `json:"order_count_10s"`
	OrderLimit10s    int64       `json:"order_limit_10s"`
	OrderCount1m     int64       `json:"order_count_1m"`
	OrderLimit1m     int64       `json:"order_limit_1m"`
	LastResponseAt   int64       `json:"last_response_at"`
	ThrottleUntil    int64       `json:"throttle_until,omitempty"`
}

type budgetKeyState struct {
	pendingWeight int64
	throttleUntil time.Time
}

type BudgetCoordinator struct {
	mu        sync.Mutex
	states    map[string]*budgetKeyState
	lowSem    chan struct{}
	collector *Collector
}

type BudgetReservation struct {
	coordinator *BudgetCoordinator
	key         string
	weight      int64
	lowSlot     bool
	once        sync.Once
}

var defaultBudget = newBudgetCoordinator(Default())

func newBudgetCoordinator(collector *Collector) *BudgetCoordinator {
	if collector == nil {
		collector = Default()
	}
	return &BudgetCoordinator{
		states:    map[string]*budgetKeyState{},
		lowSem:    make(chan struct{}, 1),
		collector: collector,
	}
}

func DefaultBudget() *BudgetCoordinator { return defaultBudget }

func (c *BudgetCoordinator) resetForTest() {
	c.mu.Lock()
	c.states = map[string]*budgetKeyState{}
	c.lowSem = make(chan struct{}, 1)
	c.mu.Unlock()
}

func (c *BudgetCoordinator) Reserve(ctx context.Context, product, environment, source, requestType, method, path string, weight int64) (*BudgetReservation, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if weight < 1 {
		weight = 1
	}
	product = normalizeLabel(product, "unknown")
	environment = normalizeLabel(environment, "mainnet")
	source = normalizeLabel(source, "unknown")
	path = normalizePath(path)
	priority := classifyBudgetPriority(source, requestType, method, path)
	key := product + "|" + environment

	c.mu.Lock()
	state, signal, level, effectivePercent := c.evaluateLocked(key, product, environment, weight)
	if rejectErr := budgetRejectError(product, method, path, priority, signal, state, level, effectivePercent); rejectErr != nil {
		c.mu.Unlock()
		RecordOptimization(source, "deferred", 1)
		return nil, rejectErr
	}
	needLowSlot := level == BudgetWarning && (priority == PriorityP2 || priority == PriorityP3)
	if !needLowSlot {
		state.pendingWeight += weight
		c.mu.Unlock()
		return &BudgetReservation{coordinator: c, key: key, weight: weight}, nil
	}
	c.mu.Unlock()

	// Low-priority work queues outside the budget mutex. Once it actually owns
	// the slot, re-read current Binance headers and pending weight before send;
	// a request admitted during warning must not blindly run after the account
	// has moved into critical/throttled state while it was waiting.
	select {
	case c.lowSem <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	c.mu.Lock()
	state, signal, level, effectivePercent = c.evaluateLocked(key, product, environment, weight)
	if rejectErr := budgetRejectError(product, method, path, priority, signal, state, level, effectivePercent); rejectErr != nil {
		c.mu.Unlock()
		<-c.lowSem
		RecordOptimization(source, "deferred", 1)
		return nil, rejectErr
	}
	state.pendingWeight += weight
	c.mu.Unlock()
	return &BudgetReservation{coordinator: c, key: key, weight: weight, lowSlot: true}, nil
}

func (c *BudgetCoordinator) evaluateLocked(key, product, environment string, weight int64) (*budgetKeyState, ExchangeLimitState, BudgetLevel, float64) {
	state := c.states[key]
	if state == nil {
		state = &budgetKeyState{}
		c.states[key] = state
	}
	signal := c.collector.limitState(product, environment)
	now := time.Now()
	if signal.LastResponseAt > 0 && now.Sub(time.UnixMilli(signal.LastResponseAt)) > budgetStaleAfter {
		signal.UsedWeight1m = 0
		signal.WeightPercent1m = 0
		signal.OrderCount10s = 0
		signal.OrderCount1m = 0
	}
	if signal.LastRateLimitedAt > 0 && signal.LastRateLimitedAt >= signal.LastResponseAt {
		until := time.UnixMilli(signal.LastRateLimitedAt).Add(retryAfterDuration(signal.RetryAfter, signal.LastStatusCode))
		if until.After(state.throttleUntil) {
			state.throttleUntil = until
		}
	}
	level, effectivePercent := budgetLevel(signal, state.pendingWeight+weight, state.throttleUntil, now)
	return state, signal, level, effectivePercent
}

func budgetRejectError(product, method, path string, priority BudgetPriority, signal ExchangeLimitState, state *budgetKeyState, level BudgetLevel, effectivePercent float64) error {
	if level == BudgetExchangeThrottled {
		return fmt.Errorf("%w: exchange throttled until %s", ErrBudgetDeferred, state.throttleUntil.Format(time.RFC3339))
	}
	if isOrderMutation(product, method, path) && orderBudgetExhausted(signal) {
		return fmt.Errorf("%w: order-count budget is near exchange limit", ErrBudgetDeferred)
	}
	if level == BudgetCritical && (priority == PriorityP2 || priority == PriorityP3) {
		return fmt.Errorf("%w: %s priority=%s effective_weight=%.1f%%", ErrBudgetDeferred, level, priority, effectivePercent)
	}
	return nil
}

func (r *BudgetReservation) Release() {
	if r == nil || r.coordinator == nil {
		return
	}
	r.once.Do(func() {
		c := r.coordinator
		c.mu.Lock()
		if state := c.states[r.key]; state != nil {
			state.pendingWeight -= r.weight
			if state.pendingWeight < 0 {
				state.pendingWeight = 0
			}
		}
		c.mu.Unlock()
		if r.lowSlot {
			<-c.lowSem
		}
	})
}

func (c *BudgetCoordinator) Snapshot(product, environment string) BudgetSnapshot {
	product = normalizeLabel(product, "unknown")
	environment = normalizeLabel(environment, "mainnet")
	key := product + "|" + environment
	now := time.Now()

	c.mu.Lock()
	state := c.states[key]
	pending := int64(0)
	throttleUntil := time.Time{}
	if state != nil {
		pending = state.pendingWeight
		throttleUntil = state.throttleUntil
	}
	c.mu.Unlock()

	signal := c.collector.limitState(product, environment)
	if signal.LastResponseAt > 0 && now.Sub(time.UnixMilli(signal.LastResponseAt)) > budgetStaleAfter {
		signal.UsedWeight1m = 0
		signal.WeightPercent1m = 0
		signal.OrderCount10s = 0
		signal.OrderCount1m = 0
	}
	level, percent := budgetLevel(signal, pending, throttleUntil, now)
	return BudgetSnapshot{
		Product: product, Environment: environment, Level: level,
		UsedWeight1m: signal.UsedWeight1m, PendingWeight: pending, WeightLimit1m: signal.WeightLimit1m,
		EffectivePercent: round2(percent), OrderCount10s: signal.OrderCount10s, OrderLimit10s: signal.OrderLimit10s,
		OrderCount1m: signal.OrderCount1m, OrderLimit1m: signal.OrderLimit1m, LastResponseAt: signal.LastResponseAt,
		ThrottleUntil: unixMilliOrZero(throttleUntil),
	}
}

func budgetLevel(signal ExchangeLimitState, pendingWeight int64, throttleUntil time.Time, now time.Time) (BudgetLevel, float64) {
	if throttleUntil.After(now) {
		return BudgetExchangeThrottled, signal.WeightPercent1m
	}
	percent := signal.WeightPercent1m
	if signal.WeightLimit1m > 0 {
		percent = float64(signal.UsedWeight1m+pendingWeight) / float64(signal.WeightLimit1m) * 100
	}
	switch {
	case percent >= budgetCriticalPercent:
		return BudgetCritical, percent
	case percent >= budgetWarningPercent:
		return BudgetWarning, percent
	default:
		return BudgetNormal, percent
	}
}

func classifyBudgetPriority(source, requestType, method, path string) BudgetPriority {
	source = strings.ToLower(strings.TrimSpace(source))
	requestType = strings.ToLower(strings.TrimSpace(requestType))
	method = strings.ToUpper(strings.TrimSpace(method))
	path = normalizePath(path)
	if requestType == "trade" {
		return PriorityP0
	}
	if source == "ownership_reconcile" || strings.Contains(source, "execution_uncertain") {
		return PriorityP0
	}
	if strings.Contains(path, "/account") || strings.Contains(path, "/positionRisk") || strings.HasSuffix(path, "/openOrders") || strings.HasSuffix(path, "/order") || strings.HasSuffix(path, "/algoOrder") || strings.HasSuffix(path, "/listenKey") {
		return PriorityP1
	}
	if source == "historical_market" || source == "system_health" || strings.Contains(source, "exchange_info") || strings.Contains(path, "/exchangeInfo") {
		return PriorityP3
	}
	if method == http.MethodGet {
		return PriorityP2
	}
	return PriorityP2
}

func retryAfterDuration(value string, status int) time.Duration {
	value = strings.TrimSpace(value)
	if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	if status == http.StatusTeapot {
		return time.Minute
	}
	return 5 * time.Second
}

func orderBudgetExhausted(state ExchangeLimitState) bool {
	if state.OrderLimit10s > 0 && float64(state.OrderCount10s)/float64(state.OrderLimit10s) >= 0.95 {
		return true
	}
	if state.OrderLimit1m > 0 && float64(state.OrderCount1m)/float64(state.OrderLimit1m) >= 0.95 {
		return true
	}
	return false
}

func isOrderMutation(product, method, path string) bool {
	method = strings.ToUpper(strings.TrimSpace(method))
	if method != http.MethodPost && method != http.MethodDelete {
		return false
	}
	path = normalizePath(path)
	switch strings.ToLower(strings.TrimSpace(product)) {
	case "futures":
		return path == "/fapi/v1/order" || path == "/fapi/v1/algoOrder"
	case "spot":
		return path == "/api/v3/order"
	case "delivery":
		return strings.HasSuffix(path, "/order")
	default:
		return false
	}
}

func unixMilliOrZero(value time.Time) int64 {
	if value.IsZero() {
		return 0
	}
	return value.UnixMilli()
}

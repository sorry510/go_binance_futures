package leadaccount

import (
	"context"
	"errors"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	binance "go_binance_futures/feature/api/binance"
)

// LeadAccountMirror is a read-only, ephemeral account-scoped WS projection.
// It is NEVER a substitute for full REST + durable Ownership reconciliation.
type LeadAccountMirror struct {
	mu            sync.RWMutex
	generation    uint64
	ready         bool
	capturedAt    time.Time
	lastEvent     time.Time
	positions     map[string]RiskPosition
	orders        map[int64]mirrorOrder
	orderEvidence map[int64]mirrorOrder
	positionAt    map[string]int64
	balanceAt     map[string]int64
	balances      map[string]float64
	metrics       LeadMirrorMetrics
}
type mirrorOrder struct {
	Symbol   string
	Side     string
	Quantity float64
	Filled   float64
	Price    float64
	Status   string
	At       int64
}
type LeadMirrorMetrics struct {
	RestSnapshots  uint64 `json:"rest_snapshots"`
	WSUpdates      uint64 `json:"ws_updates"`
	RejectedEvents uint64 `json:"rejected_events"`
}
type LeadMirrorStatus struct {
	AccountID     binance.AccountID `json:"account_id"`
	Generation    uint64            `json:"generation"`
	Ready         bool              `json:"ready"`
	CapturedAt    time.Time         `json:"captured_at"`
	LastEvent     time.Time         `json:"last_event"`
	PositionCount int               `json:"position_count"`
	OrderCount    int               `json:"order_count"`
	Metrics       LeadMirrorMetrics `json:"metrics"`
}

func NewLeadAccountMirror() *LeadAccountMirror {
	return &LeadAccountMirror{
		positions:     map[string]RiskPosition{},
		orders:        map[int64]mirrorOrder{},
		orderEvidence: map[int64]mirrorOrder{},
		positionAt:    map[string]int64{},
		balanceAt:     map[string]int64{},
		balances:      map[string]float64{},
	}
}

type LeadMirrorRESTReader interface {
	GetFuturesAccountContext(context.Context) (*futures.Account, error)
	GetPositionFreshContext(context.Context, binance.PositionParams) ([]*futures.PositionRisk, error)
	GetOpenOrderFreshContext(context.Context, ...string) ([]*futures.Order, error)
}

var ErrLeadMirrorUntrusted = errors.New("Lead mirror requires a complete Lead REST snapshot")

// BeginGeneration immediately makes old-generation WS data stale. The mirror
// never becomes ready from events alone; it requires fresh REST re-seeding.
func (m *LeadAccountMirror) BeginGeneration() uint64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.generation++
	m.ready = false
	m.positions = make(map[string]RiskPosition)
	m.orders = make(map[int64]mirrorOrder)
	m.orderEvidence = make(map[int64]mirrorOrder)
	m.positionAt = make(map[string]int64)
	m.balanceAt = make(map[string]int64)
	m.balances = make(map[string]float64)
	m.capturedAt = time.Time{}
	m.lastEvent = time.Time{}
	return m.generation
}
func (m *LeadAccountMirror) Invalidate(generation uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.generation == generation {
		m.ready = false
	}
}

// Seed performs a complete account-level snapshot; no partial state is
// promoted. Empty position/order slices are legitimate, nil account is not.
func (m *LeadAccountMirror) Seed(ctx context.Context, generation uint64, source LeadMirrorRESTReader) error {
	if m == nil || source == nil {
		return ErrLeadMirrorUntrusted
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	// A failed attempted refresh must never leave an older same-generation
	// snapshot marked authoritative.
	m.mu.Lock()
	if m.generation != generation {
		m.mu.Unlock()
		return ErrLeadMirrorUntrusted
	}
	m.ready = false
	m.mu.Unlock()
	account, err := source.GetFuturesAccountContext(ctx)
	if err != nil || account == nil || !account.CanTrade || account.MultiAssetsMargin {
		return ErrLeadMirrorUntrusted
	}
	positions, err := source.GetPositionFreshContext(ctx, binance.PositionParams{})
	if err != nil {
		return ErrLeadMirrorUntrusted
	}
	orders, err := source.GetOpenOrderFreshContext(ctx)
	if err != nil {
		return ErrLeadMirrorUntrusted
	}
	newPositions := make(map[string]RiskPosition)
	newOrders := make(map[int64]mirrorOrder)
	newBalances := make(map[string]float64)
	hasUSDT := false
	for _, a := range account.Assets {
		if a == nil {
			continue
		}
		value, e := strconv.ParseFloat(a.WalletBalance, 64)
		if e != nil || math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
			return ErrLeadMirrorUntrusted
		}
		newBalances[a.Asset] = value
		if a.Asset == "USDT" {
			hasUSDT = true
		}
	}
	if !hasUSDT {
		return ErrLeadMirrorUntrusted
	}
	for _, p := range positions {
		if p == nil {
			return ErrLeadMirrorUntrusted
		}
		q, e := strconv.ParseFloat(p.PositionAmt, 64)
		if e != nil || math.IsNaN(q) || math.IsInf(q, 0) {
			return ErrLeadMirrorUntrusted
		}
		if math.Abs(q) < 1e-12 {
			continue
		}
		price, e1 := strconv.ParseFloat(p.MarkPrice, 64)
		pnl, e2 := strconv.ParseFloat(p.UnRealizedProfit, 64)
		leverage, e3 := strconv.ParseInt(p.Leverage, 10, 64)
		side := strings.ToUpper(p.PositionSide)
		if e1 != nil || e2 != nil || e3 != nil || !riskFinitePositive(price) || math.IsNaN(pnl) || math.IsInf(pnl, 0) || leverage <= 0 ||
			(side != "LONG" && side != "SHORT") || p.Symbol == "" {
			return ErrLeadMirrorUntrusted
		}
		newPositions[riskSlot(p.Symbol, side)] = RiskPosition{Symbol: p.Symbol, Side: side, Quantity: q, MarkPrice: price, UnrealizedPNL: pnl, Leverage: leverage}
	}
	for _, o := range orders {
		if o == nil || o.OrderID <= 0 || o.Symbol == "" {
			return ErrLeadMirrorUntrusted
		}
		orig, e1 := strconv.ParseFloat(o.OrigQuantity, 64)
		filled, e2 := strconv.ParseFloat(o.ExecutedQuantity, 64)
		price, e3 := strconv.ParseFloat(o.Price, 64)
		if e1 != nil || e2 != nil || e3 != nil || !riskFinitePositive(orig) || !riskFiniteNonnegative(filled) || filled > orig || !riskFiniteNonnegative(price) {
			return ErrLeadMirrorUntrusted
		}
		newOrders[o.OrderID] = mirrorOrder{Symbol: o.Symbol, Side: string(o.PositionSide), Quantity: orig, Filled: filled, Price: price, Status: string(o.Status)}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if generation != m.generation {
		return ErrLeadMirrorUntrusted
	}
	m.positions, m.orders, m.balances = newPositions, newOrders, newBalances
	m.orderEvidence = make(map[int64]mirrorOrder, len(newOrders))
	m.positionAt = make(map[string]int64)
	m.balanceAt = make(map[string]int64)
	for k, v := range newOrders {
		m.orderEvidence[k] = v
	}
	m.capturedAt = time.Now().UTC()
	m.lastEvent = time.Time{}
	m.ready = true
	m.metrics.RestSnapshots++
	return nil
}

// ApplyEvent uses account-bound generation and per-order monotonic cumulative
// fill evidence. A stale/invalid event cannot rewind a newer fill or mutate
// Main's private state. It deliberately performs zero DB writes.
func (m *LeadAccountMirror) ApplyEvent(generation uint64, event *futures.WsUserDataEvent) bool {
	if m == nil || event == nil {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	reject := func() bool { m.metrics.RejectedEvents++; return false }
	if !m.ready || m.generation != generation || event.Time <= 0 {
		return reject()
	}
	when := time.UnixMilli(event.Time).UTC()
	if !m.lastEvent.IsZero() && when.Before(m.lastEvent.Add(-time.Minute)) {
		return reject()
	}
	switch event.Event {
	case futures.UserDataEventTypeAccountUpdate:
		for _, p := range event.AccountUpdate.Positions {
			if p.Symbol == "" || (p.Side != futures.PositionSideTypeLong && p.Side != futures.PositionSideTypeShort) {
				return reject()
			}
			qty, e := strconv.ParseFloat(p.Amount, 64)
			if e != nil || math.IsNaN(qty) || math.IsInf(qty, 0) {
				return reject()
			}
		}
		for _, b := range event.AccountUpdate.Balances {
			qty, e := strconv.ParseFloat(b.Balance, 64)
			if b.Asset == "" || e != nil || !riskFiniteNonnegative(qty) {
				return reject()
			}
			if event.Time < m.balanceAt[b.Asset] {
				return reject()
			}
		}
		for _, p := range event.AccountUpdate.Positions {
			if event.Time < m.positionAt[riskSlot(p.Symbol, string(p.Side))] {
				return reject()
			}
		}
		for _, p := range event.AccountUpdate.Positions {
			qty, _ := strconv.ParseFloat(p.Amount, 64)
			key := riskSlot(p.Symbol, string(p.Side))
			m.positionAt[key] = event.Time
			if math.Abs(qty) < 1e-12 {
				delete(m.positions, key)
				continue
			}
			prev := m.positions[key]
			prev.Symbol, prev.Side, prev.Quantity = p.Symbol, string(p.Side), qty
			if price, e := strconv.ParseFloat(p.MarkPrice, 64); e == nil && riskFinitePositive(price) {
				prev.MarkPrice = price
			}
			if pnl, e := strconv.ParseFloat(p.UnrealizedPnL, 64); e == nil && !math.IsNaN(pnl) && !math.IsInf(pnl, 0) {
				prev.UnrealizedPNL = pnl
			}
			m.positions[key] = prev
		}
		for _, b := range event.AccountUpdate.Balances {
			qty, e := strconv.ParseFloat(b.Balance, 64)
			if e != nil || !riskFiniteNonnegative(qty) {
				return reject()
			}
			m.balances[b.Asset] = qty
			m.balanceAt[b.Asset] = event.Time
		}
	case futures.UserDataEventTypeOrderTradeUpdate:
		o := event.OrderTradeUpdate
		if o.ID <= 0 || o.Symbol == "" || o.ClientOrderID == "" {
			return reject()
		}
		cum, e := strconv.ParseFloat(o.AccumulatedFilledQty, 64)
		if e != nil || !riskFiniteNonnegative(cum) {
			return reject()
		}
		prev, exists := m.orderEvidence[o.ID]
		if exists {
			if prev.Symbol != o.Symbol || cum+1e-12 < prev.Filled || event.Time < prev.At {
				return reject()
			}
			if terminalOrderStatus(prev.Status) && !terminalOrderStatus(string(o.Status)) {
				return reject()
			}
		}
		qty, e := strconv.ParseFloat(o.OriginalQty, 64)
		if e != nil || !riskFinitePositive(qty) || cum > qty+1e-12 {
			return reject()
		}
		price, _ := strconv.ParseFloat(o.OriginalPrice, 64)
		if !riskFiniteNonnegative(price) {
			return reject()
		}
		updated := mirrorOrder{Symbol: o.Symbol, Side: string(o.PositionSide), Quantity: qty, Filled: cum, Price: price, Status: string(o.Status), At: event.Time}
		m.orderEvidence[o.ID] = updated
		if terminalOrderStatus(updated.Status) {
			delete(m.orders, o.ID)
		} else {
			m.orders[o.ID] = updated
		}
	case "ALGO_UPDATE", "CONDITIONAL_ORDER_TRIGGER_REJECT":
		// Algo status never authorizes a recovered ordinary fill on its own.
		return reject()
	default:
		return reject()
	}
	if when.After(m.lastEvent) {
		m.lastEvent = when
	}
	m.metrics.WSUpdates++
	return true
}
func terminalOrderStatus(s string) bool {
	switch strings.ToUpper(s) {
	case "FILLED", "CANCELED", "EXPIRED", "REJECTED", "EXPIRED_IN_MATCH":
		return true
	}
	return false
}
func (m *LeadAccountMirror) Status() LeadMirrorStatus {
	if m == nil {
		return LeadMirrorStatus{AccountID: binance.LeadAccountID}
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return LeadMirrorStatus{AccountID: binance.LeadAccountID, Generation: m.generation, Ready: m.ready, CapturedAt: m.capturedAt, LastEvent: m.lastEvent, PositionCount: len(m.positions), OrderCount: len(m.orders), Metrics: m.metrics}
}

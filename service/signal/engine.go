package signal

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"
	"sync/atomic"

	agentevent "go_binance_futures/agent/event"
)

type SettingsProvider func() Settings

type Emitter func(Signal) bool

type EngineStats struct {
	Events             uint64 `json:"events"`
	FastMoveSignals    uint64 `json:"fast_move_signals"`
	LiquidationSignals uint64 `json:"liquidation_signals"`
	SignalsDropped     uint64 `json:"signals_dropped"`
	TrackedSymbols     int    `json:"tracked_symbols"`
}

type pricePoint struct {
	TimeMs int64
	Price  float64
}

type fastMoveState struct {
	LastSignalMs int64
	Armed        bool
}

type liquidationPoint struct {
	EventID  string
	TimeMs   int64
	Notional float64
}

type Engine struct {
	bus                *agentevent.Bus
	settings           SettingsProvider
	emit               Emitter
	mu                 sync.Mutex
	prices             map[string][]pricePoint
	fastMoveStates     map[string]map[string]*fastMoveState
	liquidations       map[string][]liquidationPoint
	liquidationLast    map[string]int64
	events             atomic.Uint64
	fastMoveSignals    atomic.Uint64
	liquidationSignals atomic.Uint64
	signalsDropped     atomic.Uint64
}

func NewEngine(bus *agentevent.Bus, settings SettingsProvider, emit Emitter) (*Engine, error) {
	if bus == nil {
		return nil, fmt.Errorf("signal engine requires event bus")
	}
	if settings == nil {
		settings = func() Settings { return DefaultSettings() }
	}
	if emit == nil {
		return nil, fmt.Errorf("signal engine requires emitter")
	}
	engine := &Engine{
		bus:             bus,
		settings:        settings,
		emit:            emit,
		prices:          make(map[string][]pricePoint),
		fastMoveStates:  make(map[string]map[string]*fastMoveState),
		liquidations:    make(map[string][]liquidationPoint),
		liquidationLast: make(map[string]int64),
	}
	if err := bus.Subscribe(agentevent.TypePriceTick, engine.handleEvent); err != nil {
		return nil, err
	}
	if err := bus.Subscribe(agentevent.TypeLiquidation, engine.handleEvent); err != nil {
		return nil, err
	}
	return engine, nil
}

func (engine *Engine) handleEvent(ctx context.Context, value agentevent.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	engine.events.Add(1)
	var signals []Signal
	switch item := value.(type) {
	case agentevent.PriceTickEvent:
		signals = engine.evaluatePriceTick(item, engine.settings())
	case *agentevent.PriceTickEvent:
		if item != nil {
			signals = engine.evaluatePriceTick(*item, engine.settings())
		}
	case agentevent.LiquidationEvent:
		signals = engine.evaluateLiquidation(item, engine.settings())
	case *agentevent.LiquidationEvent:
		if item != nil {
			signals = engine.evaluateLiquidation(*item, engine.settings())
		}
	}
	for _, current := range signals {
		if !engine.emit(current) {
			engine.signalsDropped.Add(1)
		}
	}
	return nil
}

func (engine *Engine) evaluatePriceTick(item agentevent.PriceTickEvent, settings Settings) []Signal {
	if !settings.FastMoveEnabled || item.Price <= 0 || !strings.HasSuffix(item.Meta.Symbol, "USDT") {
		return nil
	}
	if settings.FastMoveExcludedSymbols == nil {
		settings.FastMoveExcludedSymbols = DefaultSettings().FastMoveExcludedSymbols
	}
	if settings.FastMoveExcludedSymbols[item.Meta.Symbol] {
		return nil
	}
	windows := settings.FastMoveWindows
	if len(windows) == 0 {
		windows = DefaultSettings().FastMoveWindows
	}
	threshold := settings.FastMoveThresholdPct
	if threshold <= 0 {
		threshold = DefaultSettings().FastMoveThresholdPct
	}
	recoverPct := settings.FastMoveRecoverPct
	if recoverPct < 0 || recoverPct >= threshold {
		recoverPct = math.Max(0, threshold-2)
	}
	cooldown := settings.FastMoveCooldown
	if cooldown <= 0 {
		cooldown = DefaultSettings().FastMoveCooldown
	}
	maxWindow := windows[0].Duration
	for _, window := range windows[1:] {
		if window.Duration > maxWindow {
			maxWindow = window.Duration
		}
	}

	engine.mu.Lock()
	defer engine.mu.Unlock()
	symbol := item.Meta.Symbol
	history := append(engine.prices[symbol], pricePoint{TimeMs: item.Meta.EventTime, Price: item.Price})
	cutoff := item.Meta.EventTime - maxWindow.Milliseconds()
	history = trimPriceHistory(history, cutoff)
	engine.prices[symbol] = history
	if engine.fastMoveStates[symbol] == nil {
		engine.fastMoveStates[symbol] = make(map[string]*fastMoveState)
	}
	result := make([]Signal, 0, len(windows))
	for _, window := range windows {
		if window.Duration <= 0 {
			continue
		}
		basePrice, ok := findBasePrice(history, item.Meta.EventTime-window.Duration.Milliseconds())
		if !ok || basePrice <= 0 {
			continue
		}
		changePct := (item.Price - basePrice) / basePrice * 100
		absChange := math.Abs(changePct)
		state := engine.fastMoveStates[symbol][window.Name]
		if state == nil {
			state = &fastMoveState{Armed: true}
			engine.fastMoveStates[symbol][window.Name] = state
		}
		if absChange <= recoverPct {
			state.Armed = true
		}
		if absChange < threshold || !state.Armed {
			continue
		}
		if state.LastSignalMs > 0 && item.Meta.EventTime-state.LastSignalMs < cooldown.Milliseconds() {
			continue
		}
		direction := "up"
		if changePct < 0 {
			direction = "down"
		}
		current := NewSignal(item.Meta.EventID, symbol, TypeFastMove, SeverityForRatio(absChange/threshold), window.Name)
		current.CreatedAt = item.Meta.EventTime
		current.Metrics = map[string]float64{
			"price": item.Price, "base_price": basePrice, "change_percent": changePct,
			"threshold_percent": threshold, "recover_percent": recoverPct,
			"percent_change_24h": item.PercentChange24h,
		}
		current.Labels["direction"] = direction
		current.Evidence = []Evidence{{
			Source:  "price_tick",
			Finding: fmt.Sprintf("%s %s %.4f%% in %s", symbol, direction, changePct, window.Name),
		}}
		result = append(result, current)
		state.LastSignalMs = item.Meta.EventTime
		state.Armed = false
		engine.fastMoveSignals.Add(1)
	}
	return result
}

func (engine *Engine) evaluateLiquidation(item agentevent.LiquidationEvent, settings Settings) []Signal {
	if !settings.LiquidationEnabled || item.Notional <= 0 {
		return nil
	}
	symbol := strings.ToUpper(strings.TrimSpace(settings.LiquidationSymbol))
	if symbol == "" {
		symbol = DefaultSettings().LiquidationSymbol
	}
	if item.Meta.Symbol != symbol {
		return nil
	}
	window := settings.LiquidationWindow
	if window <= 0 {
		window = DefaultSettings().LiquidationWindow
	}
	threshold := settings.LiquidationThreshold
	if threshold <= 0 {
		threshold = DefaultSettings().LiquidationThreshold
	}
	cooldown := settings.LiquidationCooldown
	if cooldown <= 0 {
		cooldown = DefaultSettings().LiquidationCooldown
	}
	positionSide := strings.ToLower(strings.TrimSpace(item.LiquidationSide))
	if positionSide != "long" && positionSide != "short" {
		return nil
	}
	key := item.Meta.Symbol + "|" + positionSide
	engine.mu.Lock()
	defer engine.mu.Unlock()
	points := engine.liquidations[key]
	cutoff := item.Meta.EventTime - window.Milliseconds()
	trimmed := points[:0]
	seen := false
	for _, point := range points {
		if point.TimeMs > cutoff {
			trimmed = append(trimmed, point)
			if point.EventID == item.Meta.EventID {
				seen = true
			}
		}
	}
	points = trimmed
	if seen {
		engine.liquidations[key] = points
		return nil
	}
	points = append(points, liquidationPoint{EventID: item.Meta.EventID, TimeMs: item.Meta.EventTime, Notional: item.Notional})
	engine.liquidations[key] = points
	last := engine.liquidationLast[key]
	if last > 0 && item.Meta.EventTime-last < cooldown.Milliseconds() {
		return nil
	}
	total := 0.0
	for _, point := range points {
		total += point.Notional
	}
	if total < threshold {
		return nil
	}
	current := NewSignal(item.Meta.EventID, item.Meta.Symbol, TypeLiquidationSpike, SeverityForRatio(total/threshold), window.String())
	current.CreatedAt = item.Meta.EventTime
	current.Metrics = map[string]float64{
		"aggregate_notional": total, "order_count": float64(len(points)),
		"threshold_notional": threshold, "latest_order_notional": item.Notional,
	}
	current.Labels["liquidation_side"] = positionSide
	current.Evidence = []Evidence{{
		Source:  "liquidation_event",
		Finding: fmt.Sprintf("%s %s liquidation %.2f USDT across %d orders", item.Meta.Symbol, positionSide, total, len(points)),
	}}
	engine.liquidationLast[key] = item.Meta.EventTime
	engine.liquidations[key] = nil
	engine.liquidationSignals.Add(1)
	return []Signal{current}
}

func trimPriceHistory(values []pricePoint, cutoff int64) []pricePoint {
	index := 0
	for index < len(values) && values[index].TimeMs < cutoff {
		index++
	}
	if index == 0 {
		return values
	}
	return append([]pricePoint(nil), values[index:]...)
}

func findBasePrice(values []pricePoint, target int64) (float64, bool) {
	for index := len(values) - 1; index >= 0; index-- {
		if values[index].TimeMs <= target {
			return values[index].Price, true
		}
	}
	return 0, false
}

func (engine *Engine) Stats() EngineStats {
	engine.mu.Lock()
	tracked := len(engine.prices)
	engine.mu.Unlock()
	return EngineStats{
		Events: engine.events.Load(), FastMoveSignals: engine.fastMoveSignals.Load(),
		LiquidationSignals: engine.liquidationSignals.Load(), SignalsDropped: engine.signalsDropped.Load(),
		TrackedSymbols: tracked,
	}
}

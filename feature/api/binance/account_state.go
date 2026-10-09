package binance

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"go_binance_futures/service/binanceapiusage"

	"github.com/adshao/go-binance/v2/futures"
	"golang.org/x/sync/singleflight"
)

// Per-account state: the legacy main-account cache remains unchanged until
// the Stage 3 main trading-cycle migration. No secondary account can access it.
type accountReadState struct {
	mu         sync.RWMutex
	generation uint64
	positions  positionSnapshotEntry
	orders     openOrdersSnapshotEntry
	positionSF singleflight.Group
	orderSF    singleflight.Group
}

func (s *accountReadState) invalidate() {
	s.mu.Lock()
	s.generation++
	s.positions = positionSnapshotEntry{}
	s.orders = openOrdersSnapshotEntry{}
	s.mu.Unlock()
	s.positionSF.Forget("positions")
	s.orderSF.Forget("orders")
}
func (s *accountReadState) positionsCached(ctx context.Context, loader func(context.Context) ([]*futures.PositionRisk, error)) ([]*futures.PositionRisk, error) {
	s.mu.RLock()
	entry := s.positions
	s.mu.RUnlock()
	if entry.valid && entry.expiresAt.After(time.Now()) {
		return clonePositionRisks(entry.rows), nil
	}
	value, err, _ := s.positionSF.Do("positions", func() (any, error) {
		s.mu.RLock()
		entry := s.positions
		gen := s.generation
		s.mu.RUnlock()
		if entry.valid && entry.expiresAt.After(time.Now()) {
			return clonePositionRisks(entry.rows), nil
		}
		rows, err := loader(ctx)
		if err != nil {
			return nil, err
		}
		s.mu.Lock()
		if gen == s.generation {
			s.positions = positionSnapshotEntry{rows: clonePositionRisks(rows), valid: true, expiresAt: time.Now().Add(accountPositionSnapshotTTL)}
		}
		s.mu.Unlock()
		return rows, nil
	})
	if err != nil {
		return nil, err
	}
	rows, _ := value.([]*futures.PositionRisk)
	return clonePositionRisks(rows), nil
}
func (s *accountReadState) ordersCached(ctx context.Context, loader func(context.Context) ([]*futures.Order, error)) ([]*futures.Order, error) {
	s.mu.RLock()
	entry := s.orders
	s.mu.RUnlock()
	if entry.valid && entry.expiresAt.After(time.Now()) {
		return cloneFuturesOrders(entry.rows), nil
	}
	value, err, _ := s.orderSF.Do("orders", func() (any, error) {
		s.mu.RLock()
		entry := s.orders
		gen := s.generation
		s.mu.RUnlock()
		if entry.valid && entry.expiresAt.After(time.Now()) {
			return cloneFuturesOrders(entry.rows), nil
		}
		rows, err := loader(ctx)
		if err != nil {
			return nil, err
		}
		s.mu.Lock()
		if gen == s.generation {
			s.orders = openOrdersSnapshotEntry{rows: cloneFuturesOrders(rows), valid: true, expiresAt: time.Now().Add(accountOpenOrdersSnapshotTTL)}
		}
		s.mu.Unlock()
		return rows, nil
	})
	if err != nil {
		return nil, err
	}
	rows, _ := value.([]*futures.Order)
	return cloneFuturesOrders(rows), nil
}
func (s *accountReadState) positionsFresh(ctx context.Context, loader func(context.Context) ([]*futures.PositionRisk, error)) ([]*futures.PositionRisk, error) {
	s.invalidate()
	s.mu.RLock()
	gen := s.generation
	s.mu.RUnlock()
	rows, err := loader(ctx)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	if gen == s.generation {
		s.positions = positionSnapshotEntry{rows: clonePositionRisks(rows), valid: true, expiresAt: time.Now().Add(accountPositionSnapshotTTL)}
	}
	s.mu.Unlock()
	return clonePositionRisks(rows), nil
}
func (s *accountReadState) ordersFresh(ctx context.Context, loader func(context.Context) ([]*futures.Order, error)) ([]*futures.Order, error) {
	s.invalidate()
	s.mu.RLock()
	gen := s.generation
	s.mu.RUnlock()
	rows, err := loader(ctx)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	if gen == s.generation {
		s.orders = openOrdersSnapshotEntry{rows: cloneFuturesOrders(rows), valid: true, expiresAt: time.Now().Add(accountOpenOrdersSnapshotTTL)}
	}
	s.mu.Unlock()
	return cloneFuturesOrders(rows), nil
}

type tradeConfigState struct {
	mu    sync.RWMutex
	cache map[string]runtimeTradeConfigEntry
	sf    singleflight.Group
}

func (s *tradeConfigState) invalidate(symbol string) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" {
		return
	}
	s.mu.Lock()
	delete(s.cache, symbol)
	s.mu.Unlock()
}
func (s *tradeConfigState) has(symbol string, margin futures.MarginType, leverage int) bool {
	s.mu.RLock()
	entry, ok := s.cache[symbol]
	s.mu.RUnlock()
	return ok && entry.expiresAt.After(time.Now()) && entry.marginType == margin && entry.leverage == leverage
}
func (s *tradeConfigState) ensure(ctx context.Context, symbol string, margin futures.MarginType, leverage int,
	setMargin func(context.Context, string, futures.MarginType) error,
	setLeverage func(context.Context, string, int) (*futures.SymbolLeverage, error)) error {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" || leverage <= 0 {
		return fmt.Errorf("invalid futures trade config symbol=%q leverage=%d", symbol, leverage)
	}
	if s.has(symbol, margin, leverage) {
		return nil
	}
	key := symbol + "|" + string(margin) + "|" + fmt.Sprint(leverage)
	_, err, _ := s.sf.Do(key, func() (any, error) {
		if s.has(symbol, margin, leverage) {
			return nil, nil
		}
		if err := setMargin(ctx, symbol, margin); err != nil && !isMarginTypeAlreadyConfigured(err) {
			return nil, err
		}
		if _, err := setLeverage(ctx, symbol, leverage); err != nil {
			return nil, err
		}
		s.mu.Lock()
		if s.cache == nil {
			s.cache = make(map[string]runtimeTradeConfigEntry)
		}
		s.cache[symbol] = runtimeTradeConfigEntry{marginType: margin, leverage: leverage, expiresAt: time.Now().Add(runtimeTradeConfigTTL)}
		s.mu.Unlock()
		return nil, nil
	})
	if err != nil {
		binanceapiusage.RecordOptimization(binanceapiusage.SourceFromContext(ctx), "account_config_error", 1)
	}
	return err
}

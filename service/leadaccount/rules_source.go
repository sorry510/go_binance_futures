package leadaccount

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/adshao/go-binance/v2/futures"
	binance "go_binance_futures/feature/api/binance"
)

type LeadRuleReader interface {
	GetLeadExchangeInfoContext(context.Context) (*futures.ExchangeInfo, error)
	GetPositionFreshContext(context.Context, binance.PositionParams) ([]*futures.PositionRisk, error)
	GetLeadLeverageBracketContext(context.Context, string) ([]*futures.LeverageBracket, error)
}

const leadExchangeInfoTTL = 5 * time.Minute

type LeadRuleSource struct {
	AccountID         binance.AccountID
	Reader            LeadRuleReader
	cacheMu           sync.RWMutex
	cachedExchange    *futures.ExchangeInfo
	cacheUntil        time.Time
	cacheVersion      uint64
	cacheNow          func() time.Time
	cacheSingleflight singleflight.Group
}

// exchangeInfo caches only SUCCESSFUL public exchange-info snapshots, never
// account-specific positions, margin configuration or leverage brackets.
// Cache is scoped to this LeadRuleSource instance; no shared Main/Lead cache.
func (s *LeadRuleSource) exchangeInfo(ctx context.Context) (*futures.ExchangeInfo, error) {
	if s == nil || s.AccountID != binance.LeadAccountID || s.Reader == nil {
		return nil, fmt.Errorf("Lead exchange rule source unavailable")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	now := time.Now
	if s.cacheNow != nil {
		now = s.cacheNow
	}
	s.cacheMu.RLock()
	stored, until := s.cachedExchange, s.cacheUntil
	s.cacheMu.RUnlock()
	if stored != nil && now().Before(until) {
		return stored, nil
	}
	resultCh := s.cacheSingleflight.DoChan("lead_exchange_info", func() (any, error) {
		s.cacheMu.RLock()
		existing, expiry := s.cachedExchange, s.cacheUntil
		s.cacheMu.RUnlock()
		if existing != nil && now().Before(expiry) {
			return existing, nil
		}
		s.cacheMu.RLock()
		generation := s.cacheVersion
		s.cacheMu.RUnlock()
		info, err := s.Reader.GetLeadExchangeInfoContext(ctx)
		if err != nil {
			return nil, err
		}
		if info == nil || len(info.Symbols) == 0 {
			return nil, fmt.Errorf("Lead exchangeInfo missing symbols")
		}
		if err = ctx.Err(); err != nil {
			return nil, err
		}
		s.cacheMu.Lock()
		if generation != s.cacheVersion {
			s.cacheMu.Unlock()
			return nil, fmt.Errorf("Lead exchangeInfo invalidated during fetch")
		}
		s.cachedExchange = info
		s.cacheUntil = now().Add(leadExchangeInfoTTL)
		s.cacheMu.Unlock()
		return info, nil
	})
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-resultCh:
		if result.Err != nil {
			return nil, result.Err
		}
		info, ok := result.Val.(*futures.ExchangeInfo)
		if !ok || info == nil {
			return nil, fmt.Errorf("Lead exchangeInfo cache missing")
		}
		return info, nil
	}
}

// invalidateExchangeInfo keeps any explicit force-refresh within this Lead
// instance. No invalidation may silently substitute the Main account cache.
func (s *LeadRuleSource) invalidateExchangeInfo() {
	if s == nil {
		return
	}
	s.cacheMu.Lock()
	s.cacheVersion++
	s.cachedExchange = nil
	s.cacheUntil = time.Time{}
	s.cacheMu.Unlock()
	s.cacheSingleflight.Forget("lead_exchange_info")
}

func NewLeadRuleSource(a *binance.AccountClient) (*LeadRuleSource, error) {
	if a == nil || a.ID() != binance.LeadAccountID {
		return nil, fmt.Errorf("Lead order rules require a Lead client")
	}
	return &LeadRuleSource{AccountID: binance.LeadAccountID, Reader: a}, nil
}
func (s *LeadRuleSource) Verify(ctx context.Context, order RiskOpenOrder, margin futures.MarginType) (VerifiedOrderRules, error) {
	if s == nil || s.AccountID != binance.LeadAccountID || s.Reader == nil || order.AccountID != binance.LeadAccountID {
		return VerifiedOrderRules{}, fmt.Errorf("Lead rule source is unbound")
	}
	if err := ctx.Err(); err != nil {
		return VerifiedOrderRules{}, err
	}
	exchange, err := s.exchangeInfo(ctx)
	if err != nil || exchange == nil {
		return VerifiedOrderRules{}, fmt.Errorf("Lead exchange rules unavailable")
	}
	var match *futures.Symbol
	symbol := strings.ToUpper(strings.TrimSpace(order.Symbol))
	for i := range exchange.Symbols {
		if exchange.Symbols[i].Symbol == symbol {
			match = &exchange.Symbols[i]
			break
		}
	}
	if match == nil {
		return VerifiedOrderRules{}, fmt.Errorf("Lead trading symbol not found")
	}
	positions, err := s.Reader.GetPositionFreshContext(ctx, binance.PositionParams{Symbol: symbol})
	if err != nil {
		return VerifiedOrderRules{}, fmt.Errorf("Lead symbol position mode unavailable")
	}
	brackets, err := s.Reader.GetLeadLeverageBracketContext(ctx, symbol)
	if err != nil {
		return VerifiedOrderRules{}, fmt.Errorf("Lead symbol leverage tier unavailable")
	}
	if err := ctx.Err(); err != nil {
		return VerifiedOrderRules{}, err
	}
	return ValidateLeadOrderRules(order, *match, positions, brackets, margin)
}

package binance

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// LeadSymbolCache is the only source of Lead symbol permission in hot paths.
// The 2-second trade loop must never make a SAPI call for a cache miss.
const (
	LeadSymbolRefreshInterval = time.Hour
	LeadSymbolMaxAge          = 2 * time.Hour
	LeadSymbolRequestTimeout  = 10 * time.Second
)

var ErrLeadSymbolNotReady = errors.New("lead symbol whitelist is unavailable or expired")

type LeadSymbolLoader func(context.Context) ([]LeadTradingSymbol, error)

type LeadSymbolStatus struct {
	Ready         bool
	Count         int
	FetchedAt     time.Time
	LastAttemptAt time.Time
	NextAttemptAt time.Time
	ErrorCategory string
}

type LeadSymbolCache struct {
	mu                             sync.RWMutex
	refreshMu                      sync.Mutex
	account                        *AccountClient
	loader                         LeadSymbolLoader
	now                            func() time.Time
	generation                     uint64
	allowed                        map[string]LeadTradingSymbol
	fetchedAt, attemptedAt, nextAt time.Time
	failures                       int
	invalid                        bool
	lastError                      string
}

func NewLeadSymbolCache(client *AccountClient, loader LeadSymbolLoader) (*LeadSymbolCache, error) {
	if client == nil || client.ID() != LeadAccountID {
		return nil, fmt.Errorf("lead symbol cache requires a lead account")
	}
	if loader == nil {
		loader = func(ctx context.Context) ([]LeadTradingSymbol, error) {
			status, err := client.LeadTraderStatus(ctx)
			if err != nil {
				return nil, err
			}
			if !status.Data.IsLeadTrader {
				return nil, ErrLeadSAPIUnauthorized
			}
			return client.LeadTradingSymbols(ctx)
		}
	}
	return &LeadSymbolCache{account: client, loader: loader, now: time.Now}, nil
}

func (c *LeadSymbolCache) Status() LeadSymbolStatus {
	if c == nil {
		return LeadSymbolStatus{ErrorCategory: "not_configured"}
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	ready := c.readyLocked(c.now())
	return LeadSymbolStatus{Ready: ready, Count: len(c.allowed), FetchedAt: c.fetchedAt, LastAttemptAt: c.attemptedAt, NextAttemptAt: c.nextAt, ErrorCategory: c.lastError}
}
func (c *LeadSymbolCache) readyLocked(now time.Time) bool {
	return !c.invalid && !c.fetchedAt.IsZero() && now.Before(c.fetchedAt.Add(LeadSymbolMaxAge)) && len(c.allowed) > 0
}

// Snapshot returns a defensive copy; callers can use it for Lead's full
// Top60 prefilter and for the final pre-submit local gate.
func (c *LeadSymbolCache) Snapshot() (map[string]LeadTradingSymbol, error) {
	if c == nil {
		return nil, ErrLeadSymbolNotReady
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if !c.readyLocked(c.now()) {
		return nil, ErrLeadSymbolNotReady
	}
	result := make(map[string]LeadTradingSymbol, len(c.allowed))
	for symbol, meta := range c.allowed {
		result[symbol] = meta
	}
	return result, nil
}
func (c *LeadSymbolCache) Allows(symbol string) bool {
	if c == nil {
		return false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if !c.readyLocked(c.now()) {
		return false
	}
	_, ok := c.allowed[strings.ToUpper(strings.TrimSpace(symbol))]
	return ok
}

// Invalidate is mandatory before an account/credential replacement. It also
// invalidates any refresh already in flight via generation fencing.
func (c *LeadSymbolCache) Invalidate() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.generation++
	c.allowed = nil
	c.fetchedAt = time.Time{}
	c.nextAt = time.Time{}
	c.invalid = true
	c.lastError = "invalidated"
}

func (c *LeadSymbolCache) Refresh(ctx context.Context) error {
	if c == nil {
		return ErrLeadSymbolNotReady
	}
	c.refreshMu.Lock()
	defer c.refreshMu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	now := c.now()
	c.mu.Lock()
	if now.Before(c.nextAt) {
		c.mu.Unlock()
		return nil
	}
	generation := c.generation
	c.attemptedAt = now
	c.mu.Unlock()

	requestCtx, cancel := context.WithTimeout(ctx, LeadSymbolRequestTimeout)
	symbols, err := c.loader(requestCtx)
	cancel()
	allowed := make(map[string]LeadTradingSymbol, len(symbols))
	if err == nil {
		for _, sym := range symbols {
			name := strings.ToUpper(strings.TrimSpace(sym.Symbol))
			quote := strings.ToUpper(strings.TrimSpace(sym.QuoteAsset))
			if !strings.HasSuffix(name, "USDT") || (quote != "" && quote != "USDT") {
				continue
			}
			if name != "" {
				sym.Symbol = name
				allowed[name] = sym
			}
		}
		if len(allowed) == 0 {
			err = errors.New("empty lead symbol whitelist")
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if generation != c.generation {
		return ErrLeadSymbolNotReady
	}
	if err == nil {
		c.allowed = allowed
		c.fetchedAt = c.now()
		c.nextAt = c.fetchedAt.Add(LeadSymbolRefreshInterval)
		c.failures = 0
		c.invalid = false
		c.lastError = ""
		return nil
	}
	c.failures++
	delay := time.Minute
	if c.failures == 2 {
		delay = 5 * time.Minute
	} else if c.failures >= 3 {
		delay = 15 * time.Minute
	}
	c.nextAt = c.now().Add(delay)
	c.lastError = "refresh_failed"
	if errors.Is(err, ErrLeadSAPIUnauthorized) {
		c.invalid = true
		c.allowed = nil
		c.fetchedAt = time.Time{}
		c.lastError = "unauthorized"
	}
	if len(allowed) == 0 && err.Error() == "empty lead symbol whitelist" {
		c.invalid = true
		c.allowed = nil
		c.fetchedAt = time.Time{}
		c.lastError = "empty"
	}
	return fmt.Errorf("lead symbol refresh rejected: %w", err)
}

// Run owns no external lifecycle. The caller explicitly starts/stops it with
// a cancelable context once an authorized Lead client has been configured.
// No task is started by importing this package or by Main's StartTrade.
func (c *LeadSymbolCache) Run(ctx context.Context) {
	if c == nil {
		return
	}
	// A stopped/rotated Lead account must not leave an open-capable snapshot.
	defer c.Invalidate()
	_ = c.Refresh(ctx)
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = c.Refresh(ctx)
		}
	}
}

package feature

import (
	"sync"
	"time"
)

// A 3-minute account-level new-entry cooldown replaces the old blocking
// Sleep. Exits and ownership reconciliation must never consult this gate.
const tradeOpenCooldownDuration = 3 * time.Minute

type tradeOpenCooldown struct {
	mu           sync.Mutex
	blockedUntil time.Time
}

var (
	mainTradeOpenCooldown tradeOpenCooldown
	leadTradeOpenCooldown tradeOpenCooldown
)

func (c *tradeOpenCooldown) blockFrom(now time.Time) {
	if c == nil {
		return
	}
	c.mu.Lock()
	until := now.Add(tradeOpenCooldownDuration)
	if until.After(c.blockedUntil) {
		c.blockedUntil = until
	}
	c.mu.Unlock()
}

func (c *tradeOpenCooldown) blocksAt(now time.Time) bool {
	if c == nil {
		return false
	}
	c.mu.Lock()
	blocked := now.Before(c.blockedUntil)
	c.mu.Unlock()
	return blocked
}

package spot

import (
	"fmt"
	"sync"
	"time"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/core/logs"
)

// The spot listener runs every two seconds. Normal evaluation and skip reasons
// are logged at Info only on a state transition or at most once per minute
// per symbol; actual notifications and errors remain visible immediately.
type spotListenOutcome string

const (
	spotListenLegacy            spotListenOutcome = "legacy_mode_ignored"
	spotListenCooldown          spotListenOutcome = "notification_cooldown"
	spotListenNoStrategy        spotListenOutcome = "strategy_not_configured"
	spotListenInvalidStrategy   spotListenOutcome = "invalid_strategy_json"
	spotListenNoActiveRules     spotListenOutcome = "no_enabled_rules"
	spotListenMissingPrice      spotListenOutcome = "missing_spot_price"
	spotListenRuleError         spotListenOutcome = "rule_evaluation_error"
	spotListenNoMatch           spotListenOutcome = "evaluated_no_match"
	spotListenNoticeUpdateError spotListenOutcome = "notice_record_update_failed"
	spotListenNotified          spotListenOutcome = "notification_dispatched"
)

const spotListenLogInterval = time.Minute

type spotListenLogState struct {
	outcome spotListenOutcome
	at      time.Time
}

var spotListenLogTracker = struct {
	sync.Mutex
	states      map[int64]spotListenLogState
	lastSummary time.Time
	issues      map[string]time.Time
}{
	states: make(map[int64]spotListenLogState),
	issues: make(map[string]time.Time),
}

func spotListenLogAllowed(previous spotListenLogState, exists bool, outcome spotListenOutcome, now time.Time) bool {
	return !exists || previous.outcome != outcome || now.Sub(previous.at) >= spotListenLogInterval
}

func logSpotListenOutcome(coin models.ListenSymbols, outcome spotListenOutcome) {
	now := time.Now()
	spotListenLogTracker.Lock()
	previous, exists := spotListenLogTracker.states[coin.ID]
	allowed := spotListenLogAllowed(previous, exists, outcome, now)
	if allowed {
		spotListenLogTracker.states[coin.ID] = spotListenLogState{outcome: outcome, at: now}
	}
	spotListenLogTracker.Unlock()
	if !allowed {
		return
	}
	logs.Info("spot listen: symbol=%s id=%d mode=%s status=%s", coin.Symbol, coin.ID, coin.ListenType, outcome)
}

func logSpotListenSummary(enabled int, counts map[spotListenOutcome]int) {
	now := time.Now()
	spotListenLogTracker.Lock()
	allowed := spotListenLogTracker.lastSummary.IsZero() || now.Sub(spotListenLogTracker.lastSummary) >= spotListenLogInterval
	if allowed {
		spotListenLogTracker.lastSummary = now
	}
	spotListenLogTracker.Unlock()
	if !allowed {
		return
	}

	// Emitted even with 0 enabled spot listeners so an idle scheduler is
	// distinguishable from a stopped/failed scheduler.
	logs.Info(
		"spot listen summary: enabled=%d evaluated=%d notified=%d no_match=%d cooldown=%d no_strategy=%d no_active_rules=%d missing_price=%d legacy_ignored=%d rule_error=%d invalid_config=%d notice_update_error=%d",
		enabled,
		counts[spotListenNoMatch]+counts[spotListenNotified]+counts[spotListenRuleError]+counts[spotListenNoticeUpdateError],
		counts[spotListenNotified], counts[spotListenNoMatch], counts[spotListenCooldown],
		counts[spotListenNoStrategy], counts[spotListenNoActiveRules],
		counts[spotListenMissingPrice], counts[spotListenLegacy],
		counts[spotListenRuleError], counts[spotListenInvalidStrategy],
		counts[spotListenNoticeUpdateError],
	)
}

// logSpotListenProblem emits first errors immediately, then no more than once
// per minute for the same listener/strategy/reason. The per-coin Info status
// still reports a change without causing log storms on a two-second loop.
func logSpotListenProblem(coin models.ListenSymbols, outcome spotListenOutcome, strategy string, reason error) {
	key := fmt.Sprintf("%d:%s:%s", coin.ID, outcome, strategy)
	now := time.Now()
	spotListenLogTracker.Lock()
	previous, exists := spotListenLogTracker.issues[key]
	allowed := !exists || now.Sub(previous) >= spotListenLogInterval
	if allowed {
		spotListenLogTracker.issues[key] = now
	}
	spotListenLogTracker.Unlock()
	if !allowed {
		return
	}
	if reason != nil {
		logs.Error("spot listen failure: symbol=%s strategy=%q status=%s error=%v", coin.Symbol, strategy, outcome, reason)
		return
	}
	logs.Warning("spot listen issue: symbol=%s strategy=%q status=%s", coin.Symbol, strategy, outcome)
}

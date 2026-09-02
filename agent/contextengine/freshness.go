package contextengine

import (
	"strings"
	"time"
)

type FreshnessRule struct {
	MaxAge           time.Duration
	RequireTimestamp bool
}

type FreshnessPolicy struct {
	Default FreshnessRule
	Sources map[string]FreshnessRule
}

func DefaultFreshnessPolicy() FreshnessPolicy {
	return FreshnessPolicy{
		Default: FreshnessRule{},
		Sources: map[string]FreshnessRule{
			"get_symbol_analysis_context": {MaxAge: 3 * time.Minute, RequireTimestamp: true},
			"get_symbol_snapshot":         {MaxAge: 3 * time.Minute, RequireTimestamp: true},
			"get_features":                {MaxAge: 3 * time.Minute},
			"get_klines":                  {MaxAge: 15 * time.Minute},
			"get_liquidations":            {MaxAge: 10 * time.Minute},
			"get_funding_rate":            {MaxAge: 2 * time.Hour},
			"get_market_condition":        {MaxAge: 2 * time.Hour},
		},
	}
}

func (policy FreshnessPolicy) Evaluate(source string, asOf *time.Time, now time.Time, dataMissing []string) (Freshness, time.Duration, string) {
	for _, item := range dataMissing {
		if containsStaleMarker(item) {
			return FreshnessStale, 0, item
		}
	}
	rule, ok := policy.Sources[source]
	if !ok {
		rule = policy.Default
	}
	if rule.MaxAge <= 0 {
		if asOf == nil {
			return FreshnessUnknown, 0, ""
		}
		return FreshnessFresh, maxDuration(0, now.Sub(*asOf)), ""
	}
	if asOf == nil {
		if rule.RequireTimestamp {
			return FreshnessMissing, 0, "source timestamp is missing"
		}
		return FreshnessUnknown, 0, "source timestamp is unavailable"
	}
	age := maxDuration(0, now.Sub(*asOf))
	if age > rule.MaxAge {
		return FreshnessStale, age, "data age exceeds freshness policy"
	}
	return FreshnessFresh, age, ""
}

func containsStaleMarker(value string) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(value)), "stale")
}

func maxDuration(left, right time.Duration) time.Duration {
	if left > right {
		return left
	}
	return right
}

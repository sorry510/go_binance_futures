package strategy

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var marketConditionIdentifierPattern = regexp.MustCompile(`(^|[^A-Za-z0-9_])MarketCondition([^A-Za-z0-9_]|$)`)
var deprecatedMarketAtomPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\s*&&\s*BasicTrend\s*(?:<=|>=|==|!=|<|>)\s*-?[0-9]+(?:\.[0-9]+)?`),
	regexp.MustCompile(`BasicTrend\s*(?:<=|>=|==|!=|<|>)\s*-?[0-9]+(?:\.[0-9]+)?\s*&&\s*`),
	regexp.MustCompile(`\s*&&\s*abs\(BasicTrend\)\s*(?:<=|>=|==|!=|<|>)\s*-?[0-9]+(?:\.[0-9]+)?`),
	regexp.MustCompile(`abs\(BasicTrend\)\s*(?:<=|>=|==|!=|<|>)\s*-?[0-9]+(?:\.[0-9]+)?\s*&&\s*`),
	regexp.MustCompile(`\s*&&\s*(?:BTCUSDT|ETHUSDT|SOLUSDT|BNBUSDT)\.(?:PercentChange|Close|Open|Low|High)\s*(?:<=|>=|==|!=|<|>)\s*-?[0-9]+(?:\.[0-9]+)?`),
	regexp.MustCompile(`(?:BTCUSDT|ETHUSDT|SOLUSDT|BNBUSDT)\.(?:PercentChange|Close|Open|Low|High)\s*(?:<=|>=|==|!=|<|>)\s*-?[0-9]+(?:\.[0-9]+)?\s*&&\s*`),
}
var deprecatedMarketIdentifierPattern = regexp.MustCompile(`(^|[^A-Za-z0-9_])(BasicTrend|BTCUSDT|ETHUSDT|SOLUSDT|BNBUSDT)([^A-Za-z0-9_]|$)`)

type marketEnvStrategyRule struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Code       string `json:"code"`
	FullScreen bool   `json:"fullScreen"`
	Enable     bool   `json:"enable"`
}

func StrategyUsesMarketCondition(raw string) bool {
	var rules []marketEnvStrategyRule
	if err := json.Unmarshal([]byte(raw), &rules); err != nil {
		return marketConditionIdentifierPattern.MatchString(raw)
	}
	for _, rule := range rules {
		if marketConditionIdentifierPattern.MatchString(rule.Code) {
			return true
		}
	}
	return false
}

func RemoveDeprecatedMarketEnvFromStrategyJSON(raw string) (string, bool, error) {
	var rules []marketEnvStrategyRule
	if err := json.Unmarshal([]byte(raw), &rules); err != nil {
		return "", false, fmt.Errorf("decode strategy JSON: %w", err)
	}
	changed := false
	for index := range rules {
		code, codeChanged := removeDeprecatedMarketEnvFromCode(rules[index].Code)
		if codeChanged {
			rules[index].Code = code
			changed = true
		}
	}
	if !changed {
		return raw, false, nil
	}
	encoded, err := json.Marshal(rules)
	if err != nil {
		return "", false, fmt.Errorf("encode strategy JSON: %w", err)
	}
	return string(encoded), true, nil
}

func removeDeprecatedMarketEnvFromCode(code string) (string, bool) {
	if !deprecatedMarketIdentifierPattern.MatchString(code) {
		return code, false
	}
	lines := strings.Split(code, "\n")
	out := make([]string, 0, len(lines))
	changed := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") && deprecatedMarketIdentifierPattern.MatchString(trimmed) {
			changed = true
			continue
		}
		if !deprecatedMarketIdentifierPattern.MatchString(line) {
			out = append(out, line)
			continue
		}
		if strings.Contains(line, "let market_shock =") {
			indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
			out = append(out, indent+"let market_shock = false;")
			changed = true
			continue
		}
		updated := line
		for _, pattern := range deprecatedMarketAtomPatterns {
			updated = pattern.ReplaceAllString(updated, "")
		}
		if deprecatedMarketIdentifierPattern.MatchString(updated) {
			if strings.Contains(updated, "let ") && strings.Contains(updated, "=") {
				nameEnd := strings.Index(updated, "=")
				updated = strings.TrimRight(updated[:nameEnd+1], " \t") + " true;"
			} else {
				updated = "true"
			}
		}
		if updated != line {
			changed = true
		}
		out = append(out, updated)
	}
	return strings.Join(out, "\n"), changed
}

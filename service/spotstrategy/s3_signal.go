package spotstrategy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"go_binance_futures/technology"
)

// ClosedSignal identifies an immutable, closed Spot candle. Its key includes
// the selected rule, entire strategy/technology definition and candle close
// time, so a 2-second worker cannot reissue the same market decision.
type ClosedSignal struct {
	Symbol        string
	CandleCloseMS int64
	Direction     string // LONG or CLOSE_LONG
	SignalKey     string
}

func EvaluateClosedSignal(symbol, strategyJSON, technologyJSON string, closeMS int64,
	env map[string]interface{}, hasHolding bool) (ClosedSignal, error) {
	if symbol == "" || closeMS <= 0 || len(env) == 0 {
		return ClosedSignal{}, errors.New("closed Spot signal metadata missing")
	}
	if err := ValidateRulesWithTechnology(strategyJSON, technologyJSON, true); err != nil {
		return ClosedSignal{}, err
	}
	var rules technology.StrategyConfig
	if err := json.Unmarshal([]byte(strategyJSON), &rules); err != nil {
		return ClosedSignal{}, err
	}
	direction, ruleType := "LONG", "long"
	if hasHolding {
		direction, ruleType = "CLOSE_LONG", "close_long"
	}
	for _, rule := range rules {
		if !rule.Enable || rule.Type != ruleType {
			continue
		}
		hit, err := EvaluateRule(rule.Code, env)
		if err != nil {
			return ClosedSignal{}, err
		}
		if !hit {
			continue
		}
		payload := strings.Join([]string{AccountMain, OwnerAuto, symbol, direction,
			strconv.FormatInt(closeMS, 10), strategyJSON, technologyJSON, rule.Code}, "|")
		hash := sha256.Sum256([]byte(payload))
		return ClosedSignal{Symbol: symbol, Direction: direction, CandleCloseMS: closeMS,
			SignalKey: "s3:" + hex.EncodeToString(hash[:24])}, nil
	}
	return ClosedSignal{}, nil
}

package leadaccount

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/adshao/go-binance/v2/futures"
)

// VerifiedOrderRules must be derived from Binance exchangeInfo and the Lead
// portfolio's own signed positionRisk/leverageBracket. Caller-supplied flags
// alone are not evidence. This parser is pure and never mutates leverage or
// margin mode; any discrepancy blocks the attempt.
type VerifiedOrderRules struct {
	MinNotionalUSDT float64
	MaxNotionalUSDT float64
	MaxLeverage     int
}

func parseRuleFloat(fields map[string]any, key string) (float64, error) {
	raw, ok := fields[key]
	if !ok {
		return 0, fmt.Errorf("Lead exchange filter %s missing", key)
	}
	var n float64
	switch value := raw.(type) {
	case string:
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return 0, fmt.Errorf("Lead exchange filter %s invalid", key)
		}
		n = parsed
	case float64:
		n = value
	default:
		return 0, fmt.Errorf("Lead exchange filter %s invalid type", key)
	}
	if !riskFiniteNonnegative(n) {
		return 0, fmt.Errorf("Lead exchange filter %s non-finite", key)
	}
	return n, nil
}
func alignedToStep(value, minimum, step float64) bool {
	if !riskFinitePositive(step) {
		return false
	}
	multiples := (value - minimum) / step
	return math.Abs(multiples-math.Round(multiples)) < 1e-7
}
func ValidateLeadOrderRules(order RiskOpenOrder, public futures.Symbol, private []*futures.PositionRisk, brackets []*futures.LeverageBracket, expectedMargin futures.MarginType) (VerifiedOrderRules, error) {
	invalid := fmt.Errorf("Lead exchange/account trade rules are not confirmed")
	symbol := strings.ToUpper(strings.TrimSpace(order.Symbol))
	if symbol == "" || order.AccountID != "lead" || order.Leverage <= 0 || !riskFinitePositive(order.Quantity) || !riskFinitePositive(order.ReferencePrice) ||
		public.Symbol != symbol || public.Status != "TRADING" || public.QuoteAsset != "USDT" {
		return VerifiedOrderRules{}, invalid
	}
	matching := 0
	for _, row := range private {
		if row == nil {
			continue
		}
		if row.Symbol != symbol {
			continue
		}
		// Hedge mode returns separate LONG/SHORT row, both must agree.
		matching++
		currentMargin := strings.ToUpper(strings.TrimSpace(row.MarginType))
		if currentMargin == "CROSS" {
			currentMargin = string(futures.MarginTypeCrossed)
		}
		if currentMargin != string(expectedMargin) {
			return VerifiedOrderRules{}, fmt.Errorf("Lead margin mode mismatch")
		}
		lev, err := strconv.Atoi(row.Leverage)
		if err != nil || lev != order.Leverage {
			return VerifiedOrderRules{}, fmt.Errorf("Lead leverage mismatch")
		}
	}
	if matching == 0 {
		return VerifiedOrderRules{}, fmt.Errorf("Lead account symbol settings unavailable")
	}
	rules := VerifiedOrderRules{}
	foundBracket := false
	for _, bracket := range brackets {
		if bracket == nil || bracket.Symbol != symbol {
			continue
		}
		for _, level := range bracket.Brackets {
			if level.InitialLeverage >= order.Leverage && riskFinitePositive(level.NotionalCap) &&
				(!foundBracket || level.NotionalCap > rules.MaxNotionalUSDT) {
				rules.MaxNotionalUSDT = level.NotionalCap
				rules.MaxLeverage = level.InitialLeverage
				foundBracket = true
			}
		}
	}
	if !foundBracket {
		return VerifiedOrderRules{}, fmt.Errorf("Lead leverage bracket unavailable")
	}
	haveLot, havePrice, haveNotional := false, false, false
	for _, filter := range public.Filters {
		kind, _ := filter["filterType"].(string)
		switch kind {
		case "LOT_SIZE", "MARKET_LOT_SIZE":
			if (order.OrderType == "MARKET") != (kind == "MARKET_LOT_SIZE") {
				continue
			}
			minimum, e1 := parseRuleFloat(filter, "minQty")
			maximum, e2 := parseRuleFloat(filter, "maxQty")
			step, e3 := parseRuleFloat(filter, "stepSize")
			if e1 != nil || e2 != nil || e3 != nil || !riskFinitePositive(step) || order.Quantity < minimum || order.Quantity > maximum || !alignedToStep(order.Quantity, minimum, step) {
				return VerifiedOrderRules{}, fmt.Errorf("Lead quantity filter rejected")
			}
			haveLot = true
		case "PRICE_FILTER":
			minimum, e1 := parseRuleFloat(filter, "minPrice")
			maximum, e2 := parseRuleFloat(filter, "maxPrice")
			tick, e3 := parseRuleFloat(filter, "tickSize")
			if e1 != nil || e2 != nil || e3 != nil || !riskFinitePositive(tick) {
				return VerifiedOrderRules{}, invalid
			}
			if order.OrderType == "LIMIT" && (order.ReferencePrice < minimum || order.ReferencePrice > maximum || !alignedToStep(order.ReferencePrice, minimum, tick)) {
				return VerifiedOrderRules{}, fmt.Errorf("Lead price tick/limits rejected")
			}
			havePrice = true
		case "MIN_NOTIONAL":
			minimum, err := parseRuleFloat(filter, "notional")
			if err != nil {
				return VerifiedOrderRules{}, invalid
			}
			if minimum > rules.MinNotionalUSDT {
				rules.MinNotionalUSDT = minimum
			}
			haveNotional = true
		case "NOTIONAL":
			minimum, err := parseRuleFloat(filter, "minNotional")
			if err != nil {
				return VerifiedOrderRules{}, invalid
			}
			if minimum > rules.MinNotionalUSDT {
				rules.MinNotionalUSDT = minimum
			}
			if maximum, err := parseRuleFloat(filter, "maxNotional"); err == nil && maximum > 0 && maximum < rules.MaxNotionalUSDT {
				rules.MaxNotionalUSDT = maximum
			}
			haveNotional = true
		}
	}
	if !haveLot || !havePrice || !haveNotional || rules.MaxNotionalUSDT < rules.MinNotionalUSDT {
		return VerifiedOrderRules{}, fmt.Errorf("Lead price/quantity/notional filters incomplete")
	}
	return rules, nil
}

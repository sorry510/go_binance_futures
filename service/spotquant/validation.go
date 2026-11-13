package spotquant

import (
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/parser"
)

type Rule struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Code   string `json:"code"`
	Enable bool   `json:"enable"`
}

var positiveAmountPattern = regexp.MustCompile(`^[0-9]+(?:\.[0-9]{1,18})?$`)

// Spot Quant rejects Futures-only DSL fields. In 06-7, ROI/Position/Positions
// have independently verified Spot semantics, without Futures account data.
// OpenStrategyHash stays forbidden until the entry-rule identity is persisted.
var spotFuturesOnlyIdentifiers = map[string]bool{
	"FundingRate": true, "Leverage": true, "MarginType": true,
	"PositionSide": true, "LiquidationPrice": true,
	"UnrealizedProfit": true, "IsolatedWallet": true,
	"MarketCondition": true, "BasicTrend": true, "OpenStrategyHash": true,
}

// ValidateBudget accepts a positive decimal amount in the symbol's quote
// currency. The value is never converted through float64.
func ValidateBudget(value string) error {
	value = strings.TrimSpace(value)
	if len(value) > 48 || !positiveAmountPattern.MatchString(value) {
		return fmt.Errorf("budget must be a positive decimal with at most 18 fractional digits")
	}
	n := new(big.Rat)
	if _, ok := n.SetString(value); !ok || n.Sign() <= 0 {
		return fmt.Errorf("budget must be greater than zero")
	}
	return nil
}

// ValidateTechnology preserves the existing indicator JSON schema without
// querying Futures prices or modifying the legacy Spot monitor.
func ValidateTechnology(raw string) error {
	if raw == "" {
		return nil // editing a symbol or an initially empty template
	}
	var config map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &config); err != nil || config == nil {
		return fmt.Errorf("technology must be a JSON object")
	}
	allowed := map[string]bool{
		"ma": true, "ema": true, "macd": true, "adx": true, "mfi": true,
		"obv": true, "cci": true, "roc": true, "kdj": true, "rsi": true,
		"kc": true, "boll": true, "donchian": true, "atr": true, "supertrend": true,
	}
	for key, value := range config {
		if !allowed[key] {
			return fmt.Errorf("unsupported technology indicator %q", key)
		}
		var entries []json.RawMessage
		if err := json.Unmarshal(value, &entries); err != nil {
			return fmt.Errorf("technology.%s must be an array", key)
		}
		for _, item := range entries {
			var fields struct {
				Name   string `json:"name"`
				Enable bool   `json:"enable"`
			}
			if err := json.Unmarshal(item, &fields); err != nil {
				return fmt.Errorf("invalid technology.%s item: %w", key, err)
			}
			if fields.Enable && strings.TrimSpace(fields.Name) == "" {
				return fmt.Errorf("enabled technology.%s item requires a name", key)
			}
		}
	}
	return nil
}

// Nested index/member expressions such as Positions[0].UnrealizedProfit
// have to be rooted in the verified Spot position input. Reject other hosts.
func spotMemberRoot(node ast.Node) string {
	for node != nil {
		switch n := node.(type) {
		case *ast.IdentifierNode:
			return n.Value
		case *ast.MemberNode:
			node = n.Node
		default:
			return ""
		}
	}
	return ""
}

// ParseRules validates rule directions and expression syntax. This is
// deliberately not a live market evaluation or an order trigger.
func ParseRules(raw string) ([]Rule, error) {
	if raw == "" {
		return nil, nil
	}
	var rules []Rule
	if err := json.Unmarshal([]byte(raw), &rules); err != nil || rules == nil {
		return nil, fmt.Errorf("strategy must be a JSON array")
	}
	if len(rules) > 100 {
		return nil, fmt.Errorf("too many strategy rules")
	}
	for i, rule := range rules {
		if rule.Type != "" && rule.Type != "long" && rule.Type != "close_long" {
			return nil, fmt.Errorf("strategy[%d] type %q is not allowed for Spot (LONG/CLOSE_LONG only)", i, rule.Type)
		}
		if rule.Enable && (strings.TrimSpace(rule.Name) == "" || strings.TrimSpace(rule.Code) == "" || rule.Type == "") {
			return nil, fmt.Errorf("enabled strategy[%d] requires name, type and code", i)
		}
		if strings.TrimSpace(rule.Code) != "" {
			tree, err := parser.Parse(rule.Code)
			if err != nil {
				return nil, fmt.Errorf("strategy[%d] expression syntax: %w", i, err)
			}
			// Inspect identifier nodes, not raw substrings: a string
			// containing "FundingRate" is not a funding-rate dependency.
			forbidden := ast.Find(tree.Node, func(node ast.Node) bool {
				switch typed := node.(type) {
				case *ast.IdentifierNode:
					if spotFuturesOnlyIdentifiers[typed.Value] {
						return true
					}
					// Only a real Spot strategy position can provide ROI and
					// Position; the entry side has no position to evaluate.
					return rule.Type == "long" && (typed.Value == "ROI" || typed.Value == "Position")
				case *ast.MemberNode:
					// Position.Leverage and Positions[0]["MarginType"]
					// must remain unavailable even though Position itself
					// is now allowed for Spot close rules.
					property, ok := typed.Property.(*ast.StringNode)
					if !ok {
						return false
					}
					// This property exists on the verified Spot position,
					// not arbitrary future environment variables.
					if property.Value == "UnrealizedProfit" {
						root := spotMemberRoot(typed.Node)
						return root != "Position" && root != "Positions"
					}
					return spotFuturesOnlyIdentifiers[property.Value]
				default:
					return false
				}
			})
			if forbidden != nil {
				return nil, fmt.Errorf("strategy[%d] uses futures-only or unavailable Spot identifier %q", i, forbidden.String())
			}
		}
	}
	return rules, nil
}

func ValidateActiveStrategy(raw string) error {
	rules, err := ParseRules(raw)
	if err != nil {
		return err
	}
	for _, rule := range rules {
		if rule.Enable && rule.Type == "long" {
			return nil
		}
	}
	return fmt.Errorf("enable at least one valid LONG entry rule before activating Spot Quant")
}

// ParseJSONField accepts either an embedded JSON object/array or the
// corresponding JSON-encoded string used by existing editor forms.
func ParseJSONField(raw json.RawMessage, object bool) (string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", nil
	}
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		text = string(raw)
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return "", nil
	}
	if object {
		if err := ValidateTechnology(text); err != nil {
			return "", err
		}
	} else {
		if _, err := ParseRules(text); err != nil {
			return "", err
		}
	}
	return text, nil
}

func SpotQuoteAsset(symbol string) string {
	for _, quote := range []string{"FDUSD", "USDT", "USDC"} {
		if strings.HasSuffix(symbol, quote) && len(symbol) > len(quote) {
			return quote
		}
	}
	return ""
}

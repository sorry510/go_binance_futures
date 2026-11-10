package spotstrategy

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"regexp"
	"strings"

	"go_binance_futures/models"
	"go_binance_futures/technology"

	"github.com/beego/beego/v2/client/orm"
	"github.com/expr-lang/expr"
)

var indicatorNameText = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]{0,63}$`)

var decimalText = regexp.MustCompile(`^[0-9]{1,18}(?:\.[0-9]{1,18})?$`)

func ValidateScope(accountID, owner string) error {
	if accountID != AccountMain || owner != OwnerAuto {
		return fmt.Errorf("spot strategy only permits main/auto_strategy, got %q/%q", accountID, owner)
	}
	return nil
}

// NormalizeBudget accepts base-10 decimal strings without floating conversion,
// exponents, signs or more than 18 fractional places.
func NormalizeBudget(input string, enabled bool) (string, error) {
	input = strings.TrimSpace(input)
	if !decimalText.MatchString(input) {
		return "", errors.New("budget_quote must be a nonnegative decimal with <=18 integer and fractional digits")
	}
	v, ok := new(big.Rat).SetString(input)
	if !ok || v.Sign() < 0 || (enabled && v.Sign() <= 0) {
		return "", errors.New("enabled spot strategy requires positive quote budget")
	}
	normalized := strings.TrimRight(strings.TrimRight(v.FloatString(18), "0"), ".")
	if normalized == "" {
		normalized = "0"
	}
	return normalized, nil
}

// SpotRuleEnvironment is a compile-time shape for exactly the same names
// emitted by InitSpotParseEnv. It uses no network or Futures positions.
// Indicator names come only from enabled technology entries and cannot
// shadow built-in environment values.
func SpotRuleEnvironment(technologyJSON string) (map[string]interface{}, error) {
	var config technology.TechnologyConfig
	if err := json.Unmarshal([]byte(technologyJSON), &config); err != nil {
		return nil, fmt.Errorf("invalid Spot technology: %w", err)
	}
	env := map[string]interface{}{
		"NowPrice": float64(1), "NowSymbolClose": float64(1),
		"NowSymbolOpen": float64(1), "NowSymbolLow": float64(1),
		"NowSymbolHigh": float64(1), "NowSymbolPercentChange": float64(0),
		"NowTime": int64(0), "SystemStartTime": int64(0),
		"MarketCondition": "", "OpenStrategyHash": "",
		"KdjSimple": func([]float64, []float64, int) bool { return false },
		"IsAsc":     func([]float64) bool { return false },
		"IsDesc":    func([]float64) bool { return false },
	}
	t := reflect.ValueOf(config)
	typ := t.Type()
	for i := 0; i < t.NumField(); i++ {
		category := strings.ToLower(typ.Field(i).Name)
		indicators := t.Field(i)
		for j := 0; j < indicators.Len(); j++ {
			item := indicators.Index(j).Interface().(technology.IndicatorConfig)
			if !item.Enable {
				continue
			}
			if !indicatorNameText.MatchString(item.Name) {
				return nil, fmt.Errorf("invalid Spot indicator name %q", item.Name)
			}
			if _, exists := env[item.Name]; exists {
				return nil, fmt.Errorf("Spot indicator name %q conflicts with another variable", item.Name)
			}
			if item.KlineInterval == "" {
				return nil, fmt.Errorf("Spot indicator %q missing kline interval", item.Name)
			}
			series := []float64{1}
			fields := map[string]interface{}{"Data": series, "High": series, "Low": series, "Mid": series,
				"KlineInterval": item.KlineInterval, "Period": item.Period}
			switch category {
			case "macd":
				fields = map[string]interface{}{"DIF": series, "DEA": series, "Histogram": series, "KlineInterval": item.KlineInterval}
			case "adx":
				fields = map[string]interface{}{"ADX": series, "PlusDI": series, "MinusDI": series, "KlineInterval": item.KlineInterval}
			case "kdj":
				fields = map[string]interface{}{"K": series, "D": series, "J": series, "KlineInterval": item.KlineInterval}
			case "supertrend":
				fields = map[string]interface{}{"Data": series, "Trend": series, "KlineInterval": item.KlineInterval}
			}
			env[item.Name] = fields
			klineName := "kline_" + item.KlineInterval
			if _, ok := env[klineName]; !ok {
				env[klineName] = map[string]interface{}{
					"High": series, "Low": series, "Close": series, "Open": series, "Amount": series,
					"Qps": series, "TakerBuyAmount": series, "TakerBuyRatio": series,
				}
			}
		}
	}
	return env, nil
}

func ValidateRules(raw string, enabled bool) error {
	return ValidateRulesWithTechnology(raw, "{}", enabled)
}

func ValidateRulesWithTechnology(raw string, tech string, enabled bool) error {
	var rules technology.StrategyConfig
	if err := json.Unmarshal([]byte(raw), &rules); err != nil {
		return fmt.Errorf("invalid spot strategy JSON: %w", err)
	}
	env, err := SpotRuleEnvironment(tech)
	if err != nil {
		return err
	}
	hasLong, hasClose := false, false
	for _, rule := range rules {
		// Spot configurations cannot carry Futures SHORT directions even
		// when a rule is disabled. Otherwise a disabled SHORT payload could
		// be persisted today and accidentally enabled by a later editor.
		if rule.Type != "long" && rule.Type != "close_long" {
			return fmt.Errorf("spot strategy rule type %q is not long/close_long", rule.Type)
		}
		if !rule.Enable {
			continue
		}
		if strings.TrimSpace(rule.Code) == "" {
			return errors.New("enabled spot strategy rule code cannot be empty")
		}
		if _, err := expr.Compile(rule.Code, expr.Env(env), expr.AsBool()); err != nil {
			return fmt.Errorf("spot rule references invalid variable/type: %w", err)
		}
		if rule.Type == "long" {
			hasLong = true
		}
		if rule.Type == "close_long" {
			hasClose = true
		}
	}
	if enabled && (!hasLong || !hasClose) {
		return errors.New("enabled spot strategy requires both long and close_long rules")
	}
	return nil
}

// EvaluateRule is a pure, testable expr adapter. No Futures positions, account
// lookups, notifications or exchange orders are accessed.
func EvaluateRule(code string, env map[string]interface{}) (bool, error) {
	program, err := expr.Compile(code, expr.Env(env))
	if err != nil {
		return false, err
	}
	result, err := expr.Run(program, env)
	if err != nil {
		return false, err
	}
	matched, ok := result.(bool)
	if !ok {
		return false, errors.New("spot strategy rule must evaluate to boolean")
	}
	return matched, nil
}

func CurrentMode() (string, error) {
	// The system config is single-operator/single-row. Do not select an
	// arbitrary mode when a damaged database contains multiple config rows.
	var rows []struct {
		Mode string `orm:"column(spot_auto_trade_mode)"`
	}
	n, err := orm.NewOrm().Raw("SELECT spot_auto_trade_mode FROM config ORDER BY id LIMIT 2").QueryRows(&rows)
	if err != nil {
		return "", fmt.Errorf("load spot trade mode: %w", err)
	}
	if n != 1 {
		return "", fmt.Errorf("spot trade mode requires exactly one config row (found %d): fail closed", n)
	}
	mode := rows[0].Mode
	switch mode {
	case ModeLegacy, ModeStrategy, ModeOff:
		return mode, nil
	default:
		return "", fmt.Errorf("invalid spot trade mode %q: fail closed", mode)
	}
}

// OrderSource describes exactly which retained legacy path originated an
// order. The deprecated eatRate API still uses SourceLegacyDirect via the
// original wrapper; no further eatRate development is done in phase 02.
type OrderSource string
type OrderIntent string

const (
	SourceRush         OrderSource = "spot_rush"
	SourceNotice       OrderSource = "spot_notice_auto_order"
	SourceLegacyDirect OrderSource = "legacy_direct"
	IntentNewRisk      OrderIntent = "new_risk"
	IntentExit         OrderIntent = "exit"
)

// LegacyIntentForSide is shared by rush and alert dispatch. Unknown sides
// must not be interpreted as a buy or a verified exit.
func LegacyIntentForSide(side string) (OrderIntent, error) {
	switch side {
	case "buy":
		return IntentNewRisk, nil
	case "sell":
		return IntentExit, nil
	default:
		return "", fmt.Errorf("unknown legacy Spot side %q", side)
	}
}

// AuthorizeLegacyOrder protects the actual POST /order constructors. Intent
// is explicit even though S1 cannot yet prove old inventory ownership. Only
// legacy mode may issue either type of order until S2 reconciliation is done.
// Never infer that a SELL is verified merely because its intent is "exit".
func AuthorizeLegacyOrder(source OrderSource, symbol string, intent OrderIntent) error {
	if len(symbol) == 0 || len(symbol) > 32 || strings.TrimSpace(symbol) != symbol {
		return errors.New("spot order symbol required")
	}
	switch source {
	case SourceRush, SourceNotice, SourceLegacyDirect:
	default:
		return fmt.Errorf("unknown legacy Spot order source %q", source)
	}
	if intent != IntentNewRisk && intent != IntentExit {
		return fmt.Errorf("unknown legacy Spot order intent %q", intent)
	}
	mode, err := CurrentMode()
	if err != nil {
		return err
	}
	if mode != ModeLegacy {
		return fmt.Errorf("legacy spot order blocked: source=%s symbol=%s intent=%s mode=%s (ownership unverified)", source, symbol, intent, mode)
	}
	return nil
}

// Setting contains public symbol configuration, not private account state.
// Account ownership and runtime mode belong to order/Holding and mode APIs.
type Setting struct {
	Symbol      string `json:"symbol"`
	QuoteAsset  string `json:"quote_asset"`
	Enabled     bool   `json:"enabled"`
	BudgetQuote string `json:"budget_quote"`
	Strategy    string `json:"strategy"`
	Technology  string `json:"technology"`
	LastPrice   string `json:"last_price"`
	LastPriceAt int64  `json:"last_price_at"`
}

// Patch updates a single spot_symbols row. It has no account/owner or
// exchange order fields; missing enabled is rejected.
type Patch struct {
	Symbol      string  `json:"symbol"`
	Enabled     *bool   `json:"enabled"`
	BudgetQuote string  `json:"budget_quote"`
	Strategy    *string `json:"strategy,omitempty"`
	Technology  *string `json:"technology,omitempty"`
}

func LoadSetting(symbol string) (Setting, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if !s4SymbolPattern.MatchString(symbol) {
		return Setting{}, errors.New("invalid spot symbol")
	}
	var coin models.SpotSymbols
	if err := orm.NewOrm().QueryTable("spot_symbols").Filter("symbol", symbol).One(&coin); err != nil {
		return Setting{}, fmt.Errorf("spot symbol unavailable: %w", err)
	}
	return Setting{Symbol: coin.Symbol, QuoteAsset: coin.Type,
		Enabled: coin.StrategyTradeEnabled == 1, BudgetQuote: coin.StrategyBudgetQuote,
		Strategy: coin.Strategy, Technology: coin.Technology,
		LastPrice: coin.Close, LastPriceAt: coin.LastUpdateTime}, nil
}

func UpdateSetting(p Patch) (Setting, error) {
	if p.Enabled == nil {
		return Setting{}, errors.New("enabled must be supplied")
	}
	p.Symbol = strings.ToUpper(strings.TrimSpace(p.Symbol))
	if p.Symbol == "" || len(p.Symbol) > 32 {
		return Setting{}, errors.New("invalid spot symbol")
	}
	o := orm.NewOrm()
	var coin models.SpotSymbols
	if err := o.QueryTable("spot_symbols").Filter("symbol", p.Symbol).One(&coin); err != nil {
		return Setting{}, fmt.Errorf("spot symbol unavailable: %w", err)
	}
	if coin.Type != "USDT" && coin.Type != "USDC" && coin.Type != "FDUSD" {
		return Setting{}, errors.New("spot quote asset not supported")
	}
	if !strings.HasSuffix(coin.Symbol, coin.Type) {
		return Setting{}, errors.New("spot symbol/quote metadata inconsistent")
	}
	budget, err := NormalizeBudget(p.BudgetQuote, *p.Enabled)
	if err != nil {
		return Setting{}, err
	}
	rules := coin.Strategy
	if p.Strategy != nil {
		rules = *p.Strategy
	}
	tech := coin.Technology
	if p.Technology != nil {
		var value map[string]json.RawMessage
		if err := json.Unmarshal([]byte(*p.Technology), &value); err != nil || value == nil {
			return Setting{}, errors.New("technology must be a JSON object")
		}
		tech = *p.Technology
	}
	// Always permit an emergency disable even when an old stored rule no
	// longer compiles. Validate when enabling or replacing any rule/indicator.
	if *p.Enabled || p.Strategy != nil || p.Technology != nil {
		if err := ValidateRulesWithTechnology(rules, tech, *p.Enabled); err != nil {
			return Setting{}, err
		}
	}
	enabled := 0
	if *p.Enabled {
		enabled = 1
	}
	updates := orm.Params{"strategy_trade_enabled": enabled, "strategy_budget_quote": budget}
	if p.Strategy != nil {
		updates["strategy"] = rules
	}
	if p.Technology != nil {
		updates["technology"] = tech
	}
	if _, err := o.QueryTable("spot_symbols").Filter("id", coin.ID).Update(updates); err != nil {
		return Setting{}, fmt.Errorf("update spot strategy config: %w", err)
	}
	return LoadSetting(p.Symbol)
}

// RequestMode is intentionally read-only in S1. Turning legacy off would
// silently stop unverified legacy SELL/protection work; turning off back to
// legacy could also re-enable risk. No real transition is authorized until
// S2 verifies old orders, balances and exit ownership. Only same-mode
// requests are idempotent; unrecognized stored values fail closed.
func RequestMode(mode string) error {
	current, err := CurrentMode()
	if err != nil {
		return err
	}
	if mode != ModeLegacy && mode != ModeOff && mode != ModeStrategy {
		return fmt.Errorf("unknown requested Spot mode %q", mode)
	}
	if mode == ModeStrategy {
		return fmt.Errorf("spot_mode_reconciliation_required: strategy mode unavailable during S1")
	}
	if current != mode {
		return fmt.Errorf("spot_mode_reconciliation_required: %s -> %s; verify legacy exits, open orders and account ownership first", current, mode)
	}
	return nil
}

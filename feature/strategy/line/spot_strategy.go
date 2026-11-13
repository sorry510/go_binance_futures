package line

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"go_binance_futures/models"
	"go_binance_futures/service/spotledger"
	"go_binance_futures/service/spotquant"
	strategyservice "go_binance_futures/service/strategy"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/expr-lang/expr"
	"github.com/shopspring/decimal"
)

// SpotSignal is a strategy decision, never an exchange request. The 06-7-2
// scheduler may pass BUY/SELL to the already validated Spot Processor.
type SpotSignal struct {
	Action   string // empty means hold; BUY or SELL only
	Reason   string // long, close_long, take_profit, stop_loss
	RuleName string
	RuleHash string // identity of the rule evaluated in this cycle
	ROI      string // unleveraged Spot ROI percent, for diagnostics only
}

// A deterministic identity for the Spot-only 5m simple-close fallback.
// Client order IDs must NOT be derived solely from this hash: 06-7-2 must
// also deduplicate by durable cycle/position and bar across strategy edits.
const spotSimpleCloseRuleIdentity = "spot_simple_close_5m_v1"

// SpotPositionCode is backed exclusively by a verified Spot Fill projection.
// It exposes the familiar strategy DSL fields without Futures leverage,
// margin, liquidation or any manually held wallet inventory.
type SpotPositionCode struct {
	Symbol           string
	Side             string
	Amount           float64
	EntryPrice       float64
	MarkPrice        float64
	UnrealizedProfit float64
	CreateTime       int64
}

// SpotRuleEvaluator shares the existing Expr/indicator DSL, but its market
// loader is Spot-only. Injection keeps 06-7-1 tests isolated from Binance
// REST/WS and from the Futures account or tables.
type SpotRuleEvaluator struct {
	LoadEnvironment func(models.SpotSymbols) (map[string]interface{}, error)
	LoadKlines      func(string, string, int) ([]*futures.Kline, error)
}

func NewSpotRuleEvaluator() SpotRuleEvaluator {
	return SpotRuleEvaluator{
		LoadEnvironment: LoadVerifiedSpotRuleEnv,
		LoadKlines:      verifiedSpotKlines,
	}
}

func spotRuleEnvironment(e SpotRuleEvaluator, coin models.SpotSymbols) (map[string]interface{}, float64, error) {
	if e.LoadEnvironment == nil {
		return nil, 0, fmt.Errorf("Spot strategy environment loader is required")
	}
	if strings.TrimSpace(coin.Symbol) == "" {
		return nil, 0, fmt.Errorf("Spot strategy symbol is empty")
	}
	env, err := e.LoadEnvironment(coin)
	if err != nil {
		return nil, 0, fmt.Errorf("Spot market data %s: %w", coin.Symbol, err)
	}
	if env == nil {
		return nil, 0, fmt.Errorf("Spot market data %s is unavailable", coin.Symbol)
	}
	price, ok := env["NowPrice"].(float64)
	if !ok || price <= 0 || math.IsNaN(price) || math.IsInf(price, 0) {
		return nil, 0, fmt.Errorf("Spot market price %s is not a positive finite value", coin.Symbol)
	}
	// Rules must never mutate a shared market cache or test fixture.
	copied := make(map[string]interface{}, len(env)+3)
	for key, value := range env {
		copied[key] = value
	}
	return copied, price, nil
}

func spotRules(raw string) ([]spotquant.Rule, error) {
	rules, err := spotquant.ParseRules(raw)
	if err != nil {
		return nil, err
	}
	return rules, nil
}

func spotRuleMatches(rule spotquant.Rule, env map[string]interface{}) (bool, error) {
	program, err := expr.Compile(strings.TrimSpace(rule.Code), expr.Env(env), expr.AsBool())
	if err != nil {
		return false, fmt.Errorf("Spot strategy rule %q compile: %w", rule.Name, err)
	}
	value, err := expr.Run(program, env)
	if err != nil {
		return false, fmt.Errorf("Spot strategy rule %q run: %w", rule.Name, err)
	}
	result, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("Spot strategy rule %q did not yield bool", rule.Name)
	}
	return result, nil
}

// EvaluateEntry uses exactly the configured LONG rule order and rule hash.
// No position, Futures API, or Spot exchange order is read or created here.
func (e SpotRuleEvaluator) EvaluateEntry(coin models.SpotSymbols) (SpotSignal, error) {
	var hold SpotSignal
	rules, err := spotRules(coin.Strategy)
	if err != nil {
		return hold, err
	}
	hasLong := false
	for _, rule := range rules {
		if rule.Enable && rule.Type == "long" {
			hasLong = true
			break
		}
	}
	if !hasLong {
		return hold, nil
	}
	env, _, err := spotRuleEnvironment(e, coin)
	if err != nil {
		return hold, err
	}
	env["Positions"] = []SpotPositionCode{}
	for _, rule := range rules {
		if !rule.Enable || rule.Type != "long" {
			continue
		}
		match, err := spotRuleMatches(rule, env)
		if err != nil {
			// A broken strategy must not silently advance to a later rule
			// and unexpectedly open a position.
			return hold, err
		}
		if match {
			return SpotSignal{Action: "BUY", Reason: "long", RuleName: rule.Name, RuleHash: strategyservice.RuleHash(rule.Code)}, nil
		}
	}
	return hold, nil
}

func validSpotPosition(snapshot spotledger.Snapshot, symbol string) (decimal.Decimal, decimal.Decimal, error) {
	p := snapshot.Position
	if p.Symbol != symbol || p.Status != "OPEN" {
		return decimal.Decimal{}, decimal.Decimal{}, fmt.Errorf("Spot managed position must be OPEN with matching symbol")
	}
	// A known third-asset commission (e.g. BNB) may be unpriced in
	// quote currency without invalidating the verified base inventory.
	// Ledger.Accounting checks all Fill/Order consistency first.
	quantity, err := decimal.NewFromString(p.RemainingQty)
	if err != nil || !quantity.IsPositive() {
		return decimal.Decimal{}, decimal.Decimal{}, fmt.Errorf("Spot remaining quantity is not verified")
	}
	entry, err := decimal.NewFromString(p.AvgCostQuote)
	if err != nil || !entry.IsPositive() {
		return decimal.Decimal{}, decimal.Decimal{}, fmt.Errorf("Spot average entry cost is not verified")
	}
	return quantity, entry, nil
}

func checkedSpotThreshold(raw string) (decimal.Decimal, error) {
	value := strings.TrimSpace(raw)
	if value == "" || value == "0" {
		return decimal.NewFromInt(1000000), nil // same disabled threshold as Futures
	}
	amount, err := decimal.NewFromString(value)
	if err != nil || !amount.IsPositive() {
		return decimal.Decimal{}, fmt.Errorf("invalid Spot profit/loss ROI threshold")
	}
	return amount, nil
}

// EvaluateExit MUST load the position from the authoritative Spot Fill ledger
// itself. Accepting a caller-provided Snapshot would let stale editable
// materialized position columns fabricate ROI-based SELL decisions. The caller
// supplies only a position ID and the real ledger (never a Futures mirror).
// ROI is (SpotPrice - AvgCost)/AvgCost * 100 (no leverage). Third-asset fees
// may be unpriced; this price-movement ROI is not a complete net PnL claim.
func (e SpotRuleEvaluator) EvaluateExit(coin models.SpotSymbols, ledger *spotledger.Ledger, positionID int64) (SpotSignal, error) {
	if ledger == nil || positionID <= 0 {
		return SpotSignal{}, fmt.Errorf("Spot exit requires a valid Fill ledger and position ID")
	}
	snapshot, err := ledger.Accounting(positionID, "")
	if err != nil {
		return SpotSignal{}, fmt.Errorf("Spot exit Fill reconciliation: %w", err)
	}
	if snapshot.Position.ID != positionID {
		return SpotSignal{}, fmt.Errorf("Spot exit position identity mismatch")
	}
	return e.evaluateExitSnapshot(coin, snapshot)
}

// evaluateExitSnapshot is private: only EvaluateExit may expose tradeable
// decisions to a 06-7-2 production caller; tests separately exercise the
// expression semantics with artificial snapshots.
func (e SpotRuleEvaluator) evaluateExitSnapshot(coin models.SpotSymbols, snapshot spotledger.Snapshot) (SpotSignal, error) {
	var hold SpotSignal
	quantity, entry, err := validSpotPosition(snapshot, coin.Symbol)
	if err != nil {
		return hold, err
	}
	rules, err := spotRules(coin.Strategy)
	if err != nil {
		return hold, err
	}
	env, priceFloat, err := spotRuleEnvironment(e, coin)
	if err != nil {
		return hold, err
	}
	price := decimal.NewFromFloat(priceFloat)
	profit := price.Sub(entry).Mul(quantity)
	roi := price.Sub(entry).Div(entry).Mul(decimal.NewFromInt(100))
	roiFloat, _ := roi.Float64()
	quantityFloat, _ := quantity.Float64()
	entryFloat, _ := entry.Float64()
	unrealizedFloat, _ := profit.Float64()
	if math.IsNaN(roiFloat) || math.IsInf(roiFloat, 0) ||
		math.IsInf(quantityFloat, 0) || math.IsInf(unrealizedFloat, 0) {
		return hold, fmt.Errorf("Spot strategy ROI or position is not finite")
	}
	position := SpotPositionCode{
		Symbol: coin.Symbol, Side: "LONG", Amount: quantityFloat,
		EntryPrice: entryFloat, MarkPrice: priceFloat,
		UnrealizedProfit: unrealizedFloat, CreateTime: snapshot.Position.CreatedAt,
	}
	env["ROI"] = roiFloat
	env["Position"] = position
	env["Positions"] = []SpotPositionCode{position}
	hold.ROI = roi.String()

	// Futures evaluates CLOSE_LONG as the permission to exit when a
	// configured ROI threshold is crossed, not an unconditional SELL.
	// The current Futures AutoStopOrder implementation always returns false.
	profitThreshold, err := checkedSpotThreshold(coin.Profit)
	if err != nil {
		return hold, err
	}
	lossThreshold, err := checkedSpotThreshold(coin.Loss)
	if err != nil {
		return hold, err
	}
	reason := ""
	switch {
	case roi.LessThanOrEqual(lossThreshold.Neg()):
		reason = "stop_loss"
	case roi.GreaterThanOrEqual(profitThreshold):
		reason = "take_profit"
	}
	if reason == "" {
		return hold, nil
	}
	hasCloseRule := false
	for _, rule := range rules {
		if !rule.Enable || rule.Type != "close_long" {
			continue
		}
		hasCloseRule = true
		match, err := spotRuleMatches(rule, env)
		if err != nil {
			return hold, err
		}
		if match {
			return SpotSignal{
				Action: "SELL", Reason: reason, RuleName: rule.Name,
				RuleHash: strategyservice.RuleHash(rule.Code), ROI: hold.ROI,
			}, nil
		}
	}
	// Like Futures CanOrderComplete, an enabled CLOSE_LONG suppresses
	// the simple Kline fallback even if it evaluates false.
	if hasCloseRule {
		return hold, nil
	}
	// Preserve Futures simpleCloseStrategy's +/-3% neutral zone. An
	// explicitly smaller threshold alone must not bypass this fallback.
	neutralBand := decimal.NewFromInt(3)
	if roi.GreaterThan(neutralBand.Neg()) && roi.LessThan(neutralBand) {
		return hold, nil
	}
	// When no explicit close rule exists, use the same simple LONG 5m
	// confirmation as Futures, but with a Spot-only data source. Any error
	// fails closed instead of selling on an invalid market feed.
	if e.LoadKlines == nil {
		return hold, fmt.Errorf("Spot fallback close Klines loader is missing")
	}
	klines, err := e.LoadKlines(coin.Symbol, "5m", 2)
	if err != nil {
		return hold, fmt.Errorf("Spot 5m close confirmation: %w", err)
	}
	if len(klines) != 2 || klines[0] == nil || klines[1] == nil {
		return hold, fmt.Errorf("Spot 5m close confirmation requires 2 Klines")
	}
	first, err := strconv.ParseFloat(klines[0].Close, 64)
	if err != nil || first <= 0 || math.IsInf(first, 0) || math.IsNaN(first) {
		return hold, fmt.Errorf("invalid Spot 5m close confirmation")
	}
	second, err := strconv.ParseFloat(klines[1].Close, 64)
	if err != nil || second <= 0 || math.IsInf(second, 0) || math.IsNaN(second) {
		return hold, fmt.Errorf("invalid Spot latest 5m close confirmation")
	}
	if first < second { // identical LONG fallback ordering as Futures
		return SpotSignal{
			Action: "SELL", Reason: reason, ROI: hold.ROI,
			RuleName: spotSimpleCloseRuleIdentity,
			RuleHash: strategyservice.RuleHash(spotSimpleCloseRuleIdentity),
		}, nil
	}
	return hold, nil
}

package agenttrade

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"go_binance_futures/models"
)

type PositionSnapshot struct {
	Symbol   string
	Side     string
	Quantity float64
	Price    float64
}

type OpenOrderSnapshot struct {
	Symbol       string
	Side         string
	PositionSide string
	Quantity     float64
	Price        float64
}

type RiskDataSource interface {
	Config(context.Context) (models.Config, error)
	Symbol(context.Context, string) (models.Symbols, error)
	Positions(context.Context) ([]PositionSnapshot, error)
	OpenOrders(context.Context) ([]OpenOrderSnapshot, error)
	EstimatedFillPrice(context.Context, string, string) (float64, error)
}

type RiskEngine struct {
	Store Store
	Data  RiskDataSource
	Now   func() time.Time
}

func (e RiskEngine) Check(ctx context.Context, proposal models.AgentTradeProposal) (RiskResult, error) {
	if e.Data == nil {
		return RiskResult{}, fmt.Errorf("risk data source is required")
	}
	nowFn := e.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	now := nowFn().UTC()
	result := RiskResult{Status: RiskPass, Checks: []RiskCheck{}, CheckedAt: now.UnixMilli()}
	add := func(name string, passed bool, message string) {
		result.Checks = append(result.Checks, RiskCheck{Name: name, Passed: passed, Message: message})
		if !passed {
			result.Status = RiskFail
		}
	}
	cfg, err := e.Data.Config(ctx)
	if err != nil {
		return result, fmt.Errorf("load risk config: %w", err)
	}
	add("kill_switch", cfg.AgentTradeExecutionEnable == 1, "AI trade execution kill switch is disabled")
	add("proposal_expiry", proposal.ExpiresAt > now.UnixMilli(), "proposal is expired")
	allowed := allowedSymbols(cfg.AgentTradeAllowedSymbols)
	add("allowed_symbol", allowed[proposal.Symbol], "symbol is not in the explicit AI trade allowlist")
	if proposal.Side == "LONG" {
		add("side_enabled", cfg.FutureAllowLong == 1, "long trading is disabled")
	}
	if proposal.Side == "SHORT" {
		add("side_enabled", cfg.FutureAllowShort == 1, "short trading is disabled")
	}
	currentMarketCondition := cfg.MarketCondition
	add("market_condition", marketAligned(proposal.Side, currentMarketCondition), "current MarketCondition conflicts with proposal direction")
	add("market_condition_drift", proposal.MarketCondition > 0 && currentMarketCondition == proposal.MarketCondition, "MarketCondition changed after the source analysis")

	symbol, symbolErr := e.Data.Symbol(ctx, proposal.Symbol)
	if symbolErr != nil {
		add("symbol_state", false, "symbol state unavailable")
		return result, nil
	}
	price, priceErr := strconv.ParseFloat(symbol.Close, 64)
	freshness := time.Duration(maxInt(cfg.AgentTradePriceFreshnessSec, 1)) * time.Second
	fresh := priceErr == nil && price > 0 && symbol.UpdateTime > 0 && now.Sub(time.UnixMilli(symbol.UpdateTime)) <= freshness
	add("price_freshness", fresh, "symbol price is stale or invalid")
	if priceErr == nil && price > 0 {
		result.ReferencePrice = price
	}

	entryOK := price > 0 && proposalContainsPrice(proposal, price)
	add("entry_condition", entryOK, "current price is outside every approved entry zone")
	stopOK := validStop(proposal.Side, price, proposal.StopLoss)
	add("stop_loss", stopOK, "stop loss is missing or on the wrong side of current price")

	fill, fillErr := e.Data.EstimatedFillPrice(ctx, proposal.Symbol, proposal.Side)
	if fillErr != nil || fill <= 0 || price <= 0 {
		add("slippage", false, "estimated executable price is unavailable")
	} else {
		result.EstimatedFillPrice = fill
		result.SlippageBps = math.Abs(fill-price) / price * 10000
		limit := float64(maxInt(cfg.AgentTradeMaxSlippageBps, 0))
		add("slippage", limit > 0 && result.SlippageBps <= limit, fmt.Sprintf("estimated slippage %.2f bps exceeds limit %.2f bps", result.SlippageBps, limit))
	}

	positions, posErr := e.Data.Positions(ctx)
	orders, ordErr := e.Data.OpenOrders(ctx)
	if posErr != nil || ordErr != nil {
		add("exposure_data", false, "positions or open orders are unavailable")
		return result, nil
	}
	exposure := 0.0
	duplicate := false
	activeCount := 0
	for _, p := range positions {
		if math.Abs(p.Quantity) <= 0 {
			continue
		}
		activeCount++
		exposure += math.Abs(p.Quantity * p.Price)
		if strings.EqualFold(p.Symbol, proposal.Symbol) {
			duplicate = true
		}
	}
	for _, o := range orders {
		if o.Quantity <= 0 {
			continue
		}
		activeCount++
		exposure += math.Abs(o.Quantity * o.Price)
		if strings.EqualFold(o.Symbol, proposal.Symbol) {
			duplicate = true
		}
	}
	result.CurrentExposureUSDT = exposure
	add("duplicate_order", !duplicate, "an existing position or open order already exists for this symbol")
	if cfg.FutureMaxCount > 0 {
		add("position_count", activeCount < cfg.FutureMaxCount, "position plus open-order count reached FutureMaxCount")
	}

	cooldown := time.Duration(maxInt(cfg.AgentTradeCooldownSec, 0)) * time.Second
	if cooldown > 0 {
		recent, err := e.Store.HasRecentExecution(ctx, proposal.Symbol, now.Add(-cooldown).UnixMilli())
		if err != nil {
			return result, err
		}
		add("cooldown", !recent, "a recent AI execution for this symbol is still cooling down")
	}

	maxLeverage := maxInt(cfg.AgentTradeMaxLeverage, 1)
	leverage := int(symbol.Leverage)
	if leverage <= 0 {
		leverage = 1
	}
	if leverage > maxLeverage {
		leverage = maxLeverage
	}
	result.Leverage = leverage
	add("leverage", leverage > 0 && leverage <= maxLeverage, "leverage exceeds configured maximum")

	if price > 0 && stopOK {
		stopDistance := math.Abs(price - proposal.StopLoss)
		maxRisk := cfg.AgentTradeMaxRiskUSDT
		maxNotional := cfg.AgentTradeMaxNotionalUSDT
		quantity := 0.0
		if stopDistance > 0 && maxRisk > 0 && maxNotional > 0 {
			quantity = math.Min(maxRisk/stopDistance, maxNotional/price)
			quantity = floorToStep(quantity, symbol.StepSize)
		}
		result.Quantity = quantity
		result.NotionalUSDT = quantity * price
		result.RiskUSDT = quantity * stopDistance
		add("single_trade_risk", quantity > 0 && result.RiskUSDT <= maxRisk+1e-8, "deterministic position sizing produced no valid quantity or exceeded max risk")
		add("single_trade_notional", quantity > 0 && result.NotionalUSDT <= maxNotional+1e-8, "single-trade notional exceeds configured maximum")
		totalLimit := cfg.AgentTradeMaxTotalExposureUSDT
		add("total_exposure", totalLimit > 0 && exposure+result.NotionalUSDT <= totalLimit+1e-8, "total futures exposure would exceed configured maximum")
	}
	return result, nil
}

func allowedSymbols(raw string) map[string]bool {
	out := map[string]bool{}
	for _, item := range strings.Split(raw, ",") {
		item = strings.ToUpper(strings.TrimSpace(item))
		if item != "" {
			out[item] = true
		}
	}
	return out
}

func marketAligned(side string, condition int) bool {
	if condition <= 0 {
		return false
	}
	switch strings.ToUpper(side) {
	case "LONG":
		return condition != 4 && condition != 5 && condition != 7 && condition != 9
	case "SHORT":
		return condition != 1 && condition != 2 && condition != 6 && condition != 8
	default:
		return false
	}
}

func withinEntry(price, low, high float64) bool {
	return price > 0 && low > 0 && high >= low && price >= low && price <= high
}

func validStop(side string, price, stop float64) bool {
	if price <= 0 || stop <= 0 {
		return false
	}
	if strings.EqualFold(side, "LONG") {
		return stop < price
	}
	if strings.EqualFold(side, "SHORT") {
		return stop > price
	}
	return false
}

func floorToStep(value float64, stepRaw string) float64 {
	step, err := strconv.ParseFloat(strings.TrimSpace(stepRaw), 64)
	if err != nil || step <= 0 {
		return 0
	}
	floored := math.Floor((value+1e-12)/step) * step
	if floored <= 0 {
		return 0
	}
	return floored
}

func maxInt(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func encodeRisk(result RiskResult) string {
	raw, _ := json.Marshal(result)
	return string(raw)
}

func proposalContainsPrice(proposal models.AgentTradeProposal, price float64) bool {
	type zone struct {
		Low  float64 `json:"low"`
		High float64 `json:"high"`
	}
	var zones []zone
	if strings.TrimSpace(proposal.EntryZonesJSON) != "" && json.Unmarshal([]byte(proposal.EntryZonesJSON), &zones) == nil {
		for _, item := range zones {
			if withinEntry(price, item.Low, item.High) {
				return true
			}
		}
		return false
	}
	return withinEntry(price, proposal.EntryLow, proposal.EntryHigh)
}

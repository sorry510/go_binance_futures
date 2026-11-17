package leadaccount

import (
	"context"
	"errors"
	"math"
	"strconv"
	"strings"

	binance "go_binance_futures/feature/api/binance"
	"go_binance_futures/models"
	"go_binance_futures/service/futuresownership"
)

// These methods adapt the existing shared futures strategy to the existing
// account-bound, durable Lead order executor. No independent strategy engine,
// endpoint, scheduler, trading permission or persisted configuration is added.
var ErrLeadStrategyInput = errors.New("invalid Lead shared strategy order")

// BoundTo prevents a strategy runner from mixing two distinct Lead credentials
// or accidentally using Main's bound position source.
func (a *LeadExecutionAdapter) BoundTo(account *binance.AccountClient) bool {
	return a != nil && account != nil && account.ID() == binance.LeadAccountID && a.account == account
}

func leadStrategyRiskCheck(coin *models.Symbols, side string, quantity, referencePrice float64, orderType string) (RiskOpenOrder, error) {
	if coin == nil || strings.TrimSpace(coin.Symbol) == "" ||
		!riskSide(side) || (orderType != "MARKET" && orderType != "LIMIT") ||
		!riskFinitePositive(quantity) || !riskFinitePositive(referencePrice) ||
		coin.Leverage < 1 || coin.Leverage > 125 {
		return RiskOpenOrder{}, ErrLeadStrategyInput
	}
	configured, err := strconv.ParseFloat(coin.Usdt, 64)
	if err != nil || !riskFinitePositive(configured) || math.IsNaN(configured) {
		return RiskOpenOrder{}, ErrLeadStrategyInput
	}
	if strings.TrimSpace(coin.MarginType) == "" {
		return RiskOpenOrder{}, ErrLeadStrategyInput
	}
	return RiskOpenOrder{
		AccountID: binance.LeadAccountID,
		Symbol:    strings.ToUpper(strings.TrimSpace(coin.Symbol)), Side: side,
		OrderType: orderType, Quantity: quantity, ReferencePrice: referencePrice,
		MarginType: coin.MarginType, Leverage: int(coin.Leverage), ConfiguredUSDT: configured,
	}, nil
}

// PreflightStrategyOpen is advisory only. ExecuteStrategyOpen ALWAYS runs the
// authoritative check again under the adapter lock before claiming an order.
func (a *LeadExecutionAdapter) PreflightStrategyOpen(ctx context.Context, coin *models.Symbols, side string, quantity, referencePrice float64, orderType string) error {
	check, err := leadStrategyRiskCheck(coin, side, quantity, referencePrice, orderType)
	if err != nil {
		return err
	}
	result, err := a.CheckOpen(ctx, check)
	if err != nil {
		return err
	}
	if !result.Allowed {
		return ErrLeadOpenBlocked
	}
	return nil
}

// This is ONLY a thin adapter to the already existing, hard-gated
// ExecuteOpen -> futuresownership.Executor -> Binance order broker.
func (a *LeadExecutionAdapter) ExecuteStrategyOpen(ctx context.Context, request futuresownership.OrderRequest, coin *models.Symbols, referencePrice float64) (futuresownership.ExchangeOrder, error) {
	check, err := leadStrategyRiskCheck(coin, request.PositionSide, request.Quantity, referencePrice, request.OrderType)
	if err != nil {
		return futuresownership.ExchangeOrder{}, err
	}
	if request.Symbol != check.Symbol || request.Owner != futuresownership.OwnerAutoStrategy ||
		request.Intent != futuresownership.IntentOpen {
		return futuresownership.ExchangeOrder{}, ErrLeadStrategyInput
	}
	return a.ExecuteOpen(ctx, request, check)
}

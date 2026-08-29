package market

import (
	"context"
	"fmt"
	"strings"

	"go_binance_futures/feature/api/binance"
	markettypes "go_binance_futures/types"
	"go_binance_futures/utils"

	"github.com/adshao/go-binance/v2/futures"
)

type Service struct{}

type Condition struct {
	MarketCondition int    `json:"market_condition"`
	Name            string `json:"name"`
	Auto            bool   `json:"auto"`
}

func (Service) Klines(ctx context.Context, symbol, interval string, limit int) ([]*futures.Kline, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	symbol, interval = strings.ToUpper(strings.TrimSpace(symbol)), strings.TrimSpace(interval)
	if symbol == "" || interval == "" {
		return nil, fmt.Errorf("symbol and interval are required")
	}
	if !validKlineInterval(interval) {
		return nil, fmt.Errorf("unsupported kline interval %q", interval)
	}
	if limit < 1 {
		limit = 150
	}
	if limit > 1000 {
		limit = 1000
	}
	result, err := binance.GetKlineData(symbol, interval, limit)
	if err != nil {
		return nil, fmt.Errorf("get klines: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (Service) FundingRate(ctx context.Context, symbol string) ([]*futures.PremiumIndex, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}
	result, err := binance.GetFundingRate(binance.FundingRateParams{Symbol: symbol})
	if err != nil {
		return nil, fmt.Errorf("get funding rate: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (Service) MarketCondition(ctx context.Context) (Condition, error) {
	if err := ctx.Err(); err != nil {
		return Condition{}, err
	}
	cfg, err := utils.GetSystemConfig()
	if err != nil {
		return Condition{}, fmt.Errorf("get system config: %w", err)
	}
	return Condition{MarketCondition: cfg.MarketCondition, Name: markettypes.MarketConditionName(cfg.MarketCondition), Auto: cfg.MarketConditionIsAuto == 1}, nil
}

func validKlineInterval(value string) bool {
	switch value {
	case "1m", "3m", "5m", "15m", "30m", "1h", "2h", "4h", "6h", "8h", "12h", "1d", "3d", "1w", "1M":
		return true
	default:
		return false
	}
}

package feature

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go_binance_futures/models"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/client/orm"
)

type mainOpenConfigSnapshotReader func(context.Context, string) (*models.Symbols, *models.Config, error)

// Main OPEN-only gate: keep the existing five-second successful Binance
// configuration cache, but never treat a failed margin/leverage RPC as success.
// Refresh symbol/global settings AFTER the remote call because the user may
// have changed the budget, leverage, mode or strategy during that request.
func ensureMainOpenConfiguration(ctx context.Context, coin *models.Symbols, config *models.Config) error {
	return ensureMainOpenConfigurationWith(ctx, coin, config, UpdateSymbolTradeInfoContext, readMainOpenConfigSnapshot)
}
func ensureMainOpenConfigurationWith(
	ctx context.Context,
	coin *models.Symbols,
	config *models.Config,
	configure func(context.Context, *models.Symbols) error,
	read mainOpenConfigSnapshotReader,
) error {
	if ctx == nil {
		return errors.New("missing Main futures entry context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if coin == nil || strings.TrimSpace(coin.Symbol) == "" || coin.Enable != 1 {
		return errors.New("Main futures entry symbol disabled or invalid")
	}
	if config == nil || config.FutureEnable != 1 {
		return errors.New("Main futures new entries globally disabled")
	}
	if configure == nil || read == nil {
		return errors.New("Main futures trade config verification unavailable")
	}
	// Check mode BEFORE Binance calls; do not silently convert invalid inputs
	// into ISOLATED, nor allow missing leverage to become zero-sized orders.
	if err := validateMainMarginLeverage(coin); err != nil {
		return err
	}
	if err := configure(ctx, coin); err != nil {
		return fmt.Errorf("Main futures exchange trade config not confirmed: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	current, latestConfig, err := read(ctx, coin.Symbol)
	if err != nil {
		return fmt.Errorf("Main futures final entry settings unavailable: %w", err)
	}
	return compareMainOpenConfigSnapshot(coin, current, config, latestConfig)
}
func validateMainMarginLeverage(coin *models.Symbols) error {
	if coin == nil || coin.Leverage <= 0 || (coin.MarginType != "ISOLATED" && coin.MarginType != "CROSSED") {
		return errors.New("Main futures margin type or leverage invalid")
	}
	return nil
}
func readMainOpenConfigSnapshot(ctx context.Context, symbol string) (*models.Symbols, *models.Config, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	var latest models.Symbols
	if err := orm.NewOrm().QueryTable(new(models.Symbols)).Filter("symbol", symbol).One(&latest); err != nil {
		return nil, nil, fmt.Errorf("Main futures symbol snapshot: %w", err)
	}
	config, err := utils.GetSystemConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("Main futures global switch snapshot: %w", err)
	}
	return &latest, &config, nil
}

// Only OPEN-critical fields are compared. Fast WS market fields (close,
// updateTime, volume etc.) change continuously and must NOT invalidate a
// signal when the user has not changed its trading configuration.
func compareMainOpenConfigSnapshot(before, after *models.Symbols, oldConfig, newConfig *models.Config) error {
	if before == nil || after == nil || oldConfig == nil || newConfig == nil {
		return errors.New("Main futures entry snapshot incomplete")
	}
	if newConfig.FutureEnable != 1 || after.Enable != 1 {
		return errors.New("Main futures entry switch disabled while configuring")
	}
	if after.Symbol != before.Symbol || after.ID != before.ID ||
		after.MarginType != before.MarginType || after.Leverage != before.Leverage ||
		after.Usdt != before.Usdt || after.TickSize != before.TickSize ||
		after.StepSize != before.StepSize || after.Strategy != before.Strategy ||
		after.Technology != before.Technology || after.Profit != before.Profit ||
		after.Loss != before.Loss || after.Type != before.Type ||
		after.KlineInterval != before.KlineInterval ||
		after.StrategyTemplateID != before.StrategyTemplateID {
		return errors.New("Main futures entry settings changed during exchange preflight")
	}
	if err := validateMainMarginLeverage(after); err != nil {
		return err
	}
	if oldConfig.FutureEnable != newConfig.FutureEnable ||
		oldConfig.FutureAllowLong != newConfig.FutureAllowLong ||
		oldConfig.FutureAllowShort != newConfig.FutureAllowShort ||
		oldConfig.FutureOrderType != newConfig.FutureOrderType ||
		oldConfig.FutureMaxCount != newConfig.FutureMaxCount ||
		oldConfig.LossMaxCount != newConfig.LossMaxCount {
		return errors.New("Main futures global risk settings changed during exchange preflight")
	}
	return nil
}

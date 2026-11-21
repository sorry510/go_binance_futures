package backtest

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
)

// BacktestSymbolDefaults reads the existing live symbol settings. No trading
// configuration is changed when these values are used by a backtest.
type BacktestSymbolDefaults struct {
	Symbol           string  `json:"symbol"`
	MarginUSDT       float64 `json:"margin_usdt"`
	Leverage         int     `json:"leverage"`
	TakeProfitPct    float64 `json:"take_profit_pct"`
	StopLossPct      float64 `json:"stop_loss_pct"`
	QuantityStepSize string  `json:"quantity_step_size"`
}

func GetBacktestSymbolDefaults(symbol string) (BacktestSymbolDefaults, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if len(symbol) < 5 || len(symbol) > 32 || !strings.HasSuffix(symbol, "USDT") {
		return BacktestSymbolDefaults{}, fmt.Errorf("invalid futures symbol")
	}
	for _, ch := range symbol {
		if (ch < 'A' || ch > 'Z') && (ch < '0' || ch > '9') {
			return BacktestSymbolDefaults{}, fmt.Errorf("invalid futures symbol")
		}
	}
	var row models.Symbols
	err := orm.NewOrm().QueryTable(new(models.Symbols)).Filter("symbol", symbol).One(&row)
	if err != nil {
		return BacktestSymbolDefaults{}, fmt.Errorf("load %s futures settings: %w", symbol, err)
	}
	readPositive := func(text string) float64 {
		value, parseErr := strconv.ParseFloat(strings.TrimSpace(text), 64)
		if parseErr != nil || value <= 0 || math.IsNaN(value) || math.IsInf(value, 0) {
			return 0
		}
		return value
	}
	return BacktestSymbolDefaults{
		Symbol: symbol, MarginUSDT: readPositive(row.Usdt),
		Leverage:      int(row.Leverage),
		TakeProfitPct: readPositive(row.Profit), StopLossPct: readPositive(row.Loss),
		QuantityStepSize: strings.TrimSpace(row.StepSize),
	}, nil
}

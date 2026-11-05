package feature

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"go_binance_futures/models"
	strategyservice "go_binance_futures/service/strategy"

	"github.com/beego/beego/v2/client/orm"
)

// finalizeTestStrategyClose atomically closes one simulated position. A close
// notification is permitted ONLY when this method returns updated=true.
// Restricting the write to close fields also avoids rewriting old snapshots.
func finalizeTestStrategyClose(o orm.Ormer, result *models.TestStrategyResults, price, netProfit float64, code, name, closeType string) (updated bool, err error) {
	if result == nil || result.ID <= 0 {
		return false, fmt.Errorf("invalid test position")
	}
	if !(price > 0) || math.IsNaN(price) || math.IsInf(price, 0) ||
		math.IsNaN(netProfit) || math.IsInf(netProfit, 0) {
		return false, fmt.Errorf("invalid simulated close price or profit")
	}
	if strings.TrimSpace(code) == "" {
		return false, fmt.Errorf("simulated close strategy is missing")
	}
	priceText := strconv.FormatFloat(price, 'f', -1, 64)
	profitText := strconv.FormatFloat(netProfit, 'f', 3, 64)
	name, closeType = strings.TrimSpace(name), strings.TrimSpace(closeType)
	if len([]rune(name)) > 128 || len([]rune(closeType)) > 32 {
		return false, fmt.Errorf("simulated close strategy metadata exceeds database limits")
	}
	hash := strategyservice.RuleHash(code)
	now := time.Now().UnixMilli()

	// Compare-and-set: the WHERE condition prevents duplicate closes and
	// duplicate notifications if two evaluation cycles race.
	affected, err := o.QueryTable(new(models.TestStrategyResults)).
		Filter("id", result.ID).Filter("close_price", "0").
		Update(orm.Params{
			"close_price": priceText, "close_profit": profitText,
			"close_strategy": code, "close_strategy_name": name,
			"close_strategy_type": closeType, "close_strategy_hash": hash,
			"updateTime": now,
		})
	if err != nil {
		return false, fmt.Errorf("persist simulated close for id=%d symbol=%s: %w", result.ID, result.Symbol, err)
	}
	if affected != 1 {
		// Another cycle has already closed/deleted the row. No notification.
		return false, nil
	}
	result.ClosePrice = priceText
	result.CloseProfit = profitText
	result.CloseStrategy = code
	result.CloseStrategyName = name
	result.CloseStrategyType = closeType
	result.CloseStrategyHash = hash
	result.UpdateTime = now
	return true, nil
}

package tradefills

import (
	"fmt"
	"strings"

	"github.com/beego/beego/v2/client/orm"
)

// RecentClosedSymbols reads immutable, exchange-authentic Fill trade_time.
// Managed order updated_at is NOT a trade clock: late WS events or REST
// backfills can arrive after the order has become terminal, while unrelated
// state updates can refresh a much older fill.
//
// Matching an exact client-order identity and account/symbol/intent prevents
// unrelated exchange trades (including unowned/manual fills) from silently
// participating in strategy cooldown. WS fills are real execution events
// even if their fee is not yet REST-verified.
func RecentClosedSymbols(o orm.Ormer, accountID string, afterMillis, untilMillis int64) (map[string]bool, error) {
	if accountID != "main" && accountID != "lead" {
		return nil, fmt.Errorf("unsupported cooldown account %q", accountID)
	}
	if afterMillis <= 0 || untilMillis < afterMillis {
		return nil, fmt.Errorf("invalid cooldown trade time range")
	}
	var rows []struct {
		Symbol string `orm:"column(symbol)"`
	}
	_, err := o.Raw(`
SELECT DISTINCT f.symbol AS symbol
FROM futures_trade_fills f
JOIN futures_managed_orders m
    ON m.client_order_id=f.client_order_id
   AND m.account_id=f.account_id
   AND m.symbol=f.symbol
   AND m.intent=f.intent
WHERE f.market=?
  AND f.account_id=?
  AND f.trade_time>=?
  AND f.trade_time<=?
  AND f.qty>0
  AND f.intent IN ('close','take_profit','stop_loss')
`, MarketUSDM, accountID, afterMillis, untilMillis).QueryRows(&rows)
	if err != nil {
		return nil, fmt.Errorf("load recent authentic closed fills: %w", err)
	}
	result := make(map[string]bool, len(rows))
	for _, row := range rows {
		symbol := strings.ToUpper(strings.TrimSpace(row.Symbol))
		if symbol != "" {
			result[symbol] = true
		}
	}
	return result, nil
}

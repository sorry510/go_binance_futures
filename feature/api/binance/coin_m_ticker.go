package binance

import (
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/adshao/go-binance/v2/delivery"
	"github.com/beego/beego/v2/client/orm"
)

// persistCoinMTickers processes public price events only. It never changes
// per-symbol enable, budget, strategy, or any account order/position record.
func persistCoinMTickers(o orm.Ormer, tickers delivery.WsAllMarketTickerEvent) (int, error) {
	var changed int
	var failures []error
	for _, tick := range tickers {
		if tick == nil || !strings.HasSuffix(tick.Symbol, "_PERP") || tick.Time <= 0 {
			continue
		}
		price, ok := new(big.Rat).SetString(tick.ClosePrice)
		if !ok || price.Sign() <= 0 {
			continue
		}
		result, err := o.Raw(
			"UPDATE `delivery_symbols` SET `percentChange`=?, `close`=?, `open`=?, `low`=?, `high`=?, `updateTime`=?, `baseVolume`=?, `quoteVolume`=?, `closeQty`=?, `tradeCount`=?, `lastClose`=close, `lastUpdateTime`=updateTime WHERE `symbol`=? AND `contract_type`='PERPETUAL'",
			tick.PriceChangePercent, tick.ClosePrice, tick.OpenPrice,
			tick.LowPrice, tick.HighPrice, tick.Time, tick.BaseVolume,
			tick.QuoteVolume, tick.CloseQty, tick.TradeCount, tick.Symbol,
		).Exec()
		if err != nil {
			failures = append(failures, fmt.Errorf("%s ticker update: %w", tick.Symbol, err))
			continue
		}
		rows, err := result.RowsAffected()
		if err != nil {
			failures = append(failures, fmt.Errorf("%s ticker rows: %w", tick.Symbol, err))
			continue
		}
		changed += int(rows)
	}
	return changed, errors.Join(failures...)
}

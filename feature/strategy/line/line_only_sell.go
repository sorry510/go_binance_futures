package line

import (
	"go_binance_futures/feature/api/binance"
	"strconv"

	"github.com/adshao/go-binance/v2/futures"
)

type TradeLine0 struct {
}

// 适合震荡行情
// 检查最后的价格于上一个3min线的平均价格的变化幅度是否大于2%,进行反买
func (TradeLine0 TradeLine0) GetCanLongOrShort(symbol string) (canLong bool, canShort bool) {
	return false, false
}

// 达到止盈或止损后判断是否可以平仓
// 3min 最新价格是否跌破前一个3min的收盘价
func (TradeLine0 TradeLine0) CanOrderComplete(symbol string, positionSide string) (complete bool) {
	lines, err := binance.GetKlineData(symbol, "3m", 2)
	if err != nil {
		return true
	}
	close0, _ := strconv.ParseFloat(lines[0].Close, 64)
	close1, _ := strconv.ParseFloat(lines[1].Close, 64)
	if positionSide == "LONG" {
		return close0 < close1 // 价格在下跌中
	} else if positionSide == "SHORT" {
		return close0 > close1 // 价格在上涨中
	} else {
		return false
	}
}

// 达到止盈或止损前判定是否可以平仓
func (TradeLine0 TradeLine0) AutoStopOrder(position *futures.PositionRisk, nowProfit float64) (stop bool) {
	return false
}

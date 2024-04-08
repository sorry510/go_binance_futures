package line

import (
	"go_binance_futures/feature/api/binance"
	"strconv"

	"github.com/adshao/go-binance/v2/futures"
)

// var hold_max_time, _ = config.String("coin::hold_max_time")
// var hold_max_time_float64, _ = strconv.ParseFloat(hold_max_time, 64)
// var profit, _ = config.String("trade::profit")
// var profit_float64, _ = strconv.ParseFloat(profit, 64)

// var loss, _ = config.String("trade::loss")
// var loss_float64, _ = strconv.ParseFloat(loss, 64)

type TradeLine1 struct {
}

func (tradeLine1 TradeLine1) GetCanLongOrShort(symbol string) (canLong bool, canShort bool) {
	kline_3m, err := binance.GetKlineData(symbol, "3m", 50)
	if err != nil {
		return false, false
	}
	line3m_result := normalizationLineData(kline_3m)
	if checkLongLine3m(line3m_result) {
		return true, false
	}
	if checkShortLine3m(line3m_result) {
		return false, true
	}
	return false, false
}

// 达到止盈或止损后判断是否可以平仓
func (tradeLine1 TradeLine1) CanOrderComplete(symbol string, positionSide string) (complete bool) {
	lines, err := binance.GetKlineData(symbol, "1m", 2) // 1min 线最近2条
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

func (tradeLine1 TradeLine1) AutoStopOrder(position *futures.PositionRisk, nowProfit float64) (stop bool) {
	// symbol := position.Symbol
	// side := position.PositionSide
	// // if hold_max_time_float64 > 0 {
	// // 	// 最大持仓时间(api的bug，没有UpdateTime字段)
	// // 	// if position.UpdateTime+hold_max_time_float64*60*60*1000 < time.Now().UnixNano()/1e6 {
	// // 	// 	return true
	// // 	// }
	// // }
	// if side == "LONG" {
	// 	if (nowProfit < profit_float64 * 0.6 && nowProfit > 0) || nowProfit < -profit_float64 {
	// 		return true
	// 	}
	// }
	return false
}

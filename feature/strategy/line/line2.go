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

type TradeLine2 struct {
}

func (TradeLine2 TradeLine2) GetCanLongOrShort(symbol string) (canLong bool, canShort bool) {
	kline_6h, err1 := binance.GetKlineData(symbol, "6h", 100)
	if err1 != nil {
		return false, false
	}
	kline_6h_close := GetLineClosePrices(kline_6h)
	
	ema6h_3, _ := CalculateExponentialMovingAverage(kline_6h_close, 3) // ma3
	ema6h_7, _ := CalculateExponentialMovingAverage(kline_6h_close, 7) // ma7
	ema6h_15, _ := CalculateExponentialMovingAverage(kline_6h_close, 15) // ma15
	rsi6, _ := CalculateRSI(kline_6h_close, 6) // rsi6
	rsi14, _ := CalculateRSI(kline_6h_close, 14) // rsi14
	if Kdj(ema6h_3, ema6h_7, 4, "LANG") && Kdj(ema6h_7, ema6h_15, 4, "LANG") && rsi6[1] < 0.75 && rsi14[1] < 0.7 { // 1天之内发生过金叉, rsi 没有超买
		// 短线穿越长线金叉
		return true, false
	} else if (Kdj(ema6h_3, ema6h_7, 4, "SHORT") && Kdj(ema6h_7, ema6h_15, 4, "SHORT")) && rsi6[1] < 0.75 && rsi14[1] < 0.7 {
		return false, true
	}
	return false, false
}

// 达到止盈或止损后判断是否可以平仓
func (TradeLine2 TradeLine2) CanOrderComplete(symbol string, positionSide string) (complete bool) {
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

func (TradeLine2 TradeLine2) AutoStopOrder(position *futures.PositionRisk, nowProfit float64) (stop bool) {
	// symbol := position.Symbol
	// side := position.PositionSide
	// // if hold_max_time_float64 > 0 {
	// // 	// 最大持仓时间(api的bug，没有UpdateTime字段)
	// // 	// if position.UpdateTime+hold_max_time_float64*60*60*1000 < time.Now().UnixNano()/1e6 {
	// // 	// 	return true
	// // 	// }
	// // }
	// if side == "LONG" {xw
	// 	if (nowProfit < profit_float64 * 0.6 && nowProfit > 0) || nowProfit < -profit_float64 {
	// 		return true
	// 	}
	// }
	return false
}

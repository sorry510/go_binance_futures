package line

import (
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/utils"
	"strconv"

	"github.com/adshao/go-binance/v2/futures"
)

// var hold_max_time, _ = config.String("coin::hold_max_time")
// var hold_max_time_float64, _ = strconv.ParseFloat(hold_max_time, 64)
// var profit, _ = config.String("trade::profit")
// var profit_float64, _ = strconv.ParseFloat(profit, 64)

// var loss, _ = config.String("trade::loss")
// var loss_float64, _ = strconv.ParseFloat(loss, 64)

type TradeLine3 struct {
}

// 6 小时ma线金叉
func (TradeLine3 TradeLine3) GetCanLongOrShort(symbol string) (canLong bool, canShort bool) {
	kline_6h, err1 := binance.GetKlineData(symbol, "6h", 50)
	if err1 != nil {
		return false, false
	}
	kline_6h_close := GetLineClosePrices(kline_6h)
	
	ma6h_3, _ := CalculateSimpleMovingAverage(kline_6h_close, 3) // ma3
	ma6h_7, _ := CalculateSimpleMovingAverage(kline_6h_close, 7) // ma7
	ma6h_15, _ := CalculateSimpleMovingAverage(kline_6h_close, 15) // ma15
	rsi6, _ := CalculateRSI(kline_6h_close, 6) // rsi6
	rsi14, _ := CalculateRSI(kline_6h_close, 14) // rsi14
	if (rsi6 == nil || rsi14 == nil || len(rsi6) < 2 || len(rsi14) < 2) {
		// 开盘小于 4.5 天
		return false, false
	}
	isRsi := rsi6[1] < 80 && rsi6[1] > 30 && rsi14[1] < 75 && rsi14[1] > 28
	// logs.Info(symbol, Kdj(ma6h_3, ma6h_7, 4), Kdj(ma6h_7, ma6h_3, 4), utils.IsDesc(ma6h_3[0:2]), rsi6[1], rsi14[1])
	if Kdj(ma6h_3, ma6h_7, 4) && Kdj(ma6h_3, ma6h_15, 4) && utils.IsDesc(ma6h_3[0:2]) && isRsi { // 1天之内发生过金叉, rsi 没有超买
		// 短线穿越长线金叉
		return true, false
	}
	if Kdj(ma6h_7, ma6h_3, 4) && Kdj(ma6h_15, ma6h_3, 4) && utils.IsAsc(ma6h_3[0:2]) && isRsi {
		return false, true
	}
	return false, false
}

// 达到止盈或止损后判断是否可以平仓
func (TradeLine3 TradeLine3) CanOrderComplete(symbol string, positionSide string) (complete bool) {
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

func (TradeLine3 TradeLine3) AutoStopOrder(position *futures.PositionRisk, nowProfit float64) (stop bool) {
	if nowProfit < 3 || nowProfit > -3 {
		return false
	}
	return TradeLine3.MarketReversal(position.Symbol, position.PositionSide)
}

func (TradeLine3 TradeLine3) MarketReversal(symbol string, positionSide string) (isReversal bool) {
	kline_1d, err1 := binance.GetKlineData(symbol, "1d", 50)
	if err1 != nil {
		return false
	}
	kline_1d_close := GetLineClosePrices(kline_1d)
	
	ma1d_3, _ := CalculateSimpleMovingAverage(kline_1d_close, 3) // ma3
	ma1d_7, _ := CalculateSimpleMovingAverage(kline_1d_close, 7) // ma7
	ma1d_15, _ := CalculateSimpleMovingAverage(kline_1d_close, 15) // ma15
	
	if positionSide== "LONG" {
		if Kdj(ma1d_7, ma1d_3, 4) && Kdj(ma1d_15, ma1d_3, 4) && utils.IsAsc(ma1d_3[0:3]) {
			return true
		}
	}
	if positionSide == "SHORT" {
		if Kdj(ma1d_3, ma1d_7, 4) && Kdj(ma1d_3, ma1d_15, 4) && utils.IsDesc(ma1d_3[0:3]) {
			return true
		}
	}
	return false
}

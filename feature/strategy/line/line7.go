package line

import (
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/utils"
	"strconv"

	"github.com/adshao/go-binance/v2/futures"
)

type TradeLine7 struct {
}

// 交易逻辑: 根据肯纳特通道判断是否可以开仓
// kc1(50, 2.75)
// kc2(50, 3.75)
// 做多逻辑
// 1. 价格跌破最低轨道(kc2下轨)，然后突破到次低轨道(kc1下轨)时，做多，建议止盈50%位置在 kc1 中轨附近位置，剩余 50% 止盈50%位置在 kc1 上轨附近位置
// 做空相反
func (TradeLine TradeLine7) GetCanLongOrShort(symbol string) (canLong bool, canShort bool) {
	limit := 150
	period := 50 
	multiplier1 := 2.75 // 窄通道
	multiplier2 := 3.75 // 宽通道
	kline_1, err := binance.GetKlineData(symbol, "4h", limit)
	if err != nil {
		return false, false
	}
	kline_2, err := binance.GetKlineData(symbol, "1d", limit)
	if err != nil {
		return false, false
	}
	// kline_3, err := binance.GetKlineData(symbol, "1d", limit)
	// if err != nil {
	// 	return false, false
	// }
	
	if len(kline_1) < limit || len(kline_2) < limit {
		return false, false
	}
	
	high1, low1, close1, _ := GetLineFloatPrices(kline_1)
	upper1, _, lower1 := CalculateKeltnerChannels(high1, low1, close1, period, multiplier1) // kc1
	upper2, _, lower2 := CalculateKeltnerChannels(high1, low1, close1, period, multiplier2) // kc2
	
	close2 := GetLineClosePrices(kline_2)
	limitPeriod := 12 // 最近n根k线
	
	// 之前的最低价格跌破了 kc2 的下轨，然后当前价格超越了 kc1 下轨，止损位置在 kc1 下轨附近位置，止盈50%位置在 kc1 中规附近位置，剩余 50% 止盈50%位置在 kc1 上轨附近位置
	// 大级别看起来是上升通道
	if (close1[0] > lower1[0] && close1[1] < lower1[1]) {
		for i := 2; i < limitPeriod; i++ {
			// 最近10根k线最低价格在kc2下轨之下
			if low1[i] < lower2[i] {
				// 大级别看起来是上升通道
				if (utils.IsDesc(close2[0:3])) {
					return true, false
				}
			}
		}
	}
	
	// 之前的最高价格超越了 kc2 的上轨，然后当前价格跌破了 kc1 上轨，止损位置在 kc1 上轨附近位置，止盈50%位置在 kc1 中规附近位置，剩余 50% 止盈50%位置在 kc1 下轨附近位置
	// 大级别看起来是下降通道
	if (close1[0] < upper1[0] && close1[1] > upper1[1]) {
		for i := 1; i < limitPeriod; i++ {
			// 最近10根k线最高价格在kc2上轨之上
			if high1[i] > upper2[i] {
				// 大级别看起来是下降通道
				if (utils.IsAsc(close2[0:3])) {
					return false, true
				}
			}
		}
	}

	return false, false
}

func (TradeLine TradeLine7) CanOrderComplete(symbol string, positionSide string) (complete bool) {
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
// 1. 1天的kline线，ma7和ma3金叉，ma15和ma3金叉，ma3线3连跌
func (TradeLine TradeLine7) AutoStopOrder(position *futures.PositionRisk, nowProfit float64) (stop bool) {
	if nowProfit < 3 || nowProfit > -3 {
		return false
	}
	return TradeLine.MarketReversal(position.Symbol, position.PositionSide)
}

func (TradeLine TradeLine7) MarketReversal(symbol string, positionSide string) (isReversal bool) {
	// kline_1d, err1 := binance.GetKlineData(symbol, "1d", 50)
	// if err1 != nil {
	// 	return false
	// }
	// kline_1d_close := GetLineClosePrices(kline_1d)
	
	// ma1d_3, _ := CalculateSimpleMovingAverage(kline_1d_close, 3) // ma3
	// ma1d_7, _ := CalculateSimpleMovingAverage(kline_1d_close, 7) // ma7
	// ma1d_15, _ := CalculateSimpleMovingAverage(kline_1d_close, 15) // ma15
	
	// if positionSide== "LONG" {
	// 	if Kdj(ma1d_7, ma1d_3, 4) && Kdj(ma1d_15, ma1d_3, 4) && utils.IsAsc(ma1d_3[0:3]) {
	// 		return true
	// 	}
	// }
	// if positionSide == "SHORT" {
	// 	if Kdj(ma1d_3, ma1d_7, 4) && Kdj(ma1d_3, ma1d_15, 4) && utils.IsDesc(ma1d_3[0:3]) {
	// 		return true
	// 	}
	// }
	return false
}

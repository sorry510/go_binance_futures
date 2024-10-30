package line

import (
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/utils"
	"math"
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

// 交易逻辑: 主要看 4h 线
// 做多逻辑
// 1. rsi < 70
// 2. 2小时线最低点在 11个之内，最低点到最低点+8个line里面至少6个是红线，最低点事红线，(下影线长度 / 实体长度) > 0.5
// 3. rsi6 < 80, rsi14 < 75
// 4. 4个line之内，4小时线产生金叉
// 做空相反
func (TradeLine3 TradeLine3) GetCanLongOrShort(symbol string) (canLong bool, canShort bool) {
	limit := 50
	kline_interval1 := "4h"
	rsi_period1 := 14
	ema_period1 := 3
	ema_period2 := 7
	
	kline_1, err := binance.GetKlineData(symbol, kline_interval1, limit)
	if err != nil {
		return false, false
	}
	_, _, close1 := GetLineFloatPrices(kline_1) // high, low, close
	
	ema1, _ := CalculateExponentialMovingAverage(close1, ema_period1)
	ema2, _ := CalculateExponentialMovingAverage(close1, ema_period2)
	
	lineData := normalizationLineData(kline_1) // 归一化处理数据
	
	rsi1, _ := CalculateRSI(close1, rsi_period1) // 获取 rsi
	
	if ((Kdj(ema1, ema2, 3) && rsi1[0] < 42) || (lineData.Line[0].Position == "LONG" && rsi1[0] < 23)) &&
		TradeLine3.checkLongLine(lineData) {
		// 产生金叉 或 rsi 超卖了
		return true, false		
	}
	if ((Kdj(ema2, ema1, 3) && rsi1[0] < 42) || (lineData.Line[0].Position == "SHORT" && rsi1[0] > 77)) &&
		// 产生金叉 或 rsi 超卖了
		TradeLine3.checkShortLine(lineData) {
		return false, true
	}
	
	return false, false
}

// 达到止盈或止损后判断是否可以平仓
// 5min 最新价格是否跌破前一个5min的收盘价
func (TradeLine3 TradeLine3) CanOrderComplete(symbol string, positionSide string) (complete bool) {
	lines, err := binance.GetKlineData(symbol, "5m", 2)
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

func (TradeLine3 TradeLine3) checkLongLine(lineData *LineData) bool {
	minIndex := lineData.MinIndex
	line := lineData.Line
	if minIndex >= 0 && minIndex <= 8 {
		linePoint := line[minIndex] // 最低的那个line
		underLength := math.Abs(linePoint.Close - linePoint.Low) // 下影线长度
		entityLength := math.Abs(linePoint.Open - linePoint.Close) // 实体长度
		if entityLength / linePoint.Close > 0.02 && // 涨跌幅度 2% 以上
		   ((underLength / entityLength) < 0.15 || (underLength / entityLength) > 0.6) { // 十字形形态或者大跌形态
			return true
		}
	}
	return false
}

func (TradeLine3 TradeLine3) checkShortLine(lineData *LineData) bool {
	maxIndex := lineData.MaxIndex
	line := lineData.Line
	if maxIndex >= 0 && maxIndex <= 8 {
		linePoint := line[maxIndex] // 最高的那个line
		upperLength := math.Abs(linePoint.High - linePoint.Close) // 上影线长度
		entityLength := math.Abs(linePoint.Open - linePoint.Close) // 实体长度
		if	entityLength / linePoint.Close > 0.02 && // 涨跌幅度 2% 以上
			((upperLength / entityLength) < 0.15 || (upperLength / entityLength) > 0.6) { // 十字形形态或者大跌形态
			return true
		}
	}
	return false
}



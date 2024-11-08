package line

import (
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/utils"
	"math"
	"strconv"

	"github.com/adshao/go-binance/v2/futures"
)

type TradeLine4 struct {
}

// 交易逻辑: 看的是 6h k线 和 2h k线
// 做多逻辑
// 1. 在4个line线之内，6小时线产生金叉
// 2. 2小时线最低点在 11个之内，最低点到最低点+8个line里面至少6个是红线，最低点事红线，(下影线长度 / 实体长度) > 0.5
// 3. rsi6 < 80, rsi14 < 75
// 4. 基本盘逻辑: btc 的跌幅大于 5%，当前所有币种跌的数量>= 75% 时禁止做多，反之禁止做空
// 做空相反
func (TradeLine4 TradeLine4) GetCanLongOrShort(symbol string) (canLong bool, canShort bool) {
	kline_6h, err1 := binance.GetKlineData(symbol, "6h", 50)
	kline_1h, err2 := binance.GetKlineData(symbol, "2h", 24)
	if err1 != nil || err2 != nil {
		return false, false
	}
	kline_6h_close := GetLineClosePrices(kline_6h)
	
	ma6h_3, _ := CalculateSimpleMovingAverage(kline_6h_close, 3) // ma3
	ma6h_7, _ := CalculateSimpleMovingAverage(kline_6h_close, 7) // ma7
	rsi6, _ := CalculateRSI(kline_6h_close, 6) // rsi6
	rsi14, _ := CalculateRSI(kline_6h_close, 14) // rsi14
	if (rsi6 == nil || rsi14 == nil || len(rsi6) < 2 || len(rsi14) < 2) {
		// 开盘小于 4.5 天
		return false, false
	}
	baseCanLong, baseCanShort := BaseCheckCanLongOrShort() // 基本盘
	isRsi := rsi6[0] < 80 && rsi6[0] > 30 && rsi14[0] < 75 && rsi14[0] > 28
	// logs.Info(symbol, Kdj(ma6h_3, ma6h_7, 4), Kdj(ma6h_7, ma6h_3, 4), rsi6[1], rsi14[1])
	if Kdj(ma6h_3, ma6h_7, 4) && TradeLine4.checkLongLine(kline_1h) && isRsi && baseCanLong{ // 1天之内发生过金叉, rsi 没有超买
		// 短线穿越长线金叉
		return true, false
	}
	if Kdj(ma6h_7, ma6h_3, 4) && TradeLine4.checkShortLine(kline_1h)&& isRsi && baseCanShort {
		return false, true
	}
	return false, false
}

// 达到止盈或止损后判断是否可以平仓
// 5min 最新价格是否跌破前一个5min的收盘价
func (TradeLine4 TradeLine4) CanOrderComplete(symbol string, positionSide string) (complete bool) {
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
func (TradeLine4 TradeLine4) AutoStopOrder(position *futures.PositionRisk, nowProfit float64) (stop bool) {
	if nowProfit < 3 || nowProfit > -3 {
		return false
	}
	return TradeLine4.MarketReversal(position.Symbol, position.PositionSide)
}

func (TradeLine4 TradeLine4) MarketReversal(symbol string, positionSide string) (isReversal bool) {
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

func (TradeLine4 TradeLine4) checkLongLine(klines []*futures.Kline) bool {
	lineData := normalizationLineData(klines) // 24条线
	minIndex := lineData.MinIndex
	line := lineData.Line
	if minIndex >= 1 && minIndex <= 11 {
		linePoint := line[minIndex] // 最低的那个line
		underLength := math.Abs(linePoint.Close - linePoint.Low) // 下影线长度
		entityLength := math.Abs(linePoint.Open - linePoint.Close) // 实体长度
		if	getRightLine(line[minIndex:minIndex+8], "SHORT") && // 最低点到最低点+8个line里面至少6个是红线
			linePoint.Position == "SHORT" && // 最低点的line是跌
			(underLength / entityLength) > 0.5 { // 下影线长度  实体长度
				return true
		}
	}
	return false
}

func (TradeLine4 TradeLine4) checkShortLine(klines []*futures.Kline) bool {
	lineData := normalizationLineData(klines) // 24条线
	maxIndex := lineData.MaxIndex
	line := lineData.Line
	if maxIndex >= 1 && maxIndex <= 11 {
		linePoint := line[maxIndex] // 最高的那个line
		upperLength := math.Abs(linePoint.High - linePoint.Close) // 上影线长度
		entityLength := math.Abs(linePoint.Open - linePoint.Close) // 实体长度
		if	getRightLine(line[maxIndex:maxIndex+8], "LONG") && // 最低点到最低点+8个line里面至少6个是绿线
			linePoint.Position == "LONG" && // 最低点的line是涨
			(upperLength / entityLength) > 0.5 { // 上影线长度 > 实体长度
				return true
		}
	}
	return false
}


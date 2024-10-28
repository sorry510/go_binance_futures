package line

import (
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/utils"
	"strconv"

	"github.com/adshao/go-binance/v2/futures"
)

type TradeLine6 struct {
}

// 交易逻辑: 看的是 3min 线，高频交易，适合震荡行情，猛涨时做空，反之做多
// 做多逻辑
// 1. 当前价格 < 3min前k线的开盘价格
// 2. 最近3钟的变化幅度大于 0.15%
// 做空相反
func (TradeLine6 TradeLine6) GetCanLongOrShort(symbol string) (canLong bool, canShort bool) {
	kline_1, err1 := binance.GetKlineData(symbol, "3m", 20)
	if err1 != nil {
		return false, false
	}
	
	lastOpenPrice, _ := strconv.ParseFloat(kline_1[0].Open, 64)
	nowPrice, _ := strconv.ParseFloat(kline_1[0].Close, 64)
	
	percentLimit := 0.015 // 变化幅度
	
	if (nowPrice > lastOpenPrice) && (nowPrice - lastOpenPrice) / lastOpenPrice >= percentLimit {
		return false, true
	}
	if (nowPrice < lastOpenPrice) && (lastOpenPrice - nowPrice) / lastOpenPrice >= percentLimit {
		return true, false
	}

	return false, false
}

// 达到止盈或止损后判断是否可以平仓
// 3min 最新价格是否跌破前一个3min的收盘价
func (TradeLine6 TradeLine6) CanOrderComplete(symbol string, positionSide string) (complete bool) {
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
func (TradeLine6 TradeLine6) AutoStopOrder(position *futures.PositionRisk, nowProfit float64) (stop bool) {
	if nowProfit < 3 || nowProfit > -3 {
		return false
	}
	return TradeLine6.MarketReversal(position.Symbol, position.PositionSide)
}

func (TradeLine6 TradeLine6) MarketReversal(symbol string, positionSide string) (isReversal bool) {
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

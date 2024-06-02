package line

import (
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/utils"
	"strconv"

	"github.com/adshao/go-binance/v2/futures"
)

type TradeLine5 struct {
}

// 检查最后的价格于上一个1min线的平均价格的变化幅度是否大于2%
// todo 判断当前是否是涨幅过高，短暂回调
func (TradeLine5 TradeLine5) GetCanLongOrShort(symbol string) (canLong bool, canShort bool) {
	kline_1, err1 := binance.GetKlineData(symbol, "1m", 30)
	if err1 != nil {
		return false, false
	}
	if checkLine(kline_1) {
		return false, false
	}
	
	lastOpenPrice, _ := strconv.ParseFloat(kline_1[1].Open, 64) // 1min 前的价格
	nowPrice, _ := strconv.ParseFloat(kline_1[0].Close, 64)
	
	percentLimit := 0.0088 // 变化幅度
	
	if (nowPrice > lastOpenPrice) && (nowPrice - lastOpenPrice) / lastOpenPrice >= percentLimit {
		return true, false
	}
	if (nowPrice < lastOpenPrice) && (lastOpenPrice - nowPrice) / lastOpenPrice >= percentLimit {
		return false, true
	}

	return false, false
}

// 达到止盈或止损后判断是否可以平仓
// 5min 最新价格是否跌破前一个5min的收盘价
func (TradeLine5 TradeLine5) CanOrderComplete(symbol string, positionSide string) (complete bool) {
	lines, err := binance.GetKlineData(symbol, "1m", 2)
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
func (TradeLine5 TradeLine5) AutoStopOrder(position *futures.PositionRisk, nowProfit float64) (stop bool) {
	if nowProfit < 3 || nowProfit > -3 {
		return false
	}
	return TradeLine5.MarketReversal(position.Symbol, position.PositionSide)
}

func (TradeLine5 TradeLine5) MarketReversal(symbol string, positionSide string) (isReversal bool) {
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

func checkLine(kLines []*futures.Kline) bool {
	// 判定是否最近有个一次突变
	for _, item := range kLines {
		open, _ := strconv.ParseFloat(item.Open, 64)
		high, _ := strconv.ParseFloat(item.High, 64)
		low, _ := strconv.ParseFloat(item.Low, 64)
		if (high - low) / open >= 0.015 {
			return true
		}
	}
	return false
}

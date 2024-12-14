package line

import (
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/feature/strategy"
	"go_binance_futures/utils"
	"strconv"

	"github.com/adshao/go-binance/v2/futures"
)

type TradeLine5 struct {
}

// 交易逻辑: 看的是 1min 线，高频逻辑，适合单边行情，右侧交易，猛涨时追涨，反之依然
// 做多逻辑
// 1. 当前价格 > 1min前k线的开盘价格
// 2. 最近一分钟的变化幅度大于 0.9%
// 做空相反
func (TradeLine5 TradeLine5) GetCanLongOrShort(openParams strategy.OpenParams) (openResult strategy.OpenResult) {
	symbols := openParams.Symbols
	openResult.CanLong = false
	openResult.CanShort = false
	
	kline_1, err1 := binance.GetKlineData(symbols.Symbol, "1m", 30)
	if err1 != nil {
		return openResult
	}
	if TradeLine5.checkLine(kline_1) {
		return openResult
	}
	
	lastOpenPrice, _ := strconv.ParseFloat(kline_1[1].Open, 64) // 1min 前的价格
	nowPrice, _ := strconv.ParseFloat(kline_1[0].Close, 64)
	
	percentLimit := 0.009 // 变化幅度
	
	if (nowPrice > lastOpenPrice) && (nowPrice - lastOpenPrice) / lastOpenPrice >= percentLimit {
		openResult.CanLong = true
		return openResult
	}
	if (nowPrice < lastOpenPrice) && (lastOpenPrice - nowPrice) / lastOpenPrice >= percentLimit {
		openResult.CanShort = true
		return openResult
	}
	return openResult
}

// 达到止盈或止损后判断是否可以平仓
// 3min 最新价格是否跌破前一个3min的收盘价
func (TradeLine5 TradeLine5) CanOrderComplete(closeParams strategy.CloseParams) (closeResult strategy.CloseResult) {
	symbols := closeParams.Symbols // 交易对
	position := closeParams.Position // 当前仓位
	closeResult.Complete = false
	
	lines, err := binance.GetKlineData(symbols.Symbol, "5m", 2)
	if err != nil {
		closeResult.Complete = true
		return closeResult
	}
	close0, _ := strconv.ParseFloat(lines[0].Close, 64)
	close1, _ := strconv.ParseFloat(lines[1].Close, 64)
	if position.Side == "LONG" {
		closeResult.Complete = close0 < close1 // 价格在下跌中
	} else if position.Side == "SHORT" {
		closeResult.Complete = close0 > close1 // 价格在上涨中
	} else {
		closeResult.Complete = true
	}
	return closeResult
}

// 达到止盈或止损前判定是否可以平仓
// 1. 1天的kline线，ma7和ma3金叉，ma15和ma3金叉，ma3线3连跌
func (TradeLine5 TradeLine5) AutoStopOrder(closeParams strategy.CloseParams) (closeResult strategy.CloseResult) {
	position := closeParams.Position // 当前仓位
	closeResult.Complete = false
	
	if closeParams.NowProfit < 3 || closeParams.NowProfit > -3 {
		closeResult.Complete = false
		return closeResult
	}
	closeResult.Complete = TradeLine5.MarketReversal(position.Symbol, position.Side)
	return closeResult
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


func (TradeLine5 TradeLine5) checkLine(kLines []*futures.Kline) bool {
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

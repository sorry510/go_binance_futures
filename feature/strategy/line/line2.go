package line

import (
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/feature/strategy"
	"go_binance_futures/utils"
	"strconv"
)

// 交易逻辑: 看的是 6h k线
// 买入逻辑
// 1. 在4个line线之内，只发生一次金叉，7日 ema 金叉 15日 ema， 3 日 ema 金叉 7 日 ema
// 2. rsi6 < 80, rsi14 < 75
// 3. 3日 ema 最近3条线是上涨的
// 做空相反
type TradeLine2 struct {
}

// 6小时线的ema金叉
func (TradeLine2 TradeLine2) GetCanLongOrShort(openParams strategy.OpenParams) (openResult strategy.OpenResult) {
	symbols := openParams.Symbols
	openResult.CanLong = false
	openResult.CanShort = false
	
	kline_6h, err1 := binance.GetKlineData(symbols.Symbol, "6h", 50)
	if err1 != nil {
		return openResult
	}
	kline_6h_close := GetLineClosePrices(kline_6h)
	
	ema6h_3, _ := CalculateExponentialMovingAverage(kline_6h_close, 3) // ma3
	ema6h_7, _ := CalculateExponentialMovingAverage(kline_6h_close, 7) // ma7
	ema6h_15, _ := CalculateExponentialMovingAverage(kline_6h_close, 15) // ma15
	rsi6, _ := CalculateRSI(kline_6h_close, 6) // rsi6
	rsi14, _ := CalculateRSI(kline_6h_close, 14) // rsi14
	if (rsi6 == nil || rsi14 == nil) {
		// 开盘小于 4.5 天
		return openResult
	}
	// logs.Info(symbol, Kdj(ema6h_3, ema6h_7, 4), Kdj(ema6h_7, ema6h_3, 4), utils.IsDesc(ema6h_3[0:2]), rsi6[1], rsi14[1])
	if Kdj(ema6h_3, ema6h_7, 4) && Kdj(ema6h_7, ema6h_15, 4) && utils.IsDesc(ema6h_3[0:2]) && rsi6[0] < 80 && rsi14[0] < 75 { // 1天之内发生过金叉, rsi 没有超买
		// 短线穿越长线金叉
		openResult.CanLong = true
		return openResult
	} else if Kdj(ema6h_7, ema6h_3, 4) && Kdj(ema6h_15, ema6h_7, 4) && utils.IsAsc(ema6h_3[0:2]) && rsi6[0] < 80 && rsi14[0] < 75 {
		openResult.CanShort = true
		return openResult
	}
	return openResult
}

// 达到止盈或止损后判断是否可以平仓
func (TradeLine2 TradeLine2) CanOrderComplete(closeParams strategy.CloseParams) (closeResult strategy.CloseResult) {
	symbols := closeParams.Symbols // 交易对
	position := closeParams.Position // 当前仓位
	closeResult.Complete = false
	
	lines, err := binance.GetKlineData(symbols.Symbol, "1m", 2) // 1min 线最近2条
	if err != nil {
		closeResult.Complete = true
		return closeResult
	}
	close0, _ := strconv.ParseFloat(lines[0].Close, 64)
	close1, _ := strconv.ParseFloat(lines[1].Close, 64)
	if position.PositionSide == "LONG" {
		closeResult.Complete = close0 < close1 // 价格在下跌中
	} else if position.PositionSide == "SHORT" {
		closeResult.Complete = close0 > close1 // 价格在上涨中
	} else {
		closeResult.Complete = true
	}
	return closeResult
}

func (TradeLine2 TradeLine2) AutoStopOrder(closeParams strategy.CloseParams) (closeResult strategy.CloseResult) {
	closeResult.Complete = false
	return closeResult
}

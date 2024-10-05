package feature

import (
	"fmt"
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/feature/notify"
	"go_binance_futures/feature/strategy/line"
	"go_binance_futures/utils"
	"math"
	"strconv"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/core/logs"
)

// 测试交易逻辑
func GoTestFeature() {
	coins, _ := GetAllSymbols()
	for _, coin := range coins {
		if coin.Symbol != "FLMUSDT" {
			continue
		}
		// positionSideLong := "LONG"
		// positionSideShort := "SHORT"
		symbol := coin.Symbol
		tickSize := coin.TickSize // 交易金额精度
		stepSize := coin.StepSize // 交易数量精度
		usdt_float64, _ := strconv.ParseFloat(coin.Usdt, 64) // 交易金额
		leverage_float64 := float64(coin.Leverage) // 合约倍数
		buyPrice, _, err := binance.GetDepthAvgPrice(symbol) // 平均买价
		if err == nil {
			buyPrice = utils.GetTradePrecision(buyPrice, tickSize) // 合理精度的价格
			quantity := (usdt_float64 / buyPrice) * leverage_float64  // 购买数量
			quantity = utils.GetTradePrecision(quantity, stepSize) // 合理精度的价格
			logs.Info(fmt.Sprintf("buyPrice:%f, quantity:%f", buyPrice, quantity), buyPrice, quantity)
		}
	}
}

func GoTestLine() {
	// logs.Info(line.BaseCheckCanLongOrShort())
	// return
	logs.Info("start test line")
		
	coins, _ := GetAllSymbols()
	for _, coin := range coins {
		symbol := coin.Symbol
		if symbol != "BTCUSDT" {
			continue
		}
		
		// interval := "1d"
		// limit := 150
		// lines, _ := binance.GetKlineData(symbol, interval, limit)
		// // closePrices := line.GetLineClosePrices(lines)
		// high, low, close := line.GetLineFloatPrices(lines)
		// logs.Info(high[0], low[0], close[0])
		
		// ma3, _ := line.CalculateSimpleMovingAverage(closePrices, 3)
		// ma7, _ := line.CalculateSimpleMovingAverage(closePrices, 7)
		// ma15, _ := line.CalculateSimpleMovingAverage(closePrices, 15)
		// ma30, _ := line.CalculateSimpleMovingAverage(closePrices, 30)
		// ma50, _ := line.CalculateSimpleMovingAverage(close, 50)
		// logs.Info(ma50)
		// ema50, _ := line.CalculateExponentialMovingAverage(close, 50)
		// logs.Info(ema50)
		
		// upper1, ma1, lower1 := line.CalculateKeltnerChannels(high, low, close, 50, 2.75)
		// upper2, _, lower2 := line.CalculateKeltnerChannels(high, low, close, 50, 3.75)
		// // logs.Info(upper[0], ma[0], lower[0])
		// notify.ListenFutureCoinKlineKc(coin.Symbol, "做多信号", close[0], close[0] * (1 - 0.03), ma1[0], upper1[0], upper2[0])
		// notify.ListenFutureCoinKlineKc(coin.Symbol, "做空信号", close[0], close[0] * (1 + 0.03), ma1[0], lower1[0], lower2[0])
		
		
		// ema3, _ := line.CalculateExponentialMovingAverage(closePrices, 3)
		// ema7, _ := line.CalculateExponentialMovingAverage(closePrices, 7)
		// ema15, _ := line.CalculateExponentialMovingAverage(closePrices, 15)
		// ema30, _ := line.CalculateExponentialMovingAverage(closePrices, 30)
		// logs.Info(ema3[0], ema7[0], ema15[0], ema30[0])
		// logs.Info(ma50, high[0], low[0], close[0])
		
		
		
		// mb, up, dn, _ := line.CalculateBollingerBands(closePrices, 21, 2.0)
		// logs.Info(mb[0], up[0], dn[0], len(mb))
		
		// closePrices = utils.ReverseArray(closePrices)
		// rsi6, _ := line.CalculateRSI(closePrices, 6)
		// rsi14, _ := line.CalculateRSI(closePrices, 14)
		// logs.Info(rsi6[1])
		// logs.Info(rsi14[1])
		
		canLang, canShort := lineStrategy.GetCanLongOrShort(symbol)
		logs.Info(symbol, canLang, canShort)
		if canLang || canShort {
			logs.Info(symbol, canLang, canShort)
		}
		// logs.Info("count:", index + 1)
		
		// lineS := line.TradeLine3{}
		// logs.Info(symbol, lineS.MarketReversal(symbol, "LONG"), lineS.MarketReversal(symbol, "SHORT"))
	}
	
	logs.Info("end test line")
}

func GoTestOrder() {
	coins, _ := GetAllSymbols()
	for _, coin := range coins {
		if coin.Symbol != "CKBUSDT" {
			continue
		}
		// positionSideLong := "LONG"
		// positionSideShort := "SHORT"
		symbol := coin.Symbol
		tickSize := coin.TickSize // 交易金额精度
		stepSize := coin.StepSize // 交易数量精度
		usdt_float64, _ := strconv.ParseFloat(coin.Usdt, 64) // 交易金额
		leverage_float64 := float64(coin.Leverage) // 合约倍数
		buyPrice, sellPrice, err := binance.GetDepthAvgPrice(symbol) // 平均买价
		logs.Info(symbol, usdt_float64, leverage_float64, buyPrice, sellPrice)
		if err == nil {
			// 开多
			// buyPrice = utils.GetTradePrecision(buyPrice, tickSize) // 合理精度的价格
			// quantity := (usdt_float64 / buyPrice) * leverage_float64  // 购买数量
			// quantity = utils.GetTradePrecision(quantity, stepSize) // 合理精度的价格
			// result, _ := binance.BuyLimit(symbol, quantity, buyPrice, futures.PositionSideTypeLong)
			// logs.Info(result)
			
			// 开空
			sellPrice = utils.GetTradePrecision(sellPrice, tickSize) // 合理精度的价格
			quantity := (usdt_float64 / sellPrice) * leverage_float64  // 购买数量
			quantity = utils.GetTradePrecision(quantity, stepSize) // 合理精度的价格
			result, _ := binance.SellLimit(symbol, quantity, sellPrice, futures.PositionSideTypeShort)
			logs.Info(result)
		}
	}
}

func GoTestUtil() {
	ma1 := []float64{10, 9, 9.2, 6}
	ma2 := []float64{9, 7, 9, 6.1}
	logs.Info(line.Kdj(ma1, ma2, 4))
}

// test 平仓
func GoTestMarketOrder() {
	positions, _ := binance.GetPosition()
	for _, position := range positions {
    	positionAmtFloat, _ := strconv.ParseFloat(position.PositionAmt, 64)
    	positionAmtFloatAbs := math.Abs(positionAmtFloat) // 空单为负数,纠正为绝对值
    	unRealizedProfit, _ := strconv.ParseFloat(position.UnRealizedProfit, 64)
    	
    	if positionAmtFloatAbs < 0.0000000001 {// 没有持仓的
    		continue
    	}
    	if position.Symbol == "BELUSDT" {
			if position.PositionSide == "SHORT" {
				result, err := binance.BuyMarket(position.Symbol, positionAmtFloatAbs, futures.PositionSideTypeShort)
				if err == nil {
					// 数据库写入订单
					insertCloseOrder(position, positionAmtFloatAbs, unRealizedProfit, result.AvgPrice)
					notify.SellOrderSuccess(position.Symbol, "风向改变,自动平仓", unRealizedProfit)
				} else {
					notify.SellOrderFail(position.Symbol, err.Error())
				}
			}
    	}
	}
}
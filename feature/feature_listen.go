package feature

import (
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/feature/notify"
	"go_binance_futures/feature/strategy/line"
	"go_binance_futures/models"
	"strconv"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

func ListenCoin() {
	o := orm.NewOrm()
	var coins []models.ListenSymbols
	o.QueryTable("listen_symbols").OrderBy("ID").Filter("enable", 1).Filter("type", 2).All(&coins) // 通知币列表
	
	nowTime := time.Now().Unix()
	for _, coin := range coins {
		logs.Info("监听合约币种:", coin.Symbol)
		if nowTime - coin.LastNoticeTime < 60 * 1000 * coin.NoticeLimitMin {
			// 通知频率限制
			continue
		}
		if coin.ListenType == "kline_base" {
			klineBaseListen(coin)
		} else if coin.ListenType == "kline_kc" {
			klineKcListen(coin)
		}
	}
}

func klineBaseListen(coin models.ListenSymbols) {
	logs.Info("listen kline_base")
	kline_1, err1 := binance.GetKlineData(coin.Symbol, coin.KlineInterval, 10)
	if err1 != nil {
		logs.Error("k线错误, 合约币种是:", coin.Symbol)
		return
	}
	
	percentLimit, _ := strconv.ParseFloat(coin.ChangePercent, 64)
	percentLimit = percentLimit / 100 // 变化幅度

	lastOpenPrice, _ := strconv.ParseFloat(kline_1[1].Open, 64) // 上一根 k 线的开盘价
	nowPrice, _ := strconv.ParseFloat(kline_1[0].Close, 64) // 当前价格
	
	if (nowPrice > lastOpenPrice) && 
		(nowPrice - lastOpenPrice) / lastOpenPrice >= percentLimit &&
		((kline_1[0].CloseTime - coin.LastNoticeTime >= 60 * 1000 * coin.NoticeLimitMin && coin.LastNoticeType == "up") || coin.LastNoticeType != "up") {
		// 价格上涨
		coin.LastNoticeTime = kline_1[0].CloseTime
		coin.LastNoticeType = "up"
		orm.NewOrm().Update(&coin)
		
		notify.ListenFutureCoin(coin.Symbol, "极速上涨", (nowPrice - lastOpenPrice) / lastOpenPrice, nowPrice)
	}
	if (nowPrice < lastOpenPrice) && 
		(lastOpenPrice - nowPrice) / lastOpenPrice >= percentLimit &&
		((kline_1[0].CloseTime - coin.LastNoticeTime >= 60 * 1000 * coin.NoticeLimitMin && coin.LastNoticeType == "down") || coin.LastNoticeType != "down") {
		// 价格下跌
		coin.LastNoticeTime = kline_1[0].CloseTime
		coin.LastNoticeType = "down"
		orm.NewOrm().Update(&coin)
		
		notify.ListenFutureCoin(coin.Symbol, "极速下跌", (lastOpenPrice - nowPrice) / lastOpenPrice, nowPrice)
	}
}

func klineKcListen(coin models.ListenSymbols) {
	logs.Info("listen kline_kc")
	
	internals := []string{"15m", "1h", "4h", "1d", "3d", "1w", "1M"}
	symbol := coin.Symbol
	limit := 150
	period := 50 
	multiplier1 := 2.75 // 窄通道
	multiplier2 := 3.75 // 宽通道
	interval1 := coin.KlineInterval
	interval2 := ""
	
	for index, item := range internals {
		if item == interval1 && index < len(internals) - 1 {
			interval2 = internals[index + 1]
			break
		}
	}
	if interval2 == "" {
		return
	}
	
	kline_1, err := binance.GetKlineData(symbol, interval1, limit)
	if err != nil {
		logs.Error("k线错误, 合约币种是:", coin.Symbol)
		return
	}
	if kline_1[0].CloseTime - coin.LastNoticeTime < 60 * 1000 * coin.NoticeLimitMin {
		// 通知频率限制
		return
	}
	
	kline_2, err := binance.GetKlineData(symbol, interval2, limit) // 通知时暂不启用大级别的影响
	if err != nil {
		logs.Error("k线错误, 合约币种是:", coin.Symbol)
		return
	}
	
	if len(kline_1) < limit || len(kline_2) < limit {
		logs.Error("k线数量不足, 合约币种是:", coin.Symbol)
		return
	}
	
	high1, low1, close1 := line.GetLineFloatPrices(kline_1)
	upper1, ma1, lower1 := line.CalculateKeltnerChannels(high1, low1, close1, period, multiplier1) // kc1
	upper2, _, lower2 := line.CalculateKeltnerChannels(high1, low1, close1, period, multiplier2) // kc2
	
	limitPeriod := 12 // 最近n根k线
	lossPercent := 0.03

	// 之前的最低价格跌破了 kc2 的下轨，然后当前价格超越了 kc1 下轨，止损位置在 kc1 下轨附近位置，止盈50%位置在 kc1 中规附近位置，剩余 50% 止盈50%位置在 kc1 上轨附近位置
	if (close1[0] > lower1[0] && close1[1] < lower1[1]) {
		for i := 2; i < limitPeriod; i++ {
			// 最近10根k线最低价格在kc2下轨之下
			if low1[i] < lower2[i] {
				coin.LastNoticeTime = kline_1[0].CloseTime
				coin.LastNoticeType = "up"
				orm.NewOrm().Update(&coin)
				
				notify.ListenFutureCoinKlineKc(coin.Symbol, "做多信号", close1[0], close1[0] * (1 - lossPercent), ma1[0], upper1[0], upper2[0])
				break
			}
		}
	}
	// 之前的最高价格超越了 kc2 的上轨，然后当前价格跌破了 kc1 上轨，止损位置在 kc1 上轨附近位置，止盈50%位置在 kc1 中规附近位置，剩余 50% 止盈50%位置在 kc1 下轨附近位置
	if (close1[0] < upper1[0] && close1[1] > upper1[1]) {
		for i := 1; i < limitPeriod; i++ {
			// 最近10根k线最高价格在kc2上轨之上
			if high1[i] > upper2[i] {
				coin.LastNoticeTime = kline_1[0].CloseTime
				coin.LastNoticeType = "down"
				orm.NewOrm().Update(&coin)
				
				notify.ListenFutureCoinKlineKc(coin.Symbol, "做空信号", close1[0], close1[0] * (1 + lossPercent), ma1[0], lower1[0], lower2[0])
				break
			}
		}
	}
}

func ListenCoinFundingRate() {
	o := orm.NewOrm()
	var coins []models.SymbolFundingRates
	o.QueryTable("symbol_funding_rates").OrderBy("ID").Filter("enable", 1).All(&coins) // 通知币列表
	for _, coin := range coins {
		nowFundingRate, _ := strconv.ParseFloat(coin.NowFundingRate, 64)
		lastNoticeFundingRate, _ := strconv.ParseFloat(coin.NowFundingRate, 64)
		diff := nowFundingRate - lastNoticeFundingRate
		if (diff > 0.01 && nowFundingRate > 0.008) {
			// 正资金费率，做空可以吃资金费用
			coin.LastNoticeFundingRate = coin.NowFundingRate
			coin.LastNoticeFundingTime = coin.NowFundingTime
			coin.LastNoticePrice = coin.NowPrice
			orm.NewOrm().Update(&coin)
			
			notify.ListenFutureCoinFundingRate(coin.Symbol, "做空吃资金费费率", nowFundingRate * 100, coin.NowPrice)
		} else if (diff < -0.01 && nowFundingRate < -0.008) {
			// 负资金费率，做多可以吃资金费用
			coin.LastNoticeFundingRate = coin.NowFundingRate
			coin.LastNoticeFundingTime = coin.NowFundingTime
			coin.LastNoticePrice = coin.NowPrice
			orm.NewOrm().Update(&coin)
			
			notify.ListenFutureCoinFundingRate(coin.Symbol, "做多吃资金费费率", nowFundingRate * 100, coin.NowPrice)
		}
	}
}

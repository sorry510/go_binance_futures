package feature

import (
	"encoding/json"
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/feature/strategy/line"
	"go_binance_futures/lang"
	"go_binance_futures/models"
	"go_binance_futures/notify"
	"go_binance_futures/technology"
	"strconv"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
	"github.com/expr-lang/expr"
)

var flagFuturesListen = 0
func ListenCoin(systemConfig models.Config) {
	if (systemConfig.ListenCoinEnable == 1) {
		if (flagFuturesListen == 0) {
			logs.Info("futures listen bot start")
			flagFuturesListen = 1
		}
	} else {
		if (flagFuturesListen == 1) {
			logs.Info("futures listen bot stop")
			flagFuturesListen = 0
		}
		return
	}
	
	o := orm.NewOrm()
	var coins []models.ListenSymbols
	o.QueryTable("listen_symbols").OrderBy("ID").Filter("enable", 1).Filter("type", 2).All(&coins) // 通知币列表
	
	nowTime := time.Now().Unix()
	for _, coin := range coins {
		if nowTime * 1000 - coin.LastNoticeTime < 60 * 1000 * coin.NoticeLimitMin {
			// 通知频率限制
			logs.Info("listen futures: %s, type: %s, notice limit time < %d Min", coin.Symbol, coin.ListenType, coin.NoticeLimitMin)
			continue
		}
		logs.Info("listen futures: %s, type: %s", coin.Symbol, coin.ListenType)
		switch coin.ListenType {
			case "kline_base":
				klineBaseListen(coin)
			case "kline_kc":
				klineKcListen(coin)
			case "custom":
				klineCustomListen(coin)
			default:
				logs.Error("listen type error:", coin.ListenType)
		}
	}
}

// 自定义规则的监听
func klineCustomListen(coin models.ListenSymbols) {
	var strategyConfig technology.StrategyConfig
	err := json.Unmarshal([]byte(coin.Strategy), &strategyConfig)
	if err != nil {
		logs.Error("Error unmarshalling JSON:", err.Error())
		return
	}
	env := line.InitParseEnv(coin.Symbol, coin.Technology)
	for _, strategy := range strategyConfig {
		if strategy.Enable {
			program, err := expr.Compile(strategy.Code, expr.Env(env))
			if err != nil {
				logs.Error("Error Strategy Compile:", err.Error())
				continue
			}
			output, err := expr.Run(program, env)
			if err != nil {
				logs.Error("Error Strategy Run:", err.Error())
				continue
			}
			if result, ok := output.(bool); ok && result {
				if strategy.Type == "long" {
					coin.LastNoticeTime = time.Now().Unix() * 1000
					coin.LastNoticeType = "up"
					orm.NewOrm().Update(&coin)
					
					pusher.FuturesListenKlineCustom(notify.FuturesListenParams{
						Title: lang.Lang("futures.listen_custom_title"),
						NowPrice: env["NowPrice"].(float64),
						Symbol: coin.Symbol,
						PositionSide: strategy.Type,
						StrategyName: strategy.Name,
						Remarks: strategy.Code,
					}) 
				} else if strategy.Type == "short" {
					coin.LastNoticeTime = time.Now().Unix() * 1000
					coin.LastNoticeType = "down"
					orm.NewOrm().Update(&coin)
					
					pusher.FuturesListenKlineCustom(notify.FuturesListenParams{
						Title: lang.Lang("futures.listen_custom_title"),
						NowPrice: env["NowPrice"].(float64),
						Symbol: coin.Symbol,
						PositionSide: strategy.Type,
						StrategyName: strategy.Name,
						Remarks: strategy.Code,
					}) 
				}
			}
		}
	}
}

func klineBaseListen(coin models.ListenSymbols) {
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
		
		pusher.FuturesListenKlineBase(notify.FuturesListenParams{
			Title: lang.Lang("futures.listen_kline_base_title"),
			Symbol: coin.Symbol,
			ChangePercent: (nowPrice - lastOpenPrice) / lastOpenPrice,
			Price: nowPrice,
			Remarks: lang.Lang("futures.fast_up"),
		})
	}
	if (nowPrice < lastOpenPrice) && 
		(lastOpenPrice - nowPrice) / lastOpenPrice >= percentLimit &&
		((kline_1[0].CloseTime - coin.LastNoticeTime >= 60 * 1000 * coin.NoticeLimitMin && coin.LastNoticeType == "down") || coin.LastNoticeType != "down") {
		// 价格下跌
		coin.LastNoticeTime = kline_1[0].CloseTime
		coin.LastNoticeType = "down"
		orm.NewOrm().Update(&coin)
		
		pusher.FuturesListenKlineBase(notify.FuturesListenParams{
			Title: lang.Lang("futures.listen_kline_base_title"),
			Symbol: coin.Symbol,
			ChangePercent: (lastOpenPrice - nowPrice) / lastOpenPrice,
			Price: nowPrice,
			Remarks: lang.Lang("futures.fast_down"),
		})
	}
}

func klineKcListen(coin models.ListenSymbols) {
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
	
	high1, low1, close1, _ := line.GetLineFloatPrices(kline_1)
	upper1, ma1, lower1 := line.CalculateKeltnerChannels(high1, low1, close1, period, multiplier1) // kc1
	upper2, _, lower2 := line.CalculateKeltnerChannels(high1, low1, close1, period, multiplier2) // kc2
	
	limitPeriod := 12 // 最近n根k线
	lossPercent := 0.015

	// 之前的最低价格跌破了 kc2 的下轨，然后当前价格超越了 kc1 下轨，止损位置在 kc1 下轨附近位置，止盈50%位置在 kc1 中规附近位置，剩余 50% 止盈50%位置在 kc1 上轨附近位置
	if (close1[0] > lower1[0] && close1[1] < lower1[1]) {
		for i := 2; i < limitPeriod; i++ {
			// 最近10根k线最低价格在kc2下轨之下
			if low1[i] < lower2[i] {
				coin.LastNoticeTime = kline_1[0].CloseTime
				coin.LastNoticeType = "up"
				orm.NewOrm().Update(&coin)
				
				pusher.FuturesListenKlineKc(notify.FuturesListenParams{
					Title: lang.Lang("futures.listen_keltner_channels_title"),
					PositionSide: "long",
					Symbol: coin.Symbol,
					NowPrice: close1[0],
					StopLossPrice: close1[0] * (1 - lossPercent),
					TargetHalfProfitPrice: ma1[0],
					TargetAllProfitPrice: upper1[0],
					DesiredPrice: upper2[0],
				}) 
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
				
				pusher.FuturesListenKlineKc(notify.FuturesListenParams{
					Title: lang.Lang("futures.listen_keltner_channels_title"),
					PositionSide: "short",
					Symbol: coin.Symbol,
					NowPrice: close1[0],
					StopLossPrice: close1[0] * (1 - lossPercent),
					TargetHalfProfitPrice: ma1[0],
					TargetAllProfitPrice: lower1[0],
					DesiredPrice: lower2[0],
				})
				break
			}
		}
	}
}

// 监控资金费率
var flagListenFundingRate = 0
func ListenCoinFundingRate(systemConfig models.Config) {
	if (systemConfig.ListenFundingRateEnable == 1) {
		if (flagListenFundingRate == 0) {
			logs.Info("listen funding rate bot start")
			flagListenFundingRate = 1
		}
	} else {
		if (flagListenFundingRate == 1) {
			logs.Info("listen funding rate bot stop")
			flagListenFundingRate = 0
		}
		return
	}
	
	o := orm.NewOrm()
	var coins []models.SymbolFundingRates
	o.QueryTable("symbol_funding_rates").OrderBy("ID").Filter("enable", 1).All(&coins) // 通知币列表
	for _, coin := range coins {
		// if coin.Symbol != "CHZUSDT" {
		// 	continue
		// }
		nowFundingRate, _ := strconv.ParseFloat(coin.NowFundingRate, 64)
		lastNoticeFundingRate, _ := strconv.ParseFloat(coin.LastNoticeFundingRate, 64)
		diff := nowFundingRate - lastNoticeFundingRate
		if (diff > 0.0055 && nowFundingRate > 0.005) {
			// 正资金费率，做空可以吃资金费用
			coin.LastNoticeFundingRate = coin.NowFundingRate
			coin.LastNoticeFundingTime = coin.NowFundingTime
			coin.LastNoticePrice = coin.NowPrice
			orm.NewOrm().Update(&coin)
			
			price, _ := strconv.ParseFloat(coin.NowPrice, 64)
			pusher.FuturesListenFundingRate(notify.FuturesListenParams{
				Title: lang.Lang("futures.listen_funding_rate_title"),
				Symbol: coin.Symbol,
				PositionSide: "short",
				FundingRate: nowFundingRate * 100,
				Price: price,
				Remarks: lang.Lang("futures.profit_by_funding_rate"),
			})
			
		} else if (diff < -0.0055 && nowFundingRate < -0.005) {
			// 负资金费率，小于 -1%, 做多可以吃资金费用
			coin.LastNoticeFundingRate = coin.NowFundingRate
			coin.LastNoticeFundingTime = coin.NowFundingTime
			coin.LastNoticePrice = coin.NowPrice
			orm.NewOrm().Update(&coin)
			
			price, _ := strconv.ParseFloat(coin.NowPrice, 64)
			pusher.FuturesListenFundingRate(notify.FuturesListenParams{
				Title: lang.Lang("futures.listen_funding_rate_title"),
				Symbol: coin.Symbol,
				PositionSide: "long",
				FundingRate: nowFundingRate * 100,
				Price: price,
				Remarks: lang.Lang("futures.profit_by_funding_rate"),
			})
		}
	}
}

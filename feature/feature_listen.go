package feature

import (
	"encoding/json"
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/feature/strategy/line"
	"go_binance_futures/lang"
	"go_binance_futures/models"
	"go_binance_futures/notify"
	"go_binance_futures/technology"
	"go_binance_futures/utils"
	"strconv"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
	"github.com/expr-lang/expr"
)

var flagFuturesListen = 0

func ListenCoin(systemConfig models.Config) {
	if systemConfig.ListenCoinEnable == 1 {
		if flagFuturesListen == 0 {
			logs.Info("futures listen bot start")
			flagFuturesListen = 1
		}
	} else {
		if flagFuturesListen == 1 {
			logs.Info("futures listen bot stop")
			flagFuturesListen = 0
		}
		return
	}

	o := orm.NewOrm()
	var coins []models.ListenSymbols
	o.QueryTable("listen_symbols").OrderBy("ID").Filter("enable", 1).Filter("type", 2).Filter("listen_type", "custom").All(&coins) // 仅执行自定义监听，旧类型保留在 DB 但不触发通知

	nowTime := time.Now().Unix()
	for _, coin := range coins {
		if nowTime*1000-coin.LastNoticeTime < 60*1000*coin.NoticeLimitMin {
			// 通知频率限制
			logs.Info("listen futures: %s, type: %s, notice limit time < %d Min", coin.Symbol, coin.ListenType, coin.NoticeLimitMin)
			continue
		}
		logs.Info("listen futures custom: %s", coin.Symbol)
		klineCustomListen(coin)
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

					pusher.SetModuleName("coin_listen").FuturesListenKlineCustom(notify.FuturesListenParams{
						Title:        lang.Lang("futures.listen_custom_title"),
						NowPrice:     env["NowPrice"].(float64),
						Symbol:       coin.Symbol,
						PositionSide: strategy.Type,
						StrategyName: strategy.Name,
						Remarks:      strategy.Code,
					})
				} else if strategy.Type == "short" {
					coin.LastNoticeTime = time.Now().Unix() * 1000
					coin.LastNoticeType = "down"
					orm.NewOrm().Update(&coin)

					pusher.SetModuleName("coin_listen").FuturesListenKlineCustom(notify.FuturesListenParams{
						Title:        lang.Lang("futures.listen_custom_title"),
						NowPrice:     env["NowPrice"].(float64),
						Symbol:       coin.Symbol,
						PositionSide: strategy.Type,
						StrategyName: strategy.Name,
						Remarks:      strategy.Code,
					})
				}
			}
		}
	}
}

// 监控资金费率
var flagListenFundingRate = 0

func ListenCoinFundingRate(systemConfig models.Config) {
	if systemConfig.ListenFundingRateEnable == 1 {
		if flagListenFundingRate == 0 {
			logs.Info("listen funding rate bot start")
			flagListenFundingRate = 1
		}
	} else {
		if flagListenFundingRate == 1 {
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
		if diff > 0.0055 && nowFundingRate > 0.005 {
			// 正资金费率，做空可以吃资金费用
			coin.LastNoticeFundingRate = coin.NowFundingRate
			coin.LastNoticeFundingTime = coin.NowFundingTime
			coin.LastNoticePrice = coin.NowPrice
			orm.NewOrm().Update(&coin)

			price, _ := strconv.ParseFloat(coin.NowPrice, 64)
			pusher.SetModuleName("funding_rate").FuturesListenFundingRate(notify.FuturesListenParams{
				Title:        lang.Lang("futures.listen_funding_rate_title"),
				Symbol:       coin.Symbol,
				PositionSide: "short",
				FundingRate:  nowFundingRate * 100,
				Price:        price,
				Remarks:      lang.Lang("futures.profit_by_funding_rate"),
			})
			if coin.AutoOrder == 1 {
				// 自动下单(空单)
				OpenFundingRateShortOrder(coin.Symbol)
			}
		} else if diff < -0.0055 && nowFundingRate < -0.005 {
			// 负资金费率，小于 -1%, 做多可以吃资金费用
			coin.LastNoticeFundingRate = coin.NowFundingRate
			coin.LastNoticeFundingTime = coin.NowFundingTime
			coin.LastNoticePrice = coin.NowPrice
			orm.NewOrm().Update(&coin)

			price, _ := strconv.ParseFloat(coin.NowPrice, 64)
			pusher.SetModuleName("funding_rate").FuturesListenFundingRate(notify.FuturesListenParams{
				Title:        lang.Lang("futures.listen_funding_rate_title"),
				Symbol:       coin.Symbol,
				PositionSide: "long",
				FundingRate:  nowFundingRate * 100,
				Price:        price,
				Remarks:      lang.Lang("futures.profit_by_funding_rate"),
			})
			if coin.AutoOrder == 1 {
				OpenFundingRateLongOrder(coin.Symbol)
				// 自动下单(多单)
			}
		}
	}
}

func OpenFundingRateLongOrder(symbol string) {
	o := orm.NewOrm()
	var coinSymbol models.Symbols
	err := o.QueryTable("symbols").Filter("symbol", symbol).One(&coinSymbol)
	if err != nil {
		logs.Error("ListenCoinFundingRate auto order err:", symbol, err.Error())
		return
	}
	buyPrice, _, err := binance.GetDepthAvgPrice(coinSymbol.Symbol, 5) // 平均买价
	if err == nil {
		usdt_float64, _ := strconv.ParseFloat(coinSymbol.Usdt, 64)        // 交易金额
		leverage_float64 := float64(coinSymbol.Leverage)                  // 合约倍数
		buyPrice = utils.GetTradePrecision(buyPrice, coinSymbol.TickSize) // 合理精度的价格
		quantity := (usdt_float64 / buyPrice) * leverage_float64          // 购买数量
		quantity = utils.GetTradePrecision(quantity, coinSymbol.StepSize) // 合理精度的价格
		_, err := submitFundingRateOpen("funding_rate:"+coinSymbol.Symbol, coinSymbol.Symbol, quantity, futures.SideTypeBuy, futures.PositionSideTypeLong)
		if err != nil {
			logs.Error("ListenCoinFundingRate auto order buy long err:", symbol, err.Error())
		} else {
			logs.Info("ListenCoinFundingRate auto order buy long success:", symbol, "price:", buyPrice, "quantity:", quantity)
		}
	}
}

func OpenFundingRateShortOrder(symbol string) {
	o := orm.NewOrm()
	var coinSymbol models.Symbols
	err := o.QueryTable("symbols").Filter("symbol", symbol).One(&coinSymbol)
	if err != nil {
		logs.Error("ListenCoinFundingRate auto order err:", symbol, err.Error())
		return
	}
	_, sellPrice, err := binance.GetDepthAvgPrice(coinSymbol.Symbol, 5) // 平均卖价
	if err == nil {
		usdt_float64, _ := strconv.ParseFloat(coinSymbol.Usdt, 64)          // 交易金额
		leverage_float64 := float64(coinSymbol.Leverage)                    // 合约倍数
		sellPrice = utils.GetTradePrecision(sellPrice, coinSymbol.TickSize) // 合理精度的价格
		quantity := (usdt_float64 / sellPrice) * leverage_float64           // 购买数量
		quantity = utils.GetTradePrecision(quantity, coinSymbol.StepSize)   // 合理精度的价格
		_, err := submitFundingRateOpen("funding_rate:"+coinSymbol.Symbol, coinSymbol.Symbol, quantity, futures.SideTypeSell, futures.PositionSideTypeShort)
		if err != nil {
			logs.Error("ListenCoinFundingRate auto order buy short err:", symbol, err.Error())
		} else {
			logs.Info("ListenCoinFundingRate auto order buy short success:", symbol, "price:", sellPrice, "quantity:", quantity)
		}
	}
}

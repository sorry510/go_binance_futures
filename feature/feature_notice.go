package feature

import (
	"encoding/json"
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/feature/strategy/coin"
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

var flagFuturesNotice = 0
func NoticeAndAutoOrder() {
	systemConfig, err := utils.GetSystemConfig()
	if err != nil {
		logs.Error("GetSystemConfig:", err)
		return
	}
	if (systemConfig.NoticeCoinEnable == 1) {
		if (flagFuturesNotice == 0) {
			logs.Info("futures notice bot start")
			flagFuturesNotice = 1
		}
	} else {
		if (flagFuturesNotice == 1) {
			logs.Info("futures notice bot stop")
			flagFuturesNotice = 0
		}
		return
	}
	
	o := orm.NewOrm()
	var coins []models.NoticeSymbols
	o.QueryTable("notice_symbols").OrderBy("ID").Filter("enable", 1).Filter("type", 2).Filter("has_notice", 0).All(&coins) // 通知币列表
	
	for _, coin := range coins {
		logs.Info("notice_futures: ", coin.Symbol)
		resPrice, errPrice := binance.GetTickerPrice(coin.Symbol)
		if errPrice != nil {
			logs.Info("无法进行通知, 还未上线此币种: ", coin.Symbol)
			continue
		}
		var canOrder = false
		nowPrice, _ := strconv.ParseFloat(resPrice[0].Price, 64) // 预计交易价格
		noticePrice, _ := strconv.ParseFloat(coin.NoticePrice, 64) // 预警价格
		var autoOrderText = "no"
		if coin.AutoOrder == 1 {
			autoOrderText = "yes"
		}
		if (nowPrice <= noticePrice && coin.Side == "buy") {
			// 做多，价格低于预警价格，进行通知
			canOrder = true
			pusher.FuturesNotice(notify.FuturesNoticeParams{
				Title: lang.Lang("futures.notice_price_title"),
				Symbol: coin.Symbol,
				Side: coin.Side,
				PositionSide: "long",
				Price: noticePrice,
				AutoOrder: lang.Lang("futures." + autoOrderText),
			})
		}
		if (nowPrice >= noticePrice && coin.Side == "sell") {
			// 做空，价格高于预警价格，进行通知
			canOrder = true
			pusher.FuturesNotice(notify.FuturesNoticeParams{
				Title: lang.Lang("futures.notice_price_title"),
				Symbol: coin.Symbol,
				Side: coin.Side,
				PositionSide: "short",
				Price: noticePrice,
				AutoOrder: lang.Lang("futures." + autoOrderText),
			})
		}
		if canOrder {
			coin.HasNotice = 1
			coin.Enable = 0 // 更新为关闭
			orm.NewOrm().Update(&coin)
		}
		if coin.AutoOrder == 1 && canOrder {
			// 修改仓位模式
			if coin.MarginType == "ISOLATED" {
				binance.SetMarginType(coin.Symbol, futures.MarginTypeIsolated)
			} else {
				binance.SetMarginType(coin.Symbol, futures.MarginTypeCrossed)
			}
			binance.SetLeverage(coin.Symbol, int(coin.Leverage))  // 修改合约倍数
			
			usdt_float64, _ := strconv.ParseFloat(coin.Usdt, 64) // 交易金额
			leverage_float64 := float64(coin.Leverage) // 合约倍数
			quantity := (usdt_float64 / nowPrice) * leverage_float64  // 购买数量
			quantity = utils.GetTradePrecision(quantity, coin.StepSize) // 合理精度的数量
			
			if coin.Side == "buy" {
				_, err := binance.BuyMarket(coin.Symbol, quantity, futures.PositionSideTypeLong)
				if err != nil {
					logs.Info("合约做多失败symbol:", coin.Symbol)
					pusher.FuturesOpenOrder(notify.FuturesOrderParams{
						Title: lang.Lang("futures.open_notice_title"),
						Symbol: coin.Symbol,
						Side: "buy",
						PositionSide: "long",
						Price: nowPrice,
						Quantity: quantity,
						Leverage: leverage_float64,
						Status: "fail",
						Error: err.Error(),
					})
				} else {
					pusher.FuturesOpenOrder(notify.FuturesOrderParams{
						Title: lang.Lang("futures.open_notice_title"),
						Symbol: coin.Symbol,
						Side: "buy",
						PositionSide: "long",
						Price: nowPrice,
						Quantity: quantity,
						Leverage: leverage_float64,
						Status: "success",
					})
					if (coin.ProfitPrice != "0") {
						// 挂一个止盈单
						profit_price_float64, _ := strconv.ParseFloat(coin.ProfitPrice, 64) // 交易金额
						profit_price_float64 = utils.GetTradePrecision(profit_price_float64, coin.TickSize) // 合理精度价格
						binance.OrderTakeProfit(coin.Symbol, profit_price_float64, futures.SideTypeSell, futures.PositionSideTypeLong)
					}
					if (coin.LossPrice != "0") {
						// 挂一个止损单
						loss_price_float64, _ := strconv.ParseFloat(coin.LossPrice, 64) // 交易金额
						loss_price_float64 = utils.GetTradePrecision(loss_price_float64, coin.TickSize) // 合理精度价格
						binance.OrderStopLoss(coin.Symbol, loss_price_float64, futures.SideTypeSell, futures.PositionSideTypeLong)
					}
				}
			} else if coin.Side == "sell" {
				_, err := binance.SellMarket(coin.Symbol, quantity, futures.PositionSideTypeShort)
				if err != nil {
					logs.Info("合约做空失败symbol:", coin.Symbol)
					pusher.FuturesOpenOrder(notify.FuturesOrderParams{
						Title: lang.Lang("futures.open_notice_title"),
						Symbol: coin.Symbol,
						Side: "sell",
						PositionSide: "short",
						Price: nowPrice,
						Quantity: quantity,
						Leverage: leverage_float64,
						Status: "fail",
						Error: err.Error(),
					})
				} else {
					pusher.FuturesOpenOrder(notify.FuturesOrderParams{
						Title: lang.Lang("futures.open_notice_title"),
						Symbol: coin.Symbol,
						Side: "sell",
						PositionSide: "short",
						Price: nowPrice,
						Quantity: quantity,
						Leverage: leverage_float64,
						Status: "success",
					})
					if (coin.ProfitPrice != "0") {
						// 挂一个止盈单
						profit_price_float64, _ := strconv.ParseFloat(coin.ProfitPrice, 64) // 交易金额
						profit_price_float64 = utils.GetTradePrecision(profit_price_float64, coin.TickSize) // 合理精度价格
						binance.OrderTakeProfit(coin.Symbol, profit_price_float64, futures.SideTypeBuy, futures.PositionSideTypeShort)
					}
					if (coin.LossPrice != "0") {
						// 挂一个止损单
						loss_price_float64, _ := strconv.ParseFloat(coin.LossPrice, 64) // 交易金额
						loss_price_float64 = utils.GetTradePrecision(loss_price_float64, coin.TickSize) // 合理精度价格
						binance.OrderStopLoss(coin.Symbol, loss_price_float64, futures.SideTypeBuy, futures.PositionSideTypeShort)
					}
				}
			}
		}
	}
}

var coinNoticeLastTimeMap = make(map[string]int64) // 3min 通知一次
var FuturesTestNotice = 0
func NoticeAllSymbolByStrategy() {
	systemConfig, err := utils.GetSystemConfig()
	if err != nil {
		logs.Error("GetSystemConfig:", err)
		return
	}
	if (systemConfig.FutureTest == 1) {
		if (FuturesTestNotice == 0) {
			logs.Info("futures all symbol notice_strategy bot start")
			FuturesTestNotice = 1
		}
	} else {
		if (flagFuturesNotice == 1) {
			logs.Info("futures all symbol notice_strategy bot end")
			FuturesTestNotice = 0
		}
		return
	}
	
	coins, _ := GetAllSymbols()
	coins = coin.GetRandArr(coins, 10) // 随机10个
	
	for _, coin := range coins {
		nowTime := time.Now().Unix() * 1000 // 毫秒时间戳
		if coin.Enable != 1 {
			continue
		}
		
		lastNoticeTime, exist := coinNoticeLastTimeMap[coin.Symbol]
		if exist {
			if (nowTime - lastNoticeTime) < 3 * 60 * 1000 {
				// 5min 通知一次
				continue
			}
		}
		if coin.Technology == "" || coin.Strategy == "" {
			logs.Info("no set custom strategy, symbol: ", coin.Symbol)
			continue
		}
		
		var strategyConfig technology.StrategyConfig
		err := json.Unmarshal([]byte(coin.Strategy), &strategyConfig)
		if err != nil {
			logs.Error("Error unmarshalling JSON Symbol: ", coin.Symbol)
			logs.Error("Error unmarshalling JSON:", err.Error())
			continue
		}
		logs.Info("futures custom strategy test, symbol: ", coin.Symbol)
		env := line.InitParseEnv(coin.Symbol, coin.Technology)
		for _, strategy := range strategyConfig {
			if strategy.Enable {
				program, err := expr.Compile(strategy.Code, expr.Env(env))
				if err != nil {
					logs.Error("Error Strategy Compile Symbol: ", coin.Symbol)
					logs.Error("Error Strategy Compile:", err.Error())
					continue
				}
				output, err := expr.Run(program, env)
				if err != nil {
					logs.Error("Error Strategy Run Symbol: ", coin.Symbol)
					logs.Error("Error Strategy Run:", err.Error())
					continue
				}
				if output.(bool) {
					pusher.FuturesCustomStrategyTest(notify.FuturesTestParams{
						Title: lang.Lang("futures.custom_strategy_test"),
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

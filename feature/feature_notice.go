package feature

import (
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/feature/notify"
	"go_binance_futures/models"
	"go_binance_futures/utils"
	"strconv"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

func NoticeAndAutoOrder() {
	o := orm.NewOrm()
	var coins []models.NoticeSymbols
	o.QueryTable("notice_symbols").OrderBy("ID").Filter("enable", 1).Filter("type", 2).All(&coins) // 通知币列表
	
	for _, coin := range coins {
		resPrice, errPrice := binance.GetTickerPrice(coin.Symbol)
		if errPrice != nil {
			logs.Info("无法进行通知, 还未上线此币种: ", coin.Symbol)
			continue
		}
		var canOrder = false
		nowPrice, _ := strconv.ParseFloat(resPrice[0].Price, 64) // 预计交易价格
		noticePrice, _ := strconv.ParseFloat(coin.NoticePrice, 64) // 预警价格
		if (coin.HasNotice == 0) {
			var autoOrderText = "否"
			if coin.AutoOrder == 1 {
				autoOrderText = "是"
			}
			if (nowPrice <= noticePrice && coin.Side == "buy") {
				// 做多，价格低于预警价格，进行通知
				canOrder = true
				notify.NoticeFutureCoin(coin.Symbol, "做多", coin.NoticePrice, autoOrderText)
			}
			if (nowPrice >= noticePrice && coin.Side == "sell") {
				// 做空，价格高于预警价格，进行通知
				canOrder = true
				notify.NoticeFutureCoin(coin.Symbol, "做空", coin.NoticePrice, autoOrderText)
			}
			if canOrder {
				coin.HasNotice = 1
				coin.Enable = 0 // 更新为关闭
				orm.NewOrm().Update(&coin)
			}
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
					notify.BuyOrderFail(coin.Symbol, quantity, nowPrice, "做多", err.Error())
				} else {
					notify.BuyOrderSuccess(coin.Symbol, quantity, nowPrice, "做多")
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
					notify.BuyOrderFail(coin.Symbol, quantity, nowPrice, "做空", err.Error())
				} else {
					notify.BuyOrderSuccess(coin.Symbol, quantity, nowPrice, "做空")
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

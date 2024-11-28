package feature

import (
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/lang"
	"go_binance_futures/models"
	"go_binance_futures/notify"
	"go_binance_futures/utils"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

var flagFuturesNotice = 0
func NoticeAndAutoOrder(systemConfig models.Config) {
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


type PositionNoticeInfo struct {
	Status string // income_positive:盈利 income_negative:亏损
	NoticeTime int64
}

var flagPositionConvertNotice = 0
var positionNotices = make(map[string]PositionNoticeInfo)
func PositionConvertNotice(systemConfig models.Config) {
	if (systemConfig.FuturesPositionConvertEnable == 1) {
		if (flagPositionConvertNotice == 0) {
			logs.Info("futures position convert notice bot start")
			flagPositionConvertNotice = 1
		}
	} else {
		if (flagPositionConvertNotice == 1) {
			logs.Info("futures position convert notice bot stop")
			flagPositionConvertNotice = 0
		}
		return
	}
	
	positions, err := binance.GetPosition(binance.PositionParams{})
	if err != nil {
		logs.Error("GetPosition err in PositionConvertNotice:", err)
		return
	}
	
	for _, position := range positions {
		positionAmtFloat, _ := strconv.ParseFloat(position.PositionAmt, 64)
		positionAmtFloatAbs := math.Abs(positionAmtFloat) // 空单为负数,纠正为绝对值
		if positionAmtFloatAbs < 0.0000000001 {// 没有持仓的
			delete(positionNotices, position.Symbol + position.PositionSide) // 删除已经平仓的数据
			continue
		}
		unRealizedProfit, _ := strconv.ParseFloat(position.UnRealizedProfit, 64)
		status := "income_positive" // 盈利
		if unRealizedProfit < 0 {
			status = "income_negative" // 亏损
		}
		
		noticeInfo, ok := positionNotices[position.Symbol + position.PositionSide]
		if ok {
			// 存在记录
			if (status != noticeInfo.Status && time.Now().Unix() - noticeInfo.NoticeTime > 60 * 3) {
				// 状态变化,并且上次通知时间超过3分钟，重新记录新的状态，并通知
				positionNotices[position.Symbol + position.PositionSide] = PositionNoticeInfo{
					Status: status,
					NoticeTime: time.Now().Unix(),
				}
				// 状态变化,并且上次通知时间超过1小时
				pusher.FuturesPositionConvert(notify.FuturesPositionConvertParams{
					Title: lang.Lang("futures.notice_position_convert_title"),
					Symbol: position.Symbol,
					Status: status,
					PositionSide: strings.ToLower(position.PositionSide),
					Price: position.MarkPrice,
					Leverage: position.Leverage,
					UnRealizedProfit: position.UnRealizedProfit,
				})
			}
		} else {
			// 第一次,只进行记录
			positionNotices[position.Symbol + position.PositionSide] = PositionNoticeInfo{
				Status: status,
				NoticeTime: time.Now().Unix(),
			}
		}
	}
}
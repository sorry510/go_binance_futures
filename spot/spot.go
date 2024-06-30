package spot

import (
	"go_binance_futures/models"
	"go_binance_futures/spot/api/binance"
	"go_binance_futures/spot/notify"
	"go_binance_futures/utils"
	"strconv"

	spot_api "github.com/adshao/go-binance/v2"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

func TryRush() {
	o := orm.NewOrm()
	var coins []models.NewSymbols
	o.QueryTable("new_symbols").OrderBy("ID").Filter("enable", 1).Filter("type", 1).All(&coins) // 允许抢购的币
	
	notHasSizeSymbols := []string{}
	
	for _, coin := range coins {
		if coin.StepSize != "0" {
			if coin.Side == "buy" {
				_, err := tryBuyMarket(coin.Symbol, coin.Usdt, coin.StepSize)
				if err == nil {
					coin.Enable = 0 // 更新为禁用
				}
				orm.NewOrm().Update(&coin)
			}
			if coin.Side == "sell" {
				_, err := trySellMarket(coin.Symbol, coin.Quantity, coin.StepSize)
				if err == nil {
					coin.Enable = 0 // 更新为禁用
				}
				orm.NewOrm().Update(&coin)
			}
		} else {
			notHasSizeSymbols = append(notHasSizeSymbols, coin.Symbol)
		}
	}
	if len(notHasSizeSymbols) == 0 {
		// logs.Info("没有币需要更新交易精度")
		return
	}
	
	res, err := binance.GetExchangeInfo()
	if err != nil {
		logs.Error("GetExchangeInfoError:", err)
		return
	}
	symbolMap := make(map[string]string)
	for _, item := range res.Symbols {
		lotSizeFilter := item.LotSizeFilter()
		if lotSizeFilter != nil {
			symbolMap[item.Symbol] = lotSizeFilter.StepSize
		}
	}
	
	for _, coin := range coins {
		// 找到了币的精度，说明币可能上线了
		if stepSize, ok := symbolMap[coin.Symbol]; ok {
			logs.Info("lotSize:", stepSize)
			if coin.Side == "buy" {
				_, err := tryBuyMarket(coin.Symbol, coin.Usdt, stepSize)
			
				if err == nil {
					coin.Enable = 0 // 更新为禁用
					logs.Info("抢购成功，关闭交易")
				}
			}
			if coin.Side == "sell" {
				_, err := trySellMarket(coin.Symbol, coin.Usdt, stepSize)
			
				if err == nil {
					coin.Enable = 0 // 更新为禁用
					logs.Info("抢卖成功，关闭交易")
				}
			}
			
			coin.StepSize = stepSize
			orm.NewOrm().Update(&coin)
		} else {
			logs.Info("还未上线此币种,未确定交易价格数量精度:", coin.Symbol)
		}
	}
}

func NoticeAndAutoOrder() {
	o := orm.NewOrm()
	var coins []models.NoticeSymbols
	o.QueryTable("notice_symbols").OrderBy("ID").Filter("enable", 1).Filter("type", 1).All(&coins) // 通知币列表
	
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
				// 如果是买币，价格低于预警价格，进行通知
				canOrder = true
				notify.NoticeSpotCoin(coin.Symbol, "买入", coin.NoticePrice, autoOrderText)
			}
			if (nowPrice >= noticePrice && coin.Side == "sell") {
				// 如果是买币，价格高于预警价格，进行通知
				canOrder = true
				notify.NoticeSpotCoin(coin.Symbol, "卖出", coin.NoticePrice, autoOrderText)
			}
			if canOrder {
				coin.HasNotice = 1
				coin.Enable = 0 // 更新为关闭
				orm.NewOrm().Update(&coin)
			}
		}
		if coin.AutoOrder == 1 && canOrder {
			quantity, _ := strconv.ParseFloat(coin.Quantity, 64) // 交易数量
			if quantity == 0 {
				// 如果没有交易数量，从usdt推算出交易数量
				usdt_float64, _ := strconv.ParseFloat(coin.Usdt, 64) // 交易金额
				quantity = usdt_float64 / nowPrice  // 购买数量
			}
			quantity = utils.GetTradePrecision(quantity, coin.StepSize) // 合理精度的数量
			if (coin.Side == "buy") {
				_, err := binance.BuyMarket(coin.Symbol, quantity)
				if err != nil {
					logs.Info("购买失败symbol:", coin.Symbol)
				} else {
					// 购买成功
					notify.BuyOrderSuccess(coin.Symbol, quantity, nowPrice)
					if (coin.ProfitPrice != "0") {
						// 挂一个止盈单
						profit_price_float64, _ := strconv.ParseFloat(coin.ProfitPrice, 64) // 交易金额
						profit_price_float64 = utils.GetTradePrecision(profit_price_float64, coin.TickSize) // 合理精度价格
						binance.OrderTakeProfit(coin.Symbol, quantity, profit_price_float64)
					}
					if (coin.LossPrice != "0") {
						// 挂一个止损单
						loss_price_float64, _ := strconv.ParseFloat(coin.LossPrice, 64) // 交易金额
						loss_price_float64 = utils.GetTradePrecision(loss_price_float64, coin.TickSize) // 合理精度价格
						binance.OrderStopLoss(coin.Symbol, quantity, loss_price_float64)
					}
					coin.Quantity = strconv.FormatFloat(quantity, 'f', -1, 64) // 更新数量，为卖出时使用
					orm.NewOrm().Update(&coin)
				}
			}
			if (coin.Side == "sell") {
				_, err := binance.SellMarket(coin.Symbol, quantity)
				if err != nil {
					logs.Info("卖出失败symbol:", coin.Symbol)
				} else {
					// 卖出成功
					notify.SellOrderSuccess(coin.Symbol, quantity, resPrice[0].Price)
				}
			}
		}
	}
}

func tryBuyMarket(symbol string, usdt string, stepSize string) (res *spot_api.CreateOrderResponse, err error) {
	logs.Info("尝试开始抢币symbol:", symbol)
	resPrice, err1 := binance.GetTickerPrice(symbol)
	if err1 != nil {
		logs.Info("还未上线此币种,未确定交易价格symbol:", symbol)
		return nil, err1
	}
	usdt_float64, _ := strconv.ParseFloat(usdt, 64) // 交易金额
	buyPrice, _ := strconv.ParseFloat(resPrice[0].Price, 64) // 预计交易价格
	logs.Info("尝试开始抢币预计价格为:", buyPrice)
	quantity := usdt_float64 / buyPrice  // 购买数量
	quantity = utils.GetTradePrecision(quantity, stepSize) // 合理精度的价格
	// logs.Info("symbol:", symbol, "buyPrice:", buyPrice, "quantity:", quantity)
	
	res, err = binance.BuyMarket(symbol, quantity)
	if err != nil {
		logs.Info("购买失败symbol:", symbol)
	} else {
		// 购买成功
		notify.BuyOrderSuccess(symbol, quantity, buyPrice)
	}
	return res, err
}

func trySellMarket(symbol string, quantity string, stepSize string) (res *spot_api.CreateOrderResponse, err error) {
	logs.Info("尝试开始抢卖symbol:", symbol)
	quantity_float64, _ := strconv.ParseFloat(quantity, 64) // 卖单数量
	quantity_float64 = utils.GetTradePrecision(quantity_float64, stepSize) // 合理精度的数量
	// logs.Info("symbol:", symbol, "buyPrice:", buyPrice, "quantity:", quantity)]
	
	
	res, err = binance.SellMarket(symbol, quantity_float64)
	if err != nil {
		logs.Info("卖出失败symbol:", symbol)
	} else {
		// 卖出成功
		notify.SellOrderSuccess(symbol, quantity_float64, res.Price)
	}
	return res, err
}
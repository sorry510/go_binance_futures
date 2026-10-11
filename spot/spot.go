package spot

import (
	"context"
	"sync/atomic"
	"time"

	"go_binance_futures/lang"
	"go_binance_futures/models"
	"go_binance_futures/notify"
	"go_binance_futures/service/binanceapiusage"
	"go_binance_futures/spot/api/binance"
	"go_binance_futures/utils"
	"strconv"

	spot_api "github.com/adshao/go-binance/v2"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

var pusher = notify.GetNotifyChannel()

var flagSpotNew = 0

const (
	spotRushExchangeInfoRefreshInterval = 2 * time.Second
	spotRushFallbackProbeInterval       = 10 * time.Second
)

var spotRushExchangeInfoLastRefreshAt atomic.Int64

func TryRush(systemConfig models.Config) {
	if systemConfig.SpotNewEnable == 1 {
		if flagSpotNew == 0 {
			logs.Info("spot rush bot start")
			flagSpotNew = 1
		}
	} else {
		if flagSpotNew == 1 {
			logs.Info("spot rush bot stop")
			flagSpotNew = 0
		}
		return
	}

	o := orm.NewOrm()
	var coins []models.NewSymbols
	o.QueryTable("new_symbols").OrderBy("ID").Filter("enable", 1).Filter("type", 1).All(&coins) // 允许抢购的币

	notHasSizeSymbols := []string{}

	for _, coin := range coins {
		if coin.StepSize != "0" {
			if coin.Side == "buy" {
				_, err := tryBuyMarket(coin, coin.StepSize)
				if err == nil {
					coin.Enable = 0 // 更新为禁用
				}
				orm.NewOrm().Update(&coin)
			}
			if coin.Side == "sell" {
				_, err := trySellMarket(coin, coin.StepSize)
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

	hasFreshListedSymbol := false
	for _, symbol := range notHasSizeSymbols {
		if _, ok := binance.GetFreshSpotTickerPrice(symbol); ok {
			hasFreshListedSymbol = true
			break
		}
	}
	probeInterval := spotRushExchangeInfoRefreshInterval
	if binance.SpotTickerWSFresh() {
		if !hasFreshListedSymbol {
			return
		}
	} else {
		probeInterval = spotRushFallbackProbeInterval
	}
	if !reserveSpotRushExchangeInfoRefresh(probeInterval) {
		return
	}

	ctx := binanceapiusage.WithSource(context.Background(), "spot_new_coin_rush")
	res, err := binance.GetExchangeInfoFreshContext(ctx)
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
				_, err := tryBuyMarket(coin, stepSize)

				if err == nil {
					coin.Enable = 0 // 更新为禁用
					logs.Info("抢购成功，关闭交易")
				}
			}
			if coin.Side == "sell" {
				_, err := trySellMarket(coin, stepSize)

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

var flagSpotNotice = 0

func NoticeAndAutoOrder(systemConfig models.Config) {
	if systemConfig.NoticeCoinEnable == 1 {
		if flagSpotNotice == 0 {
			logs.Info("spot notice bot start")
			flagSpotNotice = 1
		}
	} else {
		if flagSpotNotice == 1 {
			logs.Info("spot notice bot stop")
			flagSpotNotice = 0
		}
		return
	}

	o := orm.NewOrm()
	var coins []models.NoticeSymbols
	o.QueryTable("notice_symbols").OrderBy("ID").Filter("enable", 1).Filter("type", 1).Filter("has_notice", 0).All(&coins) // 通知币列表

	for _, coin := range coins {
		logs.Info("notice_futures: ", coin.Symbol)
		priceCtx := binanceapiusage.WithSource(context.Background(), "spot_notice")
		resPrice, errPrice := binance.GetTickerPriceContext(priceCtx, coin.Symbol)
		if errPrice != nil {
			logs.Info("无法进行通知, 还未上线此币种: ", coin.Symbol)
			continue
		}
		var canOrder = false
		nowPrice, _ := strconv.ParseFloat(resPrice[0].Price, 64)   // 预计交易价格
		noticePrice, _ := strconv.ParseFloat(coin.NoticePrice, 64) // 预警价格
		if coin.HasNotice == 0 {
			var autoOrderText = "no"
			if coin.AutoOrder == 1 {
				autoOrderText = "yes"
			}
			if nowPrice <= noticePrice && coin.Side == "buy" {
				// 买币，价格低于预警价格，进行通知
				canOrder = true
				pusher.SetModuleName("coin_notice").SpotNotice(notify.SpotNoticeParams{
					Title:     lang.Lang("spot.notice_price_title"),
					Symbol:    coin.Symbol,
					Side:      "buy",
					Price:     noticePrice,
					AutoOrder: autoOrderText,
				})
			}
			if nowPrice >= noticePrice && coin.Side == "sell" {
				// 卖币，价格高于预警价格，进行通知
				canOrder = true
				pusher.SetModuleName("coin_notice").SpotNotice(notify.SpotNoticeParams{
					Title:     lang.Lang("spot.notice_price_title"),
					Symbol:    coin.Symbol,
					Side:      "sell",
					Price:     noticePrice,
					AutoOrder: autoOrderText,
				})
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
				quantity = usdt_float64 / nowPrice                   // 购买数量
			}
			quantity = utils.GetTradePrecision(quantity, coin.StepSize) // 合理精度的数量
			if coin.Side == "buy" {
				_, err := binance.BuyMarket(coin.Symbol, quantity)
				if err != nil {
					logs.Error("购买失败symbol:", coin.Symbol)
					logs.Error("err:", err.Error())
					// pusher.SetModuleName("coin_notice").SpotOrder(notify.SpotOrderParams{
					// 	Title: lang.Lang("spot.notice_title"),
					// 	Symbol: coin.Symbol,
					// 	Side: "sell",
					// 	Price: nowPrice,
					// 	Quantity: quantity,
					// 	Remarks: lang.Lang("spot.notice_auto_order"),
					// 	Status: "fail",
					// 	Error: err.Error(),
					// })
				} else {
					// 购买成功
					pusher.SetModuleName("coin_notice").SpotOrder(notify.SpotOrderParams{
						Title:    lang.Lang("spot.notice_title"),
						Symbol:   coin.Symbol,
						Side:     "sell",
						Price:    nowPrice,
						Quantity: quantity,
						Remarks:  lang.Lang("spot.notice_auto_order"),
						Status:   "success",
						Error:    "",
					})

					// 现货不支持挂单止盈止损
					// if (coin.ProfitPrice != "0") {
					// 	// 挂一个止盈单
					// 	profit_price_float64, _ := strconv.ParseFloat(coin.ProfitPrice, 64) // 交易金额
					// 	profit_price_float64 = utils.GetTradePrecision(profit_price_float64, coin.TickSize) // 合理精度价格
					// 	binance.OrderTakeProfit(coin.Symbol, quantity, profit_price_float64)
					// }
					// if (coin.LossPrice != "0") {
					// 	// 挂一个止损单
					// 	loss_price_float64, _ := strconv.ParseFloat(coin.LossPrice, 64) // 交易金额
					// 	loss_price_float64 = utils.GetTradePrecision(loss_price_float64, coin.TickSize) // 合理精度价格
					// 	binance.OrderStopLoss(coin.Symbol, quantity, loss_price_float64)
					// }

					coin.Quantity = strconv.FormatFloat(quantity, 'f', -1, 64) // 更新数量，为卖出时使用
					orm.NewOrm().Update(&coin)
				}
			}
			if coin.Side == "sell" {
				_, err := binance.SellMarket(coin.Symbol, quantity)
				if err != nil {
					logs.Error("卖出失败symbol:", coin.Symbol)
					logs.Error("err:", err.Error())
					// pusher.SetModuleName("coin_notice").SpotOrder(notify.SpotOrderParams{
					// 	Title: lang.Lang("spot.notice_title"),
					// 	Symbol: coin.Symbol,
					// 	Side: "sell",
					// 	Quantity: quantity,
					// 	Remarks: lang.Lang("spot.notice_auto_order"),
					// 	Status: "fail",
					// 	Error: err.Error(),
					// })
				} else {
					// 卖出成功
					price_float64, _ := strconv.ParseFloat(resPrice[0].Price, 64) // 卖单数量
					pusher.SetModuleName("coin_notice").SpotOrder(notify.SpotOrderParams{
						Title:    lang.Lang("spot.notice_title"),
						Symbol:   coin.Symbol,
						Side:     "sell",
						Price:    price_float64,
						Quantity: quantity,
						Remarks:  lang.Lang("spot.notice_auto_order"),
						Status:   "success",
						Error:    "",
					})
				}
			}
		}
	}
}

func tryBuyMarket(coin models.NewSymbols, stepSize string) (res *spot_api.CreateOrderResponse, err error) {
	symbol := coin.Symbol
	usdt := coin.Usdt
	usdt_float64, _ := strconv.ParseFloat(usdt, 64) // 交易金额
	buyPrice := 0.0
	logs.Info("尝试开始抢币symbol:", symbol)

	if coin.ExpectPrice != "0" {
		// 定义的挂单价格
		buyPrice, _ = strconv.ParseFloat(coin.ExpectPrice, 64) // 挂单价格
	} else {
		// 获取最新的交易价格
		priceCtx := binanceapiusage.WithSource(context.Background(), "spot_new_coin_rush")
		resPrice, err1 := binance.GetTickerPriceContext(priceCtx, symbol)
		if err1 != nil {
			logs.Info("还未上线此币种,未确定交易价格symbol:", symbol)
			return nil, err1
		}
		buyPrice, _ = strconv.ParseFloat(resPrice[0].Price, 64) // 预计交易价格
	}

	logs.Info("尝试开始抢币预计价格为:", buyPrice)
	quantity := usdt_float64 / buyPrice                    // 购买数量
	quantity = utils.GetTradePrecision(quantity, stepSize) // 合理精度的数量
	// logs.Info("symbol:", symbol, "buyPrice:", buyPrice, "quantity:", quantity)

	if coin.ExpectPrice != "0" {
		// 挂单价格
		res, err = binance.BuyLimit(symbol, quantity, buyPrice)
	} else {
		// 市价
		res, err = binance.BuyMarket(symbol, quantity)
	}

	if err != nil {
		logs.Error("购买失败symbol:", symbol)
		logs.Error("err:", err.Error())
	} else {
		// 购买成功
		pusher.SetModuleName("new_coin_rush").SpotOrder(notify.SpotOrderParams{
			Title:    lang.Lang("spot.new_coin_rush_notice_title"),
			Symbol:   symbol,
			Side:     "buy",
			Price:    buyPrice,
			Quantity: quantity,
			Remarks:  lang.Lang("spot.new_coin_rush_buy"),
			Status:   "success",
			Error:    "",
		})
	}
	return res, err
}

func trySellMarket(coin models.NewSymbols, stepSize string) (res *spot_api.CreateOrderResponse, err error) {
	symbol := coin.Symbol
	logs.Info("尝试开始抢卖symbol:", symbol)
	quantity := coin.Quantity                                              // 卖出数量
	quantity_float64, _ := strconv.ParseFloat(quantity, 64)                // 卖单数量
	quantity_float64 = utils.GetTradePrecision(quantity_float64, stepSize) // 合理精度的数量
	// logs.Info("symbol:", symbol, "buyPrice:", buyPrice, "quantity:", quantity)]

	if coin.ExpectPrice != "0" {
		// 挂单价格
		sellPrice, _ := strconv.ParseFloat(coin.ExpectPrice, 64)      // 挂单价格
		sellPrice = utils.GetTradePrecision(sellPrice, coin.TickSize) // 合理精度的价格
		res, err = binance.SellLimit(symbol, quantity_float64, sellPrice)
	} else {
		// 市价
		res, err = binance.SellMarket(symbol, quantity_float64)
	}

	if err != nil {
		logs.Error("卖出失败symbol:", symbol)
		logs.Error("err:", err.Error())
		// pusher.SetModuleName("new_coin_rush").SpotOrder(notify.SpotOrderParams{
		// 	Title: lang.Lang("spot.new_coin_rush_notice_title"),
		// 	Symbol: symbol,
		// 	Side: "sell",
		// 	Quantity: quantity_float64,
		// 	Remarks: lang.Lang("spot.new_coin_rush_sell"),
		// 	Status: "fail",
		// 	Error: err.Error(),
		// })
	} else {
		// 卖出成功
		price_float64, _ := strconv.ParseFloat(res.Price, 64) // 卖单数量
		pusher.SetModuleName("new_coin_rush").SpotOrder(notify.SpotOrderParams{
			Title:    lang.Lang("spot.new_coin_rush_notice_title"),
			Symbol:   symbol,
			Side:     "sell",
			Price:    price_float64,
			Quantity: quantity_float64,
			Remarks:  lang.Lang("spot.new_coin_rush_sell"),
			Status:   "success",
			Error:    "",
		})
	}
	return res, err
}

var flagSpotListen = 0
var spotListenDisabledLogEmitted bool

func ListenCoin(systemConfig models.Config) {
	if systemConfig.ListenCoinEnable != 1 {
		if flagSpotListen == 1 {
			logs.Info("spot listen bot stop: global listener disabled")
			flagSpotListen = 0
			spotListenDisabledLogEmitted = true
		} else if !spotListenDisabledLogEmitted {
			logs.Info("spot listen inactive: global ListenCoinEnable is disabled")
			spotListenDisabledLogEmitted = true
		}
		return
	}
	if flagSpotListen == 0 {
		logs.Info("spot listen bot start: custom strategies only")
		flagSpotListen = 1
	}
	spotListenDisabledLogEmitted = false

	o := orm.NewOrm()
	var coins []models.ListenSymbols
	if _, err := o.QueryTable("listen_symbols").OrderBy("ID").Filter("enable", 1).Filter("type", 1).All(&coins); err != nil {
		logs.Error("spot listen query enabled symbols failed: %v", err)
		return
	}

	counts := make(map[spotListenOutcome]int)
	for _, coin := range coins {
		if coin.ListenType != "custom" {
			// Legacy K-line/KC listeners remain stored but never execute.
			counts[spotListenLegacy]++
			logSpotListenOutcome(coin, spotListenLegacy)
			continue
		}
		// Like the futures listener, report every scan of every enabled symbol.
		// This intentionally logs at the scheduler cadence (currently every 2s),
		// including during notification cooldown and when no rule matches.
		logs.Info("listen spot custom: %s", coin.Symbol)
		outcome := spotKlineCustomListen(coin)
		counts[outcome]++
		logSpotListenOutcome(coin, outcome)
	}
	logSpotListenSummary(len(coins), counts)
}

// 更新币种的交易精度和插入新币
func UpdateSymbolsTradePrecision() {
	exchange, err := binance.GetExchangeInfo()
	if err != nil {
		logs.Error("spot exchange info unavailable for symbol refresh:", err)
		return
	}
	o := orm.NewOrm()
	// Read existing symbol metadata once, not once per exchange pair.
	var existing []models.SpotSymbols
	if _, err := o.QueryTable("spot_symbols").All(&existing, "Symbol", "TickSize", "StepSize"); err != nil {
		logs.Error("read existing spot symbols:", err)
		return
	}
	known := make(map[string]models.SpotSymbols, len(existing))
	for _, value := range existing {
		known[value.Symbol] = value
	}
	var created, adjusted int
	for _, symbol := range exchange.Symbols {
		if symbol.Status != "TRADING" || !symbol.IsSpotTradingAllowed {
			continue
		}
		kind := ""
		switch symbol.QuoteAsset {
		case "USDT", "USDC", "FDUSD":
			kind = symbol.QuoteAsset
		default:
			continue
		}
		tickSize, stepSize := "", ""
		if filter := symbol.PriceFilter(); filter != nil {
			tickSize = filter.TickSize
		}
		if filter := symbol.LotSizeFilter(); filter != nil {
			stepSize = filter.StepSize
		}
		previous, found := known[symbol.Symbol]
		if found {
			if previous.TickSize != tickSize || previous.StepSize != stepSize {
				if _, err := o.QueryTable("spot_symbols").Filter("symbol", symbol.Symbol).Update(orm.Params{
					"tickSize": tickSize, "stepSize": stepSize,
				}); err != nil {
					logs.Error("update spot symbol precision:", symbol.Symbol, err)
				} else {
					adjusted++
				}
			}
			continue
		}
		// Creating metadata does not enable spot trading. WS fills prices.
		_, err := o.Insert(&models.SpotSymbols{
			Symbol:       symbol.Symbol,
			Enable:       0,
			TickSize:     tickSize,
			StepSize:     stepSize,
			Usdt:         "10",
			Profit:       "10",
			Loss:         "10",
			Technology:   "{}",
			Strategy:     "[]",
			StrategyType: "global",
			Type:         kind,
		})
		if err != nil {
			// If the WS upsert inserted the pair concurrently, update only
			// missing trade precision and preserve all user preferences.
			if o.QueryTable("spot_symbols").Filter("symbol", symbol.Symbol).Exist() {
				_, err = o.QueryTable("spot_symbols").Filter("symbol", symbol.Symbol).Update(orm.Params{
					"tickSize": tickSize, "stepSize": stepSize,
				})
			}
			if err != nil {
				logs.Error("insert spot symbol metadata:", symbol.Symbol, err)
			}
			continue
		}
		created++
	}
	logs.Info("spot symbol precision refresh finished, new:", created, "adjusted:", adjusted)
}
func reserveSpotRushExchangeInfoRefresh(interval time.Duration) bool {
	if interval <= 0 {
		interval = spotRushExchangeInfoRefreshInterval
	}
	now := time.Now().UnixMilli()
	for {
		last := spotRushExchangeInfoLastRefreshAt.Load()
		if last > 0 && now-last < interval.Milliseconds() {
			return false
		}
		if spotRushExchangeInfoLastRefreshAt.CompareAndSwap(last, now) {
			return true
		}
	}
}

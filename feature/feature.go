package feature

import (
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/feature/strategy"
	"go_binance_futures/feature/strategy/coin"
	"go_binance_futures/feature/strategy/line"
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
	"github.com/beego/beego/v2/core/config"
	"github.com/beego/beego/v2/core/logs"
)

var profit, _ = config.String("trade::profit")
var profit_float64, _ = strconv.ParseFloat(profit, 64)

var loss, _ = config.String("trade::loss")
var loss_float64, _ = strconv.ParseFloat(loss, 64)

// var leverage_int, _ = strconv.Atoi(leverage)

var buy_timeout, _ = config.String("coin::buy_timeout") // 秒级别
var buy_timeout_int, _ = strconv.Atoi(buy_timeout)
var buy_timeout_int64 = int64(buy_timeout_int)

var max_count, _ = config.String("coin::max_count")
var max_count_int, _ = strconv.Atoi(max_count)

var order_type, _ = config.String("coin::order_type")
var allow_long, _ = config.String("coin::allow_long")
var allow_short, _ = config.String("coin::allow_short")

var strategy_trade, _ = config.String("trade::strategy_trade") // 交易策略
var strategy_coin, _ = config.String("trade::strategy_coin") // 选币策略

var coinStrategy = GetCoinStrategy(strategy_coin)
var lineStrategy = GetLineStrategy(strategy_trade)

var pusher = notify.GetNotifyChannel()

func StartTrade() {
	// logs.Info("StartTrade")
	
	/************************************************寻找交易币种 start******************************************************************* */
	allCoins, err := GetAllSymbols()
	if err != nil {
		logs.Error("GetAllSymbols err:", err)
	}
	coins := coinStrategy.SelectCoins(allCoins)
	if coins == nil {
		logs.Error("coins SelectCoins is nil")
		return
	}
	/************************************************寻找交易币种 end******************************************************************* */
	
	
	/************************************************获取账户信息 start******************************************************************* */
	positions, _ := binance.GetPosition()
	if err != nil {
		logs.Error("GetPosition err:", err)
	}
	
	allOpenOrders, err := binance.GetOpenOrder()
	if err != nil {
		logs.Error("GetOpenOrder err:", err)
	}
	/************************************************获取账户信息 end******************************************************************* */
	
	
	/*************************************************挂单已经超过设置的超时时间，撤销挂单 start************************************************************ */
	exclude_symbols_map := GetExcludeSymbolsMap()
	cancelTimeoutOrder(exclude_symbols_map, allOpenOrders)
	/*************************************************挂单已经超过设置的超时时间，撤销挂单 end************************************************************ */
	
	
	/*************************************************平仓(止盈或止损)已经有持仓的币(排除手动交易白名单) start************************************************************ */
	positionCount := 0 // 当前仓位数量
	for _, position := range positions {
		coin_profit_float64 := profit_float64 // 全局定义
		coin_loss_float64 := loss_float64
		for _, coin := range allCoins {
			if coin.Symbol == position.Symbol {
				// 自定义可以覆盖全局
				if coin.Profit != "0" {
					coin_profit_float64, _ = strconv.ParseFloat(coin.Profit, 64)
				}
				if coin.Loss != "0" {
					coin_loss_float64, _ = strconv.ParseFloat(coin.Loss, 64)
				}
			}
		}
		positionAmtFloat, _ := strconv.ParseFloat(position.PositionAmt, 64)
		positionAmtFloatAbs := math.Abs(positionAmtFloat) // 空单为负数,纠正为绝对值
		unRealizedProfit, _ := strconv.ParseFloat(position.UnRealizedProfit, 64)
		leverage_float64, _ := strconv.ParseFloat(position.Leverage, 64)
		entryPrice, _ := strconv.ParseFloat(position.EntryPrice, 64)
		nowProfit := (unRealizedProfit / (positionAmtFloatAbs * entryPrice)) * leverage_float64 * 100 // 当前收益率(正为盈利，负为亏损)
		
		if positionAmtFloatAbs < 0.0000000001 {// 没有持仓的
			continue
		}
		if _, exist := exclude_symbols_map[position.Symbol]; exist { // 在白名单内
			continue
		}
		if lineStrategy.AutoStopOrder(position, nowProfit) { // 触发策略,风向改变,强制平仓
			logs.Info("%s:auto_stop_start", position.Symbol)
			if position.PositionSide == "LONG" {
				result, err := binance.SellMarket(position.Symbol, positionAmtFloatAbs, futures.PositionSideTypeLong)
				if err == nil {
					// 数据库写入订单
					insertCloseOrder(position, positionAmtFloatAbs, unRealizedProfit, result.AvgPrice)
					
					avgPrice, _ := strconv.ParseFloat(result.AvgPrice, 64)
					pusher.FuturesCloseOrder(notify.FuturesOrderParams{
						Title: lang.Lang("futures.close_notice_title"),
						Symbol: position.Symbol,
						Side: "sell",
						PositionSide: "long",
						Price: avgPrice,
						Quantity: positionAmtFloat,
						Profit: unRealizedProfit,
						Remarks: lang.Lang("futures.wind_of_change"),
						Status: "success",
					})
				} else {
					pusher.FuturesCloseOrder(notify.FuturesOrderParams{
						Title: lang.Lang("futures.close_notice_title"),
						Symbol: position.Symbol,
						Side: "sell",
						PositionSide: "long",
						Quantity: positionAmtFloat,
						Profit: unRealizedProfit,
						Remarks: lang.Lang("futures.wind_of_change"),
						Status: "fail",
						Error: err.Error(),
					})
				}
			}
			if position.PositionSide == "SHORT" {
				result, err := binance.BuyMarket(position.Symbol, positionAmtFloatAbs, futures.PositionSideTypeShort)
				if err == nil {
					// 数据库写入订单
					insertCloseOrder(position, positionAmtFloatAbs, unRealizedProfit, result.AvgPrice)
					
					avgPrice, _ := strconv.ParseFloat(result.AvgPrice, 64)
					pusher.FuturesCloseOrder(notify.FuturesOrderParams{
						Title: lang.Lang("futures.close_notice_title"),
						Symbol: position.Symbol,
						Side: "buy",
						PositionSide: "short",
						Price: avgPrice,
						Quantity: positionAmtFloat,
						Profit: unRealizedProfit,
						Remarks: lang.Lang("futures.wind_of_change"),
						Status: "success",
					})
				} else {
					pusher.FuturesCloseOrder(notify.FuturesOrderParams{
						Title: lang.Lang("futures.close_notice_title"),
						Symbol: position.Symbol,
						Side: "buy",
						PositionSide: "short",
						Quantity: positionAmtFloat,
						Profit: unRealizedProfit,
						Remarks: lang.Lang("futures.wind_of_change"),
						Status: "fail",
						Error: err.Error(),
					})
				}
			}
			logs.Info("%s:auto_stop_end", position.Symbol)
			continue
		}
		if nowProfit <= -coin_loss_float64 { // 平仓(止损)
			if lineStrategy.CanOrderComplete(position.Symbol, position.PositionSide) { // 
				if position.PositionSide == "LONG" {
					result, err := binance.SellMarket(position.Symbol, positionAmtFloatAbs, futures.PositionSideTypeLong)
					if err == nil {
						// 数据库写入订单
						insertCloseOrder(position, positionAmtFloatAbs, unRealizedProfit, result.AvgPrice)
						
						avgPrice, _ := strconv.ParseFloat(result.AvgPrice, 64)
						pusher.FuturesCloseOrder(notify.FuturesOrderParams{
							Title: lang.Lang("futures.close_notice_title"),
							Symbol: position.Symbol,
							Side: "sell",
							PositionSide: "long",
							Price: avgPrice,
							Quantity: positionAmtFloat,
							Profit: unRealizedProfit,
							Remarks: lang.Lang("futures.stop_loss"),
							Status: "success",
						})
					} else {
						pusher.FuturesCloseOrder(notify.FuturesOrderParams{
							Title: lang.Lang("futures.close_notice_title"),
							Symbol: position.Symbol,
							Side: "sell",
							PositionSide: "long",
							Quantity: positionAmtFloat,
							Profit: unRealizedProfit,
							Remarks: lang.Lang("futures.stop_loss"),
							Status: "fail",
							Error: err.Error(),
						})
					}
				}
				if position.PositionSide == "SHORT" {
					result, err := binance.BuyMarket(position.Symbol, positionAmtFloatAbs, futures.PositionSideTypeShort)
					if err == nil {
						// 数据库写入订单
						insertCloseOrder(position, positionAmtFloatAbs, unRealizedProfit, result.AvgPrice)
						
						avgPrice, _ := strconv.ParseFloat(result.AvgPrice, 64)
						pusher.FuturesCloseOrder(notify.FuturesOrderParams{
							Title: lang.Lang("futures.close_notice_title"),
							Symbol: position.Symbol,
							Side: "buy",
							PositionSide: "short",
							Price: avgPrice,
							Quantity: positionAmtFloat,
							Profit: unRealizedProfit,
							Remarks: lang.Lang("futures.stop_loss"),
							Status: "success",
						})
					} else {
						pusher.FuturesCloseOrder(notify.FuturesOrderParams{
							Title: lang.Lang("futures.close_notice_title"),
							Symbol: position.Symbol,
							Side: "buy",
							PositionSide: "short",
							Quantity: positionAmtFloat,
							Profit: unRealizedProfit,
							Remarks: lang.Lang("futures.stop_loss"),
							Status: "fail",
							Error: err.Error(),
						})
					}
				}
				continue
			}
		}
		if nowProfit >= coin_profit_float64 { // 平仓(止盈)
			if lineStrategy.CanOrderComplete(position.Symbol, position.PositionSide) {
				if position.PositionSide == "LONG" {
					result, err := binance.SellMarket(position.Symbol, positionAmtFloatAbs, futures.PositionSideTypeLong)
					if err == nil {
						// 数据库写入订单
						insertCloseOrder(position, positionAmtFloatAbs, unRealizedProfit, result.AvgPrice)
						
						avgPrice, _ := strconv.ParseFloat(result.AvgPrice, 64)
						pusher.FuturesCloseOrder(notify.FuturesOrderParams{
							Title: lang.Lang("futures.close_notice_title"),
							Symbol: position.Symbol,
							Side: "sell",
							PositionSide: "long",
							Price: avgPrice,
							Quantity: positionAmtFloat,
							Profit: unRealizedProfit,
							Remarks: lang.Lang("futures.target_profit"),
							Status: "success",
						})
					} else {
						pusher.FuturesCloseOrder(notify.FuturesOrderParams{
							Title: lang.Lang("futures.close_notice_title"),
							Symbol: position.Symbol,
							Side: "sell",
							PositionSide: "long",
							Quantity: positionAmtFloat,
							Profit: unRealizedProfit,
							Remarks: lang.Lang("futures.target_profit"),
							Status: "fail",
							Error: err.Error(),
						})
					}
				}
				if position.PositionSide == "SHORT" {
					result, err := binance.BuyMarket(position.Symbol, positionAmtFloatAbs, futures.PositionSideTypeShort)
					if err == nil {
						// 数据库写入订单
						insertCloseOrder(position, positionAmtFloatAbs, unRealizedProfit, result.AvgPrice)
						
						avgPrice, _ := strconv.ParseFloat(result.AvgPrice, 64)
						pusher.FuturesCloseOrder(notify.FuturesOrderParams{
							Title: lang.Lang("futures.close_notice_title"),
							Symbol: position.Symbol,
							Side: "buy",
							PositionSide: "short",
							Price: avgPrice,
							Quantity: positionAmtFloat,
							Profit: unRealizedProfit,
							Remarks: lang.Lang("futures.target_profit"),
							Status: "success",
						})
					} else {
						pusher.FuturesCloseOrder(notify.FuturesOrderParams{
							Title: lang.Lang("futures.close_notice_title"),
							Symbol: position.Symbol,
							Side: "buy",
							PositionSide: "short",
							Quantity: positionAmtFloat,
							Profit: unRealizedProfit,
							Remarks: lang.Lang("futures.target_profit"),
							Status: "fail",
							Error: err.Error(),
						})
					}
				}
				continue
			}
		}
		// 继续持有
		positionCount += 1
	}
	/*************************************************平仓 end************************************************************ */
	
	/*************************************************检查当前仓位数量 start************************************************************ */
	allMyCount := positionCount + len(allOpenOrders)
	if allMyCount >= max_count_int {
		logs.Info("当前仓位+挂单数量为%d, 已经达到当前仓位最大数量%d,暂时无法开启新的仓位", allMyCount, max_count_int)
		return
	}
	/*************************************************检查当前仓位数量  end************************************************************ */
	
	
	/*************************************************开仓(根据选币策略选中的币) start************************************************************ */
	if allow_long != "1" && allow_short != "1" {
		logs.Info("当前策略不允许开多和开空")
		return
	}
	
	isOpen := false
	
	for _, coin := range coins {
		positionSideLong := "LONG"
      	positionSideShort := "SHORT"
		symbol := coin.Symbol
		tickSize := coin.TickSize // 交易金额精度
		stepSize := coin.StepSize // 交易数量精度
		usdt_float64, _ := strconv.ParseFloat(coin.Usdt, 64) // 交易金额
		leverage_float64 := float64(coin.Leverage) // 合约倍数
		canLang, canShort := lineStrategy.GetCanLongOrShort(symbol)
		if !canLang && !canShort {
			logs.Info("%s:没有达到条件不可开仓", symbol)
			continue
		}
		var buyOrderLong *futures.Order // 此币种开多的单
		var buyOrderShort *futures.Order // 此币种开空的单
		for _, item := range allOpenOrders {
			if item.Symbol == symbol && item.Side == futures.SideTypeBuy && item.PositionSide == futures.PositionSideTypeLong {
				buyOrderLong = item
			}
			if item.Symbol == symbol && item.Side == futures.SideTypeSell && item.PositionSide == futures.PositionSideTypeShort {
				buyOrderShort = item
			}
			if buyOrderLong != nil && buyOrderShort != nil {
				break
			}
		}
		var positionLong *futures.PositionRisk // 此币种的多仓
		var positionShort *futures.PositionRisk // 此币种的空仓
		for _, item := range positions {
			positionAmtFloat, _ := strconv.ParseFloat(item.PositionAmt, 64)
			positionAmtFloatAbs := math.Abs(positionAmtFloat) // 空单为负数,纠正为绝对值
			if positionAmtFloatAbs < 0.00000000001 {// 没有持仓的
				continue
			}
			if item.Symbol == symbol && item.PositionSide == positionSideLong {
				positionLong = item
			}
			if item.Symbol == symbol && item.PositionSide == positionSideShort {
				positionShort = item
			}
			if positionLong != nil && positionShort != nil {
				break
			}
		}
		
		if allow_long == "1" && positionLong == nil && buyOrderLong == nil && canLang {
			
			buyPrice, _, err := binance.GetDepthAvgPrice(symbol) // 平均买价
			if err == nil {
				buyPrice = utils.GetTradePrecision(buyPrice, tickSize) // 合理精度的价格
				quantity := (usdt_float64 / buyPrice) * leverage_float64  // 购买数量
				quantity = utils.GetTradePrecision(quantity, stepSize) // 合理精度的价格
				
				UpdateSymbolTradeInfo(coin) // 更新倍率和仓位模式
				
				if order_type == "MARKET" {
					result, err := binance.BuyMarket(symbol, quantity, futures.PositionSideTypeLong)
					if err == nil {
						// 数据库写入订单
						insertOpenOrder(symbol, quantity, result.AvgPrice, "LONG")
						pusher.FuturesOpenOrder(notify.FuturesOrderParams{
							Title: lang.Lang("futures.open_notice_title"),
							Symbol: symbol,
							Side: "buy",
							PositionSide: "long",
							Price: buyPrice,
							Quantity: quantity,
							Status: "success",
						})
					} else {
						pusher.FuturesOpenOrder(notify.FuturesOrderParams{
							Title: lang.Lang("futures.open_notice_title"),
							Symbol: symbol,
							Side: "buy",
							PositionSide: "long",
							Price: buyPrice,
							Quantity: quantity,
							Status: "fail",
							Error: err.Error(),
						})
					}
				} else {
					result, err := binance.BuyLimit(symbol, quantity, buyPrice, futures.PositionSideTypeLong)
					if err == nil {
						// 数据库写入订单(可能没有买入)
						insertOpenOrder(symbol, quantity, result.AvgPrice, "LONG")
						pusher.FuturesOpenOrder(notify.FuturesOrderParams{
							Title: lang.Lang("futures.open_notice_title"),
							Symbol: symbol,
							Side: "buy",
							PositionSide: "long",
							Price: buyPrice,
							Quantity: quantity,
							Status: "success",
						})
					} else {
						pusher.FuturesOpenOrder(notify.FuturesOrderParams{
							Title: lang.Lang("futures.open_notice_title"),
							Symbol: symbol,
							Side: "buy",
							PositionSide: "long",
							Price: buyPrice,
							Quantity: quantity,
							Status: "fail",
							Error: err.Error(),
						})
					}
				}
				isOpen = true
			}
		}
		if allow_short == "1" && positionShort == nil && buyOrderShort == nil && canShort {
			
			_, sellPrice, err := binance.GetDepthAvgPrice(symbol) // 平均卖价
			if err == nil {
				sellPrice = utils.GetTradePrecision(sellPrice, tickSize) // 合理精度的价格
				quantity := (usdt_float64 / sellPrice) * leverage_float64  // 购买数量
				quantity = utils.GetTradePrecision(quantity, stepSize) // 合理精度的价格
				
				UpdateSymbolTradeInfo(coin) // 更新倍率和仓位模式
				
				if order_type == "MARKET" {
					result, err := binance.SellMarket(symbol, quantity, futures.PositionSideTypeShort)
					if err == nil {
						// 数据库写入订单
						insertOpenOrder(symbol, quantity, result.AvgPrice, "SHORT")
						pusher.FuturesOpenOrder(notify.FuturesOrderParams{
							Title: lang.Lang("futures.open_notice_title"),
							Symbol: symbol,
							Side: "sell",
							PositionSide: "short",
							Price: sellPrice,
							Quantity: quantity,
							Status: "success",
						})
					} else {
						pusher.FuturesOpenOrder(notify.FuturesOrderParams{
							Title: lang.Lang("futures.open_notice_title"),
							Symbol: symbol,
							Side: "sell",
							PositionSide: "short",
							Price: sellPrice,
							Quantity: quantity,
							Status: "fail",
							Error: err.Error(),
						})
					}
				} else {
					result, err := binance.SellLimit(symbol, quantity, sellPrice, futures.PositionSideTypeShort)
					if err == nil {
						// 数据库写入订单(可能没有买入)
						insertOpenOrder(symbol, quantity, result.AvgPrice, "SHORT")
						pusher.FuturesOpenOrder(notify.FuturesOrderParams{
							Title: lang.Lang("futures.open_notice_title"),
							Symbol: symbol,
							Side: "sell",
							PositionSide: "short",
							Price: sellPrice,
							Quantity: quantity,
							Status: "success",
						})
					} else {
						pusher.FuturesOpenOrder(notify.FuturesOrderParams{
							Title: lang.Lang("futures.open_notice_title"),
							Symbol: symbol,
							Side: "sell",
							PositionSide: "short",
							Price: sellPrice,
							Quantity: quantity,
							Status: "fail",
							Error: err.Error(),
						})
					}
				}
				isOpen = true
			}
		}
	}
	if isOpen {
		time.Sleep(30 * time.Second) // 如果开单了30 秒后再开启下一次
	}
	/*************************************************开仓 end************************************************************ */
}

func GetExcludeSymbolsMap() (map[string]bool) {
	exclude_symbols_map := make(map[string]bool)
	exclude_symbolsStr , _ := config.String("coin::exclude_symbols") // 排除自动交易的币
	exclude_symbols := strings.Split(exclude_symbolsStr, ",")
	for _, symbol := range exclude_symbols {
		exclude_symbols_map[symbol] = true
	}
	return exclude_symbols_map
}

// 获取所有交易的币
func GetAllSymbols() (symbols []*models.Symbols, err error) {
	o := orm.NewOrm()
	_, err = o.QueryTable("symbols").OrderBy("ID").All(&symbols)
	return symbols, err
}

// 挂单已经超过设置的超时时间，撤销挂单
func cancelTimeoutOrder(explodeSymbolsMap map[string]bool, allOpenOrders []*futures.Order) {
	nowTime := time.Now().Unix() * 1000 // 毫秒时间戳
	for _, buyOrder := range allOpenOrders {
		if _, exist := explodeSymbolsMap[buyOrder.Symbol]; exist {
			// 在白名单内
			continue
		}
		if nowTime < (buyOrder.UpdateTime + buy_timeout_int64 * 1000) {
			// 没有超过设置的超时
			continue
		}
		res, _ := binance.CancelOrder(buyOrder.Symbol, buyOrder.OrderID)
		logs.Info("撤销超时的开仓订单")
		logs.Info("response:", res)
	}
}

func insertOpenOrder(symbol string, quantity float64, avg_price string, positionSide string) {
	order := new(models.Order)
	order.Symbol = symbol
	order.Amount = strconv.FormatFloat(quantity, 'f', -1, 64)
	order.Avg_price = avg_price
	order.PositionSide = positionSide
	order.Inexact_profit = "0.0" // 预估收益
	order.Side = "open"
	order.UpdateTime = time.Now().Unix() * 1000 
	
	o := orm.NewOrm()
	o.Insert(order)
}

func insertCloseOrder(position *futures.PositionRisk, positionAmtFloat float64, unRealizedProfit float64, avg_price string) {
	// 数据库写入订单
	order := new(models.Order)
	order.Symbol = position.Symbol
	order.Amount = strconv.FormatFloat(positionAmtFloat, 'f', -1, 64)
	order.Avg_price = avg_price
	order.Inexact_profit = strconv.FormatFloat(unRealizedProfit, 'f', -1, 64)
	order.PositionSide = position.PositionSide
	order.Side = "close"
	order.UpdateTime = time.Now().Unix() * 1000 
	
	o := orm.NewOrm()
	o.Insert(order)
}

// 更新币种的交易精度和插入新币
func UpdateSymbolsTradePrecision() {
	res, err := binance.GetExchangeInfo()
	if err == nil {
		for _, symbol := range res.Symbols {
			o := orm.NewOrm()
			var tickSize string
			var stepSize string
			priceFilter := symbol.PriceFilter()
			if priceFilter != nil {
				tickSize = priceFilter.TickSize
			}
			lotSizeFilter := symbol.LotSizeFilter()
			if lotSizeFilter != nil {
				stepSize = lotSizeFilter.StepSize
			}
			o.QueryTable("symbols").Filter("symbol", symbol.Symbol).Update(orm.Params{
				"tickSize": tickSize,
				"stepSize": stepSize,
			})
			
			if strings.HasSuffix(symbol.Symbol, "USDT") { // 非usdt结尾的不需要
				if !o.QueryTable("symbols").Filter("symbol", symbol.Symbol).Exist() {
					// 没有的币种插入
					logs.Info("新增币种：", symbol.Symbol)
					o.Insert(&models.Symbols{
						Symbol: symbol.Symbol,
						Enable: 0, // 默认不开启
						Leverage: 3,
						MarginType: "CROSSED", // 杠杆类型 ISOLATED(逐仓), CROSSED(全仓)
						TickSize: tickSize,
						StepSize: stepSize,
						Usdt: "10",
						Profit: "20",
						Loss: "20",
						KlineInterval: "1d",
					})
				}
			}
		}
	}
}

// 更新币种信息
func UpdateSymbolTradeInfo(symbols *models.Symbols) {
	marginType := futures.MarginTypeIsolated
	if symbols.MarginType == "CROSSED" {
		marginType = futures.MarginTypeCrossed
	}
	binance.SetLeverage(symbols.Symbol, int(symbols.Leverage))  // 修改合约倍数
	binance.SetMarginType(symbols.Symbol, marginType) // 修改仓位模式
}

// 更新所有币种的资金费率信息
func UpdateSymbolsFundingRates() {
	res, err := binance.GetFundingRate(binance.FundingRateParams{})
	if err == nil {
		o := orm.NewOrm()
		for _, symbol := range res {
			// 非usdt结尾的不需要
			if strings.HasSuffix(symbol.Symbol, "USDT") {
				var fundingRate models.SymbolFundingRates
				o.QueryTable("symbol_funding_rates").Filter("symbol", symbol.Symbol).One(&fundingRate)
				if (fundingRate.Symbol == "") {
					// add
					o.Insert(&models.SymbolFundingRates{
						Symbol: symbol.Symbol,
						Enable: 1,
						NowFundingRate: symbol.LastFundingRate,
						NowFundingTime: symbol.Time,
						NowPrice: symbol.MarkPrice,
						LastNoticeFundingRate: "0.0",
						LastNoticeFundingTime: 0,
					})
				} else {
					// edit
					fundingRate.NowFundingRate = symbol.LastFundingRate
					fundingRate.NowFundingTime = symbol.Time
					fundingRate.NowPrice = symbol.MarkPrice
					orm.NewOrm().Update(&fundingRate)
				}
			}
		}
	}
}

// 获取币的策略
func GetCoinStrategy(name string) (coinStrategy strategy.CoinStrategy) {
	switch (name) {
		case "coin1":
			coinStrategy = coin.TradeCoin1{}
		case "coin2":
			coinStrategy = coin.TradeCoin2{}
		case "coin3":
			coinStrategy = coin.TradeCoin3{}
		case "coin4":
			coinStrategy = coin.TradeCoin4{}
		case "coin5":
			coinStrategy = coin.TradeCoin5{}
		default:
			coinStrategy = coin.TradeCoin1{}
	}
	return coinStrategy
}

// 获取交易策略
func GetLineStrategy(name string) (lineStrategy strategy.LineStrategy) {
	switch (name) {
		case "line0":
			lineStrategy = line.TradeLine0{}
		case "line1":
			lineStrategy = line.TradeLine1{}
		case "line2":
			lineStrategy = line.TradeLine2{}
		case "line3":
			lineStrategy = line.TradeLine3{}
		case "line4":
			lineStrategy = line.TradeLine4{}
		case "line5":
			lineStrategy = line.TradeLine5{}
		case "line6":
			lineStrategy = line.TradeLine6{}
		case "line7":
			lineStrategy = line.TradeLine7{}
		default:
			lineStrategy = line.TradeLine0{}
	}
	return lineStrategy
}


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
	"github.com/beego/beego/v2/core/logs"
)

var pusher = notify.GetNotifyChannel()

var flagFutures = 0
func StartTrade() {
	systemConfig, err := utils.GetSystemConfig()
	if err != nil {
		logs.Error("GetSystemConfig:", err)
		return
	}
	if (systemConfig.FutureEnable == 1) {
		if (flagFutures == 0) {
			logs.Info("futures trade bot start")
			flagFutures = 1
		}
	} else {
		if (flagFutures == 1) {
			logs.Info("futures trade bot stop")
			flagFutures = 0
		}
		return
	}
	
	globalCoinStrategy := GetCoinStrategy(systemConfig.FutureStrategyCoin) // 交易策略
	globalLineStrategy := GetLineStrategy(systemConfig.FutureStrategyTrade) // 选币策略
	
	/************************************************寻找交易币种 start******************************************************************* */
	allCoins, err := GetAllSymbols()
	if err != nil {
		logs.Error("GetAllSymbols err:", err)
		return
	}
	coins := globalCoinStrategy.SelectCoins(allCoins)
	if coins == nil {
		logs.Error("coins SelectCoins is nil")
		return
	}
	/************************************************寻找交易币种 end******************************************************************* */
	
	
	/************************************************获取账户信息 start******************************************************************* */
	positions, err := binance.GetPosition(binance.PositionParams{})
	if err != nil {
		logs.Error("GetPosition err:", err)
		return
	}
	
	allOpenOrders, err := binance.GetOpenOrder()
	if err != nil {
		logs.Error("GetOpenOrder err:", err)
	}
	/************************************************获取账户信息 end******************************************************************* */
	
	
	/*************************************************挂单已经超过设置的超时时间，撤销挂单 start************************************************************ */
	exclude_symbols_map := GetExcludeSymbolsMap(systemConfig.FutureExcludeSymbols)
	cancelTimeoutOrder(exclude_symbols_map, allOpenOrders, int64(systemConfig.FutureBuyTimeout))
	/*************************************************挂单已经超过设置的超时时间，撤销挂单 end************************************************************ */
	
	
	/*************************************************平仓(止盈或止损)已经有持仓的币(排除手动交易白名单) start************************************************************ */
	positionCount := 0 // 当前仓位数量
	for _, position := range positions {
		positionAmtFloat, _ := strconv.ParseFloat(position.PositionAmt, 64)
		positionAmtFloatAbs := math.Abs(positionAmtFloat) // 空单为负数,纠正为绝对值
		if positionAmtFloatAbs < 0.0000000001 {// 没有持仓的
			continue
		}
		
		coin_profit_float64 := 10000.0 // 全局定义
		coin_loss_float64 := 10000.0 // 全局定义
		coin_line_strategy := globalLineStrategy // 默认为全局
		var findCoin *models.Symbols
		for _, coin := range allCoins {
			if coin.Symbol == position.Symbol {
				findCoin = coin
				// 自定义可以覆盖全局
				if coin.Profit != "0" {
					coin_profit_float64, _ = strconv.ParseFloat(coin.Profit, 64)
				}
				if coin.Loss != "0" {
					coin_loss_float64, _ = strconv.ParseFloat(coin.Loss, 64)
				}
				if coin.StrategyType != "global" {
					// 独立的策略
					coin_line_strategy = GetLineStrategy(coin.StrategyType)
				}
				break
			}
		}
		
		unRealizedProfit, _ := strconv.ParseFloat(position.UnRealizedProfit, 64)
		leverage_float64, _ := strconv.ParseFloat(position.Leverage, 64)
		markPrice_float64, _ := strconv.ParseFloat(position.MarkPrice, 64)
		nowProfit := (unRealizedProfit / (positionAmtFloatAbs * markPrice_float64)) * leverage_float64 * 100 // 当前收益率(正为盈利，负为亏损)
		
		if _, exist := exclude_symbols_map[position.Symbol]; exist { // 在白名单内
			continue
		}
		closeResult := coin_line_strategy.AutoStopOrder(strategy.CloseParams{
			Symbols: findCoin,
			Position: position,
			NowProfit: nowProfit,
		})
		if closeResult.Complete { // 触发策略,风向改变,强制平仓
			logs.Info("%s:auto_stop_start", position.Symbol)
			if position.PositionSide == "LONG" {
				order, err := binance.SellMarket(position.Symbol, positionAmtFloatAbs, futures.PositionSideTypeLong)
				if err == nil {
					// 数据库写入订单
					insertCloseOrder(position, positionAmtFloatAbs, unRealizedProfit, position.MarkPrice, order.OrderID)
					
					markPrice, _ := strconv.ParseFloat(position.MarkPrice, 64)
					pusher.FuturesCloseOrder(notify.FuturesOrderParams{
						Title: lang.Lang("futures.close_notice_title"),
						Symbol: position.Symbol,
						Side: "sell",
						PositionSide: "long",
						Price: markPrice,
						Quantity: positionAmtFloat,
						Leverage: leverage_float64,
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
						Leverage: leverage_float64,
						Profit: unRealizedProfit,
						Remarks: lang.Lang("futures.wind_of_change"),
						Status: "fail",
						Error: err.Error(),
					})
				}
			}
			if position.PositionSide == "SHORT" {
				order, err := binance.BuyMarket(position.Symbol, positionAmtFloatAbs, futures.PositionSideTypeShort)
				if err == nil {
					// 数据库写入订单
					insertCloseOrder(position, positionAmtFloatAbs, unRealizedProfit, position.MarkPrice, order.OrderID)
					
					markPrice, _ := strconv.ParseFloat(position.MarkPrice, 64)
					pusher.FuturesCloseOrder(notify.FuturesOrderParams{
						Title: lang.Lang("futures.close_notice_title"),
						Symbol: position.Symbol,
						Side: "buy",
						PositionSide: "short",
						Price: markPrice,
						Quantity: positionAmtFloat,
						Leverage: leverage_float64,
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
						Leverage: leverage_float64,
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
			// position.Symbol, position.PositionSide
			closeResult := coin_line_strategy.CanOrderComplete(strategy.CloseParams{
				Symbols: findCoin,
				Position: position,
				NowProfit: nowProfit,
			})
			if closeResult.Complete { // 
				if position.PositionSide == "LONG" {
					order, err := binance.SellMarket(position.Symbol, positionAmtFloatAbs, futures.PositionSideTypeLong)
					if err == nil {
						// 数据库写入订单
						insertCloseOrder(position, positionAmtFloatAbs, unRealizedProfit, position.MarkPrice, order.OrderID)
						
						markPrice, _ := strconv.ParseFloat(position.MarkPrice, 64)
						pusher.FuturesCloseOrder(notify.FuturesOrderParams{
							Title: lang.Lang("futures.close_notice_title"),
							Symbol: position.Symbol,
							Side: "sell",
							PositionSide: "long",
							Price: markPrice,
							Quantity: positionAmtFloat,
							Leverage: leverage_float64,
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
							Leverage: leverage_float64,
							Profit: unRealizedProfit,
							Remarks: lang.Lang("futures.stop_loss"),
							Status: "fail",
							Error: err.Error(),
						})
					}
				}
				if position.PositionSide == "SHORT" {
					order, err := binance.BuyMarket(position.Symbol, positionAmtFloatAbs, futures.PositionSideTypeShort)
					if err == nil {
						// 数据库写入订单
						insertCloseOrder(position, positionAmtFloatAbs, unRealizedProfit, position.MarkPrice, order.OrderID)
						
						markPrice, _ := strconv.ParseFloat(position.MarkPrice, 64)
						pusher.FuturesCloseOrder(notify.FuturesOrderParams{
							Title: lang.Lang("futures.close_notice_title"),
							Symbol: position.Symbol,
							Side: "buy",
							PositionSide: "short",
							Price: markPrice,
							Quantity: positionAmtFloat,
							Leverage: leverage_float64,
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
							Leverage: leverage_float64,
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
			closeResult := coin_line_strategy.CanOrderComplete(strategy.CloseParams{
				Symbols: findCoin,
				Position: position,
				NowProfit: nowProfit,
			})
			if closeResult.Complete {
				if position.PositionSide == "LONG" {
					order, err := binance.SellMarket(position.Symbol, positionAmtFloatAbs, futures.PositionSideTypeLong)
					if err == nil {
						// 数据库写入订单
						insertCloseOrder(position, positionAmtFloatAbs, unRealizedProfit, position.MarkPrice, order.OrderID)
						
						markPrice, _ := strconv.ParseFloat(position.MarkPrice, 64)
						pusher.FuturesCloseOrder(notify.FuturesOrderParams{
							Title: lang.Lang("futures.close_notice_title"),
							Symbol: position.Symbol,
							Side: "sell",
							PositionSide: "long",
							Price: markPrice,
							Quantity: positionAmtFloat,
							Leverage: leverage_float64,
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
							Leverage: leverage_float64,
							Profit: unRealizedProfit,
							Remarks: lang.Lang("futures.target_profit"),
							Status: "fail",
							Error: err.Error(),
						})
					}
				}
				if position.PositionSide == "SHORT" {
					order, err := binance.BuyMarket(position.Symbol, positionAmtFloatAbs, futures.PositionSideTypeShort)
					if err == nil {
						// 数据库写入订单
						insertCloseOrder(position, positionAmtFloatAbs, unRealizedProfit, position.MarkPrice, order.OrderID)
						
						markPrice, _ := strconv.ParseFloat(position.MarkPrice, 64)
						pusher.FuturesCloseOrder(notify.FuturesOrderParams{
							Title: lang.Lang("futures.close_notice_title"),
							Symbol: position.Symbol,
							Side: "buy",
							PositionSide: "short",
							Price: markPrice,
							Quantity: positionAmtFloat,
							Leverage: leverage_float64,
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
							Leverage: leverage_float64,
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
	if allMyCount >= systemConfig.FutureMaxCount {
		logs.Info("position+open order: %d, is over max %d, stop open new order", allMyCount, systemConfig.FutureMaxCount)
		return
	}
	/*************************************************检查当前仓位数量  end************************************************************ */
	
	
	/*************************************************开仓(根据选币策略选中的币) start************************************************************ */
	if systemConfig.FutureAllowLong != 1 && systemConfig.FutureAllowShort != 1 {
		logs.Info("the base config don't allow long and all short")
		return
	}
	
	isOpen := false
	
	for _, coin := range coins {
		if _, exist := exclude_symbols_map[coin.Symbol]; exist { // 在白名单内
			continue
		}
		positionSideLong := "LONG"
      	positionSideShort := "SHORT"
		symbol := coin.Symbol
		tickSize := coin.TickSize // 交易金额精度
		stepSize := coin.StepSize // 交易数量精度
		usdt_float64, _ := strconv.ParseFloat(coin.Usdt, 64) // 交易金额
		leverage_float64 := float64(coin.Leverage) // 合约倍数
		coin_line_strategy := globalLineStrategy // 默认为全局
		if coin.StrategyType != "global" {
			// 独立的策略
			coin_line_strategy = GetLineStrategy(coin.StrategyType)
		}
		openResult := coin_line_strategy.GetCanLongOrShort(strategy.OpenParams{
			Symbols: coin,
		})
		if !openResult.CanLong && !openResult.CanShort {
			logs.Info("%s:not meeting the order conditions", symbol)
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
		
		if systemConfig.FutureAllowLong == 1 && positionLong == nil && buyOrderLong == nil && openResult.CanLong {
			buyPrice, _, err := binance.GetDepthAvgPrice(symbol) // 平均买价
			if err == nil {
				buyPrice = utils.GetTradePrecision(buyPrice, tickSize) // 合理精度的价格
				quantity := (usdt_float64 / buyPrice) * leverage_float64  // 购买数量
				quantity = utils.GetTradePrecision(quantity, stepSize) // 合理精度的价格
				
				UpdateSymbolTradeInfo(coin) // 更新倍率和仓位模式
				
				if systemConfig.FutureOrderType == "MARKET" {
					order, err := binance.BuyMarket(symbol, quantity, futures.PositionSideTypeLong)
					if err == nil {
						// 数据库写入订单
						insertOpenOrder(symbol, quantity, strconv.FormatFloat(buyPrice, 'f', -1, 64), "LONG", order.OrderID)
						pusher.FuturesOpenOrder(notify.FuturesOrderParams{
							Title: lang.Lang("futures.open_notice_title"),
							Symbol: symbol,
							Side: "buy",
							PositionSide: "long",
							Price: buyPrice,
							Quantity: quantity,
							Leverage: leverage_float64,
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
							Leverage: leverage_float64,
							Status: "fail",
							Error: err.Error(),
						})
					}
				} else {
					order, err := binance.BuyLimit(symbol, quantity, buyPrice, futures.PositionSideTypeLong)
					if err == nil {
						// 数据库写入订单(可能没有买入)
						insertOpenOrder(symbol, quantity, strconv.FormatFloat(buyPrice, 'f', -1, 64), "LONG", order.OrderID)
						pusher.FuturesOpenOrder(notify.FuturesOrderParams{
							Title: lang.Lang("futures.open_notice_title"),
							Symbol: symbol,
							Side: "buy",
							PositionSide: "long",
							Price: buyPrice,
							Quantity: quantity,
							Leverage: leverage_float64,
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
							Leverage: leverage_float64,
							Status: "fail",
							Error: err.Error(),
						})
					}
				}
				isOpen = true
			}
		}
		if systemConfig.FutureAllowShort == 1 && positionShort == nil && buyOrderShort == nil && openResult.CanShort {
			
			_, sellPrice, err := binance.GetDepthAvgPrice(symbol) // 平均卖价
			if err == nil {
				sellPrice = utils.GetTradePrecision(sellPrice, tickSize) // 合理精度的价格
				quantity := (usdt_float64 / sellPrice) * leverage_float64  // 购买数量
				quantity = utils.GetTradePrecision(quantity, stepSize) // 合理精度的价格
				
				UpdateSymbolTradeInfo(coin) // 更新倍率和仓位模式
				
				if systemConfig.FutureOrderType == "MARKET" {
					order, err := binance.SellMarket(symbol, quantity, futures.PositionSideTypeShort)
					if err == nil {
						// 数据库写入订单
						insertOpenOrder(symbol, quantity, strconv.FormatFloat(sellPrice, 'f', -1, 64), "SHORT", order.OrderID)
						pusher.FuturesOpenOrder(notify.FuturesOrderParams{
							Title: lang.Lang("futures.open_notice_title"),
							Symbol: symbol,
							Side: "sell",
							PositionSide: "short",
							Price: sellPrice,
							Quantity: quantity,
							Leverage: leverage_float64,
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
							Leverage: leverage_float64,
							Status: "fail",
							Error: err.Error(),
						})
					}
				} else {
					order, err := binance.SellLimit(symbol, quantity, sellPrice, futures.PositionSideTypeShort)
					if err == nil {
						// 数据库写入订单(可能没有买入)
						insertOpenOrder(symbol, quantity, strconv.FormatFloat(sellPrice, 'f', -1, 64), "SHORT", order.OrderID)
						pusher.FuturesOpenOrder(notify.FuturesOrderParams{
							Title: lang.Lang("futures.open_notice_title"),
							Symbol: symbol,
							Side: "sell",
							PositionSide: "short",
							Price: sellPrice,
							Quantity: quantity,
							Leverage: leverage_float64,
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
							Leverage: leverage_float64,
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

// 排除自动交易的币
func GetExcludeSymbolsMap(exclude_symbols_str string) (map[string]bool) {
	exclude_symbols_map := make(map[string]bool)
	exclude_symbols := strings.Split(exclude_symbols_str, ",")
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
func cancelTimeoutOrder(explodeSymbolsMap map[string]bool, allOpenOrders []*futures.Order, buy_timeout int64) {
	nowTime := time.Now().Unix() * 1000 // 毫秒时间戳
	for _, buyOrder := range allOpenOrders {
		if _, exist := explodeSymbolsMap[buyOrder.Symbol]; exist {
			// 在白名单内
			continue
		}
		if nowTime < (buyOrder.UpdateTime + buy_timeout * 1000) {
			// 没有超过设置的超时
			continue
		}
		res, _ := binance.CancelOrder(buyOrder.Symbol, buyOrder.OrderID)
		logs.Info("Revoke overdue warehouse opening orders")
		logs.Info("response:", res)
	}
}

func insertOpenOrder(symbol string, quantity float64, avg_price string, positionSide string, orderId int64) {
	order := new(models.Order)
	order.Symbol = symbol
	order.Amount = strconv.FormatFloat(quantity, 'f', -1, 64)
	order.Avg_price = avg_price
	order.PositionSide = positionSide
	order.Inexact_profit = "0.0" // 预估收益
	order.Side = "open"
	order.OrderId = orderId
	order.UpdateTime = time.Now().Unix() * 1000 
	
	o := orm.NewOrm()
	o.Insert(order)
}

func insertCloseOrder(position *futures.PositionRisk, positionAmtFloat float64, unRealizedProfit float64, avg_price string, orderId int64) {
	// 数据库写入订单
	order := new(models.Order)
	order.Symbol = position.Symbol
	order.Amount = strconv.FormatFloat(positionAmtFloat, 'f', -1, 64)
	order.Avg_price = avg_price
	order.Inexact_profit = strconv.FormatFloat(unRealizedProfit, 'f', -1, 64)
	order.PositionSide = position.PositionSide
	order.Side = "close"
	order.OrderId = orderId
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
					logs.Info("add new futures symbol", symbol.Symbol)
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
						StrategyType: "global",
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
	systemConfig, _ := utils.GetSystemConfig()
	if (systemConfig.ListenFundingRateEnable == 0) {
		return
	}
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
		case "coin6":
			coinStrategy = coin.TradeCoin6{}
		default:
			coinStrategy = coin.TradeCoin1{}
	}
	return coinStrategy
}

// 获取交易策略
func GetLineStrategy(name string) (lineStrategy strategy.LineStrategy) {
	switch (name) {
		case "custom":
			lineStrategy = line.TradeLineCustom{}
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
			lineStrategy = line.TradeLine1{}
	}
	return lineStrategy
}


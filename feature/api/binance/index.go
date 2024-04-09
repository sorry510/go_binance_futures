package binance

import (
	"context"
	"go_binance_futures/utils"
	"sort"
	"strconv"

	"github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/adapter/logs"
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/config"
)


var api_key, _ = config.String("binance::api_key")
var api_secret, _ = config.String("binance::api_secret")

// var client = binance.NewClient(api_key, api_secret)
var futuresClient = binance.NewFuturesClient(api_key, api_secret)


type ListOrderParams struct {
	Symbol    string
	OrderID   int64
	StartTime int64
	EndTime   int64
	Limit     int
}

// @returns /doc/futuresAccount.js
func GetFuturesAccount() (res *futures.Account, err error) {
	res, err = futuresClient.NewGetAccountService().Do(context.Background())
	if err != nil {
		logs.Error(err)
		return nil, err
	}
	logs.Info(utils.ToJson(res))
	return res, err
}

// @returns /doc/position.js
func GetPosition() (res []*futures.PositionRisk, err error){
	res, err = futuresClient.NewGetPositionRiskService().Do(context.Background())
	if err != nil {
		logs.Error(err)
		return nil, err
	}
	// logs.Info(utils.ToJson(res))
	return res, err
}

func GetDepth(symbol string, limits ...int) (res *futures.DepthResponse, err error) {
	limit := 100 // 默认值
    if len(limits) != 0 {
        limit = limits[0]
    }
	res, err = futuresClient.NewDepthService().Symbol(symbol).Limit(limit).Do(context.Background())
	if err != nil {
		logs.Error(err)
		return nil, err
	}
	return res, err
}

func GetDepthAvgPrice(symbol string, limits ...int) (buyPrice float64, sellPrice float64, err error) {
	limit := 50 // 默认值
    if len(limits) != 0 {
        limit = limits[0]
    }
	res, err := futuresClient.NewDepthService().Symbol(symbol).Limit(limit).Do(context.Background())
	if err != nil {
		logs.Error(err)
		return 0.0, 0.0, err
	}
	buyPrice, sellPrice = avgPrice(res)
	// logs.Info(utils.ToJson(res))
	// logs.Info(buyPrice, sellPrice)
	return buyPrice, sellPrice, err
}

func avgPrice(data *futures.DepthResponse) (buyPrice float64, sellPrice float64) {
	var buyNum float64
	var sellNum float64
	for _, item := range data.Bids {
		price1, _ := strconv.ParseFloat(item.Price, 64)
		quantity1, _ := strconv.ParseFloat(item.Quantity, 64)
		buyPrice += price1 * quantity1
		buyNum += quantity1
	}
	for _, item := range data.Asks {
		price2, _ := strconv.ParseFloat(item.Price, 64)
		quantity2, _ := strconv.ParseFloat(item.Quantity, 64)
		sellPrice += price2 * quantity2
		sellNum += quantity2
	}
	if len(data.Bids) == 0 || len(data.Asks) == 0 {
		return 0.0, 0.0
	}
	return buyPrice / buyNum, sellPrice / sellNum
}

// @param symbol 交易对名称，例如：BTCUSDT
// @param interval K线的时间间隔，例如：1m, 3m, 5m, 15m, 30m, 1h等
// @param limit 返回的K线数据条数
// @returns /doc/kine.js
func GetKlineData(symbol string, interval string, limit int) (klines []*futures.Kline, err error) {
	klines, err = futuresClient.NewKlinesService().Symbol(symbol).Interval(interval).Limit(limit).Do(context.Background())
	if err != nil {
		logs.Error(err)
		return nil, err
	}
	sort.Slice(klines, func(i, j int) bool {
		return klines[i].OpenTime > klines[j].OpenTime // 按照时间降序()
	})
	// logs.Info(utils.ToJson(klines))
	return klines, err
}

// 限价买入
// @see https://binance-docs.github.io/apidocs/futures/cn/#trade-3
// @returns /doc/order.js
func BuyLimit(symbol string, quantity float64, price float64, positionSide futures.PositionSideType) (order *futures.CreateOrderResponse, err error) {
	order, err = futuresClient.NewCreateOrderService().
		Symbol(symbol).
		Side(futures.SideTypeBuy).
		PositionSide(positionSide).
		Type(futures.OrderTypeLimit).
		TimeInForce(futures.TimeInForceTypeGTC).
		Quantity(strconv.FormatFloat(quantity, 'f', -1, 64)).
		Price(strconv.FormatFloat(price, 'f', -1, 64)).
		Do(context.Background())
	if err != nil {
		logs.Error(err)
		return nil, err
	}
		
	return order, err
}

// 限价卖出
// @see https://binance-docs.github.io/apidocs/futures/cn/#trade-3
// @returns /doc/order.js
func SellLimit(symbol string, quantity float64, price float64, positionSide futures.PositionSideType) (order *futures.CreateOrderResponse, err error) {
	order, err = futuresClient.NewCreateOrderService().
		Symbol(symbol).
		Side(futures.SideTypeSell).
		PositionSide(positionSide).
		Type(futures.OrderTypeLimit).
		TimeInForce(futures.TimeInForceTypeGTC).
		Quantity(strconv.FormatFloat(quantity, 'f', -1, 64)).
		Price(strconv.FormatFloat(price, 'f', -1, 64)).
		Do(context.Background())
	if err != nil {
		logs.Error(err)
		return nil, err
	}
		
	return order, err
}

// 市价买入
// @see https://binance-docs.github.io/apidocs/futures/cn/#trade-3
// @returns /doc/order.js
func BuyMarket(symbol string, quantity float64, positionSide futures.PositionSideType) (order *futures.CreateOrderResponse, err error) {
	order, err = futuresClient.NewCreateOrderService().
		Symbol(symbol).
		Side(futures.SideTypeBuy).
		PositionSide(positionSide).
		Type(futures.OrderTypeMarket).
		TimeInForce(futures.TimeInForceTypeGTC).
		Quantity(strconv.FormatFloat(quantity, 'f', -1, 64)).
		Do(context.Background())
	if err != nil {
		logs.Error(err)
		return nil, err
	}
		
	return order, err
}

// 市价卖出
// @see https://binance-docs.github.io/apidocs/futures/cn/#trade-3
// @returns /doc/order.js
func SellMarket(symbol string, quantity float64, positionSide futures.PositionSideType) (order *futures.CreateOrderResponse, err error) {
	order, err = futuresClient.NewCreateOrderService().
		Symbol(symbol).
		Side(futures.SideTypeSell).
		PositionSide(positionSide).
		Type(futures.OrderTypeMarket).
		TimeInForce(futures.TimeInForceTypeGTC).
		Quantity(strconv.FormatFloat(quantity, 'f', -1, 64)).
		Do(context.Background())
	if err != nil {
		logs.Error(err)
		return nil, err
	}
		
	return order, err
}

// 撤销订单
// @see https://binance-docs.github.io/apidocs/futures/cn/#trade-6
func CancelOrder(symbol string, orderId int64) (res *futures.CancelOrderResponse, err error){
	res, err = futuresClient.NewCancelOrderService().Symbol(symbol).OrderID(orderId).Do(context.Background())
	if err != nil {
		logs.Error(err)
		return nil, err
	}
	return res, err
}

// 交易合约倍数
// @param string symbol
// @param Number 1-125
// @see https://binance-docs.github.io/apidocs/futures/cn/#trade-10
func SetLeverage(symbol string, leverage int) (res *futures.SymbolLeverage, err error) {
	res, err = futuresClient.NewChangeLeverageService().Symbol(symbol).Leverage(leverage).Do(context.Background())
	if err != nil {
		logs.Error(err)
		return nil, err
	}
	return res, err
}

// 合约模式 全仓与 逐仓
// @param string symbol
// @param futures.MarginType Isolated(逐仓), Crossed(全仓)
func SetMarginType(symbol string, marginType futures.MarginType) (err error) {
	err = futuresClient.NewChangeMarginTypeService().Symbol(symbol).MarginType(marginType).Do(context.Background())
	if err != nil {
		logs.Error(err)
		return err
	}
	return err
}

// 获取历史订单
// @see https://binance-docs.github.io/apidocs/futures/cn/#user_data-7
func GetOrders(listOrderParams ListOrderParams) (res []*futures.Order, err error) {
	service := futuresClient.NewListOrdersService()
	if listOrderParams.Symbol != "" {
		service = service.Symbol(listOrderParams.Symbol)
	}
	if listOrderParams.OrderID != 0 {
		service = service.OrderID(listOrderParams.OrderID)
	}
	if listOrderParams.StartTime != 0 {
		service = service.StartTime(listOrderParams.StartTime)
	}
	if listOrderParams.EndTime != 0 {
		service = service.EndTime(listOrderParams.EndTime)
	}
	if listOrderParams.Limit != 0 {
		service = service.Limit(listOrderParams.Limit)
	}
	res, err = service.Do(context.Background())
	if err != nil {
		logs.Error(err)
		return nil, err
	}
	return res, err
}

// @param startTime ex:(time.Now().Unix() - 60 * 60 * 24) // 获取最近1天的交易订单
// @see https://binance-docs.github.io/apidocs/futures/cn/#user_data-7
func GetLimitStartTimeOrders(startTime int64) (res []*futures.Order, err error) {
	service := futuresClient.NewListOrdersService()
	res, err = service.StartTime(startTime).Do(context.Background())
	if err != nil {
		logs.Error(err)
		return nil, err
	}
	return res, err
}

// 查看当前全部挂单
// @see https://binance-docs.github.io/apidocs/futures/cn/#user_data-5
func GetOpenOrder(symbols ...string) (res []*futures.Order, err error) {
	service := futuresClient.NewListOpenOrdersService()
	if len(symbols) > 0 {
		service = service.Symbol(symbols[0])
	}
	res, err = service.Do(context.Background())
	if err != nil {
		logs.Error(err)
		return nil, err
	}
	return res, err
}

// 获取交易规则和交易对
// @see https://binance-docs.github.io/apidocs/futures/cn/#0f3f2d5ee7
func GetExchangeInfo()(res *futures.ExchangeInfo, err error) {
	res, err = futuresClient.NewExchangeInfoService().Do(context.Background())
	if err != nil {
		logs.Error(err)
		return nil, err
	}
	// logs.Info(utils.ToJson(res))
	return res, err
}

// websocket 订阅全市场最新价格变化，只有币价格变化才会推送
func UpdateCoinByWs() {
	var lock = false
	var o = orm.NewOrm()
	doneC, _, err := futures.WsAllMarketTickerServe(func(event futures.WsAllMarketTickerEvent) {
		if !lock {
			lock = true
			for _, ticker := range event {
				o.Raw(
					"UPDATE `symbols` set `percentChange` = ?, `close` = ?, `open` = ?, `low` = ?, `updateTime` = ?, `lastClose` = low, `lastUpdateTime` = updateTime WHERE `symbol` = ?",
					ticker.PriceChangePercent,
					ticker.ClosePrice,
					ticker.OpenPrice,
					ticker.LowPrice,
					ticker.Time,
					
					ticker.Symbol,
				).Exec()
			}
			lock = false
		}
	}, func(err error) {
		logs.Info("ws run error:", err)
	})
	if err != nil {
		logs.Info("ws start error:", err)
		return
	}
	<-doneC
}


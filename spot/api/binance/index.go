package binance

import (
	"context"
	"go_binance_futures/utils"
	"sort"
	"strconv"

	"github.com/adshao/go-binance/v2"
	"github.com/beego/beego/v2/adapter/logs"
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/config"
)


var api_key, _ = config.String("binance::api_key")
var api_secret, _ = config.String("binance::api_secret")
var proxy_url, _ = config.String("binance::proxy_url")

var client *binance.Client

func init() {
	if proxy_url == "" {
		client = binance.NewClient(api_key, api_secret)
	} else {
		client = binance.NewProxiedClient(api_key, api_secret, proxy_url)
	}
}

func GetFuturesAccount() (res *binance.Account, err error) {
	res, err = client.NewGetAccountService().Do(context.Background())
	if err != nil {
		logs.Error(err)
		return nil, err
	}
	logs.Info(utils.ToJson(res))
	return res, err
}

// @see https://binance-docs.github.io/apidocs/futures/cn/#0f3f2d5ee7
func GetExchangeInfo(symbols ...string)(res *binance.ExchangeInfo, err error) {
	res, err = client.NewExchangeInfoService().Symbols(symbols...).Do(context.Background())
	if err != nil {
		logs.Error(err)
		return nil, err
	}
	// logs.Info(utils.ToJson(res))
	return res, err
}

// @param symbol 交易对名称，例如：BTCUSDT
// @param interval K线的时间间隔，例如：1m, 3m, 5m, 15m, 30m, 1h等
// @param limit 返回的K线数据条数
// @returns /doc/kine.js
func GetKlineData(symbol string, interval string, limit int) (klines []*binance.Kline, err error) {
	klines, err = client.NewKlinesService().Symbol(symbol).Interval(interval).Limit(limit).Do(context.Background())
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

// 获取交易价格
func GetTickerPrice(symbol string) (res []*binance.SymbolPrice, err error) {
	res, err = client.NewListPricesService().Symbol(symbol).Do(context.Background())
	if err != nil {
		logs.Error(err)
		return nil, err
	}
	return res, err
}

func BuyMarket(symbol string, quantity float64) (res *binance.CreateOrderResponse, err error) {
	res, err = client.NewCreateOrderService().
		Symbol(symbol).
        Side(binance.SideTypeBuy).
		Type(binance.OrderTypeMarket).
		Quantity(strconv.FormatFloat(quantity, 'f', -1, 64)).
		Do(context.Background())
	if err != nil {
		logs.Error(err)
		return
	}
	return res, err
}

func SellMarket(symbol string, quantity float64) (res *binance.CreateOrderResponse, err error) {
	res, err = client.NewCreateOrderService().
		Symbol(symbol).
        Side(binance.SideTypeSell).
		Type(binance.OrderTypeMarket).
		Quantity(strconv.FormatFloat(quantity, 'f', -1, 64)).
		Do(context.Background())
	if err != nil {
		logs.Error(err)
		return
	}
	return res, err
}

// 挂止盈单(现货不支持)
// @see https://binance-docs.github.io/apidocs/spot/cn/#trade-3
// @returns /doc/order.js
func OrderTakeProfit(symbol string, quantity float64, stopPrice float64) (order *binance.CreateOrderResponse, err error) {
	order, err = client.NewCreateOrderService().
		Symbol(symbol).
		Side(binance.SideTypeSell). // 止盈单是卖出
		Type(binance.OrderTypeTakeProfit). // 类型是止盈
		StopPrice(strconv.FormatFloat(stopPrice, 'f', -1, 64)). // 当触发stopPrice时，STOP_LOSS和TAKE_PROFIT将执行MARKET订单。
		Quantity(strconv.FormatFloat(quantity, 'f', -1, 64)).
		// TimeInForce(binance.TimeInForceTypeGTC).
		Do(context.Background())
	if err != nil {
		logs.Error(err)
		return nil, err
	}
		
	return order, err
}

// 挂单止损(现货不支持)
// @see https://binance-docs.github.io/apidocs/spot/cn/#trade-3
// @returns /doc/order.js
func OrderStopLoss(symbol string, quantity float64, stopPrice float64) (order *binance.CreateOrderResponse, err error) {
	order, err = client.NewCreateOrderService().
		Symbol(symbol).
		Side(binance.SideTypeSell). // 止损单是卖出
		Type(binance.OrderTypeStopLoss). // 类型是止损
		StopPrice(strconv.FormatFloat(stopPrice, 'f', -1, 64)). // 当触发stopPrice时，STOP_LOSS和TAKE_PROFIT将执行MARKET订单。
		Quantity(strconv.FormatFloat(quantity, 'f', -1, 64)).
		// TimeInForce(binance.TimeInForceTypeGTC).
		Do(context.Background())
	if err != nil {
		logs.Error(err)
		return nil, err
	}
		
	return order, err
}

type OrderParams struct {
	Symbol    string
	OrderID   int64
}

type ListOrderParams struct {
	OrderParams
	StartTime int64
	EndTime   int64
	Limit     int
}

// 获取历史订单
// @see https://binance-docs.github.io/apidocs/spot/cn/#user_data-36
func GetOrders(listOrderParams ListOrderParams) (res []*binance.Order, err error) {
	service := client.NewListOrdersService()
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

// 获取某个订单
func GetOrder(orderParams OrderParams) (res *binance.Order, err error) {
	service := client.NewGetOrderService()
	if orderParams.Symbol != "" {
		service = service.Symbol(orderParams.Symbol)
	}
	if orderParams.OrderID != 0 {
		service = service.OrderID(orderParams.OrderID)
	}
	res, err = service.Do(context.Background())
	if err != nil {
		logs.Error(err)
		return nil, err
	}
	return res, err
}

// websocket 订阅全市场最新价格变化，只有币价格变化才会推送(24小时变化)
// @doc https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/web-socket-streams#%E6%8C%89symbol%E7%9A%84%E5%AE%8C%E6%95%B4ticker
func UpdateCoinByWs() {
	binance.BaseWsMainURL = "wss://testnet.binance.vision/ws"
	var lock = false
	var o = orm.NewOrm()
	doneC, _, err := binance.WsAllMarketsStatServe(func(event binance.WsAllMarketsStatEvent) {
		if !lock {
			lock = true
			for _, ticker := range event {
				o.Raw(
					"UPDATE `spot_symbols` set `percentChange` = ?, `close` = ?, `open` = ?, `low` = ?, `high` = ?, `updateTime` = ?, `baseVolume` = ?, `quoteVolume` = ?, `closeQty` = ?,  `tradeCount` = ?, `lastClose` = close, `lastUpdateTime` = updateTime WHERE `symbol` = ?",
					ticker.PriceChangePercent,
					ticker.LastPrice, // 当前价格
					ticker.OpenPrice,
					ticker.LowPrice,
					ticker.HighPrice,
					ticker.Time,
					ticker.BaseVolume, // 成交量
					ticker.QuoteVolume, // 成交额
					ticker.CloseQty, // 最新成交价格上的成交量
					ticker.Count, // 成交数
					
					ticker.Symbol,
				).Exec()
			}
			lock = false
		}
	}, func(err error) {
		logs.Error("spot ws run error:", err.Error())
	})
	if err != nil {
		logs.Error("spot ws start error:", err.Error())
		return
	}
	<-doneC
}
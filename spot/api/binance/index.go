package binance

import (
	"context"
	"go_binance_futures/utils"
	"sort"
	"strconv"

	"github.com/adshao/go-binance/v2"
	"github.com/beego/beego/v2/adapter/logs"
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

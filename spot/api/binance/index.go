package binance

import (
	"context"
	"go_binance_futures/utils"
	"strconv"

	"github.com/adshao/go-binance/v2"
	"github.com/beego/beego/v2/adapter/logs"
	"github.com/beego/beego/v2/core/config"
)


var api_key, _ = config.String("binance::api_key")
var api_secret, _ = config.String("binance::api_secret")

var client = binance.NewClient(api_key, api_secret)

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
// @see https://binance-docs.github.io/apidocs/futures/cn/#trade-3
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

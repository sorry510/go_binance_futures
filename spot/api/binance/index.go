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
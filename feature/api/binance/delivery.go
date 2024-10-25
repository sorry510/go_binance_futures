package binance

import (
	"context"
	"go_binance_futures/utils"

	"github.com/adshao/go-binance/v2/delivery"
	"github.com/beego/beego/v2/adapter/logs"
)

// @returns /doc/futuresAccount.js
func GetDeliveryAccount() (res *delivery.Account, err error) {
	res, err = deliveryClient.NewGetAccountService().Do(context.Background())
	if err != nil {
		logs.Error(err)
		return nil, err
	}
	logs.Info(utils.ToJson(res))
	return res, err
}
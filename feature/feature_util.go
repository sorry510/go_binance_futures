package feature

import (
	"strings"

	"go_binance_futures/feature/api/binance"
	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

// 获取币的交易精度。正常交易优先使用 12 小时同步进 symbols 的本地精度，
// 只有本地缺失时才回退 ExchangeInfo。
func GetCoinOrderSize(symbol string) (string, string) {
	tickSize := "0.0"
	stepSize := "0.0"
	symbol = strings.ToUpper(strings.TrimSpace(symbol))

	var local models.Symbols
	if err := orm.NewOrm().QueryTable(new(models.Symbols)).Filter("symbol", symbol).One(&local); err == nil {
		if strings.TrimSpace(local.TickSize) != "" && local.TickSize != "0" &&
			strings.TrimSpace(local.StepSize) != "" && local.StepSize != "0" {
			return local.TickSize, local.StepSize
		}
	}

	res, err := binance.GetExchangeInfo()
	if err != nil {
		logs.Error("GetExchangeInfoError:", err)
		return tickSize, stepSize
	}
	for _, item := range res.Symbols {
		if item.Symbol == symbol {
			priceFilter := item.PriceFilter()
			if priceFilter != nil {
				tickSize = priceFilter.TickSize // 价格精度
			}
			lotSizeFilter := item.LotSizeFilter()
			if lotSizeFilter != nil {
				stepSize = lotSizeFilter.StepSize // 数量精度
			}
			return tickSize, stepSize
		}
	}
	return tickSize, stepSize
}

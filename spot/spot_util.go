package spot

import (
	"go_binance_futures/spot/api/binance"

	"github.com/beego/beego/v2/core/logs"
)

// 获取币的交易精度
func GetCoinOrderSize(symbol string) (string, string)  {
	tickSize := "0.0"
	stepSize := "0.0"
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

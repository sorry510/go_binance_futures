package feature

import (
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/models"
	"strings"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

// 更新币种的交易精度和插入新币
func UpdateDeliverySymbolsTradePrecision() {
	res, err := binance.GetDeliveryExchangeInfo()
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
			o.QueryTable("delivery_symbols").Filter("symbol", symbol.Symbol).Update(orm.Params{
				"tickSize": tickSize,
				"stepSize": stepSize,
			})
			
			suffixType := ""
			if strings.HasSuffix(symbol.Symbol, "_PERP") {
				suffixType = "PERP" // 只处理永续合约
			}
			
			if suffixType != "" { // 只要永续合约
				if !o.QueryTable("delivery_symbols").Filter("symbol", symbol.Symbol).Exist() {
					// 没有的币种插入
					logs.Info("add new futures symbol", symbol.Symbol)
					o.Insert(&models.DeliverySymbols{
						Symbol: symbol.Symbol,
						Enable: 0, // 默认不开启
						Leverage: 3,
						MarginType: "CROSSED", // 杠杆类型 ISOLATED(逐仓), CROSSED(全仓)
						TickSize: tickSize,
						StepSize: stepSize,
						Usdt: "10",
						Profit: "20",
						Loss: "20",
						StrategyType: "global",
						Type: suffixType, // 永续合约
					})
				}
			}
		}
	}
}
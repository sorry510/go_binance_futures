package spot

import (
	"go_binance_futures/models"
	"go_binance_futures/spot/api/binance"
	"go_binance_futures/spot/notify"
	"go_binance_futures/utils"
	"strconv"

	spot_api "github.com/adshao/go-binance/v2"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

func TryBuyNewSymbols() {
	o := orm.NewOrm()
	var coins []models.NewSymbols
	o.QueryTable("new_symbols").OrderBy("ID").Filter("enable", 1).Filter("type", 1).All(&coins) // 允许抢购的币
	
	countHasSize := 0
	
	for _, coin := range coins {
		if coin.StepSize != "" {
			countHasSize++
			_, err := tryBuyMarket(coin.Symbol, coin.Usdt, coin.StepSize)
			if err == nil {
				coin.Enable = 0 // 更新为禁用
			}
			orm.NewOrm().Update(&coin)
		}
	}
	
	if countHasSize == len(coins) {
		// logs.Info("不需要更新精度")
		return
	}
	
	res, err := binance.GetExchangeInfo()
	if err != nil {
		logs.Error(err)
	}
	for _, item := range res.Symbols {
		var stepSize string
		lotSizeFilter := item.LotSizeFilter()
		if lotSizeFilter != nil {
			stepSize = lotSizeFilter.StepSize
		}
		if stepSize == "" {
			continue
		}
		
		for _, coin := range coins {
			// 找到了币的精度，说明币可能上线了
			if item.Symbol == coin.Symbol {
				logs.Info("lotSize:", stepSize)
				
				_, err := tryBuyMarket(coin.Symbol, coin.Usdt, stepSize)
				
				if err == nil {
					coin.Enable = 0 // 更新为禁用
					logs.Info("抢购成功，关闭交易")
				}
				coin.StepSize = stepSize
				orm.NewOrm().Update(&coin)
			}
		}
		
			
	}
}

func tryBuyMarket(symbol string, usdt string, stepSize string) (res *spot_api.CreateOrderResponse, err error) {
	resPrice, err1 := binance.GetTickerPrice(symbol)
	if err1 != nil {
		logs.Error(err1)
	}
	usdt_float64, _ := strconv.ParseFloat(usdt, 64) // 交易金额
	price_float64, _ := strconv.ParseFloat(resPrice[0].Price, 64) // 预计交易价格
	buyPrice := utils.GetTradePrecision(price_float64, stepSize) // 合理精度的价格
	quantity := usdt_float64 / buyPrice  // 购买数量
	quantity = utils.GetTradePrecision(quantity, stepSize) // 合理精度的价格
	// logs.Info("symbol:", symbol, "buyPrice:", buyPrice, "quantity:", quantity)
	
	res, err = binance.BuyMarket(symbol, quantity)
	if err == nil {
		notify.BuyOrderSuccess(symbol, quantity, buyPrice)
	}
	return res, err
}
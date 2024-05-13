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
	
	notHasSizeSymbols := []string{}
	
	for _, coin := range coins {
		if coin.StepSize != "0" {
			if coin.Side == "buy" {
				_, err := tryBuyMarket(coin.Symbol, coin.Usdt, coin.StepSize)
				if err == nil {
					coin.Enable = 0 // 更新为禁用
				}
				orm.NewOrm().Update(&coin)
			}
			if coin.Side == "sell" {
				_, err := trySellMarket(coin.Symbol, coin.Quantity, coin.StepSize)
				if err == nil {
					coin.Enable = 0 // 更新为禁用
				}
				orm.NewOrm().Update(&coin)
			}
		} else {
			notHasSizeSymbols = append(notHasSizeSymbols, coin.Symbol)
		}
	}
	if len(notHasSizeSymbols) == 0 {
		// logs.Info("没有币需要更新交易精度")
		return
	}
	
	res, err := binance.GetExchangeInfo()
	if err != nil {
		logs.Error("GetExchangeInfoError:", err)
		return
	}
	symbolMap := make(map[string]string)
	for _, item := range res.Symbols {
		lotSizeFilter := item.LotSizeFilter()
		if lotSizeFilter != nil {
			symbolMap[item.Symbol] = lotSizeFilter.StepSize
		}
	}
	
	for _, coin := range coins {
		// 找到了币的精度，说明币可能上线了
		if stepSize, ok := symbolMap[coin.Symbol]; ok {
			logs.Info("lotSize:", stepSize)
			if coin.Side == "buy" {
				_, err := tryBuyMarket(coin.Symbol, coin.Usdt, stepSize)
			
				if err == nil {
					coin.Enable = 0 // 更新为禁用
					logs.Info("抢购成功，关闭交易")
				}
			}
			if coin.Side == "sell" {
				_, err := trySellMarket(coin.Symbol, coin.Usdt, stepSize)
			
				if err == nil {
					coin.Enable = 0 // 更新为禁用
					logs.Info("抢卖成功，关闭交易")
				}
			}
			
			coin.StepSize = stepSize
			orm.NewOrm().Update(&coin)
		} else {
			logs.Info("还未上线此币种,未确定交易价格数量精度:", coin.Symbol)
		}
	}
}

func tryBuyMarket(symbol string, usdt string, stepSize string) (res *spot_api.CreateOrderResponse, err error) {
	logs.Info("尝试开始抢币symbol:", symbol)
	resPrice, err1 := binance.GetTickerPrice(symbol)
	if err1 != nil {
		logs.Info("还未上线此币种,未确定交易价格symbol:", symbol)
		return nil, err1
	}
	usdt_float64, _ := strconv.ParseFloat(usdt, 64) // 交易金额
	price_float64, _ := strconv.ParseFloat(resPrice[0].Price, 64) // 预计交易价格
	buyPrice := utils.GetTradePrecision(price_float64, stepSize) // 合理精度的价格
	quantity := usdt_float64 / buyPrice  // 购买数量
	quantity = utils.GetTradePrecision(quantity, stepSize) // 合理精度的价格
	// logs.Info("symbol:", symbol, "buyPrice:", buyPrice, "quantity:", quantity)
	
	res, err = binance.BuyMarket(symbol, quantity)
	if err != nil {
		logs.Info("购买失败symbol:", symbol)
	} else {
		// 购买成功
		notify.BuyOrderSuccess(symbol, quantity, buyPrice)
	}
	return res, err
}

func trySellMarket(symbol string, quantity string, stepSize string) (res *spot_api.CreateOrderResponse, err error) {
	logs.Info("尝试开始抢卖symbol:", symbol)
	quantity_float64, _ := strconv.ParseFloat(quantity, 64) // 卖单数量
	quantity_float64 = utils.GetTradePrecision(quantity_float64, stepSize) // 合理精度的数量
	// logs.Info("symbol:", symbol, "buyPrice:", buyPrice, "quantity:", quantity)]
	
	res, err = binance.SellMarket(symbol, quantity_float64)
	if err != nil {
		logs.Info("卖出失败symbol:", symbol)
	} else {
		// 卖出成功
		notify.SellOrderSuccess(symbol, quantity_float64, res.Price)
	}
	return res, err
}
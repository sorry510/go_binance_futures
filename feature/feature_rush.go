package feature

import (
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/feature/notify"
	"go_binance_futures/models"
	"go_binance_futures/utils"
	"strconv"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

func TryRush() {
	o := orm.NewOrm()
	var coins []models.NewSymbols
	o.QueryTable("new_symbols").OrderBy("ID").Filter("enable", 1).Filter("type", 2).All(&coins) // 允许抢购的合约币
	
	notHasSizeSymbols := []string{}
	
	for _, coin := range coins {
		if coin.StepSize != "0" {
			_, err := tryBuyMarket(coin, coin.StepSize)
			if err == nil {
				coin.Enable = 0 // 更新为禁用
			}
			orm.NewOrm().Update(&coin)
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
			_, err := tryBuyMarket(coin, stepSize)
		
			if err == nil {
				coin.Enable = 0 // 更新为禁用
				logs.Info("合约抢购成功，关闭交易")
			}
			coin.StepSize = stepSize
			orm.NewOrm().Update(&coin)
		} else {
			logs.Info("还未上线此合约币种,未确定交易价格数量精度:", coin.Symbol)
		}
	}
}

func tryBuyMarket(coin models.NewSymbols, stepSize string) (res *futures.CreateOrderResponse, err error) {
	symbol := coin.Symbol
	usdt := coin.Usdt
	resPrice, err1 := binance.GetTickerPrice(symbol)
	if err1 != nil {
		logs.Info("还未上线此合约币种,未确定交易价格symbol:", symbol)
		return nil, err1
	}
	// 修改仓位模式
	if coin.MarginType == "ISOLATED" {
		binance.SetMarginType(symbol, futures.MarginTypeIsolated)
	} else {
		binance.SetMarginType(symbol, futures.MarginTypeCrossed)
	}
	binance.SetLeverage(symbol, int(coin.Leverage))  // 修改合约倍数
	logs.Info("尝试开始合约抢币symbol:", symbol)
	
	usdt_float64, _ := strconv.ParseFloat(usdt, 64) // 交易金额
	buyPrice, _ := strconv.ParseFloat(resPrice[0].Price, 64) // 预计交易价格
	if buyPrice < 0.00000001 {
		logs.Info("还未正式上线此合约币种,没有交易盘价格", symbol)
	}
	logs.Info("尝试开始合约抢币,价格为:", symbol)
	leverage_float64 := float64(coin.Leverage) // 合约倍数
	quantity := (usdt_float64 / buyPrice) * leverage_float64  // 购买数量
	quantity = utils.GetTradePrecision(quantity, stepSize) // 合理精度的价格
	// logs.Info("symbol:", symbol, "buyPrice:", buyPrice, "quantity:", quantity)
	
	var side string
	if coin.Side == "buy" {
		side = "做多"
		res, err = binance.BuyMarket(symbol, quantity, futures.PositionSideTypeLong)
	} else if coin.Side == "sell" {
		side = "做空"
		res, err = binance.SellMarket(symbol, quantity, futures.PositionSideTypeShort)
	}
	if err != nil {
		logs.Info("购买失败symbol:", symbol)
	} else {
		// 购买成功
		notify.RushOrderSuccess(symbol, quantity, buyPrice, side)
	}
	return res, err
}
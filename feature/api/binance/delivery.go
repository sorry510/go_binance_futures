package binance

import (
	"context"
	"go_binance_futures/utils"

	"github.com/adshao/go-binance/v2/delivery"
	"github.com/beego/beego/v2/adapter/logs"
	"github.com/beego/beego/v2/client/orm"
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

// @see https://developers.binance.com/docs/zh-CN/derivatives/coin-margined-futures/market-data/Exchange-Information
func GetDeliveryExchangeInfo()(res *delivery.ExchangeInfo, err error) {
	res, err = deliveryClient.NewExchangeInfoService().Do(context.Background())
	if err != nil {
		logs.Error(err)
		return nil, err
	}
	// logs.Info(utils.ToJson(res))
	return res, err
}

func UpdateDeliveryCoinByWs() {
	// binance.BaseWsMainURL = "wss://testnet.binance.vision/ws"
	// logs.Info("start delivery ws")
	var lock = false
	var o = orm.NewOrm()
	doneC, _, err := delivery.WsAllMarketTickerServe(func(event delivery.WsAllMarketTickerEvent) {
		if !lock {
			lock = true
			for _, ticker := range event {
				// 永续合约
				o.Raw(
					"UPDATE `delivery_symbols` set `percentChange` = ?, `close` = ?, `open` = ?, `low` = ?, `high` = ?, `updateTime` = ?, `baseVolume` = ?, `quoteVolume` = ?, `closeQty` = ?,  `tradeCount` = ?, `lastClose` = close, `lastUpdateTime` = updateTime WHERE `symbol` = ?",
					ticker.PriceChangePercent,
					ticker.ClosePrice,
					ticker.OpenPrice,
					ticker.LowPrice,
					ticker.HighPrice,
					ticker.Time,
					ticker.BaseVolume, // 成交量
					ticker.QuoteVolume, // 成交额
					ticker.CloseQty, // 最新成交价格上的成交量
					ticker.TradeCount, // 成交数
					
					ticker.Symbol,
				).Exec()
			}
			lock = false
		}
	}, func(err error) {
		lock = false
		logs.Error("delivery ws run error:", err.Error())
	})
	if err != nil {
		logs.Error("delivery ws start error:", err.Error())
		return
	}
	<-doneC
}
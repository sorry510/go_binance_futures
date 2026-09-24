package binance

import (
	"context"
	"strings"

	"go_binance_futures/service/binanceapiusage"
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
func GetDeliveryExchangeInfo() (res *delivery.ExchangeInfo, err error) {
	res, err = deliveryClient.NewExchangeInfoService().Do(context.Background())
	if err != nil {
		logs.Error(err)
		return nil, err
	}
	var orderLimit10s, orderLimit1m int64
	for _, rateLimit := range res.RateLimits {
		switch {
		case strings.EqualFold(rateLimit.RateLimitType, "REQUEST_WEIGHT") &&
			strings.EqualFold(rateLimit.Interval, "MINUTE") &&
			rateLimit.IntervalNum == 1 &&
			rateLimit.Limit > 0:
			binanceapiusage.Default().SetWeightLimit("delivery", "mainnet", rateLimit.Limit, "exchange_info")
		case strings.EqualFold(rateLimit.RateLimitType, "ORDERS") &&
			strings.EqualFold(rateLimit.Interval, "SECOND") &&
			rateLimit.IntervalNum == 10:
			orderLimit10s = rateLimit.Limit
		case strings.EqualFold(rateLimit.RateLimitType, "ORDERS") &&
			strings.EqualFold(rateLimit.Interval, "MINUTE") &&
			rateLimit.IntervalNum == 1:
			orderLimit1m = rateLimit.Limit
		}
	}
	binanceapiusage.Default().SetOrderLimits("delivery", "mainnet", orderLimit10s, orderLimit1m)
	// logs.Info(utils.ToJson(res))
	return res, err
}

var flagWsDelivery = 0

func UpdateDeliveryCoinByWs() {
	// binance.BaseWsMainURL = "wss://testnet.binance.vision/ws"
	var lock = false
	var o = orm.NewOrm()
	_, _, err := wsDeliveryAllMarketTickerServe(func(event delivery.WsAllMarketTickerEvent) {
		if flagWsDelivery == 0 {
			logs.Info("delivery ws start")
			flagWsDelivery = 1
		}
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
					ticker.BaseVolume,  // 成交量
					ticker.QuoteVolume, // 成交额
					ticker.CloseQty,    // 最新成交价格上的成交量
					ticker.TradeCount,  // 成交数

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
}

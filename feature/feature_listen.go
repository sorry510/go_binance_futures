package feature

import (
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/feature/notify"
	"go_binance_futures/models"
	"strconv"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

func ListenCoin() {
	o := orm.NewOrm()
	var coins []models.ListenSymbols
	o.QueryTable("listen_symbols").OrderBy("ID").Filter("enable", 1).Filter("type", 2).All(&coins) // 通知币列表
	
	for _, coin := range coins {
		logs.Info("监听合约币种:", coin.Symbol)
		kline_1, err1 := binance.GetKlineData(coin.Symbol, coin.KlineInterval, 10)
		if err1 != nil {
			logs.Error("k线错误, 合约币种是:", coin.Symbol)
			continue
		}
		
		percentLimit, _ := strconv.ParseFloat(coin.ChangePercent, 64)
		percentLimit = percentLimit / 100 // 变化幅度
	
		lastOpenPrice, _ := strconv.ParseFloat(kline_1[1].Open, 64) // 上一根 k 线的开盘价
		nowPrice, _ := strconv.ParseFloat(kline_1[0].Close, 64) // 当前价格
		
		if (nowPrice > lastOpenPrice) && 
			(nowPrice - lastOpenPrice) / lastOpenPrice >= percentLimit &&
			((kline_1[0].CloseTime - coin.LastNoticeTime >= 60 * 1000 * coin.NoticeLimitMin && coin.LastNoticeType == "up") || coin.LastNoticeType != "up") {
			// 价格上涨
			coin.LastNoticeTime = kline_1[0].CloseTime
			coin.LastNoticeType = "up"
			orm.NewOrm().Update(&coin)
			
			notify.ListenFutureCoin(coin.Symbol, "极速上涨", (nowPrice - lastOpenPrice) / lastOpenPrice)
		}
		if (nowPrice < lastOpenPrice) && 
			(lastOpenPrice - nowPrice) / lastOpenPrice >= percentLimit &&
			((kline_1[0].CloseTime - coin.LastNoticeTime >= 60 * 1000 * coin.NoticeLimitMin && coin.LastNoticeType == "down") || coin.LastNoticeType != "down") {
			// 价格下跌
			coin.LastNoticeTime = kline_1[0].CloseTime
			coin.LastNoticeType = "down"
			orm.NewOrm().Update(&coin)
			
			notify.ListenFutureCoin(coin.Symbol, "极速下跌", (lastOpenPrice - nowPrice) / lastOpenPrice)
		}
	}
}

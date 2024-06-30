package controllers

import (
	"github.com/beego/beego/v2/core/config"
	"github.com/beego/beego/v2/server/web"
)

type IndexController struct {
	web.Controller
}

func (ctrl *IndexController) GetServiceConfig() {
	var debug, _ = config.String("debug")
	
	var coinExcludeSymbols, _ = config.String("coin::exclude_symbols")
	var coinMaxCount, _ = config.String("coin::max_count")
	var coinOrderType, _ = config.String("coin::order_type")
	var coinAllowLong, _ = config.String("coin::allow_long")
	var coinAllShort, _ = config.String("coin::allow_short")
	
	var tradeFutureEnable, _ = config.String("trade::future_enable")
	var tradeStrategyTrade, _ = config.String("trade::strategy_trade")
	var tradeStrategyCoin, _ = config.String("trade::strategy_coin")
	var tradeNewEnable, _ = config.String("trade::new_enable")
	
	var spotNewEnable, _ = config.String("spot::new_enable")
	
	var noticeCoinEnable, _ = config.String("notice_coin::enable")
	
	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"data": map[string]interface{} {
			"debug": debug,
			
			"coinExcludeSymbols": coinExcludeSymbols,
			"coinMaxCount": coinMaxCount,
			"coinOrderType": coinOrderType,
			"coinAllowLong": coinAllowLong,
			"coinAllShort": coinAllShort,
			
			"tradeFutureEnable": tradeFutureEnable,
			"tradeStrategyTrade": tradeStrategyTrade,
			"tradeStrategyCoin": tradeStrategyCoin,
			"tradeNewEnable": tradeNewEnable,
			
			"spotNewEnable": spotNewEnable,
			
			"noticeCoinEnable": noticeCoinEnable,
		},
		"msg": "success",
	})
}
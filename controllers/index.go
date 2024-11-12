package controllers

import (
	"go_binance_futures/notify"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/config"
	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
)

type IndexController struct {
	web.Controller
}

func (ctrl *IndexController) GetServiceConfig() {
	var debug, _ = config.String("debug")
	systemConfig, err := utils.GetSystemConfig()
	if err != nil {
		logs.Error("GetSystemConfig:", err)
		return
	}
	
	var coinExcludeSymbols = systemConfig.FutureExcludeSymbols
	var coinMaxCount = systemConfig.FutureMaxCount
	var coinOrderType = systemConfig.FutureOrderType
	var coinAllowLong = systemConfig.FutureAllowLong
	var coinAllowShort = systemConfig.FutureAllowShort
	
	var tradeFutureTest = systemConfig.FutureTest
	var tradeFutureTestNoticeLimitMin = systemConfig.FutureTestNoticeLimitMin
	var tradeFutureEnable = systemConfig.FutureEnable
	var tradeStrategyTrade = systemConfig.FutureStrategyTrade
	var tradeStrategyCoin = systemConfig.FutureStrategyCoin
	var tradeNewEnable = systemConfig.FutureNewEnable
	
	var spotNewEnable = systemConfig.SpotNewEnable
	
	var noticeCoinEnable = systemConfig.NoticeCoinEnable
	
	var listenCoinEnable = systemConfig.ListenCoinEnable
	var listenFundingRate = systemConfig.ListenFundingRateEnable
	var externalLinks, _ = config.String("external::links")
	
	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"data": map[string]interface{} {
			"debug": debug,
			
			"coinExcludeSymbols": coinExcludeSymbols,
			"coinMaxCount": coinMaxCount,
			"coinOrderType": coinOrderType,
			"coinAllowLong": coinAllowLong,
			"coinAllowShort": coinAllowShort,
			
			"tradeFutureTest": tradeFutureTest,
			"tradeFutureTestNoticeLimitMin": tradeFutureTestNoticeLimitMin,
			"tradeFutureEnable": tradeFutureEnable,
			"tradeStrategyTrade": tradeStrategyTrade,
			"tradeStrategyCoin": tradeStrategyCoin,
			"tradeNewEnable": tradeNewEnable,
			
			"spotNewEnable": spotNewEnable,
			
			"noticeCoinEnable": noticeCoinEnable,
			
			"listenCoinEnable": listenCoinEnable,
			"listenFundingRate": listenFundingRate,
			
			"externalLinks": externalLinks,
		},
		"msg": "success",
	})
}

func (ctrl *IndexController) EditServiceConfig() {
	systemConfig, _ := utils.GetSystemConfig()
	
	ctrl.BindJSON(&systemConfig)
	
	_, err := orm.NewOrm().Update(&systemConfig) // _ 是受影响的条数
    if err != nil {
        // 处理错误
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "edit failed"))
		return
    }
	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"data": systemConfig,
		"msg": "success",
	})
}

func (ctrl *IndexController) TestPusher() {
	var pusher = notify.GetNotifyChannel()
	pusher.TestPusher()
	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"msg": "success",
	})
}
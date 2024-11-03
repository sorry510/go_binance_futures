package routers

import (
	"go_binance_futures/controllers"

	"github.com/beego/beego/v2/server/web"
)

func init() {
	web.Router("/login", &controllers.LoginController{}, "post:Post") // 登录
	
	web.Router("/service/config", &controllers.IndexController{}, "get:GetServiceConfig") // 服务配置信息
	web.Router("/test-pusher", &controllers.IndexController{}, "post:TestPusher") // 测试推送
	
	web.Router("/features", &controllers.FeatureController{}, "get:Get;post:Post") // 列表查询和新增
	web.Router("/features/:id", &controllers.FeatureController{}, "delete:Delete;put:Edit") // 更新和删除
	web.Router("/features/enable/:flag", &controllers.FeatureController{}, "put:UpdateEnable") // 修改所有的合约交易对开启关闭
	web.Router("/features/batch", &controllers.FeatureController{}, "put:BatchEdit") // 修改所有的合约交易
	web.Router("/features/strategy-rule/test/:id", &controllers.FeatureController{}, "post:TestStrategyRule") // 测试策略规则
	
	web.Router("/rush", &controllers.RushController{}, "get:Get;post:Post") // 列表查询和新增
	web.Router("/rush/:id", &controllers.RushController{}, "delete:Delete;put:Edit") // 更新和删除
	web.Router("/rush/enable/:flag", &controllers.RushController{}, "put:UpdateEnable") // 修改所有的交易对开启关闭
	
	web.Router("/notice/coin", &controllers.NoticeCoinController{}, "get:Get;post:Post") // 列表查询和新增
	web.Router("/notice/coin/:id", &controllers.NoticeCoinController{}, "delete:Delete;put:Edit") // 更新和删除
	web.Router("/notice/coin/enable/:flag", &controllers.NoticeCoinController{}, "put:UpdateEnable") // 修改所有的交易对开启关闭
	
	web.Router("/listen/coin", &controllers.ListenCoinController{}, "get:Get;post:Post") // 列表查询和新增
	web.Router("/listen/coin/:id", &controllers.ListenCoinController{}, "delete:Delete;put:Edit") // 更新和删除
	web.Router("/listen/coin/kc-chart/:id", &controllers.ListenCoinController{}, "get:GetKcLineChart") // kcChart
	web.Router("/listen/coin/enable/:flag", &controllers.ListenCoinController{}, "put:UpdateEnable") // 修改所有的交易对开启关闭
	web.Router("/listen/funding-rates", &controllers.ListenCoinController{}, "get:GetFundingRates") // 合约费率列表
	web.Router("/listen/funding-rate/history", &controllers.ListenCoinController{}, "get:GetFundingRateHistory") // 合约费率历史
	web.Router("/listen/strategy-rule/test/:id", &controllers.ListenCoinController{}, "post:TestStrategyRule") // 测试策略规则
	
	web.Router("/orders", &controllers.OrderController{}, "get:Get;delete:DeleteAll") // order list 和 删除所有 order
	web.Router("/config", &controllers.ConfigController{}, "get:Get;put:Edit") // config get and edit
	
	web.Router("/futures/account", &controllers.AccountController{}, "get:GetBinanceFuturesAccount") // 获取合约账户信息
	web.Router("/futures/positions", &controllers.AccountController{}, "get:GetBinanceFuturesPositions") // 获取合约持仓信息
	web.Router("/futures/open-orders", &controllers.AccountController{}, "get:GetBinanceFuturesOpenOrders") // 获取合约挂单信息
	
	web.Router("/start", &controllers.CommandController{}, "post:Start") // start
	web.Router("/stop", &controllers.CommandController{}, "post:Stop") // stop
	web.Router("/pull", &controllers.CommandController{}, "post:GitPull") // git pull
	web.Router("/pm2-log", &controllers.CommandController{}, "get:Pm2Log") // pm2-log
}


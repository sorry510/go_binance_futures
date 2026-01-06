package routers

import (
	"go_binance_futures/controllers"

	"github.com/beego/beego/v2/server/web"
)

func init() {
	web.Router("/login", &controllers.LoginController{}, "post:Post") // 登录
	
	web.Router("/service/config", &controllers.IndexController{}, "get:GetServiceConfig;put:EditServiceConfig") // 服务配置信息
	web.Router("/test-pusher", &controllers.IndexController{}, "post:TestPusher") // 测试推送
	web.Router("/update-market-condition", &controllers.IndexController{}, "post:UpdateMarketCondition") // 手动触发更新市场状态
	web.Router("/notify-config", &controllers.NotifyConfigController{}, "get:Get;post:Post") // 列表查询和新增
	web.Router("/notify-config/:id", &controllers.NotifyConfigController{}, "delete:Delete;put:Edit") // 更新和删除

	web.Router("/features", &controllers.FeatureController{}, "get:Get;post:Post") // 列表查询和新增
	web.Router("/features-options", &controllers.FeatureController{}, "get:GetOptions") // 列表查询
	web.Router("/features/:id", &controllers.FeatureController{}, "delete:Delete;put:Edit;get:Show") // 更新和删除,查询
	web.Router("/features/enable/:flag", &controllers.FeatureController{}, "put:UpdateEnable") // 修改所有的合约交易对开启关闭
	web.Router("/features/batch", &controllers.FeatureController{}, "put:BatchEdit") // 修改所有的合约交易
	web.Router("/features/strategy-rule/test/:id", &controllers.FeatureController{}, "post:TestStrategyRule") // 测试策略规则
	
	web.Router("/test-strategy-results", &controllers.TestStrategyResultController{}, "get:Get;delete:DeleteAll") // 测试策略的下单和平仓
	web.Router("/test-strategy-results/:id", &controllers.TestStrategyResultController{}, "delete:Delete;get:Show") // 删除某个测试策略结果和获取明细
	web.Router("/test-strategy-results/test/:symbol", &controllers.TestStrategyResultController{}, "post:TestStrategyRule") // 测试策略结果的某个平仓测试是否符合条件
	
	web.Router("/spots", &controllers.SpotController{}, "get:Get;post:Post") // 列表查询和新增
	web.Router("/spots/:id", &controllers.SpotController{}, "delete:Delete;put:Edit") // 更新和删除
	web.Router("/spots/enable/:flag", &controllers.SpotController{}, "put:UpdateEnable") // 修改所有的合约交易对开启关闭
	web.Router("/spots/batch", &controllers.SpotController{}, "put:BatchEdit") // 修改所有的合约交易
	// web.Router("/spots/strategy-rule/test/:id", &controllers.SpotController{}, "post:TestStrategyRule") // 测试策略规则
	
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
	web.Router("/listen/funding-rates/:id", &controllers.ListenCoinController{}, "put:EditFundingRates") // 编辑合约费率
	web.Router("/listen/funding-rate/history", &controllers.ListenCoinController{}, "get:GetFundingRateHistory") // 合约费率历史
	web.Router("/listen/strategy-rule/test/:id", &controllers.ListenCoinController{}, "post:TestStrategyRule") // 测试策略规则
	
	web.Router("/orders", &controllers.OrderController{}, "get:Get;delete:DeleteAll") // order list 和 删除所有 order
	web.Router("/orders/:id", &controllers.OrderController{}, "delete:Delete") // 删除某个订单
	web.Router("/config", &controllers.ConfigController{}, "get:Get;put:Edit") // config get and edit
	
	web.Router("/futures/account", &controllers.AccountController{}, "get:GetBinanceFuturesAccount") // 获取合约账户信息
	web.Router("/futures/positions", &controllers.AccountController{}, "get:GetBinanceFuturesPositions") // 获取合约持仓信息
	web.Router("/futures/open-orders", &controllers.AccountController{}, "get:GetBinanceFuturesOpenOrders") // 获取合约挂单信息
	web.Router("/futures/local/positions", &controllers.AccountController{}, "get:GetLocalFuturesPositions") // 获取本地存储的合约持仓信息
	web.Router("/futures/local/positions/:id", &controllers.AccountController{}, "put:EditLocalFuturesPositions;delete:DelLocalFuturesPositions") // 修复和删除本地存储的合约持仓信息
	web.Router("/futures/local/open-orders", &controllers.AccountController{}, "get:GetLocalFuturesOpenOrders") // 获取本地存储的挂单信息
	
	web.Router("/fund-rate/eat", &controllers.EatRateController{}, "get:Get;post:Post") // 列表查询和新增
	web.Router("/fund-rate/eat/:id", &controllers.EatRateController{}, "delete:Delete;put:Edit") // 更新和删除
	web.Router("/fund-rate/eat/start/:id", &controllers.EatRateController{}, "post:Start") // start
	web.Router("/fund-rate/eat/end/:id", &controllers.EatRateController{}, "post:End") // end
	
	web.Router("/strategy-templates", &controllers.StrategyTemplateController{}, "get:Get;post:Post") // 策略模板
	web.Router("/strategy-templates/:id", &controllers.StrategyTemplateController{}, "delete:Delete;put:Edit") // 策略模板更新
	web.Router("/strategy-templates/test/:symbol", &controllers.StrategyTemplateController{}, "post:TestStrategyRule") // 测试策略规则
	
	web.Router("/start", &controllers.CommandController{}, "post:Start") // start
	web.Router("/stop", &controllers.CommandController{}, "post:Stop") // stop
	web.Router("/pull", &controllers.CommandController{}, "post:GitPull") // git pull
	web.Router("/pm2-log", &controllers.CommandController{}, "get:Pm2Log") // pm2-log
}


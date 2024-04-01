package routers

import (
	"go_binance_futures/controllers"

	"github.com/beego/beego/v2/server/web"
)

func init() {
	web.Router("/login", &controllers.LoginController{}, "post:Post") // 登录
	
	web.Router("/features", &controllers.FeatureController{}, "get:Get;post:Post") // 列表查询和新增
	web.Router("/features/:id", &controllers.FeatureController{}, "delete:Delete;put:Edit") // 更新和删除
	web.Router("/features/enable/:flag", &controllers.FeatureController{}, "put:UpdateEnable") // 修改所有的合约交易对开启关闭
	
	web.Router("/rush", &controllers.RushController{}, "get:Get;post:Post") // 列表查询和新增
	web.Router("/rush/:id", &controllers.RushController{}, "delete:Delete;put:Edit") // 更新和删除
	web.Router("/rush/enable/:flag", &controllers.RushController{}, "put:UpdateEnable") // 修改所有的交易对开启关闭
	
	web.Router("/orders", &controllers.OrderController{}, "get:Get;delete:DeleteAll") // order list 和 删除所有 order
	web.Router("/config", &controllers.ConfigController{}, "get:Get;put:Edit") // config get and edit
	
	web.Router("/start", &controllers.CommandController{}, "post:Start") // start
	web.Router("/stop", &controllers.CommandController{}, "post:Stop") // stop
	web.Router("/pull", &controllers.CommandController{}, "post:GitPull") // git pull
	web.Router("/pm2-log", &controllers.CommandController{}, "get:Pm2Log") // pm2-log
}


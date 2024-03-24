package main

import (
	"go_binance_futures/middlewares"
	"go_binance_futures/models"
	_ "go_binance_futures/routers"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/config"
	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
	_ "github.com/mattn/go-sqlite3"
)

var webPort, _ = config.String("web::port")
var webIndex, _ = config.String("web::index")

func init() {
	web.BConfig.CopyRequestBody = true // post 参数
	web.SetStaticPath("/" + webIndex, "static") // 设置静态文件
	registerModels() // 注册模型
	registerMiddlewares() // 添加中间件
}

func registerModels() {
	orm.RegisterModel(new(models.Order))
	orm.RegisterModel(new(models.Symbols))
	
	orm.RegisterDriver("sqlite3", orm.DRSqlite)
	orm.RegisterDataBase("default", "sqlite3", "./db/coin.db")
	// orm.Debug = true
}


func registerMiddlewares() {
    web.InsertFilter("*", web.BeforeRouter, middlewares.JwtMiddleware)
}

func main() {
	logs.Info("webIndex:", webIndex)
	web.Run(":" + webPort)
}


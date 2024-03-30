package main

import (
	"go_binance_futures/feature"
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/middlewares"
	"go_binance_futures/models"
	_ "go_binance_futures/routers"
	"time"

	"strconv"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/config"
	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
	_ "github.com/mattn/go-sqlite3"
)

var webPort, _ = config.String("web::port")
var webIndex, _ = config.String("web::index")
var taskSleepTime, _ = config.String("coin::sleep_time")
var taskSleepTimeInt, _ = strconv.Atoi(taskSleepTime)

func init() {
	web.BConfig.CopyRequestBody = true // post 参数
	web.SetStaticPath("/" + webIndex, "static") // 设置静态文件
	logs.Info("webIndex:", webIndex)
	
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
    web.InsertFilter("*", web.BeforeRouter, middlewares.CorsMiddleware)
    web.InsertFilter("*", web.BeforeRouter, middlewares.JwtMiddleware)
}

func main() {
	
	// 更新币种交易精度
	go func() {
		for {
			feature.UpdateSymbolsTradePrecision()
			time.Sleep(5 * time.Minute) // 5分钟更新一次精度
		}
	}()
	
	// websocket 订阅更新币种价格
	go func() {
		binance.UpdateCoinByWs()
	}()
	
	// trade script
	go func() {
		for {
			feature.StartTrade()

			// 等待 taskSleepTimeInt 秒再继续执行
			time.Sleep(time.Duration(taskSleepTimeInt) * time.Second)
		}
	}()
	
	// // web
	web.Run(":" + webPort)
}


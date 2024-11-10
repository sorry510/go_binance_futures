package main

import (
	"go_binance_futures/feature"
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/middlewares"
	"go_binance_futures/models"
	"go_binance_futures/rate"
	_ "go_binance_futures/routers"
	"go_binance_futures/spot"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/config"
	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
	_ "github.com/mattn/go-sqlite3"
)

var debug, _ = config.String("debug")
var webPort, _ = config.String("web::port")
var webIndex, _ = config.String("web::index")

func init() {
	web.BConfig.CopyRequestBody = true // post 参数
	web.SetStaticPath("/" + webIndex, "static") // 设置静态文件
	logs.Info("webIndex:", webIndex)
	
	registerModels() // 注册模型
	registerMiddlewares() // 添加中间件
}

func registerModels() {
	orm.RegisterModel(new(models.Config))
	orm.RegisterModel(new(models.Order))
	orm.RegisterModel(new(models.Symbols))
	orm.RegisterModel(new(models.NewSymbols))
	orm.RegisterModel(new(models.NoticeSymbols))
	orm.RegisterModel(new(models.ListenSymbols))
	orm.RegisterModel(new(models.SymbolFundingRates))
	orm.RegisterModel(new(models.EatRateSymbols))
	orm.RegisterModel(new(models.StrategyTemplates))
	
	orm.RegisterDriver("sqlite3", orm.DRSqlite)
	orm.RegisterDataBase("default", "sqlite3", "./db/coin.db")
	// orm.Debug = true
}

func registerMiddlewares() {
    web.InsertFilter("*", web.BeforeRouter, middlewares.CorsMiddleware)
    web.InsertFilter("*", web.BeforeRouter, middlewares.JwtMiddleware)
}

func main() {
	// debug
	if debug == "1" {
		// feature.GoTestApi()
		// spot.TryRush()
		// feature.GoTestFeature()
		// feature.UpdateSymbolsFundingRates()
		// feature.ListenCoinFundingRate()
		// feature.GoTestOrder()
		// feature.GoTestUtil()
		// feature.GoTestMarketOrder()
		// feature.GoTestNotify()
		// feature.GoTestDeliveryAccount()
		// feature.GoTestParse()
		// feature.GoTestListen()
		// feature.GoTestLine()
		
		// web
		web.Run(":" + webPort)
	}
	
	// 更新币种交易精度
	go func() {
		logs.Info("update symbols trade precision and add new symbols, every 12 hours")
		for {
			feature.UpdateSymbolsTradePrecision()
			time.Sleep(12 * time.Hour) // 12小时更新一次
		}
	}()
	
	// websocket 订阅更新币种价格
	go func() {
		logs.Info("websocket start: auto update symbols price")
		binance.UpdateCoinByWs()
	}()
	
	// 自动合约交易
	go func() {
		for {
			feature.StartTrade()
			time.Sleep(time.Second * 1) // 1秒间隔
		}
	}()
	
	// 轮训测试所有合约交易的币种策略(没轮10个)
	go func() {
		for {
			feature.NoticeAllSymbolByStrategy()
			time.Sleep(time.Second * 1) // 1秒间隔
		}
	}()
	
	// 新币抢购
	go func() {
		for {
			spot.TryRush()
			time.Sleep(time.Millisecond * 100) // 0.1 秒间隔
		}
	}()
	
	// 新币合约抢购
	go func() {
		for {
			feature.TryRush()
			time.Sleep(time.Millisecond * 100) // 0.1 秒间隔
		}
	}()
	
	// 币种通知
	go func() {
		for {
			spot.NoticeAndAutoOrder()
			feature.NoticeAndAutoOrder()

			time.Sleep(time.Second * 3) // 3 秒间隔
		}
	}()
	
	// 行情监听
	go func() {
		for {
			spot.ListenCoin()
			feature.ListenCoin()

			time.Sleep(time.Second * 3) // 3 秒间隔
		}
	}()
	
	// 合约费率监听
	go func() {
		for {
			// 更新所有币种的资金费率
			feature.UpdateSymbolsFundingRates()
			// 监听费率报警信息
			feature.ListenCoinFundingRate()

			time.Sleep(time.Second * 30) // 30 秒更新一次
		}
	}()
	
	// 监听套利情况
	go func() {
		for {
			rate.ListenRateEat()
			time.Sleep(time.Hour * 1) // 1 小时更新一次
		}	
	}()
	
	// web
	web.Run(":" + webPort)
}



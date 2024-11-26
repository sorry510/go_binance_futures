package main

import (
	"fmt"
	"go_binance_futures/feature"
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/middlewares"
	"go_binance_futures/models"
	"go_binance_futures/rate"
	_ "go_binance_futures/routers"
	"go_binance_futures/spot"
	spot_api "go_binance_futures/spot/api/binance"
	"go_binance_futures/utils"
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
var dbPath, _ = config.String("database::path")
var SystemConfig models.Config

func init() {
	config.Set("system_start_time", fmt.Sprintf("%d", time.Now().Unix() * 1000))
	web.BConfig.CopyRequestBody = true // post 参数
	web.SetStaticPath("/" + webIndex, "static") // 设置静态文件
	logs.Info("server web page:", "http://localhost:" + webPort + "/" + webIndex + "/index.html")
	
	registerModels() // 注册模型
	registerMiddlewares() // 添加中间件
	updateSystemConfig() // 更新配置
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
	orm.RegisterModel(new(models.TestStrategyResults))
	orm.RegisterModel(new(models.SpotSymbols))
	orm.RegisterModel(new(models.DeliverySymbols))
	
	orm.RegisterDriver("sqlite3", orm.DRSqlite)
	orm.RegisterDataBase("default", "sqlite3", dbPath) // WAL 模式允许多个读操作和写操作并发进行，而不会互相阻塞，busy_timeout 参数来增加 SQLite 在遇到锁定时的等待时间
	// orm.Debug = true
}

func registerMiddlewares() {
    web.InsertFilter("*", web.BeforeRouter, middlewares.CorsMiddleware)
    web.InsertFilter("*", web.BeforeRouter, middlewares.JwtMiddleware)
}

func updateSystemConfig() {
	systemConfig, err := utils.GetSystemConfig()
	if err != nil {
		logs.Error("GetSystemConfig:", err.Error())
	} else {
		SystemConfig = systemConfig // 更新配置信息
	}
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
		// feature.CheckTestResults()
		// web
		web.Run(":" + webPort)
	}
	
	/*******************************************更新基本信息 start****************************************************/
	// 读取最新配置信息
	go func() {
		for {
			updateSystemConfig()
			time.Sleep(time.Second * 1) // 1秒间隔
		}
	}()
	// 自动追加币种 和 更新币种交易精度
	go func() {
		for {
			logs.Info("update symbols trade precision and add new symbols, every 12 hours")
			feature.UpdateSymbolsTradePrecision() // u本位
			feature.UpdateDeliverySymbolsTradePrecision() // 币本位
			spot.UpdateSymbolsTradePrecision() // 现货
			time.Sleep(12 * time.Hour) // 12小时更新一次
		}
	}()
	
	// websocket 订阅更新币种价格
	go func() {
		logs.Info("futures websocket start: auto update symbols price")
		binance.UpdateCoinByWs(&SystemConfig)
	}()
	go func() {
		logs.Info("spot websocket start: auto update symbols price")
		return
		spot_api.UpdateCoinByWs(&SystemConfig)
	}()
	go func() {
		logs.Info("delivery websocket start: auto update symbols price")
		return
		binance.UpdateDeliveryCoinByWs(&SystemConfig)
	}()
	/*******************************************更新基本信息 end****************************************************/
	
	// 仓位正负转换通知
	go func() {
		for {
			feature.PositionConvertNotice(SystemConfig)
			time.Sleep(time.Second * 5) // 5秒间隔
		}
	}()
	
	// 自动合约交易
	go func() {
		for {
			feature.StartTrade(SystemConfig)
			time.Sleep(time.Second * 1) // 1秒间隔
		}
	}()
	
	// 轮训测试所有开启合约交易的币种策略(每轮10个)
	go func() {
		for {
			feature.NoticeAllSymbolByStrategy(SystemConfig)
			time.Sleep(time.Second * 1) // 1秒间隔
		}
	}()
	// 监听测试的开仓是否需要平仓
	go func() {
		for {
			feature.CheckTestResults(SystemConfig)
			time.Sleep(time.Second * 1) // 1秒间隔
		}
	}()
	
	// 新币抢购
	go func() {
		for {
			spot.TryRush(SystemConfig)
			time.Sleep(time.Millisecond * 100) // 0.1 秒间隔
		}
	}()
	
	// 新币合约抢购
	go func() {
		for {
			feature.TryRush(SystemConfig)
			time.Sleep(time.Millisecond * 100) // 0.1 秒间隔
		}
	}()
	
	// 币种通知
	go func() {
		for {
			spot.NoticeAndAutoOrder(SystemConfig)
			feature.NoticeAndAutoOrder(SystemConfig)

			time.Sleep(time.Second * 3) // 3 秒间隔
		}
	}()
	
	// 行情监听
	go func() {
		for {
			spot.ListenCoin(SystemConfig)
			feature.ListenCoin(SystemConfig)

			time.Sleep(time.Second * 3) // 3 秒间隔
		}
	}()
	
	// 合约费率监听
	go func() {
		for {
			// 更新所有币种的资金费率
			feature.UpdateSymbolsFundingRates(SystemConfig)
			// 监听费率报警信息
			feature.ListenCoinFundingRate(SystemConfig)

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



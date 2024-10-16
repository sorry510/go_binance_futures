package main

import (
	"go_binance_futures/feature"
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/middlewares"
	"go_binance_futures/models"
	_ "go_binance_futures/routers"
	"go_binance_futures/spot"
	"time"

	"strconv"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/config"
	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
	_ "github.com/mattn/go-sqlite3"
)

var debug, _ = config.String("debug")
var webPort, _ = config.String("web::port")
var webIndex, _ = config.String("web::index")
var taskSleepTime, _ = config.String("coin::sleep_time")
var tradeEnable, _ = config.String("trade::future_enable")
var tradeNewEnable, _ = config.String("trade::new_enable")
var spotNewEnable, _ = config.String("spot::new_enable")
var noticeCoinEnable, _ = config.String("notice_coin::enable")
var listenCoinEnable, _ = config.String("listen_coin::enable")
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
	orm.RegisterModel(new(models.NewSymbols))
	orm.RegisterModel(new(models.NoticeSymbols))
	orm.RegisterModel(new(models.ListenSymbols))
	orm.RegisterModel(new(models.SymbolFundingRates))
	
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
		feature.GoTestLine()
		// feature.UpdateSymbolsFundingRates()
		// feature.ListenCoinFundingRate()
		// feature.GoTestOrder()
		// feature.GoTestUtil()
		// feature.GoTestMarketOrder()
		// feature.GoTestNotify()
		return
	}
	
	// 更新币种交易精度
	go func() {
		logs.Info("更新币种交易精度和增加新币")
		for {
			feature.UpdateSymbolsTradePrecision()
			time.Sleep(12 * time.Hour) // 12小时更新一次
		}
	}()
	
	// websocket 订阅更新币种价格
	go func() {
		logs.Info("websocket 自动更新币种最新价格")
		binance.UpdateCoinByWs()
	}()
	
	// 自动合约交易
	if tradeEnable == "1" {
		logs.Info("自动合约交易开启:", tradeEnable)
		// trade script
		go func() {
			for {
				feature.StartTrade()

				// 等待 taskSleepTimeInt 秒再继续执行
				time.Sleep(time.Duration(taskSleepTimeInt) * time.Second)
			}
		}()
	}
	
	// 新币抢购
	if spotNewEnable == "1" {
		logs.Info("新币抢购开启")
		go func() {
			for {
				spot.TryRush()

				time.Sleep(time.Millisecond * 100) // 0.1 秒间隔
			}
		}()
	}
	
	// 新币合约抢购
	if tradeNewEnable == "1" {
		logs.Info("合约新币抢购开启")
		go func() {
			for {
				feature.TryRush()

				time.Sleep(time.Millisecond * 100) // 0.1 秒间隔
			}
		}()
	}
	
	// 币种通知
	if noticeCoinEnable == "1" {
		logs.Info("币种通知开启")
		go func() {
			for {
				spot.NoticeAndAutoOrder()
				feature.NoticeAndAutoOrder()

				time.Sleep(time.Second * 5) // 5 秒间隔
			}
		}()
	}
	
	// 行情监听
	if listenCoinEnable == "1" {
		logs.Info("行情监听通知开启")
		go func() {
			for {
				// 更新所有币种的资金费率
				feature.UpdateSymbolsFundingRates()
				// 监听费率报警信息
				feature.ListenCoinFundingRate()

				time.Sleep(time.Second * 60 * 10) // 10 分钟间隔
			}
		}()
		go func() {
			for {
				spot.ListenCoin()
				feature.ListenCoin()

				time.Sleep(time.Second * 5) // 5 秒间隔
			}
		}()
	}
	
	// web
	web.Run(":" + webPort)
}



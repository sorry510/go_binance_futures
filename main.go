package main

import (
	"fmt"
	"go_binance_futures/command"
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
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

var dbVersion int64 = 8 // 每次变动数据库版本号 +1
var debug, _ = config.String("debug")
var webPort, _ = config.String("web::port")
var webIndex, _ = config.String("web::index")
var driver, _ = config.String("database::driver")
var dbPath, _ = config.String("database::path")
var username, _ = config.String("database::username")
var password, _ = config.String("database::password")
var host, _ = config.String("database::host")
var port, _ = config.String("database::port")
var dbname, _ = config.String("database::dbname")
var wsFuturesUserData, _ = config.String("ws::futures_user_data")
var SystemConfig models.Config

func init() {
	config.Set("system_start_time", fmt.Sprintf("%d", time.Now().Unix() * 1000))
	web.BConfig.CopyRequestBody = true // post 参数
	web.SetStaticPath("/" + webIndex, "static") // 设置静态文件
	logs.Info("server web page:", "http://localhost:" + webPort + "/" + webIndex + "/index.html")
	
	registerModels() // 注册模型
	registerMiddlewares() // 添加中间件
}

func setDriver(d string)  {
	logs.Info("use database driver:", d)
	switch d {
		case "sqlite":
			orm.RegisterDriver(d, orm.DRSqlite)
			orm.RegisterDataBase("default", "sqlite3", dbPath) // WAL 模式允许多个读操作和写操作并发进行，而不会互相阻塞，busy_timeout 参数来增加 SQLite 在遇到锁定时的等待时间
		case "mysql":
			orm.RegisterDriver(d, orm.DRMySQL)
			orm.RegisterDataBase("default", "mysql", username + ":" + password + "@tcp(" + host + ":" + port + ")/" + dbname + "?charset=utf8")
		case "postgres":
			orm.RegisterDriver(d, orm.DRPostgres)
			orm.RegisterDataBase("default", "postgres", "user=" + username + " password=" + password + " host=" + host + " port=" + port + " dbname=" + dbname + " sslmode=disable")
		default:
			orm.RegisterDriver(d, orm.DRSqlite)
			orm.RegisterDataBase("default", "sqlite3", dbPath)
	}
}

func registerModels() {
	// orm.Debug = true
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
	orm.RegisterModel(new(models.FuturesPosition))
	orm.RegisterModel(new(models.FuturesOrder))
	orm.RegisterModel(new(models.NotifyConfig))
	
	setDriver(driver) // 设置数据库驱动
	syncDb() // 同步数据库
}

func syncDb() {
	config, err := utils.GetSystemConfig()
	if err != nil {
		logs.Info("database file does not exist, create and initialize")
		orm.RunSyncdb("default", false, false) // 根据 model 创建数据表
		command.InitData(dbVersion) // 初始化配置信息
		config, err = utils.GetSystemConfig() // 重新获取配置信息
		if err != nil {
			logs.Error("get system config error", err)
			return
		}
	}
	// 根据旧版本更新数据库
	orm.RunSyncdb("default", false, false) // 根据 model 更新新创建的数据表
	oldVersion := config.Version
	if oldVersion < dbVersion {
		err = command.UpdateDatabase(oldVersion, dbVersion)
		if err != nil {
			logs.Error("!!! update database error !!!:", err)
		} else {
			logs.Info("@@@ update database success @@@")
		}
		config.Version = dbVersion
		orm.NewOrm().Update(&config)
	}
	SystemConfig = config
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
	// 设置市场行情趋势
	config.Set("MarketCondition", fmt.Sprintf("%d", systemConfig.MarketCondition))
}

func main() {
	// debug
	if debug == "1" {
		updateSystemConfig()
		// feature.UpdateOrderStatus()
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
		// feature.CheckTestResults(SystemConfig)
		// command.ExecSqlFile("./command/sql/test_strategy_results.sql")
		// feature.AutoLossScale(SystemConfig, true)
		// web
		web.Run(":" + webPort)
		return
	}
	
	/*******************************************更新基本信息 start****************************************************/
	// ws 订阅用户数据信息(仓位,当前挂单)
	// 如果开启，则使用本地数据库管理仓位信息，不再每次请求查询 api 接口，可以有效降低请求频率(openOrders, getPosition) 
	// 但是需要注意，这里面的仓位信息推送，只有仓位发生变化时才会推送数据(当前仓位的盈利多少变化不会推送，需要根据 symbols 表的 close 价格计算)
	if wsFuturesUserData == "1" {
		feature.SyncUserData()	
	}
	
	// 读取最新配置信息
	go func() {
		for {
			updateSystemConfig()
			time.Sleep(time.Second * 1) // 1秒间隔
		}
	}()
	// 更新当前行情趋势
	go func() {
		for {
			feature.UpdateMarketCondition(&SystemConfig)
			time.Sleep(time.Minute * 3) // 3分钟更新一次
		}
	}()
	
	// 自动追加币种 和 更新币种交易精度
	go func() {
		for {
			logs.Info("update symbols trade precision and add new symbols, every 12 hours")
			feature.UpdateSymbolsTradePrecision() // u本位
			// spot.UpdateSymbolsTradePrecision() // 现货
			// feature.UpdateDeliverySymbolsTradePrecision() // 币本位
			time.Sleep(12 * time.Hour) // 12小时更新一次
		}
	}()
	
	// websocket 订阅更新币种价格
	go func() {
		logs.Info("futures websocket start: auto update symbols price")
		binance.UpdateCoinByWs(&SystemConfig, 0)
	}()
	go func() {
		return
		logs.Info("spot websocket start: auto update symbols price")
		spot_api.UpdateCoinByWs(&SystemConfig, 0)
	}()
	go func() {
		return
		logs.Info("delivery websocket start: auto update symbols price")
		binance.UpdateDeliveryCoinByWs(&SystemConfig)
	}()
	
	/*******************************************更新基本信息 end****************************************************/
	
	// 仓位正负转换通知
	go func() {
		for {
			feature.PositionConvertNotice(&SystemConfig)
			time.Sleep(time.Second * 10) // 10秒间隔
		}
	}()
	
	// 自动合约交易
	go func() {
		for {
			feature.StartTrade(SystemConfig)
			time.Sleep(time.Second * 2) // 2秒间隔, 1min 中不能超过 2400 权重和
		}
	}()

	// 更新一次订单状态, 一次 100 条，平仓时也会更新对应订单的状态，此处是兜底行为
	// TODO: 删掉那些很久一起的平仓的订单
	go func() {
		for {
			feature.UpdateOrderStatus()
			time.Sleep(time.Minute * 30) // 30分钟更新一次订单状态
		}
	}()

	/*******************************************测试自定义策略 start**********************************************************/
	// 轮训测试所有开启合约交易的币种策略(每轮5个)
	go func() {
		for {
			feature.NoticeAllSymbolByStrategy(SystemConfig)
			time.Sleep(time.Millisecond * 1500) // 1.5秒间隔
		}
	}()
	// 监听测试的开仓是否需要平仓
	go func() {
		for {
			feature.CheckTestResults(SystemConfig)
			time.Sleep(time.Millisecond * 1500) // 1.5秒间隔
		}
	}()
	/*******************************************测试自定义策略 end**********************************************************/
	
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

			time.Sleep(time.Second * 2) // 2 秒间隔
		}
	}()
	
	// 行情监听
	go func() {
		for {
			spot.ListenCoin(SystemConfig)
			feature.ListenCoin(SystemConfig)

			time.Sleep(time.Second * 2) // 3 秒间隔
		}
	}()
	
	// 合约费率监听
	go func() {
		for {
			// 更新所有币种的资金费率
			feature.UpdateSymbolsFundingRates(SystemConfig)
			// 监听费率报警信息
			feature.ListenCoinFundingRate(SystemConfig)

			time.Sleep(time.Second * 90) // 90 秒更新一次
		}
	}()
	
	// 监听套利情况
	go func() {
		return
		for {
			rate.ListenRateEat()
			time.Sleep(time.Hour * 1) // 1 小时更新一次
		}	
	}()
	
	// web
	web.Run(":" + webPort)
}



package main

import (
	"context"
	"fmt"
	agentapp "go_binance_futures/agent/app"
	"go_binance_futures/command"
	"go_binance_futures/feature"
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/llm"
	"go_binance_futures/mcpserver"
	"go_binance_futures/middlewares"
	"go_binance_futures/models"
	_ "go_binance_futures/routers"
	alertpipeline "go_binance_futures/service/alertpipeline"
	"go_binance_futures/spot"
	spot_api "go_binance_futures/spot/api/binance"
	"go_binance_futures/utils"
	"go_binance_futures/webnotification"
	"sync/atomic"
	"time"

	// Loads the global config before the package-level reads below run.
	_ "go_binance_futures/bootstrap"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/config"
	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

var dbVersion int64 = 1 // 每次变动数据库版本号 +1
var debug, _ = config.String("debug")
var webPort, _ = config.String("web::port")
var webIndex, _ = config.String("web::index") // 如果不是 zmkm, 前端项目需要修改 api 请求地址，增加 /zmkm 前缀
var driver, _ = config.String("database::driver")
var dbPath, _ = config.String("database::path")
var username, _ = config.String("database::username")
var password, _ = config.String("database::password")
var host, _ = config.String("database::host")
var port, _ = config.String("database::port")
var dbname, _ = config.String("database::dbname")
var wsFuturesUserData, _ = config.String("ws::futures_user_data")
var tradeKey, _ = config.String("binance::api_key")
var mcpServerEnable, _ = config.Bool("mcp::mcp_server_enable")
var SystemConfig models.Config

func init() {
	initLogger() // 初始化日志
	config.Set("system_start_time", fmt.Sprintf("%d", time.Now().Unix()*1000))
	web.BConfig.CopyRequestBody = true        // post 参数
	web.SetStaticPath("/"+webIndex, "static") // 设置静态文件

	registerModels()      // 注册模型
	registerMiddlewares() // 添加中间件
	if mcpServerEnable {
		registerMCPHTTP() // 注册 MCP HTTP 服务
	}
}

func initLogger() {
	// 设置日志级别
	level, _ := config.Int("log_level")
	logs.SetLevel(level)
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
	orm.RegisterModel(new(models.FuturesLiquidationOrder))
	orm.RegisterModel(new(models.SymbolAnalysisHistory))
	orm.RegisterModel(new(models.AgentTask))
	orm.RegisterModel(new(models.AgentTaskEvent))
	orm.RegisterModel(new(models.AgentAlertPipelineTrace))
	orm.RegisterModel(new(models.AgentConversation))
	orm.RegisterModel(new(models.AgentConversationMessage))
	orm.RegisterModel(new(models.AgentSkill))
	orm.RegisterModel(new(models.AgentMCPServer))
	orm.RegisterModel(new(models.AgentMCPTool))
	orm.RegisterModel(new(models.AgentMCPResource))
	orm.RegisterModel(new(models.AgentMCPPrompt))
	orm.RegisterModel(new(models.AgentMCPPermission))
	orm.RegisterModel(new(models.AgentMCPSecret))
	orm.RegisterModel(new(models.AgentMCPOAuthState))
	orm.RegisterModel(new(models.LLMConfig))
	orm.RegisterModel(new(models.Notification))

	setDriver(driver) // 设置数据库驱动
	syncDb()          // 同步数据库
	if err := agentapp.EnsureDefaultSkillConfigs(); err != nil {
		logs.Error("initialize agent skill configs:", err)
	}
	if err := llm.EnsureDatabaseConfigFromLegacy(context.Background()); err != nil {
		logs.Error("initialize LLM database config:", err)
	}
	if err := feature.BackfillEmptyFuturesSymbolTypes(); err != nil {
		logs.Error("backfill empty futures symbol types error:", err)
	}
}

func setDriver(d string) {
	logs.Info("use database driver:", d)
	switch d {
	case "sqlite":
		orm.RegisterDriver(d, orm.DRSqlite)
		orm.RegisterDataBase("default", "sqlite3", dbPath) // WAL 模式允许多个读操作和写操作并发进行，而不会互相阻塞，busy_timeout 参数来增加 SQLite 在遇到锁定时的等待时间
	case "mysql":
		orm.RegisterDriver(d, orm.DRMySQL)
		connectTimeout := databaseDuration("connect_timeout_seconds", 5*time.Second)
		readTimeout := databaseDuration("read_timeout_seconds", 30*time.Second)
		writeTimeout := databaseDuration("write_timeout_seconds", 30*time.Second)
		dsn := username + ":" + password + "@tcp(" + host + ":" + port + ")/" + dbname +
			"?charset=utf8mb4&collation=utf8mb4_0900_ai_ci&timeout=" + connectTimeout.String() +
			"&readTimeout=" + readTimeout.String() + "&writeTimeout=" + writeTimeout.String()
		if err := orm.RegisterDataBase("default", "mysql", dsn); err != nil {
			panic(fmt.Errorf("register mysql database: %w", err))
		}
		configureDatabasePool()
	case "postgres":
		orm.RegisterDriver(d, orm.DRPostgres)
		orm.RegisterDataBase("default", "postgres", "user="+username+" password="+password+" host="+host+" port="+port+" dbname="+dbname+" sslmode=disable")
	default:
		orm.RegisterDriver(d, orm.DRSqlite)
		orm.RegisterDataBase("default", "sqlite3", dbPath)
	}
}

func databaseInt(name string, fallback int) int {
	value, err := config.Int("database::" + name)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func databaseDuration(name string, fallback time.Duration) time.Duration {
	value := databaseInt(name, int(fallback/time.Second))
	return time.Duration(value) * time.Second
}

func configureDatabasePool() {
	db, err := orm.GetDB("default")
	if err != nil {
		logs.Error("get database pool:", err)
		return
	}
	db.SetMaxOpenConns(databaseInt("max_open_conns", 30))
	db.SetMaxIdleConns(databaseInt("max_idle_conns", 10))
	db.SetConnMaxLifetime(databaseDuration("conn_max_lifetime_seconds", 5*time.Minute))
	db.SetConnMaxIdleTime(databaseDuration("conn_max_idle_time_seconds", time.Minute))
}

func syncDb() {
	config, err := utils.GetSystemConfig()
	hasSync := false
	if err != nil {
		logs.Info("The config table does not exist or has changed, it is automatically updating...")
		if syncErr := orm.RunSyncdb("default", false, false); syncErr != nil { // 根据 model 创建数据表
			logs.Error("sync database schema error:", syncErr)
			panic(syncErr)
		}
		hasSync = true
		command.InitData(0)                   // 初始化配置信息
		config, err = utils.GetSystemConfig() // 重新获取配置信息
		if err != nil {
			logs.Error("get system config error", err)
			return
		}
	}
	// 根据旧版本更新数据库
	if !hasSync {
		if syncErr := orm.RunSyncdb("default", false, false); syncErr != nil { // 根据 model 更新新创建的数据表
			logs.Error("sync database schema error:", syncErr)
			panic(syncErr)
		}
	}
	oldVersion := config.Version
	if oldVersion < dbVersion {
		err = command.UpdateDatabase(oldVersion, dbVersion)
		if err != nil {
			logs.Error("!!! update database error !!!:", err)
			panic(err)
		} else {
			logs.Info("@@@ update database success @@@")
			config.Version = dbVersion
			orm.NewOrm().Update(&config)
		}
	}
	SystemConfig = config
}

func registerMiddlewares() {
	web.InsertFilter("*", web.BeforeRouter, middlewares.CorsMiddleware)
	web.InsertFilter("*", web.BeforeRouter, middlewares.JwtMiddleware)
}

func registerMCPHTTP() {
	web.Handler(mcpserver.DefaultHTTPPath, mcpserver.NewHTTPHandler(mcpserver.NewServer(), mcpserver.HTTPOptions{
		JSONResponse: true,
		Stateless:    false,
	}))
	logs.Info("mcp http endpoint:", "http://localhost:"+webPort+mcpserver.DefaultHTTPPath)
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
	if err := alertpipeline.StartDefault(context.Background(), func() models.Config { return SystemConfig }); err != nil {
		logs.Error("start alert pipeline:", err)
	}
	// debug
	if debug == "1" {
		updateSystemConfig()
		go binance.UpdateCoinByWs(&SystemConfig, 0)
		// web
		logs.Info("server web index:", "http://localhost:"+webPort+"/"+webIndex+"/index.html")
		web.Run(":" + webPort)
		return
	}

	/*******************************************更新基本信息 start****************************************************/
	// ws 订阅用户数据信息(仓位,当前挂单)
	// 如果开启，则使用本地数据库管理仓位信息，不再每次请求查询 api 接口，可以有效降低请求频率(openOrders, getPosition)
	// 但是需要注意，这里面的仓位信息推送，只有仓位发生变化时才会推送数据(当前仓位的盈利多少变化不会推送，需要根据 symbols 表的 close 价格计算)
	if wsFuturesUserData == "1" && tradeKey != "" {
		feature.SyncUserData()
	}

	// 读取最新配置信息
	loopRun(updateSystemConfig, time.Second*2) // 每 2 秒更新一次系统配置信息
	// Agent 周期任务统一由 Scheduler 触发。
	if err := agentapp.StartDefaultScheduler(context.Background(), func() models.Config { return SystemConfig }); err != nil {
		logs.Error("start agent scheduler:", err)
	}
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
	go binance.UpdateCoinByWs(&SystemConfig, 0)
	// websocket 订阅全市场强平订单
	go binance.CollectFuturesLiquidationOrders(&SystemConfig)

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
		feature.PositionConvertNotice(&SystemConfig)
		ticker := time.NewTicker(time.Second * 10)
		defer ticker.Stop()
		for range ticker.C {
			feature.PositionConvertNotice(&SystemConfig)
		}
	}()

	// 自动合约交易
	loopRun(func() {
		feature.StartTrade(&SystemConfig)
	}, time.Second*2) // 2秒间隔, 1min 中不能超过 2400 权重和

	// 30 分钟检查一次所有未平仓的订单, 一次 200 条，此处是兜底行为，处理一些意外情况
	// 处理 app 上已经平仓的订单，但是系统中没有找到对应的平仓订单
	loopRun(func() {
		feature.UpdateOrderStatus()
	}, time.Minute*30) // 30分钟更新一次订单状态

	/*******************************************测试自定义策略 start**********************************************************/
	// 轮训测试所有开启合约交易的币种策略(每轮5个)
	loopRun(func() {
		feature.NoticeAllSymbolByStrategy(&SystemConfig)
	}, time.Millisecond*1500) // 1.5秒间隔, 避免请求过快
	// 监听测试的开仓是否需要平仓
	loopRun(func() {
		feature.CheckTestResults(&SystemConfig)
	}, time.Millisecond*1500) // 1.5秒间隔
	/*******************************************测试自定义策略 end**********************************************************/

	// 新币抢购
	loopRun(func() {
		spot.TryRush(SystemConfig)
	}, time.Millisecond*100) // 0.1 秒间隔

	// 新币合约抢购
	loopRun(func() {
		feature.TryRush(SystemConfig)
	}, time.Millisecond*100) // 0.1 秒间隔

	// 币种通知
	loopRun(func() {
		spot.NoticeAndAutoOrder(SystemConfig)
		feature.NoticeAndAutoOrder(SystemConfig)
	}, time.Second*2)

	// 行情监听
	loopRun(func() {
		feature.ListenCoin(SystemConfig)
		spot.ListenCoin(SystemConfig)
	}, time.Second*2)

	// 合约费率监听
	loopRun(func() {
		// 更新所有币种的资金费率
		feature.UpdateSymbolsFundingRates(SystemConfig)
		// 监听费率报警信息
		feature.ListenCoinFundingRate(SystemConfig)
	}, time.Second*120) // 120 秒更新一次

	// 监听套利情况
	// go func() {
	// 	return
	// 	for {
	// 		rate.ListenRateEat()
	// 		time.Sleep(time.Hour * 1) // 1 小时更新一次
	// 	}
	// }()

	// 合约强平订单每日清理，删除 10 天前的数据
	go binance.StartFuturesLiquidationOrderCleanupTask()
	// 清理过期消息通知
	go webnotification.StartNotificationCleanupTask()

	// web
	logs.Info("server web index:", "http://localhost:"+webPort+"/"+webIndex+"/index.html")
	web.Run(":" + webPort)
}

func loopRun(callback func(), d time.Duration) {
	go func() {
		var running int32
		ticker := time.NewTicker(d)
		for range ticker.C {
			if atomic.CompareAndSwapInt32(&running, 0, 1) {
				go func() {
					defer atomic.StoreInt32(&running, 0)
					callback()
				}()
			}
		}
	}()
}

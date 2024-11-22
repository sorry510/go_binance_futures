package feature

import (
	"encoding/json"
	"go_binance_futures/feature/strategy/line"
	"go_binance_futures/lang"
	"go_binance_futures/models"
	"go_binance_futures/notify"
	"go_binance_futures/technology"
	"go_binance_futures/utils"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
	"github.com/expr-lang/expr"
)

var coinNoticeLastTimeMap = make(map[string]int64) // limit 通知一次
var FuturesTestNotice = 0
var offsetId = 0
func NoticeAllSymbolByStrategy() {
	systemConfig, err := utils.GetSystemConfig()
	if err != nil {
		logs.Error("GetSystemConfig:", err.Error())
		return
	}
	if (systemConfig.FutureTest == 1) {
		if (FuturesTestNotice == 0) {
			logs.Info("futures all symbol notice_strategy bot start")
			FuturesTestNotice = 1
		}
	} else {
		if (flagFuturesNotice == 1) {
			logs.Info("futures all symbol notice_strategy bot end")
			FuturesTestNotice = 0
		}
		return
	}
	
	logs.Info("offsetId: ", offsetId)
	var coins []*models.Symbols
	coins, err = getSymbols(offsetId) // 按照顺序 10 个币
	if err != nil {
		logs.Error("NoticeAllSymbolByStrategy:", err.Error())
		return
	}
	if len(coins) == 0 {
		offsetId = 0
		coins, _ = getSymbols(offsetId)
	}
	if (len(coins) > 0) {
		offsetId = int(coins[len(coins) - 1].ID)
	} else {
		offsetId += 10 // 避免无限处于循环
	}
	
	for _, coin := range coins {
		nowTime := time.Now().Unix() * 1000 // 毫秒时间戳
		if coin.Enable != 1 {
			continue
		}
		
		lastNoticeTime, exist := coinNoticeLastTimeMap[coin.Symbol]
		if exist {
			if (nowTime - lastNoticeTime) < int64(systemConfig.FutureTestNoticeLimitMin) * 60 * 1000 {
				// x min 通知一次
				continue
			}
		}
		if coin.Technology == "" || coin.Strategy == "" {
			logs.Info("no set custom strategy, symbol: ", coin.Symbol)
			continue
		}
		
		var strategyConfig technology.StrategyConfig
		err := json.Unmarshal([]byte(coin.Strategy), &strategyConfig)
		if err != nil {
			logs.Error("Error unmarshalling JSON Symbol: ", coin.Symbol)
			logs.Error("Error unmarshalling JSON:", err.Error())
			continue
		}
		logs.Info("futures custom strategy test, symbol: ", coin.Symbol)
		env := line.InitParseEnv(coin.Symbol, coin.Technology)
		for _, strategy := range strategyConfig {
			if strategy.Enable && (strategy.Type == "long" || strategy.Type == "short") {
				program, err := expr.Compile(strategy.Code, expr.Env(env))
				if err != nil {
					logs.Error("Error Strategy Compile Symbol: ", coin.Symbol)
					logs.Error("Error Strategy Compile:", err.Error())
					continue
				}
				output, err := expr.Run(program, env)
				if err != nil {
					logs.Error("Error Strategy Run Symbol: ", coin.Symbol)
					logs.Error("Error Strategy Run:", err.Error())
					continue
				}
				if res, ok := output.(bool); ok && res {
					floatNowPrice := env["NowPrice"].(float64)
					testOrder := createTestResult(coin, floatNowPrice, strings.ToUpper(strategy.Type), strategy.Code)
					quantity, _ := strconv.ParseFloat(testOrder.PositionAmt, 64)
					pusher.FuturesCustomStrategyTest(notify.FuturesTestParams{
						Title: lang.Lang("futures.custom_strategy_test"),
						Symbol: coin.Symbol,
						Type: "open", // 开仓
						PositionSide: strategy.Type,
						Price: floatNowPrice,
						ClosePrice: 0.0, // 开仓时无平仓价格
						Quantity: quantity,
						Leverage: float64(coin.Leverage),
						Profit: 0.0, // 开仓时无盈利
						StrategyName: strategy.Name,
						Remarks: strategy.Code,
					})
					coinNoticeLastTimeMap[coin.Symbol] = nowTime
				}
			}
		}
	}
}

// 检查测试中的币当前是否应该平仓
func CheckTestResults() {
	systemConfig, err := utils.GetSystemConfig()
	if err != nil {
		logs.Error("GetSystemConfig:", err.Error())
		return
	}
	if (systemConfig.FutureTest == 1) {
		if (FuturesTestNotice == 0) {
			logs.Info("futures all symbol notice_strategy bot start")
			FuturesTestNotice = 1
		}
	} else {
		if (flagFuturesNotice == 1) {
			logs.Info("futures all symbol notice_strategy bot end")
			FuturesTestNotice = 0
		}
		return
	}
	
	var results []*models.TestStrategyResults
	_, err = orm.NewOrm().QueryTable("test_strategy_results").Filter("close_price", "0").All(&results)
	if err != nil {
		logs.Error("get test_strategy_results error:", err.Error())
		return
	}
	for _, result := range results {
		var strategyConfig technology.StrategyConfig
		err := json.Unmarshal([]byte(result.Strategy), &strategyConfig)
		if err != nil {
			logs.Error("Error unmarshalling JSON Symbol: ", result.Symbol)
			logs.Error("Error unmarshalling JSON:", err.Error())
			continue
		}
		
		findStrategy := false
		env := line.InitParseEnv(result.Symbol, result.Technology)
		floatNowPrice := env["NowPrice"].(float64)
		positionAmtFloat, _ := strconv.ParseFloat(result.PositionAmt, 64)
		positionAmtFloatAbs := math.Abs(positionAmtFloat) // 空单为负数,纠正为绝对值
		enterPrice_float64, _ := strconv.ParseFloat(result.Price, 64)
		unRealizedProfit := (floatNowPrice - enterPrice_float64) * positionAmtFloat // 未实现盈亏
		nowProfit := (unRealizedProfit / (positionAmtFloatAbs * floatNowPrice)) * float64(result.Leverage) * 100
		
		// 模拟仓位信息
		position := futures.PositionRisk{
			EntryPrice: result.Price, // 开仓价格
			MarkPrice: strconv.FormatFloat(floatNowPrice, 'f', -1, 64), // 当前标记价格
			PositionAmt: result.PositionAmt, // 仓位数量(正数为多仓，负数为空仓)
			UnRealizedProfit: strconv.FormatFloat(unRealizedProfit, 'f', 3, 64), // 未实现盈亏
			MarginType: "CROSSED",
			Leverage: strconv.FormatInt(result.Leverage, 10),
			PositionSide: result.PositionSide,
		}
		
		env["ROI"] = nowProfit // 当前收益率
		env["Position"] = position // 当前仓位信息
		for _, strategy := range strategyConfig {
			if strategy.Enable && (strategy.Type == "close_long" || strategy.Type == "close_short") {
				if strategy.Type == "close_long" && position.PositionSide != "LONG" {
					// 平多仓的策略，当前仓位不是多仓，跳过
					continue
				}
				if strategy.Type == "close_short" && position.PositionSide != "SHORT" {
					// 平空仓的策略，当前仓位不是空仓，跳过
					continue
				}
				
				program, err := expr.Compile(strategy.Code, expr.Env(env))
				if err != nil {
					logs.Error("Error Strategy Compile Symbol: ", result.Symbol)
					logs.Error("Error Strategy Compile:", err.Error())
					continue
				}
				output, err := expr.Run(program, env)
				if err != nil {
					logs.Error("Error Strategy Run Symbol: ", result.Symbol)
					logs.Error("Error Strategy Run:", err.Error())
					continue
				}
				findStrategy = true // 发现有正常能执行的平仓策略
				if res, ok := output.(bool); ok && res {
					result.ClosePrice = position.MarkPrice
					result.CloseProfit = position.UnRealizedProfit
					result.CloseStrategy = strategy.Code
					result.UpdateTime = time.Now().Unix() * 1000
					orm.NewOrm().Update(result)
					// 平仓通知
					quantity, _ := strconv.ParseFloat(result.PositionAmt, 64)
					profitUsdt, _ := strconv.ParseFloat(result.CloseProfit, 64)
					pusher.FuturesCustomStrategyTest(notify.FuturesTestParams{
						Title: lang.Lang("futures.custom_strategy_test"),
						Symbol: result.Symbol,
						Type: "close", // 平仓
						PositionSide: strategy.Type,
						Price: enterPrice_float64, // 开仓价格
						ClosePrice: floatNowPrice, // 平仓价格
						Quantity: quantity,
						Leverage: float64(result.Leverage),
						Profit: profitUsdt, // 平仓盈利
						StrategyName: strategy.Name,
						Remarks: strategy.Code,
					})
					return
				}
			}
		}
		if !findStrategy && (nowProfit > 10 || nowProfit < -10) {
			// 没有定义平仓策略，使用超过 10 % 就平仓
			result.ClosePrice = position.MarkPrice
			result.CloseProfit = position.UnRealizedProfit
			result.UpdateTime = time.Now().Unix() * 1000
			orm.NewOrm().Update(result)
			// 平仓通知
			
			quantity, _ := strconv.ParseFloat(result.PositionAmt, 64)
			profitUsdt, _ := strconv.ParseFloat(result.CloseProfit, 64)
			pusher.FuturesCustomStrategyTest(notify.FuturesTestParams{
				Title: lang.Lang("futures.custom_strategy_test"),
				Symbol: result.Symbol,
				Type: "close", // 平仓
				PositionSide: "close",
				Price: enterPrice_float64, // 开仓价格
				ClosePrice: floatNowPrice, // 平仓价格
				Quantity: quantity,
				Leverage: float64(result.Leverage),
				Profit: profitUsdt, // 平仓盈利
				StrategyName: "no define custom strategy",
				Remarks: "profit or loss > 10%",
			})
		}
	}
}

func getSymbols(offsetId int) (coins []*models.Symbols, err error) {
	_, err = orm.NewOrm().
		QueryTable("symbols").
		Filter("enable", 1).
		Filter("ID__gt", offsetId).
		OrderBy("ID").
		Limit(10).
		All(&coins) // 按照顺序 10个币
	return coins, err
}

// 生成测试的开仓数据
func createTestResult(coin *models.Symbols, nowPrice float64, positionSide string, openStrategyCode string) (result *models.TestStrategyResults) {
	usdt_float64, _ := strconv.ParseFloat(coin.Usdt, 64) // 交易金额
	buyPrice := utils.GetTradePrecision(nowPrice, coin.TickSize) // 合理精度的价格
	quantity := (usdt_float64 / buyPrice) * float64(coin.Leverage) // 购买数量
	quantity = utils.GetTradePrecision(quantity, coin.StepSize) // 合理精度的数量
	if positionSide == "SHORT" {
		quantity = -quantity // 空单
	}
				
	result = new(models.TestStrategyResults)
	result.Symbol = coin.Symbol
	result.Price = strconv.FormatFloat(buyPrice, 'f', -1, 64)
	result.Leverage = coin.Leverage
	result.TickSize = coin.TickSize
	result.StepSize = coin.StepSize
	result.Usdt = coin.Usdt
	result.Profit = coin.Profit
	result.Loss = coin.Loss
	result.PositionAmt = strconv.FormatFloat(quantity, 'f', -1, 64)
	result.PositionSide = positionSide
	result.Technology = coin.Technology
	result.Strategy = coin.Strategy
	result.OpenStrategy = openStrategyCode
	result.ClosePrice = "0"
	result.CloseProfit = "0"
	result.CreateTime = time.Now().Unix() * 1000
	result.UpdateTime = time.Now().Unix() * 1000
	
	o := orm.NewOrm()
	o.Insert(result)
	return result
}

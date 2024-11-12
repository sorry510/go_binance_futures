package line

import (
	"encoding/json"
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/models"
	"go_binance_futures/technology"
	"strconv"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
	"github.com/expr-lang/expr"
)

type TradeLineCustom struct {
}

// 交易逻辑: 自定义逻辑
func (TradeLine TradeLineCustom) GetCanLongOrShort(symbol string) (canLong bool, canShort bool) {
	var coin models.Symbols
	o := orm.NewOrm()
	o.QueryTable("symbols").Filter("Symbol", symbol).One(&coin)
	
	var strategyConfig technology.StrategyConfig
	err := json.Unmarshal([]byte(coin.Strategy), &strategyConfig)
	if err != nil {
		logs.Error("Error unmarshalling JSON Symbol: ", coin.Symbol)
		logs.Error("Error unmarshalling JSON:", err.Error())
		return false, false
	}
	canLong = false
	canShort = false
	env := InitParseEnv(coin.Symbol, coin.Technology)
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
			if output.(bool) {
				if strategy.Type == "long" {
					canLong = true
				} else if strategy.Type == "short" {
					canShort = true
				}
				break
			}
		}
	}
	return canLong, canShort
}

// 达到止盈或止损率之后的判定是否可以平仓
func (TradeLine TradeLineCustom) CanOrderComplete(symbol string, positionSide string) (complete bool) {
	var coin models.Symbols
	o := orm.NewOrm()
	o.QueryTable("symbols").Filter("Symbol", symbol).One(&coin)
	
	var strategyConfig technology.StrategyConfig
	err := json.Unmarshal([]byte(coin.Strategy), &strategyConfig)
	if err != nil {
		logs.Error("Error unmarshalling JSON Symbol: ", coin.Symbol)
		logs.Error("Error unmarshalling JSON:", err.Error())
		return false
	}
	findStrategy := false
	env := InitParseEnv(coin.Symbol, coin.Technology)
	for _, strategy := range strategyConfig {
		if strategy.Enable && (strategy.Type == "close_long" || strategy.Type == "close_short") {
			if strategy.Type == "close_long" && positionSide != "LONG" {
				// 平多仓的策略，当前仓位不是多仓，跳过
				continue
			}
			if strategy.Type == "close_short" && positionSide != "SHORT" {
				// 平空仓的策略，当前仓位不是空仓，跳过
				continue
			}
			
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
			findStrategy = true // 发现有正常能执行的平仓策略
			if output.(bool) {
				return true
			}
		}
	}
	if !findStrategy {
		// 没有定义平仓策略，使用简单的策略平仓
		return TradeLine.SimpleCloseStrategy(symbol, positionSide)
	}
	return false
}

func (TradeLine TradeLineCustom) SimpleCloseStrategy(symbol string, positionSide string) (stop bool) {
	lines, err := binance.GetKlineData(symbol, "5m", 2)
	if err != nil {
		return true
	}
	close0, _ := strconv.ParseFloat(lines[0].Close, 64)
	close1, _ := strconv.ParseFloat(lines[1].Close, 64)
	if positionSide == "LONG" {
		return close0 < close1 // 价格在下跌中
	} else if positionSide == "SHORT" {
		return close0 > close1 // 价格在上涨中
	} else {
		return false
	}
}

// 达到止盈或止损止前的判定是否可以平仓
func (TradeLine TradeLineCustom) AutoStopOrder(position *futures.PositionRisk, nowProfit float64) (stop bool) {
	return false
}

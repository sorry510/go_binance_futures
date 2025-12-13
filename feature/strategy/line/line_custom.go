package line

import (
	"encoding/json"
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/feature/strategy"
	"go_binance_futures/models"
	"go_binance_futures/technology"
	"go_binance_futures/types"
	"math"
	"strconv"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
	"github.com/expr-lang/expr"
)

type TradeLineCustom struct {
}

// 交易逻辑: 自定义逻辑
func (TradeLine TradeLineCustom) GetCanLongOrShort(openParams strategy.OpenParams) (openResult strategy.OpenResult) {
	coin := openParams.Symbols
	openResult.CanLong = false
	openResult.CanShort = false
	
	var strategyConfig technology.StrategyConfig
	err := json.Unmarshal([]byte(coin.Strategy), &strategyConfig)
	if err != nil {
		logs.Error("Error unmarshalling JSON Symbol: ", coin.Symbol)
		logs.Error("Error unmarshalling JSON:", err.Error())
		return openResult
	}
	env := InitParseEnv(coin.Symbol, coin.Technology)
	positions, _ := TradeLine.getTransformPositions()
	env["Positions"] = positions
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
			if result, ok := output.(bool); ok && result {
				if strategy.Type == "long" {
					openResult.CanLong = true
				} else if strategy.Type == "short" {
					openResult.CanShort = true
				}
				break
			}
		}
	}
	return openResult
}

// 达到止盈或止损率之后的判定是否可以平仓
func (TradeLine TradeLineCustom) CanOrderComplete(closeParams strategy.CloseParams) (closeResult strategy.CloseResult) {
	coin := closeParams.Symbols // 交易对
	position := closeParams.Position // 当前仓位
	closeResult.Complete = false
	
	var strategyConfig technology.StrategyConfig
	err := json.Unmarshal([]byte(coin.Strategy), &strategyConfig)
	if err != nil {
		logs.Error("Error unmarshalling JSON Symbol: ", coin.Symbol)
		logs.Error("Error unmarshalling JSON:", err.Error())
		return closeResult
	}
	findStrategy := false
	env := InitParseEnv(coin.Symbol, coin.Technology)
	env["ROI"] = closeParams.NowProfit // 当前收益率
	env["Position"] = types.FuturesPositionCode{
		Symbol: coin.Symbol,
		Side: position.Side,
		Amount: func() float64 {
			amount, err := strconv.ParseFloat(position.Amount, 64)
			if err != nil {
				logs.Error("Error parsing Amount to float64:", err.Error())
				return 0
			}
			return amount
		}(),
		Leverage: position.Leverage,
		EntryPrice: func() float64 {
			entryPrice, err := strconv.ParseFloat(position.EntryPrice, 64)
			if err != nil {
				logs.Error("Error parsing EntryPrice to float64:", err.Error())
				return 0
			}
			return entryPrice
		}(),
		MarkPrice: func() float64 {
			markPrice, err := strconv.ParseFloat(position.MarkPrice, 64)
			if err != nil {
				logs.Error("Error parsing MarkPrice to float64:", err.Error())
				return 0
			}
			return markPrice
		}(),
		UnrealizedProfit: func() float64 {
			unrealizedProfit, err := strconv.ParseFloat(position.UnrealizedProfit, 64)
			if err != nil {
				logs.Error("Error parsing UnrealizedProfit to float64:", err.Error())
				return 0
			}
			return unrealizedProfit
		}(),
		Mock: false,
		CreateTime: position.CreateTime,
		SourceType: position.SourceType,
	} // 当前仓位信息
	for _, strategy := range strategyConfig {
		if strategy.Enable && (strategy.Type == "close_long" || strategy.Type == "close_short") {
			if strategy.Type == "close_long" && position.Side != "LONG" {
				// 平多仓的策略，当前仓位不是多仓，跳过
				continue
			}
			if strategy.Type == "close_short" && position.Side != "SHORT" {
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
			if result, ok := output.(bool); ok && result {
				closeResult.Complete = true
			}
		}
	}
	if !findStrategy {
		// 没有定义平仓策略，使用简单的策略平仓
		return TradeLine.simpleCloseStrategy(closeParams)
	}
	return closeResult
}

// 达到止盈或止损止前的判定是否可以平仓
func (TradeLine TradeLineCustom) AutoStopOrder(closeParams strategy.CloseParams) (closeResult strategy.CloseResult) {
	closeResult.Complete = false
	return closeResult
}

func (TradeLine TradeLineCustom) simpleCloseStrategy(closeParams strategy.CloseParams) (closeResult strategy.CloseResult) {
	coin := closeParams.Symbols // 交易对
	position := closeParams.Position // 当前仓位
	closeResult.Complete = false
	
	if closeParams.NowProfit < 3 || closeParams.NowProfit > -3 {
		// 收益率小于3%或者大于-3%, 不平仓
		closeResult.Complete = false
		return closeResult
	}
	
	lines, err := binance.GetKlineData(coin.Symbol, "5m", 2)
	if err != nil {
		logs.Error("Error GetKlineData Symbol in line_custom: ", coin.Symbol)
		logs.Error("Error GetKlineData Symbol in line_custom:", err.Error())
		closeResult.Complete = true
		return closeResult
	}
	close0, _ := strconv.ParseFloat(lines[0].Close, 64)
	close1, _ := strconv.ParseFloat(lines[1].Close, 64)
	if position.Side == "LONG" {
		closeResult.Complete = close0 < close1 // 价格在下跌中
	} else if position.Side == "SHORT" {
		closeResult.Complete = close0 > close1 // 价格在上涨中
	} else {
		// BOTH 不处理双向持仓类型
		closeResult.Complete = true
	}
	return closeResult
}

func (TradeLine TradeLineCustom) getTransformPositions() (usePositions []types.FuturesPosition, err error) {
	var positions []models.FuturesPosition
	o := orm.NewOrm()
	sql := "SELECT f.id, f.symbol, f.side, f.amount, f.leverage, f.margin_type, f.isolated_wallet, f.entry_price, s.close as mark_price FROM `futures_positions` f LEFT JOIN symbols s ON f.symbol = s.symbol where 1 = 1"
	sql += ` and f.amount <> '0'`
	_, err = o.Raw(sql).QueryRows(&positions)
	if err != nil {
		logs.Error("GetLocalPosition err in StartTrade:", err.Error())
		return usePositions, err
	}
	for _, position := range positions {
		positionAmt, _ := strconv.ParseFloat(position.Amount, 64)
		positionAmtFloatAbs := math.Abs(positionAmt) // 空单为负数,纠正为绝对值
		if positionAmtFloatAbs < 0.0000001 {
			continue
		}
		enterPrice_float64, _ := strconv.ParseFloat(position.EntryPrice, 64)
		markPrice_float64, _ := strconv.ParseFloat(position.MarkPrice, 64)
		unRealizedProfit := (markPrice_float64 - enterPrice_float64) * positionAmt // 未实现盈亏
		position.UnrealizedProfit = strconv.FormatFloat(unRealizedProfit, 'f', -1, 64)
		
		usePositions = append(usePositions, types.FuturesPosition{
			Symbol: position.Symbol,
			Side: position.Side,
			Amount: position.Amount,
			MarginType: position.MarginType,
			Leverage: position.Leverage,
			IsolatedWallet: position.IsolatedWallet,
			EntryPrice: position.EntryPrice,
			MarkPrice: position.MarkPrice,
			UnrealizedProfit: position.UnrealizedProfit,
			SourceType: "local",
			CreateTime: position.CreateTime,
		})
	}
	return usePositions, err
}
package controllers

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"go_binance_futures/feature/strategy/line"
	"go_binance_futures/models"
	"go_binance_futures/technology"
	"go_binance_futures/types"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
	"github.com/expr-lang/expr"
)

type TestStrategyResultController struct {
	web.Controller
}

type TestStrategyResultsTableList struct {
	models.TestStrategyResults
	NowPrice      string `orm:"column(now_price)" json:"now_price"`
	ProfitPercent string `json:"profit_percent"`
}

func calculateTestStrategyResultMetrics(result *TestStrategyResultsTableList) {
	entryPrice, errEntry := strconv.ParseFloat(result.Price, 64)
	positionAmt, errAmount := strconv.ParseFloat(result.PositionAmt, 64)
	closePrice, _ := strconv.ParseFloat(result.ClosePrice, 64)
	if errEntry != nil || errAmount != nil {
		result.CloseProfit = "0.000"
		result.ProfitPercent = "0.000"
		return
	}

	effectivePrice := closePrice
	if effectivePrice <= 0 {
		currentPrice, errCurrent := strconv.ParseFloat(result.NowPrice, 64)
		if errCurrent != nil {
			result.CloseProfit = "0.000"
			result.ProfitPercent = "0.000"
			return
		}
		effectivePrice = currentPrice
	}
	if effectivePrice <= 0 || positionAmt == 0 {
		result.CloseProfit = "0.000"
		result.ProfitPercent = "0.000"
		return
	}

	profit := (effectivePrice - entryPrice) * positionAmt
	profitPercent := profit / (math.Abs(positionAmt) * effectivePrice) * float64(result.Leverage) * 100
	result.CloseProfit = strconv.FormatFloat(profit, 'f', 3, 64)
	result.ProfitPercent = strconv.FormatFloat(profitPercent, 'f', 3, 64)
}

type testStrategyResultSearchParams struct {
	Symbol       string
	PositionSide string
	StartTime    string
	EndTime      string
	Type         string
}

func (ctrl *TestStrategyResultController) getSearchParams() testStrategyResultSearchParams {
	return testStrategyResultSearchParams{
		Symbol:       strings.TrimSpace(ctrl.GetString("symbol")),
		PositionSide: strings.ToUpper(strings.TrimSpace(ctrl.GetString("position_side"))),
		StartTime:    strings.TrimSpace(ctrl.GetString("start_time")),
		EndTime:      strings.TrimSpace(ctrl.GetString("end_time")),
		Type:         strings.ToLower(strings.TrimSpace(ctrl.GetString("type"))),
	}
}

func (params testStrategyResultSearchParams) whereClause(tableAlias string) (string, []interface{}, bool, error) {
	prefix := ""
	if tableAlias != "" {
		prefix = tableAlias + "."
	}

	conditions := make([]string, 0, 5)
	args := make([]interface{}, 0, 4)
	if params.Symbol != "" {
		if strings.ContainsAny(params.Symbol, "%_") {
			return "", nil, false, fmt.Errorf("invalid symbol")
		}
		conditions = append(conditions, prefix+"symbol LIKE ?")
		args = append(args, "%"+params.Symbol+"%")
	}
	if params.PositionSide != "" && params.PositionSide != "ALL" {
		conditions = append(conditions, prefix+"position_side = ?")
		args = append(args, params.PositionSide)
	}

	var startTime int64
	var endTime int64
	if params.StartTime != "" {
		value, err := strconv.ParseInt(params.StartTime, 10, 64)
		if err != nil || value <= 0 {
			return "", nil, false, fmt.Errorf("invalid start_time")
		}
		startTime = value
		conditions = append(conditions, prefix+"createTime >= ?")
		args = append(args, startTime)
	}
	if params.EndTime != "" {
		value, err := strconv.ParseInt(params.EndTime, 10, 64)
		if err != nil || value <= 0 {
			return "", nil, false, fmt.Errorf("invalid end_time")
		}
		endTime = value
		conditions = append(conditions, prefix+"createTime <= ?")
		args = append(args, endTime)
	}
	if startTime > 0 && endTime > 0 && startTime > endTime {
		return "", nil, false, fmt.Errorf("start_time must not exceed end_time")
	}

	if params.Type == "open" {
		conditions = append(conditions, prefix+"close_price = '0'")
	} else if params.Type == "close" {
		conditions = append(conditions, prefix+"close_price != '0'")
	}

	if len(conditions) == 0 {
		return "", nil, false, nil
	}
	return " AND " + strings.Join(conditions, " AND "), args, true, nil
}

func (ctrl *TestStrategyResultController) Get() {
	paramsPage := ctrl.GetString("page", "1")
	paramsLimit := ctrl.GetString("limit", "20")
	searchParams := ctrl.getSearchParams()
	
	page, _ := strconv.Atoi(paramsPage)
	limit, _ := strconv.Atoi(paramsLimit)
	offset := (page - 1) * limit
	
	o := orm.NewOrm()
	
	var results []TestStrategyResultsTableList
	var total int64
	sql := `SELECT t.id, t.symbol, t.price, t.leverage, t.usdt, t.profit, t.loss, t.position_amt, t.position_side, t.close_price, t.close_profit, t.createTime, t.updateTime, s.close as now_price FROM test_strategy_results t LEFT JOIN symbols s ON t.symbol = s.symbol where 1 = 1`
	countSql := `SELECT COUNT(*) FROM test_strategy_results t LEFT JOIN symbols s ON t.symbol = s.symbol where 1 = 1`
	whereClause, args, _, err := searchParams.whereClause("t")
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	sql += whereClause
	countSql += whereClause
	
	sql = sql + " ORDER BY t.createTime DESC LIMIT " + strconv.Itoa(limit) + " OFFSET " + strconv.Itoa(offset)
	_, err = o.Raw(sql, args...).QueryRows(&results)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	for index := range results {
		calculateTestStrategyResultMetrics(&results[index])
	}
	err = o.Raw(countSql, args...).QueryRow(&total)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	
	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"data": map[string]interface{} {
			"total": total,
			"list": results,
		},
		"msg": "success",
	})
}

func (ctrl *TestStrategyResultController) Show() {
	id := ctrl.Ctx.Input.Param(":id")
	o := orm.NewOrm()
	var result TestStrategyResultsTableList
	err := o.Raw("SELECT * from test_strategy_results WHERE id = ?", id).QueryRow(&result)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{
		"code": 200,
		"data": result,
		"msg":  "success",
	})
}

func (ctrl *TestStrategyResultController) Delete() {
	id := ctrl.Ctx.Input.Param(":id")
	o := orm.NewOrm()
	deleteSQL := "DELETE FROM test_strategy_results WHERE 1 = 1"
	args := make([]interface{}, 0, 4)

	if id != "" {
		deleteSQL += " AND id = ?"
		args = append(args, id)
	} else {
		whereClause, searchArgs, hasSearchCondition, err := ctrl.getSearchParams().whereClause("")
		if err != nil {
			ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
			return
		}
		if !hasSearchCondition {
			ctrl.Ctx.Resp(utils.ResJson(400, nil, "at least one search condition is required"))
			return
		}
		deleteSQL += whereClause
		args = append(args, searchArgs...)
	}

	result, err := o.Raw(deleteSQL, args...).Exec()
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	deleted, _ := result.RowsAffected()
	ctrl.Ctx.Resp(utils.ResJson(200, map[string]interface{}{
		"deleted": deleted,
	}))
}

// 如果测试策略开启，合约列表页面的测试策略，使用的是这个接口
func (ctrl *TestStrategyResultController) TestStrategyRule() {
	symbol := ctrl.Ctx.Input.Param(":symbol")
	var result models.TestStrategyResults
	o := orm.NewOrm()
	o.QueryTable("test_strategy_results").
		Filter("Symbol", symbol).
		Filter("ClosePrice", "0").
		One(&result)
	
	ctrl.BindJSON(&result) // 装载前端传入的最新的策略规则
	
	var strategyConfig technology.StrategyConfig
	err := json.Unmarshal([]byte(result.Strategy), &strategyConfig)
	if err != nil {
		logs.Error("Error unmarshalling JSON:", err.Error())
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	env := line.InitParseEnv(result.Symbol, result.Technology)
	floatNowPrice, ok := env["NowPrice"].(float64)
	if !ok {
		logs.Error("Error NowPrice Symbol: ", result.Symbol)
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "Error NowPrice Symbol"))
		return
	}
	// 随意指定一个模拟信息
	env["ROI"] = 10.22
	env["Position"] = types.FuturesPositionCode{
		Symbol: result.Symbol,
		EntryPrice: 68000.0,
		MarkPrice: 72000.0,
		Amount: -0.02,
		UnrealizedProfit: 100.2,
		Leverage: 3,
		Side: "SHORT",
		Mock: true,
		CreateTime: 1234567890000,
		SourceType: "local",
	}
	if result.ID != 0 {
		// 如果查到了为平仓的测试数据，就加载仓位信息
		positionAmtFloat, _ := strconv.ParseFloat(result.PositionAmt, 64)
		positionAmtFloatAbs := math.Abs(positionAmtFloat) // 空单为负数,纠正为绝对值
		enterPrice_float64, _ := strconv.ParseFloat(result.Price, 64)
		unRealizedProfit := (floatNowPrice - enterPrice_float64) * positionAmtFloat // 未实现盈亏
		nowProfit := (unRealizedProfit / (positionAmtFloatAbs * floatNowPrice)) * float64(result.Leverage) * 100
		
		env["ROI"] = nowProfit // 当前收益率
		// 模拟仓位信息
		env["Position"] = types.FuturesPositionCode{
			Symbol: result.Symbol,
			EntryPrice: enterPrice_float64,
			MarkPrice: floatNowPrice,
			Amount: positionAmtFloat,
			UnrealizedProfit: unRealizedProfit,
			Leverage: result.Leverage,
			Side: result.PositionSide,
			Mock: false,
			CreateTime: result.CreateTime,
			SourceType: "local",
		}
	}
	
	for _, strategy := range strategyConfig {
		if strategy.Enable {
			program, err := expr.Compile(strategy.Code, expr.Env(env))
			if err != nil {
				logs.Error("Error Strategy Compile:", err.Error())
				ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
				return
			}
			output, err := expr.Run(program, env)
			if err != nil {
				logs.Error("Error Strategy Run:", err.Error())
				ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
				return
			}
			ctrl.Ctx.Resp(map[string]interface{} {
				"code": 200,
				"data": map[string]interface{} {
					"pass": output,
					"type": strategy.Type,
				},
				"msg": "success",
			})
		}
	}
}

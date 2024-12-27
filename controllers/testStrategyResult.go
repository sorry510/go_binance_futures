package controllers

import (
	"encoding/json"
	"math"
	"strconv"

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
	NowPrice string `orm:"column(now_price)" json:"now_price"`
}

func (ctrl *TestStrategyResultController) Get() {
	paramsSymbol := ctrl.GetString("symbol")
	paramPositionSide := ctrl.GetString("position_side") // LONG, SHORT, all
	paramsPage := ctrl.GetString("page", "1")
	paramsLimit := ctrl.GetString("limit", "20")
	paramsStartTime := ctrl.GetString("start_time")
	paramsEndTime := ctrl.GetString("end_time")
	paramsType := ctrl.GetString("type") // open, close, all
	
	page, _ := strconv.Atoi(paramsPage)
	limit, _ := strconv.Atoi(paramsLimit)
	offset := (page - 1) * limit
	
	o := orm.NewOrm()
	
	var results []TestStrategyResultsTableList
	var total int64
	sql := `SELECT t.*, s.close as now_price FROM test_strategy_results t LEFT JOIN symbols s ON t.symbol = s.symbol where 1 = 1`
	countSql := `SELECT COUNT(*) FROM test_strategy_results t LEFT JOIN symbols s ON t.symbol = s.symbol where 1 = 1`
	
	if (paramsSymbol != "") {
		sql += ` and t.symbol like '%` + paramsSymbol + `%'`
		countSql += ` and t.symbol like '%` + paramsSymbol + `%'`
	}
	if (paramPositionSide != "all" && paramPositionSide != "") {
		sql += ` and t.position_side = '` + paramPositionSide + `'`
		countSql += ` and t.position_side = '` + paramPositionSide + `'`
	}
	if (paramsStartTime != "") {
		sql += ` and t.createTime >= '` + paramsStartTime + `'`
		countSql += ` and t.createTime >= '` + paramsStartTime + `'`
	}
	if (paramsEndTime != "") {
		sql += ` and t.createTime <= '` + paramsEndTime + `'`
		countSql += ` and t.createTime <= '` + paramsEndTime + `'`
	}
	if (paramsType == "open") {
		sql += ` and t.close_price = '0'`
		countSql += ` and t.close_price = '0'`
	} else if (paramsType == "close") {
		sql += ` and t.close_price != '0'`
		countSql += ` and t.close_price != '0'`
	}
	
	sql = sql + " ORDER BY t.createTime DESC LIMIT " + strconv.Itoa(limit) + " OFFSET " + strconv.Itoa(offset)
    _, err := o.Raw(sql).QueryRows(&results)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
	}
	err = o.Raw(countSql).QueryRow(&total)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
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

func (ctrl *TestStrategyResultController) Delete() {
	id := ctrl.Ctx.Input.Param(":id")
	o := orm.NewOrm()
	_, err := o.Raw("DELETE FROM \"test_strategy_results\" where id = ?", id).Exec()
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(utils.ResJson(200, nil))
}

func (ctrl *TestStrategyResultController) DeleteAll() {
	o := orm.NewOrm()
	_, err := o.Raw("DELETE FROM \"test_strategy_results\" where 1=1").Exec()
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(utils.ResJson(200, nil))
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

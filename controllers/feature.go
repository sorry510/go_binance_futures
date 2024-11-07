package controllers

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"go_binance_futures/feature"
	"go_binance_futures/feature/strategy/line"
	"go_binance_futures/models"
	"go_binance_futures/technology"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
	"github.com/expr-lang/expr"
)


type BatchEditParams struct {
	Usdt string `json:"usdt"`
	Profit string `json:"profit"`
	Loss string `json:"loss"`
	Leverage string `json:"leverage"`
	MarginType string `json:"marginType"`
	StrategyType string `json:"strategyType"`
	StrategyTemplateId int64 `json:"strategyTemplateId"`
}
type FeatureController struct {
	web.Controller
}

func (ctrl *FeatureController) Get() {
	paramsSort := ctrl.GetString("sort")
	symbol := ctrl.GetString("symbol")
	enable := ctrl.GetString("enable")
	margin_type := ctrl.GetString("margin_type")
	
	o := orm.NewOrm()
	var symbols []models.Symbols
	query := o.QueryTable("symbols")
	if symbol != "" {
		query = query.Filter("symbol__contains", symbol)
	}
	if enable != "" {
		query = query.Filter("enable", enable)
	}
	if margin_type != "" {
		query = query.Filter("marginType", margin_type)
	}
	_, err := query.OrderBy("ID").All(&symbols)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "error"))
	}
	
	sort.SliceStable(symbols, func(i, j int) bool {
		if paramsSort == "+" {
			return symbols[i].PercentChange >= symbols[j].PercentChange // 涨幅从大到小排序
		} else if paramsSort == "-" {
			return symbols[i].PercentChange < symbols[j].PercentChange // 涨幅从小到大排序
		} else {
			return true
		}
	})

	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"data": symbols,
		"msg": "success",
	})
}

func (ctrl *FeatureController) Edit() {
	id := ctrl.Ctx.Input.Param(":id")
	var symbols models.Symbols
	o := orm.NewOrm()
	o.QueryTable("symbols").Filter("Id", id).One(&symbols)
	
	go func() {
		// feature.UpdateSymbolTradeInfo(symbols) // 更新合约倍率和仓位模式
		feature.UpdateSymbolsTradePrecision() // 更新合约交易精度
	}()
	
	ctrl.BindJSON(&symbols)
	
	_, err := o.Update(&symbols) // _ 是受影响的条数
    if err != nil {
        // 处理错误
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "edit failed"))
		return
    }
	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"data": symbols,
		"msg": "success",
	})
}

func (ctrl *FeatureController) Delete() {
	id := ctrl.Ctx.Input.Param(":id")
	symbols := new(models.Symbols)
	intId, _ := strconv.ParseInt(id, 10, 64)
	symbols.ID = intId
	o := orm.NewOrm()
	
	_, err := o.Delete(symbols)
    if err != nil {
        // 处理错误
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "delete failed"))
		return
    }
	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"msg": "success",
	})
}

func (ctrl *FeatureController) Post() {
	symbols := new(models.Symbols)
	ctrl.BindJSON(&symbols)
	symbols.PercentChange = 0
	symbols.Close = "0"
	symbols.Open = "0"
	symbols.Low = "0"
	
	symbols.Leverage = 3
	symbols.MarginType = "CROSSED" // 杠杆类型 ISOLATED(逐仓), CROSSED(全仓)
	symbols.StepSize = "0.1"
	symbols.TickSize = "0.1"
	symbols.Usdt = "10"
	symbols.Profit = "100"
	symbols.Loss = "100"
	symbols.KlineInterval = "1d"
	
	o := orm.NewOrm()
	id, err := o.Insert(symbols)
	
	go func() {
		// feature.UpdateSymbolTradeInfo(symbols) // 更新合约倍率和仓位模式
		feature.UpdateSymbolsTradePrecision() // 更新合约交易精度
	}()
	
    if err != nil {
        // 处理错误
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "create failed"))
		return
    }
	symbols.ID = id
	
	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"data": symbols,
		"msg": "success",
	})
}

func (ctrl *FeatureController) UpdateEnable() {
	flag := ctrl.Ctx.Input.Param(":flag")
	
	o := orm.NewOrm()
	_, err := o.Raw("UPDATE symbols SET enable = ?", flag).Exec()
	if err != nil {
		// 处理错误
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "edit failed"))
		return
	}
	ctrl.Ctx.Resp(utils.ResJson(200, nil))
}

func (ctrl *FeatureController) BatchEdit() {
	params := new(BatchEditParams)
	ctrl.BindJSON(&params)
	query := "UPDATE symbols SET"
	
	if params.Usdt != "" {
		query += " usdt = " + params.Usdt + ","
	}
	if (params.Profit != "") {
		query += " profit = " + params.Profit + ","	
	}
	if (params.Loss != "") {
		query += " loss = " + params.Loss + ","
	}
	if (params.Leverage != "") {
		query += " leverage = " + params.Leverage + ","
	}
	if (params.MarginType != "") {
		query += " marginType = '" + params.MarginType + "',"
	}
	if (params.MarginType != "") {
		query += " marginType = '" + params.MarginType + "',"
	}
	if (params.StrategyType != "") {
		query += " strategy_type = '" + params.StrategyType + "',"
	}
	if (params.StrategyTemplateId != 0) {
		var template models.StrategyTemplates
		orm.NewOrm().QueryTable("strategy_templates").Filter("Id", params.StrategyTemplateId).One(&template)
		query += " technology = '" + template.Technology + "',"
		query += " strategy = '" + template.Strategy + "',"
	}
	
	if strings.HasSuffix(query, ",") {
		query = strings.TrimSuffix(query, ",")
		query += " WHERE enable = 1" // 只更新开启的交易对
		_, err := orm.NewOrm().Raw(query).Exec()
		if err != nil {
			// 处理错误
			ctrl.Ctx.Resp(utils.ResJson(400, nil, "更新错误"))
			return
		}
	}
	
	go func() {
		feature.UpdateSymbolsTradePrecision() // 更新合约交易精度
	}()
	
	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"msg": "success",
	})
}


func (ctrl *FeatureController) TestStrategyRule() {
	id := ctrl.Ctx.Input.Param(":id")
	var symbols models.Symbols
	o := orm.NewOrm()
	o.QueryTable("symbols").Filter("Id", id).One(&symbols)
	
	ctrl.BindJSON(&symbols)
	
	var strategyConfig technology.StrategyConfig
	err := json.Unmarshal([]byte(symbols.Strategy), &strategyConfig)
	if err != nil {
		logs.Error("Error unmarshalling JSON:", err.Error())
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	env := line.InitParseEnv(symbols.Symbol, symbols.Technology)
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
package controllers

import (
	"encoding/json"
	"go_binance_futures/feature"
	"go_binance_futures/feature/strategy/line"
	"go_binance_futures/models"
	strategyservice "go_binance_futures/service/strategy"
	"go_binance_futures/technology"
	"go_binance_futures/types"
	"go_binance_futures/utils"
	"strconv"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
	"github.com/expr-lang/expr"
)

type StrategyTemplateController struct {
	web.Controller
}

func (ctrl *StrategyTemplateController) Post() {
	templates := new(models.StrategyTemplates)
	ctrl.BindJSON(&templates)

	o := orm.NewOrm()
	id, err := o.Insert(templates)

	if err != nil {
		// 处理错误
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	templates.ID = id

	ctrl.Ctx.Resp(map[string]interface{}{
		"code": 200,
		"data": templates,
		"msg":  "success",
	})
}

func (ctrl *StrategyTemplateController) Get() {
	page, _ := strconv.Atoi(ctrl.GetString("page", "1"))
	limit, _ := strconv.Atoi(ctrl.GetString("limit", "20"))
	result, err := (strategyservice.Service{}).ListTemplates(ctrl.Ctx.Request.Context(), strategyservice.TemplateListOptions{
		Name: ctrl.GetString("name", ""), Page: page, Limit: limit, DefaultLimit: 20, MaxLimit: 100,
	})
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": map[string]interface{}{"total": result.Total, "list": result.List}, "msg": "success"})
}

func queryStrategyTemplatePage(o orm.Ormer, name string, limit, offset int) ([]models.StrategyTemplates, int64, error) {
	return strategyservice.QueryTemplatePage(o, name, limit, offset)
}

func (ctrl *StrategyTemplateController) Edit() {
	id := ctrl.Ctx.Input.Param(":id")
	var templates models.StrategyTemplates
	o := orm.NewOrm()
	o.QueryTable("strategy_templates").Filter("Id", id).One(&templates)

	ctrl.BindJSON(&templates)

	_, err := o.Update(&templates) // _ 是受影响的条数
	if err != nil {
		// 处理错误
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{
		"code": 200,
		"data": templates,
		"msg":  "success",
	})
}

func (ctrl *StrategyTemplateController) Delete() {
	id := ctrl.Ctx.Input.Param(":id")
	templates := new(models.StrategyTemplates)
	intId, _ := strconv.ParseInt(id, 10, 64)
	templates.ID = intId
	o := orm.NewOrm()

	_, err := o.Delete(templates)
	if err != nil {
		// 处理错误
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{
		"code": 200,
		"msg":  "success",
	})
}

func (ctrl *StrategyTemplateController) TestStrategyRule() {
	symbol := ctrl.Ctx.Input.Param(":symbol")
	var symbols models.Symbols
	o := orm.NewOrm()
	o.QueryTable("symbols").Filter("Symbol", symbol).One(&symbols)

	ctrl.BindJSON(&symbols)

	var strategyConfig technology.StrategyConfig
	err := json.Unmarshal([]byte(symbols.Strategy), &strategyConfig)
	if err != nil {
		logs.Error("Error unmarshalling JSON:", err.Error())
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	env := line.InitParseEnv(symbols.Symbol, symbols.Technology)
	env["ROI"] = 10.2
	env["Position"] = types.FuturesPositionCode{
		Symbol:           symbols.Symbol,
		EntryPrice:       68000.0,
		MarkPrice:        72000.0,
		Amount:           -0.02,
		UnrealizedProfit: 100.2,
		Leverage:         3,
		Side:             "SHORT",
		Mock:             true,
		CreateTime:       1234567890000,
		SourceType:       "api",
	}
	positions, _ := feature.GetTransformPositions()
	env["Positions"] = positions
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
			ctrl.Ctx.Resp(map[string]interface{}{
				"code": 200,
				"data": map[string]interface{}{
					"pass": output,
					"type": strategy.Type,
				},
				"msg": "success",
			})
		}
	}
}

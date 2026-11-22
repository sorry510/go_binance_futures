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
	"strings"

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
	if err := ctrl.BindJSON(templates); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "策略模板 JSON 无效: "+err.Error()))
		return
	}
	if err := validateEditableStrategyTemplate(*templates); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}

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
	var patch struct {
		Name       *string `json:"name"`
		Technology *string `json:"technology"`
		Strategy   *string `json:"strategy"`
	}
	if err := json.Unmarshal(ctrl.Ctx.Input.RequestBody, &patch); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "策略模板 JSON 无效: "+err.Error()))
		return
	}
	if patch.Name == nil && patch.Technology == nil && patch.Strategy == nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "未提供要修改的策略字段"))
		return
	}

	var templates models.StrategyTemplates
	o := orm.NewOrm()
	if err := o.QueryTable("strategy_templates").Filter("Id", id).One(&templates); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "策略模板不存在: "+err.Error()))
		return
	}
	changedRules := patch.Technology != nil || patch.Strategy != nil
	if patch.Name != nil {
		templates.Name = strings.TrimSpace(*patch.Name)
	}
	if patch.Technology != nil {
		templates.Technology = *patch.Technology
	}
	if patch.Strategy != nil {
		templates.Strategy = *patch.Strategy
	}
	if strings.TrimSpace(templates.Name) == "" {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "模板名称不能为空"))
		return
	}
	// Name-only edits must remain possible on historical templates with
	// invalid strategy text. New/changed indicator and rule content is checked
	// together before either field is persisted.
	if changedRules {
		if err := validateEditableStrategyTemplate(templates); err != nil {
			ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
			return
		}
	}

	_, err := o.Update(&templates, "Name", "Technology", "Strategy")
	if err != nil {
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
			program, err := strategyservice.CompileRule(strategy.Code, env, false, false)
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

package controllers

import (
	"strconv"
	"strings"

	outcomereview "go_binance_futures/service/outcomereview"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/server/web"
)

type AgentOutcomeController struct{ web.Controller }

func (ctrl *AgentOutcomeController) filter() outcomereview.Filter {
	start, _ := strconv.ParseInt(strings.TrimSpace(ctrl.GetString("start_time")), 10, 64)
	end, _ := strconv.ParseInt(strings.TrimSpace(ctrl.GetString("end_time")), 10, 64)
	templateID, _ := strconv.ParseInt(strings.TrimSpace(ctrl.GetString("strategy_template_id")), 10, 64)
	marketCondition, _ := strconv.Atoi(strings.TrimSpace(ctrl.GetString("market_condition")))
	return outcomereview.Filter{
		StartTime: start, EndTime: end, StrategyTemplateID: templateID,
		Symbol: ctrl.GetString("symbol"), Side: ctrl.GetString("side"), MarketCondition: marketCondition,
	}
}

func (ctrl *AgentOutcomeController) Backtest() {
	result, err := (outcomereview.Service{}).Backtest(ctrl.Ctx.Request.Context(), ctrl.filter())
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": result, "msg": "success"})
}

func (ctrl *AgentOutcomeController) Paper() {
	result, err := (outcomereview.Service{}).Paper(ctrl.Ctx.Request.Context(), ctrl.filter())
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": result, "msg": "success"})
}

func (ctrl *AgentOutcomeController) Live() {
	result, err := (outcomereview.Service{}).Live(ctrl.Ctx.Request.Context(), ctrl.filter())
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": result, "msg": "success"})
}

package controllers

import (
	"encoding/json"
	"strconv"
	"strings"

	backtestservice "go_binance_futures/service/backtest"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/server/web"
)

type AgentBacktestController struct{ web.Controller }

func (ctrl *AgentBacktestController) List() {
	page, _ := strconv.Atoi(ctrl.GetString("page", "1"))
	limit, _ := strconv.Atoi(ctrl.GetString("limit", "20"))
	items, total, err := backtestservice.DefaultManager().List(page, limit)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(500, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": map[string]interface{}{"list": items, "total": total, "page": page, "limit": limit}, "msg": "success"})
}
func (ctrl *AgentBacktestController) Prefetch() {
	var request backtestservice.PrefetchRequest
	if err := json.Unmarshal(ctrl.Ctx.Input.RequestBody, &request); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "invalid request: "+err.Error()))
		return
	}
	item, err := backtestservice.DefaultManager().StartPrefetch(request)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": item, "msg": "success"})
}
func (ctrl *AgentBacktestController) PrefetchStatus() {
	item, err := backtestservice.DefaultManager().GetPrefetch(strings.TrimSpace(ctrl.Ctx.Input.Param(":jobId")))
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(404, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": item, "msg": "success"})
}

func (ctrl *AgentBacktestController) MarketConditionBackfill() {
	item, err := backtestservice.DefaultManager().StartMarketConditionBackfill()
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": item, "msg": "success"})
}

func (ctrl *AgentBacktestController) MarketConditionBackfillStatus() {
	item, err := backtestservice.DefaultManager().GetMarketConditionBackfill(strings.TrimSpace(ctrl.Ctx.Input.Param(":jobId")))
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(404, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": item, "msg": "success"})
}

func (ctrl *AgentBacktestController) Start() {
	var request backtestservice.StartRequest
	if err := json.Unmarshal(ctrl.Ctx.Input.RequestBody, &request); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "invalid request: "+err.Error()))
		return
	}
	item, err := backtestservice.DefaultManager().Start(request)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": item, "msg": "success"})
}
func (ctrl *AgentBacktestController) Get() {
	item, err := backtestservice.DefaultManager().Get(strings.TrimSpace(ctrl.Ctx.Input.Param(":id")))
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(404, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": item, "msg": "success"})
}
func (ctrl *AgentBacktestController) Delete() {
	if err := backtestservice.DefaultManager().Delete(strings.TrimSpace(ctrl.Ctx.Input.Param(":id"))); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": true, "msg": "success"})
}
func (ctrl *AgentBacktestController) Cancel() {
	if err := backtestservice.DefaultManager().Cancel(strings.TrimSpace(ctrl.Ctx.Input.Param(":id"))); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": true, "msg": "success"})
}
func (ctrl *AgentBacktestController) Trades() {
	limit, _ := strconv.Atoi(ctrl.GetString("limit", "5000"))
	items, err := backtestservice.DefaultManager().Trades(strings.TrimSpace(ctrl.Ctx.Input.Param(":id")), limit)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": items, "msg": "success"})
}
func (ctrl *AgentBacktestController) Events() {
	limit, _ := strconv.Atoi(ctrl.GetString("limit", "10000"))
	items, err := backtestservice.DefaultManager().Events(strings.TrimSpace(ctrl.Ctx.Input.Param(":id")), limit)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": items, "msg": "success"})
}
func (ctrl *AgentBacktestController) Equity() {
	limit, _ := strconv.Atoi(ctrl.GetString("limit", "20000"))
	items, err := backtestservice.DefaultManager().Equity(strings.TrimSpace(ctrl.Ctx.Input.Param(":id")), limit)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": items, "msg": "success"})
}

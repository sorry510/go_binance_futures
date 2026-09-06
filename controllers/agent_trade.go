package controllers

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"go_binance_futures/service/agenttrade"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/server/web"
)

type AgentTradeController struct{ web.Controller }

type agentTradeCreateRequest struct {
	TaskID string `json:"task_id"`
}

type agentTradeRejectRequest struct {
	Reason string `json:"reason"`
}

func tradeService() agenttrade.Service { return agenttrade.DefaultService() }

func (ctrl *AgentTradeController) proposalID() string {
	return strings.TrimSpace(ctrl.Ctx.Input.Param(":proposalId"))
}

func (ctrl *AgentTradeController) List() {
	page, _ := strconv.Atoi(ctrl.GetString("page", "1"))
	limit, _ := strconv.Atoi(ctrl.GetString("limit", "20"))
	store := agenttrade.Store{}
	_, _ = store.Expire(ctrl.Ctx.Request.Context(), time.Now().UTC().UnixMilli())
	result, err := store.List(ctrl.Ctx.Request.Context(), agenttrade.ListOptions{
		Status: ctrl.GetString("status"), Symbol: ctrl.GetString("symbol"), Page: page, Limit: limit,
	})
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(500, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": result, "msg": "success"})
}

func (ctrl *AgentTradeController) Create() {
	var request agentTradeCreateRequest
	if err := json.Unmarshal(ctrl.Ctx.Input.RequestBody, &request); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "请求格式错误: "+err.Error()))
		return
	}
	proposal, err := tradeService().CreateFromTask(ctrl.Ctx.Request.Context(), request.TaskID)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": proposal, "msg": "success"})
}

func (ctrl *AgentTradeController) Get() {
	id := ctrl.proposalID()
	store := agenttrade.Store{}
	proposal, err := store.GetProposal(ctrl.Ctx.Request.Context(), id)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(404, nil, "trade proposal not found"))
		return
	}
	execution, execErr := store.GetExecution(ctrl.Ctx.Request.Context(), id)
	audits, auditErr := store.Audits(ctrl.Ctx.Request.Context(), id)
	if auditErr != nil {
		ctrl.Ctx.Resp(utils.ResJson(500, nil, auditErr.Error()))
		return
	}
	data := map[string]any{"proposal": proposal, "audits": audits}
	if execErr == nil {
		data["execution"] = execution
	} else if execErr != orm.ErrNoRows {
		ctrl.Ctx.Resp(utils.ResJson(500, nil, execErr.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": data, "msg": "success"})
}

func (ctrl *AgentTradeController) Risk() {
	proposal, err := tradeService().EvaluateRisk(ctrl.Ctx.Request.Context(), ctrl.proposalID())
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": proposal, "msg": "success"})
}

func (ctrl *AgentTradeController) Approve() {
	proposal, err := tradeService().Approve(ctrl.Ctx.Request.Context(), ctrl.proposalID(), "web_admin")
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": proposal, "msg": "success"})
}

func (ctrl *AgentTradeController) Reject() {
	var request agentTradeRejectRequest
	_ = json.Unmarshal(ctrl.Ctx.Input.RequestBody, &request)
	proposal, err := tradeService().Reject(ctrl.Ctx.Request.Context(), ctrl.proposalID(), "web_admin", request.Reason)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": proposal, "msg": "success"})
}

func (ctrl *AgentTradeController) Execute() {
	proposal, execution, err := tradeService().Execute(ctrl.Ctx.Request.Context(), ctrl.proposalID(), "web_admin")
	if err != nil {
		ctrl.Ctx.Resp(map[string]any{"code": 400, "data": map[string]any{"proposal": proposal, "execution": execution}, "msg": err.Error()})
		return
	}
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": map[string]any{"proposal": proposal, "execution": execution}, "msg": "success"})
}

func (ctrl *AgentTradeController) Reconcile() {
	proposal, execution, err := tradeService().Reconcile(ctrl.Ctx.Request.Context(), ctrl.proposalID(), "web_admin")
	if err != nil {
		ctrl.Ctx.Resp(map[string]any{"code": 400, "data": map[string]any{"proposal": proposal, "execution": execution}, "msg": err.Error()})
		return
	}
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": map[string]any{"proposal": proposal, "execution": execution}, "msg": "success"})
}

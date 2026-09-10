package controllers

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"go_binance_futures/models"
	"go_binance_futures/service/agenttrade"
	futuresownership "go_binance_futures/service/futuresownership"
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
	ownership := futuresownership.DefaultService()
	if positions, positionErr := ownership.ListPositions(ctrl.Ctx.Request.Context(), futuresownership.OwnerAgentTrade, false); positionErr == nil {
		for _, position := range positions {
			if position.SourceRef == proposal.ProposalID {
				data["managed_position"] = position
				break
			}
		}
	}
	if orders, orderErr := ownership.ListOrders(ctrl.Ctx.Request.Context(), futuresownership.OwnerAgentTrade, 500); orderErr == nil {
		managedOrders := make([]models.FuturesManagedOrder, 0)
		for _, order := range orders {
			if order.SourceRef == proposal.ProposalID {
				managedOrders = append(managedOrders, order)
			}
		}
		data["managed_orders"] = managedOrders
	}
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

func (ctrl *AgentTradeController) Close() {
	proposal, result, err := tradeService().Close(ctrl.Ctx.Request.Context(), ctrl.proposalID(), "web_admin")
	if err != nil {
		ctrl.Ctx.Resp(map[string]any{"code": 400, "data": map[string]any{"proposal": proposal, "close": result}, "msg": err.Error()})
		return
	}
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": map[string]any{"proposal": proposal, "close": result}, "msg": "success"})
}

func (ctrl *AgentTradeController) Ownership() {
	ctx := ctrl.Ctx.Request.Context()
	owner := strings.TrimSpace(ctrl.GetString("owner"))
	ownership := futuresownership.DefaultService()
	positions, err := ownership.ListPositions(ctx, owner, false)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	orders, err := ownership.ListOrders(ctx, owner, 200)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(500, nil, err.Error()))
		return
	}
	data := map[string]any{"positions": positions, "orders": orders}
	overview, overviewErr := futuresownership.DefaultReconciler().PositionOverview(ctx)
	if overviewErr == nil {
		data["account_positions"] = overview
	} else {
		data["account_positions"] = []any{}
		data["account_error"] = overviewErr.Error()
	}
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": data, "msg": "success"})
}

func (ctrl *AgentTradeController) ReconcileOwnership() {
	ctx := ctrl.Ctx.Request.Context()
	owner := strings.TrimSpace(ctrl.GetString("owner"))
	reconciler := futuresownership.DefaultReconciler()
	if owner != "" {
		summary, err := reconciler.ReconcileOwner(ctx, owner)
		if err != nil {
			ctrl.Ctx.Resp(map[string]any{"code": 400, "data": summary, "msg": err.Error()})
			return
		}
		ctrl.Ctx.Resp(map[string]any{"code": 200, "data": []futuresownership.ReconcileSummary{summary}, "msg": "success"})
		return
	}
	summaries, err := reconciler.ReconcileAll(ctx)
	if err != nil {
		ctrl.Ctx.Resp(map[string]any{"code": 500, "data": summaries, "msg": err.Error()})
		return
	}
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": summaries, "msg": "success"})
}

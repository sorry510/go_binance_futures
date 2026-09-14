package controllers

import (
	"strconv"
	"strings"
	"time"

	agenttrade "go_binance_futures/service/agenttrade"
	opportunityservice "go_binance_futures/service/opportunity"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/server/web"
)

type AgentOpportunityController struct{ web.Controller }

func (ctrl *AgentOpportunityController) opportunityID() string {
	return strings.TrimSpace(ctrl.Ctx.Input.Param(":opportunityId"))
}

func (ctrl *AgentOpportunityController) List() {
	ctx := ctrl.Ctx.Request.Context()
	service := opportunityservice.DefaultService()
	_, _ = service.Expire(ctx)
	page, _ := strconv.Atoi(ctrl.GetString("page", "1"))
	limit, _ := strconv.Atoi(ctrl.GetString("limit", "20"))
	result, err := service.List(ctx, opportunityservice.ListOptions{
		Status: ctrl.GetString("status"), AnalysisStatus: ctrl.GetString("analysis_status"),
		Symbol: ctrl.GetString("symbol"), SourceType: ctrl.GetString("source_type"), Page: page, Limit: limit,
	})
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(500, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": result, "msg": "success"})
}

func (ctrl *AgentOpportunityController) Get() {
	ctx := ctrl.Ctx.Request.Context()
	service := opportunityservice.DefaultService()
	_, _ = service.Expire(ctx)
	row, err := service.Get(ctx, ctrl.opportunityID())
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(404, nil, "opportunity not found"))
		return
	}
	data := map[string]any{"opportunity": row, "proposal_eligible": opportunityservice.CanCreateProposal(row, time.Now().UTC())}
	if row.AnalysisTaskID != "" {
		if proposal, proposalErr := (agenttrade.Store{}).FindBySourceTask(ctx, row.AnalysisTaskID); proposalErr == nil {
			data["trade_proposal"] = proposal
		} else if proposalErr != orm.ErrNoRows {
			ctrl.Ctx.Resp(utils.ResJson(500, nil, proposalErr.Error()))
			return
		}
	}
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": data, "msg": "success"})
}

func (ctrl *AgentOpportunityController) Review() {
	row, err := opportunityservice.DefaultService().MarkReviewed(ctrl.Ctx.Request.Context(), ctrl.opportunityID())
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": row, "msg": "success"})
}

func (ctrl *AgentOpportunityController) CreateProposal() {
	proposal, err := opportunityservice.DefaultService().CreateTradeProposal(ctrl.Ctx.Request.Context(), ctrl.opportunityID(), agenttrade.DefaultService())
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": proposal, "msg": "success"})
}

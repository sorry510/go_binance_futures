package controllers

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/beego/beego/v2/server/web"
	agentapp "go_binance_futures/agent/app"
	workflowservice "go_binance_futures/service/workflow"
	"go_binance_futures/utils"
)

type AgentWorkflowController struct{ web.Controller }
type workflowStartRequest struct {
	Workflow string          `json:"workflow"`
	Input    json.RawMessage `json:"input"`
}

func (ctrl *AgentWorkflowController) Start() {
	var req workflowStartRequest
	if err := json.Unmarshal(ctrl.Ctx.Input.RequestBody, &req); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "请求格式错误: "+err.Error()))
		return
	}
	service, err := agentapp.DefaultWorkflowService()
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(500, nil, err.Error()))
		return
	}
	run, err := service.Start(ctrl.Ctx.Request.Context(), strings.TrimSpace(req.Workflow), req.Input)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": run, "msg": "accepted"})
}
func (ctrl *AgentWorkflowController) Get() {
	service, err := agentapp.DefaultWorkflowService()
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(500, nil, err.Error()))
		return
	}
	run, err := service.Get(ctrl.Ctx.Request.Context(), ctrl.Ctx.Input.Param(":id"))
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(404, nil, "workflow run not found"))
		return
	}
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": run, "msg": "success"})
}
func (ctrl *AgentWorkflowController) List() {
	service, err := agentapp.DefaultWorkflowService()
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(500, nil, err.Error()))
		return
	}
	page, _ := strconv.Atoi(ctrl.GetString("page", "1"))
	limit, _ := strconv.Atoi(ctrl.GetString("limit", "20"))
	result, err := service.List(ctrl.Ctx.Request.Context(), workflowservice.ListOptions{Workflow: ctrl.GetString("workflow"), Status: ctrl.GetString("status"), Page: page, Limit: limit})
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": result, "msg": "success"})
}

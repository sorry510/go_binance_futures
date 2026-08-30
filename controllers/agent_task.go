package controllers

import (
	"encoding/json"
	"strconv"
	"strings"

	agentapp "go_binance_futures/agent/app"
	agentruntime "go_binance_futures/agent/runtime"
	alertpipeline "go_binance_futures/service/alertpipeline"
	symbolanalysisservice "go_binance_futures/service/symbolanalysis"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/server/web"
)

type AgentController struct {
	web.Controller
}

type agentTaskRequest struct {
	Skill string          `json:"skill"`
	Input json.RawMessage `json:"input"`
}

func (ctrl *AgentController) StartTask() {
	var request agentTaskRequest
	if err := json.Unmarshal(ctrl.Ctx.Input.RequestBody, &request); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "请求格式错误: "+err.Error()))
		return
	}
	request.Skill = strings.TrimSpace(request.Skill)
	if request.Skill == "" || len(request.Input) == 0 || string(request.Input) == "null" {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "skill 和 input 不能为空"))
		return
	}
	manager, err := agentapp.DefaultManager()
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(500, nil, "初始化 Agent Manager 失败: "+err.Error()))
		return
	}
	item, err := manager.Start(agentruntime.Request{Skill: request.Skill, Input: string(request.Input)})
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": item, "msg": "accepted"})
}

func (ctrl *AgentController) GetTask() {
	manager, err := agentapp.DefaultManager()
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(500, nil, "初始化 Agent Manager 失败: "+err.Error()))
		return
	}
	item, err := manager.Get(ctrl.Ctx.Request.Context(), ctrl.Ctx.Input.Param(":taskId"))
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(404, nil, "task not found"))
		return
	}
	_ = agentapp.EnsureCompletion(item)
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": item, "msg": "success"})
}

func (ctrl *AgentController) GetSymbolAnalysisHistory() {
	page, _ := strconv.Atoi(ctrl.GetString("page", "1"))
	limit, _ := strconv.Atoi(ctrl.GetString("limit", "20"))
	result, err := (symbolanalysisservice.HistoryService{}).List(
		ctrl.Ctx.Request.Context(),
		symbolanalysisservice.HistoryListOptions{
			Symbol: ctrl.GetString("symbol"), Status: ctrl.GetString("status"),
			Page: page, Limit: limit,
		},
	)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": result, "msg": "success"})
}

func (ctrl *AgentController) GetAlertPipelineStatus() {
	limit, _ := strconv.Atoi(ctrl.GetString("limit", "20"))
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	ctrl.Ctx.Resp(map[string]interface{}{
		"code": 200,
		"data": alertpipeline.DefaultStatus(limit),
		"msg":  "success",
	})
}

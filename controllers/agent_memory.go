package controllers

import (
	"encoding/json"
	"strconv"
	"strings"

	"go_binance_futures/agent/memory"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/server/web"
)

type AgentMemoryController struct{ web.Controller }

type memoryCreateRequest struct {
	Type       memory.Type  `json:"type"`
	Scope      memory.Scope `json:"scope"`
	Confidence float64      `json:"confidence"`
	Content    string       `json:"content"`
	ExpiresAt  int64        `json:"expires_at"`
	Candidate  bool         `json:"candidate"`
}

type memoryUpdateRequest struct {
	Scope      memory.Scope `json:"scope"`
	Confidence float64      `json:"confidence"`
	Content    string       `json:"content"`
	ExpiresAt  int64        `json:"expires_at"`
}

func (ctrl *AgentMemoryController) List() {
	page, _ := strconv.Atoi(ctrl.GetString("page", "1"))
	limit, _ := strconv.Atoi(ctrl.GetString("limit", "20"))
	result, err := memory.NewORMStore().List(ctrl.Ctx.Request.Context(), memory.ListOptions{
		Type: ctrl.GetString("type"), Status: ctrl.GetString("status"), User: ctrl.GetString("user"),
		Skill: ctrl.GetString("skill"), Symbol: ctrl.GetString("symbol"), Strategy: ctrl.GetString("strategy"),
		SourceTaskID: ctrl.GetString("source_task_id"), IncludeExpired: ctrl.GetString("include_expired") == "1",
		Page: page, Limit: limit,
	})
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": result, "msg": "success"})
}

func (ctrl *AgentMemoryController) Create() {
	var request memoryCreateRequest
	if err := json.Unmarshal(ctrl.Ctx.Input.RequestBody, &request); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "请求格式错误: "+err.Error()))
		return
	}
	status := memory.StatusActive
	if request.Candidate {
		status = memory.StatusCandidate
	}
	item, err := memory.NewORMStore().Create(ctrl.Ctx.Request.Context(), memory.CreateInput{
		Type: request.Type, Scope: request.Scope, Confidence: request.Confidence,
		Status: status, Content: request.Content, ExpiresAt: request.ExpiresAt,
	})
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": item, "msg": "success"})
}

func (ctrl *AgentMemoryController) Update() {
	id, ok := ctrl.memoryID()
	if !ok {
		return
	}
	var request memoryUpdateRequest
	if err := json.Unmarshal(ctrl.Ctx.Input.RequestBody, &request); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "请求格式错误: "+err.Error()))
		return
	}
	item, err := memory.NewORMStore().Update(ctrl.Ctx.Request.Context(), id, memory.UpdateInput{
		Scope: request.Scope, Confidence: request.Confidence, Content: request.Content, ExpiresAt: request.ExpiresAt,
	})
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": item, "msg": "success"})
}

func (ctrl *AgentMemoryController) Delete() {
	id, ok := ctrl.memoryID()
	if !ok {
		return
	}
	if err := memory.NewORMStore().Delete(ctrl.Ctx.Request.Context(), id); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]any{"code": 200, "msg": "success"})
}

func (ctrl *AgentMemoryController) Disable() { ctrl.setStatus(memory.StatusDisabled) }
func (ctrl *AgentMemoryController) Enable()  { ctrl.setStatus(memory.StatusActive) }
func (ctrl *AgentMemoryController) Approve() { ctrl.setStatus(memory.StatusActive) }

func (ctrl *AgentMemoryController) setStatus(status string) {
	id, ok := ctrl.memoryID()
	if !ok {
		return
	}
	item, err := memory.NewORMStore().SetStatus(ctrl.Ctx.Request.Context(), id, status)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": item, "msg": "success"})
}

func (ctrl *AgentMemoryController) memoryID() (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(ctrl.Ctx.Input.Param(":id")), 10, 64)
	if err != nil || id <= 0 {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "invalid memory id"))
		return 0, false
	}
	return id, true
}

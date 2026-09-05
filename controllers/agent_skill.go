package controllers

import (
	"encoding/json"
	"strconv"
	"strings"

	agentapp "go_binance_futures/agent/app"
	"go_binance_futures/agent/skillconfig"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/server/web"
)

type AgentSkillController struct {
	web.Controller
}

type agentSkillCreateRequest struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Enabled     *int   `json:"enabled"`
}

func (ctrl *AgentSkillController) Get() {
	items, err := (skillconfig.Store{}).List(ctrl.Ctx.Request.Context())
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(500, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": items, "msg": "success"})
}

func (ctrl *AgentSkillController) GetImplementations() {
	ctrl.Ctx.Resp(map[string]interface{}{
		"code": 200, "data": agentapp.AvailableSkillImplementations(), "msg": "success",
	})
}

func (ctrl *AgentSkillController) Post() {
	var request agentSkillCreateRequest
	if err := json.Unmarshal(ctrl.Ctx.Input.RequestBody, &request); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "请求格式错误: "+err.Error()))
		return
	}
	request.Name = strings.TrimSpace(request.Name)
	implementation, ok := agentapp.SkillImplementationByName(request.Name)
	if !ok {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "后端不存在该 Skill implementation"))
		return
	}
	enabled := 1
	if request.Enabled != nil {
		enabled = *request.Enabled
	}
	if strings.TrimSpace(request.DisplayName) == "" {
		request.DisplayName = implementation.DisplayName
	}
	if strings.TrimSpace(request.Description) == "" {
		request.Description = implementation.Description
	}
	item, err := (skillconfig.Store{}).Create(ctrl.Ctx.Request.Context(), skillconfig.CreateInput{
		Name: request.Name, DisplayName: request.DisplayName, Description: request.Description,
		Enabled: enabled,
	})
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": item, "msg": "success"})
}

func parseAgentSkillID(ctrl *AgentSkillController) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(ctrl.Ctx.Input.Param(":id")), 10, 64)
}
func (ctrl *AgentSkillController) Put() {
	id, err := parseAgentSkillID(ctrl)
	if err != nil || id <= 0 {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "skill id 无效"))
		return
	}
	var request skillconfig.UpdateInput
	if err := json.Unmarshal(ctrl.Ctx.Input.RequestBody, &request); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "请求格式错误: "+err.Error()))
		return
	}
	item, err := (skillconfig.Store{}).Update(ctrl.Ctx.Request.Context(), id, request)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	if item.Type == "portable" {
		_ = agentapp.SyncDefaultPortableSkills(ctrl.Ctx.Request.Context())
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": item, "msg": "success"})
}

func (ctrl *AgentSkillController) Delete() {
	id, err := parseAgentSkillID(ctrl)
	if err != nil || id <= 0 {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "skill id 无效"))
		return
	}
	if err := (skillconfig.Store{}).Delete(ctrl.Ctx.Request.Context(), id); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	_ = agentapp.SyncDefaultPortableSkills(ctrl.Ctx.Request.Context())
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "msg": "success"})
}

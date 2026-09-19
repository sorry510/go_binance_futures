package controllers

import (
	"encoding/json"
	"strconv"
	"strings"

	agentapp "go_binance_futures/agent/app"
	"go_binance_futures/agent/conversation"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/server/web"
)

type AgentChatController struct {
	web.Controller
}

type agentChatMessageRequest struct {
	SkillMode string `json:"skill_mode,omitempty"`
	Skill     string `json:"skill,omitempty"`
	Content   string `json:"content"`
	Symbol    string `json:"symbol,omitempty"`
	ModelID   *int64 `json:"model_id,omitempty"`
}

func (ctrl *AgentChatController) ListConversations() {
	page, _ := strconv.Atoi(ctrl.GetString("page", "1"))
	limit, _ := strconv.Atoi(ctrl.GetString("limit", "30"))
	result, err := agentapp.ChatConversationStore().ListChats(ctrl.Ctx.Request.Context(), conversation.ListOptions{Page: page, Limit: limit})
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(500, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": result, "msg": "success"})
}
func (ctrl *AgentChatController) CreateConversation() {
	item, err := agentapp.ChatConversationStore().Create(ctrl.Ctx.Request.Context(), conversation.ChatSkill)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(500, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": item, "msg": "success"})
}

type agentChatConversationUpdateRequest struct {
	Title string `json:"title"`
}

func (ctrl *AgentChatController) UpdateConversation() {
	id := strings.TrimSpace(ctrl.Ctx.Input.Param(":id"))
	item, err := agentapp.ChatConversationStore().Get(ctrl.Ctx.Request.Context(), id)
	if err != nil || item.Skill != conversation.ChatSkill {
		ctrl.Ctx.Resp(utils.ResJson(404, nil, "chat conversation not found"))
		return
	}
	var request agentChatConversationUpdateRequest
	if err := json.Unmarshal(ctrl.Ctx.Input.RequestBody, &request); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "请求格式错误: "+err.Error()))
		return
	}
	if strings.TrimSpace(request.Title) == "" {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "对话标题不能为空"))
		return
	}
	if err := agentapp.ChatConversationStore().SetTitle(ctrl.Ctx.Request.Context(), id, request.Title); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(500, nil, err.Error()))
		return
	}
	updated, err := agentapp.ChatConversationStore().Get(ctrl.Ctx.Request.Context(), id)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(500, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": updated, "msg": "success"})
}

func (ctrl *AgentChatController) DeleteConversation() {
	id := strings.TrimSpace(ctrl.Ctx.Input.Param(":id"))
	item, err := agentapp.ChatConversationStore().Get(ctrl.Ctx.Request.Context(), id)
	if err != nil || item.Skill != conversation.ChatSkill {
		ctrl.Ctx.Resp(utils.ResJson(404, nil, "chat conversation not found"))
		return
	}
	if err := agentapp.ChatConversationStore().DeleteChat(ctrl.Ctx.Request.Context(), id); err != nil {
		if strings.Contains(err.Error(), "running task") {
			ctrl.Ctx.Resp(utils.ResJson(409, nil, err.Error()))
			return
		}
		ctrl.Ctx.Resp(utils.ResJson(500, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "msg": "success"})
}

func (ctrl *AgentChatController) Messages() {
	id := strings.TrimSpace(ctrl.Ctx.Input.Param(":id"))
	item, err := agentapp.ChatConversationStore().Get(ctrl.Ctx.Request.Context(), id)
	if err != nil || item.Skill != conversation.ChatSkill {
		ctrl.Ctx.Resp(utils.ResJson(404, nil, "chat conversation not found"))
		return
	}
	messages, err := agentapp.ChatConversationStore().MessagesDetailed(ctrl.Ctx.Request.Context(), id)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(500, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": messages, "msg": "success"})
}

func (ctrl *AgentChatController) SendMessage() {
	id := strings.TrimSpace(ctrl.Ctx.Input.Param(":id"))
	var request agentChatMessageRequest
	if err := json.Unmarshal(ctrl.Ctx.Input.RequestBody, &request); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "请求格式错误: "+err.Error()))
		return
	}
	item, err := agentapp.StartChatMessageWithOptions(ctrl.Ctx.Request.Context(), id, agentapp.ChatMessageOptions{
		SkillMode: request.SkillMode, Skill: request.Skill, Content: request.Content, Symbol: request.Symbol, ModelConfigID: request.ModelID,
	})
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{
			"conversation_id": id,
			"task_id":         item.ID,
			"status":          item.Status,
			"skill":           item.Skill,
			"model_config_id": item.ModelConfigID,
		},
		"msg": "accepted",
	})
}

func (ctrl *AgentChatController) ConversationSkills() {
	id := strings.TrimSpace(ctrl.Ctx.Input.Param(":id"))
	items, err := agentapp.ConversationChatSkillNames(ctrl.Ctx.Request.Context(), id)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": items, "msg": "success"})
}

type agentChatSkillRequest struct {
	Skill string `json:"skill"`
}

func (ctrl *AgentChatController) AddConversationSkill() {
	id := strings.TrimSpace(ctrl.Ctx.Input.Param(":id"))
	var request agentChatSkillRequest
	if err := json.Unmarshal(ctrl.Ctx.Input.RequestBody, &request); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "请求格式错误: "+err.Error()))
		return
	}
	if err := agentapp.AttachConversationChatSkill(ctrl.Ctx.Request.Context(), id, request.Skill); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "msg": "success"})
}

func (ctrl *AgentChatController) RemoveConversationSkill() {
	id := strings.TrimSpace(ctrl.Ctx.Input.Param(":id"))
	skillName := strings.TrimSpace(ctrl.Ctx.Input.Param(":skillName"))
	if err := agentapp.RemoveConversationChatSkill(ctrl.Ctx.Request.Context(), id, skillName); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "msg": "success"})
}

type agentChatModelRequest struct {
	ModelID int64 `json:"model_id"`
}

func (ctrl *AgentChatController) UpdateConversationModel() {
	id := strings.TrimSpace(ctrl.Ctx.Input.Param(":id"))
	var request agentChatModelRequest
	if err := json.Unmarshal(ctrl.Ctx.Input.RequestBody, &request); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "请求格式错误: "+err.Error()))
		return
	}
	if err := agentapp.UpdateConversationChatModel(ctrl.Ctx.Request.Context(), id, request.ModelID); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "msg": "success"})
}

func (ctrl *AgentChatController) Skills() {
	items, err := agentapp.ChatSkills(ctrl.Ctx.Request.Context())
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(500, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": items, "msg": "success"})
}

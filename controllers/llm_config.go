package controllers

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"go_binance_futures/llm"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/server/web"
)

type LLMConfigController struct {
	web.Controller
}

type llmConfigTestRequest struct {
	ID int64 `json:"id"`
	llm.ConfigInput
}

func (ctrl *LLMConfigController) Get() {
	items, err := (llm.Store{}).List(ctrl.Ctx.Request.Context())
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(500, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": items, "msg": "success"})
}
func (ctrl *LLMConfigController) GetPresets() {
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": llm.ProviderPresets(), "msg": "success"})
}

func (ctrl *LLMConfigController) Post() {
	var input llm.ConfigInput
	if err := json.Unmarshal(ctrl.Ctx.Input.RequestBody, &input); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "请求格式错误: "+err.Error()))
		return
	}
	item, err := (llm.Store{}).Create(ctrl.Ctx.Request.Context(), input)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": item, "msg": "success"})
}

func parseLLMConfigID(ctrl *LLMConfigController) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(ctrl.Ctx.Input.Param(":id")), 10, 64)
}
func (ctrl *LLMConfigController) Put() {
	id, err := parseLLMConfigID(ctrl)
	if err != nil || id <= 0 {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "LLM config id 无效"))
		return
	}
	var input llm.ConfigInput
	if err := json.Unmarshal(ctrl.Ctx.Input.RequestBody, &input); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "请求格式错误: "+err.Error()))
		return
	}
	item, err := (llm.Store{}).Update(ctrl.Ctx.Request.Context(), id, input)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": item, "msg": "success"})
}

func (ctrl *LLMConfigController) Delete() {
	id, err := parseLLMConfigID(ctrl)
	if err != nil || id <= 0 {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "LLM config id 无效"))
		return
	}
	if err := (llm.Store{}).Delete(ctrl.Ctx.Request.Context(), id); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "msg": "success"})
}
func (ctrl *LLMConfigController) Test() {
	var request llmConfigTestRequest
	if err := json.Unmarshal(ctrl.Ctx.Input.RequestBody, &request); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "请求格式错误: "+err.Error()))
		return
	}
	preservedAPIKey := ""
	if request.ID > 0 && strings.TrimSpace(request.APIKey) == "" {
		row, err := (llm.Store{}).Get(ctrl.Ctx.Request.Context(), request.ID)
		if err != nil {
			ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
			return
		}
		preservedAPIKey = row.APIKey
	}
	cfg, err := llm.BuildConfig(request.ConfigInput, preservedAPIKey)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	client, err := llm.NewClient(cfg)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	testCtx, cancel := context.WithTimeout(ctrl.Ctx.Request.Context(), cfg.Timeout)
	defer cancel()
	response, err := client.Generate(testCtx, llm.Request{
		Messages:  []llm.Message{{Role: llm.RoleUser, Content: "Reply with OK only."}},
		MaxTokens: 16,
	})
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	content := strings.TrimSpace(response.Content)
	if len(content) > 500 {
		content = content[:500]
	}
	ctrl.Ctx.Resp(map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{
			"provider": client.Provider(), "model": response.Model, "content": content,
		},
		"msg": "connection success",
	})
}

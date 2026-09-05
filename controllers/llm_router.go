package controllers

import (
	"encoding/json"

	"go_binance_futures/agent/modelgateway"
	"go_binance_futures/llm"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/server/web"
)

type LLMRouterController struct{ web.Controller }

func (ctrl *LLMRouterController) Get() {
	settings, err := (llm.Store{}).RouterSettings(ctrl.Ctx.Request.Context())
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(500, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{
			"settings": settings,
			"health":   modelgateway.DefaultHealth().Snapshots(),
		},
		"msg": "success",
	})
}

func (ctrl *LLMRouterController) Put() {
	var input llm.RouterSettings
	if err := json.Unmarshal(ctrl.Ctx.Input.RequestBody, &input); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "请求格式错误: "+err.Error()))
		return
	}
	settings, err := (llm.Store{}).UpdateRouterSettings(ctrl.Ctx.Request.Context(), input)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": settings, "msg": "success"})
}

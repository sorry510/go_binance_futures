package controllers

import (
	"os"

	"go_binance_futures/utils"

	"github.com/beego/beego/v2/server/web"
)

type ConfigController struct {
	web.Controller
}

func (ctrl *ConfigController) Get() {
	content, err := os.ReadFile("./conf/app.conf")
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "获取错误"))
		return
	}

	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"data": map[string]interface{}{
			"content": string(content),
		},
		"msg": "success",
	})
}

func (ctrl *ConfigController) Edit() {
	configContent := new(ConfigContent)
	ctrl.BindJSON(&configContent)
	
	err := os.WriteFile("./conf/app.conf", []byte(configContent.Code), 0644)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "获取错误"))
		return
	}

	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"msg": "success",
	})
}

type ConfigContent struct {
	Code string `json:"code"`
}

package controllers

import (
	"os/exec"

	"github.com/beego/beego/v2/core/config"
	"github.com/beego/beego/v2/server/web"
)

type CommandController struct {
	web.Controller
}

// git pull
func (ctrl *CommandController) GitPull() {
	cmd := exec.Command("git", "pull")

	// 获取标准输出
	output, err := cmd.Output()
	if err != nil {
		ctrl.Ctx.Resp(map[string]interface{} {
			"code": 400,
			"data": err,
			"msg": "error",
		})
	}
	
	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"data": string(output),
		"msg": "success",
	})
}

// 开启后台任务
func (ctrl *CommandController) Start() {
	commend_start, _ := config.String("web::commend_start")
	exec.Command(commend_start)
	
	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"msg": "success",
	})
}

// 关闭后台任务
func (ctrl *CommandController) Stop() {
	commend_start, _ := config.String("web::commend_stop")
	exec.Command(commend_start)
	
	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"msg": "success",
	})
}

// pm2-log
func (ctrl *CommandController) Pm2Log() {
	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"msg": "success",
	})
}
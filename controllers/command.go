package controllers

import (
	"io"
	"os/exec"

	"github.com/beego/beego/v2/core/config"
	"github.com/beego/beego/v2/server/web"
)

type CommandController struct {
	web.Controller
}

// git pull
func (ctrl *CommandController) GitPull() {
	cmd := exec.Command("bash", "-c", "git pull")

	// 获取标准输出
	output, err := cmd.CombinedOutput()
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
	cmd := exec.Command("bash", "-c", commend_start)
	cmd.Start() // 异步执行，不要获取结果，因为重启时此进程会挂掉导致http请求失败
	
	// // 获取标准输出
	// output, err := cmd.CombinedOutput()
	// if err != nil {
	// 	ctrl.Ctx.Resp(map[string]interface{} {
	// 		"code": 400,
	// 		"data": err,
	// 		"msg": "error",
	// 	})
	// }
	
	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"msg": "success",
	})
}

// 关闭后台任务
func (ctrl *CommandController) Stop() {
	commend_stop, _ := config.String("web::commend_stop")
	cmd := exec.Command("bash", "-c", commend_stop)
	cmd.Start() // 异步执行，不要获取结果，因为重启时此进程会挂掉导致http请求失败
	
	// // 获取标准输出
	// output, err := cmd.CombinedOutput()
	// if err != nil {
	// 	ctrl.Ctx.Resp(map[string]interface{} {
	// 		"code": 400,
	// 		"data": err,
	// 		"msg": "error",
	// 	})
	// }
	
	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"msg": "success",
	})
}

// pm2-log
func (ctrl *CommandController) Pm2Log() {
	commend_log, _ := config.String("web::commend_log")
	paramsKey := ctrl.GetString("key")
	if paramsKey != "sorry510" {
		ctrl.Ctx.Resp(map[string]interface{} {
			"code": 400,
			"msg": "error",
		})
		return
	}
	
	cmd := exec.Command("bash", "-c", commend_log)
	
	// 创建标准输出管道，用于读取命令的输出
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		ctrl.Ctx.Resp(map[string]interface{} {
			"code": 400,
			"msg": err,
		})
		return
	}
	
	ctrl.Ctx.ResponseWriter.Header().Set("Content-Type", "text/plain; charset=utf-8")
	// 开始执行命令
	if err := cmd.Start(); err != nil {
		ctrl.Ctx.Resp(map[string]interface{} {
			"code": 400,
			"msg": err,
		})
		return
	}
	// 直接从命令输出流读取数据并写入响应
	if _, err := io.Copy(ctrl.Ctx.ResponseWriter, stdout); err != nil {
		ctrl.Ctx.Resp(map[string]interface{} {
			"code": 400,
			"msg": err,
		})
		return
	}
}
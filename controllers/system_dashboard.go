package controllers

import (
	agentapp "go_binance_futures/agent/app"
	"go_binance_futures/service/systemhealth"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/server/web"
)

type SystemDashboardController struct{ web.Controller }

func (ctrl *SystemDashboardController) Health() {
	report, err := (systemhealth.Service{}).Report(ctrl.Ctx.Request.Context(), systemhealth.Options{
		CheckBinanceREST:          true,
		SchedulerRuntimeAvailable: true,
		SchedulerJobs:             agentapp.DefaultSchedulerStatus(),
	})
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(500, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": report, "msg": "success"})
}

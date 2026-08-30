package controllers

import (
	"strconv"
	"strings"

	liquidationservice "go_binance_futures/service/liquidation"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/server/web"
)

type FuturesLiquidationOrderController struct {
	web.Controller
}

func (ctrl *FuturesLiquidationOrderController) Get() {
	page, _ := strconv.Atoi(ctrl.GetString("page", "1"))
	limit, _ := strconv.Atoi(ctrl.GetString("limit", "20"))
	minNotional, _ := strconv.ParseFloat(strings.TrimSpace(ctrl.GetString("min_notional")), 64)
	result, err := (liquidationservice.Service{}).List(ctrl.Ctx.Request.Context(), liquidationservice.ListOptions{
		Symbol: ctrl.GetString("symbol"), Side: ctrl.GetString("side"),
		StartTime: liquidationservice.ParseTimestamp(ctrl.GetString("start_time")), EndTime: liquidationservice.ParseTimestamp(ctrl.GetString("end_time")),
		MinNotional: minNotional, Page: page, Limit: limit, DefaultLimit: 20, MaxLimit: 10000,
	})
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": map[string]interface{}{"total": result.Total, "list": result.List}, "msg": "success"})
}

func parseTimestamp(value string) int64 { return liquidationservice.ParseTimestamp(value) }

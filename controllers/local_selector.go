package controllers

import (
	"strconv"

	"go_binance_futures/scanner"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/server/web"
)

type LocalSelectorController struct{ web.Controller }

func (ctrl *LocalSelectorController) SmartLocalV2() {
	limit := scanner.DefaultSmartLocalV2Limit
	if raw := ctrl.GetString("limit"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 {
			limit = value
		}
	}
	mode := scanner.SmartLocalV2ModeTrade
	if ctrl.GetString("mode") == string(scanner.SmartLocalV2ModeTest) {
		mode = scanner.SmartLocalV2ModeTest
	}
	result, err := scanner.SmartLocalV2ForMode(ctrl.Ctx.Request.Context(), mode, scanner.SmartLocalV2Options{Limit: limit, IncludeExcluded: true})
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(500, nil, err.Error()))
		return
	}
	result.NextBatch, result.Rotation = scanner.PeekSmartLocalV2BatchFor(string(mode), result.Candidates, scanner.DefaultSmartLocalV2BatchSize)
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": result, "msg": "success"})
}

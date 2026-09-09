package controllers

import (
	"encoding/json"
	"strings"

	"go_binance_futures/service/historicalmarket"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/server/web"
)

type AgentHistoricalMarketController struct{ web.Controller }

func (ctrl *AgentHistoricalMarketController) Import() {
	var request historicalmarket.ImportRequest
	if err := json.Unmarshal(ctrl.Ctx.Input.RequestBody, &request); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "invalid request: "+err.Error()))
		return
	}
	if len(request.Klines)+len(request.Funding) == 0 {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "klines or funding is required"))
		return
	}
	if len(request.Klines)+len(request.Funding) > 50000 {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "one import request supports at most 50000 rows"))
		return
	}
	request.Source = strings.TrimSpace(request.Source)
	if request.Source == "" {
		request.Source = "external"
	}
	result, err := historicalmarket.DefaultRepository().Import(ctrl.Ctx.Request.Context(), request)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, map[string]interface{}{"batch_id": result.BatchID, "written_rows": result.WrittenRows, "invalid_rows": result.InvalidRows}, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": result, "msg": "success"})
}

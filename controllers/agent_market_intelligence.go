package controllers

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	marketintelligence "go_binance_futures/service/marketintelligence"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/server/web"
)

type AgentMarketIntelligenceController struct {
	web.Controller
}

type marketIntelligenceIngestRequest struct {
	Events []marketintelligence.EventInput `json:"events"`
}

func (ctrl *AgentMarketIntelligenceController) Get() {
	ctx := ctrl.Ctx.Request.Context()
	symbol := strings.ToUpper(strings.TrimSpace(ctrl.GetString("symbol")))
	if symbol == "" {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "symbol is required"))
		return
	}
	limit, _ := strconv.Atoi(ctrl.GetString("limit", "100"))
	startTime, _ := strconv.ParseInt(ctrl.GetString("start_time"), 10, 64)
	endTime, _ := strconv.ParseInt(ctrl.GetString("end_time"), 10, 64)
	service := marketintelligence.DefaultService()
	var result marketintelligence.Snapshot
	var err error
	if startTime > 0 || endTime > 0 {
		result, err = service.Timeline(ctx, symbol, startTime, endTime, limit)
	} else {
		windowMinutes, _ := strconv.Atoi(ctrl.GetString("window_minutes", "1440"))
		if windowMinutes < 1 || windowMinutes > 10080 {
			ctrl.Ctx.Resp(utils.ResJson(400, nil, "window_minutes must be 1..10080"))
			return
		}
		result, err = service.Snapshot(ctx, symbol, time.Duration(windowMinutes)*time.Minute, limit)
	}
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": result, "msg": "success"})
}

func (ctrl *AgentMarketIntelligenceController) IngestEvents() {
	var request marketIntelligenceIngestRequest
	if err := json.Unmarshal(ctrl.Ctx.Input.RequestBody, &request); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "invalid request: "+err.Error()))
		return
	}
	if len(request.Events) == 0 {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "events are required"))
		return
	}
	if len(request.Events) > 200 {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "at most 200 events can be ingested per request"))
		return
	}
	service := marketintelligence.DefaultService()
	inserted, merged := 0, 0
	items := make([]marketintelligence.Event, 0, len(request.Events))
	for _, input := range request.Events {
		item, created, err := service.IngestEvent(ctrl.Ctx.Request.Context(), input)
		if err != nil {
			ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
			return
		}
		items = append(items, item)
		if created {
			inserted++
		} else {
			merged++
		}
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": map[string]interface{}{"inserted": inserted, "merged": merged, "events": items}, "msg": "success"})
}

package controllers

import (
	"context"
	"strconv"
	"strings"

	"go_binance_futures/agent/observability"
	agenttask "go_binance_futures/agent/task"
	"go_binance_futures/models"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/server/web"
)

type AgentObservabilityController struct{ web.Controller }

type agentObservationTraceView struct {
	models.AgentObservation
	ParentTaskID string `json:"parent_task_id,omitempty"`
	TeamRunID    string `json:"team_run_id,omitempty"`
	TeamName     string `json:"team_name,omitempty"`
	TeamRole     string `json:"team_role,omitempty"`
}

func enrichObservationTeamLinks(ctx context.Context, rows []models.AgentObservation) []agentObservationTraceView {
	views := make([]agentObservationTraceView, 0, len(rows))
	store := agenttask.NewORMStore()
	cache := map[string]*agenttask.Task{}
	for _, row := range rows {
		view := agentObservationTraceView{AgentObservation: row}
		taskID := strings.TrimSpace(row.TaskID)
		if taskID != "" {
			item := cache[taskID]
			if item == nil {
				if loaded, err := store.Get(ctx, taskID); err == nil {
					item = loaded
					cache[taskID] = loaded
				}
			}
			if item != nil {
				view.ParentTaskID, view.TeamRunID = item.ParentTaskID, item.TeamRunID
				view.TeamName, view.TeamRole = item.TeamName, item.TeamRole
			}
		}
		views = append(views, view)
	}
	return views
}

func queryInt64(ctrl *web.Controller, name string) int64 {
	value, _ := strconv.ParseInt(strings.TrimSpace(ctrl.GetString(name)), 10, 64)
	return value
}

func queryInt(ctrl *web.Controller, name string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(ctrl.GetString(name)))
	if err != nil {
		return fallback
	}
	return value
}

func (ctrl *AgentObservabilityController) Summary() {
	result, err := (observability.Store{}).Summary(ctrl.Ctx.Request.Context(), queryInt64(&ctrl.Controller, "start_time"), queryInt64(&ctrl.Controller, "end_time"))
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": result, "msg": "success"})
}

func (ctrl *AgentObservabilityController) Traces() {
	result, err := (observability.Store{}).ListTraces(ctrl.Ctx.Request.Context(), observability.TraceListOptions{
		TaskID: ctrl.GetString("task_id"), Skill: ctrl.GetString("skill"), Type: ctrl.GetString("type"), Status: ctrl.GetString("status"), ToolSource: ctrl.GetString("tool_source"),
		StartTime: queryInt64(&ctrl.Controller, "start_time"), EndTime: queryInt64(&ctrl.Controller, "end_time"), Page: queryInt(&ctrl.Controller, "page", 1), Limit: queryInt(&ctrl.Controller, "limit", 20),
	})
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	data := map[string]any{
		"page": result.Page, "limit": result.Limit, "total": result.Total,
		"list": enrichObservationTeamLinks(ctrl.Ctx.Request.Context(), result.List),
	}
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": data, "msg": "success"})
}

func (ctrl *AgentObservabilityController) Changes() {
	result, err := (observability.Store{}).ListChanges(ctrl.Ctx.Request.Context(), observability.ChangeListOptions{
		Category: ctrl.GetString("category"), EntityType: ctrl.GetString("entity_type"), EntityName: ctrl.GetString("entity_name"), ChangeType: ctrl.GetString("change_type"), Status: ctrl.GetString("status"),
		StartTime: queryInt64(&ctrl.Controller, "start_time"), EndTime: queryInt64(&ctrl.Controller, "end_time"), Page: queryInt(&ctrl.Controller, "page", 1), Limit: queryInt(&ctrl.Controller, "limit", 20),
	})
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": result, "msg": "success"})
}

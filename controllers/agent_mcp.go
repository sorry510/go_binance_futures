package controllers

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	agentapp "go_binance_futures/agent/app"
	"go_binance_futures/agent/mcpclient"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/server/web"
)

type AgentMCPController struct {
	web.Controller
}

func syncDefaultMCPRuntime(ctx context.Context) error {
	if _, err := agentapp.DefaultMCPGateway(); err != nil {
		return err
	}
	return agentapp.SyncDefaultMCPTools(ctx)
}

func parseMCPID(ctrl *AgentMCPController) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(ctrl.Ctx.Input.Param(":id")), 10, 64)
}

func (ctrl *AgentMCPController) ListServers() {
	items, err := (mcpclient.Store{}).ListServers(ctrl.Ctx.Request.Context())
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(500, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": items, "msg": "success"})
}
func (ctrl *AgentMCPController) CreateServer() {
	var input mcpclient.ServerInput
	if err := json.Unmarshal(ctrl.Ctx.Input.RequestBody, &input); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "请求格式错误: "+err.Error()))
		return
	}
	item, err := (mcpclient.Store{}).SaveServer(ctrl.Ctx.Request.Context(), 0, input)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": item, "msg": "success"})
}

func (ctrl *AgentMCPController) UpdateServer() {
	id, err := parseMCPID(ctrl)
	if err != nil || id <= 0 {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "MCP server id 无效"))
		return
	}
	var input mcpclient.ServerInput
	if err := json.Unmarshal(ctrl.Ctx.Input.RequestBody, &input); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "请求格式错误: "+err.Error()))
		return
	}
	item, err := (mcpclient.Store{}).SaveServer(ctrl.Ctx.Request.Context(), id, input)
	if err == nil {
		err = syncDefaultMCPRuntime(ctrl.Ctx.Request.Context())
	}
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": item, "msg": "success"})
}
func (ctrl *AgentMCPController) DeleteServer() {
	id, err := parseMCPID(ctrl)
	if err != nil || id <= 0 {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "MCP server id 无效"))
		return
	}
	if err := (mcpclient.Store{}).DeleteServer(ctrl.Ctx.Request.Context(), id); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	if err := syncDefaultMCPRuntime(ctrl.Ctx.Request.Context()); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(500, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "msg": "success"})
}

func (ctrl *AgentMCPController) GetCatalog() {
	id, err := parseMCPID(ctrl)
	if err != nil || id <= 0 {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "MCP server id 无效"))
		return
	}
	catalog, err := (mcpclient.Store{}).Catalog(ctrl.Ctx.Request.Context(), id)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": catalog, "msg": "success"})
}
func (ctrl *AgentMCPController) TestConnection() {
	ctrl.refresh(true)
}

func (ctrl *AgentMCPController) RefreshCatalog() {
	ctrl.refresh(false)
}

func (ctrl *AgentMCPController) refresh(testOnly bool) {
	id, err := parseMCPID(ctrl)
	if err != nil || id <= 0 {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "MCP server id 无效"))
		return
	}
	gateway, err := agentapp.DefaultMCPGateway()
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(500, nil, err.Error()))
		return
	}
	var result mcpclient.RefreshResult
	if testOnly {
		result, err = gateway.TestConnection(ctrl.Ctx.Request.Context(), id)
	} else {
		result, err = gateway.RefreshCatalog(ctrl.Ctx.Request.Context(), id)
	}
	if err == nil && !testOnly {
		err = syncDefaultMCPRuntime(ctrl.Ctx.Request.Context())
	}
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": result, "msg": "success"})
}
func (ctrl *AgentMCPController) UpdateTool() {
	id, err := parseMCPID(ctrl)
	if err != nil || id <= 0 {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "MCP tool id 无效"))
		return
	}
	var input mcpclient.ToolUpdateInput
	if err := json.Unmarshal(ctrl.Ctx.Input.RequestBody, &input); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "请求格式错误: "+err.Error()))
		return
	}
	item, err := (mcpclient.Store{}).UpdateTool(ctrl.Ctx.Request.Context(), id, input)
	if err == nil {
		err = syncDefaultMCPRuntime(ctrl.Ctx.Request.Context())
	}
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": item, "msg": "success"})
}

func (ctrl *AgentMCPController) SavePermission() {
	var input mcpclient.PermissionInput
	if err := json.Unmarshal(ctrl.Ctx.Input.RequestBody, &input); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "请求格式错误: "+err.Error()))
		return
	}
	item, err := (mcpclient.Store{}).SavePermission(ctrl.Ctx.Request.Context(), input)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": item, "msg": "success"})
}

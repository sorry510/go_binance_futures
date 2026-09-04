package mcpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"go_binance_futures/agent/permission"
	agenttools "go_binance_futures/agent/tools"
	"go_binance_futures/models"
)

type RemoteTool struct {
	gateway *Gateway
	server  models.AgentMCPServer
	tool    models.AgentMCPTool
}

func NewRemoteTool(gateway *Gateway, server models.AgentMCPServer, tool models.AgentMCPTool) *RemoteTool {
	return &RemoteTool{gateway: gateway, server: server, tool: tool}
}

func (tool *RemoteTool) Name() string        { return tool.tool.CanonicalName }
func (tool *RemoteTool) Description() string { return tool.tool.Description }
func (tool *RemoteTool) Risk() permission.RiskLevel {
	return permission.RiskLevel(tool.tool.Risk)
}

func (tool *RemoteTool) Metadata() agenttools.Metadata {
	return agenttools.Metadata{
		InputSchema: json.RawMessage(tool.tool.InputSchema), OutputSchema: json.RawMessage(tool.tool.OutputSchema),
		Timeout:        time.Duration(tool.tool.TimeoutMs) * time.Millisecond,
		MaxResultBytes: tool.tool.MaxResultBytes, Idempotent: tool.tool.Idempotent,
		SourceType: "mcp", ProviderRef: "mcp-server:" + strconv.FormatInt(tool.server.ID, 10),
		ProtocolVersion: tool.server.ProtocolVersion, CatalogHash: tool.tool.CatalogHash, SchemaHash: tool.tool.SchemaHash,
		CacheTTL: time.Duration(tool.tool.CacheTTLms) * time.Millisecond,
	}
}
func (tool *RemoteTool) Execute(ctx context.Context, arguments json.RawMessage) (any, error) {
	if tool.gateway == nil {
		return nil, fmt.Errorf("MCP gateway is unavailable")
	}
	if tool.server.Enabled != 1 || tool.tool.Enabled != 1 || tool.tool.Status != ToolGranted {
		return nil, fmt.Errorf("MCP tool %q is not granted", tool.tool.CanonicalName)
	}
	call := func(attemptCtx context.Context) (any, error) {
		return tool.gateway.CallTool(attemptCtx, tool.server.ID, tool.tool.RemoteName, arguments)
	}
	if tool.tool.Risk == string(permission.RiskRead) && tool.tool.Idempotent {
		return withMCPReadRetry(ctx, call)
	}
	return call(ctx)
}

func (tool *RemoteTool) RestoreCheckpoint(raw json.RawMessage) (any, error) {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	return value, nil
}

func (g *Gateway) ActiveRemoteTools(ctx context.Context) ([]agenttools.Tool, error) {
	rows, err := g.Store.ActiveTools(ctx)
	if err != nil {
		return nil, err
	}
	servers := map[int64]models.AgentMCPServer{}
	result := make([]agenttools.Tool, 0, len(rows))
	for _, row := range rows {
		server, ok := servers[row.ServerID]
		if !ok {
			server, err = g.Store.GetServer(ctx, row.ServerID)
			if err != nil {
				return nil, err
			}
			servers[row.ServerID] = server
		}
		if server.Enabled != 1 {
			continue
		}
		result = append(result, NewRemoteTool(g, server, row))
	}
	return result, nil
}

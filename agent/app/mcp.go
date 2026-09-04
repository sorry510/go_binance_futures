package app

import (
	"context"
	"fmt"
	"sync"

	"go_binance_futures/agent/contextengine"
	"go_binance_futures/agent/mcpclient"
	"go_binance_futures/agent/skill"
	agenttools "go_binance_futures/agent/tools"
)

var defaultMCPMu sync.Mutex
var defaultMCPGateway *mcpclient.Gateway
var defaultMCPTools *agenttools.Registry
var defaultMCPRegistered = map[string]bool{}

func initializeDefaultMCP(tools *agenttools.Registry) error {
	gateway := mcpclient.NewGateway(mcpclient.Store{})
	defaultMCPMu.Lock()
	defaultMCPGateway = gateway
	defaultMCPTools = tools
	defaultMCPMu.Unlock()
	return SyncDefaultMCPTools(context.Background())
}

func DefaultMCPGateway() (*mcpclient.Gateway, error) {
	if _, err := DefaultManager(); err != nil {
		return nil, err
	}
	defaultMCPMu.Lock()
	defer defaultMCPMu.Unlock()
	if defaultMCPGateway == nil {
		return nil, fmt.Errorf("MCP gateway is not initialized")
	}
	return defaultMCPGateway, nil
}
func SyncDefaultMCPTools(ctx context.Context) error {
	defaultMCPMu.Lock()
	gateway, registry := defaultMCPGateway, defaultMCPTools
	defaultMCPMu.Unlock()
	if gateway == nil || registry == nil {
		return fmt.Errorf("MCP runtime is not initialized")
	}
	remoteTools, err := gateway.ActiveRemoteTools(ctx)
	if err != nil {
		return err
	}
	active := make(map[string]bool, len(remoteTools))
	for _, tool := range remoteTools {
		if err := registry.Upsert(tool); err != nil {
			return err
		}
		active[tool.Name()] = true
	}
	defaultMCPMu.Lock()
	for name := range defaultMCPRegistered {
		if !active[name] {
			registry.Unregister(name)
		}
	}
	defaultMCPRegistered = active
	defaultMCPMu.Unlock()
	return nil
}

func MCPToolAllowlist(ctx context.Context, skillName string) ([]string, error) {
	gateway, err := DefaultMCPGateway()
	if err != nil {
		return nil, err
	}
	return gateway.Store.GrantedToolNames(ctx, skillName)
}

func MCPContextResources(ctx context.Context, skillName string, _ skill.Request) ([]contextengine.Resource, error) {
	gateway, err := DefaultMCPGateway()
	if err != nil {
		return nil, err
	}
	return gateway.ContextResourcesForSkill(ctx, skillName)
}

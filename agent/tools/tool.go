package tools

import (
	"context"
	"encoding/json"

	"go_binance_futures/agent/permission"
)

type Tool interface {
	Name() string
	Description() string
	Risk() permission.RiskLevel
	Execute(ctx context.Context, arguments json.RawMessage) (any, error)
}

type Func struct {
	ToolName        string
	ToolDescription string
	ToolRisk        permission.RiskLevel
	ExecuteFunc     func(context.Context, json.RawMessage) (any, error)
}

func (tool Func) Name() string               { return tool.ToolName }
func (tool Func) Description() string        { return tool.ToolDescription }
func (tool Func) Risk() permission.RiskLevel { return tool.ToolRisk }
func (tool Func) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	return tool.ExecuteFunc(ctx, args)
}

package tools

import (
	"context"
	"encoding/json"
	"time"

	"go_binance_futures/agent/permission"
)

type Metadata struct {
	InputSchema    json.RawMessage `json:"input_schema,omitempty"`
	OutputSchema   json.RawMessage `json:"output_schema,omitempty"`
	Timeout        time.Duration   `json:"-"`
	MaxResultBytes int             `json:"max_result_bytes,omitempty"`
	Idempotent     bool            `json:"idempotent"`
}

type Tool interface {
	Name() string
	Description() string
	Risk() permission.RiskLevel
	Metadata() Metadata
	Execute(ctx context.Context, arguments json.RawMessage) (any, error)
}

type Func struct {
	ToolName        string
	ToolDescription string
	ToolRisk        permission.RiskLevel
	ToolMetadata    Metadata
	ExecuteFunc     func(context.Context, json.RawMessage) (any, error)
}

func (tool Func) Name() string               { return tool.ToolName }
func (tool Func) Description() string        { return tool.ToolDescription }
func (tool Func) Risk() permission.RiskLevel { return tool.ToolRisk }
func (tool Func) Metadata() Metadata         { return tool.ToolMetadata }
func (tool Func) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	if tool.ExecuteFunc == nil {
		return nil, nil
	}
	return tool.ExecuteFunc(ctx, args)
}

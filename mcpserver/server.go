package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"go_binance_futures/scanner"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	ServerName           = "go-binance-futures-local-scanner"
	ServerVersion        = "0.1.0"
	Top30ToolName        = "futures_prefilter_top30"
	DefaultHTTPAddr      = ":3334"
	DefaultHTTPPath      = "/mcp"
	DefaultHealthzPath   = "/healthz"
	defaultToolMaxResult = 30
)

type Top30Input struct {
	Limit           int     `json:"limit,omitempty" jsonschema:"返回候选数量，默认 30，最大 30"`
	MinQuoteVolume  float64 `json:"min_quote_volume,omitempty" jsonschema:"24h 最小成交额 USDT，默认 10000000"`
	MaxDataAgeMs    int64   `json:"max_data_age_ms,omitempty" jsonschema:"允许的本地 symbols 行情最大延迟毫秒，0 表示不检查"`
	IncludeExcluded bool    `json:"include_excluded,omitempty" jsonschema:"是否返回完整剔除列表，默认 false；false 时只在 meta.excluded_count 返回数量"`
}

type HTTPOptions struct {
	JSONResponse bool
	Stateless    bool
}

var scanTop30 = scanner.ScanTop30

func NewServer() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    ServerName,
		Version: ServerVersion,
	}, nil)

	server.AddTool(&mcp.Tool{
		Name:         Top30ToolName,
		Description:  "查询本地合约表，按成交额、24h 涨幅、24h 中枢距离、长上影/冲高回落和本地最近价格动量筛选 Binance U 本位永续合约 Top30。",
		InputSchema:  mustSchema[Top30Input](),
		OutputSchema: mustSchema[scanner.PrefilterResult](),
	}, futuresPrefilterTop30)

	return server
}

func NewHTTPHandler(server *mcp.Server, opts HTTPOptions) http.Handler {
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{
		JSONResponse: opts.JSONResponse,
		Stateless:    opts.Stateless,
	})
}

func futuresPrefilterTop30(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input, err := decodeTop30Input(req)
	if err != nil {
		return toolError(err.Error()), nil
	}

	limit := input.Limit
	if limit == 0 {
		limit = defaultToolMaxResult
	}

	result, err := scanTop30(ctx, scanner.PrefilterOptions{
		Limit:          limit,
		MinQuoteVolume: input.MinQuoteVolume,
		MaxDataAgeMs:   input.MaxDataAgeMs,
	})
	if err != nil {
		return toolError(err.Error()), nil
	}
	if result.Meta == nil {
		result.Meta = make(map[string]any)
	}
	result.Meta["excluded_count"] = len(result.Excluded)
	if !input.IncludeExcluded {
		result.Excluded = nil
	}

	text, err := json.Marshal(result)
	if err != nil {
		return toolError(err.Error()), nil
	}
	return &mcp.CallToolResult{
		Content:           []mcp.Content{&mcp.TextContent{Text: string(text)}},
		StructuredContent: result,
	}, nil
}

func decodeTop30Input(req *mcp.CallToolRequest) (Top30Input, error) {
	var input Top30Input
	if req == nil || len(req.Params.Arguments) == 0 {
		return input, nil
	}
	dec := json.NewDecoder(bytes.NewReader(req.Params.Arguments))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&input); err != nil {
		return input, err
	}
	return input, nil
}

func toolError(message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: message}},
		IsError: true,
	}
}

func mustSchema[T any]() *jsonschema.Schema {
	schema, err := jsonschema.For[T](nil)
	if err != nil {
		panic(err)
	}
	return schema
}

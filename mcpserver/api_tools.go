package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/beego/beego/v2/core/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const maxAPIResponseBytes = 10 << 20

type APIToolInput struct {
	PathParams map[string]string `json:"path_params,omitempty" jsonschema:"路径参数，例如 id、flag、symbol"`
	Query      map[string]any    `json:"query,omitempty" jsonschema:"查询参数，与对应 HTTP 接口保持一致"`
	Body       map[string]any    `json:"body,omitempty" jsonschema:"JSON 请求体，与对应 HTTP 接口保持一致"`
}

type APIResult struct {
	StatusCode int `json:"status_code"`
	Body       any `json:"body"`
}

type APIToolDefinition struct {
	Name        string
	Category    string
	Description string
	Method      string
	Path        string
}

var apiHTTPClient = &http.Client{
	Timeout: 60 * time.Second,
	CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

var executeAPIRequest = executeInternalAPIRequest
var getInternalAPIBaseURL = defaultInternalAPIBaseURL

var apiToolDefinitions = []APIToolDefinition{
	{Name: "futures_symbols_list", Category: "合约交易对", Description: "查询合约交易对", Method: http.MethodGet, Path: "/features"},
	{Name: "futures_liquidation_orders_list", Category: "合约交易对", Description: "获取市场强平订单", Method: http.MethodGet, Path: "/futures/liquidation-orders"},

	{Name: "coin_notice_list", Category: "币种通知", Description: "查询币种通知", Method: http.MethodGet, Path: "/notice/coin"},
	{Name: "coin_notice_create", Category: "币种通知", Description: "新增币种通知", Method: http.MethodPost, Path: "/notice/coin"},
	{Name: "coin_notice_update", Category: "币种通知", Description: "更新币种通知", Method: http.MethodPut, Path: "/notice/coin/:id"},
	{Name: "coin_notice_delete", Category: "币种通知", Description: "删除币种通知", Method: http.MethodDelete, Path: "/notice/coin/:id"},
	{Name: "coin_notice_set_all_enable", Category: "币种通知", Description: "批量开启或关闭币种通知", Method: http.MethodPut, Path: "/notice/coin/enable/:flag"},

	{Name: "coin_listen_list", Category: "行情监听", Description: "查询行情监听配置", Method: http.MethodGet, Path: "/listen/coin"},
	{Name: "coin_listen_create", Category: "行情监听", Description: "新增行情监听配置", Method: http.MethodPost, Path: "/listen/coin"},
	{Name: "coin_listen_update", Category: "行情监听", Description: "更新行情监听配置", Method: http.MethodPut, Path: "/listen/coin/:id"},
	{Name: "coin_listen_delete", Category: "行情监听", Description: "删除行情监听配置", Method: http.MethodDelete, Path: "/listen/coin/:id"},
	{Name: "coin_listen_set_all_enable", Category: "行情监听", Description: "批量开启或关闭行情监听", Method: http.MethodPut, Path: "/listen/coin/enable/:flag"},
}

func registerAPITools(server *mcp.Server) {
	for _, definition := range apiToolDefinitions {
		definition := definition
		server.AddTool(&mcp.Tool{
			Name:         definition.Name,
			Description:  fmt.Sprintf("[%s] %s。对应 HTTP 接口：%s %s", definition.Category, definition.Description, definition.Method, definition.Path),
			InputSchema:  mustSchema[APIToolInput](),
			OutputSchema: mustSchema[APIResult](),
		}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return callAPITool(ctx, req, definition)
		})
	}
}

func callAPITool(ctx context.Context, req *mcp.CallToolRequest, definition APIToolDefinition) (*mcp.CallToolResult, error) {
	input, err := decodeInput[APIToolInput](req)
	if err != nil {
		return toolError(err.Error()), nil
	}

	result, err := executeAPIRequest(ctx, definition, input, authorizationFromContext(ctx))
	if err != nil {
		return toolError(err.Error()), nil
	}

	text, err := json.Marshal(result)
	if err != nil {
		return toolError(err.Error()), nil
	}
	return &mcp.CallToolResult{
		Content:           []mcp.Content{&mcp.TextContent{Text: string(text)}},
		StructuredContent: result,
		IsError:           result.StatusCode >= http.StatusBadRequest,
	}, nil
}

func executeInternalAPIRequest(ctx context.Context, definition APIToolDefinition, input APIToolInput, authorization string) (*APIResult, error) {
	if strings.TrimSpace(authorization) == "" {
		return nil, fmt.Errorf("authorization header is required")
	}

	path, err := resolveAPIPath(definition.Path, input.PathParams)
	if err != nil {
		return nil, err
	}
	endpoint, err := buildAPIURL(path, input.Query)
	if err != nil {
		return nil, err
	}

	var body io.Reader
	if input.Body != nil {
		payload, err := json.Marshal(input.Body)
		if err != nil {
			return nil, fmt.Errorf("encode request body: %w", err)
		}
		body = bytes.NewReader(payload)
	}

	request, err := http.NewRequestWithContext(ctx, definition.Method, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("create internal API request: %w", err)
	}
	request.Header.Set("Authorization", authorization)
	request.Header.Set("Accept", "application/json")
	if input.Body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := apiHTTPClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("call internal API: %w", err)
	}
	defer response.Body.Close()

	payload, err := io.ReadAll(io.LimitReader(response.Body, maxAPIResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("read internal API response: %w", err)
	}

	var responseBody any
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &responseBody); err != nil {
			responseBody = string(payload)
		}
	}
	return &APIResult{StatusCode: response.StatusCode, Body: responseBody}, nil
}

func resolveAPIPath(pathTemplate string, pathParams map[string]string) (string, error) {
	segments := strings.Split(pathTemplate, "/")
	for index, segment := range segments {
		if !strings.HasPrefix(segment, ":") {
			continue
		}
		name := strings.TrimPrefix(segment, ":")
		value := strings.TrimSpace(pathParams[name])
		if value == "" {
			return "", fmt.Errorf("path_params.%s is required", name)
		}
		segments[index] = url.PathEscape(value)
	}
	return strings.Join(segments, "/"), nil
}

func buildAPIURL(path string, query map[string]any) (string, error) {
	baseURL, err := getInternalAPIBaseURL()
	if err != nil {
		return "", err
	}
	endpoint, err := url.Parse(strings.TrimRight(baseURL, "/") + path)
	if err != nil {
		return "", fmt.Errorf("build internal API URL: %w", err)
	}
	values := endpoint.Query()
	for key, value := range query {
		addQueryValue(values, key, value)
	}
	endpoint.RawQuery = values.Encode()
	return endpoint.String(), nil
}

func defaultInternalAPIBaseURL() (string, error) {
	port, err := config.String("web::port")
	if err != nil || strings.TrimSpace(port) == "" {
		return "", fmt.Errorf("web::port is not configured")
	}
	return "http://127.0.0.1:" + strings.TrimPrefix(strings.TrimSpace(port), ":"), nil
}

func addQueryValue(values url.Values, key string, value any) {
	if value == nil {
		return
	}
	switch items := value.(type) {
	case []any:
		for _, item := range items {
			values.Add(key, fmt.Sprint(item))
		}
	case []string:
		for _, item := range items {
			values.Add(key, item)
		}
	default:
		values.Set(key, fmt.Sprint(value))
	}
}

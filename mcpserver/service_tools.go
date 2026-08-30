package mcpserver

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	liquidationservice "go_binance_futures/service/liquidation"
	symbolservice "go_binance_futures/service/symbol"
)

var listSymbolsForMCP = (symbolservice.Service{}).List
var listLiquidationsForMCP = (liquidationservice.Service{}).List

func executeAPIRequestDefault(ctx context.Context, definition APIToolDefinition, input APIToolInput, authorization string) (*APIResult, error) {
	if strings.TrimSpace(authorization) == "" {
		return nil, fmt.Errorf("authorization header is required")
	}
	switch definition.Name {
	case "futures_symbols_list":
		page := queryInt(input.Query, "page")
		limit := queryInt(input.Query, "limit")
		result, err := listSymbolsForMCP(ctx, symbolservice.ListOptions{
			Sort: queryString(input.Query, "sort"), SymbolType: queryString(input.Query, "symbol_type"),
			Symbol: queryString(input.Query, "symbol"), Enable: queryString(input.Query, "enable"),
			MarginType: queryString(input.Query, "margin_type"), Pin: queryString(input.Query, "pin"),
			Page: page, Limit: limit, DefaultLimit: 20,
		})
		if err != nil {
			return nil, err
		}
		return successAPIResult(map[string]any{"total": result.Total, "list": result.List}), nil
	case "futures_liquidation_orders_list":
		result, err := listLiquidationsForMCP(ctx, liquidationservice.ListOptions{
			Symbol: queryString(input.Query, "symbol"), Side: queryString(input.Query, "side"),
			StartTime:   liquidationservice.ParseTimestamp(queryString(input.Query, "start_time")),
			EndTime:     liquidationservice.ParseTimestamp(queryString(input.Query, "end_time")),
			MinNotional: queryFloat(input.Query, "min_notional"), Page: queryInt(input.Query, "page"),
			Limit: queryInt(input.Query, "limit"), DefaultLimit: 20, MaxLimit: 10000,
		})
		if err != nil {
			return nil, err
		}
		return successAPIResult(map[string]any{"total": result.Total, "list": result.List}), nil
	default:
		return executeInternalAPIRequest(ctx, definition, input, authorization)
	}
}

func successAPIResult(data any) *APIResult {
	return &APIResult{StatusCode: http.StatusOK, Body: map[string]any{"code": 200, "data": data, "msg": "success"}}
}

func queryString(values map[string]any, key string) string {
	if values == nil || values[key] == nil {
		return ""
	}
	switch value := values[key].(type) {
	case string:
		return strings.TrimSpace(value)
	default:
		return strings.TrimSpace(fmt.Sprint(value))
	}
}

func queryInt(values map[string]any, key string) int {
	if values == nil {
		return 0
	}
	switch value := values[key].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	case string:
		parsed, _ := strconv.Atoi(strings.TrimSpace(value))
		return parsed
	default:
		parsed, _ := strconv.Atoi(queryString(values, key))
		return parsed
	}
}

func queryFloat(values map[string]any, key string) float64 {
	if values == nil {
		return 0
	}
	switch value := values[key].(type) {
	case float64:
		return value
	case float32:
		return float64(value)
	case int:
		return float64(value)
	case int64:
		return float64(value)
	case string:
		parsed, _ := strconv.ParseFloat(strings.TrimSpace(value), 64)
		return parsed
	default:
		parsed, _ := strconv.ParseFloat(queryString(values, key), 64)
		return parsed
	}
}

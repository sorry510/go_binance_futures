package controllers

import (
	"context"
	"encoding/json"
	"fmt"

	strategyservice "go_binance_futures/service/strategy"
	symbolservice "go_binance_futures/service/symbol"
)

const (
	strategyTemplateAIFeaturesPageSize    = 20
	strategyTemplateAITestResultsPageSize = 100
)

type strategyTemplateAIFeaturesArgs struct {
	Sort       string `json:"sort"`
	SymbolType string `json:"symbol_type"`
	Symbol     string `json:"symbol"`
	Enable     string `json:"enable"`
	MarginType string `json:"margin_type"`
	Pin        string `json:"pin"`
	Page       int    `json:"page"`
	Limit      int    `json:"limit"`
}

type strategyTemplateAITestResultsArgs struct {
	Symbol       string `json:"symbol"`
	PositionSide string `json:"position_side"`
	StartTime    string `json:"start_time"`
	EndTime      string `json:"end_time"`
	Type         string `json:"type"`
	Page         int    `json:"page"`
	Limit        int    `json:"limit"`
}

func executeStrategyTemplateAITool(name string, arguments json.RawMessage) (string, error) {
	switch name {
	case "get_features":
		var args strategyTemplateAIFeaturesArgs
		if err := decodeStrategyTemplateAIToolArguments(arguments, &args); err != nil {
			return "", err
		}
		result, err := getStrategyTemplateAIFeatures(args)
		if err != nil {
			return "", err
		}
		return marshalStrategyTemplateAIToolResult(result)
	case "get_test_strategy_results":
		var args strategyTemplateAITestResultsArgs
		if err := decodeStrategyTemplateAIToolArguments(arguments, &args); err != nil {
			return "", err
		}
		result, err := getStrategyTemplateAITestResults(args)
		if err != nil {
			return "", err
		}
		return marshalStrategyTemplateAIToolResult(result)
	default:
		return "", fmt.Errorf("不支持的工具 %q", name)
	}
}

func decodeStrategyTemplateAIToolArguments(arguments json.RawMessage, target interface{}) error {
	if len(arguments) == 0 || string(arguments) == "null" {
		arguments = []byte("{}")
	}
	if err := decodeStrictStrategyTemplateJSON(arguments, target); err != nil {
		return fmt.Errorf("工具参数格式错误: %w", err)
	}
	return nil
}

func strategyTemplateAIToolPagination(page, limit, defaultLimit, maxLimit int) (int, int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	return page, limit, (page - 1) * limit
}

func getStrategyTemplateAIFeatures(args strategyTemplateAIFeaturesArgs) (map[string]interface{}, error) {
	result, err := (symbolservice.Service{}).List(context.Background(), symbolservice.ListOptions{
		Sort: args.Sort, SymbolType: args.SymbolType, Symbol: args.Symbol, Enable: args.Enable,
		MarginType: args.MarginType, Pin: args.Pin, Page: args.Page, Limit: args.Limit,
		DefaultLimit: strategyTemplateAIFeaturesPageSize, MaxLimit: strategyTemplateAIFeaturesPageSize,
	})
	if err != nil {
		return nil, fmt.Errorf("get_features 查询失败: %w", err)
	}
	return map[string]interface{}{"page": result.Page, "limit": result.Limit, "total": result.Total, "list": result.List}, nil
}

func getStrategyTemplateAITestResults(args strategyTemplateAITestResultsArgs) (map[string]interface{}, error) {
	result, err := (strategyservice.Service{}).ListTestResults(context.Background(), strategyservice.TestResultsOptions{
		Symbol: args.Symbol, PositionSide: args.PositionSide, StartTime: args.StartTime, EndTime: args.EndTime,
		Type: args.Type, Page: args.Page, Limit: args.Limit,
		DefaultLimit: strategyTemplateAITestResultsPageSize, MaxLimit: strategyTemplateAITestResultsPageSize,
	})
	if err != nil {
		return nil, fmt.Errorf("get_test_strategy_results 查询失败: %w", err)
	}
	return map[string]interface{}{"page": result.Page, "limit": result.Limit, "total": result.Total, "current_profit": result.CurrentProfit, "list": result.List}, nil
}

func marshalStrategyTemplateAIToolResult(result interface{}) (string, error) {
	data, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("工具结果编码失败: %w", err)
	}
	return string(data), nil
}

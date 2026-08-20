package controllers

import (
	"encoding/json"
	"fmt"
	"go_binance_futures/models"
	"strconv"
	"strings"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/config"
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
	page, limit, offset := strategyTemplateAIToolPagination(
		args.Page,
		args.Limit,
		strategyTemplateAIFeaturesPageSize,
		strategyTemplateAIFeaturesPageSize,
	)
	dbDriver, _ := config.String("database::driver")
	selectedFields := []string{
		"id", "symbol", "percentChange", "close", "open", "low", "high", "enable", "updateTime", "quoteVolume", "tradeCount", "leverage", "marginType",
		"stepSize", "tickSize", "usdt", "profit", "loss", "strategy_type", "pin",
	}

	o := orm.NewOrm()
	var symbols []models.Symbols
	var total int64
	sortExpr, orderDirection, hasCustomSort := featureSortExpr(strings.TrimSpace(args.Sort), dbDriver)
	if hasCustomSort {
		whereClauses := make([]string, 0, 5)
		bindData := make([]interface{}, 0, 7)
		if args.SymbolType != "" {
			whereClauses = append(whereClauses, featureSQLColumn(dbDriver, "type")+" = ?")
			bindData = append(bindData, args.SymbolType)
		}
		if args.Symbol != "" {
			whereClauses = append(whereClauses, featureSQLColumn(dbDriver, "symbol")+" LIKE ?")
			bindData = append(bindData, "%"+args.Symbol+"%")
		}
		if args.Enable != "" {
			whereClauses = append(whereClauses, featureSQLColumn(dbDriver, "enable")+" = ?")
			bindData = append(bindData, args.Enable)
		}
		if args.MarginType != "" {
			whereClauses = append(whereClauses, featureSQLColumn(dbDriver, "marginType")+" = ?")
			bindData = append(bindData, args.MarginType)
		}
		if args.Pin != "" {
			whereClauses = append(whereClauses, featureSQLColumn(dbDriver, "pin")+" = ?")
			bindData = append(bindData, 1)
		}
		whereSQL := ""
		if len(whereClauses) > 0 {
			whereSQL = " WHERE " + strings.Join(whereClauses, " AND ")
		}
		listSQL := "SELECT " + strings.Join(selectedFields, ", ") +
			" FROM " + featureSQLColumn(dbDriver, "symbols") + whereSQL +
			" ORDER BY " + sortExpr + " " + orderDirection + ", " + featureSQLColumn(dbDriver, "id") + " ASC LIMIT ? OFFSET ?"
		listBindData := append(append([]interface{}{}, bindData...), limit, offset)
		if _, err := o.Raw(listSQL, listBindData...).QueryRows(&symbols); err != nil {
			return nil, fmt.Errorf("get_features 查询列表失败: %w", err)
		}
		countSQL := "SELECT COUNT(*) FROM " + featureSQLColumn(dbDriver, "symbols") + whereSQL
		if err := o.Raw(countSQL, bindData...).QueryRow(&total); err != nil {
			return nil, fmt.Errorf("get_features 查询总数失败: %w", err)
		}
	} else {
		query := o.QueryTable("symbols")
		countQuery := o.QueryTable("symbols")
		if args.SymbolType != "" {
			query = query.Filter("type", args.SymbolType)
			countQuery = countQuery.Filter("type", args.SymbolType)
		}
		if args.Symbol != "" {
			query = query.Filter("symbol__contains", args.Symbol)
			countQuery = countQuery.Filter("symbol__contains", args.Symbol)
		}
		if args.Enable != "" {
			query = query.Filter("enable", args.Enable)
			countQuery = countQuery.Filter("enable", args.Enable)
		}
		if args.MarginType != "" {
			query = query.Filter("marginType", args.MarginType)
			countQuery = countQuery.Filter("marginType", args.MarginType)
		}
		if args.Pin != "" {
			query = query.Filter("pin", 1)
			countQuery = countQuery.Filter("pin", 1)
		}
		if _, err := query.OrderBy("ID").Limit(limit, offset).All(&symbols, selectedFields...); err != nil {
			return nil, fmt.Errorf("get_features 查询列表失败: %w", err)
		}
		var err error
		total, err = countQuery.Count()
		if err != nil {
			return nil, fmt.Errorf("get_features 查询总数失败: %w", err)
		}
	}

	return map[string]interface{}{
		"page":  page,
		"limit": limit,
		"total": total,
		"list":  symbols,
	}, nil
}

func getStrategyTemplateAITestResults(args strategyTemplateAITestResultsArgs) (map[string]interface{}, error) {
	page, limit, offset := strategyTemplateAIToolPagination(
		args.Page,
		args.Limit,
		strategyTemplateAITestResultsPageSize,
		strategyTemplateAITestResultsPageSize,
	)
	searchParams := testStrategyResultSearchParams{
		Symbol:       strings.TrimSpace(args.Symbol),
		PositionSide: strings.ToUpper(strings.TrimSpace(args.PositionSide)),
		StartTime:    strings.TrimSpace(args.StartTime),
		EndTime:      strings.TrimSpace(args.EndTime),
		Type:         strings.ToLower(strings.TrimSpace(args.Type)),
	}
	whereClause, queryArgs, _, err := searchParams.whereClause("t")
	if err != nil {
		return nil, fmt.Errorf("get_test_strategy_results 参数错误: %w", err)
	}

	listSQL := `SELECT t.id, t.symbol, t.price, t.leverage, t.usdt, t.profit, t.loss, t.position_amt, t.position_side, t.close_price, t.close_profit, t.createTime, t.updateTime, s.close as now_price FROM test_strategy_results t LEFT JOIN symbols s ON t.symbol = s.symbol where 1 = 1` + whereClause +
		" ORDER BY t.createTime DESC LIMIT " + strconv.Itoa(limit) + " OFFSET " + strconv.Itoa(offset)
	countSQL := `SELECT COUNT(*) FROM test_strategy_results t LEFT JOIN symbols s ON t.symbol = s.symbol where 1 = 1` + whereClause
	profitSQL := `SELECT t.price, t.leverage, t.position_amt, t.close_price, s.close as now_price FROM test_strategy_results t LEFT JOIN symbols s ON t.symbol = s.symbol where 1 = 1` + whereClause

	o := orm.NewOrm()
	var results []TestStrategyResultsTableList
	var profitResults []TestStrategyResultsTableList
	var total int64
	if _, err := o.Raw(listSQL, queryArgs...).QueryRows(&results); err != nil {
		return nil, fmt.Errorf("get_test_strategy_results 查询列表失败: %w", err)
	}
	for index := range results {
		calculateTestStrategyResultMetrics(&results[index])
	}
	if err := o.Raw(countSQL, queryArgs...).QueryRow(&total); err != nil {
		return nil, fmt.Errorf("get_test_strategy_results 查询总数失败: %w", err)
	}
	if _, err := o.Raw(profitSQL, queryArgs...).QueryRows(&profitResults); err != nil {
		return nil, fmt.Errorf("get_test_strategy_results 汇总收益失败: %w", err)
	}

	return map[string]interface{}{
		"page":           page,
		"limit":          limit,
		"total":          total,
		"current_profit": calculateTestStrategyResultsCurrentProfit(profitResults),
		"list":           results,
	}, nil
}

func marshalStrategyTemplateAIToolResult(result interface{}) (string, error) {
	data, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("工具结果编码失败: %w", err)
	}
	return string(data), nil
}

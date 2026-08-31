package strategy

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
)

type Service struct{}

type TestResult struct {
	models.TestStrategyResults
	NowPrice           string `orm:"column(now_price)" json:"now_price"`
	GrossProfit        string `json:"gross_profit"`
	OpenFee            string `json:"open_fee"`
	CloseFee           string `json:"close_fee"`
	TotalFee           string `json:"total_fee"`
	GrossProfitPercent string `json:"gross_profit_percent"`
	ProfitPercent      string `json:"profit_percent"`
}

type TestTradeProfit struct {
	GrossProfit        float64
	OpenFee            float64
	CloseFee           float64
	TotalFee           float64
	NetProfit          float64
	GrossProfitPercent float64
	NetProfitPercent   float64
}

type TestResultsOptions struct {
	Symbol       string
	PositionSide string
	StartTime    string
	EndTime      string
	Type         string
	Page         int
	Limit        int
	DefaultLimit int
	MaxLimit     int
}

type TestResultsResult struct {
	Page          int          `json:"page"`
	Limit         int          `json:"limit"`
	Total         int64        `json:"total"`
	CurrentProfit string       `json:"current_profit"`
	List          []TestResult `json:"list"`
}

type TemplateListOptions struct {
	Name         string
	Page         int
	Limit        int
	DefaultLimit int
	MaxLimit     int
}

type TemplateListResult struct {
	Page  int                        `json:"page"`
	Limit int                        `json:"limit"`
	Total int64                      `json:"total"`
	List  []models.StrategyTemplates `json:"list"`
}

type TemplateQuery struct {
	ID   int64
	Name string
}

func (Service) ListTestResults(ctx context.Context, opts TestResultsOptions) (TestResultsResult, error) {
	if err := ctx.Err(); err != nil {
		return TestResultsResult{}, err
	}
	page, limit, offset := normalizePagination(opts.Page, opts.Limit, opts.DefaultLimit, opts.MaxLimit)
	where, args, err := buildTestResultsWhere(opts, "t")
	if err != nil {
		return TestResultsResult{}, err
	}
	listSQL := `SELECT t.id, t.symbol, t.price, t.leverage, t.usdt, t.profit, t.loss, t.position_amt, t.position_side, t.close_price, t.close_profit, t.open_fee_rate, t.close_fee_rate, t.createTime, t.updateTime, s.close as now_price FROM test_strategy_results t LEFT JOIN symbols s ON t.symbol = s.symbol where 1 = 1` + where +
		" ORDER BY t.createTime DESC LIMIT " + strconv.Itoa(limit) + " OFFSET " + strconv.Itoa(offset)
	countSQL := `SELECT COUNT(*) FROM test_strategy_results t LEFT JOIN symbols s ON t.symbol = s.symbol where 1 = 1` + where
	profitSQL := `SELECT t.price, t.leverage, t.position_amt, t.close_price, t.open_fee_rate, t.close_fee_rate, s.close as now_price FROM test_strategy_results t LEFT JOIN symbols s ON t.symbol = s.symbol where 1 = 1` + where
	o := orm.NewOrm()
	var list []TestResult
	var profitResults []TestResult
	var total int64
	if _, err := o.Raw(listSQL, args...).QueryRows(&list); err != nil {
		return TestResultsResult{}, fmt.Errorf("list test strategy results: %w", err)
	}
	for index := range list {
		CalculateTestResultMetrics(&list[index])
	}
	if err := o.Raw(countSQL, args...).QueryRow(&total); err != nil {
		return TestResultsResult{}, fmt.Errorf("count test strategy results: %w", err)
	}
	if _, err := o.Raw(profitSQL, args...).QueryRows(&profitResults); err != nil {
		return TestResultsResult{}, fmt.Errorf("sum test strategy results: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return TestResultsResult{}, err
	}
	return TestResultsResult{Page: page, Limit: limit, Total: total, CurrentProfit: CalculateCurrentProfit(profitResults), List: list}, nil
}

func (Service) ListTemplates(ctx context.Context, opts TemplateListOptions) (TemplateListResult, error) {
	if err := ctx.Err(); err != nil {
		return TemplateListResult{}, err
	}
	page, limit, offset := normalizePagination(opts.Page, opts.Limit, opts.DefaultLimit, opts.MaxLimit)
	list, total, err := QueryTemplatePage(orm.NewOrm(), strings.TrimSpace(opts.Name), limit, offset)
	if err != nil {
		return TemplateListResult{}, err
	}
	return TemplateListResult{Page: page, Limit: limit, Total: total, List: list}, nil
}

func (Service) GetTemplate(ctx context.Context, query TemplateQuery) (models.StrategyTemplates, error) {
	if err := ctx.Err(); err != nil {
		return models.StrategyTemplates{}, err
	}
	qs := orm.NewOrm().QueryTable("strategy_templates")
	if query.ID > 0 {
		qs = qs.Filter("id", query.ID)
	} else if name := strings.TrimSpace(query.Name); name != "" {
		qs = qs.Filter("name", name)
	} else {
		return models.StrategyTemplates{}, fmt.Errorf("id or name is required")
	}
	var result models.StrategyTemplates
	if err := qs.One(&result); err != nil {
		return models.StrategyTemplates{}, fmt.Errorf("get strategy template: %w", err)
	}
	return result, nil
}

func QueryTemplatePage(o orm.Ormer, name string, limit, offset int) ([]models.StrategyTemplates, int64, error) {
	query := o.QueryTable("strategy_templates")
	if name != "" {
		query = query.Filter("Name__icontains", name)
	}
	total, err := query.Count()
	if err != nil {
		return nil, 0, err
	}
	var templates []models.StrategyTemplates
	_, err = query.OrderBy("-ID").Limit(limit, offset).All(&templates)
	return templates, total, err
}

func CalculateTestTradeProfit(entryPrice, exitPrice, positionAmt float64, leverage int64, openFeeRate, closeFeeRate float64) TestTradeProfit {
	quantity := math.Abs(positionAmt)
	if entryPrice <= 0 || exitPrice <= 0 || quantity == 0 {
		return TestTradeProfit{}
	}
	if openFeeRate < 0 {
		openFeeRate = 0
	}
	if closeFeeRate < 0 {
		closeFeeRate = 0
	}

	grossProfit := (exitPrice - entryPrice) * positionAmt
	openFee := entryPrice * quantity * openFeeRate
	closeFee := exitPrice * quantity * closeFeeRate
	totalFee := openFee + closeFee
	netProfit := grossProfit - totalFee
	positionValue := quantity * exitPrice
	grossProfitPercent := grossProfit / positionValue * float64(leverage) * 100
	netProfitPercent := netProfit / positionValue * float64(leverage) * 100

	return TestTradeProfit{
		GrossProfit: grossProfit, OpenFee: openFee, CloseFee: closeFee,
		TotalFee: totalFee, NetProfit: netProfit,
		GrossProfitPercent: grossProfitPercent, NetProfitPercent: netProfitPercent,
	}
}

func CalculateTestResultMetrics(result *TestResult) float64 {
	entryPrice, errEntry := strconv.ParseFloat(result.Price, 64)
	positionAmt, errAmount := strconv.ParseFloat(result.PositionAmt, 64)
	closePrice, _ := strconv.ParseFloat(result.ClosePrice, 64)
	if errEntry != nil || errAmount != nil {
		setZeroTestResultMetrics(result)
		return 0
	}
	effectivePrice := closePrice
	if effectivePrice <= 0 {
		currentPrice, err := strconv.ParseFloat(result.NowPrice, 64)
		if err != nil {
			setZeroTestResultMetrics(result)
			return 0
		}
		effectivePrice = currentPrice
	}
	metrics := CalculateTestTradeProfit(entryPrice, effectivePrice, positionAmt, result.Leverage, result.OpenFeeRate, result.CloseFeeRate)
	if effectivePrice <= 0 || positionAmt == 0 {
		setZeroTestResultMetrics(result)
		return 0
	}

	result.GrossProfit = strconv.FormatFloat(metrics.GrossProfit, 'f', 3, 64)
	result.OpenFee = strconv.FormatFloat(metrics.OpenFee, 'f', 3, 64)
	result.CloseFee = strconv.FormatFloat(metrics.CloseFee, 'f', 3, 64)
	result.TotalFee = strconv.FormatFloat(metrics.TotalFee, 'f', 3, 64)
	result.CloseProfit = strconv.FormatFloat(metrics.NetProfit, 'f', 3, 64)
	result.GrossProfitPercent = strconv.FormatFloat(metrics.GrossProfitPercent, 'f', 3, 64)
	result.ProfitPercent = strconv.FormatFloat(metrics.NetProfitPercent, 'f', 3, 64)
	formatted, _ := strconv.ParseFloat(result.CloseProfit, 64)
	return formatted
}

func setZeroTestResultMetrics(result *TestResult) {
	result.GrossProfit = "0.000"
	result.OpenFee = "0.000"
	result.CloseFee = "0.000"
	result.TotalFee = "0.000"
	result.CloseProfit = "0.000"
	result.GrossProfitPercent = "0.000"
	result.ProfitPercent = "0.000"
}

func CalculateCurrentProfit(results []TestResult) string {
	total := 0.0
	for index := range results {
		total += CalculateTestResultMetrics(&results[index])
	}
	return strconv.FormatFloat(total, 'f', 2, 64)
}

func buildTestResultsWhere(opts TestResultsOptions, alias string) (string, []interface{}, error) {
	prefix := ""
	if alias != "" {
		prefix = alias + "."
	}
	conditions := make([]string, 0, 5)
	args := make([]interface{}, 0, 4)
	symbol := strings.TrimSpace(opts.Symbol)
	if symbol != "" {
		if strings.ContainsAny(symbol, "%_") {
			return "", nil, fmt.Errorf("invalid symbol")
		}
		conditions = append(conditions, prefix+"symbol LIKE ?")
		args = append(args, "%"+symbol+"%")
	}
	positionSide := strings.ToUpper(strings.TrimSpace(opts.PositionSide))
	if positionSide != "" && positionSide != "ALL" {
		conditions = append(conditions, prefix+"position_side = ?")
		args = append(args, positionSide)
	}
	var startTime, endTime int64
	if value := strings.TrimSpace(opts.StartTime); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil || parsed <= 0 {
			return "", nil, fmt.Errorf("invalid start_time")
		}
		startTime = parsed
		conditions = append(conditions, prefix+"createTime >= ?")
		args = append(args, parsed)
	}
	if value := strings.TrimSpace(opts.EndTime); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil || parsed <= 0 {
			return "", nil, fmt.Errorf("invalid end_time")
		}
		endTime = parsed
		conditions = append(conditions, prefix+"createTime <= ?")
		args = append(args, parsed)
	}
	if startTime > 0 && endTime > 0 && startTime > endTime {
		return "", nil, fmt.Errorf("start_time must not exceed end_time")
	}
	switch strings.ToLower(strings.TrimSpace(opts.Type)) {
	case "open":
		conditions = append(conditions, prefix+"close_price = '0'")
	case "close":
		conditions = append(conditions, prefix+"close_price != '0'")
	}
	if len(conditions) == 0 {
		return "", args, nil
	}
	return " AND " + strings.Join(conditions, " AND "), args, nil
}

func normalizePagination(page, limit, defaultLimit, maxLimit int) (int, int, int) {
	if page < 1 {
		page = 1
	}
	if defaultLimit < 1 {
		defaultLimit = 20
	}
	if limit < 1 {
		limit = defaultLimit
	}
	if maxLimit > 0 && limit > maxLimit {
		limit = maxLimit
	}
	return page, limit, (page - 1) * limit
}

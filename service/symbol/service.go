package symbol

import (
	"context"
	"fmt"
	"strings"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/config"
)

type Service struct{}

type ListOptions struct {
	Sort         string
	SymbolType   string
	Symbol       string
	Enable       string
	MarginType   string
	Pin          string
	Page         int
	Limit        int
	DefaultLimit int
	MaxLimit     int
}

type ListResult struct {
	Page  int              `json:"page"`
	Limit int              `json:"limit"`
	Total int64            `json:"total"`
	List  []models.Symbols `json:"list"`
}

func (Service) List(ctx context.Context, opts ListOptions) (ListResult, error) {
	if err := ctx.Err(); err != nil {
		return ListResult{}, err
	}
	page, limit, offset := normalizePagination(opts.Page, opts.Limit, opts.DefaultLimit, opts.MaxLimit)
	driver, _ := config.String("database::driver")
	selectedFields := []string{
		"id", "symbol", "percentChange", "close", "open", "low", "high", "enable", "updateTime", "quoteVolume", "tradeCount", "leverage", "marginType",
		"stepSize", "tickSize", "usdt", "profit", "loss", "strategy_type", "pin",
	}

	o := orm.NewOrm()
	var list []models.Symbols
	var total int64
	sortExpression, direction, customSort := sortExpr(strings.TrimSpace(opts.Sort), driver)
	if customSort {
		where, args := symbolWhere(driver, opts)
		listSQL := "SELECT " + strings.Join(selectedFields, ", ") + " FROM " + sqlColumn(driver, "symbols") + where +
			" ORDER BY " + sortExpression + " " + direction + ", " + sqlColumn(driver, "id") + " ASC LIMIT ? OFFSET ?"
		listArgs := append(append([]interface{}{}, args...), limit, offset)
		if _, err := o.Raw(listSQL, listArgs...).QueryRows(&list); err != nil {
			return ListResult{}, fmt.Errorf("list symbols: %w", err)
		}
		countSQL := "SELECT COUNT(*) FROM " + sqlColumn(driver, "symbols") + where
		if err := o.Raw(countSQL, args...).QueryRow(&total); err != nil {
			return ListResult{}, fmt.Errorf("count symbols: %w", err)
		}
	} else {
		query := o.QueryTable("symbols")
		countQuery := o.QueryTable("symbols")
		query, countQuery = applyFilters(query, countQuery, opts)
		if _, err := query.OrderBy("ID").Limit(limit, offset).All(&list, selectedFields...); err != nil {
			return ListResult{}, fmt.Errorf("list symbols: %w", err)
		}
		var err error
		total, err = countQuery.Count()
		if err != nil {
			return ListResult{}, fmt.Errorf("count symbols: %w", err)
		}
	}
	if err := ctx.Err(); err != nil {
		return ListResult{}, err
	}
	return ListResult{Page: page, Limit: limit, Total: total, List: list}, nil
}

func (Service) Snapshot(ctx context.Context, symbol string) (models.Symbols, error) {
	if err := ctx.Err(); err != nil {
		return models.Symbols{}, err
	}
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" {
		return models.Symbols{}, fmt.Errorf("symbol is required")
	}
	var item models.Symbols
	if err := orm.NewOrm().QueryTable("symbols").Filter("symbol", symbol).One(&item); err != nil {
		return models.Symbols{}, fmt.Errorf("get symbol %s: %w", symbol, err)
	}
	return item, nil
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

func applyFilters(query, count orm.QuerySeter, opts ListOptions) (orm.QuerySeter, orm.QuerySeter) {
	if opts.SymbolType != "" {
		query, count = query.Filter("type", opts.SymbolType), count.Filter("type", opts.SymbolType)
	}
	if opts.Symbol != "" {
		query, count = query.Filter("symbol__contains", opts.Symbol), count.Filter("symbol__contains", opts.Symbol)
	}
	if opts.Enable != "" {
		query, count = query.Filter("enable", opts.Enable), count.Filter("enable", opts.Enable)
	}
	if opts.MarginType != "" {
		query, count = query.Filter("marginType", opts.MarginType), count.Filter("marginType", opts.MarginType)
	}
	if opts.Pin != "" {
		query, count = query.Filter("pin", 1), count.Filter("pin", 1)
	}
	return query, count
}

func symbolWhere(driver string, opts ListOptions) (string, []interface{}) {
	clauses := make([]string, 0, 5)
	args := make([]interface{}, 0, 5)
	add := func(field string, value interface{}) {
		clauses = append(clauses, sqlColumn(driver, field)+" = ?")
		args = append(args, value)
	}
	if opts.SymbolType != "" {
		add("type", opts.SymbolType)
	}
	if opts.Symbol != "" {
		clauses = append(clauses, sqlColumn(driver, "symbol")+" LIKE ?")
		args = append(args, "%"+opts.Symbol+"%")
	}
	if opts.Enable != "" {
		add("enable", opts.Enable)
	}
	if opts.MarginType != "" {
		add("marginType", opts.MarginType)
	}
	if opts.Pin != "" {
		add("pin", 1)
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func sortExpr(raw, driver string) (string, string, bool) {
	if raw == "" {
		return "", "", false
	}
	direction, field := "ASC", raw
	if strings.HasSuffix(raw, "+") {
		direction, field = "DESC", strings.TrimSuffix(raw, "+")
	} else if strings.HasSuffix(raw, "-") {
		field = strings.TrimSuffix(raw, "-")
	} else {
		return "", "", false
	}
	if field == "percent_change" {
		field = "percentChange"
	}
	if field == "quote_volume" {
		field = "quoteVolume"
	}
	switch field {
	case "percentChange", "close":
		castType := "DECIMAL(20,8)"
		if driver == "sqlite" {
			castType = "REAL"
		}
		return "CAST(" + sqlColumn(driver, field) + " AS " + castType + ")", direction, true
	case "quoteVolume", "symbol":
		return sqlColumn(driver, field), direction, true
	default:
		return "", "", false
	}
}

func sqlColumn(driver, name string) string {
	quote := "`"
	if driver == "postgres" {
		quote = "\""
	}
	return quote + name + quote
}

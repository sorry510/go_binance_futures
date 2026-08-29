package liquidation

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
)

type Service struct{}

type ListOptions struct {
	Symbol       string
	Side         string
	StartTime    int64
	EndTime      int64
	MinNotional  float64
	Page         int
	Limit        int
	DefaultLimit int
	MaxLimit     int
}

type ListResult struct {
	Page  int                              `json:"page"`
	Limit int                              `json:"limit"`
	Total int64                            `json:"total"`
	List  []models.FuturesLiquidationOrder `json:"list"`
}

func (Service) List(ctx context.Context, opts ListOptions) (ListResult, error) {
	if err := ctx.Err(); err != nil {
		return ListResult{}, err
	}
	page, limit := opts.Page, opts.Limit
	if page < 1 {
		page = 1
	}
	if opts.DefaultLimit < 1 {
		opts.DefaultLimit = 20
	}
	if limit < 1 {
		limit = opts.DefaultLimit
	}
	if opts.MaxLimit > 0 && limit > opts.MaxLimit {
		limit = opts.MaxLimit
	}
	offset := (page - 1) * limit

	query := orm.NewOrm().QueryTable(new(models.FuturesLiquidationOrder))
	if value := strings.ToUpper(strings.TrimSpace(opts.Symbol)); value != "" {
		query = query.Filter("symbol__icontains", value)
	}
	if value := strings.ToUpper(strings.TrimSpace(opts.Side)); value != "" && value != "ALL" {
		query = query.Filter("side", value)
	}
	if opts.StartTime > 0 {
		query = query.Filter("event_time__gte", opts.StartTime)
	}
	if opts.EndTime > 0 {
		query = query.Filter("event_time__lte", opts.EndTime)
	}
	if opts.StartTime > 0 && opts.EndTime > 0 && opts.StartTime > opts.EndTime {
		return ListResult{}, fmt.Errorf("start_time must not exceed end_time")
	}
	if opts.MinNotional > 0 {
		query = query.Filter("notional__gte", opts.MinNotional)
	}
	total, err := query.Count()
	if err != nil {
		return ListResult{}, fmt.Errorf("count liquidations: %w", err)
	}
	var list []models.FuturesLiquidationOrder
	if _, err := query.OrderBy("-event_time").Limit(limit, offset).All(&list); err != nil {
		return ListResult{}, fmt.Errorf("list liquidations: %w", err)
	}
	return ListResult{Page: page, Limit: limit, Total: total, List: list}, nil
}

func ParseTimestamp(value string) int64 {
	timestamp, _ := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if timestamp > 0 && timestamp < 1_000_000_000_000 {
		return timestamp * 1000
	}
	return timestamp
}

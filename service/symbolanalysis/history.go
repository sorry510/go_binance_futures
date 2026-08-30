package symbolanalysis

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"go_binance_futures/models"
	symbolservice "go_binance_futures/service/symbol"

	"github.com/beego/beego/v2/client/orm"
)

type HistoryService struct {
	Alias string
}

type HistorySaveRequest struct {
	TaskID        string
	Symbol        string
	Prompt        string
	Status        string
	Result        json.RawMessage
	Error         string
	Provider      string
	Model         string
	AnalysisPrice float64
	CreatedAt     int64
	CompletedAt   int64
}

type HistoryListOptions struct {
	Symbol string
	Status string
	Page   int
	Limit  int
}
type HistoryItem struct {
	models.SymbolAnalysisHistory
	Result         json.RawMessage `json:"result,omitempty"`
	CurrentPrice   float64         `json:"current_price,omitempty"`
	PriceChangePct float64         `json:"price_change_pct,omitempty"`
}

type HistoryListResult struct {
	Page  int           `json:"page"`
	Limit int           `json:"limit"`
	Total int64         `json:"total"`
	List  []HistoryItem `json:"list"`
}

type storedPlan struct {
	Direction       string  `json:"direction"`
	Confidence      float64 `json:"confidence"`
	MarketCondition *int    `json:"market_condition"`
	Summary         string  `json:"summary"`
}

func (service HistoryService) Save(ctx context.Context, req HistorySaveRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	req.TaskID = strings.TrimSpace(req.TaskID)
	req.Symbol = strings.ToUpper(strings.TrimSpace(req.Symbol))
	if req.TaskID == "" || req.Symbol == "" {
		return fmt.Errorf("history requires task_id and symbol")
	}
	item := models.SymbolAnalysisHistory{
		TaskID: req.TaskID, Symbol: req.Symbol, Prompt: req.Prompt,
		Status: req.Status, AnalysisPrice: req.AnalysisPrice, Error: req.Error,
		Provider: req.Provider, Model: req.Model, CreatedAt: req.CreatedAt,
		CompletedAt: req.CompletedAt, ResultJSON: string(req.Result),
	}
	if len(req.Result) > 0 {
		var plan storedPlan
		if err := json.Unmarshal(req.Result, &plan); err == nil {
			item.Direction = plan.Direction
			item.Confidence = plan.Confidence
			item.Summary = plan.Summary
			if plan.MarketCondition != nil {
				item.MarketCondition = *plan.MarketCondition
			}
		}
	}
	var o orm.Ormer
	if service.Alias != "" {
		o = orm.NewOrmUsingDB(service.Alias)
	} else {
		o = orm.NewOrm()
	}
	var existing models.SymbolAnalysisHistory
	err := o.QueryTable(new(models.SymbolAnalysisHistory)).Filter("task_id", req.TaskID).One(&existing)
	if err == orm.ErrNoRows {
		_, err = o.Insert(&item)
		return err
	}
	if err != nil {
		return fmt.Errorf("find symbol analysis history: %w", err)
	}
	item.ID = existing.ID
	_, err = o.Update(&item)
	return err
}

func (service HistoryService) List(ctx context.Context, opts HistoryListOptions) (HistoryListResult, error) {
	if err := ctx.Err(); err != nil {
		return HistoryListResult{}, err
	}
	page, limit := opts.Page, opts.Limit
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	var o orm.Ormer
	if service.Alias != "" {
		o = orm.NewOrmUsingDB(service.Alias)
	} else {
		o = orm.NewOrm()
	}
	query := o.QueryTable(new(models.SymbolAnalysisHistory))
	if symbol := strings.ToUpper(strings.TrimSpace(opts.Symbol)); symbol != "" {
		query = query.Filter("symbol", symbol)
	}
	if status := strings.TrimSpace(opts.Status); status != "" {
		query = query.Filter("status", status)
	}
	total, err := query.Count()
	if err != nil {
		return HistoryListResult{}, fmt.Errorf("count symbol analysis history: %w", err)
	}
	var rows []models.SymbolAnalysisHistory
	if _, err := query.OrderBy("-created_at").Limit(limit, (page-1)*limit).All(&rows); err != nil {
		return HistoryListResult{}, fmt.Errorf("list symbol analysis history: %w", err)
	}
	currentPrice := 0.0
	if service.Alias == "" {
		currentPrice = currentSymbolPrice(ctx, opts.Symbol)
	}
	items := make([]HistoryItem, 0, len(rows))
	for _, row := range rows {
		item := HistoryItem{SymbolAnalysisHistory: row, CurrentPrice: currentPrice}
		if row.ResultJSON != "" {
			item.Result = json.RawMessage(row.ResultJSON)
		}
		if currentPrice > 0 && row.AnalysisPrice > 0 {
			item.PriceChangePct = (currentPrice - row.AnalysisPrice) / row.AnalysisPrice * 100
		}
		items = append(items, item)
	}
	return HistoryListResult{Page: page, Limit: limit, Total: total, List: items}, nil
}

func currentSymbolPrice(ctx context.Context, symbol string) float64 {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" {
		return 0
	}
	item, err := (symbolservice.Service{}).Snapshot(ctx, symbol)
	if err != nil {
		return 0
	}
	value, _ := strconv.ParseFloat(item.Close, 64)
	return value
}

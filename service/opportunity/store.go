package opportunity

import (
	"context"
	"fmt"
	"strings"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
)

type Store struct{ Alias string }

func (s Store) orm() orm.Ormer {
	if strings.TrimSpace(s.Alias) != "" {
		return orm.NewOrmUsingDB(s.Alias)
	}
	return orm.NewOrm()
}

func (s Store) Insert(ctx context.Context, row *models.AgentOpportunity) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if row == nil || strings.TrimSpace(row.OpportunityID) == "" {
		return fmt.Errorf("opportunity is required")
	}
	_, err := s.orm().Insert(row)
	return err
}

func (s Store) Save(ctx context.Context, row *models.AgentOpportunity) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if row == nil || row.ID <= 0 || strings.TrimSpace(row.OpportunityID) == "" {
		return fmt.Errorf("persisted opportunity is required")
	}
	_, err := s.orm().Update(row)
	return err
}

func (s Store) Get(ctx context.Context, opportunityID string) (models.AgentOpportunity, error) {
	if err := ctx.Err(); err != nil {
		return models.AgentOpportunity{}, err
	}
	var row models.AgentOpportunity
	err := s.orm().QueryTable(new(models.AgentOpportunity)).
		Filter("opportunity_id", strings.TrimSpace(opportunityID)).One(&row)
	return row, err
}

func (s Store) FindBySource(ctx context.Context, sourceType, sourceID string) (models.AgentOpportunity, error) {
	if err := ctx.Err(); err != nil {
		return models.AgentOpportunity{}, err
	}
	var row models.AgentOpportunity
	err := s.orm().QueryTable(new(models.AgentOpportunity)).
		Filter("source_type", strings.TrimSpace(sourceType)).
		Filter("source_id", strings.TrimSpace(sourceID)).One(&row)
	return row, err
}

func (s Store) FindRecent(ctx context.Context, symbol, sourceType string, since int64) (models.AgentOpportunity, error) {
	if err := ctx.Err(); err != nil {
		return models.AgentOpportunity{}, err
	}
	var row models.AgentOpportunity
	err := s.orm().QueryTable(new(models.AgentOpportunity)).
		Filter("symbol", strings.ToUpper(strings.TrimSpace(symbol))).
		Filter("source_type", strings.TrimSpace(sourceType)).
		Filter("created_at__gte", since).
		OrderBy("-created_at", "-id").One(&row)
	return row, err
}

func (s Store) List(ctx context.Context, opt ListOptions) (ListResult, error) {
	if err := ctx.Err(); err != nil {
		return ListResult{}, err
	}
	if opt.Page < 1 {
		opt.Page = 1
	}
	if opt.Limit < 1 {
		opt.Limit = 20
	}
	if opt.Limit > 100 {
		opt.Limit = 100
	}
	q := s.orm().QueryTable(new(models.AgentOpportunity))
	if v := strings.TrimSpace(opt.Status); v != "" {
		q = q.Filter("status", v)
	}
	if v := strings.TrimSpace(opt.AnalysisStatus); v != "" {
		q = q.Filter("analysis_status", v)
	}
	if v := strings.ToUpper(strings.TrimSpace(opt.Symbol)); v != "" {
		q = q.Filter("symbol", v)
	}
	if v := strings.TrimSpace(opt.SourceType); v != "" {
		q = q.Filter("source_type", v)
	}
	total, err := q.Count()
	if err != nil {
		return ListResult{}, err
	}
	rows := []models.AgentOpportunity{}
	if _, err := q.OrderBy("-created_at", "-id").Limit(opt.Limit, (opt.Page-1)*opt.Limit).All(&rows); err != nil {
		return ListResult{}, err
	}
	return ListResult{Page: opt.Page, Limit: opt.Limit, Total: total, List: rows}, nil
}

func (s Store) Expire(ctx context.Context, now int64) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return s.orm().QueryTable(new(models.AgentOpportunity)).
		Filter("expires_at__gt", 0).
		Filter("expires_at__lte", now).
		Filter("status__in", StatusNew, StatusReviewed).
		Update(orm.Params{"status": StatusExpired, "updated_at": now})
}

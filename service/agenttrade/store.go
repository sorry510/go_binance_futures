package agenttrade

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go_binance_futures/agent/security"
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

func (s Store) SaveProposal(ctx context.Context, row *models.AgentTradeProposal) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if row == nil || strings.TrimSpace(row.ProposalID) == "" {
		return fmt.Errorf("proposal is required")
	}
	o := s.orm()
	var existing models.AgentTradeProposal
	err := o.QueryTable(new(models.AgentTradeProposal)).Filter("proposal_id", row.ProposalID).One(&existing)
	if err == orm.ErrNoRows {
		_, err = o.Insert(row)
		return err
	}
	if err != nil {
		return err
	}
	row.ID = existing.ID
	_, err = o.Update(row)
	return err
}

func (s Store) GetProposal(ctx context.Context, proposalID string) (models.AgentTradeProposal, error) {
	if err := ctx.Err(); err != nil {
		return models.AgentTradeProposal{}, err
	}
	var row models.AgentTradeProposal
	err := s.orm().QueryTable(new(models.AgentTradeProposal)).Filter("proposal_id", strings.TrimSpace(proposalID)).One(&row)
	return row, err
}

func (s Store) FindBySourceTask(ctx context.Context, taskID string) (models.AgentTradeProposal, error) {
	if err := ctx.Err(); err != nil {
		return models.AgentTradeProposal{}, err
	}
	var row models.AgentTradeProposal
	err := s.orm().QueryTable(new(models.AgentTradeProposal)).Filter("source_task_id", strings.TrimSpace(taskID)).OrderBy("-created_at").One(&row)
	return row, err
}

func (s Store) List(ctx context.Context, opt ListOptions) (ListResult, error) {
	if err := ctx.Err(); err != nil {
		return ListResult{}, err
	}
	page, limit := opt.Page, opt.Limit
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	q := s.orm().QueryTable(new(models.AgentTradeProposal))
	if v := strings.TrimSpace(opt.Status); v != "" {
		q = q.Filter("status", v)
	}
	if v := strings.ToUpper(strings.TrimSpace(opt.Symbol)); v != "" {
		q = q.Filter("symbol", v)
	}
	total, err := q.Count()
	if err != nil {
		return ListResult{}, err
	}
	rows := []models.AgentTradeProposal{}
	if _, err := q.OrderBy("-created_at").Limit(limit, (page-1)*limit).All(&rows); err != nil {
		return ListResult{}, err
	}
	return ListResult{Page: page, Limit: limit, Total: total, List: rows}, nil
}

func (s Store) SaveExecution(ctx context.Context, row *models.AgentTradeExecution) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if row == nil || strings.TrimSpace(row.ProposalID) == "" {
		return fmt.Errorf("execution is required")
	}
	o := s.orm()
	var existing models.AgentTradeExecution
	err := o.QueryTable(new(models.AgentTradeExecution)).Filter("proposal_id", row.ProposalID).One(&existing)
	if err == orm.ErrNoRows {
		_, err = o.Insert(row)
		return err
	}
	if err != nil {
		return err
	}
	row.ID = existing.ID
	_, err = o.Update(row)
	return err
}

func (s Store) GetExecution(ctx context.Context, proposalID string) (models.AgentTradeExecution, error) {
	if err := ctx.Err(); err != nil {
		return models.AgentTradeExecution{}, err
	}
	var row models.AgentTradeExecution
	err := s.orm().QueryTable(new(models.AgentTradeExecution)).Filter("proposal_id", strings.TrimSpace(proposalID)).One(&row)
	return row, err
}

func (s Store) Audit(ctx context.Context, proposalID, event, status, actor string, detail any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	raw := ""
	if detail != nil {
		encoded, err := json.Marshal(detail)
		if err != nil {
			return err
		}
		raw = security.RedactPayload(string(encoded))
	}
	row := models.AgentTradeAudit{
		ProposalID: strings.TrimSpace(proposalID), Event: strings.TrimSpace(event),
		Status: strings.TrimSpace(status), Actor: strings.TrimSpace(actor), DetailJSON: raw,
		CreatedAt: time.Now().UTC().UnixMilli(),
	}
	_, err := s.orm().Insert(&row)
	return err
}

func (s Store) Audits(ctx context.Context, proposalID string) ([]models.AgentTradeAudit, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rows := []models.AgentTradeAudit{}
	_, err := s.orm().QueryTable(new(models.AgentTradeAudit)).Filter("proposal_id", strings.TrimSpace(proposalID)).OrderBy("created_at", "id").All(&rows)
	return rows, err
}

func (s Store) HasRecentExecution(ctx context.Context, symbol string, since int64) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	count, err := s.orm().QueryTable(new(models.AgentTradeProposal)).
		Filter("symbol", strings.ToUpper(strings.TrimSpace(symbol))).
		Filter("status", StatusExecuted).Filter("executed_at__gte", since).Count()
	return count > 0, err
}

func (s Store) Expire(ctx context.Context, now int64) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return s.orm().QueryTable(new(models.AgentTradeProposal)).
		Filter("expires_at__lte", now).
		Filter("status__in", StatusRiskRejected, StatusAwaitingApproval, StatusApproved).
		Update(orm.Params{"status": StatusExpired, "updated_at": now})
}

func (s Store) CompareAndSetProposalStatus(ctx context.Context, proposalID, from, to string, updatedAt int64) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	count, err := s.orm().QueryTable(new(models.AgentTradeProposal)).
		Filter("proposal_id", strings.TrimSpace(proposalID)).
		Filter("status", strings.TrimSpace(from)).
		Update(orm.Params{"status": strings.TrimSpace(to), "updated_at": updatedAt})
	if err != nil {
		return false, err
	}
	return count == 1, nil
}

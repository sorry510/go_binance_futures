package orderhistory

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"go_binance_futures/models"
)

// Bulk edits are limited to presentation-only status/completeness overrides.
// Physical deletion is implemented separately with terminal-trade guards.
type BulkRequest struct {
	Filters              Filter `json:"filters"`
	Action               string `json:"action"` // update only
	StatusOverride       string `json:"status_override"`
	CompletenessOverride string `json:"completeness_override"`
	ClearStatus          bool   `json:"clear_status"`
	ClearCompleteness    bool   `json:"clear_completeness"`
	ExpectedDigest       string `json:"expected_digest"`
	Reason               string `json:"reason"`
}
type BulkPreview struct {
	Count       int    `json:"count"`
	ActiveCount int    `json:"active_count"`
	Digest      string `json:"digest"`
	Limit       int    `json:"limit"`
}
type BulkResult struct {
	Updated   int `json:"updated"`
	AuditRows int `json:"audit_rows"`
}

const maxHistoryBulk = 5000

var ErrBulkConflict = errors.New("history batch changed since preview")
var ErrBulkTooLarge = errors.New("history batch exceeds 5000 positions")
var ErrBulkActive = errors.New("cannot hide active managed positions")

func (req BulkRequest) normalized() (BulkRequest, error) {
	q := req.Filters
	q.Page = 1
	q.Limit = 100
	validated, err := q.validated()
	if err != nil {
		return req, err
	}
	req.Filters = validated
	statusOk := map[string]bool{"active": true, "suspended": true, "closed": true, "reconcile_required": true}
	completenessOk := map[string]bool{"pending": true, "incomplete": true, "manual_reviewed": true}
	if (req.StatusOverride != "" && !statusOk[req.StatusOverride]) ||
		(req.CompletenessOverride != "" && !completenessOk[req.CompletenessOverride]) ||
		(req.ClearStatus && req.StatusOverride != "") ||
		(req.ClearCompleteness && req.CompletenessOverride != "") ||
		(req.StatusOverride == "" && req.CompletenessOverride == "" && !req.ClearStatus && !req.ClearCompleteness) {
		return req, fmt.Errorf("%w: invalid view-only overrides", ErrInvalidQuery)
	}

	return req, nil
}

type historyQueryTable interface {
	QueryTable(interface{}) orm.QuerySeter
}

func batchScope(q Filter, o historyQueryTable) orm.QuerySeter {
	qs := o.QueryTable(new(models.FuturesManagedPosition)).Filter("account_id", q.AccountID)
	if q.Owner != "" {
		qs = qs.Filter("owner", q.Owner)
	}
	if q.Symbol != "" {
		qs = qs.Filter("symbol__icontains", q.Symbol)
	}
	if q.PositionSide != "" {
		qs = qs.Filter("position_side", q.PositionSide)
	}
	// Filter by the real system state, not the operator's presentation override.
	if q.Status != "" {
		qs = qs.Filter("status", q.Status)
	}
	if q.Settlement != "" {
		qs = qs.Filter("settlement_status", q.Settlement)
	}
	if q.StartTime > 0 {
		qs = qs.Filter("created_at__gte", q.StartTime)
	}
	if q.EndTime > 0 {
		qs = qs.Filter("created_at__lte", q.EndTime)
	}
	return qs
}
func batchCandidates(ctx context.Context, o historyQueryTable, req BulkRequest) ([]models.FuturesManagedPosition, BulkPreview, error) {
	if err := ctx.Err(); err != nil {
		return nil, BulkPreview{}, err
	}
	var positions []models.FuturesManagedPosition
	// At most max+1; never accidentally modify a million rows in one action.
	scope := batchScope(req.Filters, o)
	_, err := scope.OrderBy("id").Limit(maxHistoryBulk + 1).All(&positions)
	if err != nil {
		return nil, BulkPreview{}, err
	}
	if len(positions) > maxHistoryBulk {
		return nil, BulkPreview{}, ErrBulkTooLarge
	}
	preview := BulkPreview{Count: len(positions), Limit: maxHistoryBulk}
	if len(positions) == 0 {
		return positions, preview, nil
	}
	for _, p := range positions {
		if p.Status != "closed" {
			preview.ActiveCount++
		}
	}
	// Review digest binds action, all filters, overrides, IDs, state and update
	// versions. Confirming after a new fill/reconciliation/mutation fails 409.
	canonical := struct {
		Action            string
		Filter            Filter
		Status            string
		Completeness      string
		ClearStatus       bool
		ClearCompleteness bool
		Rows              []models.FuturesManagedPosition
	}{req.Action, req.Filters, req.StatusOverride, req.CompletenessOverride, req.ClearStatus, req.ClearCompleteness, positions}
	raw, err := json.Marshal(canonical)
	if err != nil {
		return nil, BulkPreview{}, err
	}
	hash := sha256.Sum256(raw)
	preview.Digest = fmt.Sprintf("%x", hash[:])
	return positions, preview, nil
}
func (s Store) Preview(ctx context.Context, req BulkRequest) (BulkPreview, error) {
	req, err := req.normalized()
	if err != nil {
		return BulkPreview{}, err
	}
	_, res, err := batchCandidates(ctx, s.db(), req)
	return res, err
}
func (s Store) Apply(ctx context.Context, req BulkRequest, operator string) (BulkResult, error) {
	req, err := req.normalized()
	if err != nil {
		return BulkResult{}, err
	}
	if len(req.ExpectedDigest) != 64 || len(strings.TrimSpace(req.Reason)) < 8 ||
		len(req.Reason) > 512 || len(operator) == 0 || len(operator) > 128 {
		return BulkResult{}, fmt.Errorf("%w: confirmation token, reason or operator", ErrInvalidQuery)
	}
	tx, err := s.db().BeginWithCtx(ctx)
	if err != nil {
		return BulkResult{}, err
	}
	defer tx.Rollback()
	positions, preview, err := batchCandidates(ctx, tx, req)
	if err != nil {
		return BulkResult{}, err
	}
	if preview.Count == 0 || preview.Digest != req.ExpectedDigest {
		return BulkResult{}, ErrBulkConflict
	}
	// A transaction covers both state and audit; no partial mutation on failure.
	stamped := time.Now().UnixNano()
	now := time.Now().UnixMilli()
	changed := 0
	for _, p := range positions {
		if err := ctx.Err(); err != nil {
			return BulkResult{}, err
		}
		status := p.HistoryStatusOverride
		completeness := p.HistoryCompletenessOverride
		if req.ClearStatus {
			status = ""
		} else if req.StatusOverride != "" {
			status = req.StatusOverride
		}
		if req.ClearCompleteness {
			completeness = ""
		} else if req.CompletenessOverride != "" {
			completeness = req.CompletenessOverride
		}
		if status == p.HistoryStatusOverride && completeness == p.HistoryCompletenessOverride {
			continue
		}
		result, err := tx.Raw(`UPDATE futures_managed_positions
    SET history_status_override=?,history_completeness_override=?,
        history_modified_at=?
    WHERE id=? AND account_id=? AND history_modified_at=?
      AND updated_at=? AND status=?`,
			status, completeness, stamped,
			p.ID, p.AccountID, p.HistoryModifiedAt, p.UpdatedAt, p.Status).Exec()
		if err != nil {
			return BulkResult{}, err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return BulkResult{}, err
		}
		if affected != 1 {
			return BulkResult{}, ErrBulkConflict
		}
		event := &models.OrderHistoryAdminAudit{
			AccountID: p.AccountID, PositionID: p.ID, Action: req.Action,
			OldStatus: p.HistoryStatusOverride, NewStatus: status,
			OldCompleteness: p.HistoryCompletenessOverride, NewCompleteness: completeness,
			Operator: operator, Reason: strings.TrimSpace(req.Reason),
			BatchDigest: preview.Digest, CreatedAt: now,
		}
		if _, err = tx.Insert(event); err != nil {
			return BulkResult{}, err
		}
		changed++
	}
	if err = tx.Commit(); err != nil {
		return BulkResult{}, err
	}
	return BulkResult{Updated: changed, AuditRows: changed}, nil
}

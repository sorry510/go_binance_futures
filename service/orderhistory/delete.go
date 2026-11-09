package orderhistory

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"go_binance_futures/models"
)

// PhysicalDeleteRequest deletes a complete, closed, terminal position cycle
// with its historical child records. It is scoped to the same filters as the
// history page, not just the visible pagination window.
type PhysicalDeleteRequest struct {
	Filters       Filter `json:"filters"`
	ExpectedCount int    `json:"expected_count"`
}
type PhysicalDeleteResult struct {
	Positions int   `json:"positions"`
	Orders    int64 `json:"orders"`
	Fills     int64 `json:"fills"`
}

var ErrDeleteNotTerminal = errors.New("only fully closed, terminal position cycles can be deleted")
var ErrDeleteCooldown = errors.New("recent closing trades are still used by trading cooldown")
var ErrDeleteChanged = errors.New("filtered history changed; refresh and confirm again")

// 30 days is deliberately longer than normal recent-close cooldown windows.
// Re-check both close time and authentic Fill trade_time: async REST backfills
// may arrive after a position was originally closed.
const deleteCooldownRetention = 30 * 24 * time.Hour

func (s Store) DeleteMatching(ctx context.Context, request PhysicalDeleteRequest) (PhysicalDeleteResult, error) {
	if request.Filters.AccountID != "main" && request.Filters.AccountID != "lead" {
		return PhysicalDeleteResult{}, fmt.Errorf("%w: explicit account required for deletion", ErrInvalidQuery)
	}
	q := request.Filters
	q.Page = 1
	q.Limit = 100
	q, err := q.validated()
	if err != nil {
		return PhysicalDeleteResult{}, err
	}
	if request.ExpectedCount <= 0 || request.ExpectedCount > maxHistoryBulk {
		return PhysicalDeleteResult{}, fmt.Errorf("%w: expected_count", ErrInvalidQuery)
	}
	if err = ctx.Err(); err != nil {
		return PhysicalDeleteResult{}, err
	}
	tx, err := s.db().BeginWithCtx(ctx)
	if err != nil {
		return PhysicalDeleteResult{}, err
	}
	defer tx.Rollback()

	// In MySQL/InnoDB this locks the selected position rows until the entire
	// cycle and its child records have been removed or the tx is rolled back.
	var positions []models.FuturesManagedPosition
	scope := batchScope(q, tx).OrderBy("id").Limit(maxHistoryBulk + 1)
	if !s.DisableRowLockForSQLiteTests {
		scope = scope.ForUpdate()
	}
	_, err = scope.All(&positions)
	if err != nil {
		return PhysicalDeleteResult{}, err
	}
	if len(positions) != request.ExpectedCount {
		return PhysicalDeleteResult{}, ErrDeleteChanged
	}

	cutoff := s.now().Add(-deleteCooldownRetention).UnixMilli()
	// One blocked cycle aborts the entire batch; never partially delete.
	for _, p := range positions {
		if p.Status != "closed" || p.ClosedAt <= 0 || p.ClosedAt > cutoff ||
			math.Abs(p.ManagedQty) > 1e-9 ||
			(p.SettlementStatus != "settled" && p.SettlementStatus != "legacy") {
			return PhysicalDeleteResult{}, ErrDeleteNotTerminal
		}
		var pending int
		if err = tx.Raw(`SELECT COUNT(*) FROM futures_managed_orders
   WHERE position_id=? AND account_id=?
   AND (status IS NULL OR status NOT IN ('filled','canceled','failed'))`, p.ID, p.AccountID).QueryRow(&pending); err != nil {
			return PhysicalDeleteResult{}, err
		}
		if pending > 0 {
			return PhysicalDeleteResult{}, ErrDeleteNotTerminal
		}
		var recent int
		if err = tx.Raw(`SELECT COUNT(*) FROM futures_trade_fills
   WHERE account_id=? AND trade_time>?
    AND (position_id=? OR (position_id IS NULL AND client_order_id IN
     (SELECT client_order_id FROM futures_managed_orders
      WHERE account_id=? AND position_id=?)))`,
			p.AccountID, cutoff, p.ID, p.AccountID, p.ID).QueryRow(&recent); err != nil {
			return PhysicalDeleteResult{}, err
		}
		if recent > 0 {
			return PhysicalDeleteResult{}, ErrDeleteCooldown
		}
		// A damaged Fill pointing to another position while using a
		// client-order ID belonging to this cycle must block deletion.
		var foreignLinked int
		if err = tx.Raw(`SELECT COUNT(*) FROM futures_trade_fills f
          JOIN futures_managed_orders m
            ON m.client_order_id=f.client_order_id AND m.account_id=f.account_id
          WHERE m.position_id=? AND m.account_id=?
            AND f.position_id IS NOT NULL AND f.position_id<>?`,
			p.ID, p.AccountID, p.ID).QueryRow(&foreignLinked); err != nil {
			return PhysicalDeleteResult{}, err
		}
		if foreignLinked > 0 {
			return PhysicalDeleteResult{}, ErrDeleteNotTerminal
		}
	}
	result := PhysicalDeleteResult{}
	for _, p := range positions {
		if err = ctx.Err(); err != nil {
			return PhysicalDeleteResult{}, err
		}
		// Each child delete is account + position scoped; never use just symbol or
		// client order ID. Do not delete global exchange order mirror records.
		for _, table := range []string{
			"futures_trade_fills", "futures_managed_orders",
			"futures_ownership_adjustments", "order_history_admin_audits",
		} {
			stmt := "DELETE FROM " + table + " WHERE position_id=? AND account_id=?"
			args := []interface{}{p.ID, p.AccountID}
			if table == "futures_trade_fills" {
				// Include fills received before position_id was assigned,
				// but ONLY when their exact client_order_id belongs to
				// a child Managed order of this same account/cycle.
				stmt = `DELETE FROM futures_trade_fills WHERE account_id=? AND
                    (position_id=? OR (position_id IS NULL AND client_order_id IN
                      (SELECT client_order_id FROM futures_managed_orders
                       WHERE account_id=? AND position_id=?)))`
				args = []interface{}{p.AccountID, p.ID, p.AccountID, p.ID}
			}
			deletion, execErr := tx.Raw(stmt, args...).Exec()
			if execErr != nil {
				return PhysicalDeleteResult{}, fmt.Errorf("delete %s: %w", table, execErr)
			}
			n, execErr := deletion.RowsAffected()
			if execErr != nil {
				return PhysicalDeleteResult{}, execErr
			}
			switch table {
			case "futures_trade_fills":
				result.Fills += n
			case "futures_managed_orders":
				result.Orders += n
			}
		}
		deleted, execErr := tx.Raw(`DELETE FROM futures_managed_positions
   WHERE id=? AND account_id=? AND status='closed' AND managed_qty=0
    AND settlement_status IN ('settled','legacy')
    AND closed_at>0 AND closed_at<=?`, p.ID, p.AccountID, cutoff).Exec()
		if execErr != nil {
			return PhysicalDeleteResult{}, execErr
		}
		n, execErr := deleted.RowsAffected()
		if execErr != nil {
			return PhysicalDeleteResult{}, execErr
		}
		if n != 1 {
			return PhysicalDeleteResult{}, ErrDeleteChanged
		}
		result.Positions++
	}
	if err = tx.Commit(); err != nil {
		return PhysicalDeleteResult{}, err
	}
	return result, nil
}

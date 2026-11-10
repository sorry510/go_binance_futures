package spotstrategy

import (
	"context"
	"fmt"
)

// RecoverAndSettle first completes the signed REST state inquiry and only
// then attaches newly verified tradeIds to strategy inventory. It never
// retries CREATE, never adopts an unmanaged clientId and never unfreezes a
// reconcile_required row automatically.
func (e *Engine) RecoverAndSettle(ctx context.Context, limit int) (int, error) {
	if e == nil || e.db == nil {
		return 0, ErrReconciliation
	}
	if !e.mockOnly {
		if !e.liveReady {
			return 0, ErrLiveWriteDisabled
		}
		if err := AuthorizeLiveMode(); err != nil {
			return 0, err
		}
	}
	count, firstErr := e.RecoverPending(ctx, limit)
	rows, err := e.db.QueryContext(ctx, `SELECT DISTINCT o.id
 FROM spot_strategy_orders o JOIN spot_strategy_fills f
 ON f.account_id=o.account_id AND f.symbol=o.symbol
 AND f.order_id=o.exchange_order_id AND f.client_order_id=o.client_order_id
 WHERE o.account_id=? AND o.owner=? AND f.rest_verified=1 AND f.holding_id IS NULL
 AND o.status IN ('new','partially_filled','filled','canceled','expired','rejected')
 ORDER BY o.id LIMIT ?`, AccountMain, OwnerAuto, limit)
	if err != nil {
		return count, err
	}
	ids := []int64{}
	for rows.Next() {
		var id int64
		if err = rows.Scan(&id); err != nil {
			break
		}
		ids = append(ids, id)
	}
	if cerr := rows.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return count, err
	}
	for _, id := range ids {
		if _, err = e.ApplyVerifiedFills(ctx, id); err != nil {
			e.markReconcile(ctx, id, "spot_verified_fill_settlement_failed")
			if firstErr == nil {
				firstErr = fmt.Errorf("%w: settlement for %d", ErrReconciliation, id)
			}
		}
	}
	return count, firstErr
}

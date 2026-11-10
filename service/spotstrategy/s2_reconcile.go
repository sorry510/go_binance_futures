package spotstrategy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

const (
	spotTradePageLimit = 500
	spotTradeMaxPages  = 20
)

func safeReadError(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "429"):
		return "spot_rest_rate_limited_429"
	case strings.Contains(msg, "418"):
		return "spot_rest_banned_418"
	case errors.Is(err, context.DeadlineExceeded):
		return "spot_rest_timeout"
	case errors.Is(err, context.Canceled):
		return "spot_rest_cancelled"
	case strings.Contains(msg, "not found"), strings.Contains(msg, "-2013"):
		return "spot_order_not_confirmed"
	default:
		return "spot_rest_unavailable"
	}
}

func (e *Engine) retrieveTrades(ctx context.Context, o LocalOrder, remoteOrderID string) ([]Trade, error) {
	if remoteOrderID == "" {
		return nil, ErrReconciliation
	}
	all := make([]Trade, 0)
	nextID := int64(0)
	seen := map[string]struct{}{}
	for page := 0; page < spotTradeMaxPages; page++ {
		trades, err := e.client.Trades(ctx, o.AccountID, o.Symbol, remoteOrderID, nextID, spotTradePageLimit)
		if err != nil {
			return nil, err
		}
		if len(trades) > spotTradePageLimit {
			return nil, ErrTradePageIncomplete
		}
		var high int64 = nextID - 1
		for _, trade := range trades {
			if err := validateTrade(trade, o.Symbol, remoteOrderID, o.Side); err != nil {
				return nil, err
			}
			id, err := strconv.ParseInt(trade.TradeID, 10, 64)
			if err != nil || id < 0 {
				return nil, fmt.Errorf("%w: invalid Spot trade id", ErrReconciliation)
			}
			if nextID > 0 && id < nextID {
				return nil, fmt.Errorf("%w: backward trade pagination", ErrReconciliation)
			}
			if _, ok := seen[trade.TradeID]; ok {
				continue
			}
			seen[trade.TradeID] = struct{}{}
			if id > high {
				high = id
			}
			all = append(all, trade)
		}
		if len(trades) < spotTradePageLimit {
			return all, nil
		}
		if high < nextID || high == int64(^uint64(0)>>1) {
			return nil, ErrTradePageIncomplete
		}
		nextID = high + 1
	}
	return nil, ErrTradePageIncomplete
}

func (e *Engine) Reconcile(ctx context.Context, id int64) (result LocalOrder, retErr error) {
	o, err := e.loadOrder(ctx, id)
	if err != nil {
		return LocalOrder{}, err
	}
	if ctx.Err() != nil {
		return o, ctx.Err()
	}
	if o.Status == StateReconcile {
		// Quarantine is sticky. Recovery can probe exchange health but must
		// never clear or downgrade evidence without an explicit audited manual
		// procedure (not implemented in S2).
		_, probeErr := e.client.QueryOrder(ctx, o.AccountID, o.Symbol, o.ClientOrderID, o.OrderID)
		_, _ = e.db.ExecContext(ctx, `UPDATE spot_strategy_orders SET last_checked_at=? WHERE id=? AND status='reconcile_required'`,
			time.Now().UnixMilli(), id)
		if probeErr != nil {
			return o, fmt.Errorf("%w: frozen order probe: %s", ErrReconciliation, safeReadError(probeErr))
		}
		return o, ErrReconciliation
	}
	snapshot, readErr := e.client.QueryOrder(ctx, o.AccountID, o.Symbol, o.ClientOrderID, o.OrderID)
	if readErr != nil {
		if errors.Is(readErr, ErrUnmanagedOrder) {
			e.markReconcile(ctx, id, "spot_order_external_identity")
		} else {
			e.markUnknown(ctx, id, safeReadError(readErr))
		}
		updated, _ := e.loadOrder(ctx, id)
		return updated, fmt.Errorf("%w: order query unavailable", ErrReconciliation)
	}
	exchangeID, idErr := strconv.ParseInt(snapshot.OrderID, 10, 64)
	if idErr != nil || exchangeID <= 0 || snapshot.Symbol != o.Symbol || snapshot.Side != o.Side {
		e.markReconcile(ctx, id, "spot_order_identity_mismatch")
		updated, _ := e.loadOrder(ctx, id)
		return updated, ErrUnmanagedOrder
	}
	if o.OrderID != "" && o.OrderID != snapshot.OrderID {
		e.markReconcile(ctx, id, "spot_exchange_order_id_changed")
		updated, _ := e.loadOrder(ctx, id)
		return updated, ErrUnmanagedOrder
	}
	// Even a terminal/canceled order must resolve to our original client ID.
	// A replaced client ID is ambiguous until manual ownership review; neither
	// symbol/side nor an orderId alone proves the external trade is ours.
	if snapshot.ClientOrderID != o.ClientOrderID {
		e.markReconcile(ctx, id, "spot_client_order_id_mismatch")
		updated, _ := e.loadOrder(ctx, id)
		return updated, ErrUnmanagedOrder
	}
	requested, err := parseAmount(o.RequestedQty, false)
	if err != nil {
		return o, err
	}
	if snapshot.RequestedQty == "" || snapshot.ExecutedQuoteQty == "" {
		e.markReconcile(ctx, id, "spot_order_cumulative_fields_missing")
		updated, _ := e.loadOrder(ctx, id)
		return updated, ErrReconciliation
	}
	v, er := parseAmount(snapshot.RequestedQty, false)
	if er != nil || !v.Equal(requested) {
		e.markUnknown(ctx, id, "spot_order_requested_qty_mismatch")
		updated, _ := e.loadOrder(ctx, id)
		return updated, ErrReconciliation
	}
	expected, err := parseAmount(snapshot.ExecutedQty, true)
	if err != nil || expected.GreaterThan(requested) {
		e.markUnknown(ctx, id, "spot_order_executed_qty_invalid")
		return e.loadOrder(ctx, id)
	}
	remoteState, stateErr := nextState(snapshot.Status)
	if stateErr != nil {
		e.markUnknown(ctx, id, "spot_order_state_unknown")
		return e.loadOrder(ctx, id)
	}
	fills, err := e.retrieveTrades(ctx, o, snapshot.OrderID)
	if err != nil {
		if errors.Is(err, ErrReconciliation) || errors.Is(err, ErrTradePageIncomplete) {
			e.markReconcile(ctx, id, "spot_trade_response_invalid_or_incomplete")
		} else {
			e.markUnknown(ctx, id, safeReadError(err))
		}
		// Remember an accepted exchange identity even when the fills endpoint is
		// unavailable. Next pass will query this same orderId, never resubmit.
		_, _ = e.db.ExecContext(ctx, "UPDATE spot_strategy_orders SET exchange_order_id=? WHERE id=? AND (exchange_order_id IS NULL OR exchange_order_id='')", snapshot.OrderID, id)
		now, _ := e.loadOrder(ctx, id)
		return now, fmt.Errorf("%w: trades unavailable or incomplete", ErrReconciliation)
	}
	// Authenticated transport provenance is checked outside the SQL
	// transaction. CurrentMode queries the config table and must never
	// require another pooled connection while holding a Fill row lock.
	signedEvidence := canAttestLiveFills(e)
	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return o, err
	}
	defer func() {
		_ = tx.Rollback()
		// If a fill identity or order-state invariant fails inside the
		// transaction, rollback first, THEN mark the pending order as requiring
		// reconciliation. Never release pending_slot on unverified data.
		if errors.Is(retErr, ErrReconciliation) {
			e.markReconcile(ctx, id, "spot_fill_or_order_invariant_failed")
		}
	}()
	// Lock the owned order inside the same transaction that records all fills.
	var currentStatus string
	var currentExchangeID sql.NullString
	err = tx.QueryRowContext(ctx, "SELECT status,exchange_order_id FROM spot_strategy_orders WHERE id=?"+e.forUpdate(), id).Scan(&currentStatus, &currentExchangeID)
	if err != nil {
		return o, err
	}
	if currentExchangeID.Valid && currentExchangeID.String != "" && currentExchangeID.String != snapshot.OrderID {
		return o, ErrReconciliation
	}
	if currentStatus == StateReconcile {
		return o, fmt.Errorf("%w: quarantined while reconciliation was in progress", ErrReconciliation)
	}
	if isTerminal(currentStatus) && currentStatus != remoteState {
		return o, fmt.Errorf("%w: stale exchange status after terminal", ErrReconciliation)
	}
	for _, trade := range fills {
		_, insertErr := tx.ExecContext(ctx, `INSERT INTO spot_strategy_fills
 (account_id,symbol,order_id,trade_id,holding_id,client_order_id,side,qty,price,quote_qty,commission,commission_asset,trade_time,rest_verified)
 VALUES (?,?,?,?,NULL,?,?,?,?,?,?,?,?,?)`,
			o.AccountID, o.Symbol, snapshot.OrderID, trade.TradeID, o.ClientOrderID, o.Side,
			trade.Qty, trade.Price, trade.QuoteQty, trade.Commission, trade.CommissionAsset,
			trade.Time, signedEvidence) // Signed production REST and independent live mode are both required.
		if insertErr != nil {
			msg := strings.ToLower(insertErr.Error())
			if !strings.Contains(msg, "unique constraint") && !strings.Contains(msg, "duplicate entry") {
				return o, insertErr
			}
			// Identical replay is idempotent, but a conflicting tradeId with changed
			// execution data must fail closed instead of silently reusing the record.
			var qty, price, quote, commission, asset string
			err = tx.QueryRowContext(ctx, `SELECT qty,price,quote_qty,commission,commission_asset FROM spot_strategy_fills
 WHERE account_id=? AND symbol=? AND order_id=? AND trade_id=?`,
				o.AccountID, o.Symbol, snapshot.OrderID, trade.TradeID).Scan(&qty, &price, &quote, &commission, &asset)
			if err != nil {
				return o, err
			}
			for _, pair := range [][2]string{{qty, trade.Qty}, {price, trade.Price}, {quote, trade.QuoteQty}, {commission, trade.Commission}} {
				a, e1 := decimal.NewFromString(pair[0])
				b, e2 := decimal.NewFromString(pair[1])
				if e1 != nil || e2 != nil || !a.Equal(b) {
					return o, fmt.Errorf("%w: changed fill for same tradeId", ErrReconciliation)
				}
			}
			if asset != trade.CommissionAsset {
				return o, fmt.Errorf("%w: changed commission asset", ErrReconciliation)
			}
		}
	}
	rows, err := tx.QueryContext(ctx, `SELECT qty,quote_qty FROM spot_strategy_fills WHERE account_id=? AND symbol=? AND order_id=?`,
		o.AccountID, o.Symbol, snapshot.OrderID)
	if err != nil {
		return o, err
	}
	qtyTotal, quoteTotal := decimal.Zero, decimal.Zero
	for rows.Next() {
		var q, qt string
		if err = rows.Scan(&q, &qt); err != nil {
			break
		}
		var v, w decimal.Decimal
		v, err = decimal.NewFromString(q)
		if err != nil {
			break
		}
		w, err = decimal.NewFromString(qt)
		if err != nil {
			break
		}
		qtyTotal = qtyTotal.Add(v)
		quoteTotal = quoteTotal.Add(w)
	}
	if closeErr := rows.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return o, err
	}
	if qtyTotal.GreaterThan(requested) {
		return o, ErrReconciliation
	}
	newStatus := remoteState
	if !qtyTotal.Equal(expected) {
		newStatus = StateReconcile
	}
	if remoteState == StateNew && qtyTotal.IsPositive() {
		newStatus = StateReconcile
	}
	if remoteState == StatePartial && (qtyTotal.IsZero() || qtyTotal.Equal(requested)) {
		newStatus = StateReconcile
	}
	if remoteState == StateFilled && !qtyTotal.Equal(requested) {
		newStatus = StateReconcile
	}
	if snapshot.ExecutedQuoteQty != "" {
		remoteQuote, parseErr := decimal.NewFromString(snapshot.ExecutedQuoteQty)
		if parseErr != nil || remoteQuote.IsNegative() || !remoteQuote.Equal(quoteTotal) {
			newStatus = StateReconcile
		}
	}
	reserved := o.ReservedQty
	slot := sql.NullString{String: AccountMain + ":" + OwnerAuto + ":" + o.Symbol, Valid: true}
	if isTerminal(newStatus) {
		reserved = "0"
		slot.Valid = false
	}
	lastErr := ""
	if newStatus == StateReconcile {
		lastErr = "spot_fills_incomplete_or_mismatch"
	}
	now := time.Now().UnixMilli()
	_, err = tx.ExecContext(ctx, `UPDATE spot_strategy_orders SET exchange_order_id=?,status=?,filled_qty=?,executed_quote_qty=?,
 reserved_qty=?,pending_slot=?,last_error=?,last_checked_at=?,updated_at=? WHERE id=?`,
		snapshot.OrderID, newStatus, qtyTotal.String(), quoteTotal.String(), reserved, slot,
		lastErr, now, now, id)
	if err != nil {
		return o, err
	}
	if err = tx.Commit(); err != nil {
		return o, err
	}
	updated, err := e.loadOrder(ctx, id)
	if newStatus == StateReconcile {
		return updated, ErrReconciliation
	}
	return updated, err
}

// CancelMock implements query -> Fake cancel -> query/trades again; it cannot
// cancel an external/unmanaged order and never interprets DELETE ACK as flat.
func (e *Engine) CancelMock(ctx context.Context, id int64) (LocalOrder, error) {
	if !e.mockOnly {
		return LocalOrder{}, ErrLiveWriteDisabled
	}
	o, err := e.Reconcile(ctx, id)
	if err != nil {
		return o, err
	}
	if o.OrderID == "" || (o.Status != StateNew && o.Status != StatePartial) {
		return o, fmt.Errorf("Spot cancellation not permitted in state %s", o.Status)
	}
	if err = e.client.Cancel(ctx, o.AccountID, o.Symbol, o.ClientOrderID); err != nil {
		e.markUnknown(ctx, id, "spot_cancel_outcome_unknown")
		now, _ := e.loadOrder(ctx, id)
		return now, ErrReconciliation
	}
	return e.Reconcile(ctx, id)
}

// RecoverPending is an explicit, bounded, read-only-exchange operation;
// it is NOT a new goroutine/scheduler and never resubmits unknown orders.
func (e *Engine) RecoverPending(ctx context.Context, limit int) (int, error) {
	if limit < 1 || limit > 50 {
		return 0, errors.New("recovery limit must be 1..50")
	}
	cutoff := time.Now().Add(-3 * time.Second).UnixMilli()
	rows, err := e.db.QueryContext(ctx, `SELECT id FROM spot_strategy_orders
 WHERE account_id=? AND owner=? AND status IN ('reserved','submitting','unknown','new','partially_filled','reconcile_required')
 AND last_checked_at < ? ORDER BY id LIMIT ?`, AccountMain, OwnerAuto, cutoff, limit)
	if err != nil {
		return 0, err
	}
	ids := []int64{}
	for rows.Next() {
		var id int64
		if err = rows.Scan(&id); err != nil {
			break
		}
		ids = append(ids, id)
	}
	if closeErr := rows.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return 0, err
	}
	var firstErr error
	for _, id := range ids {
		_, e2 := e.Reconcile(ctx, id)
		if e2 != nil && firstErr == nil {
			firstErr = e2
		}
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
	}
	return len(ids), firstErr
}

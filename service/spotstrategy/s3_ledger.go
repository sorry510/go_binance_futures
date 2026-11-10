package spotstrategy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

type Holding struct {
	ID           int64  `json:"holding_id"`
	AccountID    string `json:"account_id"`
	Owner        string `json:"owner"`
	Symbol       string `json:"symbol"`
	BaseAsset    string `json:"base_asset"`
	QuoteAsset   string `json:"quote_asset"`
	OwnedQty     string `json:"owned_qty"`
	AvgCostQuote string `json:"avg_cost_quote"`
	Status       string `json:"status"`
}
type HoldingSummary struct {
	Holding
	CostBasisQuote          string            `json:"cost_basis_quote"`
	RealizedGrossQuote      string            `json:"realized_gross_quote"`
	RealizedNetQuote        string            `json:"realized_net_quote,omitempty"`
	Fees                    map[string]string `json:"fees"`
	Complete                bool              `json:"complete"`
	IncompleteReason        string            `json:"incomplete_reason,omitempty"`
	UnrealizedEstimateQuote string            `json:"unrealized_estimate_quote,omitempty"`
	ReferencePriceQuote     string            `json:"reference_price_quote,omitempty"`
	ReferencePriceTime      int64             `json:"reference_price_time,omitempty"`
	ReferencePriceFresh     bool              `json:"reference_price_fresh"`
}

func readHolding(row *sql.Row) (Holding, error) {
	var h Holding
	err := row.Scan(&h.ID, &h.AccountID, &h.Owner, &h.Symbol, &h.BaseAsset, &h.QuoteAsset, &h.OwnedQty, &h.AvgCostQuote, &h.Status)
	return h, err
}

const holdingCols = "id,account_id,owner,symbol,base_asset,quote_asset,CAST(owned_qty AS CHAR),CAST(avg_cost_quote AS CHAR),status"

type settlementFill struct {
	orderID                string
	tradeID                string
	side                   string
	qty, quote, commission string
	feeAsset               string
	verified               int
	linked                 sql.NullInt64
	timestamp              int64
}

// ApplyVerifiedFills consumes ONLY verified signed REST trades into the
// strategy-owned Holding, using the same SQL transaction for both changes.
// No ACK, cumulative order quantity, unverified fixture or external Spot
// balance can create inventory. This is the real production settlement path.
func (e *Engine) ApplyVerifiedFills(ctx context.Context, localOrderID int64) (result Holding, retErr error) {
	if e == nil || e.db == nil {
		return Holding{}, errors.New("Spot ledger unavailable")
	}
	if e.client == nil {
		// Read-only listing uses NewReadOnlyEngine(db,nil), which has no
		// instrument or signed REST source and must never settle inventory.
		return Holding{}, ErrLiveWriteDisabled
	}
	// Read authoritative instrument metadata BEFORE acquiring a database
	// transaction; do not hold an account/holding row lock during REST I/O.
	orderBefore, err := e.loadOrder(ctx, localOrderID)
	if err != nil {
		return Holding{}, err
	}
	var buySpec Instrument
	if orderBefore.Side == "BUY" {
		buySpec, err = e.client.Instrument(ctx, orderBefore.Symbol)
		if err != nil {
			return Holding{}, fmt.Errorf("%w: exchangeInfo unavailable", ErrReconciliation)
		}
		if buySpec.Symbol != orderBefore.Symbol || buySpec.BaseAsset == "" || buySpec.QuoteAsset == "" ||
			buySpec.Status != "TRADING" || !buySpec.SpotAllowed {
			return Holding{}, ErrReconciliation
		}
	}
	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return Holding{}, err
	}
	defer func() {
		_ = tx.Rollback()
		// An authenticated trade settlement failure is a safety event. Leave
		// the original fills intact, freeze the order and block fresh risk.
		// Old isolated S2 test fixtures are not treated as real inventory.
		if e.liveReady && errors.Is(retErr, ErrReconciliation) {
			e.markReconcile(ctx, localOrderID, "spot_inventory_settlement_failed")
		}
	}()
	var account, owner, symbol, clientID, exchangeID, side, status string
	var orderHolding sql.NullInt64
	err = tx.QueryRowContext(ctx, `SELECT account_id,owner,symbol,client_order_id,
 COALESCE(exchange_order_id,''),side,status,holding_id
 FROM spot_strategy_orders WHERE id=?`+e.forUpdate(), localOrderID).
		Scan(&account, &owner, &symbol, &clientID, &exchangeID, &side, &status, &orderHolding)
	if errors.Is(err, sql.ErrNoRows) {
		return Holding{}, ErrUnmanagedOrder
	}
	if err != nil {
		return Holding{}, err
	}
	if err = ValidateScope(account, owner); err != nil {
		return Holding{}, ErrUnmanagedOrder
	}
	if exchangeID == "" || status == StateReserved || status == StateSubmitting ||
		status == StateUnknown || status == StateReconcile {
		return Holding{}, ErrReconciliation
	}
	if side != "BUY" && side != "SELL" {
		return Holding{}, ErrReconciliation
	}
	rows, err := tx.QueryContext(ctx, `SELECT order_id,trade_id,side,CAST(qty AS CHAR),CAST(quote_qty AS CHAR),
 CAST(commission AS CHAR),commission_asset,rest_verified,holding_id,trade_time
 FROM spot_strategy_fills WHERE account_id=? AND symbol=? AND order_id=?
 AND client_order_id=? ORDER BY trade_time,trade_id`, account, symbol, exchangeID, clientID)
	if err != nil {
		return Holding{}, err
	}
	fills := []settlementFill{}
	for rows.Next() {
		var f settlementFill
		if err = rows.Scan(&f.orderID, &f.tradeID, &f.side, &f.qty, &f.quote,
			&f.commission, &f.feeAsset, &f.verified, &f.linked, &f.timestamp); err != nil {
			break
		}
		fills = append(fills, f)
	}
	if closeErr := rows.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return Holding{}, err
	}
	if len(fills) == 0 {
		return Holding{}, ErrReconciliation
	}
	for _, f := range fills {
		if f.verified != 1 || f.side != side || f.orderID != exchangeID {
			return Holding{}, ErrReconciliation
		}
		if f.linked.Valid && (!orderHolding.Valid || f.linked.Int64 != orderHolding.Int64) {
			return Holding{}, ErrReconciliation
		}
	}
	const slotPrefix = AccountMain + ":" + OwnerAuto + ":"
	slot := slotPrefix + symbol
	var h Holding
	if orderHolding.Valid {
		h, err = readHolding(tx.QueryRowContext(ctx, "SELECT "+holdingCols+" FROM spot_strategy_holdings WHERE id=?"+e.forUpdate(), orderHolding.Int64))
		if err != nil {
			return Holding{}, ErrReconciliation
		}
	} else if side == "SELL" {
		return Holding{}, ErrUnmanagedOrder
	} else {
		h, err = readHolding(tx.QueryRowContext(ctx, "SELECT "+holdingCols+" FROM spot_strategy_holdings WHERE active_slot=?"+e.forUpdate(), slot))
		if errors.Is(err, sql.ErrNoRows) {
			// When the first genuine BUY Fill arrives, establish the only active
			// inventory cycle. MySQL serializes on a stable spot_symbols row.
			var symbolID int64
			if err = tx.QueryRowContext(ctx, "SELECT id FROM spot_symbols WHERE symbol=?"+e.forUpdate(), symbol).Scan(&symbolID); err != nil {
				return Holding{}, err
			}
			if buySpec.Symbol != symbol {
				return Holding{}, ErrReconciliation
			}
			now := time.Now().UnixMilli()
			inserted, insertErr := tx.ExecContext(ctx, `INSERT INTO spot_strategy_holdings
   (account_id,owner,symbol,base_asset,quote_asset,owned_qty,avg_cost_quote,status,active_slot,created_at,updated_at)
   VALUES (?,?,?,?,?,'0','0','active',?,?,?)`,
				account, owner, symbol, buySpec.BaseAsset, buySpec.QuoteAsset, slot, now, now)
			if insertErr != nil {
				return Holding{}, fmt.Errorf("%w: holding cycle busy", ErrReconciliation)
			}
			holdingID, idErr := inserted.LastInsertId()
			if idErr != nil {
				return Holding{}, idErr
			}
			h = Holding{ID: holdingID, AccountID: account, Owner: owner, Symbol: symbol,
				BaseAsset: buySpec.BaseAsset, QuoteAsset: buySpec.QuoteAsset, OwnedQty: "0", AvgCostQuote: "0", Status: "active"}
		} else if err != nil {
			return Holding{}, err
		}
		// A distinct BUY cannot add inventory to an existing cycle. Replayed Fill
		// from the SAME original BUY order can still update it incrementally.
		var another int64
		err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM spot_strategy_orders
   WHERE holding_id=? AND side='BUY' AND id<>?`, h.ID, localOrderID).Scan(&another)
		if err != nil || another > 0 {
			return Holding{}, ErrReconciliation
		}
	}
	if h.AccountID != account || h.Owner != owner || h.Symbol != symbol {
		return Holding{}, ErrUnmanagedOrder
	}
	if h.Status != "active" {
		allLinked := len(fills) > 0
		for _, f := range fills {
			if !f.linked.Valid || f.linked.Int64 != h.ID {
				allLinked = false
				break
			}
		}
		if h.Status == "closed" && side == "SELL" && allLinked && isTerminal(status) {
			if err = tx.Commit(); err != nil {
				return Holding{}, err
			}
			return h, nil
		}
		return Holding{}, ErrReconciliation
	}
	if side == "SELL" && !orderHolding.Valid {
		return Holding{}, ErrUnmanagedOrder
	}
	owned, err := decimal.NewFromString(h.OwnedQty)
	if err != nil {
		return Holding{}, err
	}
	for _, f := range fills {
		if f.linked.Valid {
			continue
		} // idempotence: already consumed Fill
		qty, qtyErr := parseAmount(f.qty, false)
		_, quoteErr := parseAmount(f.quote, true)
		fee, feeErr := parseAmount(f.commission, true)
		if qtyErr != nil || quoteErr != nil || feeErr != nil {
			return Holding{}, ErrReconciliation
		}
		baseFee := decimal.Zero
		if f.feeAsset == h.BaseAsset {
			baseFee = fee
		}
		if side == "BUY" {
			credited := qty.Sub(baseFee)
			if !credited.IsPositive() {
				return Holding{}, ErrReconciliation
			}
			// Inventory increases only by actual credited base. All quote
			// acquisition cost is rebuilt from the immutable linked Fill
			// history after linking; never round-trip through stored avg.
			owned = owned.Add(credited)
		} else {
			// Base-denominated fee on SELL is an additional base debit, never a
			// quote expense. Refuse if it would spend another owner's inventory.
			consumed := qty.Add(baseFee)
			if consumed.GreaterThan(owned) {
				return Holding{}, ErrReconciliation
			}
			owned = owned.Sub(consumed)
		}
		res, linkErr := tx.ExecContext(ctx, `UPDATE spot_strategy_fills SET holding_id=?
    WHERE account_id=? AND symbol=? AND order_id=? AND trade_id=?
    AND holding_id IS NULL AND rest_verified=1`,
			h.ID, account, symbol, exchangeID, f.tradeID)
		if linkErr != nil {
			return Holding{}, linkErr
		}
		affected, affectedErr := res.RowsAffected()
		if affectedErr != nil || affected != 1 {
			return Holding{}, ErrReconciliation
		}
	}
	if !orderHolding.Valid {
		res, updateErr := tx.ExecContext(ctx, `UPDATE spot_strategy_orders SET holding_id=?
   WHERE id=? AND holding_id IS NULL`, h.ID, localOrderID)
		if updateErr != nil {
			return Holding{}, updateErr
		}
		count, countErr := res.RowsAffected()
		if countErr != nil || count != 1 {
			return Holding{}, ErrReconciliation
		}
	}
	// Reconstruct from ALL linked, authenticated Fill rows in chronological
	// order. Stored avg_cost_quote is a rounded cache, not an input.
	replayedQty, exactBasis, replayErr := replayHoldingCost(ctx, tx, h)
	if replayErr != nil || !replayedQty.Equal(owned) {
		return Holding{}, ErrReconciliation
	}
	avg := decimal.Zero
	if owned.IsPositive() {
		avg = exactBasis.DivRound(owned, 18)
	}
	newStatus := "active"
	var activeSlot interface{} = slot
	closedAt := int64(0)
	if side == "SELL" && owned.IsZero() && isTerminal(status) {
		newStatus = "closed"
		activeSlot = nil
		closedAt = time.Now().UnixMilli()
	}
	_, err = tx.ExecContext(ctx, `UPDATE spot_strategy_holdings
 SET owned_qty=?,avg_cost_quote=?,status=?,active_slot=?,closed_at=?,updated_at=?
 WHERE id=?`, owned.String(), avg.String(), newStatus, activeSlot, closedAt, time.Now().UnixMilli(), h.ID)
	if err != nil {
		return Holding{}, err
	}
	if err = tx.Commit(); err != nil {
		return Holding{}, err
	}
	h.OwnedQty = owned.String()
	h.AvgCostQuote = avg.String()
	h.Status = newStatus
	return h, nil
}

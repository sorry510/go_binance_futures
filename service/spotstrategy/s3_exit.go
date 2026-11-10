package spotstrategy

import (
	"context"
	"errors"
	"fmt"
	"github.com/shopspring/decimal"
	"strings"
	"time"
)

// PrepareExit creates a real SELL intent against an existing strategy-owned
// inventory cycle. It NEVER takes the account's entire free quantity as the
// owned quantity; the position is derived exclusively from signed Fill links.
// Empty Quantity means close the maximum conservative amount.
func (e *Engine) PrepareExit(ctx context.Context, req OrderRequest) (LocalOrder, error) {
	if e == nil || e.db == nil || e.client == nil {
		return LocalOrder{}, ErrLiveWriteDisabled
	}
	if !e.mockOnly {
		if !e.liveReady {
			return LocalOrder{}, ErrLiveWriteDisabled
		}
		if err := AuthorizeLiveMode(); err != nil {
			return LocalOrder{}, err
		}
	}
	if err := ValidateScope(req.AccountID, req.Owner); err != nil {
		return LocalOrder{}, err
	}
	if req.Side != "SELL" || req.Symbol == "" || strings.ToUpper(req.Symbol) != req.Symbol ||
		!signalKeyPattern.MatchString(req.SignalKey) {
		return LocalOrder{}, errors.New("invalid Spot exit scope")
	}
	if req.Type != "LIMIT" && req.Type != "MARKET" {
		return LocalOrder{}, errors.New("Spot exit only LIMIT or MARKET")
	}
	// Require a fresh price supplied by the caller; the caller must not use a
	// cached/expired ticker to justify a new executable order.
	priceText := req.Price
	if req.Type == "MARKET" {
		priceText = req.EstimatePrice
	}
	price, err := parseAmount(priceText, false)
	if err != nil {
		return LocalOrder{}, err
	}
	spec, err := e.client.Instrument(ctx, req.Symbol)
	if err != nil {
		return LocalOrder{}, fmt.Errorf("exchangeInfo: %s", safeReadError(err))
	}
	if spec.Symbol != req.Symbol || spec.Status != "TRADING" || !spec.SpotAllowed ||
		spec.BaseAsset == "" || spec.QuoteAsset == "" {
		return LocalOrder{}, ErrReconciliation
	}
	balances, err := e.client.Balances(ctx, AccountMain)
	if err != nil {
		return LocalOrder{}, fmt.Errorf("Spot balance: %s", safeReadError(err))
	}
	base, ok := balances[spec.BaseAsset]
	if !ok {
		return LocalOrder{}, ErrReconciliation
	}
	free, err := parseAmount(base.Free, true)
	if err != nil {
		return LocalOrder{}, err
	}
	if _, err = parseAmount(base.Locked, true); err != nil {
		return LocalOrder{}, err
	}
	open, err := e.client.OpenOrders(ctx, AccountMain, req.Symbol)
	if err != nil {
		return LocalOrder{}, fmt.Errorf("Spot openOrders: %s", safeReadError(err))
	}
	if len(open) != 0 {
		return LocalOrder{}, fmt.Errorf("%w: unknown external open Spot orders", ErrReconciliation)
	}
	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return LocalOrder{}, err
	}
	defer tx.Rollback()
	var configCount int
	if err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM config").Scan(&configCount); err != nil || configCount != 1 {
		return LocalOrder{}, ErrReconciliation
	}
	var accountID int64
	if err = tx.QueryRowContext(ctx, "SELECT id FROM config LIMIT 1"+e.forUpdate()).Scan(&accountID); err != nil {
		return LocalOrder{}, err
	}
	var enabled int
	var quote string
	err = tx.QueryRowContext(ctx, "SELECT strategy_trade_enabled,type FROM spot_symbols WHERE symbol=?"+e.forUpdate(), req.Symbol).Scan(&enabled, &quote)
	if err != nil || enabled != 1 || quote != spec.QuoteAsset {
		return LocalOrder{}, ErrReconciliation
	}
	h, err := readHolding(tx.QueryRowContext(ctx,
		"SELECT "+holdingCols+" FROM spot_strategy_holdings WHERE active_slot=?"+e.forUpdate(),
		AccountMain+":"+OwnerAuto+":"+req.Symbol))
	if err != nil {
		return LocalOrder{}, ErrUnmanagedOrder
	}
	if h.AccountID != AccountMain || h.Owner != OwnerAuto || h.Symbol != req.Symbol ||
		h.BaseAsset != spec.BaseAsset || h.QuoteAsset != spec.QuoteAsset || h.Status != "active" {
		return LocalOrder{}, ErrUnmanagedOrder
	}
	owned, err := parseAmount(h.OwnedQty, false)
	if err != nil {
		return LocalOrder{}, ErrReconciliation
	}
	// A Holding row by itself is NOT ownership proof. Reconstruct exact
	// strategy inventory from its authenticated, linked Fill identities.
	// In particular a manually inserted Holding can never authorize a SELL.
	verifiedRows, verifyErr := tx.QueryContext(ctx, `SELECT side,CAST(qty AS CHAR),
	CAST(commission AS CHAR),commission_asset,rest_verified
	FROM spot_strategy_fills WHERE holding_id=? AND account_id=? AND symbol=?`,
		h.ID, AccountMain, req.Symbol)
	if verifyErr != nil {
		return LocalOrder{}, verifyErr
	}
	provenBase := decimal.Zero
	provenCount := 0
	for verifiedRows.Next() {
		var fillSide, qtyText, feeText, asset string
		var flag int
		if verifyErr = verifiedRows.Scan(&fillSide, &qtyText, &feeText, &asset, &flag); verifyErr != nil {
			break
		}
		provenCount++
		if flag != 1 {
			verifyErr = ErrReconciliation
			break
		}
		q, qErr := parseAmount(qtyText, false)
		fee, feeErr := parseAmount(feeText, true)
		if qErr != nil || feeErr != nil {
			verifyErr = ErrReconciliation
			break
		}
		baseFee := decimal.Zero
		if asset == h.BaseAsset {
			baseFee = fee
		}
		switch fillSide {
		case "BUY":
			provenBase = provenBase.Add(q.Sub(baseFee))
		case "SELL":
			provenBase = provenBase.Sub(q.Add(baseFee))
		default:
			verifyErr = ErrReconciliation
		}
		if verifyErr != nil {
			break
		}
	}
	if closeErr := verifiedRows.Close(); verifyErr == nil {
		verifyErr = closeErr
	}
	if verifyErr != nil || provenCount == 0 || !provenBase.Equal(owned) {
		return LocalOrder{}, ErrReconciliation
	}
	// If an earlier order cannot be trusted or is still active, never take a
	// new SELL; this also stops BUY/SELL crossing the same inventory cycle.
	var pending int
	err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM spot_strategy_orders
 WHERE account_id=? AND owner=? AND symbol=? AND
 (pending_slot IS NOT NULL OR status='reconcile_required')`,
		AccountMain, OwnerAuto, req.Symbol).Scan(&pending)
	if err != nil {
		return LocalOrder{}, err
	}
	if pending > 0 {
		return LocalOrder{}, ErrReconciliation
	}
	if free.LessThan(owned) {
		return LocalOrder{}, fmt.Errorf("%w: physical Spot free below owned inventory", ErrReconciliation)
	}
	lotMin, lotStep, lotMax := spec.LotMin, spec.LotStep, spec.LotMax
	if req.Type == "MARKET" && spec.MarketLotStep != "" && spec.MarketLotStep != "0" {
		lotMin, lotStep, lotMax = spec.MarketLotMin, spec.MarketLotStep, spec.MarketLotMax
	}
	if lotStep == "" || lotStep == "0" || lotMin == "" {
		return LocalOrder{}, ErrReconciliation
	}
	step, err := parseAmount(lotStep, false)
	if err != nil {
		return LocalOrder{}, err
	}
	// Never request more than proven strategy inventory. Exchange-confirmed
	// SELL base fees (if any) are reconciled separately before closing; if
	// they would consume extra inventory, quarantine rather than guessing.
	maxSafe := owned.Div(step).Floor().Mul(step)
	qty := maxSafe
	if req.Quantity != "" {
		qty, err = parseAmount(req.Quantity, false)
		if err != nil {
			return LocalOrder{}, err
		}
		if qty.GreaterThan(maxSafe) {
			return LocalOrder{}, ErrReconciliation
		}
	}
	if qty.IsZero() || qty.GreaterThan(free) {
		return LocalOrder{}, ErrReconciliation
	}
	if !qty.Mod(step).IsZero() {
		return LocalOrder{}, fmt.Errorf("Spot SELL step invalid")
	}
	if lotMin != "" && lotMin != "0" {
		min, parseErr := parseAmount(lotMin, false)
		if parseErr != nil || qty.LessThan(min) {
			return LocalOrder{}, fmt.Errorf("Spot SELL below minQty, retain dust")
		}
	}
	if lotMax != "" && lotMax != "0" {
		max, parseErr := parseAmount(lotMax, false)
		if parseErr != nil || qty.GreaterThan(max) {
			return LocalOrder{}, fmt.Errorf("Spot SELL above maxQty")
		}
	}
	if req.Type == "LIMIT" {
		tick, tickErr := parseAmount(spec.TickSize, false)
		if tickErr != nil || !price.Mod(tick).IsZero() {
			return LocalOrder{}, fmt.Errorf("Spot SELL tick invalid")
		}
		if spec.PriceMin != "" && spec.PriceMin != "0" {
			minPrice, e := parseAmount(spec.PriceMin, false)
			if e != nil || price.LessThan(minPrice) {
				return LocalOrder{}, fmt.Errorf("Spot SELL price below minPrice")
			}
		}
		if spec.PriceMax != "" && spec.PriceMax != "0" {
			maxPrice, e := parseAmount(spec.PriceMax, false)
			if e != nil || price.GreaterThan(maxPrice) {
				return LocalOrder{}, fmt.Errorf("Spot SELL price above maxPrice")
			}
		}
	}
	notional := qty.Mul(price)
	if spec.MinNotional == "" {
		return LocalOrder{}, ErrReconciliation
	}
	minNotional, err := parseAmount(spec.MinNotional, true)
	if err != nil || notional.LessThan(minNotional) {
		return LocalOrder{}, fmt.Errorf("Spot SELL below min notional, retain dust")
	}
	if spec.MaxNotional != "" && spec.MaxNotional != "0" {
		maxNotional, parseErr := parseAmount(spec.MaxNotional, false)
		if parseErr != nil || notional.GreaterThan(maxNotional) {
			return LocalOrder{}, fmt.Errorf("Spot SELL above max notional")
		}
	}
	if req.ClientOrderID == "" {
		req.ClientOrderID, err = newClientID()
		if err != nil {
			return LocalOrder{}, err
		}
	}
	if !clientIDPattern.MatchString(req.ClientOrderID) {
		return LocalOrder{}, errors.New("invalid Spot clientOrderID")
	}
	now := time.Now().UnixMilli()
	res, err := tx.ExecContext(ctx, `INSERT INTO spot_strategy_orders
 (holding_id,account_id,owner,symbol,side,order_type,intent,
 client_order_id,signal_key,pending_slot,requested_qty,reserved_qty,status,created_at,updated_at)
 VALUES (?,?,?,?,'SELL',?,'close',?,?,?,?,?,'reserved',?,?)`,
		h.ID, AccountMain, OwnerAuto, req.Symbol, req.Type, req.ClientOrderID, req.SignalKey,
		AccountMain+":"+OwnerAuto+":"+req.Symbol, qty.String(), qty.String(), now, now)
	if err != nil {
		lower := strings.ToLower(err.Error())
		if strings.Contains(lower, "unique") || strings.Contains(lower, "duplicate") {
			return LocalOrder{}, ErrDuplicate
		}
		return LocalOrder{}, fmt.Errorf("persist Spot exit intent: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return LocalOrder{}, err
	}
	if err = tx.Commit(); err != nil {
		return LocalOrder{}, err
	}
	return e.loadOrder(ctx, id)
}

// CancelLive is the actual authenticated exchange cancellation chain. A
// canceled ACK is NOT a closing trade; a fresh signed query and myTrades
// reconciliation follow before inventory is released.
func (e *Engine) CancelLive(ctx context.Context, id int64) (LocalOrder, error) {
	if e == nil || !e.liveReady || e.mockOnly {
		return LocalOrder{}, ErrLiveWriteDisabled
	}
	if err := AuthorizeLiveMode(); err != nil {
		return LocalOrder{}, err
	}
	o, err := e.Reconcile(ctx, id)
	if err != nil {
		return o, err
	}
	if o.OrderID == "" || (o.Status != StateNew && o.Status != StatePartial) {
		return o, ErrReconciliation
	}
	if err = e.client.Cancel(ctx, o.AccountID, o.Symbol, o.ClientOrderID); err != nil {
		e.markUnknown(ctx, id, "spot_cancel_outcome_unknown")
		updated, _ := e.loadOrder(ctx, id)
		return updated, ErrReconciliation
	}
	return e.Reconcile(ctx, id)
}

package spotstrategy

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/beego/beego/v2/core/logs"
	"github.com/shopspring/decimal"
)

type Engine struct {
	db        *sql.DB
	client    Client
	mockOnly  bool
	liveReady bool
	mysql     bool
}

// NewMockEngine can be used by S2 tests or S3 test fixtures only. There is
// deliberately no production order-writer constructor or HTTP route.
func NewMockEngine(db *sql.DB, fake Client) *Engine {
	return &Engine{db: db, client: fake, mockOnly: true, mysql: strings.Contains(strings.ToLower(fmt.Sprintf("%T", db.Driver())), "mysql")}
}
func NewReadOnlyEngine(db *sql.DB, client Client) *Engine {
	return &Engine{db: db, client: client, mockOnly: false, mysql: strings.Contains(strings.ToLower(fmt.Sprintf("%T", db.Driver())), "mysql")}
}

// NewLiveEngine wires production order handling but only with a signed REST
// adapter. It is NOT permission to trade. Runtime calls separately require
// current strategy mode and explicit out-of-band operator approval.
func NewLiveEngine(db *sql.DB, client Client) (*Engine, error) {
	provenance, ok := client.(signedRESTTradeEvidence)
	if !ok || !provenance.SpotSignedRESTProvenance() {
		return nil, ErrLiveWriteDisabled
	}
	return &Engine{db: db, client: client, liveReady: true,
		mysql: strings.Contains(strings.ToLower(fmt.Sprintf("%T", db.Driver())), "mysql")}, nil
}

type LocalOrder struct {
	ID                                                                                    int64
	AccountID, Owner, Symbol, ClientOrderID, OrderID, SignalKey                           string
	Side, Type, Status, RequestedQty, FilledQty, ExecutedQuoteQty, ReservedQty, LastError string
	LastCheckedAt                                                                         int64
}

func (e *Engine) loadOrder(ctx context.Context, id int64) (LocalOrder, error) {
	var out LocalOrder
	err := e.db.QueryRowContext(ctx, `SELECT id,account_id,owner,symbol,client_order_id,COALESCE(exchange_order_id,''),
 signal_key,side,order_type,status,requested_qty,filled_qty,executed_quote_qty,reserved_qty,
 last_error,last_checked_at FROM spot_strategy_orders WHERE id=?`, id).Scan(
		&out.ID, &out.AccountID, &out.Owner, &out.Symbol, &out.ClientOrderID, &out.OrderID,
		&out.SignalKey, &out.Side, &out.Type, &out.Status, &out.RequestedQty, &out.FilledQty,
		&out.ExecutedQuoteQty, &out.ReservedQty, &out.LastError, &out.LastCheckedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return LocalOrder{}, ErrUnmanagedOrder
		}
		return LocalOrder{}, err
	}
	if err := ValidateScope(out.AccountID, out.Owner); err != nil {
		return LocalOrder{}, ErrUnmanagedOrder
	}
	return out, nil
}
func (e *Engine) Load(ctx context.Context, id int64) (LocalOrder, error) { return e.loadOrder(ctx, id) }

func newClientID() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "s510-" + hex.EncodeToString(b), nil
}
func (e *Engine) forUpdate() string {
	if e.mysql {
		return " FOR UPDATE"
	}
	return ""
}

func (e *Engine) Prepare(ctx context.Context, req OrderRequest) (LocalOrder, error) {
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
	if len(req.Symbol) == 0 || len(req.Symbol) > 32 || strings.ToUpper(req.Symbol) != req.Symbol || !signalKeyPattern.MatchString(req.SignalKey) {
		return LocalOrder{}, errors.New("invalid Spot symbol or signal key")
	}
	if req.Type != "LIMIT" && req.Type != "MARKET" {
		return LocalOrder{}, errors.New("only Spot LIMIT/MARKET supported")
	}
	if req.Side != "BUY" {
		return LocalOrder{}, errors.New("S2 only permits Mock BUY; strategy SELL ownership belongs to S3")
	}
	qty, err := parseAmount(req.Quantity, false)
	if err != nil {
		return LocalOrder{}, err
	}
	var price decimal.Decimal
	if req.Type == "LIMIT" {
		price, err = parseAmount(req.Price, false)
	} else {
		price, err = parseAmount(req.EstimatePrice, false)
	}
	if err != nil {
		return LocalOrder{}, err
	}
	spec, err := e.client.Instrument(ctx, req.Symbol)
	if err != nil {
		return LocalOrder{}, fmt.Errorf("spot exchangeInfo unavailable: %s", safeReadError(err))
	}
	if spec.Symbol != req.Symbol || spec.BaseAsset == "" || spec.QuoteAsset == "" || spec.Status != "TRADING" || !spec.SpotAllowed {
		return LocalOrder{}, errors.New("Spot exchangeInfo does not confirm trading asset/status")
	}
	min, step, max := spec.LotMin, spec.LotStep, spec.LotMax
	if req.Type == "MARKET" && spec.MarketLotStep != "" {
		marketStep, parseErr := parseAmount(spec.MarketLotStep, true)
		if parseErr != nil {
			return LocalOrder{}, errors.New("invalid MARKET_LOT_SIZE specification")
		}
		if marketStep.IsPositive() {
			min, step, max = spec.MarketLotMin, spec.MarketLotStep, spec.MarketLotMax
		}
	}
	if step == "" || step == "0" || min == "" {
		return LocalOrder{}, errors.New("Spot quantity filter missing; refuse Mock reservation")
	}
	if min != "" && min != "0" {
		v, er := parseAmount(min, false)
		if er != nil || qty.LessThan(v) {
			return LocalOrder{}, errors.New("Spot LOT_SIZE minimum invalid")
		}
	}
	if max != "" && max != "0" {
		v, er := parseAmount(max, false)
		if er != nil || qty.GreaterThan(v) {
			return LocalOrder{}, errors.New("Spot LOT_SIZE maximum invalid")
		}
	}
	if step != "" && step != "0" {
		v, er := parseAmount(step, false)
		if er != nil || !qty.Mod(v).IsZero() {
			return LocalOrder{}, errors.New("Spot LOT_SIZE step invalid")
		}
	}
	if req.Type == "LIMIT" {
		if spec.TickSize == "" || spec.TickSize == "0" {
			return LocalOrder{}, errors.New("Spot PRICE_FILTER missing")
		}
		if spec.TickSize != "" && spec.TickSize != "0" {
			v, er := parseAmount(spec.TickSize, false)
			if er != nil || !price.Mod(v).IsZero() {
				return LocalOrder{}, errors.New("Spot PRICE_FILTER tick invalid")
			}
		}
		if spec.PriceMin != "" && spec.PriceMin != "0" {
			v, er := parseAmount(spec.PriceMin, false)
			if er != nil || price.LessThan(v) {
				return LocalOrder{}, errors.New("Spot PRICE_FILTER minimum invalid")
			}
		}
		if spec.PriceMax != "" && spec.PriceMax != "0" {
			v, er := parseAmount(spec.PriceMax, false)
			if er != nil || price.GreaterThan(v) {
				return LocalOrder{}, errors.New("Spot PRICE_FILTER maximum invalid")
			}
		}
	}
	notional := qty.Mul(price)
	if spec.MinNotional == "" {
		return LocalOrder{}, errors.New("Spot notional filter missing")
	}
	if spec.MinNotional != "0" {
		v, er := parseAmount(spec.MinNotional, false)
		if er != nil || notional.LessThan(v) {
			return LocalOrder{}, errors.New("Spot MIN_NOTIONAL invalid")
		}
	}
	if spec.MaxNotional != "" && spec.MaxNotional != "0" {
		v, er := parseAmount(spec.MaxNotional, false)
		if er != nil || notional.GreaterThan(v) {
			return LocalOrder{}, errors.New("Spot NOTIONAL maximum invalid")
		}
	}
	if req.ClientOrderID == "" {
		req.ClientOrderID, err = newClientID()
		if err != nil {
			return LocalOrder{}, err
		}
	}
	if !clientIDPattern.MatchString(req.ClientOrderID) {
		return LocalOrder{}, errors.New("invalid clientOrderId")
	}
	balances, err := e.client.Balances(ctx, AccountMain)
	if err != nil {
		return LocalOrder{}, fmt.Errorf("spot balance unavailable: %s", safeReadError(err))
	}
	quote, ok := balances[spec.QuoteAsset]
	if !ok {
		return LocalOrder{}, errors.New("quote asset absent from Spot account")
	}
	free, err := parseAmount(quote.Free, true)
	if err != nil {
		return LocalOrder{}, err
	}
	if _, err = parseAmount(quote.Locked, true); err != nil {
		return LocalOrder{}, err
	}
	open, err := e.client.OpenOrders(ctx, AccountMain, req.Symbol)
	if err != nil {
		return LocalOrder{}, fmt.Errorf("spot openOrders unavailable: %s", safeReadError(err))
	}
	if len(open) > 0 {
		return LocalOrder{}, errors.New("Spot open orders may conflict with strategy, fail closed")
	}
	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return LocalOrder{}, err
	}
	defer tx.Rollback()
	// A stable, existing config row serializes quote budgets across symbols
	// on MySQL, not only concurrent first BUYs for the same symbol.
	var accountCount int64
	if err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM config").Scan(&accountCount); err != nil {
		return LocalOrder{}, err
	}
	if accountCount != 1 {
		return LocalOrder{}, errors.New("Spot config is not a unique account scope")
	}
	var accountID int64
	if err = tx.QueryRowContext(ctx, "SELECT id FROM config ORDER BY id LIMIT 1"+e.forUpdate()).Scan(&accountID); err != nil {
		return LocalOrder{}, err
	}
	var symbolID int64
	var enabled int
	var quoteType, budgetText string
	err = tx.QueryRowContext(ctx, "SELECT id,strategy_trade_enabled,type,strategy_budget_quote FROM spot_symbols WHERE symbol=?"+e.forUpdate(), req.Symbol).Scan(&symbolID, &enabled, &quoteType, &budgetText)
	if err != nil {
		return LocalOrder{}, err
	}
	if enabled != 1 || quoteType != spec.QuoteAsset {
		return LocalOrder{}, errors.New("Spot strategy switch or exchangeInfo quote mismatch")
	}
	var blocked int64
	err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM spot_strategy_orders WHERE account_id=? AND owner=? AND symbol=? AND status='reconcile_required'`,
		AccountMain, OwnerAuto, req.Symbol).Scan(&blocked)
	if err != nil {
		return LocalOrder{}, err
	}
	if blocked > 0 {
		return LocalOrder{}, ErrReconciliation
	}
	// No averaging-in/additional BUY for an already active Spot cycle.
	var existingCycle int64
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM spot_strategy_holdings
 WHERE active_slot=? AND status='active'`, AccountMain+":"+OwnerAuto+":"+req.Symbol).Scan(&existingCycle); err != nil {
		return LocalOrder{}, err
	}
	if existingCycle > 0 {
		return LocalOrder{}, ErrReconciliation
	}
	budget, err := parseAmount(budgetText, false)
	if err != nil {
		return LocalOrder{}, err
	}
	if notional.GreaterThan(budget) {
		return LocalOrder{}, errors.New("order notional above per-symbol quote budget")
	}
	var reservedText sql.NullString
	err = tx.QueryRowContext(ctx, `SELECT CAST(COALESCE(SUM(o.reserved_qty),0) AS CHAR) FROM spot_strategy_orders o
 JOIN spot_symbols s ON s.symbol=o.symbol WHERE o.account_id=? AND o.owner=? AND s.type=? AND o.side='BUY' AND o.pending_slot IS NOT NULL`,
		AccountMain, OwnerAuto, spec.QuoteAsset).Scan(&reservedText)
	if err != nil {
		return LocalOrder{}, err
	}
	reserved := decimal.Zero
	if reservedText.Valid {
		reserved, err = decimal.NewFromString(reservedText.String)
		if err != nil {
			return LocalOrder{}, err
		}
	}
	if free.LessThan(notional.Add(reserved)) {
		return LocalOrder{}, errors.New("quote free minus pending reservations insufficient")
	}
	now := time.Now().UnixMilli()
	res, err := tx.ExecContext(ctx, `INSERT INTO spot_strategy_orders
 (account_id,owner,symbol,side,order_type,intent,client_order_id,signal_key,pending_slot,
 requested_qty,reserved_qty,status,created_at,updated_at)
 VALUES (?,?,?,?,?,'open',?,?,?,?,?,'reserved',?,?)`,
		AccountMain, OwnerAuto, req.Symbol, req.Side, req.Type, req.ClientOrderID, req.SignalKey,
		AccountMain+":"+OwnerAuto+":"+req.Symbol, qty.String(), notional.String(), now, now)
	if err != nil {
		message := strings.ToLower(err.Error())
		if strings.Contains(message, "duplicate") || strings.Contains(message, "unique constraint") {
			return LocalOrder{}, ErrDuplicate
		}
		return LocalOrder{}, fmt.Errorf("persist Spot order intent: %w", err)
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

func (e *Engine) Submit(ctx context.Context, id int64, price, estimatedPrice string) (LocalOrder, error) {
	if !e.mockOnly {
		if !e.liveReady {
			return LocalOrder{}, ErrLiveWriteDisabled
		}
		if err := AuthorizeLiveMode(); err != nil {
			return LocalOrder{}, err
		}
	}
	order, err := e.loadOrder(ctx, id)
	if err != nil {
		return LocalOrder{}, err
	}
	if order.Status != StateReserved {
		return order, fmt.Errorf("order not reserved: %s", order.Status)
	}
	res, err := e.db.ExecContext(ctx, `UPDATE spot_strategy_orders SET status=?,updated_at=? WHERE id=? AND status=?`, StateSubmitting, time.Now().UnixMilli(), id, StateReserved)
	if err != nil {
		return LocalOrder{}, err
	}
	n, err := res.RowsAffected()
	if err != nil || n != 1 {
		return LocalOrder{}, errors.New("order submission already claimed")
	}
	ack, callErr := e.client.Place(ctx, OrderRequest{AccountID: order.AccountID, Owner: order.Owner, Symbol: order.Symbol,
		Side: order.Side, Type: order.Type, ClientOrderID: order.ClientOrderID, Quantity: order.RequestedQty,
		Price: price, EstimatePrice: estimatedPrice, SignalKey: order.SignalKey})
	if callErr != nil {
		e.markUnknown(ctx, id, "submit outcome unknown")
		unknown, _ := e.loadOrder(ctx, id)
		return unknown, fmt.Errorf("%w: Spot submit ambiguous, query original client ID", ErrReconciliation)
	}
	if ack.Symbol != order.Symbol || ack.ClientOrderID != order.ClientOrderID || ack.OrderID == "" || ack.Side != order.Side {
		e.markUnknown(ctx, id, "invalid submit ACK identity")
		unknown, _ := e.loadOrder(ctx, id)
		return unknown, fmt.Errorf("%w: Spot submit ACK identity mismatch", ErrReconciliation)
	}
	write, writeErr := e.db.ExecContext(ctx, `UPDATE spot_strategy_orders SET exchange_order_id=?,status=?,updated_at=? WHERE id=? AND status=?`,
		ack.OrderID, StateNew, time.Now().UnixMilli(), id, StateSubmitting)
	if writeErr != nil {
		return LocalOrder{}, writeErr
	}
	affected, rowsErr := write.RowsAffected()
	if rowsErr != nil || affected != 1 {
		// Another recovery process may have updated the same order. Do not
		// claim the ACK was persisted if the guarded transition did not apply.
		return LocalOrder{}, fmt.Errorf("%w: submit state changed before ACK persistence", ErrReconciliation)
	}
	// ACK (even FILLED status) must NOT create fills; Reconcile uses myTrades.
	return e.loadOrder(ctx, id)
}
func (e *Engine) markUnknown(ctx context.Context, id int64, reason string) {
	_, _ = e.db.ExecContext(ctx, `UPDATE spot_strategy_orders SET status=?,last_error=?,last_checked_at=?,updated_at=?
 WHERE id=? AND status NOT IN ('filled','canceled','expired','rejected','reconcile_required')`,
		StateUnknown, reason, time.Now().UnixMilli(), time.Now().UnixMilli(), id)
}

// Mark data-quality failures separately from temporary transport errors;
// both states preserve the pending slot and block new risks.
// markReconcile is a monotonic quarantine. The first transition retains
// the original cause; subsequent probes must not overwrite it. Logging
// produces a structured operational event, whereas order.status persists the
// freeze across process restarts. S2 has no automatic unfreeze endpoint.
func (e *Engine) markReconcile(ctx context.Context, id int64, reason string) {
	var before string
	if err := e.db.QueryRowContext(ctx, "SELECT status FROM spot_strategy_orders WHERE id=?", id).Scan(&before); err != nil {
		return
	}
	if before == StateReconcile {
		return
	}
	now := time.Now().UnixMilli()
	write, err := e.db.ExecContext(ctx, `UPDATE spot_strategy_orders SET status=?,last_error=?,last_checked_at=?,updated_at=?
	WHERE id=? AND status=?`, StateReconcile, reason, now, now, id, before)
	if err != nil {
		return
	}
	rows, err := write.RowsAffected()
	if err == nil && rows == 1 {
		logs.Warn("spot_strategy_order_event order_id=%d old_status=%s new_status=%s reason=%s source=auto_reconcile at_ms=%d",
			id, before, StateReconcile, reason, now)
	}
}

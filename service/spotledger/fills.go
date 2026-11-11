package spotledger

import (
	"fmt"
	"strings"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
	"github.com/shopspring/decimal"
)

var orderStatuses = map[string]bool{
	"NEW": true, "PARTIALLY_FILLED": true, "FILLED": true,
	"CANCELED": true, "EXPIRED": true, "REJECTED": true,
	"PENDING_CANCEL": true,
}

func terminalStatus(s string) bool {
	return s == "FILLED" || s == "CANCELED" || s == "EXPIRED" || s == "REJECTED"
}

// BindExchangeOrderID associates an existing local intent with the order
// returned by Binance, but never creates or submits one.
func (l *Ledger) BindExchangeOrderID(orderID int64, exchangeID string) (models.SpotManagedOrder, error) {
	var empty models.SpotManagedOrder
	if !exchangeIDPattern.MatchString(exchangeID) {
		return empty, fmt.Errorf("invalid exchange order id")
	}
	tx, unlock, err := l.begin()
	if err != nil {
		return empty, err
	}
	defer unlock()
	defer tx.Rollback()
	first, err := readOrder(tx, orderID, "")
	if err != nil {
		return empty, err
	}
	if _, err = readPosition(tx, first.PositionID, l.lock()); err != nil {
		return empty, err
	}
	order, err := readOrder(tx, orderID, l.lock())
	if err != nil {
		return empty, err
	}
	if order.ExchangeOrderID != "" && order.ExchangeOrderID != exchangeID {
		return empty, fmt.Errorf("exchange order id cannot be rebound")
	}
	if order.ExchangeOrderID == "" {
		_, err = tx.Raw("UPDATE spot_managed_orders SET exchange_order_id=?, updated_at=? WHERE id=? AND exchange_order_id IS NULL", exchangeID, now(), orderID).Exec()
		if err != nil {
			return empty, err
		}
	}
	order, err = readOrder(tx, orderID, "")
	if err != nil {
		return empty, err
	}
	if err = tx.Commit(); err != nil {
		return empty, err
	}
	return order, nil
}

func transitionAllowed(before, after string) bool {
	if !orderStatuses[after] {
		return false
	}
	if before == after {
		return true
	}
	if terminalStatus(before) {
		return false
	}
	switch before {
	case "NEW":
		return true
	case "PARTIALLY_FILLED":
		return after == "FILLED" || after == "CANCELED" || after == "EXPIRED" || after == "PENDING_CANCEL"
	case "PENDING_CANCEL":
		return after == "PARTIALLY_FILLED" || after == "FILLED" || after == "CANCELED" || after == "EXPIRED"
	}
	return false
}

// SetExchangeStatus persists only a legal exchange status transition. It does
// not fake any fill quantity: exchange order statuses can precede myTrades.
func (l *Ledger) SetExchangeStatus(orderID int64, status string) (models.SpotManagedOrder, error) {
	var empty models.SpotManagedOrder
	if !orderStatuses[status] {
		return empty, fmt.Errorf("unsupported Spot exchange status %q", status)
	}
	tx, unlock, err := l.begin()
	if err != nil {
		return empty, err
	}
	defer unlock()
	defer tx.Rollback()
	seen, err := readOrder(tx, orderID, "")
	if err != nil {
		return empty, err
	}
	if _, err = readPosition(tx, seen.PositionID, l.lock()); err != nil {
		return empty, err
	}
	order, err := readOrder(tx, orderID, l.lock())
	if err != nil {
		return empty, err
	}
	if !transitionAllowed(order.Status, status) {
		return empty, fmt.Errorf("illegal Spot order status %s -> %s", order.Status, status)
	}
	if order.Status != status {
		_, err = tx.Raw("UPDATE spot_managed_orders SET status=?,updated_at=? WHERE id=?", status, now(), orderID).Exec()
		if err != nil {
			return empty, err
		}
	}
	order, err = readOrder(tx, orderID, "")
	if err != nil {
		return empty, err
	}
	if err = tx.Commit(); err != nil {
		return empty, err
	}
	return order, nil
}

func validateFillEvent(e FillEvent) error {
	if e.OrderID <= 0 || !assetPattern.MatchString(e.Symbol) || !exchangeIDPattern.MatchString(e.ExchangeOrderID) ||
		!exchangeIDPattern.MatchString(e.TradeID) || (e.Side != "BUY" && e.Side != "SELL") || e.TradeTime <= 0 {
		return fmt.Errorf("invalid Spot fill identity")
	}
	if _, err := parseDBDecimal(e.Qty, true); err != nil {
		return fmt.Errorf("fill quantity: %w", err)
	}
	if _, err := parseDBDecimal(e.Price, true); err != nil {
		return fmt.Errorf("fill price: %w", err)
	}
	quote, err := parseDBDecimal(e.QuoteQty, true)
	if err != nil {
		return fmt.Errorf("fill quote quantity: %w", err)
	}
	qty, _ := parseDBDecimal(e.Qty, true)
	price, _ := parseDBDecimal(e.Price, true)
	// Binance Spot trade quoteQty is the executed quote amount, not the
	// budget or commission. Allow a tiny exchange rounding difference but
	// never accept a mismatched quote currency/cost basis as a valid fill.
	if quote.Sub(qty.Mul(price)).Abs().GreaterThan(decimal.RequireFromString("0.000001")) {
		return fmt.Errorf("fill quote quantity does not match qty * price")
	}
	if strings.TrimSpace(e.Commission) == "" {
		return fmt.Errorf("fill commission must be explicit, even when zero")
	}
	fee, err := parseDBDecimal(e.Commission, false)
	if err != nil {
		return fmt.Errorf("fill commission: %w", err)
	}
	if fee.IsPositive() && !assetPattern.MatchString(e.CommissionAsset) {
		return fmt.Errorf("nonzero Spot fee requires valid commission asset")
	}
	if e.CommissionAsset != "" && !assetPattern.MatchString(e.CommissionAsset) {
		return fmt.Errorf("invalid Spot commission asset")
	}
	return nil
}

func getPositionFills(q orm.QueryExecutor, positionID int64) ([]models.SpotTradeFill, error) {
	rows := make([]models.SpotTradeFill, 0)
	_, err := q.Raw("SELECT id,position_id,order_id,exchange_order_id,trade_id,symbol,side,qty,price,quote_qty,commission,commission_asset,trade_time FROM spot_trade_fills WHERE position_id=? ORDER BY trade_time ASC,id ASC", positionID).QueryRows(&rows)
	return rows, err
}

func getPositionOrders(q orm.QueryExecutor, positionID int64) ([]models.SpotManagedOrder, error) {
	rows := make([]models.SpotManagedOrder, 0)
	_, err := q.Raw("SELECT id,position_id,symbol,side,order_type,client_order_id,COALESCE(exchange_order_id,'') AS exchange_order_id,requested_qty,filled_qty,status,created_at,updated_at FROM spot_managed_orders WHERE position_id=? ORDER BY id", positionID).QueryRows(&rows)
	return rows, err
}

// positionProjection is computed exclusively from immutable, confirmed Spot
// trades. It never trusts the mutable quantities and PnL in position rows.
// Keep this one calculator for both writes and read-only Accounting.
type positionProjection struct {
	position     models.SpotManagedPosition
	orders       []models.SpotManagedOrder
	fills        []models.SpotTradeFill
	orderFillQty map[int64]decimal.Decimal
	orderStatus  map[int64]string
	reviewIssues []string
}

func projectPosition(q orm.QueryExecutor, p models.SpotManagedPosition) (positionProjection, error) {
	proj := positionProjection{position: p}
	fills, err := getPositionFills(q, p.ID)
	if err != nil {
		return proj, err
	}
	orders, err := getPositionOrders(q, p.ID)
	if err != nil {
		return proj, err
	}
	proj.fills = fills
	proj.orders = orders
	proj.orderFillQty = make(map[int64]decimal.Decimal, len(orders))
	proj.orderStatus = make(map[int64]string, len(orders))
	byID := make(map[int64]models.SpotManagedOrder, len(orders))
	for _, o := range orders {
		if !orderStatuses[o.Status] {
			return proj, fmt.Errorf("%w: order %d has unsupported status %q", ErrNeedsReview, o.ID, o.Status)
		}
		byID[o.ID] = o
		proj.orderFillQty[o.ID] = zero
	}
	bought, remaining, cost, realized := zero, zero, zero, zero
	closedAt := int64(0)
	for _, f := range fills {
		o, exists := byID[f.OrderID]
		if !exists || o.PositionID != p.ID || o.Symbol != p.Symbol || o.Side != f.Side ||
			f.PositionID != p.ID || f.Symbol != p.Symbol || o.ExchangeOrderID != f.ExchangeOrderID {
			return proj, fmt.Errorf("%w: foreign or mismatched Spot fill %d (position/order identity)", ErrNeedsReview, f.ID)
		}
		qty, err := parseDBDecimal(f.Qty, true)
		if err != nil {
			return proj, err
		}
		quote, err := parseDBDecimal(f.QuoteQty, true)
		if err != nil {
			return proj, err
		}
		price, err := parseDBDecimal(f.Price, true)
		if err != nil {
			return proj, err
		}
		if quote.Sub(qty.Mul(price)).Abs().GreaterThan(decimal.RequireFromString("0.000001")) {
			return proj, fmt.Errorf("%w: stored fill %d has mismatched quote quantity", ErrNeedsReview, f.ID)
		}
		commission, err := parseDBDecimal(f.Commission, false)
		if err != nil {
			return proj, err
		}
		if f.CommissionAsset != "" && !assetPattern.MatchString(f.CommissionAsset) {
			return proj, fmt.Errorf("%w: invalid stored fee asset for fill %d", ErrNeedsReview, f.ID)
		}
		if commission.IsPositive() && f.CommissionAsset == "" {
			return proj, fmt.Errorf("%w: stored nonzero commission missing asset for fill %d", ErrNeedsReview, f.ID)
		}
		proj.orderFillQty[o.ID] = proj.orderFillQty[o.ID].Add(qty)
		if f.Side == "BUY" {
			baseReceived := qty
			quotePaid := quote
			if f.CommissionAsset == p.BaseAsset {
				baseReceived = baseReceived.Sub(commission)
			}
			if f.CommissionAsset == p.QuoteAsset {
				quotePaid = quotePaid.Add(commission)
			}
			if !baseReceived.IsPositive() {
				return proj, fmt.Errorf("%w: BUY fill %d has zero/negative net strategy inventory", ErrNeedsReview, f.ID)
			}
			bought = bought.Add(baseReceived)
			remaining = remaining.Add(baseReceived)
			cost = cost.Add(quotePaid)
		} else {
			if !remaining.IsPositive() {
				return proj, fmt.Errorf("%w: SELL fill %d without a strategy BUY", ErrNeedsReview, f.ID)
			}
			debit := qty
			quoteReceived := quote
			if f.CommissionAsset == p.BaseAsset {
				debit = debit.Add(commission)
			}
			if f.CommissionAsset == p.QuoteAsset {
				quoteReceived = quoteReceived.Sub(commission)
			}
			if debit.GreaterThan(remaining) {
				return proj, fmt.Errorf("%w: SELL fill %d exceeds strategy-owned inventory", ErrNeedsReview, f.ID)
			}
			allocated := cost
			if debit.LessThan(remaining) {
				allocated = cost.Mul(debit).DivRound(remaining, scale)
			}
			realized = realized.Add(quoteReceived.Sub(allocated))
			cost = cost.Sub(allocated)
			remaining = remaining.Sub(debit)
			if remaining.IsZero() {
				closedAt = f.TradeTime
				cost = zero
			}
		}
	}
	for _, o := range orders {
		filled := proj.orderFillQty[o.ID]
		requested, err := parseDBDecimal(o.RequestedQty, true)
		if err != nil {
			return proj, err
		}
		if filled.GreaterThan(requested) {
			return proj, fmt.Errorf("%w: order %d fill qty exceeds requested qty; reconcile exchange order first", ErrNeedsReview, o.ID)
		}
		status := o.Status
		if !terminalStatus(status) {
			if filled.Equal(requested) && filled.IsPositive() {
				status = "FILLED"
			} else if filled.IsPositive() && (status == "NEW" || status == "PARTIALLY_FILLED") {
				status = "PARTIALLY_FILLED"
			}
		}
		// Terminal statuses reflect exchange reports, not synthetic Fill
		// arithmetic. Contradictory terminal evidence requires review, not
		// silent rewriting of CANCELED/REJECTED to FILLED.
		if (status == "CANCELED" || status == "EXPIRED" || status == "REJECTED") &&
			((filled.IsPositive() && status == "REJECTED") || (filled.IsPositive() && filled.Equal(requested))) {
			proj.reviewIssues = append(proj.reviewIssues, fmt.Sprintf("order %d is %s but verified fills total %s of %s", o.ID, status, filled, requested))
		}
		if status == "FILLED" && !filled.Equal(requested) {
			proj.reviewIssues = append(proj.reviewIssues, fmt.Sprintf("order %d is FILLED but only %s of %s confirmed in myTrades", o.ID, filled, requested))
		}
		proj.orderStatus[o.ID] = status
	}
	status := "PENDING_BUY"
	avg := zero
	if bought.IsPositive() {
		if remaining.IsPositive() {
			status = "OPEN"
			avg = cost.DivRound(remaining, scale)
			closedAt = 0
		} else {
			status = "CLOSED"
		}
	}
	proj.position.BoughtQty, err = numeric(bought)
	if err != nil {
		return proj, err
	}
	proj.position.RemainingQty, err = numeric(remaining)
	if err != nil {
		return proj, err
	}
	proj.position.AvgCostQuote, err = numeric(avg)
	if err != nil {
		return proj, err
	}
	proj.position.RealizedPnL, err = numeric(realized)
	if err != nil {
		return proj, err
	}
	proj.position.Status = status
	proj.position.ClosedAt = closedAt
	return proj, nil
}

func writeProjection(tx orm.TxOrmer, proj positionProjection) (models.SpotManagedPosition, error) {
	p := proj.position
	for _, o := range proj.orders {
		filledText, err := numeric(proj.orderFillQty[o.ID])
		if err != nil {
			return p, err
		}
		_, err = tx.Raw("UPDATE spot_managed_orders SET filled_qty=?,status=?,updated_at=? WHERE id=?", filledText, proj.orderStatus[o.ID], now(), o.ID).Exec()
		if err != nil {
			return p, err
		}
	}
	_, err := tx.Raw("UPDATE spot_managed_positions SET bought_qty=?,remaining_qty=?,avg_cost_quote=?,realized_pnl=?,status=?,updated_at=?,closed_at=? WHERE id=?", p.BoughtQty, p.RemainingQty, p.AvgCostQuote, p.RealizedPnL, p.Status, now(), p.ClosedAt, p.ID).Exec()
	if err != nil {
		return p, err
	}
	return readPosition(tx, p.ID, "")
}

// reprojectPosition writes values calculated from immutable Fill rows.
// It may correct corrupt materialized data but never upgrades an exchange
// terminal status merely because recorded Fill quantity reaches requested.
func reprojectPosition(tx orm.TxOrmer, p models.SpotManagedPosition) (models.SpotManagedPosition, error) {
	proj, err := projectPosition(tx, p)
	if err != nil {
		return p, err
	}
	return writeProjection(tx, proj)
}

// ReviewPosition detects contradictory exchange status and missing myTrades
// without inventing persistent status or modifying any Binance order.
func (l *Ledger) ReviewPosition(id int64) ([]string, error) {
	tx, unlock, err := l.begin()
	if err != nil {
		return nil, err
	}
	defer unlock()
	defer tx.Rollback()
	p, err := readPosition(tx, id, l.lock())
	if err != nil {
		return nil, err
	}
	proj, err := projectPosition(tx, p)
	if err != nil {
		return nil, err
	}
	return proj.reviewIssues, nil
}

func sameFill(existing models.SpotTradeFill, e FillEvent) bool {
	if existing.OrderID != e.OrderID || existing.Symbol != e.Symbol || existing.ExchangeOrderID != e.ExchangeOrderID ||
		existing.TradeID != e.TradeID || existing.Side != e.Side || existing.TradeTime != e.TradeTime ||
		existing.CommissionAsset != e.CommissionAsset {
		return false
	}
	for _, pair := range [][2]string{
		{existing.Qty, e.Qty}, {existing.Price, e.Price},
		{existing.QuoteQty, e.QuoteQty}, {existing.Commission, e.Commission},
	} {
		left, err := parseDBDecimal(pair[0], false)
		if err != nil {
			return false
		}
		right, err := parseDBDecimal(pair[1], false)
		if err != nil || !left.Equal(right) {
			return false
		}
	}
	return true
}

// RecordFill persists a confirmed exchange trade, not a balance delta.
// Identical exchange trade IDs are idempotent. Different content reusing
// an existing identity is an error, not a silent overwrite.
func (l *Ledger) RecordFill(e FillEvent) (FillResult, error) {
	var result FillResult
	if err := validateFillEvent(e); err != nil {
		return result, err
	}
	tx, unlock, err := l.begin()
	if err != nil {
		return result, err
	}
	defer unlock()
	defer tx.Rollback()
	seen, err := readOrder(tx, e.OrderID, "")
	if err != nil {
		return result, err
	}
	p, err := readPosition(tx, seen.PositionID, l.lock())
	if err != nil {
		return result, err
	}
	o, err := readOrder(tx, e.OrderID, l.lock())
	if err != nil {
		return result, err
	}
	if o.Symbol != e.Symbol || o.Side != e.Side || p.Symbol != e.Symbol {
		return result, fmt.Errorf("Spot fill symbol/side mismatch with strategy-owned order")
	}
	if p.Status == "CLOSED" && e.TradeTime > p.ClosedAt {
		return result, fmt.Errorf("cannot append a new trade to already-closed Spot position")
	}
	if o.ExchangeOrderID != "" && o.ExchangeOrderID != e.ExchangeOrderID {
		return result, fmt.Errorf("Spot fill exchange order id mismatch")
	}
	// Bind first, including the symbol+exchange-order uniqueness constraint.
	if o.ExchangeOrderID == "" {
		if _, err = tx.Raw("UPDATE spot_managed_orders SET exchange_order_id=?,updated_at=? WHERE id=? AND exchange_order_id IS NULL", e.ExchangeOrderID, now(), o.ID).Exec(); err != nil {
			return result, err
		}
		o.ExchangeOrderID = e.ExchangeOrderID
	}
	var duplicates []models.SpotTradeFill
	_, err = tx.Raw("SELECT id,position_id,order_id,exchange_order_id,trade_id,symbol,side,qty,price,quote_qty,commission,commission_asset,trade_time FROM spot_trade_fills WHERE symbol=? AND exchange_order_id=? AND trade_id=?", e.Symbol, e.ExchangeOrderID, e.TradeID).QueryRows(&duplicates)
	if err != nil {
		return result, err
	}
	if len(duplicates) > 0 {
		if len(duplicates) != 1 || !sameFill(duplicates[0], e) {
			return result, fmt.Errorf("conflicting payload for existing Spot exchange trade id %s", e.TradeID)
		}
		if err = tx.Commit(); err != nil {
			return result, err
		}
		result.Order = o
		result.Position = p
		return result, nil // no write, no PnL change
	}
	qty, _ := parseDBDecimal(e.Qty, true)
	price, _ := parseDBDecimal(e.Price, true)
	quote, _ := parseDBDecimal(e.QuoteQty, true)
	commission, _ := parseDBDecimal(e.Commission, false)
	qtyText, _ := numeric(qty)
	priceText, _ := numeric(price)
	quoteText, _ := numeric(quote)
	feeText, _ := numeric(commission)
	if _, err = tx.Raw("INSERT INTO spot_trade_fills(position_id,order_id,exchange_order_id,trade_id,symbol,side,qty,price,quote_qty,commission,commission_asset,trade_time) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)", p.ID, o.ID, e.ExchangeOrderID, e.TradeID, e.Symbol, e.Side, qtyText, priceText, quoteText, feeText, e.CommissionAsset, e.TradeTime).Exec(); err != nil {
		return result, err
	}
	p, err = reprojectPosition(tx, p)
	if err != nil {
		return result, err
	}
	o, err = readOrder(tx, o.ID, "")
	if err != nil {
		return result, err
	}
	if err = tx.Commit(); err != nil {
		return result, err
	}
	result.Applied = true
	result.Position = p
	result.Order = o
	return result, nil
}

// RebuildPosition can be called after process restart. It does not need
// Binance wallet holdings or ephemeral in-memory state.
func (l *Ledger) RebuildPosition(positionID int64) (models.SpotManagedPosition, error) {
	var empty models.SpotManagedPosition
	tx, unlock, err := l.begin()
	if err != nil {
		return empty, err
	}
	defer unlock()
	defer tx.Rollback()
	p, err := readPosition(tx, positionID, l.lock())
	if err != nil {
		return empty, err
	}
	p, err = reprojectPosition(tx, p)
	if err != nil {
		return empty, err
	}
	if err = tx.Commit(); err != nil {
		return empty, err
	}
	return p, nil
}

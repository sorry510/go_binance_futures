package spotstrategy

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strings"
	"time"
)

var ErrInvalidS4Filter = errors.New("invalid Spot strategy read filter")

const s4MaxLimit = 50

// Single-character fragments are valid for symbol list search. This
// alphanumeric filter also prohibits LIKE wildcards and SQL metacharacters.
var s4SymbolPattern = regexp.MustCompile(`^[A-Z0-9]{1,32}$`)

type PageRequest struct {
	Symbol     string
	Status     string
	QuoteAsset string // Config list only: empty, USDT, USDC or FDUSD
	Page       int
	Limit      int
}
type PageResult[T any] struct {
	Items []T   `json:"items"`
	Total int64 `json:"total"`
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
}
type OrderView struct {
	ID               int64  `json:"id"`
	HoldingID        *int64 `json:"holding_id"`
	Symbol           string `json:"symbol"`
	Side             string `json:"side"`
	Type             string `json:"order_type"`
	Intent           string `json:"intent"`
	ClientOrderID    string `json:"client_order_id"`
	ExchangeOrderID  string `json:"exchange_order_id"`
	RequestedQty     string `json:"requested_qty"`
	FilledQty        string `json:"filled_qty"`
	ReservedQty      string `json:"reserved_qty"`
	ExecutedQuoteQty string `json:"executed_quote_qty"`
	RemainingQty     string `json:"remaining_qty"`
	Status           string `json:"status"`
	// ErrorCategory is deliberately a safe status code, never raw REST errors.
	ErrorCategory string `json:"error_category,omitempty"`
	CreatedAt     int64  `json:"created_at"`
	UpdatedAt     int64  `json:"updated_at"`
	LastCheckedAt int64  `json:"last_checked_at"`
}
type FillView struct {
	Symbol          string `json:"symbol"`
	OrderID         string `json:"order_id"`
	TradeID         string `json:"trade_id"`
	Side            string `json:"side"`
	Qty             string `json:"qty"`
	Price           string `json:"price"`
	QuoteQty        string `json:"quote_qty"`
	Commission      string `json:"commission"`
	CommissionAsset string `json:"commission_asset"`
	Time            int64  `json:"trade_time"`
	RESTVerified    bool   `json:"rest_verified"`
}

func normalizedPage(req PageRequest, allowedStatus map[string]bool) (PageRequest, error) {
	if req.Page == 0 {
		req.Page = 1
	}
	if req.Limit == 0 {
		req.Limit = 20
	}
	if req.Page < 1 || req.Page > 100000 || req.Limit < 1 || req.Limit > s4MaxLimit {
		return req, ErrInvalidS4Filter
	}
	if req.Symbol != "" && !s4SymbolPattern.MatchString(req.Symbol) {
		return req, ErrInvalidS4Filter
	}
	if req.Status != "" && !allowedStatus[req.Status] {
		return req, ErrInvalidS4Filter
	}
	return req, nil
}

var holdingStatusFilter = map[string]bool{"active": true, "closed": true}
var orderStatusFilter = map[string]bool{
	StateReserved: true, StateSubmitting: true, StateUnknown: true,
	StateNew: true, StatePartial: true, StateFilled: true,
	StateCanceled: true, StateExpired: true, StateRejected: true,
	StateReconcile: true,
}

func calcPageOffset(req PageRequest) int { return (req.Page - 1) * req.Limit }
func sanitizedOrderReason(status string) string {
	switch status {
	case StateUnknown:
		return "order_outcome_unknown"
	case StateReconcile:
		return "reconciliation_required"
	case StateReserved, StateSubmitting:
		return "order_pending"
	default:
		return ""
	}
}

// ListSettings mirrors the Futures symbol list: symbol configuration comes
// only from spot_symbols, never from an exchange account or runtime mode.
// JWT authorization remains at the controller boundary.
func ListSettings(ctx context.Context, db *sql.DB, req PageRequest) (PageResult[Setting], error) {
	out := PageResult[Setting]{Items: []Setting{}}
	// Fuzzy matching applies only to the Spot symbol list. Financial
	// Holding/Order/Fill queries retain exact symbol SQL predicates.
	req.Symbol = strings.ToUpper(strings.TrimSpace(req.Symbol))
	req, err := normalizedPage(req, map[string]bool{"enabled": true, "disabled": true})
	if err != nil {
		return out, err
	}
	// The quote filter must be part of the database COUNT and LIMIT query,
	// not a frontend-only filter after pagination.
	switch req.QuoteAsset {
	case "", "USDT", "USDC", "FDUSD":
	default:
		return out, ErrInvalidS4Filter
	}
	// Only list symbols UpdateSetting actually permits editing. No legacy
	// unsupported quote pair is presented as an editable strategy.
	where := " FROM spot_symbols WHERE ((type='USDT' AND symbol LIKE '%USDT') OR (type='USDC' AND symbol LIKE '%USDC') OR (type='FDUSD' AND symbol LIKE '%FDUSD'))"
	args := []any{}
	if req.QuoteAsset != "" {
		where += " AND type=?"
		args = append(args, req.QuoteAsset)
	}
	if req.Symbol != "" {
		where += " AND symbol LIKE ?"
		args = append(args, "%"+req.Symbol+"%")
	}
	if req.Status != "" {
		enabled := 0
		if req.Status == "enabled" {
			enabled = 1
		}
		where += " AND strategy_trade_enabled=?"
		args = append(args, enabled)
	}
	if err = db.QueryRowContext(ctx, "SELECT COUNT(*)"+where, args...).Scan(&out.Total); err != nil {
		return out, err
	}
	rows, err := db.QueryContext(ctx, `SELECT symbol,type,strategy_trade_enabled,
  COALESCE(strategy_budget_quote,'0'),COALESCE(strategy,''),COALESCE(technology,'{}'),
  COALESCE(close,''),COALESCE(lastUpdateTime,0)`+
		where+" ORDER BY symbol LIMIT ? OFFSET ?", append(append([]any{}, args...), req.Limit, calcPageOffset(req))...)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var v Setting
		var enabled int
		if err = rows.Scan(&v.Symbol, &v.QuoteAsset, &enabled, &v.BudgetQuote, &v.Strategy, &v.Technology, &v.LastPrice, &v.LastPriceAt); err != nil {
			return out, err
		}
		v.Enabled = enabled == 1
		out.Items = append(out.Items, v)
	}
	if err = rows.Err(); err != nil {
		return out, err
	}
	out.Page = req.Page
	out.Limit = req.Limit
	return out, nil
}

// ListHoldings reads ONLY the built-in main auto_strategy inventory. The SQL
// ownership restriction is not a caller-selectable argument.
func ListHoldings(ctx context.Context, db *sql.DB, req PageRequest) (PageResult[HoldingSummary], error) {
	out := PageResult[HoldingSummary]{Items: []HoldingSummary{}}
	req, err := normalizedPage(req, holdingStatusFilter)
	if err != nil {
		return out, err
	}
	where := " FROM spot_strategy_holdings WHERE account_id=? AND owner=?"
	args := []any{AccountMain, OwnerAuto}
	if req.Symbol != "" {
		where += " AND symbol=?"
		args = append(args, req.Symbol)
	}
	if req.Status != "" {
		where += " AND status=?"
		args = append(args, req.Status)
	}
	if err = db.QueryRowContext(ctx, "SELECT COUNT(*)"+where, args...).Scan(&out.Total); err != nil {
		return out, err
	}
	rows, err := db.QueryContext(ctx, "SELECT id"+where+" ORDER BY id DESC LIMIT ? OFFSET ?",
		append(append([]any{}, args...), req.Limit, calcPageOffset(req))...)
	if err != nil {
		return out, err
	}
	ids := []int64{}
	for rows.Next() {
		var id int64
		if err = rows.Scan(&id); err != nil {
			break
		}
		ids = append(ids, id)
	}
	// rows.Err reports streaming driver failures not surfaced by Scan or Close.
	if scanErr := rows.Err(); err == nil {
		err = scanErr
	}
	if closeErr := rows.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return out, err
	}
	engine := NewReadOnlyEngine(db, nil)
	for _, id := range ids {
		// Only a fresh, locally observed Spot close can be a reference
		// valuation. It is not an exchange-executable price and is never used
		// for placing orders. Missing/stale/mismatched quote data stays empty.
		var symbol, quoteAsset, markPrice string
		quoteErr := db.QueryRowContext(ctx, `SELECT symbol,quote_asset FROM spot_strategy_holdings
 WHERE id=? AND account_id=? AND owner=?`, id, AccountMain, OwnerAuto).Scan(&symbol, &quoteAsset)
		if quoteErr != nil {
			return out, quoteErr
		}
		var marketQuote string
		var lastTime int64
		lookupErr := db.QueryRowContext(ctx, `SELECT COALESCE(type,''),COALESCE(close,''),
 COALESCE(lastUpdateTime,0) FROM spot_symbols WHERE symbol=?`, symbol).
			Scan(&marketQuote, &markPrice, &lastTime)
		fresh := false
		if lookupErr == nil && marketQuote == quoteAsset && lastTime > 0 {
			age := time.Now().UnixMilli() - lastTime
			if age >= 0 && age <= 60000 {
				_, priceErr := parseAmount(markPrice, false)
				fresh = priceErr == nil
			}
		} else if lookupErr != nil && !errors.Is(lookupErr, sql.ErrNoRows) {
			return out, lookupErr
		}
		priceForReport := ""
		if fresh {
			priceForReport = markPrice
		}
		item, readErr := engine.HoldingReport(ctx, id, priceForReport)
		if readErr != nil {
			if errors.Is(readErr, ErrUnmanagedOrder) {
				return out, ErrUnmanagedOrder
			}
			h, loadErr := readHolding(db.QueryRowContext(ctx, "SELECT "+holdingCols+
				" FROM spot_strategy_holdings WHERE id=? AND account_id=? AND owner=?", id, AccountMain, OwnerAuto))
			if loadErr != nil {
				return out, loadErr
			}
			// Never show stale Holding numbers when Fill aggregation fails.
			h.OwnedQty = ""
			h.AvgCostQuote = ""
			item = HoldingSummary{Holding: h, Fees: map[string]string{}, IncompleteReason: "fill_evidence_unavailable"}
		}
		if fresh {
			item.ReferencePriceQuote = markPrice
			item.ReferencePriceTime = lastTime
			item.ReferencePriceFresh = true
		}
		out.Items = append(out.Items, item)
	}
	out.Page = req.Page
	out.Limit = req.Limit
	return out, nil
}

// ListOrders always scopes to local strategy-owned orders. A holding ID is
// checked against that ownership before any child orders are returned.
func ListOrders(ctx context.Context, db *sql.DB, holdingID int64, req PageRequest) (PageResult[OrderView], error) {
	out := PageResult[OrderView]{Items: []OrderView{}}
	req, err := normalizedPage(req, orderStatusFilter)
	if err != nil {
		return out, err
	}
	if holdingID < 0 {
		return out, ErrUnmanagedOrder
	}
	if holdingID > 0 {
		var scope int
		err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM spot_strategy_holdings
 WHERE id=? AND account_id=? AND owner=?`, holdingID, AccountMain, OwnerAuto).Scan(&scope)
		if err != nil || scope != 1 {
			return out, ErrUnmanagedOrder
		}
	}
	where := " FROM spot_strategy_orders WHERE account_id=? AND owner=?"
	args := []any{AccountMain, OwnerAuto}
	if holdingID > 0 {
		where += " AND holding_id=?"
		args = append(args, holdingID)
	}
	if req.Symbol != "" {
		where += " AND symbol=?"
		args = append(args, req.Symbol)
	}
	if req.Status != "" {
		where += " AND status=?"
		args = append(args, req.Status)
	}
	if err = db.QueryRowContext(ctx, "SELECT COUNT(*)"+where, args...).Scan(&out.Total); err != nil {
		return out, err
	}
	rows, err := db.QueryContext(ctx, `SELECT id,holding_id,symbol,side,order_type,intent,
  client_order_id,COALESCE(exchange_order_id,''),CAST(requested_qty AS CHAR),
  CAST(filled_qty AS CHAR),CAST(reserved_qty AS CHAR),
  CAST(executed_quote_qty AS CHAR),status,created_at,updated_at,last_checked_at`+
		where+" ORDER BY id DESC LIMIT ? OFFSET ?", append(append([]any{}, args...), req.Limit, calcPageOffset(req))...)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		v := OrderView{}
		var hid sql.NullInt64
		if err = rows.Scan(&v.ID, &hid, &v.Symbol, &v.Side, &v.Type, &v.Intent,
			&v.ClientOrderID, &v.ExchangeOrderID, &v.RequestedQty, &v.FilledQty,
			&v.ReservedQty, &v.ExecutedQuoteQty, &v.Status, &v.CreatedAt, &v.UpdatedAt,
			&v.LastCheckedAt); err != nil {
			return out, err
		}
		if hid.Valid {
			v.HoldingID = &hid.Int64
		}
		// BUY reserved_qty is QUOTE notional, whereas SELL reserves BASE
		// inventory. Neither value is an appropriate "unfilled base qty".
		// Always calculate the remaining order quantity from requested minus
		// exchange-confirmed fills, using exact decimal arithmetic.
		requested, reqErr := parseAmount(v.RequestedQty, false)
		filled, fillErr := parseAmount(v.FilledQty, true)
		if reqErr != nil || fillErr != nil || filled.GreaterThan(requested) {
			v.RemainingQty = ""
			v.ErrorCategory = "reconciliation_required"
		} else {
			v.RemainingQty = requested.Sub(filled).String()
			v.ErrorCategory = sanitizedOrderReason(v.Status)
		}
		out.Items = append(out.Items, v)
	}
	if err = rows.Err(); err != nil {
		return out, err
	}
	out.Page = req.Page
	out.Limit = req.Limit
	return out, nil
}

// ListOrderFills enforces main/auto_strategy internally. It must be
// called with a local owned order ID; passing an
// exchange orderId or an arbitrary symbol cannot fetch someone else's fills.
func ListOrderFills(ctx context.Context, db *sql.DB, localOrderID int64, req PageRequest) (PageResult[FillView], error) {
	out := PageResult[FillView]{Items: []FillView{}}
	req, err := normalizedPage(req, map[string]bool{"verified": true, "unverified": true})
	if err != nil {
		return out, err
	}
	if localOrderID <= 0 {
		return out, ErrUnmanagedOrder
	}
	var symbol, orderID, clientID string
	err = db.QueryRowContext(ctx, `SELECT symbol,COALESCE(exchange_order_id,''),client_order_id
 FROM spot_strategy_orders WHERE id=? AND account_id=? AND owner=?`,
		localOrderID, AccountMain, OwnerAuto).Scan(&symbol, &orderID, &clientID)
	if errors.Is(err, sql.ErrNoRows) {
		return out, ErrUnmanagedOrder
	}
	if err != nil {
		return out, err
	}
	// The source of truth is the local order ID. A supplied symbol acts as
	// an additional filter; it must never be silently ignored.
	if req.Symbol != "" && req.Symbol != symbol {
		out.Page = req.Page
		out.Limit = req.Limit
		return out, nil
	}
	where := " FROM spot_strategy_fills WHERE account_id=? AND symbol=? AND order_id=? AND client_order_id=?"
	args := []any{AccountMain, symbol, orderID, clientID}
	if req.Status != "" {
		verified := 0
		if req.Status == "verified" {
			verified = 1
		}
		where += " AND rest_verified=?"
		args = append(args, verified)
	}
	if err = db.QueryRowContext(ctx, "SELECT COUNT(*)"+where, args...).Scan(&out.Total); err != nil {
		return out, err
	}
	rows, err := db.QueryContext(ctx, `SELECT symbol,order_id,trade_id,side,CAST(qty AS CHAR),
 CAST(price AS CHAR),CAST(quote_qty AS CHAR),CAST(commission AS CHAR),
 commission_asset,trade_time,rest_verified`+
		where+" ORDER BY trade_time,trade_id LIMIT ? OFFSET ?",
		append(append([]any{}, args...), req.Limit, calcPageOffset(req))...)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		v := FillView{}
		var verified int
		if err = rows.Scan(&v.Symbol, &v.OrderID, &v.TradeID, &v.Side, &v.Qty, &v.Price,
			&v.QuoteQty, &v.Commission, &v.CommissionAsset, &v.Time, &verified); err != nil {
			return out, err
		}
		v.RESTVerified = verified == 1
		out.Items = append(out.Items, v)
	}
	if err = rows.Err(); err != nil {
		return out, err
	}
	out.Page = req.Page
	out.Limit = req.Limit
	return out, nil
}

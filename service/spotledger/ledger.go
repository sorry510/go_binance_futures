package spotledger

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
	"github.com/shopspring/decimal"
)

type Ledger struct {
	db      orm.Ormer
	dialect string
}

// SQLite needs a process-local writer mutex for deterministic test writes.
// MySQL/Postgres use SELECT FOR UPDATE on the relevant position.
var sqliteWrites sync.Mutex

// ErrNeedsReview is returned for inconsistent exchange/Fill evidence. A
// 06-4 reconciler must query Binance order/myTrades and stop automated
// execution; do not infer a manual correction or silently widen orders.
var ErrNeedsReview = errors.New("spot order needs reconciliation")

func New(db orm.Ormer, dialect string) (*Ledger, error) {
	switch dialect {
	case "mysql", "postgres", "sqlite":
		if db == nil {
			return nil, fmt.Errorf("nil Spot ledger DB")
		}
		return &Ledger{db: db, dialect: dialect}, nil
	default:
		return nil, fmt.Errorf("unsupported Spot ledger dialect %q", dialect)
	}
}

type OrderIntent struct {
	PositionID    int64
	Symbol        string
	Side          string // BUY or SELL, strictly no SHORT
	Type          string // MARKET or LIMIT
	ClientOrderID string
	RequestedQty  string
}

type FillEvent struct {
	OrderID         int64
	Symbol          string
	ExchangeOrderID string
	TradeID         string
	Side            string
	Qty             string
	Price           string
	QuoteQty        string
	Commission      string
	CommissionAsset string
	TradeTime       int64
}

type FillResult struct {
	Applied  bool                       `json:"applied"`
	Position models.SpotManagedPosition `json:"position"`
	Order    models.SpotManagedOrder    `json:"order"`
}

type Snapshot struct {
	Position      models.SpotManagedPosition `json:"position"`
	MarkPrice     string                     `json:"mark_price,omitempty"`
	UnrealizedPnL string                     `json:"unrealized_pnl,omitempty"`
	TotalPnL      string                     `json:"total_pnl,omitempty"`
	RealizedPnL   string                     `json:"realized_pnl"`
	// Quote/base fees are included in PnL. Other-asset commissions cannot
	// be converted without a verified historical exchange rate.
	UnpricedFees map[string]string `json:"unpriced_fees"`
	FullyPriced  bool              `json:"fully_priced"`
	HasMarkPrice bool              `json:"has_mark_price"`
}

var assetPattern = regexp.MustCompile(`^[A-Z0-9]{1,32}$`)
var orderIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
var exchangeIDPattern = regexp.MustCompile(`^[0-9]{1,64}$`)

func validateIdentity(symbol, base, quote string) error {
	for _, value := range []string{symbol, base, quote} {
		if !assetPattern.MatchString(value) {
			return fmt.Errorf("invalid Spot symbol/asset identity")
		}
	}
	if base == quote || base+quote != symbol {
		return fmt.Errorf("Spot symbol %q does not match %s/%s", symbol, base, quote)
	}
	if quote != "USDT" && quote != "USDC" && quote != "FDUSD" {
		return fmt.Errorf("Spot Quant only supports USDT/USDC/FDUSD quote assets")
	}
	return nil
}

func now() int64 { return time.Now().UnixMilli() }

func (l *Ledger) begin() (orm.TxOrmer, func(), error) {
	if l.dialect == "sqlite" {
		sqliteWrites.Lock()
	}
	tx, err := l.db.Begin()
	if err != nil {
		if l.dialect == "sqlite" {
			sqliteWrites.Unlock()
		}
		return nil, nil, err
	}
	cleanup := func() {
		if l.dialect == "sqlite" {
			sqliteWrites.Unlock()
		}
	}
	return tx, cleanup, nil
}

func (l *Ledger) lock() string {
	if l.dialect != "sqlite" {
		return " FOR UPDATE"
	}
	return ""
}

// CreatePosition starts a new, unfilled local position cycle; it does not
// claim any pre-existing Binance wallet balance and does not make an order.
func (l *Ledger) CreatePosition(symbol, base, quote string) (models.SpotManagedPosition, error) {
	var row models.SpotManagedPosition
	if err := validateIdentity(symbol, base, quote); err != nil {
		return row, err
	}
	tx, unlock, err := l.begin()
	if err != nil {
		return row, err
	}
	defer unlock()
	defer tx.Rollback()
	if l.dialect != "sqlite" {
		// Serialize new position cycles on the existing symbol row: a
		// COUNT(*) query alone cannot prevent two concurrent creators from
		// opening independent cycles under MySQL REPEATABLE READ.
		var symbolID int64
		if err = tx.Raw("SELECT id FROM spot_symbols WHERE symbol=? FOR UPDATE", symbol).QueryRow(&symbolID); err != nil {
			return row, fmt.Errorf("Spot symbol must exist before opening a strategy position: %w", err)
		}
	}
	var count int64
	if err = tx.Raw("SELECT COUNT(*) FROM spot_managed_positions WHERE symbol=? AND status IN ('PENDING_BUY','OPEN')", symbol).QueryRow(&count); err != nil {
		return row, err
	}
	if count > 0 {
		return row, fmt.Errorf("active Spot strategy position already exists for %s", symbol)
	}
	// A manually corrupted CLOSED cache is not proof that an earlier
	// strategy cycle really sold its inventory. Check verified fills before
	// allocating a new cycle (rare operation, bounded by symbol history).
	priorCycles := make([]models.SpotManagedPosition, 0)
	_, err = tx.Raw("SELECT id,symbol,base_asset,quote_asset,bought_qty,remaining_qty,avg_cost_quote,realized_pnl,status,created_at,updated_at,closed_at FROM spot_managed_positions WHERE symbol=? ORDER BY id DESC", symbol).QueryRows(&priorCycles)
	if err != nil {
		return row, err
	}
	for _, prior := range priorCycles {
		projection, err := projectPosition(tx, prior)
		if err != nil {
			return row, err
		}
		if len(projection.reviewIssues) > 0 {
			return row, fmt.Errorf("%w: previous cycle %d: %s", ErrNeedsReview, prior.ID, strings.Join(projection.reviewIssues, "; "))
		}
		if projection.position.Status != "CLOSED" {
			return row, fmt.Errorf("%w: previous Spot cycle %d still holds strategy inventory", ErrNeedsReview, prior.ID)
		}
	}
	t := now()
	res, err := tx.Raw("INSERT INTO spot_managed_positions (symbol,base_asset,quote_asset,bought_qty,remaining_qty,avg_cost_quote,realized_pnl,status,created_at,updated_at,closed_at) VALUES(?,?,?,0,0,0,0,'PENDING_BUY',?,?,0)", symbol, base, quote, t, t).Exec()
	if err != nil {
		return row, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return row, err
	}
	row, err = readPosition(tx, id, "")
	if err != nil {
		return row, err
	}
	if err = tx.Commit(); err != nil {
		return row, err
	}
	return row, nil
}

func readPosition(q orm.QueryExecutor, id int64, lock string) (models.SpotManagedPosition, error) {
	var p models.SpotManagedPosition
	if id <= 0 {
		return p, fmt.Errorf("invalid position id")
	}
	err := q.Raw("SELECT id,symbol,base_asset,quote_asset,bought_qty,remaining_qty,avg_cost_quote,realized_pnl,status,created_at,updated_at,closed_at FROM spot_managed_positions WHERE id=?"+lock, id).
		QueryRow(&p.ID, &p.Symbol, &p.BaseAsset, &p.QuoteAsset, &p.BoughtQty, &p.RemainingQty, &p.AvgCostQuote, &p.RealizedPnL, &p.Status, &p.CreatedAt, &p.UpdatedAt, &p.ClosedAt)
	return p, err
}

func readOrder(q orm.QueryExecutor, id int64, lock string) (models.SpotManagedOrder, error) {
	var o models.SpotManagedOrder
	if id <= 0 {
		return o, fmt.Errorf("invalid order id")
	}
	err := q.Raw("SELECT id,position_id,symbol,side,order_type,client_order_id,COALESCE(exchange_order_id,''),requested_qty,filled_qty,status,created_at,updated_at FROM spot_managed_orders WHERE id=?"+lock, id).
		QueryRow(&o.ID, &o.PositionID, &o.Symbol, &o.Side, &o.OrderType, &o.ClientOrderID, &o.ExchangeOrderID, &o.RequestedQty, &o.FilledQty, &o.Status, &o.CreatedAt, &o.UpdatedAt)
	return o, err
}

func (l *Ledger) GetPosition(id int64) (models.SpotManagedPosition, error) {
	return readPosition(l.db, id, "")
}
func (l *Ledger) GetOrder(id int64) (models.SpotManagedOrder, error) { return readOrder(l.db, id, "") }

func (l *Ledger) ListOrders(positionID int64) ([]models.SpotManagedOrder, error) {
	if positionID <= 0 {
		return nil, fmt.Errorf("invalid position id")
	}
	var rows []models.SpotManagedOrder
	_, err := l.db.Raw("SELECT id,position_id,symbol,side,order_type,client_order_id,COALESCE(exchange_order_id,'') AS exchange_order_id,requested_qty,filled_qty,status,created_at,updated_at FROM spot_managed_orders WHERE position_id=? ORDER BY id", positionID).QueryRows(&rows)
	return rows, err
}

func (l *Ledger) ListFills(positionID int64) ([]models.SpotTradeFill, error) {
	if positionID <= 0 {
		return nil, fmt.Errorf("invalid position id")
	}
	var rows []models.SpotTradeFill
	_, err := l.db.Raw("SELECT id,position_id,order_id,exchange_order_id,trade_id,symbol,side,qty,price,quote_qty,commission,commission_asset,trade_time FROM spot_trade_fills WHERE position_id=? ORDER BY trade_time,id", positionID).QueryRows(&rows)
	return rows, err
}

func reserveOpenOrders(tx orm.QueryExecutor, positionID int64) (decimal.Decimal, error) {
	rows := []struct {
		Side         string
		RequestedQty string
		FilledQty    string
		Status       string
	}{}
	_, err := tx.Raw("SELECT side,requested_qty,filled_qty,status FROM spot_managed_orders WHERE position_id=?", positionID).QueryRows(&rows)
	if err != nil {
		return zero, err
	}
	remaining := zero
	for _, row := range rows {
		if row.Side != "SELL" || row.Status == "FILLED" || row.Status == "CANCELED" || row.Status == "EXPIRED" || row.Status == "REJECTED" {
			continue
		}
		requested, err := parseDBDecimal(row.RequestedQty, false)
		if err != nil {
			return zero, err
		}
		filled, err := parseDBDecimal(row.FilledQty, false)
		if err != nil {
			return zero, err
		}
		remaining = remaining.Add(requested.Sub(filled))
	}
	return remaining, nil
}

func (l *Ledger) CreateOrder(intent OrderIntent) (models.SpotManagedOrder, bool, error) {
	var empty models.SpotManagedOrder
	if intent.PositionID <= 0 || !assetPattern.MatchString(intent.Symbol) ||
		(intent.Side != "BUY" && intent.Side != "SELL") || (intent.Type != "MARKET" && intent.Type != "LIMIT") ||
		!orderIDPattern.MatchString(intent.ClientOrderID) {
		return empty, false, fmt.Errorf("invalid Spot BUY/SELL order intent")
	}
	qty, err := parseDBDecimal(intent.RequestedQty, true)
	if err != nil {
		return empty, false, err
	}
	normalized, err := numeric(qty)
	if err != nil {
		return empty, false, err
	}
	tx, unlock, err := l.begin()
	if err != nil {
		return empty, false, err
	}
	defer unlock()
	defer tx.Rollback()
	position, err := readPosition(tx, intent.PositionID, l.lock())
	if err != nil {
		return empty, false, err
	}
	if position.Symbol != intent.Symbol {
		return empty, false, fmt.Errorf("Spot order does not belong to strategy position")
	}
	// Idempotent create: retrying the same client order must not create a new
	// local intent or permit a different request to reuse its identity.
	var existingID int64
	err = tx.Raw("SELECT id FROM spot_managed_orders WHERE client_order_id=?", intent.ClientOrderID).QueryRow(&existingID)
	if err == nil {
		existing, err := readOrder(tx, existingID, "")
		if err != nil {
			return empty, false, err
		}
		savedQty, err := parseDBDecimal(existing.RequestedQty, true)
		if err != nil {
			return empty, false, err
		}
		if existing.PositionID != intent.PositionID || existing.Side != intent.Side ||
			existing.Symbol != intent.Symbol || existing.OrderType != intent.Type || !savedQty.Equal(qty) {
			return empty, false, fmt.Errorf("client order id already used for different Spot order")
		}
		if err = tx.Commit(); err != nil {
			return empty, false, err
		}
		return existing, true, nil
	}
	if err != orm.ErrNoRows {
		return empty, false, err
	}
	// Financial fields on spot_managed_positions are a materialized cache,
	// not an authority. Rebuild from verified fills in this SAME locked
	// transaction before accepting ANY new BUY or SELL order.
	projection, err := projectPosition(tx, position)
	if err != nil {
		return empty, false, err
	}
	if len(projection.reviewIssues) > 0 {
		return empty, false, fmt.Errorf("%w: %s", ErrNeedsReview, strings.Join(projection.reviewIssues, "; "))
	}
	position, err = writeProjection(tx, projection)
	if err != nil {
		return empty, false, err
	}
	if position.Status == "CLOSED" {
		return empty, false, fmt.Errorf("cannot create order for closed Spot strategy position")
	}
	var unresolved int64
	if err = tx.Raw("SELECT COUNT(*) FROM spot_managed_orders WHERE position_id=? AND (status IN ('NEW','PARTIALLY_FILLED','PENDING_CANCEL') OR (status='FILLED' AND filled_qty < requested_qty))", position.ID).QueryRow(&unresolved); err != nil {
		return empty, false, err
	}
	if unresolved > 0 {
		return empty, false, fmt.Errorf("one unresolved Spot strategy order per position")
	}
	if intent.Side == "SELL" {
		if position.Status != "OPEN" {
			return empty, false, fmt.Errorf("cannot SELL before strategy BUY fills")
		}
		available, err := parseDBDecimal(position.RemainingQty, false)
		if err != nil {
			return empty, false, err
		}
		reserved, err := reserveOpenOrders(tx, position.ID)
		if err != nil {
			return empty, false, err
		}
		if qty.GreaterThan(available.Sub(reserved)) {
			return empty, false, fmt.Errorf("SELL exceeds remaining strategy-owned base asset")
		}
	}
	t := now()
	result, err := tx.Raw("INSERT INTO spot_managed_orders(position_id,symbol,side,order_type,client_order_id,exchange_order_id,requested_qty,filled_qty,status,created_at,updated_at) VALUES(?,?,?,?,?,NULL,?,0,'NEW',?,?)", position.ID, intent.Symbol, intent.Side, intent.Type, intent.ClientOrderID, normalized, t, t).Exec()
	if err != nil {
		return empty, false, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return empty, false, err
	}
	created, err := readOrder(tx, id, "")
	if err != nil {
		return empty, false, err
	}
	if err = tx.Commit(); err != nil {
		return empty, false, err
	}
	return created, false, nil
}

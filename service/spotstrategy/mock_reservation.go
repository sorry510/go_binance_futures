package spotstrategy

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// MockReservation is intentionally not wired to any HTTP route, live trader
// or Binance POST. It exercises persistent S1 idempotency and serialization.
type MockReservation struct {
	AccountID     string
	Owner         string
	Symbol        string
	ClientOrderID string
	SignalKey     string
	RequestedQty  string
}

// ReserveMockOrder persists only a local pending BUY intent. S2 will implement
// the real order lifecycle, and S3 will validate quote balance/ownership.
func ReserveMockOrder(req MockReservation) (id int64, err error) {
	if err := ValidateScope(req.AccountID, req.Owner); err != nil {
		return 0, err
	}
	if strings.TrimSpace(req.Symbol) == "" || len(req.Symbol) > 32 {
		return 0, errors.New("mock symbol required")
	}
	if len(req.ClientOrderID) == 0 || len(req.ClientOrderID) > 36 ||
		len(req.SignalKey) == 0 || len(req.SignalKey) > 64 {
		return 0, errors.New("invalid mock order identity")
	}
	if req.Symbol != strings.ToUpper(req.Symbol) {
		return 0, errors.New("mock symbol must be uppercase")
	}
	qty, err := NormalizeBudget(req.RequestedQty, true)
	if err != nil {
		return 0, err
	}
	o := orm.NewOrm()
	tx, err := o.Begin()
	if err != nil {
		return 0, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Select a stable pre-existing row. A holding row does NOT yet exist on
	// the first BUY. The pending_slot unique index handles cross-process races.
	query := "SELECT id FROM spot_symbols WHERE symbol = ?"
	db, dbErr := orm.GetDB("default")
	if dbErr != nil {
		return 0, dbErr
	}
	if strings.Contains(strings.ToLower(fmt.Sprintf("%T", db.Driver())), "mysql") {
		query += " FOR UPDATE"
	}
	var symbolID int64
	if err = tx.Raw(query, req.Symbol).QueryRow(&symbolID); err != nil {
		return 0, fmt.Errorf("mock symbol lock: %w", err)
	}
	var enabled int
	if err = tx.Raw("SELECT strategy_trade_enabled FROM spot_symbols WHERE id = ?", symbolID).QueryRow(&enabled); err != nil {
		return 0, err
	}
	if enabled != 1 {
		return 0, errors.New("spot strategy not enabled for mock reservation")
	}
	slot := req.AccountID + ":" + req.Owner + ":" + req.Symbol
	now := time.Now().UnixMilli()
	result, execErr := tx.Raw(`INSERT INTO spot_strategy_orders (
	 account_id,owner,symbol,side,order_type,intent,client_order_id,signal_key,
	 pending_slot,requested_qty,reserved_qty,status,created_at,updated_at
	) VALUES (?,?,?,'BUY','MARKET','open',?,?,?, ?,?,'reserved',?,?)`,
		req.AccountID, req.Owner, req.Symbol, req.ClientOrderID, req.SignalKey, slot, qty, qty, now, now).Exec()
	if execErr != nil {
		err = execErr
		return 0, fmt.Errorf("reserve mock order (duplicate or busy): %w", err)
	}
	id, err = result.LastInsertId()
	if err != nil {
		return 0, err
	}
	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit mock reservation: %w", err)
	}
	return id, nil
}

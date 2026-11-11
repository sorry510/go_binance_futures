package spotledger

import (
	"fmt"

	"github.com/beego/beego/v2/client/orm"
	"go_binance_futures/models"
)

// FindOrderByClientID is the durable, unique identity for a Spot order.
// Retrying a network-ambiguous order must NOT create another local intent.
func (l *Ledger) FindOrderByClientID(clientID string) (models.SpotManagedOrder, error) {
	if !orderIDPattern.MatchString(clientID) {
		return models.SpotManagedOrder{}, fmt.Errorf("invalid client order id")
	}
	var id int64
	if err := l.db.Raw("SELECT id FROM spot_managed_orders WHERE client_order_id=?", clientID).QueryRow(&id); err != nil {
		return models.SpotManagedOrder{}, err
	}
	return readOrder(l.db, id, "")
}

// IsOrderNotFound is used by the adapter only after a definitive -2013
// exchange response. Network timeouts must not be treated as not-found.
func IsOrderNotFound(err error) bool { return err == orm.ErrNoRows }

// ConfirmNeverAccepted is called ONLY after an operator has explicitly
// authorized two authoritative absent-order REST confirmations. A network
// error or one not-found response must never reach this method.
func (l *Ledger) ConfirmNeverAccepted(orderID int64) (models.SpotManagedOrder, error) {
	var empty models.SpotManagedOrder
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
	p, err := readPosition(tx, first.PositionID, l.lock())
	if err != nil {
		return empty, err
	}
	o, err := readOrder(tx, orderID, l.lock())
	if err != nil {
		return empty, err
	}
	if o.Status != "NEW" || o.ExchangeOrderID != "" || o.Side != "BUY" {
		return empty, fmt.Errorf("%w: only a never-bound NEW BUY can be marked absent", ErrNeedsReview)
	}
	orders, err := getPositionOrders(tx, p.ID)
	if err != nil {
		return empty, err
	}
	if len(orders) != 1 {
		return empty, fmt.Errorf("%w: ambiguous position order history", ErrNeedsReview)
	}
	fills, err := getPositionFills(tx, p.ID)
	if err != nil {
		return empty, err
	}
	if len(fills) != 0 {
		return empty, fmt.Errorf("%w: cannot abandon position with verified fills", ErrNeedsReview)
	}
	if _, err = tx.Raw("UPDATE spot_managed_orders SET status='REJECTED',updated_at=? WHERE id=? AND status='NEW' AND exchange_order_id IS NULL", now(), orderID).Exec(); err != nil {
		return empty, err
	}
	if _, err = reprojectPosition(tx, p); err != nil {
		return empty, err
	}
	o, err = readOrder(tx, orderID, "")
	if err != nil {
		return empty, err
	}
	if err = tx.Commit(); err != nil {
		return empty, err
	}
	return o, nil
}

package coinmexecution

import (
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/models"
)

const orderCols = "id,position_id,symbol,position_side,intent,side,order_type,client_order_id,COALESCE(exchange_order_id,'') AS exchange_order_id,requested_contracts,filled_contracts,status,created_at,updated_at"
const positionCols = "id,symbol,position_side,margin_asset,contract_size,entry_price,opened_contracts,remaining_contracts,realized_pnl,status,created_at,updated_at,closed_at"

func loadOrder(db orm.Ormer, id string) (models.CoinMManagedOrder, error) {
	var rows []models.CoinMManagedOrder
	_, err := db.Raw("SELECT "+orderCols+" FROM coin_m_managed_orders WHERE client_order_id=?", id).QueryRows(&rows)
	if err != nil {
		return models.CoinMManagedOrder{}, err
	}
	if len(rows) == 0 {
		return models.CoinMManagedOrder{}, orm.ErrNoRows
	}
	return rows[0], nil
}
func loadPosition(db orm.Ormer, id int64) (models.CoinMManagedPosition, error) {
	var rows []models.CoinMManagedPosition
	_, err := db.Raw("SELECT "+positionCols+" FROM coin_m_managed_positions WHERE id=?", id).QueryRows(&rows)
	if err != nil {
		return models.CoinMManagedPosition{}, err
	}
	if len(rows) == 0 {
		return models.CoinMManagedPosition{}, orm.ErrNoRows
	}
	return rows[0], nil
}
func localOrder(db orm.Ormer, id string) (models.CoinMManagedOrder, bool, error) {
	o, e := loadOrder(db, id)
	if errors.Is(e, orm.ErrNoRows) {
		return o, false, nil
	}
	return o, e == nil, e
}
func (p *Processor) FindOrder(id string) (models.CoinMManagedOrder, error) {
	if p == nil || p.DB == nil {
		return models.CoinMManagedOrder{}, errors.New("COIN-M database unconfigured")
	}
	return loadOrder(p.DB, id)
}
func insertOrder(tx orm.TxOrmer, posID int64, req binance.CoinMTradeOrder, intent string) error {
	now := time.Now().UnixMilli()
	_, err := tx.Raw("INSERT INTO coin_m_managed_orders (position_id,symbol,position_side,intent,side,order_type,client_order_id,exchange_order_id,requested_contracts,filled_contracts,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?,NULL,?,?,?, ?,?)",
		posID, req.Symbol, req.PositionSide, intent, req.Side, req.Type, req.ClientOrderID, req.Contracts, "0", "PENDING_SUBMIT", now, now).Exec()
	return err
}
func (p *Processor) updateAck(ctxErr error, clientID string, ack *OrderAck) error {
	order, err := loadOrder(p.DB, clientID)
	if err != nil {
		return err
	}
	if ctxErr != nil {
		_, err = p.DB.Raw("UPDATE coin_m_managed_orders SET status='UNKNOWN',updated_at=? WHERE id=? AND status='PENDING_SUBMIT'", time.Now().UnixMilli(), order.ID).Exec()
		return err
	}
	if err := validateAck(binance.CoinMTradeOrder{ClientOrderID: order.ClientOrderID, Symbol: order.Symbol,
		Side: order.Side, PositionSide: order.PositionSide}, ack); err != nil {
		_, _ = p.DB.Raw("UPDATE coin_m_managed_orders SET status='UNKNOWN',updated_at=? WHERE id=?", time.Now().UnixMilli(), order.ID).Exec()
		return err
	}
	_, err = p.DB.Raw("UPDATE coin_m_managed_orders SET exchange_order_id=?,status=?,updated_at=? WHERE id=?",
		strconv.FormatInt(ack.ExchangeOrderID, 10), ack.Status, time.Now().UnixMilli(), order.ID).Exec()
	return err
}

// applyFill strictly requires an OWNED order and exact symbol/side/orderId.
// It never accepts manual/unmanaged wallet positions or fabricates a Fill from
// executedQty in a simple REST/WS order acknowledgement.
func (p *Processor) ApplyFill(tr binance.CoinMTrade) (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.applyFill(tr)
}
func (p *Processor) applyFill(tr binance.CoinMTrade) (bool, error) {
	if p == nil || p.DB == nil {
		return false, errors.New("COIN-M database unconfigured")
	}
	if tr.Symbol == "" || tr.OrderID <= 0 || tr.TradeID <= 0 ||
		(tr.Side != "BUY" && tr.Side != "SELL") || (tr.PositionSide != "LONG" && tr.PositionSide != "SHORT") ||
		tr.Time <= 0 || tr.MarginAsset == "" || tr.CommissionAsset == "" {
		return false, errors.New("COIN-M Fill missing exchange identity/asset/side")
	}
	qty, e := positive(tr.Quantity)
	if e != nil {
		return false, e
	}
	if _, e = positive(tr.Price); e != nil {
		return false, e
	}
	pnl, e := decimal(tr.RealizedPnL)
	if e != nil {
		return false, e
	}
	fee, e := decimal(tr.Commission)
	if e != nil {
		return false, e
	}
	if fee.Sign() < 0 {
		return false, errors.New("negative COIN-M Fill commission")
	}
	var orders []models.CoinMManagedOrder
	_, e = p.DB.Raw("SELECT "+orderCols+" FROM coin_m_managed_orders WHERE symbol=? AND exchange_order_id=?",
		tr.Symbol, strconv.FormatInt(tr.OrderID, 10)).QueryRows(&orders)
	if e != nil {
		return false, e
	}
	if len(orders) != 1 {
		return false, errors.New("COIN-M Fill not owned by exactly one managed order")
	}
	o := orders[0]
	if o.Side != tr.Side || o.PositionSide != tr.PositionSide ||
		(o.OrderType != "MARKET" && o.OrderType != "LIMIT" &&
			!((o.OrderType == "STOP_MARKET" || o.OrderType == "TAKE_PROFIT_MARKET") &&
				o.Intent == "PROTECT" && (o.Status == "TRIGGERED" || o.Status == "PARTIALLY_FILLED" || o.Status == "FILLED"))) {
		return false, errors.New("COIN-M Fill does not match managed order")
	}
	pos, e := loadPosition(p.DB, o.PositionID)
	if e != nil {
		return false, e
	}
	if pos.Symbol != tr.Symbol || pos.PositionSide != tr.PositionSide || pos.MarginAsset != tr.MarginAsset {
		return false, errors.New("COIN-M Fill belongs to another managed symbol/asset/hedge side")
	}
	// Must match exactly one of the two expected trade directions.
	intent := o.Intent
	if intent == "PROTECT" {
		intent = "CLOSE"
	}
	expected, e := orderSide(o.PositionSide, intent)
	if e != nil || expected != tr.Side {
		return false, errors.New("COIN-M Fill direction conflicts with order intent")
	}
	tx, e := p.DB.Begin()
	if e != nil {
		return false, e
	}
	defer tx.Rollback()
	var count int64
	if e = tx.Raw("SELECT COUNT(*) FROM coin_m_trade_fills WHERE symbol=? AND exchange_order_id=? AND trade_id=?",
		tr.Symbol, strconv.FormatInt(tr.OrderID, 10), strconv.FormatInt(tr.TradeID, 10)).QueryRow(&count); e != nil {
		return false, e
	}
	if count > 0 {
		return false, nil
	}
	_, e = tx.Raw("INSERT INTO coin_m_trade_fills (position_id,order_id,symbol,position_side,side,exchange_order_id,trade_id,contracts,price,realized_pnl,pnl_asset,commission,commission_asset,trade_time) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)",
		pos.ID, o.ID, tr.Symbol, tr.PositionSide, tr.Side, strconv.FormatInt(tr.OrderID, 10), strconv.FormatInt(tr.TradeID, 10),
		str(qty), tr.Price, str(pnl), tr.MarginAsset, str(fee), tr.CommissionAsset, tr.Time).Exec()
	if e != nil {
		return false, e
	}
	if e = refreshManagedCycle(tx, pos.ID); e != nil {
		return false, e
	}
	if e = tx.Commit(); e != nil {
		return false, e
	}
	return true, nil
}

type cycleFill struct {
	Intent    string `orm:"column(intent)"`
	Contracts string `orm:"column(contracts)"`
	Price     string `orm:"column(price)"`
	PnL       string `orm:"column(realized_pnl)"`
}

func refreshManagedCycle(tx orm.TxOrmer, positionID int64) error {
	var rows []cycleFill
	_, err := tx.Raw("SELECT o.intent AS intent,f.contracts AS contracts,f.price AS price,f.realized_pnl AS realized_pnl FROM coin_m_trade_fills f JOIN coin_m_managed_orders o ON o.id=f.order_id AND o.position_id=f.position_id WHERE f.position_id=?", positionID).QueryRows(&rows)
	if err != nil {
		return err
	}
	open, closed, realized, invSum := new(big.Rat), new(big.Rat), new(big.Rat), new(big.Rat)
	for _, row := range rows {
		qty, e := positive(row.Contracts)
		if e != nil {
			return e
		}
		pnl, e := decimal(row.PnL)
		if e != nil {
			return e
		}
		price, e := positive(row.Price)
		if e != nil {
			return e
		}
		switch row.Intent {
		case "OPEN":
			open.Add(open, qty)
			invSum.Add(invSum, new(big.Rat).Quo(qty, price))
		case "CLOSE", "PROTECT":
			closed.Add(closed, qty)
		default:
			return errors.New("COIN-M Fill intent not OPEN/CLOSE")
		}
		realized.Add(realized, pnl)
	}
	if closed.Cmp(open) > 0 {
		return errors.New("COIN-M closing Fills exceed opened contract count")
	}
	entry := "0"
	if open.Sign() > 0 {
		if invSum.Sign() <= 0 {
			return errors.New("invalid COIN-M weighted inverse entry price")
		}
		entry = str(new(big.Rat).Quo(open, invSum))
	}
	remain := new(big.Rat).Sub(open, closed)
	status := "PENDING_OPEN"
	closedAt := int64(0)
	if open.Sign() > 0 {
		status = "OPEN"
		if remain.Sign() == 0 {
			status = "CLOSED"
			closedAt = time.Now().UnixMilli()
		}
	}
	now := time.Now().UnixMilli()
	if _, err = tx.Raw("UPDATE coin_m_managed_positions SET entry_price=?,opened_contracts=?,remaining_contracts=?,realized_pnl=?,status=?,updated_at=?,closed_at=? WHERE id=?",
		entry, str(open), str(remain), str(realized), status, now, closedAt, positionID).Exec(); err != nil {
		return err
	}
	// The order's FILLED status is not enough to prove Fills, but once the
	// aggregated real Fills equal requested contracts the order is settled.
	var orders []models.CoinMManagedOrder
	_, err = tx.Raw("SELECT "+orderCols+" FROM coin_m_managed_orders WHERE position_id=?", positionID).QueryRows(&orders)
	if err != nil {
		return err
	}
	for _, o := range orders {
		var fills []struct {
			Qty string `orm:"column(contracts)"`
		}
		_, err = tx.Raw("SELECT contracts FROM coin_m_trade_fills WHERE order_id=?", o.ID).QueryRows(&fills)
		if err != nil {
			return err
		}
		filled := new(big.Rat)
		for _, f := range fills {
			q, e := positive(f.Qty)
			if e != nil {
				return e
			}
			filled.Add(filled, q)
		}
		requested, e := positive(o.RequestedContracts)
		if e != nil {
			return e
		}
		if filled.Cmp(requested) > 0 {
			return fmt.Errorf("COIN-M Fill sum exceeds order %d requested contracts", o.ID)
		}
		next := o.Status
		if filled.Cmp(requested) == 0 {
			next = "FILLED"
		} else if filled.Sign() > 0 && (o.Status == "NEW" || o.Status == "PENDING_SUBMIT" || o.Status == "UNKNOWN") {
			next = "PARTIALLY_FILLED"
		}
		_, err = tx.Raw("UPDATE coin_m_managed_orders SET filled_contracts=?,status=?,updated_at=? WHERE id=?",
			str(filled), next, now, o.ID).Exec()
		if err != nil {
			return err
		}
	}
	return nil
}

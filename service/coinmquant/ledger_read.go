package coinmquant

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/beego/beego/v2/client/orm"
	"go_binance_futures/models"
)

// CM-3 only: direct, read-only local ledger projection. CM-4 will add
// inverse PnL, mark prices, asset-level summaries and lifecycle details.
// No wallet, FAPI, Spot or Binance order endpoint is queried by this code.
type LedgerFilter struct {
	Symbol, Status string
	Page, Limit    int
}

func ledgerWhere(f LedgerFilter) (string, []interface{}, error) {
	var predicates []string
	var args []interface{}
	if v := strings.ToUpper(strings.TrimSpace(f.Symbol)); v != "" {
		if len(v) > 32 || strings.ContainsAny(v, "%") {
			return "", nil, fmt.Errorf("invalid COIN-M ledger symbol search")
		}
		predicates = append(predicates, "symbol=?")
		args = append(args, v)
	}
	if s := strings.TrimSpace(f.Status); s != "" {
		if len(s) > 32 || strings.ContainsAny(s, " %") {
			return "", nil, fmt.Errorf("invalid ledger status")
		}
		predicates = append(predicates, "status=?")
		args = append(args, s)
	}
	if len(predicates) == 0 {
		return "", args, nil
	}
	return " WHERE " + strings.Join(predicates, " AND "), args, nil
}
func ledgerPage(f LedgerFilter) (int, int) { return pageSize(f.Page, f.Limit) }
func Positions(o orm.Ormer, f LedgerFilter) (Page[models.CoinMManagedPosition], error) {
	where, args, err := ledgerWhere(f)
	if err != nil {
		return Page[models.CoinMManagedPosition]{}, err
	}
	var total int64
	if err = o.Raw("SELECT COUNT(*) FROM coin_m_managed_positions"+where, args...).QueryRow(&total); err != nil {
		return Page[models.CoinMManagedPosition]{}, err
	}
	page, limit := ledgerPage(f)
	sql := "SELECT id,symbol,position_side,margin_asset,contract_size,entry_price,opened_contracts,remaining_contracts,realized_pnl,status,created_at,updated_at,closed_at FROM coin_m_managed_positions" + where +
		" ORDER BY id DESC LIMIT " + strconv.Itoa(limit) + " OFFSET " + strconv.Itoa((page-1)*limit)
	rows := make([]models.CoinMManagedPosition, 0)
	if _, err = o.Raw(sql, args...).QueryRows(&rows); err != nil {
		return Page[models.CoinMManagedPosition]{}, err
	}
	return Page[models.CoinMManagedPosition]{Total: total, List: rows}, nil
}
func Orders(o orm.Ormer, f LedgerFilter) (Page[models.CoinMManagedOrder], error) {
	where, args, err := ledgerWhere(f)
	if err != nil {
		return Page[models.CoinMManagedOrder]{}, err
	}
	var total int64
	if err = o.Raw("SELECT COUNT(*) FROM coin_m_managed_orders"+where, args...).QueryRow(&total); err != nil {
		return Page[models.CoinMManagedOrder]{}, err
	}
	page, limit := ledgerPage(f)
	sql := "SELECT id,position_id,symbol,position_side,intent,side,order_type,client_order_id,COALESCE(exchange_order_id,'') AS exchange_order_id,requested_contracts,filled_contracts,status,created_at,updated_at FROM coin_m_managed_orders" + where +
		" ORDER BY id DESC LIMIT " + strconv.Itoa(limit) + " OFFSET " + strconv.Itoa((page-1)*limit)
	rows := make([]models.CoinMManagedOrder, 0)
	if _, err = o.Raw(sql, args...).QueryRows(&rows); err != nil {
		return Page[models.CoinMManagedOrder]{}, err
	}
	return Page[models.CoinMManagedOrder]{Total: total, List: rows}, nil
}

type PositionDetail struct {
	Position models.CoinMManagedPosition `json:"position"`
	Orders   []models.CoinMManagedOrder  `json:"orders"`
	Fills    []models.CoinMTradeFill     `json:"fills"`
}

func Detail(o orm.Ormer, id int64) (PositionDetail, error) {
	var result PositionDetail
	if id < 1 {
		return result, fmt.Errorf("invalid COIN-M position id")
	}
	rows := make([]models.CoinMManagedPosition, 0)
	if _, err := o.Raw("SELECT id,symbol,position_side,margin_asset,contract_size,entry_price,opened_contracts,remaining_contracts,realized_pnl,status,created_at,updated_at,closed_at FROM coin_m_managed_positions WHERE id=?", id).QueryRows(&rows); err != nil {
		return result, err
	}
	if len(rows) == 0 {
		return result, fmt.Errorf("COIN-M managed position not found")
	}
	result.Position = rows[0]
	result.Orders = make([]models.CoinMManagedOrder, 0)
	result.Fills = make([]models.CoinMTradeFill, 0)
	_, err := o.Raw("SELECT id,position_id,symbol,position_side,intent,side,order_type,client_order_id,COALESCE(exchange_order_id,'') AS exchange_order_id,requested_contracts,filled_contracts,status,created_at,updated_at FROM coin_m_managed_orders WHERE position_id=? ORDER BY id", id).QueryRows(&result.Orders)
	if err != nil {
		return result, err
	}
	_, err = o.Raw("SELECT id,position_id,order_id,symbol,position_side,side,exchange_order_id,trade_id,contracts,price,realized_pnl,pnl_asset,commission,commission_asset,trade_time FROM coin_m_trade_fills WHERE position_id=? ORDER BY trade_time,id", id).QueryRows(&result.Fills)
	return result, err
}

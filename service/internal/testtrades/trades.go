package testtrades

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"go_binance_futures/models"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/client/orm"
)

// Fake REST trade history is append-only per ORM order ID. Every increase of
// Binance cumulative quantity generates a new exchange trade ID; cancellation
// and duplicate receipt replay keep the existing immutable trades.
var mu sync.Mutex
var history = map[int64][]*futures.AccountTrade{}

type fakeMoney struct{ fee, asset, pnl string }

var moneyByExchangeOrder = map[int64]fakeMoney{}

// Reset must be called in every test DB fixture to avoid process-global
// trade IDs and financial values leaking between independent tests.
func Reset() {
	mu.Lock()
	history = map[int64][]*futures.AccountTrade{}
	moneyByExchangeOrder = map[int64]fakeMoney{}
	mu.Unlock()
}
func SetAmountsForExchangeOrder(orderID int64, fee, asset, pnl string) {
	mu.Lock()
	moneyByExchangeOrder[orderID] = fakeMoney{fee: fee, asset: asset, pnl: pnl}
	mu.Unlock()
}

func FromTestManagedOrder(_ context.Context, account, symbol string, orderID, fromID int64, limit int) ([]*futures.AccountTrade, error) {
	if orderID <= 0 || limit <= 0 {
		return nil, fmt.Errorf("invalid fake trade query")
	}
	o := orm.NewOrm()
	var direct, actual []models.FuturesManagedOrder
	_, err := o.QueryTable(new(models.FuturesManagedOrder)).Filter("symbol", symbol).
		Filter("exchange_order_id", strconv.FormatInt(orderID, 10)).All(&direct)
	if err != nil {
		return nil, err
	}
	_, err = o.QueryTable(new(models.FuturesManagedOrder)).Filter("symbol", symbol).
		Filter("actual_order_id", strconv.FormatInt(orderID, 10)).All(&actual)
	if err != nil {
		return nil, err
	}
	found := make([]models.FuturesManagedOrder, 0, 1)
	for _, m := range append(direct, actual...) {
		if (account != "" && m.AccountID != account) || m.FilledQty <= 0 {
			continue
		}
		if len(found) > 0 && found[0].ID == m.ID {
			continue
		}
		found = append(found, m)
	}
	if len(found) > 1 {
		return nil, fmt.Errorf("ambiguous fake exchange order")
	}
	if len(found) == 0 {
		return nil, nil
	}
	m := found[0]
	price := "100"
	if m.PositionID != nil {
		var p models.FuturesManagedPosition
		if err := o.QueryTable(new(models.FuturesManagedPosition)).Filter("id", *m.PositionID).One(&p); err == nil && p.EntryPrice > 0 {
			price = strconv.FormatFloat(p.EntryPrice, 'f', 8, 64)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	prior := 0.0
	for _, trade := range history[m.ID] {
		v, _ := strconv.ParseFloat(trade.Quantity, 64)
		prior += v
	}
	if m.FilledQty > prior+1e-9 {
		id := m.ID*10000 + int64(len(history[m.ID])) + 1
		financial := fakeMoney{fee: "0", asset: "USDT", pnl: "0"}
		if custom, ok := moneyByExchangeOrder[orderID]; ok {
			financial = custom
		}
		history[m.ID] = append(history[m.ID], &futures.AccountTrade{
			ID: id, OrderID: orderID, Symbol: m.Symbol,
			Quantity: strconv.FormatFloat(m.FilledQty-prior, 'f', 8, 64),
			Price:    price, Commission: financial.fee, CommissionAsset: financial.asset, RealizedPnl: financial.pnl,
			Time: 1800000000000 + id, PositionSide: futures.PositionSideType(m.PositionSide),
		})
	}
	out := make([]*futures.AccountTrade, 0, limit)
	for _, trade := range history[m.ID] {
		if trade.ID >= fromID && len(out) < limit {
			out = append(out, trade)
		}
	}
	return out, nil
}

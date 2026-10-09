package binance

import (
	"fmt"
	"strconv"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/client/orm"
	"go_binance_futures/models"
)

// persistFuturesAccountEvent is the account-scoped state writer, independent of
// the WebSocket connection lifecycle. Stage 5 can connect the Lead WS to it.
// It deliberately never adjusts ownership or claims an unmanaged position.
func persistFuturesAccountEvent(accountID AccountID, account *AccountClient, event *futures.WsUserDataEvent) error {
	return persistFuturesAccountEventWithDB(orm.NewOrm(), accountID, account, event)
}

func persistFuturesAccountEventWithDB(o orm.Ormer, accountID AccountID, account *AccountClient, event *futures.WsUserDataEvent) error {
	if accountID != MainAccountID && accountID != LeadAccountID {
		return fmt.Errorf("invalid user stream account")
	}
	if accountID == LeadAccountID && (account == nil || account.ID() != LeadAccountID) {
		return fmt.Errorf("lead user stream needs bound lead client")
	}
	if accountID == MainAccountID && account != nil && account.ID() != MainAccountID {
		return fmt.Errorf("main user stream account mismatch")
	}
	if event == nil {
		return fmt.Errorf("nil user stream event")
	}
	id := string(accountID)
	invalidate := func(symbol string) {
		if account != nil {
			account.InvalidateTradeConfig(symbol)
		} else {
			InvalidateTradeConfig(symbol)
		}
	}
	switch event.Event {
	case "ACCOUNT_UPDATE":
		for _, position := range event.AccountUpdate.Positions {
			invalidate(position.Symbol)
			amount, _ := strconv.ParseFloat(position.Amount, 64)
			var row models.FuturesPosition
			err := o.QueryTable("futures_positions").Filter("account_id", id).Filter("symbol", position.Symbol).Filter("side", position.Side).One(&row)
			if err != nil && err != orm.ErrNoRows {
				return err
			}
			if row.ID != 0 && event.Time > 0 && row.UpdateTime > event.Time {
				continue
			}
			var symbol models.Symbols
			_ = o.QueryTable("symbols").Filter("symbol", position.Symbol).One(&symbol)
			row.AccountID = id
			row.Symbol = position.Symbol
			row.Side = string(position.Side)
			row.Amount = strconv.FormatFloat(amount, 'f', -1, 64)
			row.MarginType = string(position.MarginType)
			row.Leverage = symbol.Leverage
			row.IsolatedWallet = position.IsolatedWallet
			row.EntryPrice = position.EntryPrice
			row.MarkPrice = position.MarkPrice
			row.UnrealizedProfit = position.UnrealizedPnL
			row.AccumulatedRealized = position.AccumulatedRealized
			row.MaintenanceMarginRequired = position.MaintenanceMarginRequired
			row.UpdateTime = event.Time
			if row.ID == 0 {
				row.CreateTime = event.Time
				if _, err := o.Insert(&row); err != nil {
					return err
				}
			} else if _, err := o.Update(&row); err != nil {
				return err
			}
		}
	case "ORDER_TRADE_UPDATE":
		order := event.OrderTradeUpdate
		var row models.FuturesOrder
		err := o.QueryTable("futures_orders").Filter("account_id", id).Filter("order_id", order.ID).One(&row)
		if err != nil && err != orm.ErrNoRows {
			return err
		}
		if row.ID != 0 && event.Time > 0 && row.UpdateTime > event.Time {
			return nil
		}
		row.AccountID = id
		row.Symbol = order.Symbol
		row.ClientOrderId = order.ClientOrderID
		row.OrderId = strconv.FormatInt(order.ID, 10)
		row.Side = string(order.Side)
		row.PositionSide = string(order.PositionSide)
		row.Type = string(order.Type)
		row.Status = string(order.Status)
		row.Price = order.OriginalPrice
		row.OrigQty = order.OriginalQty
		row.ExecutedQty = order.AccumulatedFilledQty
		row.AveragePrice = order.AveragePrice
		row.StopPrice = order.StopPrice
		row.CommissionAsset = order.CommissionAsset
		row.Commission = order.Commission
		row.RealizedPnL = order.RealizedPnL
		row.UpdateTime = event.Time
		if row.ID == 0 {
			row.CreateTime = event.Time
			if _, err := o.Insert(&row); err != nil {
				return err
			}
		} else if _, err := o.Update(&row); err != nil {
			return err
		}
	case "ACCOUNT_CONFIG_UPDATE":
		c := event.AccountConfigUpdate
		invalidate(c.Symbol)
		if c.Leverage == 0 {
			return nil
		}
		var rows []models.FuturesPosition
		if _, err := o.QueryTable("futures_positions").Filter("account_id", id).Filter("symbol", c.Symbol).All(&rows); err != nil {
			return err
		}
		for i := range rows {
			rows[i].Leverage = c.Leverage
			if _, err := o.Update(&rows[i]); err != nil {
				return err
			}
		}
	}
	return nil
}

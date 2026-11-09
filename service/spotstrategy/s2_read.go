package spotstrategy

import (
	"context"
	"database/sql"
)

// RecordedFill exposes the exact source fee currency and distinguishes
// a simulated myTrades replay from a truly signed Spot REST verification.
type RecordedFill struct {
	Trade
	RESTVerified bool
}

// ListFills never queries another owner's account and does not infer fills
// from an order ACK. It returns only persisted exchange-trade identities.
func (e *Engine) ListFills(ctx context.Context, localOrderID int64) ([]RecordedFill, error) {
	order, err := e.loadOrder(ctx, localOrderID)
	if err != nil {
		return nil, err
	}
	if order.OrderID == "" {
		return []RecordedFill{}, nil
	}
	rows, err := e.db.QueryContext(ctx, `SELECT symbol,order_id,trade_id,side,qty,price,quote_qty,
 commission,commission_asset,trade_time,rest_verified
 FROM spot_strategy_fills WHERE account_id=? AND symbol=? AND order_id=?
 AND client_order_id=? ORDER BY trade_time,trade_id`,
		order.AccountID, order.Symbol, order.OrderID, order.ClientOrderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []RecordedFill{}
	for rows.Next() {
		var trade RecordedFill
		var verified sql.NullBool
		err = rows.Scan(&trade.Symbol, &trade.OrderID, &trade.TradeID, &trade.Side,
			&trade.Qty, &trade.Price, &trade.QuoteQty, &trade.Commission,
			&trade.CommissionAsset, &trade.Time, &verified)
		if err != nil {
			return nil, err
		}
		trade.RESTVerified = verified.Valid && verified.Bool
		out = append(out, trade)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

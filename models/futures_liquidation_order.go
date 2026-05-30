package models

type FuturesLiquidationOrder struct {
	ID                   int64   `orm:"column(id)" json:"id"`
	Symbol               string  `orm:"column(symbol);size(32);index" json:"symbol"`
	Side                 string  `orm:"column(side);size(16);index" json:"side"` // SELL: 多单爆仓, BUY: 空单爆仓
	OrderType            string  `orm:"column(order_type);size(32)" json:"order_type"`
	TimeInForce          string  `orm:"column(time_in_force);size(16)" json:"time_in_force"`
	OrigQuantity         string  `orm:"column(orig_quantity);size(64)" json:"orig_quantity"`
	Price                string  `orm:"column(price);size(64)" json:"price"`
	AvgPrice             string  `orm:"column(avg_price);size(64)" json:"avg_price"`
	OrderStatus          string  `orm:"column(order_status);size(32)" json:"order_status"`
	LastFilledQty        string  `orm:"column(last_filled_qty);size(64)" json:"last_filled_qty"`
	AccumulatedFilledQty string  `orm:"column(accumulated_filled_qty);size(64)" json:"accumulated_filled_qty"`
	Notional             float64 `orm:"column(notional);digits(30);decimals(8)" json:"notional"`
	TradeTime            int64   `orm:"column(trade_time);index" json:"trade_time"`
	EventTime            int64   `orm:"column(event_time);index" json:"event_time"`
	CreateTime           int64   `orm:"column(createTime)" json:"createTime"`
	UpdateTime           int64   `orm:"column(updateTime)" json:"updateTime"`
}

func (u *FuturesLiquidationOrder) TableName() string {
	return "futures_liquidation_order"
}

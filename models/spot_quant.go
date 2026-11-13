package models

// Spot Quant lifecycle records are deliberately separate from Futures.
// Decimal fields are kept as strings and are stored in SQL DECIMAL(36,18)
// columns by the versioned migration, not by Beego ORM. Do not RegisterModel
// these types: ORM synchronization would change exact decimals to VARCHAR.
type SpotManagedPosition struct {
	ID           int64  `orm:"column(id)" json:"id"`
	Symbol       string `orm:"column(symbol)" json:"symbol"`
	BaseAsset    string `orm:"column(base_asset)" json:"base_asset"`
	QuoteAsset   string `orm:"column(quote_asset)" json:"quote_asset"`
	BoughtQty    string `orm:"column(bought_qty)" json:"bought_qty"`
	RemainingQty string `orm:"column(remaining_qty)" json:"remaining_qty"`
	AvgCostQuote string `orm:"column(avg_cost_quote)" json:"avg_cost_quote"`
	RealizedPnL  string `orm:"column(realized_pnl)" json:"realized_pnl"`
	Status       string `orm:"column(status)" json:"status"`
	CreatedAt    int64  `orm:"column(created_at)" json:"created_at"`
	UpdatedAt    int64  `orm:"column(updated_at)" json:"updated_at"`
	ClosedAt     int64  `orm:"column(closed_at)" json:"closed_at"`
}

type SpotManagedOrder struct {
	ID              int64  `orm:"column(id)" json:"id"`
	PositionID      int64  `orm:"column(position_id)" json:"position_id"`
	Symbol          string `orm:"column(symbol)" json:"symbol"`
	Side            string `orm:"column(side)" json:"side"`
	OrderType       string `orm:"column(order_type)" json:"order_type"`
	ClientOrderID   string `orm:"column(client_order_id)" json:"client_order_id"`
	ExchangeOrderID string `orm:"column(exchange_order_id)" json:"exchange_order_id"`
	RequestedQty    string `orm:"column(requested_qty)" json:"requested_qty"`
	FilledQty       string `orm:"column(filled_qty)" json:"filled_qty"`
	Status          string `orm:"column(status)" json:"status"`
	CreatedAt       int64  `orm:"column(created_at)" json:"created_at"`
	UpdatedAt       int64  `orm:"column(updated_at)" json:"updated_at"`
}

type SpotTradeFill struct {
	ID              int64  `orm:"column(id)" json:"id"`
	PositionID      int64  `orm:"column(position_id)" json:"position_id"`
	OrderID         int64  `orm:"column(order_id)" json:"order_id"`
	ExchangeOrderID string `orm:"column(exchange_order_id)" json:"exchange_order_id"`
	TradeID         string `orm:"column(trade_id)" json:"trade_id"`
	Symbol          string `orm:"column(symbol)" json:"symbol"`
	Side            string `orm:"column(side)" json:"side"`
	Qty             string `orm:"column(qty)" json:"qty"`
	Price           string `orm:"column(price)" json:"price"`
	QuoteQty        string `orm:"column(quote_qty)" json:"quote_qty"`
	Commission      string `orm:"column(commission)" json:"commission"`
	CommissionAsset string `orm:"column(commission_asset)" json:"commission_asset"`
	TradeTime       int64  `orm:"column(trade_time)" json:"trade_time"`
}

type SpotStrategyTemplate struct {
	ID         int64  `orm:"column(id)" json:"id"`
	Name       string `orm:"column(name)" json:"name"`
	Technology string `orm:"column(technology)" json:"technology"`
	Strategy   string `orm:"column(strategy)" json:"strategy"`
	CreateTime int64  `orm:"column(create_time)" json:"create_time"`
	UpdateTime int64  `orm:"column(update_time)" json:"update_time"`
}

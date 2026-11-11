package models

// Spot Quant lifecycle records are deliberately separate from Futures.
// Decimal fields are kept as strings and are stored in SQL DECIMAL(36,18)
// columns by the versioned migration, not by Beego ORM. Do not RegisterModel
// these types: ORM synchronization would change exact decimals to VARCHAR.
type SpotManagedPosition struct {
	ID           int64  `json:"id"`
	Symbol       string `json:"symbol"`
	BaseAsset    string `json:"base_asset"`
	QuoteAsset   string `json:"quote_asset"`
	BoughtQty    string `json:"bought_qty"`
	RemainingQty string `json:"remaining_qty"`
	AvgCostQuote string `json:"avg_cost_quote"`
	RealizedPnL  string `json:"realized_pnl"`
	Status       string `json:"status"`
	CreatedAt    int64  `json:"created_at"`
	UpdatedAt    int64  `json:"updated_at"`
	ClosedAt     int64  `json:"closed_at"`
}

type SpotManagedOrder struct {
	ID              int64  `json:"id"`
	PositionID      int64  `json:"position_id"`
	Symbol          string `json:"symbol"`
	Side            string `json:"side"`
	OrderType       string `json:"order_type"`
	ClientOrderID   string `json:"client_order_id"`
	ExchangeOrderID string `json:"exchange_order_id"`
	RequestedQty    string `json:"requested_qty"`
	FilledQty       string `json:"filled_qty"`
	Status          string `json:"status"`
	CreatedAt       int64  `json:"created_at"`
	UpdatedAt       int64  `json:"updated_at"`
}

type SpotTradeFill struct {
	ID              int64  `json:"id"`
	PositionID      int64  `json:"position_id"`
	OrderID         int64  `json:"order_id"`
	ExchangeOrderID string `json:"exchange_order_id"`
	TradeID         string `json:"trade_id"`
	Symbol          string `json:"symbol"`
	Side            string `json:"side"`
	Qty             string `json:"qty"`
	Price           string `json:"price"`
	QuoteQty        string `json:"quote_qty"`
	Commission      string `json:"commission"`
	CommissionAsset string `json:"commission_asset"`
	TradeTime       int64  `json:"trade_time"`
}

type SpotStrategyTemplate struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Technology string `json:"technology"`
	Strategy   string `json:"strategy"`
	CreateTime int64  `json:"create_time"`
	UpdateTime int64  `json:"update_time"`
}

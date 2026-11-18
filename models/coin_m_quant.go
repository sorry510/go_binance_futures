package models

// COIN-M SQL columns for quantities, prices, and PnL are exact DECIMAL(36,18).
// The orm:"column(...)" tags are REQUIRED by Beego Raw.QueryRows to map SQL
// snake_case columns onto fields of these unregistered read-only DTOs.
// They do NOT register models or manage schema. NEVER call orm.RegisterModel on
// these DTOs: string fields would become VARCHAR instead of SQL DECIMAL.
// A contract quantity is a count of USD-denominated inverse contracts, not BTC or ETH.
type CoinMManagedPosition struct {
	ID                 int64  `orm:"column(id)" json:"id"`
	Symbol             string `orm:"column(symbol)" json:"symbol"`
	PositionSide       string `orm:"column(position_side)" json:"position_side"`
	MarginAsset        string `orm:"column(margin_asset)" json:"margin_asset"`
	ContractSize       string `orm:"column(contract_size)" json:"contract_size"`
	EntryPrice         string `orm:"column(entry_price)" json:"entry_price"`
	OpenedContracts    string `orm:"column(opened_contracts)" json:"opened_contracts"`
	RemainingContracts string `orm:"column(remaining_contracts)" json:"remaining_contracts"`
	RealizedPnL        string `orm:"column(realized_pnl)" json:"realized_pnl"`
	Status             string `orm:"column(status)" json:"status"`
	CreatedAt          int64  `orm:"column(created_at)" json:"created_at"`
	UpdatedAt          int64  `orm:"column(updated_at)" json:"updated_at"`
	ClosedAt           int64  `orm:"column(closed_at)" json:"closed_at"`
}

type CoinMManagedOrder struct {
	ID                 int64  `orm:"column(id)" json:"id"`
	PositionID         int64  `orm:"column(position_id)" json:"position_id"`
	Symbol             string `orm:"column(symbol)" json:"symbol"`
	PositionSide       string `orm:"column(position_side)" json:"position_side"`
	Intent             string `orm:"column(intent)" json:"intent"`
	Side               string `orm:"column(side)" json:"side"`
	OrderType          string `orm:"column(order_type)" json:"order_type"`
	ClientOrderID      string `orm:"column(client_order_id)" json:"client_order_id"`
	ExchangeOrderID    string `orm:"column(exchange_order_id)" json:"exchange_order_id"`
	RequestedContracts string `orm:"column(requested_contracts)" json:"requested_contracts"`
	FilledContracts    string `orm:"column(filled_contracts)" json:"filled_contracts"`
	Status             string `orm:"column(status)" json:"status"`
	CreatedAt          int64  `orm:"column(created_at)" json:"created_at"`
	UpdatedAt          int64  `orm:"column(updated_at)" json:"updated_at"`
}

type CoinMTradeFill struct {
	ID              int64  `orm:"column(id)" json:"id"`
	PositionID      int64  `orm:"column(position_id)" json:"position_id"`
	OrderID         int64  `orm:"column(order_id)" json:"order_id"`
	Symbol          string `orm:"column(symbol)" json:"symbol"`
	PositionSide    string `orm:"column(position_side)" json:"position_side"`
	Side            string `orm:"column(side)" json:"side"`
	ExchangeOrderID string `orm:"column(exchange_order_id)" json:"exchange_order_id"`
	TradeID         string `orm:"column(trade_id)" json:"trade_id"`
	Contracts       string `orm:"column(contracts)" json:"contracts"`
	Price           string `orm:"column(price)" json:"price"`
	RealizedPnL     string `orm:"column(realized_pnl)" json:"realized_pnl"`
	PnLAsset        string `orm:"column(pnl_asset)" json:"pnl_asset"`
	Commission      string `orm:"column(commission)" json:"commission"`
	CommissionAsset string `orm:"column(commission_asset)" json:"commission_asset"`
	TradeTime       int64  `orm:"column(trade_time)" json:"trade_time"`
}

type CoinMStrategyTemplate struct {
	ID         int64  `orm:"column(id)" json:"id"`
	Name       string `orm:"column(name)" json:"name"`
	Technology string `orm:"column(technology)" json:"technology"`
	Strategy   string `orm:"column(strategy)" json:"strategy"`
	CreateTime int64  `orm:"column(create_time)" json:"create_time"`
	UpdateTime int64  `orm:"column(update_time)" json:"update_time"`
}

// DeliverySymbols is an ORM model, so budget_usd is accessed via raw SQL
// to preserve DECIMAL storage and avoid converting an exact value to float.
type CoinMSymbolTradingConfig struct {
	Symbol    string `orm:"column(symbol)" json:"symbol"`
	Enabled   bool   `orm:"column(enabled)" json:"enabled"`
	BudgetUSD string `orm:"column(budget_usd)" json:"budget_usd"`
}

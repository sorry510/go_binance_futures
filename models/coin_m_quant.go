package models

// COIN-M SQL columns for quantities, prices, and PnL are exact DECIMAL(36,18).
// Do not register these DTOs with Beego ORM: string fields would become VARCHAR.
// A contract quantity is a count of USD-denominated inverse contracts, not BTC or ETH.
type CoinMManagedPosition struct {
	ID                 int64  `json:"id"`
	Symbol             string `json:"symbol"`
	PositionSide       string `json:"position_side"`
	MarginAsset        string `json:"margin_asset"`
	ContractSize       string `json:"contract_size"`
	EntryPrice         string `json:"entry_price"`
	OpenedContracts    string `json:"opened_contracts"`
	RemainingContracts string `json:"remaining_contracts"`
	RealizedPnL        string `json:"realized_pnl"`
	Status             string `json:"status"`
	CreatedAt          int64  `json:"created_at"`
	UpdatedAt          int64  `json:"updated_at"`
	ClosedAt           int64  `json:"closed_at"`
}

type CoinMManagedOrder struct {
	ID                 int64  `json:"id"`
	PositionID         int64  `json:"position_id"`
	Symbol             string `json:"symbol"`
	PositionSide       string `json:"position_side"`
	Intent             string `json:"intent"`
	Side               string `json:"side"`
	OrderType          string `json:"order_type"`
	ClientOrderID      string `json:"client_order_id"`
	ExchangeOrderID    string `json:"exchange_order_id"`
	RequestedContracts string `json:"requested_contracts"`
	FilledContracts    string `json:"filled_contracts"`
	Status             string `json:"status"`
	CreatedAt          int64  `json:"created_at"`
	UpdatedAt          int64  `json:"updated_at"`
}

type CoinMTradeFill struct {
	ID              int64  `json:"id"`
	PositionID      int64  `json:"position_id"`
	OrderID         int64  `json:"order_id"`
	Symbol          string `json:"symbol"`
	PositionSide    string `json:"position_side"`
	Side            string `json:"side"`
	ExchangeOrderID string `json:"exchange_order_id"`
	TradeID         string `json:"trade_id"`
	Contracts       string `json:"contracts"`
	Price           string `json:"price"`
	RealizedPnL     string `json:"realized_pnl"`
	PnLAsset        string `json:"pnl_asset"`
	Commission      string `json:"commission"`
	CommissionAsset string `json:"commission_asset"`
	TradeTime       int64  `json:"trade_time"`
}

type CoinMStrategyTemplate struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Technology string `json:"technology"`
	Strategy   string `json:"strategy"`
	CreateTime int64  `json:"create_time"`
	UpdateTime int64  `json:"update_time"`
}

// DeliverySymbols is an ORM model, so budget_usd is accessed via raw SQL
// to preserve DECIMAL storage and avoid converting an exact value to float.
type CoinMSymbolTradingConfig struct {
	Symbol    string `json:"symbol"`
	Enabled   bool   `json:"enabled"`
	BudgetUSD string `json:"budget_usd"`
}

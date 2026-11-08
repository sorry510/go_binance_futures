package models

// FuturesTradeFill is a lossless transfer object for the immutable exchange
// trade ledger. Decimal values stay as strings until persisted by parameterized
// SQL into DECIMAL(36,18) columns. Deliberately do not register this struct in
// Beego ORM: its string fields would make RunSyncdb convert DECIMAL to VARCHAR.
type FuturesTradeFill struct {
	Market          string `json:"market"`
	AccountID       string `json:"account_id"`
	Symbol          string `json:"symbol"`
	OrderID         string `json:"order_id"`
	TradeID         string `json:"trade_id"`
	PositionID      *int64 `json:"position_id,omitempty"`
	ClientOrderID   string `json:"client_order_id,omitempty"`
	Intent          string `json:"intent,omitempty"`
	PositionSide    string `json:"position_side,omitempty"`
	Qty             string `json:"qty"`
	Price           string `json:"price"`
	Commission      string `json:"commission"`
	CommissionAsset string `json:"commission_asset"`
	RealizedPnL     string `json:"realized_pnl"`
	PnLAsset        string `json:"pnl_asset"`
	TradeTime       int64  `json:"trade_time"`
	// REST is authoritative for fee, PnL and exchange event timestamp.
	// WS trades remain provisional until matched by userTrades.
	RESTVerified bool `json:"rest_verified"`
}

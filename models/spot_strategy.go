package models

// The following DTOs are deliberately not ORM-registered. The matching SQL
// tables use DECIMAL(36,18); Beego ORM's string fields would rewrite those to
// VARCHAR during RunSyncdb. Future S2/S3 queries use parameterized SQL and
// these exact decimal strings rather than converting through float64.
type SpotStrategyHolding struct {
	ID           int64
	AccountID    string
	Owner        string
	Symbol       string
	BaseAsset    string
	QuoteAsset   string
	OwnedQty     string
	AvgCostQuote string
	Status       string
	ActiveSlot   *string
	CreatedAt    int64
	UpdatedAt    int64
	ClosedAt     int64
}

type SpotStrategyOrder struct {
	ID               int64
	HoldingID        *int64
	AccountID        string
	Owner            string
	Symbol           string
	Side             string
	OrderType        string
	Intent           string
	ClientOrderID    string
	ExchangeOrderID  *string
	SignalKey        string
	PendingSlot      *string
	RequestedQty     string
	FilledQty        string
	ExecutedQuoteQty string
	ReservedQty      string
	Status           string
	LastError        string
	CreatedAt        int64
	UpdatedAt        int64
	LastCheckedAt    int64
}

type SpotStrategyFill struct {
	AccountID       string
	Symbol          string
	OrderID         string
	TradeID         string
	HoldingID       *int64
	ClientOrderID   string
	Side            string
	Qty             string
	Price           string
	QuoteQty        string
	Commission      string
	CommissionAsset string
	TradeTime       int64
	RESTVerified    bool
}

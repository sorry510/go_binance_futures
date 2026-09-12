package models

type FuturesManagedPosition struct {
	ID               int64   `orm:"column(id);auto" json:"id"`
	Owner            string  `orm:"column(owner);size(32);index" json:"owner"`
	Symbol           string  `orm:"column(symbol);size(32);index" json:"symbol"`
	PositionSide     string  `orm:"column(position_side);size(8);index" json:"position_side"`
	ManagedQty       float64 `orm:"column(managed_qty);digits(30);decimals(12);default(0)" json:"managed_qty"`
	EntryPrice       float64 `orm:"column(entry_price);digits(30);decimals(12);default(0)" json:"entry_price"`
	Status           string  `orm:"column(status);size(32);index" json:"status"`
	SourceRef        string  `orm:"column(source_ref);size(128);null;index" json:"source_ref,omitempty"`
	CreatedAt        int64   `orm:"column(created_at);index" json:"created_at"`
	UpdatedAt        int64   `orm:"column(updated_at);index" json:"updated_at"`
	ClosedAt         int64   `orm:"column(closed_at);default(0);index" json:"closed_at"`
	LastReconciledAt int64   `orm:"column(last_reconciled_at);default(0);index" json:"last_reconciled_at"`
}

func (*FuturesManagedPosition) TableName() string { return "futures_managed_positions" }

type FuturesManagedOrder struct {
	ID               int64   `orm:"column(id);auto" json:"id"`
	Owner            string  `orm:"column(owner);size(32);index" json:"owner"`
	Symbol           string  `orm:"column(symbol);size(32);index" json:"symbol"`
	PositionSide     string  `orm:"column(position_side);size(8);index" json:"position_side"`
	Intent           string  `orm:"column(intent);size(24);index" json:"intent"`
	ClientOrderID    string  `orm:"column(client_order_id);size(64);unique" json:"client_order_id"`
	ExchangeOrderID  string  `orm:"column(exchange_order_id);size(64);null;index" json:"exchange_order_id,omitempty"`
	RequestedQty     float64 `orm:"column(requested_qty);digits(30);decimals(12);default(0)" json:"requested_qty"`
	FilledQty        float64 `orm:"column(filled_qty);digits(30);decimals(12);default(0)" json:"filled_qty"`
	OrderType        string  `orm:"column(order_type);size(24)" json:"order_type"`
	Status           string  `orm:"column(status);size(32);index" json:"status"`
	SourceRef        string  `orm:"column(source_ref);size(128);null;index" json:"source_ref,omitempty"`
	CreatedAt        int64   `orm:"column(created_at);index" json:"created_at"`
	UpdatedAt        int64   `orm:"column(updated_at);index" json:"updated_at"`
	LastReconciledAt int64   `orm:"column(last_reconciled_at);default(0);index" json:"last_reconciled_at"`
}

func (*FuturesManagedOrder) TableName() string { return "futures_managed_orders" }

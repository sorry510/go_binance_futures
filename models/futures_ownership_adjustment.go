package models

// FuturesOwnershipAdjustment is an append-only audit of explicit web-admin
// ownership repairs. It is not an exchange order or a fill.
type FuturesOwnershipAdjustment struct {
	ID                int64   `orm:"column(id);auto" json:"id"`
	AccountID         string  `orm:"column(account_id);size(16);index" json:"account_id"`
	Symbol            string  `orm:"column(symbol);size(32);index" json:"symbol"`
	PositionSide      string  `orm:"column(position_side);size(8)" json:"position_side"`
	Owner             string  `orm:"column(owner);size(32)" json:"owner"`
	AccountQty        float64 `orm:"column(account_qty);digits(30);decimals(12)" json:"account_qty"`
	ManagedQty        float64 `orm:"column(managed_qty);digits(30);decimals(12)" json:"managed_qty"`
	PreviousOwner     string  `orm:"column(previous_owner);size(32);null" json:"previous_owner"`
	PreviousStatus    string  `orm:"column(previous_status);size(32);null" json:"previous_status"`
	PreviousSourceRef string  `orm:"column(previous_source_ref);size(128);null" json:"previous_source_ref"`
	PreviousQty       float64 `orm:"column(previous_qty);digits(30);decimals(12)" json:"previous_qty"`
	PositionID        int64   `orm:"column(position_id);index" json:"position_id"`
	Operator          string  `orm:"column(operator);size(128)" json:"operator"`
	Reason            string  `orm:"column(reason);size(512)" json:"reason"`
	CreatedAt         int64   `orm:"column(created_at);index" json:"created_at"`
}

func (*FuturesOwnershipAdjustment) TableName() string { return "futures_ownership_adjustments" }

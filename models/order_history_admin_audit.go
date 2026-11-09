package models

// OrderHistoryAdminAudit records presentation-only status/completeness edits.
// Physical cycle deletion removes these records along with the cycle.
type OrderHistoryAdminAudit struct {
	ID              int64  `orm:"column(id);auto" json:"id"`
	AccountID       string `orm:"column(account_id);size(16);index" json:"account_id"`
	PositionID      int64  `orm:"column(position_id);index" json:"position_id"`
	Action          string `orm:"column(action);size(24)" json:"action"`
	OldStatus       string `orm:"column(old_status);size(32);null" json:"old_status"`
	NewStatus       string `orm:"column(new_status);size(32);null" json:"new_status"`
	OldCompleteness string `orm:"column(old_completeness);size(32);null" json:"old_completeness"`
	NewCompleteness string `orm:"column(new_completeness);size(32);null" json:"new_completeness"`
	Operator        string `orm:"column(operator);size(128)" json:"operator"`
	Reason          string `orm:"column(reason);size(512)" json:"reason"`
	BatchDigest     string `orm:"column(batch_digest);size(64);index" json:"batch_digest"`
	CreatedAt       int64  `orm:"column(created_at);index" json:"created_at"`
}

func (*OrderHistoryAdminAudit) TableName() string { return "order_history_admin_audits" }

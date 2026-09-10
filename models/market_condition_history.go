package models

// MarketConditionHistory stores live MarketCondition updates and historical
// backfill values used by deterministic backtests. Rows are append-only; the
// backfill path skips any hour that already contains a persisted value.
type MarketConditionHistory struct {
	ID              int64 `orm:"column(id);auto" json:"id"`
	ConfigID        int64 `orm:"column(config_id);index" json:"config_id"`
	MarketCondition int   `orm:"column(market_condition);index" json:"market_condition"`
	CreatedAt       int64 `orm:"column(created_at);index" json:"created_at"`
}

func (*MarketConditionHistory) TableName() string { return "market_condition_histories" }

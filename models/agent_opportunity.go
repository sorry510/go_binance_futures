package models

// AgentOpportunity is the durable user-facing result of an automatically
// discovered market opportunity. The full AI analysis remains in AgentTask;
// this table stores only lifecycle, source linkage and list-friendly summary.
type AgentOpportunity struct {
	ID              int64   `orm:"column(id);auto" json:"id"`
	OpportunityID   string  `orm:"column(opportunity_id);size(64);unique" json:"opportunity_id"`
	Symbol          string  `orm:"column(symbol);size(32);index" json:"symbol"`
	Direction       string  `orm:"column(direction);size(16);index" json:"direction"`
	SourceType      string  `orm:"column(source_type);size(48);index" json:"source_type"`
	SourceID        string  `orm:"column(source_id);size(128);index" json:"source_id"`
	AnalysisTaskID  string  `orm:"column(analysis_task_id);size(64);null;index" json:"analysis_task_id,omitempty"`
	Summary         string  `orm:"column(summary);type(text);null" json:"summary,omitempty"`
	Confidence      float64 `orm:"column(confidence);digits(8);decimals(6);default(0)" json:"confidence"`
	MarketCondition int     `orm:"column(market_condition);default(0);index" json:"market_condition"`
	AnalysisStatus  string  `orm:"column(analysis_status);size(32);index" json:"analysis_status"`
	AnalysisError   string  `orm:"column(analysis_error);type(text);null" json:"analysis_error,omitempty"`
	Status          string  `orm:"column(status);size(16);index" json:"status"`
	ExpiresAt       int64   `orm:"column(expires_at);index" json:"expires_at"`
	ReviewedAt      int64   `orm:"column(reviewed_at);default(0);index" json:"reviewed_at,omitempty"`
	CreatedAt       int64   `orm:"column(created_at);index" json:"created_at"`
	UpdatedAt       int64   `orm:"column(updated_at);index" json:"updated_at"`
}

func (*AgentOpportunity) TableName() string { return "agent_opportunities" }

func (*AgentOpportunity) TableUnique() [][]string {
	return [][]string{{"SourceType", "SourceID"}}
}

func (*AgentOpportunity) TableIndex() [][]string {
	return [][]string{{"Symbol", "SourceType", "CreatedAt"}}
}

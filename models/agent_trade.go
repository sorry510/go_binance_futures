package models

type AgentTradeProposal struct {
	ID                int64   `orm:"column(id);auto" json:"id"`
	ProposalID        string  `orm:"column(proposal_id);size(64);unique" json:"proposal_id"`
	SourceTaskID      string  `orm:"column(source_task_id);size(64);index" json:"source_task_id"`
	SourceSkill       string  `orm:"column(source_skill);size(96);index" json:"source_skill"`
	ContentHash       string  `orm:"column(content_hash);size(64);index" json:"content_hash"`
	Symbol            string  `orm:"column(symbol);size(32);index" json:"symbol"`
	Side              string  `orm:"column(side);size(8);index" json:"side"`
	EntryCondition    string  `orm:"column(entry_condition);type(text)" json:"entry_condition"`
	EntryZonesJSON    string  `orm:"column(entry_zones_json);type(text)" json:"entry_zones_json"`
	EntryLow          float64 `orm:"column(entry_low);digits(30);decimals(12);default(0)" json:"entry_low"`
	EntryHigh         float64 `orm:"column(entry_high);digits(30);decimals(12);default(0)" json:"entry_high"`
	StopLoss          float64 `orm:"column(stop_loss);digits(30);decimals(12);default(0)" json:"stop_loss"`
	TakeProfitsJSON   string  `orm:"column(take_profits_json);type(text)" json:"take_profits_json"`
	InvalidationsJSON string  `orm:"column(invalidations_json);type(text)" json:"invalidations_json"`
	EvidenceJSON      string  `orm:"column(evidence_json);type(text)" json:"evidence_json"`
	MarketCondition   int     `orm:"column(market_condition);default(0);index" json:"market_condition"`
	Status            string  `orm:"column(status);size(32);index" json:"status"`
	RiskStatus        string  `orm:"column(risk_status);size(16);index" json:"risk_status"`
	RiskJSON          string  `orm:"column(risk_json);type(text);null" json:"risk_json,omitempty"`
	RiskCheckedAt     int64   `orm:"column(risk_checked_at);default(0);index" json:"risk_checked_at"`
	Quantity          float64 `orm:"column(quantity);digits(30);decimals(12);default(0)" json:"quantity"`
	ReferencePrice    float64 `orm:"column(reference_price);digits(30);decimals(12);default(0)" json:"reference_price"`
	Leverage          int     `orm:"column(leverage);default(0)" json:"leverage"`
	NotionalUSDT      float64 `orm:"column(notional_usdt);digits(30);decimals(8);default(0)" json:"notional_usdt"`
	RiskUSDT          float64 `orm:"column(risk_usdt);digits(30);decimals(8);default(0)" json:"risk_usdt"`
	ApprovedBy        string  `orm:"column(approved_by);size(64);null" json:"approved_by,omitempty"`
	RejectedReason    string  `orm:"column(rejected_reason);type(text);null" json:"rejected_reason,omitempty"`
	CreatedAt         int64   `orm:"column(created_at);index" json:"created_at"`
	UpdatedAt         int64   `orm:"column(updated_at);index" json:"updated_at"`
	ExpiresAt         int64   `orm:"column(expires_at);index" json:"expires_at"`
	ApprovedAt        int64   `orm:"column(approved_at);default(0);index" json:"approved_at"`
	RejectedAt        int64   `orm:"column(rejected_at);default(0);index" json:"rejected_at"`
	ExecutedAt        int64   `orm:"column(executed_at);default(0);index" json:"executed_at"`
}

func (*AgentTradeProposal) TableName() string { return "agent_trade_proposals" }

type AgentTradeExecution struct {
	ID              int64   `orm:"column(id);auto" json:"id"`
	ProposalID      string  `orm:"column(proposal_id);size(64);unique" json:"proposal_id"`
	IdempotencyKey  string  `orm:"column(idempotency_key);size(64);unique" json:"idempotency_key"`
	ClientOrderID   string  `orm:"column(client_order_id);size(64);unique" json:"client_order_id"`
	ExchangeOrderID string  `orm:"column(exchange_order_id);size(64);null;index" json:"exchange_order_id,omitempty"`
	Status          string  `orm:"column(status);size(32);index" json:"status"`
	Symbol          string  `orm:"column(symbol);size(32);index" json:"symbol"`
	Side            string  `orm:"column(side);size(8);index" json:"side"`
	OrderType       string  `orm:"column(order_type);size(16)" json:"order_type"`
	Quantity        float64 `orm:"column(quantity);digits(30);decimals(12)" json:"quantity"`
	ReferencePrice  float64 `orm:"column(reference_price);digits(30);decimals(12)" json:"reference_price"`
	AveragePrice    float64 `orm:"column(average_price);digits(30);decimals(12);default(0)" json:"average_price"`
	Leverage        int     `orm:"column(leverage);default(1)" json:"leverage"`
	Error           string  `orm:"column(error);type(text);null" json:"error,omitempty"`
	CreatedAt       int64   `orm:"column(created_at);index" json:"created_at"`
	UpdatedAt       int64   `orm:"column(updated_at);index" json:"updated_at"`
	SubmittedAt     int64   `orm:"column(submitted_at);default(0);index" json:"submitted_at"`
	CompletedAt     int64   `orm:"column(completed_at);default(0);index" json:"completed_at"`
}

func (*AgentTradeExecution) TableName() string { return "agent_trade_executions" }

type AgentTradeAudit struct {
	ID         int64  `orm:"column(id);auto" json:"id"`
	ProposalID string `orm:"column(proposal_id);size(64);index" json:"proposal_id"`
	Event      string `orm:"column(event);size(48);index" json:"event"`
	Status     string `orm:"column(status);size(32);index" json:"status"`
	Actor      string `orm:"column(actor);size(64);null;index" json:"actor,omitempty"`
	DetailJSON string `orm:"column(detail_json);type(text);null" json:"detail_json,omitempty"`
	CreatedAt  int64  `orm:"column(created_at);index" json:"created_at"`
}

func (*AgentTradeAudit) TableName() string { return "agent_trade_audits" }

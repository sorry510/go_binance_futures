package models

type AgentMemory struct {
	ID            int64   `orm:"column(id);auto" json:"id"`
	Type          string  `orm:"column(type);size(32);index" json:"type"`
	ScopeUser     string  `orm:"column(scope_user);size(64);null;index" json:"scope_user"`
	ScopeSkill    string  `orm:"column(scope_skill);size(96);null;index" json:"scope_skill"`
	ScopeSymbol   string  `orm:"column(scope_symbol);size(64);null;index" json:"scope_symbol"`
	ScopeStrategy string  `orm:"column(scope_strategy);size(128);null;index" json:"scope_strategy"`
	SourceTaskID  string  `orm:"column(source_task_id);size(64);null;index" json:"source_task_id"`
	Confidence    float64 `orm:"column(confidence);digits(5);decimals(4);default(1)" json:"confidence"`
	Status        string  `orm:"column(status);size(32);index" json:"status"`
	Content       string  `orm:"column(content);type(text)" json:"content"`
	ContentHash   string  `orm:"column(content_hash);size(64);index" json:"content_hash"`
	CreatedAt     int64   `orm:"column(created_at);index" json:"created_at"`
	UpdatedAt     int64   `orm:"column(updated_at);index" json:"updated_at"`
	ExpiresAt     int64   `orm:"column(expires_at);index" json:"expires_at"`
}

func (*AgentMemory) TableName() string { return "agent_memories" }

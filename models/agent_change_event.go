package models

type AgentChangeEvent struct {
	ID          int64  `orm:"column(id);auto" json:"id"`
	Category    string `orm:"column(category);size(32);index" json:"category"`
	EntityType  string `orm:"column(entity_type);size(32);index" json:"entity_type"`
	EntityID    int64  `orm:"column(entity_id);default(0);index" json:"entity_id,omitempty"`
	EntityName  string `orm:"column(entity_name);size(191);index" json:"entity_name"`
	ChangeType  string `orm:"column(change_type);size(64);index" json:"change_type"`
	FromVersion string `orm:"column(from_version);size(128);null" json:"from_version,omitempty"`
	ToVersion   string `orm:"column(to_version);size(128);null" json:"to_version,omitempty"`
	BeforeHash  string `orm:"column(before_hash);size(64);null" json:"before_hash,omitempty"`
	AfterHash   string `orm:"column(after_hash);size(64);null" json:"after_hash,omitempty"`
	Status      string `orm:"column(status);size(32);index" json:"status"`
	DetailJSON  string `orm:"column(detail_json);type(text);null" json:"detail_json,omitempty"`
	CreatedAt   int64  `orm:"column(created_at);index" json:"created_at"`
}

func (*AgentChangeEvent) TableName() string { return "agent_change_events" }

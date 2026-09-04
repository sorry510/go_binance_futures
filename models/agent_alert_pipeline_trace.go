package models

// AgentAlertPipelineTrace persists the Event -> Signal -> Agent Task -> Notification audit chain.
type AgentAlertPipelineTrace struct {
	ID             int64  `orm:"column(id);auto" json:"id"`
	EventID        string `orm:"column(event_id);size(96);index" json:"event_id"`
	SignalID       string `orm:"column(signal_id);size(96);unique" json:"signal_id"`
	TaskID         string `orm:"column(task_id);size(96);index;null" json:"task_id,omitempty"`
	NotificationID int64  `orm:"column(notification_id);index;default(0)" json:"notification_id,omitempty"`
	Symbol         string `orm:"column(symbol);size(32);index" json:"symbol"`
	Type           string `orm:"column(signal_type);size(64);index" json:"type"`
	Severity       string `orm:"column(severity);size(16);index" json:"severity"`
	Action         string `orm:"column(action);size(32);index;null" json:"action,omitempty"`
	Status         string `orm:"column(status);size(32);index" json:"status"`
	Fallback       int    `orm:"column(fallback);default(0);index" json:"fallback"`
	Error          string `orm:"column(error);type(text);null" json:"error,omitempty"`
	CreatedAt      int64  `orm:"column(created_at);index" json:"created_at"`
	UpdatedAt      int64  `orm:"column(updated_at);index" json:"updated_at"`
}

func (*AgentAlertPipelineTrace) TableName() string { return "agent_alert_pipeline_traces" }

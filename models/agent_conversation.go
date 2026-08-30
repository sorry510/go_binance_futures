package models

type AgentConversation struct {
	ID        string `orm:"column(id);pk;size(64)" json:"id"`
	Skill     string `orm:"column(skill);size(64);index" json:"skill"`
	Status    string `orm:"column(status);size(32);index" json:"status"`
	CreatedAt int64  `orm:"column(created_at);index" json:"created_at"`
	UpdatedAt int64  `orm:"column(updated_at);index" json:"updated_at"`
	ClosedAt  int64  `orm:"column(closed_at);index" json:"closed_at"`
}

func (*AgentConversation) TableName() string { return "agent_conversations" }

type AgentConversationMessage struct {
	ID             int64  `orm:"column(id);auto" json:"id"`
	ConversationID string `orm:"column(conversation_id);size(64);index" json:"conversation_id"`
	TaskID         string `orm:"column(task_id);size(64);index" json:"task_id"`
	Sequence       int    `orm:"column(sequence);index" json:"sequence"`
	Role           string `orm:"column(role);size(32)" json:"role"`
	Content        string `orm:"column(content);type(text)" json:"content"`
	CreatedAt      int64  `orm:"column(created_at);index" json:"created_at"`
}

func (*AgentConversationMessage) TableName() string { return "agent_conversation_messages" }

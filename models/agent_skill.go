package models

type AgentSkill struct {
	ID              int64  `orm:"column(id);auto" json:"id"`
	Name            string `orm:"column(name);size(96);unique" json:"name"`
	DisplayName     string `orm:"column(display_name);size(128)" json:"display_name"`
	Description     string `orm:"column(description);type(text)" json:"description"`
	Type            string `orm:"column(type);size(16);default(native)" json:"type"`
	ActiveVersionID int64  `orm:"column(active_version_id);default(0);index" json:"active_version_id"`
	Enabled         int    `orm:"column(enabled);default(1)" json:"enabled"`
	ChatEnabled     int    `orm:"column(chat_enabled);default(-1)" json:"chat_enabled"`
	CreatedAt       int64  `orm:"column(created_at)" json:"created_at"`
	UpdatedAt       int64  `orm:"column(updated_at)" json:"updated_at"`
	Deleted         int    `orm:"column(deleted);default(0)" json:"-"`
}

func (*AgentSkill) TableName() string {
	return "agent_skills"
}

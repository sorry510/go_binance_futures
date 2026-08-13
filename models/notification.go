package models

type Notification struct {
	ID         int64  `orm:"auto;column(id)" json:"id"`
	Title      string `orm:"size(255);column(title)" json:"title"`
	Content    string `orm:"type(text);column(content)" json:"content"`
	Module     string `orm:"size(64);index;column(module)" json:"module"`
	Level      string `orm:"size(20);default(info);column(level)" json:"level"`
	IsRead     int    `orm:"default(0);index;column(is_read)" json:"is_read"`
	CreateTime int64  `orm:"index;column(create_time)" json:"create_time"`
	ReadTime   int64  `orm:"default(0);column(read_time)" json:"read_time"`
}

func (notification *Notification) TableName() string {
	return "notifications"
}

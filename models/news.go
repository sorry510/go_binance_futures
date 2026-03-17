package models

type News struct {
	ID          int64  `orm:"column(id)" json:"id"`
	Type        string `orm:"column(type);size(32)" json:"type"`
	Source      string `orm:"column(source);size(64)" json:"source"`
	ExternalID  string `orm:"column(external_id);size(128)" json:"external_id"`
	Title       string `orm:"column(title);type(text)" json:"title"`
	Content     string `orm:"column(content);type(text)" json:"content"`
	Url         string `orm:"column(url);size(512)" json:"url"`
	PublishedAt int64  `orm:"column(published_at)" json:"published_at"`
	CreateTime  int64  `orm:"column(createTime)" json:"createTime"`
	UpdateTime  int64  `orm:"column(updateTime)" json:"updateTime"`
}

func (u *News) TableName() string {
	return "news"
}

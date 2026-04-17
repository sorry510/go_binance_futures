package models

type FuturesMarketNoticeLog struct {
	ID               int64   `orm:"column(id)" json:"id"`
	Symbol           string  `orm:"column(symbol);size(32)" json:"symbol"`
	NoticeType       string  `orm:"column(notice_type);size(32)" json:"notice_type"` // price_change, fast_move
	Window           string  `orm:"column(window_size);size(16)" json:"window"`
	Direction        string  `orm:"column(direction);size(16)" json:"direction"` // up, down
	Source           string  `orm:"column(source);size(32)" json:"source"`       // ws_all_market_ticker
	Price            float64 `orm:"column(price)" json:"price"`
	BasePrice        float64 `orm:"column(base_price)" json:"base_price"`
	ChangePercent    float64 `orm:"column(change_percent)" json:"change_percent"`
	ThresholdPercent float64 `orm:"column(threshold_percent)" json:"threshold_percent"`
	Content          string  `orm:"column(content);type(text)" json:"content"`
	CreateTime       int64   `orm:"column(createTime)" json:"createTime"`
	UpdateTime       int64   `orm:"column(updateTime)" json:"updateTime"`
}

func (u *FuturesMarketNoticeLog) TableName() string {
	return "futures_market_notice_log"
}

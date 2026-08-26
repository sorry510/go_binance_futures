package models

type Notification struct {
	ID                int64   `orm:"auto;column(id)" json:"id"`
	Title             string  `orm:"size(255);column(title)" json:"title"`
	Content           string  `orm:"type(text);column(content)" json:"content"`
	Module            string  `orm:"size(64);index;column(module)" json:"module"`
	Level             string  `orm:"size(20);default(info);column(level)" json:"level"`
	EventType         string  `orm:"size(64);index;null;column(event_type)" json:"event_type"`
	Symbol            string  `orm:"size(32);index;null;column(symbol)" json:"symbol"`
	LiquidationSide   string  `orm:"size(16);index;null;column(liquidation_side)" json:"liquidation_side"`
	AggregateNotional float64 `orm:"null;column(aggregate_notional);digits(30);decimals(8)" json:"aggregate_notional"`
	OrderCount        int     `orm:"null;column(order_count)" json:"order_count"`
	WindowStart       int64   `orm:"null;column(window_start)" json:"window_start"`
	WindowEnd         int64   `orm:"null;column(window_end)" json:"window_end"`
	IsRead            int     `orm:"default(0);index;column(is_read)" json:"is_read"`
	CreateTime        int64   `orm:"index;column(create_time)" json:"create_time"`
	ReadTime          int64   `orm:"default(0);column(read_time)" json:"read_time"`
}

func (notification *Notification) TableName() string {
	return "notifications"
}

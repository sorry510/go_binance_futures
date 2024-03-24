package models

type OrderTable struct {
	Order
	SideText string `orm:"column(sideText)" json:"sideText"`
	PositionText string `orm:"column(positionText)" json:"positionText"`
	UpdateDate string `orm:"column(updateDate)" json:"updateDate"`
}

type Order struct {
	ID int64 `orm:"column(id)" json:"id,omitempty"`
	Symbol string `orm:"column(symbol)" json:"symbol"`
	Amount string `orm:"column(amount)" json:"amount"`
	Avg_price string `orm:"column(avg_price)" json:"avg_price"`
	Inexact_profit string `orm:"column(inexact_profit)" json:"inexact_profit"`
	Side string `orm:"column(side)" json:"side"`
	PositionSide string `orm:"column(positionSide)" json:"positionSide"`
	UpdateTime int64 `orm:"column(updateTime)" json:"updateTime"`
}

type Symbols struct {
	ID int64 `orm:"column(id)" json:"id"`
	Symbol string `orm:"column(symbol)" json:"symbol"`
	Quantity string `orm:"column(quantity)" json:"quantity"`
	PercentChange float64 `orm:"column(percentChange)" json:"percentChange"`
	Close string `orm:"column(close)" json:"close"`
	Open string `orm:"column(open)" json:"open"`
	Low string `orm:"column(low)" json:"low"`
	Enable int `orm:"column(enable)" json:"enable"`
	UpdateTime int64 `orm:"column(updateTime)" json:"updateTime"`
	LastClose string `orm:"column(lastClose)" json:"lastClose"`
	LastUpdateTime int64 `orm:"column(lastUpdateTime)" json:"lastUpdateTime"`
}

func (u *Order) TableName() string {
    return "order"
}

func (u *Symbols) TableName() string {
    return "symbols"
}

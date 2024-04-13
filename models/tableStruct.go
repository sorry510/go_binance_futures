package models

// 切记需要注册model后才能使用
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
	PercentChange float64 `orm:"column(percentChange)" json:"percentChange"`
	Close string `orm:"column(close)" json:"close"`
	Open string `orm:"column(open)" json:"open"`
	Low string `orm:"column(low)" json:"low"`
	Enable int `orm:"column(enable)" json:"enable"`
	UpdateTime int64 `orm:"column(updateTime)" json:"updateTime"`
	LastClose string `orm:"column(lastClose)" json:"lastClose"`
	LastUpdateTime int64 `orm:"column(lastUpdateTime)" json:"lastUpdateTime"`
	
	Leverage int64 `orm:"column(leverage)" json:"leverage"` // 合约倍数
	MarginType string `orm:"column(marginType)" json:"marginType"` // 杠杆类型 ISOLATED(逐仓), CROSSED(全仓)
	TickSize string `orm:"column(tickSize)" json:"tickSize"` // 交易金额精度
	StepSize string `orm:"column(stepSize)" json:"stepSize"` // 交易数量精度
	Usdt string `orm:"column(usdt)" json:"usdt"` // 交易金额
	Profit string `orm:"column(profit)" json:"profit"` // 盈利率
	Loss string `orm:"column(loss)" json:"loss"` // 损失率
}

type NewSymbols struct {
	ID int64 `orm:"column(id)" json:"id"`
	Symbol string `orm:"column(symbol)" json:"symbol"`
	Enable int `orm:"column(enable)" json:"enable"`
	CreateTime int64 `orm:"column(createTime)" json:"createTime"`
	UpdateTime int64 `orm:"column(updateTime)" json:"updateTime"`
	Type int64 `orm:"column(type)" json:"type"` // 1:币币交易 2:合约交易
	
	Leverage int64 `orm:"column(leverage)" json:"leverage"` // 合约倍数
	MarginType string `orm:"column(marginType)" json:"marginType"` // 杠杆类型 ISOLATED(逐仓), CROSSED(全仓)
	TickSize string `orm:"column(tickSize)" json:"tickSize"` // 交易金额精度
	StepSize string `orm:"column(stepSize)" json:"stepSize"` // 交易数量精度
	Usdt string `orm:"column(usdt)" json:"usdt"` // 交易金额
}

func (u *Order) TableName() string {
    return "order"
}

func (u *Symbols) TableName() string {
    return "symbols"
}

func (u *NewSymbols) TableName() string {
    return "new_symbols"
}

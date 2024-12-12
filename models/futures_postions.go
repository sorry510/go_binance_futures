package models

type FuturesPosition struct {
	ID int64 `orm:"column(id)" json:"id"`
	Symbol string `orm:"column(symbol)" json:"symbol"`
	Side string `orm:"column(side)" json:"side"` // 持仓方向, BOTH, LONG, SHORT
	Amount string `orm:"column(amount)" json:"amount"` // 持仓数量
	MarginType string `orm:"column(margin_type)" json:"margin_type"` // isolated, cross
	Leverage int64 `orm:"column(leverage)" json:"leverage"` // 合约倍数
	IsolatedWallet string `orm:"column(isolated_wallet)" json:"isolated_wallet"` // 逐仓钱包余额
	EntryPrice string `orm:"column(entry_price)" json:"entry_price"` // 开仓价格
	MarkPrice string `orm:"column(mark_price)" json:"mark_price"` // 标记价格
	UnrealizedProfit string `orm:"column(unrealized_profit)" json:"unrealized_profit"` // 未实现盈亏
	AccumulatedRealized string `orm:"column(accumulated_realized)" json:"accumulated_realized"` // (费前)累计实现损益
	MaintenanceMarginRequired string `orm:"column(maintenance_margin_required)" json:"maintenance_margin_required"` // 维持保证金
	CreateTime int64 `orm:"column(createTime)" json:"createTime"`
	UpdateTime int64 `orm:"column(updateTime)" json:"updateTime"`
}

func (u *FuturesPosition) TableName() string {
    return "futures_positions"
}
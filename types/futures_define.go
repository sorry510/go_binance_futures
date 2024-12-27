package types

type FuturesPosition struct {
	Symbol string `orm:"column(symbol)" json:"symbol"`
	Side string `orm:"column(side)" json:"side"` // 持仓方向, BOTH, LONG, SHORT
	Amount string `orm:"column(amount)" json:"amount"` // 持仓数量
	MarginType string `orm:"column(margin_type)" json:"margin_type"` // isolated, cross
	Leverage int64 `orm:"column(leverage)" json:"leverage"` // 合约倍数
	IsolatedWallet string `orm:"column(isolated_wallet)" json:"isolated_wallet"` // 逐仓钱包余额
	EntryPrice string `orm:"column(entry_price)" json:"entry_price"` // 开仓价格
	MarkPrice string `orm:"column(mark_price)" json:"mark_price"` // 标记价格
	UnrealizedProfit string `orm:"column(unrealized_profit)" json:"unrealized_profit"` // 未实现盈亏
}

// 自定义策略中的持仓信息
type FuturesPositionCode struct {
	Symbol string `orm:"column(symbol)" json:"symbol"`
	Side string `orm:"column(side)" json:"side"` // 持仓方向, BOTH, LONG, SHORT
	Amount float64 `orm:"column(amount)" json:"amount"` // 持仓数量
	Leverage int64 `orm:"column(leverage)" json:"leverage"` // 合约倍数
	EntryPrice float64 `orm:"column(entry_price)" json:"entry_price"` // 开仓价格
	MarkPrice float64 `orm:"column(mark_price)" json:"mark_price"` // 标记价格
	UnrealizedProfit float64 `orm:"column(unrealized_profit)" json:"unrealized_profit"` // 未实现盈亏
	Mock bool `orm:"column(mock)" json:"mock"` // 是否是模拟持仓
}

type FuturesOrder struct {
	Symbol string `orm:"column(symbol)" json:"symbol"`
	ClientOrderId string `orm:"column(client_order_id)" json:"client_order_id"` // 客户端自定义ID
	OrderId int64 `orm:"column(order_id)" json:"order_id"` // 订单ID
	Side string `orm:"column(side)" json:"side"` // BUY, SELL
	PositionSide string `orm:"column(position_side)" json:"position_side"` // 持仓方向 BOTH LONG SHORT
	Type string `orm:"column(type)" json:"type"` // 下单类型 LIMIT 限价单, MARKET 市价单, STOP 止盈止损单, STOP_MARKET 止损市价单, TAKE_PROFIT 止盈限价单, TAKE_PROFIT_MARKET 止盈市价单, TRAILING_STOP_MARKET 跟踪止损单
	Status string `orm:"column(status)" json:"status"` // 订单状态 NEW(新建订单), PARTIALLY_FILLED(部分成交), FILLED(全部成交), CANCELED(已撤销), REJECTED(订单被拒绝), EXPIRED(订单过期)
	Price string `orm:"column(price)" json:"price"` // 下单价格
	OrigQty string `orm:"column(orig_qty)" json:"orig_qty"` // 下单委托数量
	ExecutedQty string `orm:"column(executed_qty)" json:"executed_qty"` // 已成交数量
	UpdateTime int64 `orm:"column(updateTime)" json:"updateTime"`
}
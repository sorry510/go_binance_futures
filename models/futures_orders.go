package models

type FuturesOrder struct {
	ID int64 `orm:"column(id)" json:"id"`
	Symbol string `orm:"column(symbol)" json:"symbol"`
	ClientOrderId string `orm:"column(client_order_id)" json:"client_order_id"` // 客户端自定义ID
	OrderId string `orm:"column(order_id)" json:"order_id"` // 订单ID
	Side string `orm:"column(side)" json:"side"` // BUY, SELL
	PositionSide string `orm:"column(position_side)" json:"position_side"` // 持仓方向 BOTH LONG SHORT
	Type string `orm:"column(type)" json:"type"` // 下单类型 LIMIT 限价单, MARKET 市价单, STOP 止盈止损单, STOP_MARKET 止损市价单, TAKE_PROFIT 止盈限价单, TAKE_PROFIT_MARKET 止盈市价单, TRAILING_STOP_MARKET 跟踪止损单
	Status string `orm:"column(status)" json:"status"` // 订单状态 NEW(新建订单), PARTIALLY_FILLED(部分成交), FILLED(全部成交), CANCELED(已撤销), REJECTED(订单被拒绝), EXPIRED(订单过期)
	Price string `orm:"column(price)" json:"price"` // 下单价格
	OrigQty string `orm:"column(orig_qty)" json:"orig_qty"` // 下单委托数量
	ExecutedQty string `orm:"column(executed_qty)" json:"executed_qty"` // 已成交数量
	AveragePrice string `orm:"column(average_price)" json:"average_price"` // 平均成交价
	StopPrice string `orm:"column(stop_price)" json:"stop_price"` // 条件订单触发价格，对追踪止损单无效
	CommissionAsset string `orm:"column(commission_asset)" json:"commission_asset"` // 手续费类型 USDT
	Commission string `orm:"column(commission)" json:"commission"` // 手续费
	RealizedPnL string `orm:"column(realized_pnl)" json:"realized_pnl"` // 已实现盈亏
	
	CreateTime int64 `orm:"column(createTime)" json:"createTime"`
	UpdateTime int64 `orm:"column(updateTime)" json:"updateTime"`
}

func (u *FuturesOrder) TableName() string {
    return "futures_orders"
}
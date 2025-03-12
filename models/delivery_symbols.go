package models

type DeliverySymbols struct {
	ID int64 `orm:"column(id)" json:"id"`
	Symbol string `orm:"column(symbol)" json:"symbol"`
	PercentChange float64 `orm:"column(percentChange)" json:"percentChange"` // 24小时涨跌幅
	Close string `orm:"column(close)" json:"close"` // 最新成交价格
	Open string `orm:"column(open)" json:"open"` // 24小时开盘价
	Low string `orm:"column(low)" json:"low"` // 24小时最低价
	High string `orm:"column(high)" json:"high"` // 24小时最高价
	Enable int `orm:"column(enable)" json:"enable"` // 是否开启
	UpdateTime int64 `orm:"column(updateTime)" json:"updateTime"` // 更新时间
	LastClose string `orm:"column(lastClose)" json:"lastClose"` // 上一次的最新成交价格
	LastUpdateTime int64 `orm:"column(lastUpdateTime)" json:"lastUpdateTime"` // 上一次的更新时间
	BaseVolume float64 `orm:"column(baseVolume)" json:"baseVolume"` // 24小时成交量
	QuoteVolume float64 `orm:"column(quoteVolume)" json:"quoteVolume"` // 24小时成交额 USDT
	CloseQty float64 `orm:"column(closeQty)" json:"closeQty"` // 最新成交价格上的成交量
	TradeCount float64 `orm:"column(tradeCount)" json:"tradeCount"` // 24小时交易数
	
	Leverage int64 `orm:"column(leverage)" json:"leverage"` // 合约倍数
	MarginType string `orm:"column(marginType)" json:"marginType"` // 杠杆类型 ISOLATED(逐仓), CROSSED(全仓)
	TickSize string `orm:"column(tickSize)" json:"tickSize"` // 交易金额精度
	StepSize string `orm:"column(stepSize)" json:"stepSize"` // 交易数量精度
	Usdt string `orm:"column(usdt)" json:"usdt"` // 交易金额
	Profit string `orm:"column(profit)" json:"profit"` // 盈利率
	Loss string `orm:"column(loss)" json:"loss"` // 损失率
	Technology string `orm:"column(technology);type(text)" json:"technology"` // 技术指标配置 json
	Strategy string `orm:"column(strategy);type(text)" json:"strategy"` // 策略 json
	StrategyType string `orm:"column(strategy_type)" json:"strategy_type"` // 策略类型 // global, line_x, custom
	Pin int64 `orm:"column(pin)" json:"pin"` // 置顶
	Sort int64 `orm:"column(sort)" json:"sort"` // 排序
	Type string `orm:"column(type)" json:"type"` // USDT, USDC
}

func (u *DeliverySymbols) TableName() string {
    return "delivery_symbols"
}
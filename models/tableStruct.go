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
	KlineInterval string `orm:"column(kline_interval)" json:"kline_interval"` // 选定的k线周期
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
	Side string `orm:"column(side)" json:"side"` // 买卖方向
	Quantity string `orm:"column(quantity)" json:"quantity"` // 卖单数量
}

type NoticeSymbols struct {
	ID int64 `orm:"column(id)" json:"id"`
	Symbol string `orm:"column(symbol)" json:"symbol"`
	Enable int `orm:"column(enable)" json:"enable"` // 是否开启
	Type int64 `orm:"column(type)" json:"type"` // 1:币币交易 2:合约交易
	NoticePrice string `orm:"column(notice_price)" json:"notice_price"` // 设定的预警价格
	HasNotice int64 `orm:"column(has_notice)" json:"has_notice"` // 1:已通知 0:未通知
	AutoOrder int64 `orm:"column(auto_order)" json:"auto_order"` // 1:自动下单 0:手动下单
	ProfitPrice string `orm:"column(profit_price)" json:"profit_price"` // 止盈价格
	LossPrice string `orm:"column(loss_price)" json:"loss_price"` // 止损价格
	
	// 以下为合约交易所需字段
	Leverage int64 `orm:"column(leverage)" json:"leverage"` // 合约倍数
	MarginType string `orm:"column(marginType)" json:"marginType"` // 杠杆类型 ISOLATED(逐仓), CROSSED(全仓)
	TickSize string `orm:"column(tickSize)" json:"tickSize"` // 交易金额精度
	StepSize string `orm:"column(stepSize)" json:"stepSize"` // 交易数量精度
	Usdt string `orm:"column(usdt)" json:"usdt"` // 交易金额
	Side string `orm:"column(side)" json:"side"` // 买卖方向 buy/sell
	Quantity string `orm:"column(quantity)" json:"quantity"` // 卖单数量
	CreateTime int64 `orm:"column(createTime)" json:"createTime"`
	UpdateTime int64 `orm:"column(updateTime)" json:"updateTime"`
}

type ListenSymbols struct {
	ID int64 `orm:"column(id)" json:"id"`
	Symbol string `orm:"column(symbol)" json:"symbol"`
	Enable int `orm:"column(enable)" json:"enable"` // 是否开启
	Type int64 `orm:"column(type)" json:"type"` // 1:币币交易 2:合约交易
	ListenType string `orm:"column(listen_type)" json:"listen_type"` // 监听类型 kline_base:K线变化 kline_kc:肯纳特通道
	KlineInterval string `orm:"column(kline_interval)" json:"kline_interval"` // 选定的k线周期
	ChangePercent string `orm:"column(change_percent)" json:"change_percent"` // 通知的变化百分比阈值
	LastNoticeTime int64 `orm:"column(last_notice_time)" json:"last_notice_time"` // 上一次通知的时间
	LastNoticeType string `orm:"column(last_notice_type)" json:"last_notice_type"` // 上一次通知的类型(up/down)
	NoticeLimitMin int64 `orm:"column(notice_limit_min)" json:"notice_limit_min"` // 通知频率限制(分钟)
	CreateTime int64 `orm:"column(createTime)" json:"createTime"`
	UpdateTime int64 `orm:"column(updateTime)" json:"updateTime"`
}

type SymbolFundingRates struct {
	ID int64 `orm:"column(id)" json:"id"`
	Symbol string `orm:"column(symbol)" json:"symbol"`
	Enable int `orm:"column(enable)" json:"enable"` // 是否开启
	NowFundingRate string `orm:"column(now_funding_rate)" json:"now_funding_rate"` // 当前资金费率
	NowFundingTime int64 `orm:"column(now_funding_time)" json:"now_funding_time"` // 当前资金费率时间
	NowPrice string `orm:"column(now_price)" json:"now_price"` // 当前价格
	LastNoticeFundingRate string `orm:"column(last_notice_funding_rate)" json:"last_notice_funding_rate"` // 上次报警时的资金费率
	LastNoticeFundingTime int64 `orm:"column(last_notice_funding_time)" json:"last_notice_funding_time"` // 上次报警时资金费率时间
	LastNoticePrice string `orm:"column(last_notice_price)" json:"last_notice_price"` // 上次报警时价格
	CreateTime int64 `orm:"column(createTime)" json:"createTime"`
	UpdateTime int64 `orm:"column(updateTime)" json:"updateTime"`
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

func (u *NoticeSymbols) TableName() string {
    return "notice_symbols"
}

func (u *ListenSymbols) TableName() string {
    return "listen_symbols"
}

func (u *SymbolFundingRates) TableName() string {
    return "symbol_funding_rates"
}

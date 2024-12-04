package models

type Config struct {
	ID int64 `orm:"column(id)" json:"id"`
	Version int64 `orm:"column(version)" json:"version"`
	FutureEnable int `orm:"column(future_enable)" json:"future_enable"`
	SpotEnable int `orm:"column(spot_enable)" json:"spot_enable"`
	DeliveryEnable int `orm:"column(delivery_enable)" json:"delivery_enable"`
	FuturesPositionConvertEnable int `orm:"column(futures_position_convert_enable)" json:"futures_position_convert_enable"` // 合约持仓正负转换通知
	FutureBuyTimeout int `orm:"column(future_buy_timeout)" json:"future_buy_timeout"`
	FutureExcludeSymbols string `orm:"column(future_exclude_symbols)" json:"future_exclude_symbols"`
	FutureMaxCount int `orm:"column(future_max_count)" json:"future_max_count"`
	FutureOrderType string `orm:"column(future_order_type)" json:"future_order_type"`
	FutureAllowLong int `orm:"column(future_allow_long)" json:"future_allow_long"`
	FutureAllowShort int `orm:"column(future_allow_short)" json:"future_allow_short"`
	FutureStrategyTrade string `orm:"column(future_strategy_trade)" json:"future_strategy_trade"`
	FutureStrategyCoin string `orm:"column(future_strategy_coin)" json:"future_strategy_coin"`
	FutureNewEnable int `orm:"column(future_new_enable)" json:"future_new_enable"`
	SpotNewEnable int `orm:"column(spot_new_enable)" json:"spot_new_enable"`
	NoticeCoinEnable int `orm:"column(notice_coin_enable)" json:"notice_coin_enable"`
	ListenCoinEnable int `orm:"column(listen_coin_enable)" json:"listen_coin_enable"`
	ListenFundingRateEnable int `orm:"column(listen_funding_rate_enable)" json:"listen_funding_rate_enable"`
	FutureTest int `orm:"column(future_test)" json:"future_test"`
	FutureTestNoticeLimitMin int `orm:"column(future_test_notice_limit_min)" json:"future_test_notice_limit_min"`
	WsFuturesEnable int `orm:"column(ws_futures_enable)" json:"ws_futures_enable"`
	WsSpotEnable int `orm:"column(ws_spot_enable)" json:"ws_spot_enable"`
	WsDeliveryEnable int `orm:"column(ws_delivery_enable)" json:"ws_delivery_enable"`
}

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
	Amount string `orm:"column(amount)" json:"amount"` // 数量，都是正数
	Avg_price string `orm:"column(avg_price)" json:"avg_price"` // 开仓价 或 平仓价
	Leverage int64 `orm:"column(leverage)" json:"leverage"`
	Inexact_profit string `orm:"column(inexact_profit)" json:"inexact_profit"` // 平仓的收益 usdt，开仓是 0.0
	Side string `orm:"column(side)" json:"side"` // open, close
	PositionSide string `orm:"column(positionSide)" json:"positionSide"` // LONG, SHORT
	OrderId int64 `orm:"column(order_id)" json:"order_id"`
	UpdateTime int64 `orm:"column(updateTime)" json:"updateTime"`
}

type Symbols struct {
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
	KlineInterval string `orm:"column(kline_interval)" json:"kline_interval"` // 选定的k线周期 (废弃)
	Technology string `orm:"column(technology)" json:"technology"` // 技术指标配置 json
	Strategy string `orm:"column(strategy)" json:"strategy"` // 策略 json
	StrategyType string `orm:"column(strategy_type)" json:"strategy_type"` // 策略类型 // global, line_x, custom
	Pin int64 `orm:"column(pin)" json:"pin"` // 置顶
	Sort int64 `orm:"column(sort)" json:"sort"` // 排序
	Type string `orm:"column(type)" json:"type"` // USDT, USDC
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
	Technology string `orm:"column(technology)" json:"technology"` // 技术指标配置 json
	Strategy string `orm:"column(strategy)" json:"strategy"` // 策略 json
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

// 吃资金费率
type EatRateSymbols struct {
	ID int64 `orm:"column(id)" json:"id"`
	Type int64 `orm:"column(type)" json:"type"` // 1:正向套利 2:反向套利, 目前只有正向套利
	Enable int `orm:"column(enable)" json:"enable"`
	Symbol string `orm:"column(symbol)" json:"symbol"` // 交易对(usdt)
	SpotSymbol string `orm:"column(spot_symbol)" json:"spot_symbol"` // 现货的交易对
	FuturesSymbol string `orm:"column(futures_symbol)" json:"futures_symbol"` // 合约的交易对
	TotalAmount string `orm:"column(total_amount)" json:"total_amount"` // 总交易金额
	SpotAmount string `orm:"column(spot_amount)" json:"spot_amount"` // 现货交易金额
	FuturesAmount string `orm:"column(futures_amount)" json:"futures_amount"` // 合约交易金额 (没有乘杠杆倍率)
	SpotPrice string `orm:"column(spot_price)" json:"spot_price"` // 现货的买入价格
	FuturesPrice string `orm:"column(futures_price)" json:"futures_price"` // 合约做空的价格
	SpotQuantity float64 `orm:"column(spot_quantity)" json:"spot_quantity"` // 现货的买入的数量
	FuturesQuantity float64 `orm:"column(futures_quantity)" json:"futures_quantity"` // 合约做空的数量
	SpotOrderId int64 `orm:"column(spot_order_id)" json:"spot_order_id"` // 现货订单id
	FuturesOrderId int64 `orm:"column(futures_order_id)" json:"futures_order_id"` // 合约订单id
	
	Leverage int64 `orm:"column(leverage)" json:"leverage"` // 合约杠杆倍率
	MarginType string `orm:"column(marginType)" json:"marginType"` // 杠杆类型 ISOLATED(逐仓), CROSSED(全仓)
	TickSize string `orm:"column(tickSize)" json:"tickSize"` // 交易金额精度
	StepSize string `orm:"column(stepSize)" json:"stepSize"` // 交易数量精度
	
	Profit float64 `orm:"column(profit)" json:"profit"` // 盈利(usdt)，不带交易买卖的手续费
	LastProfitTime int64 `orm:"column(last_profit_time)" json:"last_profit_time"` // 上次统计时间
	StartTime int64 `orm:"column(start_time)" json:"start_time"` // 套利开始时间
	EndTime int64 `orm:"column(end_time)" json:"end_time"` // 套利结束时间
	
	CreateTime int64 `orm:"column(createTime)" json:"createTime"`
	UpdateTime int64 `orm:"column(updateTime)" json:"updateTime"`
}

// 策略模板
type StrategyTemplates struct {
	ID int64 `orm:"column(id)" json:"id"`
	Name string `orm:"column(name)" json:"name"`
	Technology string `orm:"column(technology)" json:"technology"` // 技术指标配置 json
	Strategy string `orm:"column(strategy)" json:"strategy"` // 策略 json
	CreateTime int64 `orm:"column(createTime)" json:"createTime"`
	UpdateTime int64 `orm:"column(updateTime)" json:"updateTime"`
}

type TestStrategyResults struct {
	ID int64 `orm:"column(id)" json:"id"`
	Symbol string `orm:"column(symbol)" json:"symbol"`
	Price string `orm:"column(price)" json:"price"` // 触发开仓策略时通知价格
	Leverage int64 `orm:"column(leverage)" json:"leverage"` // 合约倍数
	TickSize string `orm:"column(tickSize)" json:"tickSize"` // 交易金额精度
	StepSize string `orm:"column(stepSize)" json:"stepSize"` // 交易数量精度
	Usdt string `orm:"column(usdt)" json:"usdt"` // 交易金额
	Profit string `orm:"column(profit)" json:"profit"` // 盈利率
	Loss string `orm:"column(loss)" json:"loss"` // 损失率
	PositionAmt string `orm:"column(position_amt)" json:"position_amt"` // 买入的数量, 做多为正, 做空为负
	PositionSide string `orm:"column(position_side)" json:"position_side"` // LONG, SHORT
	Technology string `orm:"column(technology)" json:"technology"` // 技术指标配置 json
	Strategy string `orm:"column(strategy)" json:"strategy"` // 策略 json
	OpenStrategy string `orm:"column(open_strategy)" json:"open_strategy"` // 开仓策略
	CloseStrategy string `orm:"column(close_strategy)" json:"close_strategy"` // 平仓策略
	ClosePrice string `orm:"column(close_price)" json:"close_price"` // 触发平仓策略时通知价格
	CloseProfit string `orm:"column(close_profit)" json:"close_profit"` // 触发平仓策略时收益 usdt
	CreateTime int64 `orm:"column(createTime)" json:"createTime"`
	UpdateTime int64 `orm:"column(updateTime)" json:"updateTime"`
}

func (u *Config) TableName() string {
    return "config"
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

func (u *EatRateSymbols) TableName() string {
    return "eat_rate_symbols"
}

func (u *StrategyTemplates) TableName() string {
    return "strategy_templates"
}

func (u *TestStrategyResults) TableName() string {
    return "test_strategy_results"
}
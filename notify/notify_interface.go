package notify

type FuturesOrderParams struct {
	Title string
	Symbol string
	Side string // buy,sell
	PositionSide string // long, short
	Price float64
	Quantity float64
	Leverage float64
	Profit float64
	ChangePercent float64
	Remarks string
	Status string // success, fail 
	Error string // 错误信息
}

type FuturesNoticeParams struct {
	Title string
	Symbol string
	Side string // buy,sell
	PositionSide string // long, short
	Price float64
	AutoOrder string
	Status string // success, fail 
	Error string // 错误信息
}

type FuturesListenParams struct {
	Title string
	Symbol string
	Side string // buy,sell
	PositionSide string // long, short
	Price float64
	StrategyName string
	Remarks string
	ChangePercent float64
	FundingRate float64
	
	NowPrice float64
	StopLossPrice float64
	TargetHalfProfitPrice float64
	TargetAllProfitPrice float64
	DesiredPrice float64
}

type SpotOrderParams struct {
	Title string
	Symbol string
	Side string // buy,sell
	Price float64
	Quantity float64
	Profit float64
	ChangePercent float64
	Remarks string
	Status string // success, fail 
	Error string // 错误信息
}

type SpotNoticeParams struct {
	Title string
	Symbol string
	Side string // buy,sell
	Price float64
	AutoOrder string
}

type SpotListenParams struct {
	Title string
	Symbol string
	Side string // buy,sell
	Price float64
	Remarks string
	ChangePercent float64
	FundingRate float64
	
	NowPrice float64
	StopLossPrice float64
	TargetHalfProfitPrice float64
	TargetAllProfitPrice float64
	DesiredPrice float64
}

type Pusher interface {
	FuturesOpenOrder(params FuturesOrderParams)
	FuturesCloseOrder(params FuturesOrderParams)
	FuturesNotice(params FuturesNoticeParams)
	FuturesListenKlineBase(params FuturesListenParams)
	FuturesListenKlineKc(params FuturesListenParams)
	FuturesListenKlineCustom(params FuturesListenParams)
	FuturesListenFundingRate(params FuturesListenParams)
	
	SpotOrder(params SpotOrderParams)
	SpotNotice(params SpotNoticeParams)
	SpotListenKlineBase(params SpotListenParams)
}
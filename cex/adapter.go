package cex

import "go_binance_futures/cex/futures"

type Cex interface {
	Name() string
	Version() string
	Futures
	Spot
	Delivery
}

type Futures interface {
	// public api
	SetConfig(config *futures.Config) // 设置配置
	GetDepths(params *futures.DepthParams) (depths *[]futures.Depth, err error)
	GetTickerPrices(params *futures.TicketPriceParams) (tickerPrices *[]futures.TickerPrice, err error)
	GetLine(params *futures.KlineParams) (kline *[]futures.Kline, err error)
	GetExchangeInfo() (exchangeInfo *futures.ExchangeInfo, err error)
	GetFundingRate(params *futures.FundingRateParams) (fundingRate *futures.FundingRate, err error)
	GetFundingRateHistory(params *futures.FundingRateHistoryParams) (fundingRateHistory *[]futures.FundingRateHistory, err error)
	
	// account api
	GetAccount() (account *futures.Account, err error)
	GetPositions(params *futures.PositionParams) (positions *[]futures.Position, err error)
	BuyLimit(params *futures.CreateOrder) (order *futures.CreateOrder, err error)
	SellLimit(params *futures.CreateOrder) (order *futures.CreateOrder, err error)
	BuyMarket(params *futures.CreateOrder) (order *futures.CreateOrder, err error)
	SellMarket(params *futures.CreateOrder) (order *futures.CreateOrder, err error)
	CancelOrder(params *futures.CancelOrderParams) (order *futures.CancelOrder, err error)
	SetLeverage(params *futures.LeverageParams) (res *futures.Leverage, err error)
	SetMarginType(params *futures.MarginTypeParams) (err error)
	GetOrders(params *futures.OrderParams) (orders *[]futures.Order, err error)
	GetOrder(params *futures.OrderParams) (order *futures.Order, err error)
	
	// ws api
	WsAllMarketTicker() (err error)
	WsUserData() (err error)
}

type Spot interface {
	
}

type Delivery interface {
	
}
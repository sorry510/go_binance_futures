package binance

import (
	"github.com/adshao/go-binance/v2/delivery"
	"github.com/adshao/go-binance/v2/futures"
)

func wsFuturesAllMarketTickerServe(handler futures.WsAllMarketTickerHandler, errHandler futures.ErrHandler) (chan struct{}, chan struct{}, error) {
	return proxyPool.WithFuturesWS(func() (chan struct{}, chan struct{}, error) {
		return futures.WsAllMarketTickerServe(handler, errHandler)
	})
}

func wsFuturesUserDataServe(listenKey string, handler futures.WsUserDataHandler, errHandler futures.ErrHandler) (chan struct{}, chan struct{}, error) {
	return proxyPool.WithFuturesWS(func() (chan struct{}, chan struct{}, error) {
		return futures.WsUserDataServe(listenKey, handler, errHandler)
	})
}

func wsFuturesKlineServe(symbol string, interval string, handler futures.WsKlineHandler, errHandler futures.ErrHandler) (chan struct{}, chan struct{}, error) {
	return proxyPool.WithFuturesWS(func() (chan struct{}, chan struct{}, error) {
		return futures.WsKlineServe(symbol, interval, handler, errHandler)
	})
}

func wsFuturesAllLiquidationOrderServe(handler futures.WsLiquidationOrderHandler, errHandler futures.ErrHandler) (chan struct{}, chan struct{}, error) {
	return proxyPool.WithFuturesWS(func() (chan struct{}, chan struct{}, error) {
		return futures.WsAllLiquidationOrderServe(handler, errHandler)
	})
}

func wsDeliveryAllMarketTickerServe(handler delivery.WsAllMarketTickerHandler, errHandler delivery.ErrHandler) (chan struct{}, chan struct{}, error) {
	return proxyPool.WithDeliveryWS(func() (chan struct{}, chan struct{}, error) {
		return delivery.WsAllMarketTickerServe(handler, errHandler)
	})
}

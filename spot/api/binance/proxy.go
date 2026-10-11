package binance

import binance "github.com/adshao/go-binance/v2"

func wsSpotAllMarketsStatServe(handler binance.WsAllMarketsStatHandler, errHandler binance.ErrHandler) (chan struct{}, chan struct{}, error) {
	return proxyPool.WithSpotWS(func() (chan struct{}, chan struct{}, error) {
		return binance.WsAllMarketsStatServe(handler, errHandler)
	})
}

func wsSpotAllMiniMarketsStatServe(handler binance.WsAllMiniMarketsStatServeHandler, errHandler binance.ErrHandler) (chan struct{}, chan struct{}, error) {
	return proxyPool.WithSpotWS(func() (chan struct{}, chan struct{}, error) {
		return binance.WsAllMiniMarketsStatServe(handler, errHandler)
	})
}

func wsSpotCombinedKlineServeMultiInterval(symbolIntervals map[string][]string, handler binance.WsKlineHandler, errHandler binance.ErrHandler) (chan struct{}, chan struct{}, error) {
	return proxyPool.WithSpotWS(func() (chan struct{}, chan struct{}, error) {
		return binance.WsCombinedKlineServeMultiInterval(symbolIntervals, handler, errHandler)
	})
}

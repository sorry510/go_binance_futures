package binance

import binance "github.com/adshao/go-binance/v2"

func wsSpotAllMarketsStatServe(handler binance.WsAllMarketsStatHandler, errHandler binance.ErrHandler) (chan struct{}, chan struct{}, error) {
	return proxyPool.WithSpotWS(func() (chan struct{}, chan struct{}, error) {
		return binance.WsAllMarketsStatServe(handler, errHandler)
	})
}

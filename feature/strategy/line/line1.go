package line

import "github.com/adshao/go-binance/v2/futures"


type TradeLine1 struct {
}

func (tradeLine1 TradeLine1) GetCanLongOrShort(symbol string) (canLong bool, canShort bool) {
	return false, false
}

func (tradeLine1 TradeLine1) CanOrderComplete(symbol string, positionSide string) (complete bool) {
	return false
}

func (tradeLine1 TradeLine1) AutoStopOrder(position *futures.PositionRisk, nowProfit float64) (stop bool) {
	return false
}
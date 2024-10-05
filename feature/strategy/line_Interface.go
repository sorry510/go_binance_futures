package strategy

import "github.com/adshao/go-binance/v2/futures"

type LineStrategy interface {
    // 是否可以买多或者买空
    GetCanLongOrShort(symbol string) (canLong bool, canShort bool)
    // 达到止盈或止损点后判断是否可以平仓
    CanOrderComplete(symbol string, positionSide string) (complete bool)
    // 达到止盈或止损前判定是否可以平仓
    AutoStopOrder(position *futures.PositionRisk, nowProfit float64) (stop bool)
}


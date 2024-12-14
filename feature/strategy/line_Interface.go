package strategy

import (
	"go_binance_futures/models"
	"go_binance_futures/types"
)


type OpenParams struct {
    Symbols *models.Symbols
}

type OpenResult struct {
    CanLong bool
    CanShort bool
}

type CloseParams struct {
    Symbols *models.Symbols
    Position types.FuturesPosition
    NowProfit float64 // 当前收益率%
}

type CloseResult struct {
    Complete bool
}

type LineStrategy interface {
    // 是否可以买多或者买空
    GetCanLongOrShort(openParams OpenParams) (openResult OpenResult)
    // 达到止盈或止损点后判断是否可以平仓
    CanOrderComplete(closeParams CloseParams) (closeResult CloseResult)
    // 达到止盈或止损前判定是否可以平仓
    AutoStopOrder(closeParams CloseParams) (closeResult CloseResult)
}


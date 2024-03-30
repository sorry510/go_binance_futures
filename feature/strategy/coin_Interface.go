package strategy

import "go_binance_futures/models"

type CoinStrategy interface {
    SelectCoins(allCoins []*models.Symbols) (coins []*models.Symbols)
}


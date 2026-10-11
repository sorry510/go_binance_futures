package binance

import (
	"strconv"
	"strings"

	spotbinance "github.com/adshao/go-binance/v2"
)

// normalizeSpotMiniTickers converts Binance's active !miniTicker@arr feed to
// the existing spot price snapshot/persistence representation. The mini
// feed contains price, 24h OHLCV, but not last-trade quantity or trade count.
func normalizeSpotMiniTickers(event spotbinance.WsAllMiniMarketsStatEvent) spotbinance.WsAllMarketsStatEvent {
	result := make(spotbinance.WsAllMarketsStatEvent, 0, len(event))
	for _, item := range event {
		if item == nil || item.Symbol == "" || item.LastPrice == "" {
			continue
		}
		change := "0"
		open, openErr := strconv.ParseFloat(item.OpenPrice, 64)
		close, closeErr := strconv.ParseFloat(item.LastPrice, 64)
		if openErr == nil && closeErr == nil && open > 0 {
			change = strconv.FormatFloat((close/open-1)*100, 'f', 8, 64)
		}
		baseVolume := strings.TrimSpace(item.BaseVolume)
		if baseVolume == "" {
			baseVolume = "0"
		}
		quoteVolume := strings.TrimSpace(item.QuoteVolume)
		if quoteVolume == "" {
			quoteVolume = "0"
		}
		result = append(result, &spotbinance.WsMarketStatEvent{
			Event:              item.Event,
			Time:               item.Time,
			Symbol:             item.Symbol,
			PriceChangePercent: change,
			LastPrice:          item.LastPrice,
			OpenPrice:          item.OpenPrice,
			HighPrice:          item.HighPrice,
			LowPrice:           item.LowPrice,
			BaseVolume:         baseVolume,
			QuoteVolume:        quoteVolume,
			CloseQty:           "0",
			Count:              0,
		})
	}
	return result
}

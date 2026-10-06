package line

import (
	"strconv"

	"github.com/adshao/go-binance/v2/futures"
)

// GetLineFloatPrices converts K-line OHLC values to float slices.
func GetLineFloatPrices(data []*futures.Kline) (high, low, close, open []float64) {
	high = make([]float64, len(data))
	low = make([]float64, len(data))
	close = make([]float64, len(data))
	open = make([]float64, len(data))
	for key, item := range data {
		highPrice, _ := strconv.ParseFloat(item.High, 64)
		lowPrice, _ := strconv.ParseFloat(item.Low, 64)
		closePrice, _ := strconv.ParseFloat(item.Close, 64)
		openPrice, _ := strconv.ParseFloat(item.Open, 64)
		high[key] = highPrice
		low[key] = lowPrice
		close[key] = closePrice
		open[key] = openPrice
	}
	return high, low, close, open
}

func GetLineFloatValues(data []*futures.Kline) (high, low, close, open, amount, qps, takerBuyAmount, takerBuyRatio []float64) {
	high = make([]float64, len(data))
	low = make([]float64, len(data))
	close = make([]float64, len(data))
	open = make([]float64, len(data))
	amount = make([]float64, len(data))
	qps = make([]float64, len(data))
	takerBuyAmount = make([]float64, len(data))
	takerBuyRatio = make([]float64, len(data))
	for key, item := range data {
		highPrice, _ := strconv.ParseFloat(item.High, 64)
		lowPrice, _ := strconv.ParseFloat(item.Low, 64)
		closePrice, _ := strconv.ParseFloat(item.Close, 64)
		openPrice, _ := strconv.ParseFloat(item.Open, 64)
		amountFloat, _ := strconv.ParseFloat(item.QuoteAssetVolume, 64)
		takerBuyFloat, _ := strconv.ParseFloat(item.TakerBuyQuoteAssetVolume, 64)
		high[key] = highPrice
		low[key] = lowPrice
		close[key] = closePrice
		open[key] = openPrice
		amount[key] = amountFloat
		takerBuyAmount[key] = takerBuyFloat
		if amountFloat > 0 {
			takerBuyRatio[key] = takerBuyFloat / amountFloat
		}
		if durationMilliseconds := item.CloseTime - item.OpenTime; durationMilliseconds > 0 {
			qps[key] = amountFloat / (float64(durationMilliseconds) / 1000)
		}
	}
	return high, low, close, open, amount, qps, takerBuyAmount, takerBuyRatio
}

func autoStopNeutralROI(nowProfit float64) bool {
	return nowProfit > -3 && nowProfit < 3
}

package line

import (
	"strconv"
	"time"

	"go_binance_futures/models"
	spotbinance "go_binance_futures/spot/api/binance"
	"go_binance_futures/utils"

	binance "github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/config"
	"github.com/beego/beego/v2/core/logs"
)

// ConvertSpotKlines preserves OHLCV, timestamps and taker volume, so the
// indicator implementation can operate on spot prices without changing math.
func ConvertSpotKlines(rows []*binance.Kline) []*futures.Kline {
	converted := make([]*futures.Kline, 0, len(rows))
	for _, k := range rows {
		if k == nil {
			continue
		}
		converted = append(converted, &futures.Kline{
			OpenTime: k.OpenTime, CloseTime: k.CloseTime,
			Open: k.Open, High: k.High, Low: k.Low, Close: k.Close,
			Volume: k.Volume, QuoteAssetVolume: k.QuoteAssetVolume,
			TradeNum: k.TradeNum, TakerBuyBaseAssetVolume: k.TakerBuyBaseAssetVolume,
			TakerBuyQuoteAssetVolume: k.TakerBuyQuoteAssetVolume,
		})
	}
	return converted
}

func spotIndicatorKlines(symbol, interval string, limit int) ([]*futures.Kline, error) {
	rows, err := spotbinance.GetKlineData(symbol, interval, limit)
	if err != nil {
		return nil, err
	}
	return ConvertSpotKlines(rows), nil
}

// InitSpotParseEnv deliberately uses spot tickers and spot klines, not the
// futures market used by InitParseEnv. Spot has no funding-rate field.
func InitSpotParseEnv(symbol string, strTechnology string) map[string]interface{} {
	var ticker models.SpotSymbols
	err := orm.NewOrm().QueryTable("spot_symbols").Filter("symbol", symbol).One(&ticker)
	if err != nil {
		logs.Error("spot ticker lookup failed:", symbol, err)
	}
	marketCondition, _ := config.String("MarketCondition")
	systemStartTimeText, _ := config.String("system_start_time")
	systemStartTime, _ := strconv.ParseInt(systemStartTimeText, 10, 64)
	indicators, klines := ParseTechnologyConfigWithKlineLoader(symbol, strTechnology, spotIndicatorKlines)

	closePrice, _ := strconv.ParseFloat(ticker.Close, 64)
	// Prefer the live spot WebSocket price when the DB flush has not finished.
	if livePrice, ok := spotbinance.GetFreshSpotTickerPrice(symbol); ok {
		if parsed, err := strconv.ParseFloat(livePrice, 64); err == nil && parsed > 0 {
			closePrice = parsed
		}
	}
	openPrice, _ := strconv.ParseFloat(ticker.Open, 64)
	lowPrice, _ := strconv.ParseFloat(ticker.Low, 64)
	highPrice, _ := strconv.ParseFloat(ticker.High, 64)
	env := map[string]interface{}{
		"SystemStartTime":        systemStartTime,
		"MarketCondition":        marketCondition,
		"NowTime":                time.Now().UnixMilli(),
		"NowPrice":               closePrice,
		"NowSymbolPercentChange": ticker.PercentChange,
		"NowSymbolClose":         closePrice,
		"NowSymbolOpen":          openPrice,
		"NowSymbolLow":           lowPrice,
		"NowSymbolHigh":          highPrice,
		"OpenStrategyHash":       "",
		"KdjSimple":              KdjSimple,
		"IsAsc":                  utils.IsAsc,
		"IsDesc":                 utils.IsDesc,
	}
	for key, value := range indicators {
		env[key] = value
	}
	for interval, price := range klines {
		env["kline_"+interval] = price
	}
	return env
}

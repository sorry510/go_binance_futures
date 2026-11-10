package line

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"time"

	spotapi "go_binance_futures/spot/api/binance"
	"go_binance_futures/technology"
	"go_binance_futures/utils"

	sdk "github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/futures"
)

// InitClosedSpotParseEnv reads only fully closed spot kline records. In
// particular, the S3 signal must not use the still-forming indicator candle,
// and must never load Futures positions, FundingRate or benchmark variables.
// The current Spot tick is used separately for order amount/freshness checks.
func InitClosedSpotParseEnv(ctx context.Context, symbol, technologyJSON string,
	bar *sdk.Kline) (map[string]interface{}, error) {
	if bar == nil || symbol == "" || bar.CloseTime <= 0 || bar.OpenTime <= 0 ||
		bar.CloseTime < bar.OpenTime || bar.CloseTime > time.Now().UnixMilli() ||
		time.Since(time.UnixMilli(bar.CloseTime)) > 3*time.Minute {
		return nil, errors.New("Spot closed candle missing/stale")
	}
	var config technology.TechnologyConfig
	if err := json.Unmarshal([]byte(technologyJSON), &config); err != nil {
		return nil, err
	}
	// Retrieve an extra incomplete candle; reject any indicator that cannot
	// supply its full closed history rather than reusing incomplete kline data.
	loader := func(sym, interval string, limit int) ([]*futures.Kline, error) {
		if limit < 1 || limit > 997 {
			return nil, errors.New("Spot indicator history limit invalid")
		}
		raw, err := spotapi.GetKlineDataContext(ctx, sym, interval, limit+3)
		if err != nil {
			return nil, err
		}
		filtered := make([]*sdk.Kline, 0, len(raw))
		for _, row := range raw {
			if row != nil && row.CloseTime <= bar.CloseTime {
				filtered = append(filtered, row)
			}
		}
		if len(filtered) < limit {
			return nil, errors.New("Spot closed indicator history incomplete")
		}
		return ConvertSpotKlines(filtered[len(filtered)-limit:]), nil
	}
	indicators, klines := ParseTechnologyConfigWithKlineLoader(symbol, technologyJSON, loader)
	v := reflect.ValueOf(config)
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		for j := 0; j < field.Len(); j++ {
			item := field.Index(j).Interface().(technology.IndicatorConfig)
			if item.Enable {
				if _, ok := indicators[item.Name]; !ok {
					return nil, fmt.Errorf("Spot indicator %q unavailable for closed candle", item.Name)
				}
				if _, ok := klines[item.KlineInterval]; !ok {
					return nil, fmt.Errorf("Spot indicator %q closed kline unavailable", item.Name)
				}
			}
		}
	}
	closePrice, err := strconv.ParseFloat(bar.Close, 64)
	if err != nil || closePrice <= 0 {
		return nil, errors.New("bad Spot close price")
	}
	openPrice, err := strconv.ParseFloat(bar.Open, 64)
	if err != nil || openPrice <= 0 {
		return nil, errors.New("bad Spot open price")
	}
	highPrice, err := strconv.ParseFloat(bar.High, 64)
	if err != nil {
		return nil, err
	}
	lowPrice, err := strconv.ParseFloat(bar.Low, 64)
	if err != nil {
		return nil, err
	}
	env := map[string]interface{}{
		"NowTime": bar.CloseTime, "NowPrice": closePrice,
		"NowSymbolClose": closePrice, "NowSymbolOpen": openPrice,
		"NowSymbolHigh": highPrice, "NowSymbolLow": lowPrice,
		"NowSymbolPercentChange": (closePrice/openPrice - 1) * 100,
		"SystemStartTime":        int64(0), "MarketCondition": "",
		"OpenStrategyHash": "",
		"KdjSimple":        KdjSimple, "IsAsc": utils.IsAsc, "IsDesc": utils.IsDesc,
	}
	for name, value := range indicators {
		env[name] = value
	}
	for interval, value := range klines {
		env["kline_"+interval] = value
	}
	return env, nil
}

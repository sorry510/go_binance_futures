package binance

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/adshao/go-binance/v2/futures"
)

// CoinMKlinesContext retrieves ONLY public DAPI candles using the configured
// COIN-M client. futures.Kline is an in-memory indicator data shape; no FAPI
// client or USD-M market endpoint is used.
func CoinMKlinesContext(ctx context.Context, symbol, interval string, limit int) ([]*futures.Kline, error) {
	if ctx == nil {
		return nil, errors.New("COIN-M market context required")
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if coinMSetupErr != nil {
		return nil, coinMSetupErr
	}
	if deliveryClient == nil || deliveryClient.HTTPClient == nil ||
		coinMRuntime.RESTURL == "" || deliveryClient.BaseURL != coinMRuntime.RESTURL {
		return nil, errors.New("COIN-M DAPI candle client is unconfigured")
	}
	if !strings.HasSuffix(symbol, "USD_PERP") || len(symbol) > 32 {
		return nil, fmt.Errorf("invalid COIN-M perpetual symbol")
	}
	switch interval {
	case "1m", "3m", "5m", "15m", "30m", "1h", "2h", "4h", "6h", "8h", "12h", "1d", "3d", "1w", "1M":
	default:
		return nil, fmt.Errorf("invalid COIN-M Kline interval %q", interval)
	}
	if limit < 2 || limit > 1500 {
		return nil, fmt.Errorf("invalid COIN-M Kline limit")
	}
	data, err := deliveryClient.NewKlinesService().Symbol(symbol).Interval(interval).Limit(limit).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("COIN-M DAPI Klines: %w", err)
	}
	if len(data) < 2 {
		return nil, errors.New("insufficient COIN-M Klines")
	}
	out := make([]*futures.Kline, 0, len(data))
	var prev int64
	for _, row := range data {
		if row == nil || row.OpenTime <= prev || row.CloseTime <= row.OpenTime {
			return nil, errors.New("COIN-M Kline timeline invalid")
		}
		prev = row.OpenTime
		o, e1 := coinMPositivePrice(row.Open)
		h, e2 := coinMPositivePrice(row.High)
		l, e3 := coinMPositivePrice(row.Low)
		c, e4 := coinMPositivePrice(row.Close)
		if e1 != nil || e2 != nil || e3 != nil || e4 != nil || h < l || o > h || o < l || c > h || c < l {
			return nil, errors.New("invalid COIN-M Kline OHLC")
		}
		out = append(out, &futures.Kline{
			OpenTime: row.OpenTime, CloseTime: row.CloseTime, Open: row.Open, High: row.High, Low: row.Low, Close: row.Close,
			Volume: row.Volume, QuoteAssetVolume: row.QuoteAssetVolume, TradeNum: row.TradeNum,
			TakerBuyBaseAssetVolume: row.TakerBuyBaseAssetVolume, TakerBuyQuoteAssetVolume: row.TakerBuyQuoteAssetVolume,
		})
	}
	last := out[len(out)-1]
	span := coinMKlineSpan(interval)
	age := time.Since(time.UnixMilli(last.OpenTime))
	if span <= 0 || age < -5*time.Minute || age > span+5*time.Minute {
		return nil, errors.New("COIN-M Klines stale or not from current selected market")
	}
	return out, nil
}
func coinMPositivePrice(s string) (float64, error) {
	v, e := strconv.ParseFloat(s, 64)
	if e != nil || v <= 0 || math.IsInf(v, 0) || math.IsNaN(v) {
		return 0, fmt.Errorf("invalid COIN-M price")
	}
	return v, nil
}
func coinMKlineSpan(interval string) time.Duration {
	switch interval {
	case "1d":
		return 24 * time.Hour
	case "3d":
		return 72 * time.Hour
	case "1w":
		return 7 * 24 * time.Hour
	case "1M":
		return 31 * 24 * time.Hour
	default:
		v, _ := time.ParseDuration(interval)
		return v
	}
}

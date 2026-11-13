package binance

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"go_binance_futures/service/binanceapiusage"
)

// SyncSpotMarketServerTime synchronizes signed REST/WS timestamps using
// the current account environment; it never touches SDK global mode.
func (a *QuantAdapter) SyncSpotMarketServerTime(ctx context.Context) error {
	if err := a.marketOnly(); err != nil {
		return err
	}
	_, err := a.c.NewSetServerTimeService().Do(ctx)
	return err
}

// SpotMarketSnapshot must originate from the same Testnet REST client
// that is used for placing orders. The ordinary local spot_symbols price and
// Mainnet Spot ticker MUST NOT drive Testnet automatic trading decisions.
type SpotMarketSnapshot struct {
	Symbol, Price, Open, High, Low string
	PercentChange                  float64
	AsOf                           time.Time
}

func (a *QuantAdapter) marketOnly() error {
	if a == nil || a.c == nil || !spotQuantKnownREST(a.endpoint) || a.c.BaseURL != a.endpoint {
		return fmt.Errorf("Spot market must match the trading account environment")
	}
	return nil
}

func (a *QuantAdapter) MarketSnapshot(ctx context.Context, symbol string) (SpotMarketSnapshot, error) {
	var out SpotMarketSnapshot
	if err := a.marketOnly(); err != nil {
		return out, err
	}
	stats, err := a.c.NewListPriceChangeStatsService().Symbol(symbol).Do(binanceapiusage.WithSource(ctx, "spot_quant_24h"))
	if err != nil {
		return out, err
	}
	if len(stats) != 1 || stats[0] == nil || stats[0].Symbol != symbol {
		return out, fmt.Errorf("Spot market ticker %s is not unique or unavailable", symbol)
	}
	v := stats[0]
	pct, err := strconv.ParseFloat(v.PriceChangePercent, 64)
	if err != nil || math.IsNaN(pct) || math.IsInf(pct, 0) {
		return out, fmt.Errorf("invalid Testnet 24h change percentage")
	}
	now := time.Now().UTC()
	age := now.Sub(time.UnixMilli(v.CloseTime))
	if v.CloseTime <= 0 || age < -5*time.Minute || age > 5*time.Minute {
		return out, fmt.Errorf("stale Spot market ticker %s (age %s)", symbol, age)
	}
	for _, raw := range []string{v.LastPrice, v.OpenPrice, v.HighPrice, v.LowPrice} {
		num, err := strconv.ParseFloat(raw, 64)
		if err != nil || num <= 0 || math.IsNaN(num) || math.IsInf(num, 0) {
			return out, fmt.Errorf("invalid Spot market 24h price")
		}
	}
	return SpotMarketSnapshot{
		Symbol: symbol, Price: v.LastPrice, Open: v.OpenPrice,
		High: v.HighPrice, Low: v.LowPrice, PercentChange: pct, AsOf: now,
	}, nil
}

// Futures.Kline is only the existing indicator-compatible OHLC struct;
// these bars are fetched exclusively from Binance SPOT MARKET, not futures.
func (a *QuantAdapter) MarketKlines(ctx context.Context, symbol, interval string, limit int) ([]*futures.Kline, error) {
	if err := a.marketOnly(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(symbol) == "" || limit <= 0 || limit > 1000 {
		return nil, fmt.Errorf("invalid Spot market kline request")
	}
	raw, err := a.c.NewKlinesService().Symbol(symbol).Interval(interval).Limit(limit).Do(binanceapiusage.WithSource(ctx, "spot_quant_klines"))
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("Spot market %s %s history empty", symbol, interval)
	}
	rows := make([]*futures.Kline, 0, len(raw))
	for _, k := range raw {
		if k == nil {
			return nil, fmt.Errorf("nil Spot market Kline")
		}
		rows = append(rows, &futures.Kline{
			OpenTime: k.OpenTime, CloseTime: k.CloseTime, Open: k.Open, High: k.High, Low: k.Low, Close: k.Close,
			Volume: k.Volume, QuoteAssetVolume: k.QuoteAssetVolume, TradeNum: k.TradeNum,
			TakerBuyBaseAssetVolume: k.TakerBuyBaseAssetVolume, TakerBuyQuoteAssetVolume: k.TakerBuyQuoteAssetVolume,
		})
	}
	return rows, nil
}

package feature

import (
	"context"
	"fmt"
	"time"

	"go_binance_futures/feature/strategy/line"
	"go_binance_futures/models"
	spotbinance "go_binance_futures/spot/api/binance"

	"github.com/adshao/go-binance/v2/futures"
)

// SpotMarketStrategy sources ALL signal prices and indicator candles from
// the same dedicated Spot market adapter as the execution Processor.
// Local spot_symbols are used ONLY for the user's strategy/settings;
// local another market ticker fields must never contribute to Spot signals.
func SpotMarketStrategy(exchange *spotbinance.QuantAdapter) (line.SpotRuleEvaluator, error) {
	if exchange == nil {
		return line.SpotRuleEvaluator{}, fmt.Errorf("nil Spot market market adapter")
	}
	loader := func(symbol, interval string, limit int) ([]*futures.Kline, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return exchange.MarketKlines(ctx, symbol, interval, limit)
	}
	return line.SpotRuleEvaluator{
		LoadEnvironment: func(coin models.SpotSymbols) (map[string]interface{}, error) {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			tick, err := exchange.MarketSnapshot(ctx, coin.Symbol)
			if err != nil {
				return nil, err
			}
			// Use a local COPY; do not modify the persisted mainnet ticker row.
			snapshot := coin
			snapshot.Open = tick.Open
			snapshot.High = tick.High
			snapshot.Low = tick.Low
			snapshot.Close = tick.Price
			snapshot.PercentChange = tick.PercentChange
			snapshot.UpdateTime = tick.AsOf.UnixMilli()
			return line.LoadVerifiedSpotRuleEnvWithSource(snapshot, tick.AsOf, tick.Price, loader)
		},
		LoadKlines: loader,
	}, nil
}

// 3d signal events must use Spot Exchange OPEN, not another market or epoch
// truncation. Unavailable, future or old OpenTimes block any NEW BUY.
func SpotMarketThreeDayOpen(exchange *spotbinance.QuantAdapter) func(string, time.Time) (time.Time, error) {
	return func(symbol string, at time.Time) (time.Time, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		rows, err := exchange.MarketKlines(ctx, symbol, "3d", 2)
		if err != nil {
			return time.Time{}, err
		}
		if len(rows) == 0 {
			return time.Time{}, fmt.Errorf("Spot market 3d Kline is absent")
		}
		open := time.UnixMilli(rows[len(rows)-1].OpenTime).UTC()
		if open.After(at) || at.Sub(open) >= 3*24*time.Hour {
			return time.Time{}, fmt.Errorf("stale Spot 3d Kline")
		}
		return open, nil
	}
}

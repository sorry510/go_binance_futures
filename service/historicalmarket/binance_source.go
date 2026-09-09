package historicalmarket

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	binanceapi "go_binance_futures/feature/api/binance"
)

type BinanceSource struct{}

func (BinanceSource) Klines(ctx context.Context, market, symbol, interval string, start, end int64) ([]Kline, error) {
	if market != MarketFuturesUSDT {
		return nil, fmt.Errorf("binance historical source does not support market %q", market)
	}
	rows, err := binanceapi.GetHistoricalKlines(ctx, symbol, interval, start, end)
	if err != nil {
		return nil, err
	}
	result := make([]Kline, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		parse := func(value string) (float64, error) { return strconv.ParseFloat(value, 64) }
		open, err := parse(row.Open)
		if err != nil {
			return nil, err
		}
		high, err := parse(row.High)
		if err != nil {
			return nil, err
		}
		low, err := parse(row.Low)
		if err != nil {
			return nil, err
		}
		closePrice, err := parse(row.Close)
		if err != nil {
			return nil, err
		}
		volume, err := parse(row.Volume)
		if err != nil {
			return nil, err
		}
		quoteVolume, err := parse(row.QuoteAssetVolume)
		if err != nil {
			return nil, err
		}
		takerBase, err := parse(row.TakerBuyBaseAssetVolume)
		if err != nil {
			return nil, err
		}
		takerQuote, err := parse(row.TakerBuyQuoteAssetVolume)
		if err != nil {
			return nil, err
		}
		result = append(result, Kline{Market: market, Symbol: strings.ToUpper(symbol), Interval: interval, OpenTime: row.OpenTime, CloseTime: row.CloseTime, Open: open, High: high, Low: low, Close: closePrice, Volume: volume, QuoteVolume: quoteVolume, TradeCount: row.TradeNum, TakerBuyBaseVolume: takerBase, TakerBuyQuoteVolume: takerQuote, Source: SourceBinanceREST})
	}
	return result, nil
}

func (BinanceSource) Funding(ctx context.Context, market, symbol string, start, end int64) ([]FundingRate, error) {
	if market != MarketFuturesUSDT {
		return nil, fmt.Errorf("binance historical source does not support market %q", market)
	}
	rows, err := binanceapi.GetHistoricalFundingRates(ctx, symbol, start, end)
	if err != nil {
		return nil, err
	}
	result := make([]FundingRate, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		rate, err := strconv.ParseFloat(row.FundingRate, 64)
		if err != nil {
			return nil, err
		}
		mark, _ := strconv.ParseFloat(row.MarkPrice, 64)
		result = append(result, FundingRate{Market: market, Symbol: strings.ToUpper(symbol), FundingTime: row.FundingTime, FundingRate: rate, MarkPrice: mark, Source: SourceBinanceREST})
	}
	return result, nil
}

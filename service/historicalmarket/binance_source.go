package historicalmarket

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	binanceapi "go_binance_futures/feature/api/binance"

	"github.com/adshao/go-binance/v2/futures"
)

type BinanceSource struct{}

func (source BinanceSource) Klines(ctx context.Context, market, symbol, interval string, start, end int64) ([]Kline, error) {
	return source.KlinesWithProgress(ctx, market, symbol, interval, start, end, nil)
}

func (BinanceSource) KlinesWithProgress(ctx context.Context, market, symbol, interval string, start, end int64, progress KlineProgressCallback) ([]Kline, error) {
	if market != MarketFuturesUSDT {
		return nil, fmt.Errorf("binance historical source does not support market %q", market)
	}
	estimated, _ := expectedKlineCount(interval, start, end)
	rows, err := binanceapi.GetHistoricalKlinesWithProgress(ctx, symbol, interval, start, end, func(completed int) {
		if progress != nil {
			progress(completed, estimated)
		}
	})
	if err != nil {
		return nil, err
	}
	result := make([]Kline, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		item, err := convertBinanceKline(market, symbol, interval, row)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if progress != nil {
		progress(len(result), estimated)
	}
	return result, nil
}

func (BinanceSource) EarliestKline(ctx context.Context, market, symbol, interval string) (Kline, error) {
	if market != MarketFuturesUSDT {
		return Kline{}, fmt.Errorf("binance historical source does not support market %q", market)
	}
	row, err := binanceapi.GetEarliestHistoricalKline(ctx, symbol, interval)
	if err != nil {
		return Kline{}, err
	}
	return convertBinanceKline(market, symbol, interval, row)
}

func convertBinanceKline(market, symbol, interval string, row *futures.Kline) (Kline, error) {
	if row == nil {
		return Kline{}, fmt.Errorf("nil Binance K-line")
	}
	parse := func(value string) (float64, error) { return strconv.ParseFloat(value, 64) }
	open, err := parse(row.Open)
	if err != nil {
		return Kline{}, err
	}
	high, err := parse(row.High)
	if err != nil {
		return Kline{}, err
	}
	low, err := parse(row.Low)
	if err != nil {
		return Kline{}, err
	}
	closePrice, err := parse(row.Close)
	if err != nil {
		return Kline{}, err
	}
	volume, err := parse(row.Volume)
	if err != nil {
		return Kline{}, err
	}
	quoteVolume, err := parse(row.QuoteAssetVolume)
	if err != nil {
		return Kline{}, err
	}
	takerBase, err := parse(row.TakerBuyBaseAssetVolume)
	if err != nil {
		return Kline{}, err
	}
	takerQuote, err := parse(row.TakerBuyQuoteAssetVolume)
	if err != nil {
		return Kline{}, err
	}
	return Kline{Market: market, Symbol: strings.ToUpper(symbol), Interval: interval, OpenTime: row.OpenTime, CloseTime: row.CloseTime, Open: open, High: high, Low: low, Close: closePrice, Volume: volume, QuoteVolume: quoteVolume, TradeCount: row.TradeNum, TakerBuyBaseVolume: takerBase, TakerBuyQuoteVolume: takerQuote, Source: SourceBinanceREST}, nil
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

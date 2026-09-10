package historicalmarket

import (
	"context"
	"fmt"
	"strings"
)

const (
	MarketFuturesUSDT = "futures_usdt"
	SourceBinanceREST = "binance_rest"
)

var SupportedIntervals = []string{"1m", "3m", "5m", "15m", "30m", "1h", "2h", "4h", "6h", "8h", "12h", "1d", "3d", "1w", "1M"}

var intervalTables = map[string]string{
	"1m": "market_klines_1m", "3m": "market_klines_3m", "5m": "market_klines_5m",
	"15m": "market_klines_15m", "30m": "market_klines_30m", "1h": "market_klines_1h",
	"2h": "market_klines_2h", "4h": "market_klines_4h", "6h": "market_klines_6h",
	"8h": "market_klines_8h", "12h": "market_klines_12h", "1d": "market_klines_1d",
	"3d": "market_klines_3d", "1w": "market_klines_1w", "1M": "market_klines_1mo",
}

func KlineTable(interval string) (string, error) {
	table, ok := intervalTables[strings.TrimSpace(interval)]
	if !ok {
		return "", fmt.Errorf("unsupported historical K-line interval %q", interval)
	}
	return table, nil
}

type Kline struct {
	Market              string  `json:"market,omitempty"`
	Symbol              string  `json:"symbol"`
	Interval            string  `json:"interval"`
	Source              string  `json:"source,omitempty"`
	SourceRef           string  `json:"source_ref,omitempty"`
	OpenTime            int64   `json:"open_time"`
	CloseTime           int64   `json:"close_time"`
	TradeCount          int64   `json:"trade_count,omitempty"`
	Open                float64 `json:"open"`
	High                float64 `json:"high"`
	Low                 float64 `json:"low"`
	Close               float64 `json:"close"`
	Volume              float64 `json:"volume,omitempty"`
	QuoteVolume         float64 `json:"quote_volume,omitempty"`
	TakerBuyBaseVolume  float64 `json:"taker_buy_base_volume,omitempty"`
	TakerBuyQuoteVolume float64 `json:"taker_buy_quote_volume,omitempty"`
}

type FundingRate struct {
	Market      string  `json:"market,omitempty"`
	Symbol      string  `json:"symbol"`
	Source      string  `json:"source,omitempty"`
	SourceRef   string  `json:"source_ref,omitempty"`
	FundingTime int64   `json:"funding_time"`
	FundingRate float64 `json:"funding_rate"`
	MarkPrice   float64 `json:"mark_price,omitempty"`
}

type ImportRequest struct {
	Source    string        `json:"source"`
	SourceRef string        `json:"source_ref,omitempty"`
	Klines    []Kline       `json:"klines,omitempty"`
	Funding   []FundingRate `json:"funding,omitempty"`
}

type ImportResult struct {
	BatchID     string `json:"batch_id"`
	TotalRows   int    `json:"total_rows"`
	WrittenRows int    `json:"written_rows"`
	InvalidRows int    `json:"invalid_rows"`
}

type Source interface {
	Klines(context.Context, string, string, string, int64, int64) ([]Kline, error)
	Funding(context.Context, string, string, int64, int64) ([]FundingRate, error)
}

// KlineProgressCallback reports completed/estimated rows while a remote K-line
// source is paging data. It is optional and does not change the Source contract.
type KlineProgressCallback func(completed, total int)

type KlineProgressSource interface {
	KlinesWithProgress(context.Context, string, string, string, int64, int64, KlineProgressCallback) ([]Kline, error)
}

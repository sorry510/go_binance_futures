package backtest

import (
	"reflect"
	"testing"

	"go_binance_futures/service/historicalmarket"
)

func TestReplayKlineConversionPreservesDatasetHashInputs(t *testing.T) {
	full := []historicalmarket.Kline{
		{Market: historicalmarket.MarketFuturesUSDT, Symbol: "BTCUSDT", Interval: "1m", Source: "fixture", SourceRef: "ignored", OpenTime: 1000, CloseTime: 1999, Open: 100, High: 110, Low: 90, Close: 105, Volume: 12, QuoteVolume: 1234, TradeCount: 17, TakerBuyBaseVolume: 4, TakerBuyQuoteVolume: 555},
		{Market: historicalmarket.MarketFuturesUSDT, Symbol: "BTCUSDT", Interval: "1m", Source: "fixture2", SourceRef: "ignored2", OpenTime: 2000, CloseTime: 2999, Open: 105, High: 112, Low: 101, Close: 109, Volume: 13, QuoteVolume: 1334, TradeCount: 19, TakerBuyBaseVolume: 5, TakerBuyQuoteVolume: 666},
	}
	lite := make([]historicalmarket.ReplayKline, 0, len(full))
	for _, row := range full {
		lite = append(lite, historicalmarket.ReplayKline{OpenTime: row.OpenTime, CloseTime: row.CloseTime, TradeCount: row.TradeCount, Open: row.Open, High: row.High, Low: row.Low, Close: row.Close, Volume: row.Volume, QuoteVolume: row.QuoteVolume, TakerBuyQuoteVolume: row.TakerBuyQuoteVolume})
	}
	oldBars := convertHistoricalKlines(full)
	newBars := convertReplayKlines("BTCUSDT", "1m", lite)
	if !reflect.DeepEqual(oldBars, newBars) {
		t.Fatalf("replay conversion changed Bars\nold=%+v\nnew=%+v", oldBars, newBars)
	}
	oldDataset := Dataset{Market: historicalmarket.MarketFuturesUSDT, Symbol: "BTCUSDT", ExecutionInterval: "1m", Intervals: []string{"1m"}, StartTime: 1000, EndTime: 2999, WarmupStartTime: 1000, Bars: map[string][]Bar{BarSeriesKey("BTCUSDT", "1m"): oldBars}}
	newDataset := oldDataset
	newDataset.Bars = map[string][]Bar{BarSeriesKey("BTCUSDT", "1m"): newBars}
	oldDataset.DatasetSpecHash = DatasetSpecHash(oldDataset)
	newDataset.DatasetSpecHash = DatasetSpecHash(newDataset)
	if DatasetDataHash(oldDataset) != DatasetDataHash(newDataset) {
		t.Fatal("lightweight replay query inputs changed DatasetDataHash")
	}
}

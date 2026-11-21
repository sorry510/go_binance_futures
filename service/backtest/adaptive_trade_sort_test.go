package backtest

import (
	"math/rand"
	"reflect"
	"sort"
	"testing"

	"go_binance_futures/service/historicalmarket"
)

func legacySortAdaptiveTrades(trades []historicalmarket.PublicDataTrade) {
	sort.Slice(trades, func(i, j int) bool {
		if trades[i].TradeTime == trades[j].TradeTime {
			return trades[i].TradeID < trades[j].TradeID
		}
		return trades[i].TradeTime < trades[j].TradeTime
	})
}

func TestAdaptiveSortedTradeFastPathMatchesLegacy(t *testing.T) {
	tests := [][]historicalmarket.PublicDataTrade{
		nil,
		{{TradeTime: 123, TradeID: 4}},
		{{TradeTime: 123, TradeID: 1}, {TradeTime: 123, TradeID: 2}},
		{{TradeTime: 124, TradeID: 5}, {TradeTime: 123, TradeID: 2}, {TradeTime: 123, TradeID: 1}},
	}
	generator := rand.New(rand.NewSource(510))
	for size := 2; size <= 1000; size *= 5 {
		trades := make([]historicalmarket.PublicDataTrade, size)
		for i := range trades {
			trades[i] = historicalmarket.PublicDataTrade{
				TradeTime: 1_000 + generator.Int63n(500),
				TradeID:   int64(i + 1),
			}
		}
		tests = append(tests, trades)
	}
	for _, input := range tests {
		previous := append([]historicalmarket.PublicDataTrade(nil), input...)
		fast := append([]historicalmarket.PublicDataTrade(nil), input...)
		legacySortAdaptiveTrades(previous)
		ensureChronologicalTrades(fast)
		if !reflect.DeepEqual(previous, fast) {
			t.Fatalf("not equal to legacy sort for %d input trades", len(input))
		}
		ensureChronologicalTrades(fast)
		if !reflect.DeepEqual(previous, fast) {
			t.Fatal("second sort changed already sorted trades")
		}
	}
}

func benchmarkSortedTradeSort(b *testing.B, fast bool) {
	trades := make([]historicalmarket.PublicDataTrade, 128)
	for i := range trades {
		trades[i].TradeTime = 1_000 + int64(i/4)
		trades[i].TradeID = int64(i + 1)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if fast {
			ensureChronologicalTrades(trades)
		} else {
			legacySortAdaptiveTrades(trades)
		}
	}
}

func BenchmarkAdaptiveTradeSortLegacy(b *testing.B) {
	benchmarkSortedTradeSort(b, false)
}

func BenchmarkAdaptiveTradeSortSortedFastPath(b *testing.B) {
	benchmarkSortedTradeSort(b, true)
}

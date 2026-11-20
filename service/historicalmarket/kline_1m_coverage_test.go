package historicalmarket

import (
	"context"
	"testing"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

func TestCompleteKline1mChunkMonthsFastCoverageAndPartialRange(t *testing.T) {
	setupRepositoryTest(t)
	ctx := context.Background()
	repo := NewRepository(nil)
	start := time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)
	bars := []Kline{minuteBar(start, 101), minuteBar(start.Add(time.Minute), 102)}
	if _, err := repo.Import(ctx, ImportRequest{
		Source: SourceBinancePublicData, Klines: bars,
	}); err != nil {
		t.Fatal(err)
	}

	monthKey := start.UnixMilli()
	covered, err := repo.CompleteKline1mChunkMonths(ctx,
		MarketFuturesUSDT, "BTCUSDT", start.UnixMilli(),
		start.Add(2*time.Minute-time.Millisecond).UnixMilli())
	if err != nil || !covered[monthKey] {
		t.Fatalf("two-minute range should be covered by chunk metadata: %+v err=%v", covered, err)
	}

	covered, err = repo.CompleteKline1mChunkMonths(ctx,
		MarketFuturesUSDT, "BTCUSDT", start.UnixMilli(),
		start.Add(3*time.Minute-time.Millisecond).UnixMilli())
	if err != nil || covered[monthKey] {
		t.Fatalf("partial chunk must not claim missing 3rd minute: %+v err=%v", covered, err)
	}

	// Metadata that would falsely claim full coverage must be rejected.
	if _, err := orm.NewOrm().Raw(
		"UPDATE market_klines_1m_chunks SET point_count=99 WHERE month_start=?", monthKey,
	).Exec(); err != nil {
		t.Fatal(err)
	}
	covered, err = repo.CompleteKline1mChunkMonths(ctx,
		MarketFuturesUSDT, "BTCUSDT", start.UnixMilli(),
		start.Add(2*time.Minute-time.Millisecond).UnixMilli())
	if err != nil || covered[monthKey] {
		t.Fatalf("invalid point count must not trigger fast skip: %+v err=%v", covered, err)
	}
}

func TestCompleteKline1mChunkMonthsEmptyRange(t *testing.T) {
	setupRepositoryTest(t)
	covered, err := NewRepository(nil).CompleteKline1mChunkMonths(
		context.Background(), MarketFuturesUSDT, "BTCUSDT", 200000, 100000)
	if err != nil || len(covered) != 0 {
		t.Fatalf("invalid time range should not claim coverage: %+v err=%v", covered, err)
	}
}

package backtest

import (
	"context"
	"sync"
	"testing"
	"time"

	"go_binance_futures/service/historicalmarket"
)

type trackedHistorySource struct {
	mu           sync.Mutex
	klineCalls   int
	fundingCalls int
}

func (source *trackedHistorySource) Klines(ctx context.Context, market, symbol, interval string, start, end int64) ([]historicalmarket.Kline, error) {
	source.mu.Lock()
	source.klineCalls++
	source.mu.Unlock()
	return fixtureHistorySource{}.Klines(ctx, market, symbol, interval, start, end)
}

func (source *trackedHistorySource) Funding(_ context.Context, market, symbol string, start, _ int64) ([]historicalmarket.FundingRate, error) {
	source.mu.Lock()
	source.fundingCalls++
	source.mu.Unlock()
	return []historicalmarket.FundingRate{{Market: market, Symbol: symbol, FundingTime: start, FundingRate: 0.0001, MarkPrice: 100}}, nil
}
func (source *trackedHistorySource) counts() (int, int) {
	source.mu.Lock()
	defer source.mu.Unlock()
	return source.klineCalls, source.fundingCalls
}

func TestPrefetchPlanAlwaysIncludesOneMinuteReplayAndStrategyIntervals(t *testing.T) {
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	technologyJSON := `{"ma":[{"name":"ma_fast","enable":true,"kline_interval":"15m","period":7}]}`
	plan, err := (DatasetBuilder{WarmupBars: 2}).PrefetchPlan(DatasetRequest{
		Symbol: "dogeusdt", StartTime: start.UnixMilli(), EndTime: start.Add(time.Hour).UnixMilli(), TechnologyJSON: technologyJSON,
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.ReplayInterval != ReplayInterval || ReplayInterval != "1m" {
		t.Fatalf("replay interval=%q", plan.ReplayInterval)
	}
	if len(plan.Intervals) != 2 || plan.Intervals[0] != "1m" || plan.Intervals[1] != "15m" {
		t.Fatalf("unexpected required intervals: %+v", plan.Intervals)
	}
	if plan.WarmupStartTime != start.Add(-30*time.Minute).UnixMilli() {
		t.Fatalf("warmup start=%d", plan.WarmupStartTime)
	}
}

func TestPrefetchFillsDataSoImmediateBuildUsesLocalCache(t *testing.T) {
	setupBacktestStoreTest(t)
	start := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	source := &trackedHistorySource{}
	builder := DatasetBuilder{Repository: historicalmarket.NewRepository(source), WarmupBars: 2}
	request := DatasetRequest{
		Symbol: "DOGEUSDT", ExecutionInterval: ReplayInterval,
		StartTime: start.UnixMilli(), EndTime: start.Add(5 * time.Minute).UnixMilli(), TechnologyJSON: "{}",
	}
	result, err := builder.Prefetch(context.Background(), request, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Plan.ReplayInterval != "1m" || result.RemoteRows == 0 || result.RemoteCalls == 0 {
		t.Fatalf("unexpected prefetch result: %+v", result)
	}
	klineCalls, fundingCalls := source.counts()
	if klineCalls != 1 || fundingCalls != 1 {
		t.Fatalf("unexpected remote calls after prefetch: klines=%d funding=%d", klineCalls, fundingCalls)
	}
	if _, err := builder.Build(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	afterKlines, afterFunding := source.counts()
	if afterKlines != klineCalls || afterFunding != fundingCalls {
		t.Fatalf("build refetched Binance data after successful prefetch: before=%d/%d after=%d/%d", klineCalls, fundingCalls, afterKlines, afterFunding)
	}
}

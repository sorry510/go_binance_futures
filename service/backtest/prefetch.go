package backtest

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"go_binance_futures/service/historicalmarket"
)

type PrefetchRequest struct {
	StrategyTemplateID int64  `json:"strategy_template_id"`
	Symbol             string `json:"symbol"`
	StartTime          int64  `json:"start_time"`
	EndTime            int64  `json:"end_time"`
}

type PrefetchPlan struct {
	Symbol          string   `json:"symbol"`
	ReplayInterval  string   `json:"replay_interval"`
	Intervals       []string `json:"intervals"`
	StartTime       int64    `json:"start_time"`
	EndTime         int64    `json:"end_time"`
	WarmupStartTime int64    `json:"warmup_start_time"`
}
type PrefetchResult struct {
	Plan        PrefetchPlan `json:"plan"`
	RemoteCalls int          `json:"remote_calls"`
	RemoteRows  int          `json:"remote_rows"`
}

type prefetchCountingSource struct {
	inner historicalmarket.Source
	mu    sync.Mutex
	calls int
	rows  int
}

func (source *prefetchCountingSource) add(rows int) {
	source.mu.Lock()
	source.calls++
	source.rows += rows
	source.mu.Unlock()
}

func (source *prefetchCountingSource) stats() (int, int) {
	source.mu.Lock()
	defer source.mu.Unlock()
	return source.calls, source.rows
}
func (source *prefetchCountingSource) Klines(ctx context.Context, market, symbol, interval string, start, end int64) ([]historicalmarket.Kline, error) {
	if source.inner == nil {
		return nil, fmt.Errorf("historical K-line source is unavailable")
	}
	rows, err := source.inner.Klines(ctx, market, symbol, interval, start, end)
	if err == nil {
		source.add(len(rows))
	}
	return rows, err
}

func (source *prefetchCountingSource) Funding(ctx context.Context, market, symbol string, start, end int64) ([]historicalmarket.FundingRate, error) {
	if source.inner == nil {
		return nil, fmt.Errorf("historical funding source is unavailable")
	}
	rows, err := source.inner.Funding(ctx, market, symbol, start, end)
	if err == nil {
		source.add(len(rows))
	}
	return rows, err
}

func (builder DatasetBuilder) PrefetchPlan(request DatasetRequest) (PrefetchPlan, error) {
	if builder.WarmupBars <= 0 {
		builder.WarmupBars = DefaultWarmupBars
	}
	request.Symbol = strings.ToUpper(strings.TrimSpace(request.Symbol))
	if request.Symbol == "" || !strings.HasSuffix(request.Symbol, "USDT") {
		return PrefetchPlan{}, fmt.Errorf("symbol must be a USDT futures contract")
	}
	if request.StartTime <= 0 || request.EndTime <= request.StartTime {
		return PrefetchPlan{}, fmt.Errorf("valid historical start_time and end_time are required")
	}
	if request.EndTime > time.Now().Add(time.Minute).UnixMilli() {
		return PrefetchPlan{}, fmt.Errorf("historical end_time must not be in the future")
	}
	intervals, err := strategyIntervals(ReplayInterval, request.TechnologyJSON)
	if err != nil {
		return PrefetchPlan{}, err
	}
	warmupStart := request.StartTime
	for _, interval := range intervals {
		candidate, err := subtractBars(request.StartTime, interval, builder.WarmupBars)
		if err != nil {
			return PrefetchPlan{}, err
		}
		if candidate < warmupStart {
			warmupStart = candidate
		}
	}
	return PrefetchPlan{
		Symbol: request.Symbol, ReplayInterval: ReplayInterval, Intervals: intervals,
		StartTime: request.StartTime, EndTime: request.EndTime, WarmupStartTime: warmupStart,
	}, nil
}
func (builder DatasetBuilder) Prefetch(ctx context.Context, request DatasetRequest, progress ProgressCallback) (PrefetchResult, error) {
	plan, err := builder.PrefetchPlan(request)
	if err != nil {
		return PrefetchResult{}, err
	}
	repo := builder.Repository
	if repo == nil {
		repo = historicalmarket.DefaultRepository()
	}
	counter := &prefetchCountingSource{inner: repo.Source}
	fetchRepo := *repo
	fetchRepo.Source = counter
	warmupBars := builder.WarmupBars
	if warmupBars <= 0 {
		warmupBars = DefaultWarmupBars
	}
	total := len(plan.Intervals) + 1
	completed := 0
	report := func() {
		if progress != nil {
			progress(completed, total)
		}
	}
	report()
	for _, interval := range plan.Intervals {
		start, err := subtractBars(plan.StartTime, interval, warmupBars)
		if err != nil {
			return PrefetchResult{}, err
		}
		if _, err := fetchRepo.LoadKlines(ctx, historicalmarket.MarketFuturesUSDT, plan.Symbol, interval, start, plan.EndTime); err != nil {
			return PrefetchResult{}, fmt.Errorf("prefetch %s %s: %w", plan.Symbol, interval, err)
		}
		completed++
		report()
	}
	if _, err := fetchRepo.LoadFunding(ctx, historicalmarket.MarketFuturesUSDT, plan.Symbol, plan.StartTime, plan.EndTime); err != nil {
		return PrefetchResult{}, fmt.Errorf("prefetch funding %s: %w", plan.Symbol, err)
	}
	completed++
	report()
	calls, rows := counter.stats()
	return PrefetchResult{Plan: plan, RemoteCalls: calls, RemoteRows: rows}, nil
}

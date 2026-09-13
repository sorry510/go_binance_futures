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

const defaultPrefetchChunkBars = 10000

type prefetchKlineChunk struct {
	Start int64
	End   int64
}

func (builder DatasetBuilder) prefetchKlineChunks(interval string, start, end int64) ([]prefetchKlineChunk, error) {
	if start > end {
		return nil, nil
	}
	chunkBars := builder.PrefetchChunkBars
	if chunkBars <= 0 {
		chunkBars = defaultPrefetchChunkBars
	}
	chunks := make([]prefetchKlineChunk, 0)
	for cursor := start; cursor <= end; {
		var next int64
		if interval == "1M" {
			next = time.UnixMilli(cursor).UTC().AddDate(0, chunkBars, 0).UnixMilli()
		} else {
			duration, err := intervalDuration(interval, time.UnixMilli(cursor))
			if err != nil {
				return nil, err
			}
			next = cursor + int64(chunkBars)*duration.Milliseconds()
		}
		if next <= cursor {
			return nil, fmt.Errorf("invalid prefetch chunk progression for %s", interval)
		}
		chunkEnd := next - 1
		if chunkEnd > end {
			chunkEnd = end
		}
		chunks = append(chunks, prefetchKlineChunk{Start: cursor, End: chunkEnd})
		if chunkEnd >= end {
			break
		}
		cursor = chunkEnd + 1
	}
	return chunks, nil
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
	return source.KlinesWithProgress(ctx, market, symbol, interval, start, end, nil)
}

func (source *prefetchCountingSource) KlinesWithProgress(ctx context.Context, market, symbol, interval string, start, end int64, progress historicalmarket.KlineProgressCallback) ([]historicalmarket.Kline, error) {
	if source.inner == nil {
		return nil, fmt.Errorf("historical K-line source is unavailable")
	}
	var rows []historicalmarket.Kline
	var err error
	progressive := false
	if inner, ok := source.inner.(historicalmarket.KlineProgressSource); ok {
		progressive = true
		rows, err = inner.KlinesWithProgress(ctx, market, symbol, interval, start, end, progress)
	} else {
		rows, err = source.inner.Klines(ctx, market, symbol, interval, start, end)
	}
	if err == nil {
		source.add(len(rows))
		if progress != nil && !progressive {
			progress(len(rows), len(rows))
		}
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
	const (
		intervalProgressUnits = 1000
		fundingProgressUnits  = 100
	)
	total := len(plan.Intervals)*intervalProgressUnits + fundingProgressUnits
	completed := 0
	report := func(value int) {
		if progress != nil {
			progress(value, total)
		}
	}
	report(0)
	for _, interval := range plan.Intervals {
		start, err := subtractBars(plan.StartTime, interval, warmupBars)
		if err != nil {
			return PrefetchResult{}, err
		}
		chunks, err := builder.prefetchKlineChunks(interval, start, plan.EndTime)
		if err != nil {
			return PrefetchResult{}, err
		}
		base := completed
		for chunkIndex, chunk := range chunks {
			if err := ctx.Err(); err != nil {
				return PrefetchResult{}, err
			}
			if _, err := fetchRepo.LoadKlinesWithProgress(ctx, historicalmarket.MarketFuturesUSDT, plan.Symbol, interval, chunk.Start, chunk.End, func(done, chunkTotal int) {
				if chunkTotal <= 0 || len(chunks) == 0 {
					return
				}
				chunkFraction := float64(done) / float64(chunkTotal)
				if chunkFraction < 0 {
					chunkFraction = 0
				}
				if chunkFraction > 1 {
					chunkFraction = 1
				}
				intervalFraction := (float64(chunkIndex) + chunkFraction) / float64(len(chunks))
				units := int(intervalFraction * float64(intervalProgressUnits))
				if units >= intervalProgressUnits {
					units = intervalProgressUnits - 1
				}
				report(base + units)
			}); err != nil {
				return PrefetchResult{}, fmt.Errorf("prefetch %s %s chunk %d/%d: %w", plan.Symbol, interval, chunkIndex+1, len(chunks), err)
			}
			// A chunk is counted complete only after LoadKlinesWithProgress has re-read
			// the cache and verified that no gaps remain. This keeps the UI moving
			// during multi-year imports without pretending a download is complete
			// before its rows have actually been persisted.
			units := (chunkIndex + 1) * intervalProgressUnits / len(chunks)
			if units >= intervalProgressUnits {
				units = intervalProgressUnits - 1
			}
			report(base + units)
		}
		completed += intervalProgressUnits
		report(completed)
	}
	if _, err := fetchRepo.LoadFunding(ctx, historicalmarket.MarketFuturesUSDT, plan.Symbol, plan.StartTime, plan.EndTime); err != nil {
		return PrefetchResult{}, fmt.Errorf("prefetch funding %s: %w", plan.Symbol, err)
	}
	completed += fundingProgressUnits
	report(completed)
	calls, rows := counter.stats()
	return PrefetchResult{Plan: plan, RemoteCalls: calls, RemoteRows: rows}, nil
}

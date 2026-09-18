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

type PrefetchProgress struct {
	Stage     string
	Detail    string
	Completed int
	Total     int
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
	warmupBars, err := builder.effectiveWarmupBars(request.TechnologyJSON)
	if err != nil {
		return PrefetchPlan{}, err
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
		candidate, err := subtractBars(request.StartTime, interval, warmupBars)
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
	return builder.PrefetchDetailed(ctx, request, func(item PrefetchProgress) {
		if progress != nil {
			progress(item.Completed, item.Total)
		}
	})
}

func (builder DatasetBuilder) PrefetchDetailed(ctx context.Context, request DatasetRequest, progress func(PrefetchProgress)) (PrefetchResult, error) {
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
	warmupBars, err := builder.effectiveWarmupBars(request.TechnologyJSON)
	if err != nil {
		return PrefetchResult{}, err
	}
	const (
		intervalProgressUnits = 1000
		fundingProgressUnits  = 100
		publicProgressUnits   = 400
	)
	total := len(plan.Intervals)*intervalProgressUnits + fundingProgressUnits
	completed := 0
	lastReported := -1
	report := func(stage, detail string, value int) {
		if value < lastReported {
			value = lastReported
		}
		if value > total {
			value = total
		}
		lastReported = value
		if progress != nil {
			progress(PrefetchProgress{Stage: stage, Detail: detail, Completed: value, Total: total})
		}
	}
	report("checking", "", 0)

	publicCalls, publicRows := 0, 0
	var publicClient *historicalmarket.PublicDataClient
	defer func() {
		if publicClient != nil {
			_ = publicClient.Close()
		}
	}()

	for _, interval := range plan.Intervals {
		start, err := subtractBars(plan.StartTime, interval, warmupBars)
		if err != nil {
			return PrefetchResult{}, err
		}
		chunks, err := builder.prefetchKlineChunks(interval, start, plan.EndTime)
		if err != nil {
			return PrefetchResult{}, err
		}
		usePublicMonthly := interval == ReplayInterval && canUsePublicDataPrefetch(repo.Source)
		if usePublicMonthly {
			months := prefetchMonthRanges(start, plan.EndTime)
			chunks = make([]prefetchKlineChunk, 0, len(months))
			for _, month := range months {
				chunks = append(chunks, prefetchKlineChunk{Start: month.Start, End: month.End})
			}
		}
		base := completed
		restBase := base
		restUnits := intervalProgressUnits
		completeMonths := map[int64]bool{}

		if usePublicMonthly {
			months := completedPrefetchMonths(start, plan.EndTime, time.Now())
			if len(months) > 0 {
				if publicClient == nil {
					publicClient, _ = newPrefetchPublicDataClient()
				}
				if publicClient != nil {
					stats, err := prefetchPublicMonthly1m(ctx, repo, publicClient, plan.Symbol, months,
						func(index int, stage, detail string) {
							units := index * publicProgressUnits / len(months)
							report(stage, detail, base+units)
						})
					if err != nil {
						return PrefetchResult{}, err
					}
					publicCalls += stats.Calls
					publicRows += stats.Rows
					completeMonths = stats.CompleteMonths
					restBase = base + publicProgressUnits
					restUnits = intervalProgressUnits - publicProgressUnits
				} else {
					report("rest_fallback", ReplayInterval, base)
				}
			}
		}

		for chunkIndex, chunk := range chunks {
			if err := ctx.Err(); err != nil {
				return PrefetchResult{}, err
			}
			if usePublicMonthly {
				at := time.UnixMilli(chunk.Start).UTC()
				monthKey := time.Date(at.Year(), at.Month(), 1, 0, 0, 0, 0, time.UTC).UnixMilli()
				if completeMonths[monthKey] {
					units := (chunkIndex + 1) * restUnits / maxInt(len(chunks), 1)
					report("checking", fmt.Sprintf("%s %d/%d", interval, chunkIndex+1, len(chunks)), restBase+units)
					continue
				}
			}
			complete, _, err := fetchRepo.KlineRangeComplete(
				ctx, historicalmarket.MarketFuturesUSDT, plan.Symbol, interval, chunk.Start, chunk.End,
			)
			if err != nil {
				return PrefetchResult{}, fmt.Errorf("check %s %s chunk %d/%d: %w", plan.Symbol, interval, chunkIndex+1, len(chunks), err)
			}
			if complete {
				units := (chunkIndex + 1) * restUnits / maxInt(len(chunks), 1)
				report("checking", fmt.Sprintf("%s %d/%d", interval, chunkIndex+1, len(chunks)), restBase+units)
				continue
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
				units := int(intervalFraction * float64(restUnits))
				if units >= restUnits {
					units = restUnits - 1
				}
				report("rest_fallback", fmt.Sprintf("%s %d/%d", interval, chunkIndex+1, len(chunks)), restBase+units)
			}); err != nil {
				return PrefetchResult{}, fmt.Errorf("prefetch %s %s chunk %d/%d: %w", plan.Symbol, interval, chunkIndex+1, len(chunks), err)
			}
			units := (chunkIndex + 1) * restUnits / maxInt(len(chunks), 1)
			report("checking", fmt.Sprintf("%s %d/%d", interval, chunkIndex+1, len(chunks)), restBase+units)
		}
		completed += intervalProgressUnits
		report("checking", interval, completed)
	}

	report("funding", plan.Symbol, completed)
	if _, err := fetchRepo.LoadFunding(ctx, historicalmarket.MarketFuturesUSDT, plan.Symbol, plan.StartTime, plan.EndTime); err != nil {
		return PrefetchResult{}, fmt.Errorf("prefetch funding %s: %w", plan.Symbol, err)
	}
	completed += fundingProgressUnits
	report("completed", "", completed)
	calls, rows := counter.stats()
	return PrefetchResult{
		Plan: plan, RemoteCalls: calls + publicCalls, RemoteRows: rows + publicRows,
	}, nil
}

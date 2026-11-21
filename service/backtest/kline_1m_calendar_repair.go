package backtest

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go_binance_futures/service/historicalmarket"
)

type Kline1mRepairRequest struct {
	Symbol string `json:"symbol"`
	From   string `json:"from"`
	To     string `json:"to"`
}

// StartKline1mRepair uses the existing single active prefetch slot, so a
// calendar repair and the strategy prefetch cannot overwrite the same months
// concurrently. No strategy/template is needed for plain 1m market data.
func (manager *Manager) StartKline1mRepair(req Kline1mRepairRequest) (PrefetchSummary, error) {
	req.Symbol = strings.ToUpper(strings.TrimSpace(req.Symbol))
	if len(req.Symbol) < 5 || len(req.Symbol) > 32 || !strings.HasSuffix(req.Symbol, "USDT") {
		return PrefetchSummary{}, fmt.Errorf("invalid USDT futures symbol")
	}
	for _, c := range req.Symbol {
		if (c < 'A' || c > 'Z') && (c < '0' || c > '9') {
			return PrefetchSummary{}, fmt.Errorf("invalid symbol")
		}
	}
	first, last, err := historicalmarket.ParseMinuteMonthRange(req.From, req.To, time.Now())
	if err != nil {
		return PrefetchSummary{}, err
	}
	end := last.AddDate(0, 1, 0).Add(-time.Millisecond)
	if end.After(time.Now().Add(-time.Minute)) {
		end = time.Now().UTC().Truncate(time.Minute).Add(-time.Millisecond)
	}
	if !end.After(first) {
		return PrefetchSummary{}, fmt.Errorf("no closed 1m candles in selected date range")
	}
	now := time.Now().UnixMilli()
	job := PrefetchSummary{
		JobID:  "prefetch_" + newRunID()[4:],
		Status: "queued", Stage: "queued", Symbol: req.Symbol,
		ReplayInterval: ReplayInterval, Intervals: []string{ReplayInterval},
		StartTime: first.UnixMilli(), EndTime: end.UnixMilli(),
		CreatedAt: now, UpdatedAt: now,
	}
	manager.prefetchMu.Lock()
	if manager.prefetchActive != "" {
		active := manager.prefetchJobs[manager.prefetchActive]
		if active.Status == "queued" || active.Status == "running" {
			manager.prefetchMu.Unlock()
			return PrefetchSummary{}, fmt.Errorf("another historical data download is already running")
		}
	}
	manager.prefetchJobs[job.JobID] = job
	manager.prefetchActive = job.JobID
	manager.prefetchMu.Unlock()
	go manager.runKline1mRepair(job.JobID)
	return clonePrefetchSummary(job), nil
}

func (manager *Manager) runKline1mRepair(jobID string) {
	manager.updatePrefetch(jobID, func(job *PrefetchSummary) {
		job.Status, job.Stage, job.Progress = "running", "checking", 1
	})
	job, err := manager.GetPrefetch(jobID)
	if err != nil {
		return
	}
	fail := func(err error) {
		manager.updatePrefetch(jobID, func(job *PrefetchSummary) {
			job.Status, job.Stage, job.Error, job.Progress = "failed", "failed", err.Error(), 100
			job.CompletedAt = time.Now().UnixMilli()
		})
	}
	ctx := context.Background()
	repo := manager.builder.Repository
	if repo == nil {
		repo = historicalmarket.DefaultRepository()
	}
	start := job.StartTime
	if effectiveStart, has, err := repo.EffectiveKlineStart(ctx, historicalmarket.MarketFuturesUSDT, job.Symbol, ReplayInterval, start, job.EndTime); err == nil && has {
		start = effectiveStart
	} else if err == nil && !has {
		fail(fmt.Errorf("no listed 1m candles in selected range"))
		return
	}
	if start > job.EndTime {
		fail(fmt.Errorf("no available 1m candles in selected range"))
		return
	}
	months := prefetchMonthRanges(start, job.EndTime)
	publicMonths := completedPrefetchMonths(start, job.EndTime, time.Now())
	skipped := map[int64]bool{}
	publicCalls, publicRows := 0, 0
	if len(publicMonths) > 0 && canUsePublicDataPrefetch(repo.Source) {
		client, err := newPrefetchPublicDataClient()
		if err == nil {
			stats, prefetchErr := prefetchPublicMonthly1m(ctx, repo, client, job.Symbol, publicMonths,
				func(index int, stage, detail string) {
					value := 1 + index*69/len(publicMonths)
					manager.updatePrefetch(jobID, func(job *PrefetchSummary) {
						job.Stage, job.StageDetail, job.Progress = stage, detail, value
					})
				})
			_ = client.Close()
			if prefetchErr != nil {
				fail(prefetchErr)
				return
			}
			skipped = stats.CompleteMonths
			publicCalls, publicRows = stats.Calls, stats.Rows
		}
	}
	counter := &prefetchCountingSource{inner: repo.Source}
	fetchRepo := *repo
	fetchRepo.Source = counter
	for i, month := range months {
		if skipped[month.Date.UnixMilli()] {
			manager.updatePrefetch(jobID, func(job *PrefetchSummary) {
				job.Stage, job.StageDetail, job.Progress = "checking", month.Date.Format("2006-01"), 70+(i+1)*29/len(months)
			})
			continue
		}
		label := month.Date.Format("2006-01")
		manager.updatePrefetch(jobID, func(job *PrefetchSummary) {
			job.Stage, job.StageDetail, job.Progress = "checking", label, 70+i*29/len(months)
		})
		complete, _, err := fetchRepo.KlineRangeComplete(ctx, historicalmarket.MarketFuturesUSDT, job.Symbol, ReplayInterval, month.Start, month.End)
		if err != nil {
			fail(fmt.Errorf("check %s: %w", label, err))
			return
		}
		if !complete {
			manager.updatePrefetch(jobID, func(job *PrefetchSummary) {
				job.Stage, job.StageDetail = "rest_fallback", label
			})
			if _, err := fetchRepo.LoadKlinesWithProgress(ctx, historicalmarket.MarketFuturesUSDT, job.Symbol, ReplayInterval, month.Start, month.End, nil); err != nil {
				fail(fmt.Errorf("repair %s: %w", label, err))
				return
			}
			complete, _, err = fetchRepo.KlineRangeComplete(ctx, historicalmarket.MarketFuturesUSDT, job.Symbol, ReplayInterval, month.Start, month.End)
			if err != nil {
				fail(fmt.Errorf("verify %s: %w", label, err))
				return
			}
			if !complete {
				fail(fmt.Errorf("1m candles remain missing in %s", label))
				return
			}
		}
	}
	calls, rows := counter.stats()
	manager.updatePrefetch(jobID, func(job *PrefetchSummary) {
		job.Status, job.Stage, job.Progress = "succeeded", "completed", 100
		job.RemoteCalls, job.RemoteRows = publicCalls+calls, publicRows+rows
		job.CompletedAt = time.Now().UnixMilli()
	})
}

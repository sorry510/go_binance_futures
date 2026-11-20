package backtest

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"go_binance_futures/service/historicalmarket"

	"github.com/beego/beego/v2/core/config"
)

type prefetchMonthRange struct {
	Date  time.Time
	Start int64
	End   int64
}

func prefetchMonthRanges(start, end int64) []prefetchMonthRange {
	if start > end {
		return nil
	}
	cursorTime := time.UnixMilli(start).UTC()
	cursor := time.Date(cursorTime.Year(), cursorTime.Month(), 1, 0, 0, 0, 0, time.UTC)
	out := make([]prefetchMonthRange, 0)
	for cursor.UnixMilli() <= end {
		next := cursor.AddDate(0, 1, 0)
		rangeStart := cursor.UnixMilli()
		if rangeStart < start {
			rangeStart = start
		}
		rangeEnd := next.Add(-time.Millisecond).UnixMilli()
		if rangeEnd > end {
			rangeEnd = end
		}
		if rangeStart <= rangeEnd {
			out = append(out, prefetchMonthRange{Date: cursor, Start: rangeStart, End: rangeEnd})
		}
		if rangeEnd >= end {
			break
		}
		cursor = next
	}
	return out
}

func completedPrefetchMonths(start, end int64, now time.Time) []prefetchMonthRange {
	now = now.UTC()
	currentMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	all := prefetchMonthRanges(start, end)
	out := make([]prefetchMonthRange, 0, len(all))
	for _, month := range all {
		if month.Date.UnixMilli() >= currentMonthStart {
			break
		}
		out = append(out, month)
	}
	return out
}

func newPrefetchPublicDataClient() (*historicalmarket.PublicDataClient, error) {
	proxyURL, _ := config.String("binance::proxy_url")
	tempRootDir, _ := config.String("binance::public_data_cache_dir")
	if strings.TrimSpace(tempRootDir) == "" {
		tempRootDir = "./cache/tmp"
	}
	return historicalmarket.NewPublicDataClient(historicalmarket.PublicDataClientConfig{
		ProxyURL: proxyURL, TempRootDir: tempRootDir,
	})
}

type publicPrefetchStats struct {
	Calls          int
	Rows           int
	CompleteMonths map[int64]bool
}

func canUsePublicDataPrefetch(source historicalmarket.Source) bool {
	switch source.(type) {
	case historicalmarket.BinanceSource, *historicalmarket.BinanceSource:
		return true
	default:
		return false
	}
}

func prefetchPublicMonthly1m(
	ctx context.Context,
	repo *historicalmarket.Repository,
	client *historicalmarket.PublicDataClient,
	symbol string,
	months []prefetchMonthRange,
	progress func(index int, stage, detail string),
) (publicPrefetchStats, error) {
	stats := publicPrefetchStats{CompleteMonths: map[int64]bool{}}
	if client == nil || len(months) == 0 {
		return stats, nil
	}

	// One metadata-only query covers existing compressed months; for the
	// remaining months perform the regular check (including legacy row data).
	// Each month is checked once, then cached for both phases of PrefetchDetailed.
	covered, err := repo.CompleteKline1mChunkMonths(
		ctx, historicalmarket.MarketFuturesUSDT, symbol,
		months[0].Start, months[len(months)-1].End,
	)
	if err != nil {
		return stats, err
	}
	missing := make([]int, 0, len(months))
	for i, month := range months {
		if err := ctx.Err(); err != nil {
			return stats, err
		}
		key := month.Date.UnixMilli()
		if covered[key] {
			stats.CompleteMonths[key] = true
			continue
		}
		complete, _, err := repo.KlineRangeComplete(
			ctx, historicalmarket.MarketFuturesUSDT, symbol, ReplayInterval,
			month.Start, month.End,
		)
		if err != nil {
			return stats, fmt.Errorf("check public-data month %s: %w", month.Date.Format("2006-01"), err)
		}
		if complete {
			stats.CompleteMonths[key] = true
		} else {
			missing = append(missing, i)
		}
	}

	// Keep at most three downloaded (or in-flight) archives ahead of the
	// sequential importer. This bounds temp disk and memory usage; no MySQL
	// writes or ZIP parsing take place in download goroutines.
	const maxConcurrentMonthlyDownloads = 3
	type fetchResult struct {
		archive historicalmarket.PublicDataArchive
		err     error
	}
	ready := make([]chan fetchResult, len(months))
	downloadCtx, cancel := context.WithCancel(ctx)
	var downloads sync.WaitGroup
	defer func() {
		cancel()
		downloads.Wait()
	}()
	startDownload := func(index int) {
		resultCh := make(chan fetchResult, 1)
		ready[index] = resultCh
		month := months[index]
		downloads.Add(1)
		go func() {
			defer downloads.Done()
			archive, err := client.FetchArchive(downloadCtx, historicalmarket.PublicDataArchiveSpec{
				Kind:   historicalmarket.ArchiveKindKlines,
				Period: historicalmarket.ArchivePeriodMonthly,
				Symbol: symbol, Interval: ReplayInterval, Date: month.Date,
			})
			resultCh <- fetchResult{archive: archive, err: err}
		}()
	}
	next := 0
	for next < len(missing) && next < maxConcurrentMonthlyDownloads {
		startDownload(missing[next])
		next++
	}

	for index, month := range months {
		if err := ctx.Err(); err != nil {
			return stats, err
		}
		label := month.Date.Format("2006-01")
		if stats.CompleteMonths[month.Date.UnixMilli()] {
			if progress != nil {
				progress(index+1, "checking", label)
			}
			continue
		}
		if progress != nil {
			progress(index, "public_data_downloading", label)
		}
		var downloaded fetchResult
		select {
		case <-ctx.Done():
			return stats, ctx.Err()
		case downloaded = <-ready[index]:
		}
		// Pipeline the next download while this month's ZIP is parsed and
		// imported. Imports remain serial and preserve chronological order.
		if next < len(missing) {
			startDownload(missing[next])
			next++
		}
		if downloaded.err != nil {
			if err := ctx.Err(); err != nil {
				return stats, err
			}
			if errors.Is(downloaded.err, historicalmarket.ErrPublicDataChecksumMismatch) {
				return stats, fmt.Errorf("public-data month %s checksum: %w", label, downloaded.err)
			}
			// Missing archives and network errors retain the existing
			// REST fallback; corrupt ZIP checksums fail closed.
			if progress != nil {
				progress(index+1, "rest_fallback", label)
			}
			continue
		}
		stats.Calls++
		if progress != nil {
			progress(index, "public_data_importing", label)
		}
		archiveStart := time.Date(month.Date.Year(), month.Date.Month(), 1, 0, 0, 0, 0, time.UTC)
		archiveEnd := archiveStart.AddDate(0, 1, 0).Add(-time.Millisecond)
		rows, err := client.ParseKlines(ctx, downloaded.archive, archiveStart.UnixMilli(), archiveEnd.UnixMilli())
		if err != nil {
			return stats, fmt.Errorf("parse public-data month %s: %w", label, err)
		}
		if len(rows) > 0 {
			result, err := repo.Import(ctx, historicalmarket.ImportRequest{
				Source:    historicalmarket.SourceBinancePublicData,
				SourceRef: downloaded.archive.URL + "#sha256=" + downloaded.archive.SHA256,
				Klines:    rows,
			})
			if err != nil {
				return stats, fmt.Errorf("import public-data month %s: %w", label, err)
			}
			stats.Rows += result.WrittenRows
		}
		complete, _, err := repo.KlineRangeComplete(
			ctx, historicalmarket.MarketFuturesUSDT, symbol, ReplayInterval,
			month.Start, month.End,
		)
		if err != nil {
			return stats, fmt.Errorf("verify public-data month %s: %w", label, err)
		}
		if complete {
			stats.CompleteMonths[month.Date.UnixMilli()] = true
		}
		if progress != nil {
			progress(index+1, "public_data_importing", label)
		}
	}
	return stats, nil
}

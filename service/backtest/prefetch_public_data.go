package backtest

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	if client == nil {
		return stats, nil
	}
	for index, month := range months {
		if err := ctx.Err(); err != nil {
			return stats, err
		}
		label := month.Date.Format("2006-01")
		if progress != nil {
			progress(index, "checking", label)
		}
		complete, _, err := repo.KlineRangeComplete(
			ctx, historicalmarket.MarketFuturesUSDT, symbol, ReplayInterval, month.Start, month.End,
		)
		if err != nil {
			return stats, fmt.Errorf("check public-data month %s: %w", label, err)
		}
		if complete {
			stats.CompleteMonths[month.Date.UnixMilli()] = true
			if progress != nil {
				progress(index+1, "checking", label)
			}
			continue
		}
		if progress != nil {
			progress(index, "public_data_downloading", label)
		}
		archive, err := client.FetchArchive(ctx, historicalmarket.PublicDataArchiveSpec{
			Kind: historicalmarket.ArchiveKindKlines, Period: historicalmarket.ArchivePeriodMonthly,
			Symbol: symbol, Interval: ReplayInterval, Date: month.Date,
		})
		if err != nil {
			if errors.Is(err, historicalmarket.ErrPublicDataArchiveNotFound) {
				if progress != nil {
					progress(index+1, "rest_fallback", label)
				}
				continue
			}
			if errors.Is(err, historicalmarket.ErrPublicDataChecksumMismatch) {
				return stats, fmt.Errorf("public-data month %s checksum: %w", label, err)
			}
			// Network/archive service errors fall back to REST; checksum failures do not.
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
		rows, err := client.ParseKlines(ctx, archive, archiveStart.UnixMilli(), archiveEnd.UnixMilli())
		if err != nil {
			return stats, fmt.Errorf("parse public-data month %s: %w", label, err)
		}
		if len(rows) > 0 {
			result, err := repo.Import(ctx, historicalmarket.ImportRequest{
				Source:    historicalmarket.SourceBinancePublicData,
				SourceRef: archive.URL + "#sha256=" + archive.SHA256,
				Klines:    rows,
			})
			if err != nil {
				return stats, fmt.Errorf("import public-data month %s: %w", label, err)
			}
			stats.Rows += result.WrittenRows
		}
		complete, _, err = repo.KlineRangeComplete(
			ctx, historicalmarket.MarketFuturesUSDT, symbol, ReplayInterval, month.Start, month.End,
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

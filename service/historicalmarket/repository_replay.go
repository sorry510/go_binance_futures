package historicalmarket

import (
	"context"
	"fmt"
	"strings"

	"github.com/beego/beego/v2/client/orm"
)

// LoadReplayKlines preserves the same gap detection/fill semantics as LoadKlines
// but returns only fields consumed by the backtest replay engine.
func (repo *Repository) LoadReplayKlines(ctx context.Context, market, symbol, interval string, start, end int64) ([]ReplayKline, error) {
	rows, _, err := repo.LoadReplayKlinesWithStats(ctx, market, symbol, interval, start, end)
	return rows, err
}

func (repo *Repository) LoadReplayKlinesWithStats(ctx context.Context, market, symbol, interval string, start, end int64) ([]ReplayKline, ReplayKlineCacheStats, error) {
	var stats ReplayKlineCacheStats
	if err := validateRange(market, symbol, start, end); err != nil {
		return nil, stats, err
	}
	if _, err := KlineTable(interval); err != nil {
		return nil, stats, err
	}
	rows, queryStats, err := repo.queryReplayKlines(market, symbol, interval, start, end)
	stats.add(queryStats)
	if err != nil {
		return nil, stats, err
	}
	missing, err := missingReplayKlineRanges(interval, start, end, rows)
	if err != nil {
		return nil, stats, err
	}
	if len(missing) > 0 && repo.Source != nil {
		for _, gap := range missing {
			remote, err := repo.Source.Klines(ctx, market, symbol, interval, gap[0], gap[1])
			if err != nil {
				return nil, stats, err
			}
			if len(remote) > 0 {
				if _, err := repo.Import(ctx, ImportRequest{Source: SourceBinanceREST, SourceRef: fmt.Sprintf("%s:%s:%d-%d", symbol, interval, gap[0], gap[1]), Klines: remote}); err != nil {
					return nil, stats, err
				}
			}
		}
		var refreshStats ReplayKlineCacheStats
		rows, refreshStats, err = repo.queryReplayKlines(market, symbol, interval, start, end)
		stats.add(refreshStats)
		if err != nil {
			return nil, stats, err
		}
		missing, err = missingReplayKlineRanges(interval, start, end, rows)
		if err != nil {
			return nil, stats, err
		}
	}
	if len(missing) > 0 {
		return nil, stats, fmt.Errorf("historical K-line gaps remain for %s %s: %v", symbol, interval, missing)
	}
	return rows, stats, nil
}

func (repo *Repository) queryReplayKlines(market, symbol, interval string, start, end int64) ([]ReplayKline, ReplayKlineCacheStats, error) {
	if interval == "1m" {
		return repo.queryReplay1mCached(market, symbol, start, end)
	}
	rows, err := repo.queryReplayKlinesDB(market, symbol, interval, start, end)
	return rows, ReplayKlineCacheStats{}, err
}

func (repo *Repository) queryReplayKlinesDB(market, symbol, interval string, start, end int64) ([]ReplayKline, error) {
	first, last, ok, err := expectedKlineBounds(interval, start, end)
	if err != nil {
		return nil, err
	}
	if !ok {
		return []ReplayKline{}, nil
	}
	return repo.queryReplayKlinesDBOpenRange(market, symbol, interval, first, last)
}

func (repo *Repository) queryReplayKlinesDBOpenRange(market, symbol, interval string, first, last int64) ([]ReplayKline, error) {
	table, err := KlineTable(interval)
	if err != nil {
		return nil, err
	}
	from := " FROM " + table
	if repo.mysql() {
		from += " FORCE INDEX (market)"
	}
	query := "SELECT open_time,close_time,open_price,high_price,low_price,close_price,volume,quote_volume,trade_count,taker_buy_quote_volume" + from + " WHERE market=? AND symbol=? AND open_time>=? AND open_time<=? ORDER BY open_time"
	var rows []ReplayKline
	if _, err := orm.NewOrm().Raw(query, market, strings.ToUpper(symbol), first, last).QueryRows(&rows); err != nil {
		return nil, err
	}
	return rows, nil
}

func missingReplayKlineRanges(interval string, start, end int64, rows []ReplayKline) ([][2]int64, error) {
	first, last, ok, err := expectedKlineBounds(interval, start, end)
	if err != nil || !ok {
		return nil, err
	}
	byOpen := make(map[int64]struct{}, len(rows))
	for _, row := range rows {
		byOpen[row.OpenTime] = struct{}{}
	}
	missing := make([][2]int64, 0)
	var gapStart int64
	for cursor := first; cursor <= last; {
		_, exists := byOpen[cursor]
		if !exists && gapStart == 0 {
			gapStart = cursor
		}
		next, err := nextOpen(interval, cursor)
		if err != nil {
			return nil, err
		}
		if exists && gapStart != 0 {
			prev, _ := previousOpen(interval, cursor)
			closeTime, _ := barClose(interval, prev)
			missing = append(missing, [2]int64{gapStart, closeTime})
			gapStart = 0
		}
		if next <= cursor {
			return nil, fmt.Errorf("invalid interval progression %s", interval)
		}
		cursor = next
	}
	if gapStart != 0 {
		closeTime, _ := barClose(interval, last)
		missing = append(missing, [2]int64{gapStart, closeTime})
	}
	return missing, nil
}

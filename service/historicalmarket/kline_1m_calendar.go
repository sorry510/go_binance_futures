package historicalmarket

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// MinuteDayCoverage represents the locally persisted 1m candles for a UTC day.
type MinuteDayCoverage struct {
	Date     string  `json:"date"`
	Expected int     `json:"expected"`
	Stored   int     `json:"stored"`
	Percent  float64 `json:"percent"`
	Status   string  `json:"status"`
}

type MinuteMonthCoverage struct {
	Month    string              `json:"month"`
	Expected int                 `json:"expected"`
	Stored   int                 `json:"stored"`
	Percent  float64             `json:"percent"`
	Status   string              `json:"status"`
	Days     []MinuteDayCoverage `json:"days"`
}

type MinuteCalendarCoverage struct {
	Symbol string                `json:"symbol"`
	From   string                `json:"from"`
	To     string                `json:"to"`
	Months []MinuteMonthCoverage `json:"months"`
}

func ParseMinuteMonthRange(from, to string, now time.Time) (time.Time, time.Time, error) {
	first, err := time.Parse("2006-01", from)
	if err != nil || first.Format("2006-01") != from {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid from month (YYYY-MM)")
	}
	last, err := time.Parse("2006-01", to)
	if err != nil || last.Format("2006-01") != to {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid to month (YYYY-MM)")
	}
	nowMonth := time.Date(now.UTC().Year(), now.UTC().Month(), 1, 0, 0, 0, 0, time.UTC)
	if last.Before(first) || last.After(nowMonth) || first.Before(time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)) {
		return time.Time{}, time.Time{}, fmt.Errorf("month range must be ordered, historical, and not in the future")
	}
	if count := (last.Year()-first.Year())*12 + int(last.Month()-first.Month()) + 1; count > 60 {
		return time.Time{}, time.Time{}, fmt.Errorf("month range cannot exceed 60 months")
	}
	return first, last, nil
}

func minuteCoverageStatus(stored, expected int) string {
	if expected <= 0 {
		return "future"
	}
	if stored <= 0 {
		return "missing"
	}
	if stored >= expected {
		return "complete"
	}
	return "partial"
}

func minuteCoveragePct(stored, expected int) float64 {
	if expected <= 0 {
		return 0
	}
	if stored > expected {
		stored = expected
	}
	return float64(stored) * 100 / float64(expected)
}

// Kline1mCalendar is read-only. It uses only chunk metadata and indexed
// individual 1m timestamps, avoiding expensive payload decompression and any
// automatic compaction or remote API call.
func (repo *Repository) Kline1mCalendar(ctx context.Context, symbol, from, to string, now time.Time) (MinuteCalendarCoverage, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if len(symbol) < 5 || len(symbol) > 32 || !strings.HasSuffix(symbol, "USDT") {
		return MinuteCalendarCoverage{}, fmt.Errorf("invalid USDT futures symbol")
	}
	for _, c := range symbol {
		if c < 'A' || c > 'Z' {
			if c < '0' || c > '9' {
				return MinuteCalendarCoverage{}, fmt.Errorf("invalid symbol")
			}
		}
	}
	first, last, err := ParseMinuteMonthRange(from, to, now)
	if err != nil {
		return MinuteCalendarCoverage{}, err
	}
	result := MinuteCalendarCoverage{Symbol: symbol, From: from, To: to, Months: make([]MinuteMonthCoverage, 0, 12)}
	endExclusive := last.AddDate(0, 1, 0)
	currentMinute := now.UTC().Truncate(time.Minute)
	if endExclusive.After(currentMinute) {
		endExclusive = currentMinute
	}
	for month := first; !month.After(last); month = month.AddDate(0, 1, 0) {
		item := MinuteMonthCoverage{Month: month.Format("2006-01"), Days: make([]MinuteDayCoverage, 0, 31)}
		for day := month; day.Before(month.AddDate(0, 1, 0)); day = day.AddDate(0, 0, 1) {
			dayEnd := day.AddDate(0, 0, 1)
			if dayEnd.After(endExclusive) {
				dayEnd = endExclusive
			}
			expected := 0
			if dayEnd.After(day) {
				expected = int(dayEnd.Sub(day) / time.Minute)
			}
			item.Days = append(item.Days, MinuteDayCoverage{Date: day.Format("2006-01-02"), Expected: expected})
			item.Expected += expected
		}
		result.Months = append(result.Months, item)
	}
	db, err := orm.GetDB("default")
	if err != nil {
		return MinuteCalendarCoverage{}, err
	}

	type chunkRange struct{ start, end int64 }
	byMonth := make(map[int64]chunkRange, len(result.Months))
	chunks, err := db.QueryContext(ctx, `SELECT month_start,start_time,end_time,point_count,encoding,compression
		FROM market_klines_1m_chunks WHERE market=? AND symbol=? AND month_start>=? AND month_start<?
		ORDER BY month_start`, MarketFuturesUSDT, symbol, first.UnixMilli(), last.AddDate(0, 1, 0).UnixMilli())
	if err != nil {
		return MinuteCalendarCoverage{}, err
	}
	for chunks.Next() {
		var key, start, end int64
		var points int
		var encoding, compression string
		if err := chunks.Scan(&key, &start, &end, &points, &encoding, &compression); err != nil {
			chunks.Close()
			return MinuteCalendarCoverage{}, err
		}
		if encoding == kline1mChunkEncoding && compression == kline1mChunkCompression &&
			start <= end && start%60000 == 0 && end%60000 == 0 &&
			int64(points) == (end-start)/60000+1 && points > 0 {
			byMonth[key] = chunkRange{start: start, end: end}
		}
	}
	if err := chunks.Err(); err != nil {
		chunks.Close()
		return MinuteCalendarCoverage{}, err
	}
	chunks.Close()
	// A valid chunk contains continuous minute opens. Count intersections
	// directly from its bounds. Never count a timestamp both in a chunk and
	// in a legacy row.
	for m := range result.Months {
		month := first.AddDate(0, m, 0)
		chunk, found := byMonth[month.UnixMilli()]
		if !found {
			continue
		}
		for d := range result.Months[m].Days {
			dayStart := month.AddDate(0, 0, d).UnixMilli()
			dayEnd := dayStart + 86400000
			if dayEnd > endExclusive.UnixMilli() {
				dayEnd = endExclusive.UnixMilli()
			}
			start := dayStart
			if start < chunk.start {
				start = chunk.start
			}
			end := dayEnd
			if end > chunk.end+60000 {
				end = chunk.end + 60000
			}
			if end > start {
				result.Months[m].Days[d].Stored += int((end - start) / 60000)
			}
		}
	}
	rows, err := db.QueryContext(ctx, `SELECT open_time FROM market_klines_1m
		WHERE market=? AND symbol=? AND open_time>=? AND open_time<? ORDER BY open_time`,
		MarketFuturesUSDT, symbol, first.UnixMilli(), endExclusive.UnixMilli())
	if err != nil {
		return MinuteCalendarCoverage{}, err
	}
	for rows.Next() {
		var at int64
		if err := rows.Scan(&at); err != nil {
			rows.Close()
			return MinuteCalendarCoverage{}, err
		}
		date := time.UnixMilli(at).UTC()
		mi := (date.Year()-first.Year())*12 + int(date.Month()-first.Month())
		if mi < 0 || mi >= len(result.Months) {
			continue
		}
		if chunk, found := byMonth[time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, time.UTC).UnixMilli()]; found && at >= chunk.start && at <= chunk.end {
			continue
		}
		di := date.Day() - 1
		if di < len(result.Months[mi].Days) {
			result.Months[mi].Days[di].Stored++
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return MinuteCalendarCoverage{}, err
	}
	rows.Close()
	for i := range result.Months {
		item := &result.Months[i]
		for d := range item.Days {
			day := &item.Days[d]
			if day.Stored > day.Expected {
				day.Stored = day.Expected
			}
			day.Percent = minuteCoveragePct(day.Stored, day.Expected)
			day.Status = minuteCoverageStatus(day.Stored, day.Expected)
			item.Stored += day.Stored
		}
		item.Percent = minuteCoveragePct(item.Stored, item.Expected)
		item.Status = minuteCoverageStatus(item.Stored, item.Expected)
	}
	return result, nil
}

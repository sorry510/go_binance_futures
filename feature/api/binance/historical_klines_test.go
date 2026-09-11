package binance

import (
	"testing"
	"time"
)

func TestHistoricalKlinePageEndOneMinuteNeverRequestsMoreThanLimit(t *testing.T) {
	start := time.Date(2025, 8, 31, 12, 40, 0, 0, time.UTC).UnixMilli()
	end := time.Date(2026, 9, 9, 15, 59, 59, 999000000, time.UTC).UnixMilli()
	pageEnd := historicalKlinePageEnd(start, end, "1m")
	want := start + int64(historicalKlinePageLimit)*time.Minute.Milliseconds() - 1
	if pageEnd != want {
		t.Fatalf("1m page end=%d want=%d", pageEnd, want)
	}
	if pageEnd-start+1 > int64(historicalKlinePageLimit)*time.Minute.Milliseconds() {
		t.Fatal("1m page exceeds Binance row window")
	}
}

func TestHistoricalKlinePageWindowsCoverLongRangeWithoutExchangeOversize(t *testing.T) {
	start := time.Date(2025, 8, 31, 12, 40, 0, 0, time.UTC).UnixMilli()
	end := time.Date(2026, 9, 9, 15, 59, 59, 999000000, time.UTC).UnixMilli()
	cursor := start
	pages := 0
	for cursor <= end {
		pageEnd := historicalKlinePageEnd(cursor, end, "1m")
		if pageEnd < cursor || pageEnd > end {
			t.Fatalf("invalid page %d-%d", cursor, pageEnd)
		}
		if pageEnd-cursor+1 > historicalKlineMaxWindow.Milliseconds() {
			t.Fatalf("page exceeds exchange time-window limit: %dms", pageEnd-cursor+1)
		}
		if pageEnd-cursor+1 > int64(historicalKlinePageLimit)*time.Minute.Milliseconds() {
			t.Fatalf("page exceeds 1m row limit: %dms", pageEnd-cursor+1)
		}
		pages++
		if pageEnd == end {
			break
		}
		cursor = pageEnd + 1
	}
	if pages < 500 {
		t.Fatalf("expected long range to be truly paged, pages=%d", pages)
	}
}

func TestHistoricalKlinePageEndCapsLongIntervalsByExchangeWindow(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	end := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	pageEnd := historicalKlinePageEnd(start, end, "1d")
	if pageEnd-start+1 != historicalKlineMaxWindow.Milliseconds() {
		t.Fatalf("1d page span=%s want=%s", time.Duration(pageEnd-start+1)*time.Millisecond, historicalKlineMaxWindow)
	}
}

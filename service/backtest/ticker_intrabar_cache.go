package backtest

import (
	"sort"
	"time"
)

// intrabarTickerSnapshot retains only the canonical 24-hour window statistics.
// The current forming 1m bar is applied fresh on EVERY intrabar evaluation.
// It must not reuse a completed-minute ticker preview, which can see future
// prices relative to the second or individual trade being evaluated.
type intrabarTickerSnapshot struct {
	minuteOpen int64
	start      int
	end        int
	ready      bool
	hasBase    bool
	open       float64
	high       float64
	low        float64
}

// prepare intentionally rechecks the visible canonical range. With irregular
// historical timestamps a previous bar can enter/leave a rolling window even
// inside one minute; when that happens we rebuild rather than alter semantics.
func (snapshot *intrabarTickerSnapshot) prepare(all []Bar, partial Bar, asOf int64) {
	end := sort.Search(len(all), func(i int) bool { return all[i].CloseTime > asOf })
	startTime := asOf - (24 * time.Hour).Milliseconds()
	start := sort.Search(end, func(i int) bool { return all[i].CloseTime >= startTime })
	// Canonical 1m may become visible at its close. The partial overlay
	// replaces it, instead of counting the same minute twice.
	canonicalEnd := end
	if end > start && all[end-1].OpenTime == partial.OpenTime {
		canonicalEnd--
	}
	if snapshot.ready && snapshot.minuteOpen == partial.OpenTime &&
		snapshot.start == start && snapshot.end == canonicalEnd {
		return
	}
	snapshot.minuteOpen = partial.OpenTime
	snapshot.start, snapshot.end = start, canonicalEnd
	snapshot.ready = true
	snapshot.hasBase = canonicalEnd > start
	if !snapshot.hasBase {
		snapshot.open, snapshot.high, snapshot.low = 0, 0, 0
		return
	}
	snapshot.open = all[start].Open
	snapshot.low, snapshot.high = all[start].Low, all[start].High
	for i := start + 1; i < canonicalEnd; i++ {
		if all[i].Low < snapshot.low {
			snapshot.low = all[i].Low
		}
		if all[i].High > snapshot.high {
			snapshot.high = all[i].High
		}
	}
}

// values matches historicalEnvironment.tickerStats for the same visible 1m
// snapshot, including the rolling 24h high/low and percent-change denominator.
func (snapshot *intrabarTickerSnapshot) values(partial Bar) map[string]interface{} {
	open, high, low := partial.Open, partial.High, partial.Low
	if snapshot.hasBase {
		open = snapshot.open
		if snapshot.high > high {
			high = snapshot.high
		}
		if snapshot.low < low {
			low = snapshot.low
		}
	}
	change := 0.0
	if open > 0 {
		change = (partial.Close - open) / open * 100
	}
	return map[string]interface{}{
		"PercentChange": change, "Close": partial.Close,
		"Open": open, "Low": low, "High": high,
	}
}

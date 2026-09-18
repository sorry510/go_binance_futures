package backtest

import (
	"sort"
	"time"
)

type rollingTickerStats struct {
	asOf     int64
	window   []Bar
	head     int
	highs    []Bar
	highHead int
	lows     []Bar
	lowHead  int
}

func (state *rollingTickerStats) reset(all []Bar, asOf int64) {
	state.asOf = asOf
	state.window = state.window[:0]
	state.head = 0
	state.highs = state.highs[:0]
	state.highHead = 0
	state.lows = state.lows[:0]
	state.lowHead = 0
	end := sort.Search(len(all), func(i int) bool { return all[i].CloseTime > asOf })
	startTime := asOf - (24 * time.Hour).Milliseconds()
	start := sort.Search(end, func(i int) bool { return all[i].CloseTime >= startTime })
	for i := start; i < end; i++ {
		state.append(all[i])
	}
}

func (state *rollingTickerStats) advance(bar Bar, asOf int64) bool {
	if state.asOf <= 0 || bar.CloseTime != asOf || asOf-state.asOf != time.Minute.Milliseconds() {
		return false
	}
	if len(state.window) > state.head && bar.OpenTime <= state.window[len(state.window)-1].OpenTime {
		return false
	}
	startTime := asOf - (24 * time.Hour).Milliseconds()
	state.expire(startTime)
	state.append(bar)
	state.asOf = asOf
	state.compact()
	return true
}

func (state *rollingTickerStats) append(bar Bar) {
	state.window = append(state.window, bar)
	for len(state.highs) > state.highHead && state.highs[len(state.highs)-1].High <= bar.High {
		state.highs = state.highs[:len(state.highs)-1]
	}
	state.highs = append(state.highs, bar)
	for len(state.lows) > state.lowHead && state.lows[len(state.lows)-1].Low >= bar.Low {
		state.lows = state.lows[:len(state.lows)-1]
	}
	state.lows = append(state.lows, bar)
}

func (state *rollingTickerStats) expire(startTime int64) {
	for state.head < len(state.window) && state.window[state.head].CloseTime < startTime {
		state.head++
	}
	for state.highHead < len(state.highs) && state.highs[state.highHead].CloseTime < startTime {
		state.highHead++
	}
	for state.lowHead < len(state.lows) && state.lows[state.lowHead].CloseTime < startTime {
		state.lowHead++
	}
}

func (state *rollingTickerStats) compact() {
	if state.head > 2048 && state.head*2 > len(state.window) {
		state.window = append([]Bar(nil), state.window[state.head:]...)
		state.head = 0
	}
	if state.highHead > 2048 && state.highHead*2 > len(state.highs) {
		state.highs = append([]Bar(nil), state.highs[state.highHead:]...)
		state.highHead = 0
	}
	if state.lowHead > 2048 && state.lowHead*2 > len(state.lows) {
		state.lows = append([]Bar(nil), state.lows[state.lowHead:]...)
		state.lowHead = 0
	}
}

func (state *rollingTickerStats) values() map[string]interface{} {
	if state.head >= len(state.window) {
		return map[string]interface{}{"PercentChange": 0.0, "Close": 0.0, "Open": 0.0, "Low": 0.0, "High": 0.0}
	}
	first := state.window[state.head]
	last := state.window[len(state.window)-1]
	low, high := 0.0, 0.0
	if state.lowHead < len(state.lows) {
		low = state.lows[state.lowHead].Low
	}
	if state.highHead < len(state.highs) {
		high = state.highs[state.highHead].High
	}
	change := 0.0
	if first.Open > 0 {
		change = (last.Close - first.Open) / first.Open * 100
	}
	return map[string]interface{}{"PercentChange": change, "Close": last.Close, "Open": first.Open, "Low": low, "High": high}
}

func (builder *historicalEnvironment) prepareRollingTickerStats(minute Bar, asOf int64) {
	if !builder.standardOptimizations || builder.dataset.ExecutionInterval != "1m" || minute.Interval != "1m" || minute.CloseTime != asOf {
		builder.tickerStatsCache = nil
		return
	}
	if builder.tickerStatsCache != nil && builder.tickerStatsCache.advance(minute, asOf) {
		return
	}
	state := &rollingTickerStats{}
	state.reset(builder.dataset.Bars[BarSeriesKey(builder.dataset.Symbol, builder.dataset.ExecutionInterval)], asOf)
	builder.tickerStatsCache = state
}

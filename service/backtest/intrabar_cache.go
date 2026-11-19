package backtest

import (
	"fmt"
	"sort"
)

// intrabarPrefix is the already observed portion of a higher-interval bar
// strictly before the currently replayed minute. It is immutable within that
// minute; only the partial minute changes on every 1s/trade update.
type intrabarPrefix struct {
	windowStart int64
	bar         Bar
	hasBars     bool
}

// cachedIntrabarOverlays avoids scanning all completed 1m bars on every
// intraminute evaluation. Its cache is owned by one sequential Engine run.
func (builder *historicalEnvironment) cachedIntrabarOverlays(partial Bar, asOf int64) (map[string]Bar, error) {
	if partial.OpenTime <= 0 || partial.CloseTime != asOf || partial.Open <= 0 || partial.Close <= 0 || partial.High < partial.Low {
		return nil, fmt.Errorf("invalid intrabar partial minute at %d", asOf)
	}
	if !builder.intrabarReady || builder.intrabarMinuteOpen != partial.OpenTime {
		prefixes := make(map[string]intrabarPrefix, len(builder.dataset.Intervals))
		minutes := builder.dataset.Bars[BarSeriesKey(builder.dataset.Symbol, "1m")]
		end := sort.Search(len(minutes), func(i int) bool { return minutes[i].OpenTime >= partial.OpenTime })
		for _, interval := range builder.dataset.Intervals {
			if interval == "1m" {
				continue
			}
			startAt, err := builder.cachedIntervalWindowStart(interval, asOf)
			if err != nil {
				return nil, err
			}
			start := sort.Search(end, func(i int) bool { return minutes[i].OpenTime >= startAt })
			stop := end
			for i := start; i < end; i++ {
				if minutes[i].CloseTime > asOf {
					stop = i
					break
				}
			}
			prefix := intrabarPrefix{windowStart: startAt}
			if stop > start {
				prefix.bar = aggregatePartialBars(builder.dataset.Symbol, interval, startAt, asOf, minutes[start:stop])
				prefix.hasBars = true
			}
			prefixes[interval] = prefix
		}
		builder.intrabarPrefixes = prefixes
		builder.intrabarMinuteOpen = partial.OpenTime
		builder.intrabarReady = true
		if builder.intrabarOverlays == nil {
			builder.intrabarOverlays = make(map[string]Bar, len(builder.dataset.Intervals))
		}
	}
	overlays := builder.intrabarOverlays
	overlays[BarSeriesKey(builder.dataset.Symbol, "1m")] = partial
	for _, interval := range builder.dataset.Intervals {
		if interval == "1m" {
			continue
		}
		prefix := builder.intrabarPrefixes[interval]
		var merged Bar
		if prefix.hasBars {
			merged = aggregatePartialBars(builder.dataset.Symbol, interval, prefix.windowStart, asOf, []Bar{prefix.bar, partial})
		} else {
			merged = aggregatePartialBars(builder.dataset.Symbol, interval, prefix.windowStart, asOf, []Bar{partial})
		}
		overlays[BarSeriesKey(builder.dataset.Symbol, interval)] = merged
	}
	return overlays, nil
}

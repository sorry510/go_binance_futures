package backtest

import "sort"

// sampleImportantEquityPoints preserves the information needed for an overview
// chart without storing the complete one-minute equity archive. It runs after
// the replay and metrics calculation; it never changes execution decisions.
func sampleImportantEquityPoints(points []EquityPoint, trades []Trade, limit int) []EquityPoint {
	n := len(points)
	if limit <= 0 || n <= limit {
		return append([]EquityPoint(nil), points...)
	}
	marked := make([]bool, n)
	count := 0
	add := func(i int) {
		if i >= 0 && i < n && !marked[i] && count < limit {
			marked[i] = true
			count++
		}
	}

	// Essential points take precedence over every other sampling category.
	add(0)
	add(n - 1)
	high, low, drawdown, drawdownPeak, rollingPeak := 0, 0, 0, 0, 0
	for i := 1; i < n; i++ {
		if points[i].Equity > points[high].Equity {
			high = i
		}
		if points[i].Equity < points[low].Equity {
			low = i
		}
		if points[i].Equity > points[rollingPeak].Equity {
			rollingPeak = i
		}
		if points[i].DrawdownPct > points[drawdown].DrawdownPct {
			drawdown, drawdownPeak = i, rollingPeak
		}
	}
	add(high)
	add(low)
	add(drawdownPeak)
	add(drawdown)

	// Preserve the minute before and the first minute at/after each actual fill.
	// Adaptive intrabar fills can lie between the minute-close snapshots.
	// Reserve at most 1/4 of the points for this category to keep chart
	// coverage stable when a strategy generates many trades.
	tradeIndices := make([]int, 0, len(trades)*4)
	for _, trade := range trades {
		for _, at := range [...]int64{trade.EntryTime, trade.ExitTime} {
			if at <= 0 || at < points[0].BarTime || at > points[n-1].BarTime {
				continue
			}
			i := sort.Search(n, func(j int) bool { return points[j].BarTime >= at })
			if i < n {
				tradeIndices = append(tradeIndices, i)
			}
			if i > 0 {
				tradeIndices = append(tradeIndices, i-1)
			}
		}
	}
	sort.Ints(tradeIndices)
	candidates := tradeIndices[:0]
	for _, i := range tradeIndices {
		if len(candidates) == 0 || candidates[len(candidates)-1] != i {
			candidates = append(candidates, i)
		}
	}
	tradeBudget := limit / 4
	if tradeBudget > limit-count {
		tradeBudget = limit - count
	}
	if tradeBudget > 0 && len(candidates) > 0 {
		if len(candidates) <= tradeBudget {
			for _, i := range candidates {
				add(i)
			}
		} else if tradeBudget == 1 {
			add(candidates[0])
		} else {
			for j := 0; j < tradeBudget; j++ {
				add(candidates[j*(len(candidates)-1)/(tradeBudget-1)])
			}
		}
	}

	// Keep local maxima and minima across the entire run, independent of
	// whether the drawdown happens to align with uniform time sampling.
	// At most ~2/3 of the final budget is used by these regional extrema.
	buckets := limit / 3
	if buckets < 1 {
		buckets = 1
	}
	for b := 0; b < buckets && count < limit; b++ {
		from, to := b*n/buckets, (b+1)*n/buckets
		if from >= to {
			continue
		}
		minIndex, maxIndex := from, from
		for i := from + 1; i < to; i++ {
			if points[i].Equity < points[minIndex].Equity {
				minIndex = i
			}
			if points[i].Equity > points[maxIndex].Equity {
				maxIndex = i
			}
		}
		add(minIndex)
		add(maxIndex)
	}

	// Uniformly fill gaps, guaranteeing broad coverage of quiet periods.
	// The dense fallback avoids bias towards the beginning of long runs.
	slots := limit * 4
	for i := 0; i < slots && count < limit; i++ {
		add(i * (n - 1) / (slots - 1))
	}
	for i := 0; i < n && count < limit; i++ {
		add(i)
	}
	out := make([]EquityPoint, 0, count)
	for i, keep := range marked {
		if keep {
			out = append(out, points[i])
		}
	}
	return out
}

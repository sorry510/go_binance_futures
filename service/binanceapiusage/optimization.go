package binanceapiusage

import (
	"sort"
	"strings"
	"sync"
)

type OptimizationStat struct {
	Source                  string `json:"source"`
	CacheHits               uint64 `json:"cache_hits"`
	CoalescedRequests       uint64 `json:"coalesced_requests"`
	LocalWSHits             uint64 `json:"local_ws_hits"`
	PreventedDuplicateCalls uint64 `json:"prevented_duplicate_calls"`
	DeferredRequests        uint64 `json:"deferred_requests"`
}

var optimizationCounters = struct {
	sync.Mutex
	bySource map[string]OptimizationStat
}{bySource: map[string]OptimizationStat{}}

func RecordOptimization(source, kind string, count uint64) {
	if count == 0 {
		return
	}
	source = normalizeLabel(source, "unknown")
	optimizationCounters.Lock()
	defer optimizationCounters.Unlock()
	row := optimizationCounters.bySource[source]
	row.Source = source
	switch strings.TrimSpace(kind) {
	case "cache_hit":
		row.CacheHits += count
	case "coalesced":
		row.CoalescedRequests += count
	case "local_ws_hit":
		row.LocalWSHits += count
	case "prevented_duplicate":
		row.PreventedDuplicateCalls += count
	case "deferred":
		row.DeferredRequests += count
	default:
		return
	}
	optimizationCounters.bySource[source] = row
}

func optimizationSnapshot() []OptimizationStat {
	optimizationCounters.Lock()
	defer optimizationCounters.Unlock()
	rows := make([]OptimizationStat, 0, len(optimizationCounters.bySource))
	for _, row := range optimizationCounters.bySource {
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Source < rows[j].Source })
	return rows
}

func resetOptimizationForTest() {
	optimizationCounters.Lock()
	defer optimizationCounters.Unlock()
	optimizationCounters.bySource = map[string]OptimizationStat{}
}

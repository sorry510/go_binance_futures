package historicalmarket

import (
	"container/list"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Keep enough decoded months for a typical five-year single-symbol backtest
// while remaining strictly bounded. This avoids sequential-scan thrashing when
// the same symbol/range is backtested repeatedly.
const replay1mMemoryCacheCapacity = 64

type ReplayKlineCacheStats struct {
	FilesHit  int
	FilesMiss int
	ReadMs    int64
	WriteMs   int64
}

func (stats *ReplayKlineCacheStats) add(other ReplayKlineCacheStats) {
	stats.FilesHit += other.FilesHit
	stats.FilesMiss += other.FilesMiss
	stats.ReadMs += other.ReadMs
	stats.WriteMs += other.WriteMs
}

type replay1mMemoryEntry struct {
	key  string
	rows []ReplayKline
}
type replay1mMemoryLRU struct {
	mu       sync.Mutex
	capacity int
	items    map[string]*list.Element
	order    *list.List
}

func newReplay1mMemoryLRU(capacity int) *replay1mMemoryLRU {
	if capacity < 1 {
		capacity = 1
	}
	return &replay1mMemoryLRU{
		capacity: capacity,
		items:    make(map[string]*list.Element),
		order:    list.New(),
	}
}

var globalReplay1mMemoryCache = newReplay1mMemoryLRU(replay1mMemoryCacheCapacity)

func replay1mMemoryKey(market, symbol string, monthStart int64) string {
	return market + "|" + strings.ToUpper(symbol) + "|" + fmt.Sprintf("%d", monthStart)
}
func (cache *replay1mMemoryLRU) get(key string) ([]ReplayKline, bool) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	element, ok := cache.items[key]
	if !ok {
		return nil, false
	}
	cache.order.MoveToFront(element)
	return element.Value.(*replay1mMemoryEntry).rows, true
}

func (cache *replay1mMemoryLRU) put(key string, rows []ReplayKline) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if element, ok := cache.items[key]; ok {
		element.Value.(*replay1mMemoryEntry).rows = rows
		cache.order.MoveToFront(element)
		return
	}
	element := cache.order.PushFront(&replay1mMemoryEntry{key: key, rows: rows})
	cache.items[key] = element
	for cache.order.Len() > cache.capacity {
		last := cache.order.Back()
		if last == nil {
			break
		}
		cache.order.Remove(last)
		delete(cache.items, last.Value.(*replay1mMemoryEntry).key)
	}
}

func (cache *replay1mMemoryLRU) remove(key string) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if element, ok := cache.items[key]; ok {
		cache.order.Remove(element)
		delete(cache.items, key)
	}
}

func invalidateReplay1mMemoryChunk(market, symbol string, monthStart int64) {
	globalReplay1mMemoryCache.remove(replay1mMemoryKey(market, symbol, monthStart))
}

func filterReplayKlines(rows []ReplayKline, start, end int64) []ReplayKline {
	first := 0
	for first < len(rows) && rows[first].OpenTime < start {
		first++
	}
	last := first
	for last < len(rows) && rows[last].OpenTime <= end {
		last++
	}
	if first >= last {
		return nil
	}
	return rows[first:last]
}

func klineRowsToReplay(rows []Kline) []ReplayKline {
	out := make([]ReplayKline, len(rows))
	for i, row := range rows {
		out[i] = ReplayKline{
			OpenTime: row.OpenTime, CloseTime: row.CloseTime, TradeCount: row.TradeCount,
			Open: row.Open, High: row.High, Low: row.Low, Close: row.Close,
			Volume: row.Volume, QuoteVolume: row.QuoteVolume,
			TakerBuyQuoteVolume: row.TakerBuyQuoteVolume,
		}
	}
	return out
}

func (repo *Repository) replay1mSegmentWithRows(market, symbol string, cached []ReplayKline, start, end int64) ([]ReplayKline, error) {
	if len(cached) == 0 {
		return repo.queryReplayKlinesDBOpenRange(market, symbol, "1m", start, end)
	}
	minuteMs := int64(time.Minute / time.Millisecond)
	out := make([]ReplayKline, 0)
	if start < cached[0].OpenTime {
		rowEnd := cached[0].OpenTime - minuteMs
		if rowEnd > end {
			rowEnd = end
		}
		if rowEnd >= start {
			rows, err := repo.queryReplayKlinesDBOpenRange(market, symbol, "1m", start, rowEnd)
			if err != nil {
				return nil, err
			}
			out = append(out, rows...)
		}
	}
	out = append(out, filterReplayKlines(cached, start, end)...)
	if end > cached[len(cached)-1].OpenTime {
		rowStart := cached[len(cached)-1].OpenTime + minuteMs
		if rowStart < start {
			rowStart = start
		}
		if rowStart <= end {
			rows, err := repo.queryReplayKlinesDBOpenRange(market, symbol, "1m", rowStart, end)
			if err != nil {
				return nil, err
			}
			out = append(out, rows...)
		}
	}
	return out, nil
}

func (repo *Repository) queryReplay1mMemoryCached(market, symbol string, start, end int64) ([]ReplayKline, ReplayKlineCacheStats, error) {
	var stats ReplayKlineCacheStats
	first, last, ok, err := expectedKlineBounds("1m", start, end)
	if err != nil || !ok {
		return nil, stats, err
	}
	type monthSegment struct {
		monthStart   int64
		segmentStart int64
		segmentEnd   int64
		closed       bool
	}
	segments := make([]monthSegment, 0)
	decoded := make(map[int64][]ReplayKline)
	missing := make(map[int64]struct{})
	var minMissing, maxMissing int64

	for month := kline1mMonthStart(first); !month.After(kline1mMonthStart(last)); month = month.AddDate(0, 1, 0) {
		monthStart, monthEnd := kline1mMonthBounds(month)
		segmentStart, segmentEnd := first, last
		if segmentStart < monthStart {
			segmentStart = monthStart
		}
		if segmentEnd > monthEnd {
			segmentEnd = monthEnd
		}
		closed := repo.kline1mMonthClosed(month)
		segments = append(segments, monthSegment{
			monthStart: monthStart, segmentStart: segmentStart, segmentEnd: segmentEnd, closed: closed,
		})
		if !closed {
			continue
		}
		key := replay1mMemoryKey(market, symbol, monthStart)
		if cached, hit := globalReplay1mMemoryCache.get(key); hit {
			stats.FilesHit++
			decoded[monthStart] = cached
			continue
		}
		stats.FilesMiss++
		missing[monthStart] = struct{}{}
		if minMissing == 0 || monthStart < minMissing {
			minMissing = monthStart
		}
		if monthStart > maxMissing {
			maxMissing = monthStart
		}
	}

	if len(missing) > 0 {
		readStarted := time.Now()
		chunks, err := loadKline1mChunksRange(market, symbol, minMissing, maxMissing)
		if err != nil {
			return nil, stats, err
		}
		for monthStart := range missing {
			chunk, found := chunks[monthStart]
			if !found {
				continue
			}
			rows, err := decodeKline1mChunk(chunk)
			if err != nil {
				return nil, stats, err
			}
			replay := klineRowsToReplay(rows)
			decoded[monthStart] = replay
			globalReplay1mMemoryCache.put(replay1mMemoryKey(market, symbol, monthStart), replay)
		}
		stats.ReadMs += time.Since(readStarted).Milliseconds()
	}

	out := make([]ReplayKline, 0)
	for _, segment := range segments {
		if !segment.closed {
			rows, err := repo.queryReplayKlinesDBOpenRange(market, symbol, "1m", segment.segmentStart, segment.segmentEnd)
			if err != nil {
				return nil, stats, err
			}
			out = append(out, rows...)
			continue
		}
		if cached, found := decoded[segment.monthStart]; found {
			rows, err := repo.replay1mSegmentWithRows(market, symbol, cached, segment.segmentStart, segment.segmentEnd)
			if err != nil {
				return nil, stats, err
			}
			out = append(out, rows...)
			continue
		}
		rows, err := repo.queryReplayKlinesDBOpenRange(market, symbol, "1m", segment.segmentStart, segment.segmentEnd)
		if err != nil {
			return nil, stats, err
		}
		out = append(out, rows...)
	}
	return out, stats, nil
}

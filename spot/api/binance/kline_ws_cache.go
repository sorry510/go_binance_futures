package binance

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"go_binance_futures/service/binanceapiusage"

	spotbinance "github.com/adshao/go-binance/v2"
	"github.com/beego/beego/v2/adapter/logs"
	"golang.org/x/sync/singleflight"
)

const (
	spotLiveKlineWSFreshTTL    = 3 * time.Second
	spotLiveKlineWSRetryGap    = 3 * time.Second
	spotLiveKlineWSDebounce    = 250 * time.Millisecond
	spotLiveKlineWSMaxStreams  = 256
	spotLiveKlineWSInactiveTTL = 15 * time.Minute
	spotLiveKlineCacheMax      = 256
	spotLiveKlineNegativeTTL   = 2 * time.Second
)

type spotLiveKlineCacheEntry struct {
	rows        []*spotbinance.Kline
	err         error
	expiresAt   time.Time
	wsUpdatedAt int64
	maxLimit    int
}

var (
	spotLiveKlineCacheMu sync.Mutex
	spotLiveKlineCache   = make(map[string]spotLiveKlineCacheEntry)
	spotLiveKlineGroup   singleflight.Group
)

var spotLiveKlineWS = struct {
	sync.Mutex
	subscriptions map[string]map[string]struct{}
	lastUsed      map[string]int64
	active        map[string]bool
	updateCh      chan struct{}
	started       bool
}{
	subscriptions: make(map[string]map[string]struct{}),
	lastUsed:      make(map[string]int64),
	active:        make(map[string]bool),
	updateCh:      make(chan struct{}, 1),
}

func spotLiveKlineCacheTTL(interval string) time.Duration {
	switch strings.TrimSpace(interval) {
	case "1m":
		return time.Second
	case "3m", "5m":
		return 2 * time.Second
	case "15m", "30m":
		return 5 * time.Second
	default:
		return 10 * time.Second
	}
}

func spotLiveKlineCacheKey(symbol, interval string) string {
	return strings.ToUpper(strings.TrimSpace(symbol)) + "|" + strings.TrimSpace(interval)
}

func cloneSpotKlines(rows []*spotbinance.Kline) []*spotbinance.Kline {
	if rows == nil {
		return nil
	}
	return append([]*spotbinance.Kline(nil), rows...)
}

func loadSpotLiveKlineCache(key string, limit int) ([]*spotbinance.Kline, error, bool, bool) {
	now := time.Now()
	spotLiveKlineCacheMu.Lock()
	defer spotLiveKlineCacheMu.Unlock()

	entry, ok := spotLiveKlineCache[key]
	if !ok {
		return nil, nil, false, false
	}
	if !entry.expiresAt.After(now) {
		delete(spotLiveKlineCache, key)
		return nil, nil, false, false
	}
	if entry.err == nil && entry.maxLimit < limit {
		return nil, nil, false, false
	}
	rows := cloneSpotKlines(entry.rows)
	if limit > 0 && len(rows) > limit {
		rows = rows[:limit]
	}
	wsFresh := entry.wsUpdatedAt > 0 && now.Sub(time.UnixMilli(entry.wsUpdatedAt)) <= spotLiveKlineWSFreshTTL
	return rows, entry.err, true, wsFresh
}

func storeSpotLiveKlineCache(key string, rows []*spotbinance.Kline, err error, ttl time.Duration, limit int) {
	now := time.Now()
	spotLiveKlineCacheMu.Lock()
	defer spotLiveKlineCacheMu.Unlock()

	for cacheKey, entry := range spotLiveKlineCache {
		if !entry.expiresAt.After(now) {
			delete(spotLiveKlineCache, cacheKey)
		}
	}
	if len(spotLiveKlineCache) >= spotLiveKlineCacheMax {
		var oldestKey string
		var oldest time.Time
		for cacheKey, entry := range spotLiveKlineCache {
			if oldestKey == "" || entry.expiresAt.Before(oldest) {
				oldestKey, oldest = cacheKey, entry.expiresAt
			}
		}
		if oldestKey != "" {
			delete(spotLiveKlineCache, oldestKey)
		}
	}

	if existing, ok := spotLiveKlineCache[key]; ok && err == nil && existing.err == nil && existing.maxLimit > limit {
		merged := cloneSpotKlines(existing.rows)
		for i := 0; i < len(rows) && i < len(merged); i++ {
			merged[i] = rows[i]
		}
		spotLiveKlineCache[key] = spotLiveKlineCacheEntry{
			rows: merged, expiresAt: now.Add(ttl), maxLimit: existing.maxLimit,
		}
		return
	}
	spotLiveKlineCache[key] = spotLiveKlineCacheEntry{
		rows: cloneSpotKlines(rows), err: err, expiresAt: now.Add(ttl), maxLimit: limit,
	}
}

func getSpotLiveKlineData(ctx context.Context, symbol, interval string, limit int) ([]*spotbinance.Kline, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	interval = strings.TrimSpace(interval)
	if symbol == "" || interval == "" || limit <= 0 {
		return nil, fmt.Errorf("spot K-line requires symbol, interval and positive limit")
	}
	ensureSpotLiveKlineWS(symbol, interval)
	key := spotLiveKlineCacheKey(symbol, interval)
	source := binanceapiusage.SourceFromContext(ctx)
	if rows, err, ok, wsFresh := loadSpotLiveKlineCache(key, limit); ok {
		recordSpotLiveKlineHit(source, wsFresh)
		return rows, err
	}

	value, err, shared := spotLiveKlineGroup.Do(key, func() (interface{}, error) {
		if rows, cachedErr, ok, wsFresh := loadSpotLiveKlineCache(key, limit); ok {
			recordSpotLiveKlineHit(source, wsFresh)
			return rows, cachedErr
		}
		rows, requestErr := client.NewKlinesService().Symbol(symbol).Interval(interval).Limit(limit).Do(ctx)
		if requestErr != nil {
			storeSpotLiveKlineCache(key, nil, requestErr, spotLiveKlineNegativeTTL, limit)
			return nil, requestErr
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i].OpenTime > rows[j].OpenTime })
		storeSpotLiveKlineCache(key, rows, nil, spotLiveKlineCacheTTL(interval), limit)
		return cloneSpotKlines(rows), nil
	})
	if shared {
		binanceapiusage.RecordOptimization(source, "coalesced", 1)
		binanceapiusage.RecordOptimization(source, "prevented_duplicate", 1)
	}
	if err != nil {
		return nil, err
	}
	rows, ok := value.([]*spotbinance.Kline)
	if !ok {
		return nil, fmt.Errorf("unexpected spot K-line cache value type %T", value)
	}
	return cloneSpotKlines(rows), nil
}

func recordSpotLiveKlineHit(source string, wsFresh bool) {
	if wsFresh {
		binanceapiusage.RecordOptimization(source, "local_ws_hit", 1)
	} else {
		binanceapiusage.RecordOptimization(source, "cache_hit", 1)
	}
	binanceapiusage.RecordOptimization(source, "prevented_duplicate", 1)
}

func ensureSpotLiveKlineWS(symbol, interval string) {
	spotLiveKlineWS.Lock()
	intervals := spotLiveKlineWS.subscriptions[symbol]
	if intervals == nil {
		intervals = make(map[string]struct{})
		spotLiveKlineWS.subscriptions[symbol] = intervals
	}
	key := symbol + "|" + interval
	_, exists := intervals[interval]
	if !exists {
		intervals[interval] = struct{}{}
	}
	wasActive := spotLiveKlineWS.active[key]
	spotLiveKlineWS.lastUsed[key] = time.Now().UnixMilli()
	if !spotLiveKlineWS.started {
		spotLiveKlineWS.started = true
		go runSpotLiveKlineWS()
	}
	spotLiveKlineWS.Unlock()

	if !exists || !wasActive {
		select {
		case spotLiveKlineWS.updateCh <- struct{}{}:
		default:
		}
	}
}

func spotLiveKlineSubscriptions() map[string][]string {
	type pair struct {
		symbol, interval string
		lastUsed         int64
	}
	spotLiveKlineWS.Lock()
	defer spotLiveKlineWS.Unlock()

	cutoff := time.Now().Add(-spotLiveKlineWSInactiveTTL).UnixMilli()
	pairs := make([]pair, 0, len(spotLiveKlineWS.lastUsed))
	for symbol, intervalSet := range spotLiveKlineWS.subscriptions {
		for interval := range intervalSet {
			key := symbol + "|" + interval
			lastUsed := spotLiveKlineWS.lastUsed[key]
			if lastUsed < cutoff {
				delete(intervalSet, interval)
				delete(spotLiveKlineWS.lastUsed, key)
				continue
			}
			pairs = append(pairs, pair{symbol: symbol, interval: interval, lastUsed: lastUsed})
		}
		if len(intervalSet) == 0 {
			delete(spotLiveKlineWS.subscriptions, symbol)
		}
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].lastUsed != pairs[j].lastUsed {
			return pairs[i].lastUsed > pairs[j].lastUsed
		}
		if pairs[i].symbol != pairs[j].symbol {
			return pairs[i].symbol < pairs[j].symbol
		}
		return pairs[i].interval < pairs[j].interval
	})
	if len(pairs) > spotLiveKlineWSMaxStreams {
		pairs = pairs[:spotLiveKlineWSMaxStreams]
	}
	out := make(map[string][]string)
	for _, item := range pairs {
		out[item.symbol] = append(out[item.symbol], item.interval)
	}
	for symbol := range out {
		sort.Strings(out[symbol])
	}
	return out
}

func setActiveSpotLiveKlineSubscriptions(subscriptions map[string][]string) {
	active := make(map[string]bool)
	for symbol, intervals := range subscriptions {
		for _, interval := range intervals {
			active[symbol+"|"+interval] = true
		}
	}
	spotLiveKlineWS.Lock()
	spotLiveKlineWS.active = active
	spotLiveKlineWS.Unlock()
}

func runSpotLiveKlineWS() {
	for {
		<-spotLiveKlineWS.updateCh
		debounceSpotLiveKlineWSUpdates()
		for {
			subscriptions := spotLiveKlineSubscriptions()
			if len(subscriptions) == 0 {
				break
			}
			setActiveSpotLiveKlineSubscriptions(subscriptions)
			doneC, stopC, err := wsSpotCombinedKlineServeMultiInterval(
				subscriptions,
				func(event *spotbinance.WsKlineEvent) { applySpotLiveKlineWSEvent(event, time.Now()) },
				func(runErr error) {
					if runErr != nil {
						logs.Error("spot combined kline ws run error:", runErr)
					}
				},
			)
			if err != nil {
				logs.Error("spot combined kline ws start error:", err)
				time.Sleep(spotLiveKlineWSRetryGap)
				continue
			}
			if doneC == nil || stopC == nil {
				logs.Error("spot combined kline ws returned nil channel")
				time.Sleep(spotLiveKlineWSRetryGap)
				continue
			}
			select {
			case <-doneC:
				time.Sleep(spotLiveKlineWSRetryGap)
				continue
			case <-spotLiveKlineWS.updateCh:
				select {
				case stopC <- struct{}{}:
				case <-time.After(time.Second):
				}
				debounceSpotLiveKlineWSUpdates()
				continue
			}
		}
	}
}

func debounceSpotLiveKlineWSUpdates() {
	timer := time.NewTimer(spotLiveKlineWSDebounce)
	defer timer.Stop()
	for {
		select {
		case <-spotLiveKlineWS.updateCh:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(spotLiveKlineWSDebounce)
		case <-timer.C:
			return
		}
	}
}

func applySpotLiveKlineWSEvent(event *spotbinance.WsKlineEvent, receivedAt time.Time) {
	if event == nil {
		return
	}
	symbol := strings.ToUpper(strings.TrimSpace(event.Symbol))
	interval := strings.TrimSpace(event.Kline.Interval)
	if symbol == "" || interval == "" {
		return
	}
	row := &spotbinance.Kline{
		OpenTime: event.Kline.StartTime, Open: event.Kline.Open, High: event.Kline.High,
		Low: event.Kline.Low, Close: event.Kline.Close, Volume: event.Kline.Volume,
		CloseTime: event.Kline.EndTime, QuoteAssetVolume: event.Kline.QuoteVolume,
		TradeNum: event.Kline.TradeNum, TakerBuyBaseAssetVolume: event.Kline.ActiveBuyVolume,
		TakerBuyQuoteAssetVolume: event.Kline.ActiveBuyQuoteVolume,
	}
	key := spotLiveKlineCacheKey(symbol, interval)
	spotLiveKlineCacheMu.Lock()
	defer spotLiveKlineCacheMu.Unlock()
	entry, ok := spotLiveKlineCache[key]
	if !ok || entry.err != nil || len(entry.rows) == 0 || entry.rows[0] == nil {
		return
	}
	latest := entry.rows[0]
	switch {
	case row.OpenTime == latest.OpenTime:
		entry.rows[0] = row
	case row.OpenTime == latest.CloseTime+1:
		entry.rows = append([]*spotbinance.Kline{row}, entry.rows...)
		if entry.maxLimit > 0 && len(entry.rows) > entry.maxLimit {
			entry.rows = entry.rows[:entry.maxLimit]
		}
	case row.OpenTime > latest.CloseTime+1:
		entry.expiresAt = time.Time{}
		entry.wsUpdatedAt = 0
		spotLiveKlineCache[key] = entry
		return
	default:
		return
	}
	entry.wsUpdatedAt = receivedAt.UnixMilli()
	entry.expiresAt = receivedAt.Add(spotLiveKlineWSFreshTTL)
	spotLiveKlineCache[key] = entry
}

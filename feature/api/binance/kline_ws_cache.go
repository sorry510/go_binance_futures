package binance

import (
	"sort"
	"strings"
	"sync"
	"time"

	"go_binance_futures/service/binanceapiusage"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/core/logs"
)

const (
	liveKlineWSFreshTTL    = 3 * time.Second
	liveKlineWSRetryGap    = 3 * time.Second
	liveKlineWSDebounce    = 250 * time.Millisecond
	liveKlineWSMaxStreams  = 256
	liveKlineWSInactiveTTL = 15 * time.Minute
)

var liveKlineWS = struct {
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

func ensureLiveKlineWS(symbol, interval string) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	interval = strings.TrimSpace(interval)
	if symbol == "" || interval == "" {
		return
	}

	liveKlineWS.Lock()
	intervals := liveKlineWS.subscriptions[symbol]
	if intervals == nil {
		intervals = make(map[string]struct{})
		liveKlineWS.subscriptions[symbol] = intervals
	}
	key := symbol + "|" + interval
	_, exists := intervals[interval]
	if !exists {
		intervals[interval] = struct{}{}
	}
	wasActive := liveKlineWS.active[key]
	liveKlineWS.lastUsed[key] = time.Now().UnixMilli()
	if !liveKlineWS.started {
		liveKlineWS.started = true
		go runLiveKlineWS()
	}
	liveKlineWS.Unlock()

	if !exists || !wasActive {
		select {
		case liveKlineWS.updateCh <- struct{}{}:
		default:
		}
	}
}

func liveKlineSubscriptions() map[string][]string {
	type pair struct {
		symbol   string
		interval string
		lastUsed int64
	}

	liveKlineWS.Lock()
	defer liveKlineWS.Unlock()

	cutoff := time.Now().Add(-liveKlineWSInactiveTTL).UnixMilli()
	pairs := make([]pair, 0, len(liveKlineWS.lastUsed))
	for symbol, intervalSet := range liveKlineWS.subscriptions {
		for interval := range intervalSet {
			key := symbol + "|" + interval
			lastUsed := liveKlineWS.lastUsed[key]
			if lastUsed < cutoff {
				delete(intervalSet, interval)
				delete(liveKlineWS.lastUsed, key)
				continue
			}
			pairs = append(pairs, pair{symbol: symbol, interval: interval, lastUsed: lastUsed})
		}
		if len(intervalSet) == 0 {
			delete(liveKlineWS.subscriptions, symbol)
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
	if len(pairs) > liveKlineWSMaxStreams {
		pairs = pairs[:liveKlineWSMaxStreams]
	}

	result := make(map[string][]string)
	for _, item := range pairs {
		result[item.symbol] = append(result[item.symbol], item.interval)
	}
	for symbol := range result {
		sort.Strings(result[symbol])
	}
	return result
}

func setActiveLiveKlineSubscriptions(subscriptions map[string][]string) {
	active := make(map[string]bool)
	for symbol, intervals := range subscriptions {
		for _, interval := range intervals {
			active[symbol+"|"+interval] = true
		}
	}
	liveKlineWS.Lock()
	liveKlineWS.active = active
	liveKlineWS.Unlock()
}

func runLiveKlineWS() {
	for {
		<-liveKlineWS.updateCh
		debounceLiveKlineWSUpdates()

		for {
			subscriptions := liveKlineSubscriptions()
			if len(subscriptions) == 0 {
				break
			}
			setActiveLiveKlineSubscriptions(subscriptions)
			doneC, stopC, err := wsFuturesCombinedKlineServeMultiInterval(
				subscriptions,
				func(event *futures.WsKlineEvent) {
					applyLiveKlineWSEvent(event, time.Now())
				},
				func(runErr error) {
					if runErr != nil {
						logs.Error("futures combined kline ws run error:", runErr)
					}
				},
			)
			if err != nil {
				logs.Error("futures combined kline ws start error:", err)
				time.Sleep(liveKlineWSRetryGap)
				continue
			}
			if doneC == nil || stopC == nil {
				logs.Error("futures combined kline ws returned nil channel")
				time.Sleep(liveKlineWSRetryGap)
				continue
			}

			select {
			case <-doneC:
				time.Sleep(liveKlineWSRetryGap)
				continue
			case <-liveKlineWS.updateCh:
				// Subscription sets are immutable for the SDK combined stream.
				// Rebuild the one shared stream only when a new symbol/interval
				// appears, not on every strategy evaluation.
				select {
				case stopC <- struct{}{}:
				case <-time.After(time.Second):
				}
				debounceLiveKlineWSUpdates()
				continue
			}
		}
	}
}

func debounceLiveKlineWSUpdates() {
	timer := time.NewTimer(liveKlineWSDebounce)
	defer timer.Stop()
	for {
		select {
		case <-liveKlineWS.updateCh:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(liveKlineWSDebounce)
		case <-timer.C:
			return
		}
	}
}

func applyLiveKlineWSEvent(event *futures.WsKlineEvent, receivedAt time.Time) {
	if event == nil {
		return
	}
	symbol := strings.ToUpper(strings.TrimSpace(event.Symbol))
	interval := strings.TrimSpace(event.Kline.Interval)
	if symbol == "" || interval == "" {
		return
	}

	row := &futures.Kline{
		OpenTime:                 event.Kline.StartTime,
		Open:                     event.Kline.Open,
		High:                     event.Kline.High,
		Low:                      event.Kline.Low,
		Close:                    event.Kline.Close,
		Volume:                   event.Kline.Volume,
		CloseTime:                event.Kline.EndTime,
		QuoteAssetVolume:         event.Kline.QuoteVolume,
		TradeNum:                 event.Kline.TradeNum,
		TakerBuyBaseAssetVolume:  event.Kline.ActiveBuyVolume,
		TakerBuyQuoteAssetVolume: event.Kline.ActiveBuyQuoteVolume,
	}

	key := liveKlineCacheKey(symbol, interval, 0)
	liveKlineCacheMu.Lock()
	defer liveKlineCacheMu.Unlock()

	entry, ok := liveKlineCache[key]
	if !ok || entry.err != nil || len(entry.rows) == 0 || entry.rows[0] == nil {
		return
	}
	latest := entry.rows[0]
	switch {
	case row.OpenTime == latest.OpenTime:
		entry.rows[0] = row
	case row.OpenTime == latest.CloseTime+1:
		entry.rows = append([]*futures.Kline{row}, entry.rows...)
		if entry.maxLimit > 0 && len(entry.rows) > entry.maxLimit {
			entry.rows = entry.rows[:entry.maxLimit]
		}
	case row.OpenTime > latest.CloseTime+1:
		// A missed bar makes incremental state unsafe. Expire the cache so
		// the next caller performs an authoritative REST bootstrap.
		entry.expiresAt = time.Time{}
		entry.wsUpdatedAt = 0
		liveKlineCache[key] = entry
		return
	default:
		return
	}
	entry.wsUpdatedAt = receivedAt.UnixMilli()
	entry.expiresAt = receivedAt.Add(liveKlineWSFreshTTL)
	liveKlineCache[key] = entry
}

func recordLiveKlineCacheHit(source string, wsFresh bool) {
	if wsFresh {
		binanceapiusage.RecordOptimization(source, "local_ws_hit", 1)
	} else {
		binanceapiusage.RecordOptimization(source, "cache_hit", 1)
	}
	binanceapiusage.RecordOptimization(source, "prevented_duplicate", 1)
}

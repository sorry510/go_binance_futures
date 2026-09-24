package binance

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"go_binance_futures/notify"
	"go_binance_futures/service/binanceapiusage"

	"github.com/adshao/go-binance/v2/common"
	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/core/logs"
	"golang.org/x/sync/singleflight"
)

// The global V4-5 Budget Coordinator now owns normal request pacing across
// Futures/Spot/Delivery. This legacy guard only keeps the explicit exchange-ban
// cooldown parsed from Binance -1003 errors; it no longer applies an unrelated
// fixed 1000-weight/minute throttle to K-line/history requests.
const (
	// A single rate-limit incident sends exactly three notifications:
	// immediately, after two minutes, and after four minutes.
	futuresRateLimitAlertCount    = 3
	futuresRateLimitAlertInterval = 2 * time.Minute
)

var (
	futuresMarketDataMu            sync.Mutex
	futuresMarketDataCooldownUntil time.Time

	liveKlineCacheMu sync.Mutex
	liveKlineCache   = make(map[string]liveKlineCacheEntry)
	liveKlineGroup   singleflight.Group

	futuresRateLimitAlertMu     sync.Mutex
	futuresRateLimitAlertActive bool
)

type liveKlineCacheEntry struct {
	rows        []*futures.Kline
	err         error
	expiresAt   time.Time
	wsUpdatedAt int64
	maxLimit    int
}

const (
	liveKlineNegativeCacheTTL = 2 * time.Second
	liveKlineCacheMaxEntries  = 256
)

var futuresBanUntilPattern = regexp.MustCompile("(?i)banned until\\s+([0-9]{10,})")

func futuresKlineRequestWeight(limit int) int {
	switch {
	case limit < 100:
		return 1
	case limit < 500:
		return 2
	case limit <= 1000:
		return 5
	default:
		return 10
	}
}

// waitFuturesMarketDataREST keeps only an explicit Binance ban/cooldown.
func waitFuturesMarketDataREST(ctx context.Context, weight int) error {
	if ctx == nil {
		ctx = context.Background()
	}
	_ = weight

	futuresMarketDataMu.Lock()
	readyAt := futuresMarketDataCooldownUntil
	futuresMarketDataMu.Unlock()
	if !readyAt.After(time.Now()) {
		return nil
	}

	timer := time.NewTimer(time.Until(readyAt))
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func noteFuturesAPIError(err error) {
	if err == nil {
		return
	}

	var apiErr *common.APIError
	if !errors.As(err, &apiErr) || apiErr.Code != -1003 {
		return
	}

	until := time.Now().Add(time.Minute)
	if match := futuresBanUntilPattern.FindStringSubmatch(apiErr.Message); len(match) == 2 {
		if millis, parseErr := strconv.ParseInt(match[1], 10, 64); parseErr == nil {
			parsed := time.UnixMilli(millis)
			if parsed.After(until) {
				until = parsed
			}
		}
	}

	// Add a small safety margin so the first retry does not land exactly on the
	// exchange's ban boundary.
	until = until.Add(2 * time.Second)

	futuresMarketDataMu.Lock()
	if until.After(futuresMarketDataCooldownUntil) {
		futuresMarketDataCooldownUntil = until
	}
	futuresMarketDataMu.Unlock()

	startFuturesRateLimitAlert(apiErr.Message, until)
}

func startFuturesRateLimitAlert(message string, banUntil time.Time) {
	futuresRateLimitAlertMu.Lock()
	if futuresRateLimitAlertActive {
		futuresRateLimitAlertMu.Unlock()
		return
	}
	futuresRateLimitAlertActive = true
	futuresRateLimitAlertMu.Unlock()

	go func() {
		defer func() {
			futuresRateLimitAlertMu.Lock()
			futuresRateLimitAlertActive = false
			futuresRateLimitAlertMu.Unlock()
		}()

		runFuturesRateLimitAlertSequence(
			futuresRateLimitAlertCount,
			futuresRateLimitAlertInterval,
			func(sequence int) {
				sendFuturesRateLimitAlert(sequence, message, banUntil)
			},
		)
	}()
}

func runFuturesRateLimitAlertSequence(count int, interval time.Duration, send func(sequence int)) {
	if count <= 0 || send == nil {
		return
	}
	for sequence := 1; sequence <= count; sequence++ {
		if sequence > 1 {
			timer := time.NewTimer(interval)
			<-timer.C
		}
		send(sequence)
	}
}

func sendFuturesRateLimitAlert(sequence int, message string, banUntil time.Time) {
	content := fmt.Sprintf(`## Binance API 限流告警
#### **状态**：Futures REST API 已触发限流
#### **通知次数**：%d/%d
#### **解禁/冷却至**：%s
#### **发生时间**：%s
#### **错误**：%s`,
		sequence,
		futuresRateLimitAlertCount,
		banUntil.Local().Format("2006-01-02 15:04:05"),
		time.Now().Format("2006-01-02 15:04:05"),
		message,
	)

	alertPusher := pusher.SetModuleName("system")
	switch alertPusher.(type) {
	case notify.DingDing:
		notify.DingDingApi(content, alertPusher)
	case notify.Slack:
		notify.SlackApi(content, alertPusher)
	default:
		logs.Error("unsupported notification channel for Binance API rate-limit alert")
	}
}

func liveKlineCacheTTL(interval string) time.Duration {
	switch strings.TrimSpace(interval) {
	case "1m":
		return time.Second
	case "3m", "5m":
		return 2 * time.Second
	case "15m", "30m":
		return 5 * time.Second
	default:
		// 1h+ strategies do not benefit from hitting REST every 1.5 seconds.
		// Ten seconds still keeps the in-progress bar fresh while collapsing
		// repeated scanner/close-check requests.
		return 10 * time.Second
	}
}

func liveKlineCacheKey(symbol, interval string, limit int) string {
	_ = limit // one canonical cache per symbol+interval; maxLimit lives in the entry.
	return strings.ToUpper(strings.TrimSpace(symbol)) + "|" + strings.TrimSpace(interval)
}

func cloneKlines(rows []*futures.Kline) []*futures.Kline {
	if rows == nil {
		return nil
	}
	return append([]*futures.Kline(nil), rows...)
}

func loadLiveKlineCache(key string, limit int) ([]*futures.Kline, error, bool, bool) {
	now := time.Now()
	liveKlineCacheMu.Lock()
	defer liveKlineCacheMu.Unlock()

	entry, ok := liveKlineCache[key]
	if !ok {
		return nil, nil, false, false
	}
	if !entry.expiresAt.After(now) {
		delete(liveKlineCache, key)
		return nil, nil, false, false
	}
	if entry.err == nil && entry.maxLimit < limit {
		return nil, nil, false, false
	}
	rows := cloneKlines(entry.rows)
	if limit > 0 && len(rows) > limit {
		rows = rows[:limit]
	}
	wsFresh := entry.wsUpdatedAt > 0 && now.Sub(time.UnixMilli(entry.wsUpdatedAt)) <= liveKlineWSFreshTTL
	return rows, entry.err, true, wsFresh
}

func storeLiveKlineCache(key string, rows []*futures.Kline, err error, ttl time.Duration, limit int) {
	if ttl <= 0 {
		return
	}
	now := time.Now()
	liveKlineCacheMu.Lock()
	defer liveKlineCacheMu.Unlock()

	for cacheKey, entry := range liveKlineCache {
		if !entry.expiresAt.After(now) {
			delete(liveKlineCache, cacheKey)
		}
	}

	if len(liveKlineCache) >= liveKlineCacheMaxEntries {
		oldestKey := ""
		var oldestExpiry time.Time
		for cacheKey, entry := range liveKlineCache {
			if oldestKey == "" || entry.expiresAt.Before(oldestExpiry) {
				oldestKey = cacheKey
				oldestExpiry = entry.expiresAt
			}
		}
		if oldestKey != "" {
			delete(liveKlineCache, oldestKey)
		}
	}

	if existing, ok := liveKlineCache[key]; ok && err == nil && existing.err == nil && existing.maxLimit > limit {
		merged := cloneKlines(existing.rows)
		for i := 0; i < len(rows) && i < len(merged); i++ {
			merged[i] = rows[i]
		}
		liveKlineCache[key] = liveKlineCacheEntry{
			rows: merged, expiresAt: now.Add(ttl), maxLimit: existing.maxLimit,
		}
		return
	}
	liveKlineCache[key] = liveKlineCacheEntry{
		rows: cloneKlines(rows), err: err, expiresAt: now.Add(ttl), maxLimit: limit,
	}
}

func getLiveKlineData(ctx context.Context, symbol, interval string, limit int) ([]*futures.Kline, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	interval = strings.TrimSpace(interval)
	if symbol == "" || interval == "" || limit <= 0 {
		return nil, fmt.Errorf("K-line requires symbol, interval and positive limit")
	}

	ensureLiveKlineWS(symbol, interval)
	key := liveKlineCacheKey(symbol, interval, limit)
	source := binanceapiusage.SourceFromContext(ctx)
	if rows, err, ok, wsFresh := loadLiveKlineCache(key, limit); ok {
		recordLiveKlineCacheHit(source, wsFresh)
		return rows, err
	}

	value, err, shared := liveKlineGroup.Do(key, func() (interface{}, error) {
		if rows, cachedErr, ok, wsFresh := loadLiveKlineCache(key, limit); ok {
			recordLiveKlineCacheHit(source, wsFresh)
			return rows, cachedErr
		}

		if waitErr := waitFuturesMarketDataREST(ctx, futuresKlineRequestWeight(limit)); waitErr != nil {
			return nil, waitErr
		}
		rows, requestErr := futuresClient.NewKlinesService().
			Symbol(symbol).
			Interval(interval).
			Limit(limit).
			Do(ctx)
		if requestErr != nil {
			noteFuturesAPIError(requestErr)
			storeLiveKlineCache(key, nil, requestErr, liveKlineNegativeCacheTTL, limit)
			return nil, requestErr
		}
		sort.Slice(rows, func(i, j int) bool {
			return rows[i].OpenTime > rows[j].OpenTime
		})
		storeLiveKlineCache(key, rows, nil, liveKlineCacheTTL(interval), limit)
		return cloneKlines(rows), nil
	})
	if shared {
		binanceapiusage.RecordOptimization(source, "coalesced", 1)
		binanceapiusage.RecordOptimization(source, "prevented_duplicate", 1)
	}
	if err != nil {
		return nil, err
	}
	rows, ok := value.([]*futures.Kline)
	if !ok {
		return nil, fmt.Errorf("unexpected K-line cache value type %T", value)
	}
	return cloneKlines(rows), nil
}

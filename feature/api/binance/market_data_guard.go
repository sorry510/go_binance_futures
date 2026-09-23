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

	"github.com/adshao/go-binance/v2/common"
	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/core/logs"
	"golang.org/x/sync/singleflight"
)

// Keep public market-data traffic well below Binance's IP-wide REQUEST_WEIGHT
// ceiling. Trading requests are intentionally not delayed by this guard.
//
// The configured budget is conservative because the same public IP is also
// used by depth/ticker/OI/account endpoints elsewhere in the application.
const (
	futuresMarketDataWeightPerMinute = 1000

	// A single rate-limit incident sends exactly three notifications:
	// immediately, after two minutes, and after four minutes.
	futuresRateLimitAlertCount    = 3
	futuresRateLimitAlertInterval = 2 * time.Minute
)

var (
	futuresMarketDataMu            sync.Mutex
	futuresMarketDataNextRequestAt time.Time
	futuresMarketDataCooldownUntil time.Time

	liveKlineCacheMu sync.Mutex
	liveKlineCache   = make(map[string]liveKlineCacheEntry)
	liveKlineGroup   singleflight.Group

	futuresRateLimitAlertMu     sync.Mutex
	futuresRateLimitAlertActive bool
)

type liveKlineCacheEntry struct {
	rows      []*futures.Kline
	err       error
	expiresAt time.Time
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

func futuresMarketDataSpacing(weight int) time.Duration {
	if weight < 1 {
		weight = 1
	}
	return time.Minute * time.Duration(weight) / futuresMarketDataWeightPerMinute
}

// waitFuturesMarketDataREST reserves REQUEST_WEIGHT before a public market-data
// request. Reservations are process-wide so live strategy evaluation and
// historical prefetch cannot independently consume the same IP budget.
func waitFuturesMarketDataREST(ctx context.Context, weight int) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if weight < 1 {
		weight = 1
	}

	futuresMarketDataMu.Lock()
	defer futuresMarketDataMu.Unlock()

	for {
		now := time.Now()
		readyAt := futuresMarketDataNextRequestAt
		if futuresMarketDataCooldownUntil.After(readyAt) {
			readyAt = futuresMarketDataCooldownUntil
		}
		if !readyAt.After(now) {
			futuresMarketDataNextRequestAt = now.Add(futuresMarketDataSpacing(weight))
			return nil
		}

		timer := time.NewTimer(time.Until(readyAt))
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return ctx.Err()
		case <-timer.C:
		}
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
	return strings.ToUpper(strings.TrimSpace(symbol)) + "|" + strings.TrimSpace(interval) + "|" + strconv.Itoa(limit)
}

func cloneKlines(rows []*futures.Kline) []*futures.Kline {
	if rows == nil {
		return nil
	}
	return append([]*futures.Kline(nil), rows...)
}

func loadLiveKlineCache(key string) ([]*futures.Kline, error, bool) {
	now := time.Now()
	liveKlineCacheMu.Lock()
	defer liveKlineCacheMu.Unlock()

	entry, ok := liveKlineCache[key]
	if !ok {
		return nil, nil, false
	}
	if !entry.expiresAt.After(now) {
		delete(liveKlineCache, key)
		return nil, nil, false
	}
	return cloneKlines(entry.rows), entry.err, true
}

func storeLiveKlineCache(key string, rows []*futures.Kline, err error, ttl time.Duration) {
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

	liveKlineCache[key] = liveKlineCacheEntry{
		rows:      cloneKlines(rows),
		err:       err,
		expiresAt: now.Add(ttl),
	}
}

func getLiveKlineData(ctx context.Context, symbol, interval string, limit int) ([]*futures.Kline, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	interval = strings.TrimSpace(interval)
	if symbol == "" || interval == "" || limit <= 0 {
		return nil, fmt.Errorf("K-line requires symbol, interval and positive limit")
	}

	key := liveKlineCacheKey(symbol, interval, limit)
	if rows, err, ok := loadLiveKlineCache(key); ok {
		return rows, err
	}

	value, err, _ := liveKlineGroup.Do(key, func() (interface{}, error) {
		if rows, cachedErr, ok := loadLiveKlineCache(key); ok {
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
			storeLiveKlineCache(key, nil, requestErr, liveKlineNegativeCacheTTL)
			return nil, requestErr
		}
		sort.Slice(rows, func(i, j int) bool {
			return rows[i].OpenTime > rows[j].OpenTime
		})
		storeLiveKlineCache(key, rows, nil, liveKlineCacheTTL(interval))
		return cloneKlines(rows), nil
	})
	if err != nil {
		return nil, err
	}
	rows, ok := value.([]*futures.Kline)
	if !ok {
		return nil, fmt.Errorf("unexpected K-line cache value type %T", value)
	}
	return cloneKlines(rows), nil
}

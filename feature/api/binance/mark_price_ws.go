package binance

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go_binance_futures/service/binanceapiusage"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/core/logs"
)

const (
	futuresMarkPriceWSRate   = 3 * time.Second
	futuresMarkPriceMaxAge   = 6 * time.Second
	futuresMarkPriceRetryGap = 3 * time.Second
)

type markPriceSnapshotEntry struct {
	Data       futures.PremiumIndex
	ReceivedAt int64
}

var (
	futuresMarkPriceOnce       sync.Once
	futuresMarkPriceUp         atomic.Bool
	futuresMarkPriceLastFullAt atomic.Int64
	futuresMarkPriceData       = struct {
		sync.RWMutex
		Items map[string]markPriceSnapshotEntry
	}{Items: make(map[string]markPriceSnapshotEntry)}
)

func ensureFuturesMarkPriceWS() {
	futuresMarkPriceOnce.Do(func() {
		go runFuturesMarkPriceWS()
	})
}

func runFuturesMarkPriceWS() {
	for {
		runErrCh := make(chan error, 1)
		doneC, _, err := wsFuturesAllMarkPriceServeWithRate(
			futuresMarkPriceWSRate,
			func(events futures.WsAllMarkPriceEvent) {
				storeFuturesMarkPriceEvents(events, time.Now())
				futuresMarkPriceUp.Store(true)
			},
			func(runErr error) {
				select {
				case runErrCh <- runErr:
				default:
				}
			},
		)
		if err != nil {
			futuresMarkPriceUp.Store(false)
			logs.Error("futures mark-price ws start error:", err)
			time.Sleep(futuresMarkPriceRetryGap)
			continue
		}
		if doneC == nil {
			futuresMarkPriceUp.Store(false)
			logs.Error("futures mark-price ws closed immediately: done channel is nil")
			time.Sleep(futuresMarkPriceRetryGap)
			continue
		}

		<-doneC
		futuresMarkPriceUp.Store(false)
		select {
		case runErr := <-runErrCh:
			if runErr != nil {
				logs.Error("futures mark-price ws run error, restarting:", runErr)
			}
		default:
			logs.Error("futures mark-price ws closed, restarting")
		}
		time.Sleep(futuresMarkPriceRetryGap)
	}
}

func storeFuturesMarkPriceEvents(events futures.WsAllMarkPriceEvent, receivedAt time.Time) {
	if len(events) == 0 {
		return
	}
	receivedMS := receivedAt.UnixMilli()
	futuresMarkPriceData.Lock()
	defer futuresMarkPriceData.Unlock()
	for _, event := range events {
		if event == nil {
			continue
		}
		symbol := strings.ToUpper(strings.TrimSpace(event.Symbol))
		if symbol == "" {
			continue
		}
		futuresMarkPriceData.Items[symbol] = markPriceSnapshotEntry{
			Data: futures.PremiumIndex{
				Symbol:               symbol,
				MarkPrice:            event.MarkPrice,
				IndexPrice:           event.IndexPrice,
				EstimatedSettlePrice: event.EstimatedSettlePrice,
				LastFundingRate:      event.FundingRate,
				NextFundingTime:      event.NextFundingTime,
				Time:                 event.Time,
			},
			ReceivedAt: receivedMS,
		}
	}
	futuresMarkPriceLastFullAt.Store(receivedMS)
}

func loadFreshFuturesPremiumIndex(symbol string, maxAge time.Duration) ([]*futures.PremiumIndex, bool) {
	ensureFuturesMarkPriceWS()
	return loadFreshFuturesPremiumIndexSnapshot(symbol, maxAge)
}

func loadFreshFuturesPremiumIndexSnapshot(symbol string, maxAge time.Duration) ([]*futures.PremiumIndex, bool) {
	now := time.Now()
	symbol = strings.ToUpper(strings.TrimSpace(symbol))

	futuresMarkPriceData.RLock()
	defer futuresMarkPriceData.RUnlock()

	if symbol != "" {
		entry, ok := futuresMarkPriceData.Items[symbol]
		if !ok || entry.ReceivedAt <= 0 || (maxAge > 0 && now.Sub(time.UnixMilli(entry.ReceivedAt)) > maxAge) {
			return nil, false
		}
		row := entry.Data
		return []*futures.PremiumIndex{&row}, true
	}

	lastFullAt := futuresMarkPriceLastFullAt.Load()
	if lastFullAt <= 0 || (maxAge > 0 && now.Sub(time.UnixMilli(lastFullAt)) > maxAge) {
		return nil, false
	}
	rows := make([]*futures.PremiumIndex, 0, len(futuresMarkPriceData.Items))
	for _, entry := range futuresMarkPriceData.Items {
		if entry.ReceivedAt != lastFullAt {
			continue
		}
		row := entry.Data
		rows = append(rows, &row)
	}
	if len(rows) == 0 {
		return nil, false
	}
	return rows, true
}

func GetFreshFuturesMarkPrice(symbol string) (string, bool) {
	rows, ok := loadFreshFuturesPremiumIndex(symbol, futuresMarkPriceMaxAge)
	if !ok || len(rows) != 1 || rows[0] == nil || strings.TrimSpace(rows[0].MarkPrice) == "" {
		return "", false
	}
	return rows[0].MarkPrice, true
}

func recordMarkPriceWSHit(ctxSource string) {
	binanceapiusage.RecordOptimization(ctxSource, "local_ws_hit", 1)
	binanceapiusage.RecordOptimization(ctxSource, "prevented_duplicate", 1)
}

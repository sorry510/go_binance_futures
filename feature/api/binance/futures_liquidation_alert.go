package binance

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/beego/beego/v2/client/orm"

	"go_binance_futures/lang"
	"go_binance_futures/models"
	"go_binance_futures/notify"
)

const (
	futuresLiquidationAlertSymbol                   = "BTCUSDT"
	futuresLiquidationAlertModule                   = "futures_liquidation"
	futuresLiquidationAlertEventType                = "futures_liquidation_aggregate"
	defaultFuturesLiquidationAlertWindowSec         = 60
	defaultFuturesLiquidationAlertNotionalThreshold = 5_000_000
	defaultFuturesLiquidationAlertCooldownSec       = 300
)

type futuresLiquidationAlertAggregate struct {
	notional   float64
	orderCount int
	startTime  int64
	endTime    int64
}

type futuresLiquidationAlertStore interface {
	listOrders(symbol, orderSide string, startExclusive, endInclusive int64) ([]models.FuturesLiquidationOrder, error)
	latestAlert(symbol, liquidationSide string) (*models.Notification, error)
}

type ormFuturesLiquidationAlertStore struct{}

func (ormFuturesLiquidationAlertStore) listOrders(
	symbol string,
	orderSide string,
	startExclusive int64,
	endInclusive int64,
) ([]models.FuturesLiquidationOrder, error) {
	var orders []models.FuturesLiquidationOrder
	_, err := orm.NewOrm().QueryTable(new(models.FuturesLiquidationOrder)).
		Filter("symbol", symbol).
		Filter("side", orderSide).
		Filter("event_time__gt", startExclusive).
		Filter("event_time__lte", endInclusive).
		OrderBy("event_time", "id").
		All(&orders)
	return orders, err
}

func (ormFuturesLiquidationAlertStore) latestAlert(
	symbol string,
	liquidationSide string,
) (*models.Notification, error) {
	notification := &models.Notification{}
	err := orm.NewOrm().QueryTable(new(models.Notification)).
		Filter("event_type", futuresLiquidationAlertEventType).
		Filter("symbol", symbol).
		Filter("liquidation_side", liquidationSide).
		OrderBy("-create_time", "-id").
		One(notification)
	if err == orm.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return notification, nil
}

type futuresLiquidationAlertAggregator struct {
	mu    sync.Mutex
	now   func() time.Time
	push  func(notify.FuturesLiquidationAggregateParams)
	store futuresLiquidationAlertStore
}

type futuresLiquidationAlertSettings struct {
	windowSec   int
	threshold   float64
	cooldownSec int
}

var defaultFuturesLiquidationAlertAggregator = newFuturesLiquidationAlertAggregator(
	time.Now,
	func(params notify.FuturesLiquidationAggregateParams) {
		pusher := notify.GetNotifyChannel()
		pusher.SetModuleName(futuresLiquidationAlertModule).FuturesLiquidationAggregate(params)
	},
	ormFuturesLiquidationAlertStore{},
)

func newFuturesLiquidationAlertAggregator(
	now func() time.Time,
	push func(notify.FuturesLiquidationAggregateParams),
	store futuresLiquidationAlertStore,
) *futuresLiquidationAlertAggregator {
	return &futuresLiquidationAlertAggregator{
		now:   now,
		push:  push,
		store: store,
	}
}

func (aggregator *futuresLiquidationAlertAggregator) Evaluate(
	item models.FuturesLiquidationOrder,
	systemConfig *models.Config,
) (bool, error) {
	if strings.ToUpper(strings.TrimSpace(item.Symbol)) != futuresLiquidationAlertSymbol || item.Notional <= 0 {
		return false, nil
	}

	liquidationSide, ok := liquidationPositionSide(item.Side)
	if !ok {
		return false, nil
	}

	aggregator.mu.Lock()
	defer aggregator.mu.Unlock()

	settings := getFuturesLiquidationAlertSettings(systemConfig)
	nowMillis := aggregator.now().UnixMilli()
	windowEnd := nowMillis
	if item.EventTime > windowEnd {
		windowEnd = item.EventTime
	}
	windowStart := windowEnd - int64(settings.windowSec)*int64(time.Second/time.Millisecond)

	latestAlert, err := aggregator.store.latestAlert(futuresLiquidationAlertSymbol, liquidationSide)
	if err != nil {
		return false, err
	}
	if latestAlert != nil {
		cooldownMillis := int64(settings.cooldownSec) * int64(time.Second/time.Millisecond)
		if nowMillis-latestAlert.CreateTime < cooldownMillis {
			return false, nil
		}
		if latestAlert.WindowEnd > windowStart {
			windowStart = latestAlert.WindowEnd
		}
	}

	orders, err := aggregator.store.listOrders(
		futuresLiquidationAlertSymbol,
		strings.ToUpper(strings.TrimSpace(item.Side)),
		windowStart,
		windowEnd,
	)
	if err != nil {
		return false, err
	}
	aggregate := aggregateFuturesLiquidationOrders(orders)
	if aggregate.orderCount == 0 || aggregate.notional < settings.threshold {
		return false, nil
	}

	aggregator.push(notify.FuturesLiquidationAggregateParams{
		Title:             lang.Lang("futures.liquidation_aggregate_title"),
		Symbol:            futuresLiquidationAlertSymbol,
		LiquidationSide:   liquidationSide,
		AggregateNotional: aggregate.notional,
		OrderCount:        aggregate.orderCount,
		WindowSec:         settings.windowSec,
		Threshold:         settings.threshold,
		WindowStart:       aggregate.startTime,
		WindowEnd:         aggregate.endTime,
	})
	return true, nil
}

func aggregateFuturesLiquidationOrders(orders []models.FuturesLiquidationOrder) futuresLiquidationAlertAggregate {
	result := futuresLiquidationAlertAggregate{}
	seen := make(map[string]struct{}, len(orders))
	for _, order := range orders {
		if order.Notional <= 0 {
			continue
		}
		key := futuresLiquidationAlertEventKey(order)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result.notional += order.Notional
		result.orderCount++
		if result.startTime == 0 || order.EventTime < result.startTime {
			result.startTime = order.EventTime
		}
		if order.EventTime > result.endTime {
			result.endTime = order.EventTime
		}
	}
	return result
}

func getFuturesLiquidationAlertSettings(systemConfig *models.Config) futuresLiquidationAlertSettings {
	settings := futuresLiquidationAlertSettings{
		windowSec:   defaultFuturesLiquidationAlertWindowSec,
		threshold:   defaultFuturesLiquidationAlertNotionalThreshold,
		cooldownSec: defaultFuturesLiquidationAlertCooldownSec,
	}
	if systemConfig == nil {
		return settings
	}
	if systemConfig.WsFuturesLiquidationAlertWindowSec > 0 {
		settings.windowSec = systemConfig.WsFuturesLiquidationAlertWindowSec
	}
	if systemConfig.WsFuturesLiquidationAlertNotionalThreshold > 0 {
		settings.threshold = systemConfig.WsFuturesLiquidationAlertNotionalThreshold
	}
	if systemConfig.WsFuturesLiquidationAlertCooldownSec > 0 {
		settings.cooldownSec = systemConfig.WsFuturesLiquidationAlertCooldownSec
	}
	return settings
}

func liquidationPositionSide(orderSide string) (string, bool) {
	switch strings.ToUpper(strings.TrimSpace(orderSide)) {
	case "SELL":
		return "long", true
	case "BUY":
		return "short", true
	default:
		return "", false
	}
}

func futuresLiquidationAlertEventKey(item models.FuturesLiquidationOrder) string {
	return fmt.Sprintf("%s|%s|%d|%d|%s|%s|%s|%.8f",
		strings.ToUpper(strings.TrimSpace(item.Symbol)),
		strings.ToUpper(strings.TrimSpace(item.Side)),
		item.EventTime,
		item.TradeTime,
		item.AvgPrice,
		item.Price,
		item.AccumulatedFilledQty,
		item.Notional,
	)
}

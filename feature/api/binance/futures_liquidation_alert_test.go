package binance

import (
	"errors"
	"strings"
	"testing"
	"time"

	"go_binance_futures/models"
	"go_binance_futures/notify"
)

func TestFuturesLiquidationAlertAggregatorQueriesDatabaseAndDeduplicates(t *testing.T) {
	now := time.UnixMilli(1_700_000_060_000)
	store := &fakeFuturesLiquidationAlertStore{}
	longFirst := liquidationAlertTestOrder("SELL", 60, now.Add(-2*time.Second).UnixMilli())
	longSecond := liquidationAlertTestOrder("SELL", 50, now.Add(-time.Second).UnixMilli())
	shortOrder := liquidationAlertTestOrder("BUY", 100, now.UnixMilli())
	store.orders = []models.FuturesLiquidationOrder{longFirst, longFirst, longSecond, shortOrder}

	var alerts []notify.FuturesLiquidationAggregateParams
	aggregator := newFuturesLiquidationAlertAggregator(
		func() time.Time { return now },
		func(params notify.FuturesLiquidationAggregateParams) {
			alerts = append(alerts, params)
			store.saveAlert(params, now.UnixMilli())
		},
		store,
	)
	config := liquidationAlertTestConfig(60, 100, 120)

	alerted, err := aggregator.Evaluate(longSecond, config)
	if err != nil {
		t.Fatal(err)
	}
	if !alerted {
		t.Fatal("database aggregate should alert after crossing threshold")
	}
	alerted, err = aggregator.Evaluate(longSecond, config)
	if err != nil {
		t.Fatal(err)
	}
	if alerted {
		t.Fatal("persisted cooldown should suppress a duplicate alert")
	}
	alerted, err = aggregator.Evaluate(shortOrder, config)
	if err != nil {
		t.Fatal(err)
	}
	if !alerted {
		t.Fatal("short liquidation should alert independently")
	}

	if len(alerts) != 2 {
		t.Fatalf("alert count %d, want 2", len(alerts))
	}
	if alerts[0].LiquidationSide != "long" || alerts[0].OrderCount != 2 || alerts[0].AggregateNotional != 110 {
		t.Fatalf("unexpected long alert: %+v", alerts[0])
	}
	if alerts[1].LiquidationSide != "short" || alerts[1].OrderCount != 1 || alerts[1].AggregateNotional != 100 {
		t.Fatalf("unexpected short alert: %+v", alerts[1])
	}
	if store.listCalls[0].orderSide != "SELL" || store.listCalls[0].startExclusive != now.Add(-60*time.Second).UnixMilli() {
		t.Fatalf("unexpected database query: %+v", store.listCalls[0])
	}
}

func TestFuturesLiquidationAlertAggregatorUsesPersistedCooldownAndCheckpoint(t *testing.T) {
	base := time.UnixMilli(1_700_000_000_000)
	now := base.Add(10 * time.Second)
	checkpoint := base.Add(5 * time.Second).UnixMilli()
	store := &fakeFuturesLiquidationAlertStore{
		orders: []models.FuturesLiquidationOrder{
			liquidationAlertTestOrder("SELL", 100, checkpoint),
			liquidationAlertTestOrder("SELL", 100, base.Add(120*time.Second).UnixMilli()),
		},
		alerts: map[string]*models.Notification{
			"long": {
				EventType:       futuresLiquidationAlertEventType,
				Symbol:          futuresLiquidationAlertSymbol,
				LiquidationSide: "long",
				WindowEnd:       checkpoint,
				CreateTime:      base.UnixMilli(),
			},
		},
	}
	var alerts []notify.FuturesLiquidationAggregateParams
	aggregator := newFuturesLiquidationAlertAggregator(
		func() time.Time { return now },
		func(params notify.FuturesLiquidationAggregateParams) {
			alerts = append(alerts, params)
			store.saveAlert(params, now.UnixMilli())
		},
		store,
	)
	config := liquidationAlertTestConfig(300, 100, 120)

	alerted, err := aggregator.Evaluate(store.orders[0], config)
	if err != nil {
		t.Fatal(err)
	}
	if alerted || len(store.listCalls) != 0 {
		t.Fatal("cooldown should stop before querying the aggregation window")
	}

	now = base.Add(121 * time.Second)
	alerted, err = aggregator.Evaluate(store.orders[1], config)
	if err != nil {
		t.Fatal(err)
	}
	if !alerted {
		t.Fatal("fresh orders after cooldown should alert")
	}
	if len(store.listCalls) != 1 || store.listCalls[0].startExclusive != checkpoint {
		t.Fatalf("query should continue after persisted checkpoint: %+v", store.listCalls)
	}
	if len(alerts) != 1 || alerts[0].OrderCount != 1 || alerts[0].AggregateNotional != 100 {
		t.Fatalf("previously consumed order was reused: %+v", alerts)
	}
}

func TestFuturesLiquidationAlertAggregatorHonorsDatabaseWindow(t *testing.T) {
	base := time.UnixMilli(1_700_000_000_000)
	now := base.Add(61 * time.Second)
	freshFirst := liquidationAlertTestOrder("SELL", 50, base.Add(2*time.Second).UnixMilli())
	freshSecond := liquidationAlertTestOrder("SELL", 50, base.Add(60*time.Second).UnixMilli())
	store := &fakeFuturesLiquidationAlertStore{
		orders: []models.FuturesLiquidationOrder{
			liquidationAlertTestOrder("SELL", 100, base.UnixMilli()),
			freshFirst,
			freshSecond,
		},
	}
	var alerts []notify.FuturesLiquidationAggregateParams
	aggregator := newFuturesLiquidationAlertAggregator(
		func() time.Time { return now },
		func(params notify.FuturesLiquidationAggregateParams) { alerts = append(alerts, params) },
		store,
	)

	alerted, err := aggregator.Evaluate(freshSecond, liquidationAlertTestConfig(60, 100, 120))
	if err != nil {
		t.Fatal(err)
	}
	if !alerted {
		t.Fatal("orders inside the database window should alert")
	}
	if len(alerts) != 1 || alerts[0].OrderCount != 2 || alerts[0].AggregateNotional != 100 {
		t.Fatalf("expired database row contributed to aggregate: %+v", alerts)
	}
}

func TestFuturesLiquidationAlertAggregatorIgnoresInvalidTriggerAndReturnsStoreError(t *testing.T) {
	store := &fakeFuturesLiquidationAlertStore{}
	aggregator := newFuturesLiquidationAlertAggregator(time.Now, func(notify.FuturesLiquidationAggregateParams) {
		t.Fatal("unexpected alert")
	}, store)
	config := liquidationAlertTestConfig(60, 1, 120)

	otherSymbol := liquidationAlertTestOrder("SELL", 100, time.Now().UnixMilli())
	otherSymbol.Symbol = "ETHUSDT"
	alerted, err := aggregator.Evaluate(otherSymbol, config)
	if err != nil || alerted {
		t.Fatalf("non-BTC trigger result = %v, %v", alerted, err)
	}
	invalidSide := liquidationAlertTestOrder("UNKNOWN", 100, time.Now().UnixMilli())
	alerted, err = aggregator.Evaluate(invalidSide, config)
	if err != nil || alerted {
		t.Fatalf("invalid side trigger result = %v, %v", alerted, err)
	}
	if store.latestCalls != 0 || len(store.listCalls) != 0 {
		t.Fatal("invalid trigger should not query the database")
	}

	store.latestErr = errors.New("notification query failed")
	_, err = aggregator.Evaluate(liquidationAlertTestOrder("SELL", 100, time.Now().UnixMilli()), config)
	if err == nil || !strings.Contains(err.Error(), "notification query failed") {
		t.Fatalf("unexpected store error: %v", err)
	}

	store.latestErr = nil
	store.listErr = errors.New("liquidation query failed")
	_, err = aggregator.Evaluate(liquidationAlertTestOrder("SELL", 100, time.Now().UnixMilli()), config)
	if err == nil || !strings.Contains(err.Error(), "liquidation query failed") {
		t.Fatalf("unexpected liquidation query error: %v", err)
	}
}

type fakeFuturesLiquidationAlertStore struct {
	orders      []models.FuturesLiquidationOrder
	alerts      map[string]*models.Notification
	latestErr   error
	listErr     error
	latestCalls int
	listCalls   []fakeFuturesLiquidationAlertListCall
}

type fakeFuturesLiquidationAlertListCall struct {
	symbol         string
	orderSide      string
	startExclusive int64
	endInclusive   int64
}

func (store *fakeFuturesLiquidationAlertStore) listOrders(
	symbol string,
	orderSide string,
	startExclusive int64,
	endInclusive int64,
) ([]models.FuturesLiquidationOrder, error) {
	store.listCalls = append(store.listCalls, fakeFuturesLiquidationAlertListCall{
		symbol:         symbol,
		orderSide:      orderSide,
		startExclusive: startExclusive,
		endInclusive:   endInclusive,
	})
	if store.listErr != nil {
		return nil, store.listErr
	}
	var result []models.FuturesLiquidationOrder
	for _, order := range store.orders {
		if strings.EqualFold(order.Symbol, symbol) && strings.EqualFold(order.Side, orderSide) &&
			order.EventTime > startExclusive && order.EventTime <= endInclusive {
			result = append(result, order)
		}
	}
	return result, nil
}

func (store *fakeFuturesLiquidationAlertStore) latestAlert(
	_ string,
	liquidationSide string,
) (*models.Notification, error) {
	store.latestCalls++
	if store.latestErr != nil {
		return nil, store.latestErr
	}
	return store.alerts[liquidationSide], nil
}

func (store *fakeFuturesLiquidationAlertStore) saveAlert(
	params notify.FuturesLiquidationAggregateParams,
	createTime int64,
) {
	if store.alerts == nil {
		store.alerts = make(map[string]*models.Notification)
	}
	store.alerts[params.LiquidationSide] = &models.Notification{
		EventType:       futuresLiquidationAlertEventType,
		Symbol:          params.Symbol,
		LiquidationSide: params.LiquidationSide,
		WindowStart:     params.WindowStart,
		WindowEnd:       params.WindowEnd,
		CreateTime:      createTime,
	}
}

func liquidationAlertTestConfig(windowSec int, threshold float64, cooldownSec int) *models.Config {
	return &models.Config{
		WsFuturesLiquidationAlertWindowSec:         windowSec,
		WsFuturesLiquidationAlertNotionalThreshold: threshold,
		WsFuturesLiquidationAlertCooldownSec:       cooldownSec,
	}
}

func liquidationAlertTestOrder(side string, notional float64, eventTime int64) models.FuturesLiquidationOrder {
	return models.FuturesLiquidationOrder{
		Symbol:               futuresLiquidationAlertSymbol,
		Side:                 side,
		AvgPrice:             "50000",
		AccumulatedFilledQty: "1",
		Notional:             notional,
		TradeTime:            eventTime,
		EventTime:            eventTime,
	}
}

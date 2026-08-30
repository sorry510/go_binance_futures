package binance

import (
	"database/sql/driver"
	"errors"
	agentevent "go_binance_futures/agent/event"
	"go_binance_futures/models"
	"strconv"
	"strings"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

const (
	futuresLiquidationOrderKeepDays          = 5
	futuresLiquidationOrderCleanupBatchSize  = 1000
	futuresLiquidationOrderCleanupMaxRetries = 3
	futuresLiquidationOrderCleanupRetryDelay = time.Second
)

// CollectFuturesLiquidationOrders 采集全市场强平订单快照。
// @see https://developers.binance.com/docs/derivatives/usds-margined-futures/websocket-market-streams/All-Market-Liquidation-Order-Streams
func CollectFuturesLiquidationOrders(systemConfig *models.Config) {
	var retryNum int64
	for {
		if systemConfig.WsFuturesLiquidationEnable != 1 {
			time.Sleep(3 * time.Second)
			continue
		}

		if retryNum > 0 {
			logs.Info("futures liquidation ws restart num:", retryNum)
		}
		logs.Info("futures liquidation websocket start: collect liquidation orders")

		doneC, stopC, err := futures.WsAllLiquidationOrderServe(func(event *futures.WsLiquidationOrderEvent) {
			if systemConfig.WsFuturesLiquidationEnable != 1 {
				return
			}

			item, err := saveFuturesLiquidationOrder(event)
			if err != nil {
				logs.Error("save futures liquidation order error:", err)
				return
			}
			publishFuturesLiquidationEvent(item)
			if systemConfig.AgentAlertPipelineEnable != 1 {
				if _, err := defaultFuturesLiquidationAlertAggregator.Evaluate(item, systemConfig); err != nil {
					logs.Error("check futures liquidation aggregate alert error:", err)
				}
			}
		}, func(err error) {
			logs.Error("futures liquidation ws run error:", err)
		})
		if err != nil {
			logs.Error("futures liquidation ws start error:", err)
			retryNum++
			time.Sleep(3 * time.Second)
			continue
		}

		if doneC == nil {
			logs.Error("futures liquidation ws closed immediately: done channel is nil")
			retryNum++
			time.Sleep(3 * time.Second)
			continue
		}

		if waitFuturesLiquidationWs(systemConfig, doneC, stopC) {
			retryNum = 0
			continue
		}

		logs.Error("futures liquidation ws closed, restarting")
		retryNum++
		time.Sleep(3 * time.Second)
	}
}

func waitFuturesLiquidationWs(systemConfig *models.Config, doneC chan struct{}, stopC chan struct{}) bool {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-doneC:
			return false
		case <-ticker.C:
			if systemConfig.WsFuturesLiquidationEnable == 1 {
				continue
			}

			if stopC != nil {
				close(stopC)
				<-doneC
			}
			logs.Info("futures liquidation websocket stop")
			return true
		}
	}
}

func SaveFuturesLiquidationOrder(event *futures.WsLiquidationOrderEvent) error {
	_, err := saveFuturesLiquidationOrder(event)
	return err
}

func saveFuturesLiquidationOrder(event *futures.WsLiquidationOrderEvent) (models.FuturesLiquidationOrder, error) {
	item, err := newFuturesLiquidationOrder(event)
	if err != nil {
		return models.FuturesLiquidationOrder{}, err
	}

	_, err = orm.NewOrm().Insert(&item)
	if err != nil {
		return models.FuturesLiquidationOrder{}, err
	}
	return item, nil
}

func newFuturesLiquidationOrder(event *futures.WsLiquidationOrderEvent) (models.FuturesLiquidationOrder, error) {
	if event == nil {
		return models.FuturesLiquidationOrder{}, errors.New("liquidation event is nil")
	}

	order := event.LiquidationOrder
	symbol := strings.ToUpper(strings.TrimSpace(order.Symbol))
	if symbol == "" {
		return models.FuturesLiquidationOrder{}, errors.New("liquidation order symbol is empty")
	}

	eventTime := event.Time
	if eventTime <= 0 {
		eventTime = time.Now().UnixMilli()
	}
	tradeTime := order.TradeTime
	if tradeTime <= 0 {
		tradeTime = eventTime
	}

	return models.FuturesLiquidationOrder{
		Symbol:               symbol,
		Side:                 string(order.Side),
		OrderType:            string(order.OrderType),
		TimeInForce:          string(order.TimeInForce),
		OrigQuantity:         order.OrigQuantity,
		Price:                order.Price,
		AvgPrice:             order.AvgPrice,
		OrderStatus:          string(order.OrderStatus),
		LastFilledQty:        order.LastFilledQty,
		AccumulatedFilledQty: order.AccumulatedFilledQty,
		Notional:             calculateLiquidationNotional(order),
		TradeTime:            tradeTime,
		EventTime:            eventTime,
		CreateTime:           eventTime,
		UpdateTime:           eventTime,
	}, nil
}

func calculateLiquidationNotional(order futures.WsLiquidationOrder) float64 {
	price, _ := strconv.ParseFloat(order.AvgPrice, 64)
	if price <= 0 {
		price, _ = strconv.ParseFloat(order.Price, 64)
	}

	quantity, _ := strconv.ParseFloat(order.AccumulatedFilledQty, 64)
	if quantity <= 0 {
		quantity, _ = strconv.ParseFloat(order.OrigQuantity, 64)
	}
	return price * quantity
}

func publishFuturesLiquidationEvent(item models.FuturesLiquidationOrder) {
	liquidationSide, ok := liquidationPositionSide(item.Side)
	if !ok || item.Notional <= 0 {
		return
	}
	price, _ := strconv.ParseFloat(item.AvgPrice, 64)
	if price <= 0 {
		price, _ = strconv.ParseFloat(item.Price, 64)
	}
	quantity, _ := strconv.ParseFloat(item.AccumulatedFilledQty, 64)
	if quantity <= 0 {
		quantity, _ = strconv.ParseFloat(item.OrigQuantity, 64)
	}
	value := agentevent.NewLiquidation(item.Symbol, "binance_ws_all_liquidation", item.EventTime,
		item.Side, price, quantity, item.Notional, liquidationSide)
	value.Meta.EventID = agentevent.StableID("liq", futuresLiquidationAlertEventKey(item))
	agentevent.DefaultBus().Publish(value)
}

func CleanupOldFuturesLiquidationOrders(keepDays int) (int64, error) {
	if keepDays <= 0 {
		keepDays = futuresLiquidationOrderKeepDays
	}

	cutoff := time.Now().AddDate(0, 0, -keepDays).UnixMilli()
	o := orm.NewOrm()
	var totalDeleted int64

	for {
		var orders []models.FuturesLiquidationOrder
		_, err := o.QueryTable(new(models.FuturesLiquidationOrder)).
			Filter("event_time__lt", cutoff).
			OrderBy("id").
			Limit(futuresLiquidationOrderCleanupBatchSize).
			All(&orders, "ID")
		if err != nil {
			return totalDeleted, err
		}
		if len(orders) == 0 {
			return totalDeleted, nil
		}

		ids := make([]interface{}, len(orders))
		for i := range orders {
			ids[i] = orders[i].ID
		}

		deleted, err := o.QueryTable(new(models.FuturesLiquidationOrder)).Filter("id__in", ids...).Delete()
		totalDeleted += deleted
		if err != nil {
			return totalDeleted, err
		}
	}
}

func cleanupOldFuturesLiquidationOrdersWithRetry(keepDays int) (int64, error) {
	return retryFuturesLiquidationOrderCleanup(CleanupOldFuturesLiquidationOrders, keepDays, time.Sleep)
}

func retryFuturesLiquidationOrderCleanup(cleanup func(int) (int64, error), keepDays int, wait func(time.Duration)) (int64, error) {
	var totalDeleted int64
	for attempt := 1; ; attempt++ {
		deleted, err := cleanup(keepDays)
		totalDeleted += deleted
		if err == nil {
			return totalDeleted, nil
		}
		if !errors.Is(err, driver.ErrBadConn) || attempt == futuresLiquidationOrderCleanupMaxRetries {
			return totalDeleted, err
		}
		wait(futuresLiquidationOrderCleanupRetryDelay)
	}
}

func StartFuturesLiquidationOrderCleanupTask() {
	logs.Debug("futures liquidation order cleanup task start, keep days:", futuresLiquidationOrderKeepDays)

	deleted, err := cleanupOldFuturesLiquidationOrdersWithRetry(futuresLiquidationOrderKeepDays)
	if err != nil {
		logs.Error("futures liquidation order cleanup task init error:", err)
	} else {
		logs.Debug("futures liquidation order cleanup task init done, deleted:", deleted)
	}

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		deleted, err := cleanupOldFuturesLiquidationOrdersWithRetry(futuresLiquidationOrderKeepDays)
		if err != nil {
			logs.Error("futures liquidation order cleanup task error:", err)
			continue
		}
		logs.Debug("futures liquidation order cleanup task done, deleted:", deleted)
	}
}

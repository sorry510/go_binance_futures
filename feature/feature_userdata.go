package feature

import (
	"context"
	"math"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"go_binance_futures/feature/api/binance"
	"go_binance_futures/models"
	"go_binance_futures/service/binanceapiusage"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

const futuresUserDataFullSyncMaxAge = 35 * time.Minute

var (
	futuresUserDataLastFullSyncAt         atomic.Int64
	futuresUserDataLastFullSyncGeneration atomic.Uint64
	futuresUserDataFullSyncMu             sync.Mutex
)

func SyncUserData() {
	deleteOldUserData()
	go func() {
		binance.WsUserData()
	}()

	// A User Data reconnect can miss events while the socket is down. Require
	// one authoritative full snapshot for each new WS generation before the
	// local mirror is allowed to replace REST account reads.
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for range ticker.C {
			generation := binance.FuturesUserDataWSGeneration()
			if generation == 0 || futuresUserDataLastFullSyncGeneration.Load() == generation {
				continue
			}
			getNowUserData()
		}
	}()

	go func() {
		for {
			time.Sleep(30 * time.Minute)
			getNowUserData()
		}
	}()
}

func futuresUserDataMirrorUsable() bool {
	generation := binance.FuturesUserDataWSGeneration()
	if generation == 0 || futuresUserDataLastFullSyncGeneration.Load() != generation {
		return false
	}
	lastSync := futuresUserDataLastFullSyncAt.Load()
	if lastSync <= 0 {
		return false
	}
	return time.Since(time.UnixMilli(lastSync)) <= futuresUserDataFullSyncMaxAge
}

// 删除数据表旧数据
func deleteOldUserData() {
	o := orm.NewOrm()
	o.Raw("DELETE FROM futures_orders where 1=1 and (status = 'NEW' or status = 'PARTIALLY_FILLED')").Exec() // 只删除未成交的订单
	o.Raw("DELETE FROM futures_positions where 1=1").Exec()
}

// 查询 api 接口获取最新数据。
// Full sync deliberately bypasses the short account-read cache so a healthy
// User Data WS mirror always starts from a fresh Binance account snapshot.
func getNowUserData() bool {
	futuresUserDataFullSyncMu.Lock()
	defer futuresUserDataFullSyncMu.Unlock()

	generation := binance.FuturesUserDataWSGeneration()
	o := orm.NewOrm()
	nowTime := time.Now().UnixMilli()
	ctx := binanceapiusage.WithSource(context.Background(), "user_data_sync")
	// Fetch both authoritative snapshots first. Do not mark/clean the local
	// mirror unless both reads succeed. Fresh variants bypass the short V4-5
	// account cache and invalidate older in-flight cache generations.
	allPositions, err := binance.GetPositionFreshContext(ctx, binance.PositionParams{})
	if err != nil {
		logs.Error("GetPosition err in feature_userdata:", err.Error())
		return false
	}
	allOpenOrders, err := binance.GetOpenOrderFreshContext(ctx)
	if err != nil {
		logs.Error("GetOpenOrder err in feature_userdata:", err.Error())
		return false
	}

	for _, position := range allPositions {
		positionAmt, _ := strconv.ParseFloat(position.PositionAmt, 64)
		positionAmtFloatAbs := math.Abs(positionAmt) // 空单为负数,纠正为绝对值
		if positionAmtFloatAbs < 0.0000001 {
			continue
		}
		leverage, _ := strconv.ParseInt(position.Leverage, 10, 64)

		var positionModel models.FuturesPosition
		o.QueryTable("futures_positions").Filter("symbol", position.Symbol).Filter("side", position.PositionSide).One(&positionModel)
		positionModel.Symbol = position.Symbol
		positionModel.Side = position.PositionSide
		positionModel.Amount = position.PositionAmt
		positionModel.Leverage = leverage
		positionModel.MarginType = position.MarginType
		positionModel.IsolatedWallet = position.IsolatedWallet
		positionModel.EntryPrice = position.EntryPrice
		positionModel.MarkPrice = position.MarkPrice
		positionModel.UnrealizedProfit = position.UnRealizedProfit
		positionModel.AccumulatedRealized = "0"
		positionModel.MaintenanceMarginRequired = "0"
		positionModel.CreateTime = nowTime
		positionModel.UpdateTime = nowTime
		if positionModel.ID == 0 {
			o.Insert(&positionModel)
		} else {
			o.Update(&positionModel)
		}
	}

	// open orders
	for _, order := range allOpenOrders {
		var orderModel models.FuturesOrder
		o.QueryTable("futures_orders").Filter("order_id", order.OrderID).One(&orderModel)
		orderModel.Symbol = order.Symbol
		orderModel.ClientOrderId = order.ClientOrderID
		orderModel.OrderId = strconv.FormatInt(order.OrderID, 10)
		orderModel.Side = string(order.Side)
		orderModel.PositionSide = string(order.PositionSide)
		orderModel.Type = string(order.Type)
		orderModel.Status = string(order.Status) // NEW
		orderModel.Price = string(order.Price)
		orderModel.OrigQty = order.OrigQuantity
		orderModel.ExecutedQty = order.ExecutedQuantity
		orderModel.CreateTime = nowTime
		orderModel.UpdateTime = nowTime
		if orderModel.ID == 0 {
			o.Insert(&orderModel)
		} else {
			o.Update(&orderModel)
		}
	}

	// Remove active mirror rows that were not present in the successful REST
	// snapshot. Rows updated by a concurrent WS event have updateTime >= nowTime
	// and are deliberately preserved.
	if _, err := o.Raw("DELETE FROM futures_positions WHERE updateTime < ?", nowTime).Exec(); err != nil {
		logs.Error("cleanup stale futures_positions after full sync:", err)
		return false
	}
	if _, err := o.Raw("DELETE FROM futures_orders WHERE (status = 'NEW' OR status = 'PARTIALLY_FILLED') AND updateTime < ?", nowTime).Exec(); err != nil {
		logs.Error("cleanup stale futures_orders after full sync:", err)
		return false
	}

	// If the socket reconnected while the REST snapshot was loading, this
	// snapshot belongs to the previous generation and must not authorize the
	// new mirror. The watcher will immediately request another full sync.
	if generation != 0 && binance.FuturesUserDataWSGeneration() == generation {
		futuresUserDataLastFullSyncAt.Store(time.Now().UnixMilli())
		futuresUserDataLastFullSyncGeneration.Store(generation)
	}
	return true
}

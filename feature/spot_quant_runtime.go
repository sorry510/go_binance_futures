package feature

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"go_binance_futures/models"
	"go_binance_futures/service/spotexecution"
	"go_binance_futures/service/spotledger"
	spotbinance "go_binance_futures/spot/api/binance"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/config"
	"github.com/beego/beego/v2/core/logs"
)

// Spot Quant has one runtime for both environments. Only config's existing
// binance.testnet selects the account; both ordinary DB switches authorize
// new BUY entries. Verified existing Managed positions may still SELL.
type spotQuantLiveExecutor struct {
	processor *spotexecution.Processor
	wsReady   *atomic.Bool
}

func (e spotQuantLiveExecutor) ExecuteBuy(ctx context.Context, symbol, amount, id string) (models.SpotManagedOrder, error) {
	if e.processor == nil || e.wsReady == nil || !e.wsReady.Load() {
		return models.SpotManagedOrder{}, spotexecution.ErrDisabled
	}
	return e.processor.ExecuteBuy(ctx, symbol, amount, id)
}
func (e spotQuantLiveExecutor) ExecuteSell(ctx context.Context, positionID int64, id string) (models.SpotManagedOrder, error) {
	if e.processor == nil || e.wsReady == nil || !e.wsReady.Load() {
		return models.SpotManagedOrder{}, spotexecution.ErrDisabled
	}
	return e.processor.ExecuteSell(ctx, positionID, id)
}
func (e spotQuantLiveExecutor) Reconcile(ctx context.Context, orderID int64) (models.SpotManagedOrder, error) {
	// Existing orders may still be reconciled over REST when WS disconnects.
	if e.processor == nil {
		return models.SpotManagedOrder{}, spotexecution.ErrDisabled
	}
	return e.processor.Reconcile(ctx, orderID)
}

type spotQuantRuntime struct {
	runner    *SpotQuantRunner
	exchange  *spotbinance.QuantAdapter
	processor *spotexecution.Processor
	ready     atomic.Bool
	needsWork func() (bool, error)
}

const spotQuantRuntimeIdlePoll = 15 * time.Second

func newSpotQuantRuntime(db orm.Ormer, dialect string, testnet bool) (*spotQuantRuntime, error) {
	if db == nil {
		return nil, fmt.Errorf("Spot Quant runtime requires registered DB")
	}
	exchange, err := spotbinance.NewSpotQuantTradingAdapterFromConfig(testnet)
	if err != nil {
		return nil, err
	}
	ledger, err := spotledger.New(db, dialect)
	if err != nil {
		return nil, err
	}
	strategy, err := SpotMarketStrategy(exchange)
	if err != nil {
		return nil, err
	}
	processor, err := spotexecution.NewProcessor(exchange, spotexecution.DBGate{DB: db}, ledger)
	if err != nil {
		return nil, err
	}
	runtime := &spotQuantRuntime{
		exchange: exchange, processor: processor,
		// The WS watcher runs concurrently with strategy execution. Do not
		// reuse the Runner's Ormer across goroutines.
		needsWork: func() (bool, error) { return spotQuantNeedsRuntime(orm.NewOrm()) },
	}
	runner, err := NewSpotQuantRunner(db, ledger, strategy, spotQuantLiveExecutor{
		processor: processor, wsReady: &runtime.ready,
	})
	if err != nil {
		return nil, err
	}
	// Different ClientOrderID prefixes prevent replay of testnet orders as
	// mainnet positions (and vice versa), without extra tables or switches.
	runner.orderPrefix = "sqm"
	if testnet {
		runner.orderPrefix = "sqt"
	}
	runner.spot3dOpen = SpotMarketThreeDayOpen(exchange)
	runtime.runner = runner
	return runtime, nil
}

// Only keep the authenticated user stream connected when there is new-entry
// or managed recovery work. After the last Managed position/order closes and
// both entry switches are OFF, pause the WS/REST time-sync loop; resume when
// the existing config/managed state needs it again. No extra control flags.
func (r *spotQuantRuntime) streamLoop(ctx context.Context) {
	for ctx.Err() == nil {
		if r.needsWork != nil {
			needed, err := r.needsWork()
			if err != nil {
				r.ready.Store(false)
				logs.Warning("spot quant runtime work check failed: ", spotQuantSafeMessage(err.Error()))
			}
			if err != nil || !needed {
				r.ready.Store(false)
				select {
				case <-ctx.Done():
					return
				case <-time.After(spotQuantRuntimeIdlePoll):
				}
				continue
			}
		}
		// The signed WS is a long-lived blocking call. Monitor whether the
		// system becomes idle and cancel its context so it closes cleanly.
		streamCtx, cancel := context.WithCancel(ctx)
		watchDone := make(chan struct{})
		go func() {
			defer close(watchDone)
			if r.needsWork == nil {
				<-streamCtx.Done()
				return
			}
			ticker := time.NewTicker(spotQuantRuntimeIdlePoll)
			defer ticker.Stop()
			for {
				select {
				case <-streamCtx.Done():
					return
				case <-ticker.C:
					needed, err := r.needsWork()
					if err != nil {
						logs.Warning("spot quant runtime work check failed: ", spotQuantSafeMessage(err.Error()))
						cancel() // fail closed if DB work state is unknown
						return
					}
					if !needed {
						cancel()
						return
					}
				}
			}
		}()
		timeCtx, stopTime := context.WithTimeout(streamCtx, 12*time.Second)
		err := r.exchange.SyncSpotMarketServerTime(timeCtx)
		stopTime()
		if err == nil && streamCtx.Err() == nil {
			err = r.exchange.RunSpotQuantUserStreamWithStatus(streamCtx, r.processor, func(ready bool) {
				r.ready.Store(ready && streamCtx.Err() == nil)
			})
		}
		r.ready.Store(false)
		cancel()
		<-watchDone
		if ctx.Err() != nil {
			return
		}
		if r.needsWork != nil {
			needed, stateErr := r.needsWork()
			if stateErr == nil && !needed {
				continue // idle: no unnecessary reconnect or serverTime REST
			}
		}
		logs.Warning("spot quant signed user WS unavailable; orders paused: ", spotQuantSafeMessage(fmt.Sprint(err)))
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
		}
	}
}

var spotQuantRuntimeState struct {
	sync.Once
	instance *spotQuantRuntime
	err      error
}

// The two existing business switches control new entries; an OFF global
// flag still initializes the live adapter when Managed exits or unsettled
// orders need processing. No new flags or environment variables.
// Do not initialize signed Spot WS when there is no new entry or managed
// recovery work. Mutable status alone cannot hide nonzero inventory.
func spotQuantNeedsRuntime(db orm.Ormer) (bool, error) {
	var enabled int64
	if err := db.Raw("SELECT spot_quant_enable FROM config WHERE id=1").QueryRow(&enabled); err != nil {
		return false, err
	}
	if enabled == 1 {
		var symbols int64
		if err := db.Raw("SELECT COUNT(*) FROM spot_symbols WHERE spot_quant_enabled=1").QueryRow(&symbols); err != nil {
			return false, err
		}
		if symbols > 0 {
			return true, nil
		}
	}
	var managed int64
	if err := db.Raw("SELECT COUNT(*) FROM spot_managed_positions WHERE status IN ('OPEN','PENDING_BUY') OR remaining_qty > 0").QueryRow(&managed); err != nil {
		return false, err
	}
	if managed > 0 {
		return true, nil
	}
	if err := db.Raw("SELECT COUNT(*) FROM spot_managed_orders WHERE status IN ('NEW','PARTIALLY_FILLED','PENDING_CANCEL') OR (status='FILLED' AND filled_qty <> requested_qty)").QueryRow(&managed); err != nil {
		return false, err
	}
	return managed > 0, nil
}

func spotQuantRunLive(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// Beego NewOrm panics if the default database was never registered.
	// Fail closed in CLI/test startup instead of relying on panic recovery.
	if _, err := orm.GetDB("default"); err != nil {
		return fmt.Errorf("Spot Quant default database unavailable: %w", err)
	}
	db := orm.NewOrm()
	needed, err := spotQuantNeedsRuntime(db)
	if err != nil || !needed {
		return err
	}
	spotQuantRuntimeState.Do(func() {
		testnet, err := config.Bool("binance::testnet")
		if err != nil {
			spotQuantRuntimeState.err = err
			return
		}
		driver, err := config.String("database::driver")
		if err != nil {
			spotQuantRuntimeState.err = err
			return
		}
		instance, err := newSpotQuantRuntime(db, driver, testnet)
		if err != nil {
			spotQuantRuntimeState.err = err
			return
		}
		spotQuantRuntimeState.instance = instance
		go instance.streamLoop(context.Background())
		logs.Info("spot quant runner started: testnet=", testnet)
	})
	if spotQuantRuntimeState.err != nil {
		return spotQuantRuntimeState.err
	}
	if spotQuantRuntimeState.instance == nil {
		return fmt.Errorf("Spot Quant runtime unavailable")
	}
	return spotQuantRuntimeState.instance.runner.Run(ctx)
}

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
// every new BUY/SELL through the existing DBGate and Processor.
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
}

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
	runtime := &spotQuantRuntime{exchange: exchange, processor: processor}
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

func (r *spotQuantRuntime) streamLoop(ctx context.Context) {
	for ctx.Err() == nil {
		timeCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
		err := r.exchange.SyncSpotMarketServerTime(timeCtx)
		cancel()
		if err == nil {
			err = r.exchange.RunSpotQuantUserStreamWithStatus(ctx, r.processor, func(ready bool) {
				r.ready.Store(ready)
			})
		}
		r.ready.Store(false)
		if ctx.Err() != nil {
			return
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

// The two existing business switches are sufficient: an OFF global flag
// never initializes the live adapter. No special environment variables and
// no hardcoded symbol/amount limits. The user's configured budget still
// goes through the exchange's quantity/available-balance validation.
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
	var enabled int64
	if err := db.Raw("SELECT spot_quant_enable FROM config WHERE id=1").QueryRow(&enabled); err != nil {
		return err
	}
	if enabled != 1 {
		return nil
	}
	var symbols int64
	if err := db.Raw("SELECT COUNT(*) FROM spot_symbols WHERE spot_quant_enabled=1").QueryRow(&symbols); err != nil {
		return err
	}
	if symbols == 0 {
		return nil
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

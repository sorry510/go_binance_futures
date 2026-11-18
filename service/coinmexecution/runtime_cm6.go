package coinmexecution

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"go_binance_futures/feature/api/binance"

	"github.com/adshao/go-binance/v2/delivery"
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

const coinMIdleCheckPeriod = 15 * time.Second

type coinMRuntime struct {
	runner     *CoinMRunner
	processor  *Processor
	gateway    *BinanceGateway
	stream     *UserStream
	events     chan *delivery.WsUserDataEvent
	processing sync.Mutex // serializes WS event drain + Runner Tick on one Ormer
}

func newCoinMRuntime(db orm.Ormer) (*coinMRuntime, error) {
	if db == nil {
		return nil, errors.New("COIN-M runtime requires default DB")
	}
	runtime := &coinMRuntime{events: make(chan *delivery.WsUserDataEvent, 256)}
	// Private WS receives exchange events concurrently with a two-second
	// scheduler. Queue them; only the Runner goroutine writes with its Ormer.
	runtime.stream = NewUserStream(func(ev *delivery.WsUserDataEvent) error {
		if ev == nil {
			return nil
		}
		copyEvent := *ev
		select {
		case runtime.events <- &copyEvent:
			return nil
		default:
			return errors.New("COIN-M private WS event queue overflow; orders paused until reconciliation")
		}
	})
	runtime.gateway = NewBinanceGateway(runtime.stream)
	runtime.processor = NewProcessor(db, runtime.gateway)
	var err error
	runtime.runner, err = NewCoinMRunner(db, runtime.processor, newCoinMStrategy())
	if err != nil {
		return nil, err
	}
	return runtime, nil
}
func (r *coinMRuntime) drain(ctx context.Context) error {
	// A bounded buffer avoids unbounded memory use under disconnection
	// and ensures at most 256 events are processed each tick.
	for n := 0; n < cap(r.events); n++ {
		select {
		case ev := <-r.events:
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := r.processor.OnWSOrder(ev); err != nil {
				return fmt.Errorf("COIN-M private WS order event requires REST reconciliation: %w", err)
			}
		default:
			return nil
		}
	}
	return nil
}
func (r *coinMRuntime) Tick(ctx context.Context) error {
	if r == nil || r.runner == nil {
		return errors.New("COIN-M runtime unavailable")
	}
	// The existing 2s tick may fire while a slow DAPI reconciliation is
	// still running. Skip that tick instead of accumulating goroutines
	// waiting on one shared ORM session.
	if !r.processing.TryLock() {
		return nil
	}
	defer r.processing.Unlock()
	if err := r.drain(ctx); err != nil {
		return err
	}
	return r.runner.Tick(ctx)
}
func (r *coinMRuntime) streamLoop(ctx context.Context) {
	for ctx.Err() == nil {
		needed, err := CoinMNeedsRuntime(orm.NewOrm())
		if err != nil {
			logs.Warning("coin-m stream work state unavailable: ", coinMSafeLog(err))
		}
		if err != nil || !needed {
			select {
			case <-ctx.Done():
				return
			case <-time.After(coinMIdleCheckPeriod):
			}
			continue
		}
		session, cancel := context.WithCancel(ctx)
		watcher := make(chan struct{})
		go func() {
			defer close(watcher)
			ticker := time.NewTicker(coinMIdleCheckPeriod)
			defer ticker.Stop()
			for {
				select {
				case <-session.Done():
					return
				case <-ticker.C:
					required, workErr := CoinMNeedsRuntime(orm.NewOrm())
					if workErr != nil || !required {
						cancel()
						return
					}
				}
			}
		}()
		err = r.stream.Run(session)
		cancel()
		<-watcher
		if ctx.Err() != nil {
			return
		}
		if err != nil && !errors.Is(err, context.Canceled) {
			logs.Warning("coin-m private WS stopped: ", coinMSafeLog(err))
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
		}
	}
}

var coinMRuntimeState struct {
	sync.Once
	instance *coinMRuntime
	err      error
}

func StartCoinMTrade(ctx context.Context) error {
	if ctx == nil {
		return errors.New("missing COIN-M runtime context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := orm.GetDB("default"); err != nil {
		return fmt.Errorf("COIN-M default database unavailable: %w", err)
	}
	needed, err := CoinMNeedsRuntime(orm.NewOrm())
	if err != nil {
		return err
	}
	if !needed {
		return nil
	} // no signed account requests when OFF and empty
	if binance.CoinMEnvironment() == "" {
		return errors.New("COIN-M account environment is unconfigured")
	}
	coinMRuntimeState.Do(func() {
		runtime, setupErr := newCoinMRuntime(orm.NewOrm())
		if setupErr != nil {
			coinMRuntimeState.err = setupErr
			return
		}
		coinMRuntimeState.instance = runtime
		go runtime.streamLoop(context.Background())
		logs.Info("coin-m shared scheduler initialized: environment=", binance.CoinMEnvironment())
	})
	if coinMRuntimeState.err != nil {
		return coinMRuntimeState.err
	}
	if coinMRuntimeState.instance == nil {
		return errors.New("COIN-M runtime failed to initialize")
	}
	return coinMRuntimeState.instance.Tick(ctx)
}

var coinMLogSecret = regexp.MustCompile(`(?i)(signature|api[_-]?key|secret|listenkey)\s*[=:]\s*[^\s&]+\b`)

func coinMSafeLog(err error) string {
	if err == nil {
		return ""
	}
	msg := coinMLogSecret.ReplaceAllString(err.Error(), "$1=[redacted]")
	if len(msg) > 350 {
		msg = msg[:350] + "..."
	}
	return strings.ReplaceAll(msg, "\n", " ")
}

var coinMLogState struct {
	sync.Mutex
	value string
}

func ReportCoinMScheduler(err error) {
	coinMLogState.Lock()
	defer coinMLogState.Unlock()
	msg := coinMSafeLog(err)
	if msg == "" {
		coinMLogState.value = ""
		return
	}
	if msg != coinMLogState.value {
		coinMLogState.value = msg
		logs.Warning("coin-m scheduler: ", msg)
	}
}
func RunCoinMProtected(callback func()) {
	defer func() {
		if v := recover(); v != nil {
			logs.Warning("coin-m scheduler recovered panic: ", coinMSafeLog(fmt.Errorf("%v", v)))
		}
	}()
	callback()
}

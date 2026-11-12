package feature

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go_binance_futures/feature/strategy/line"
	"go_binance_futures/models"
	"go_binance_futures/service/spotexecution"
	"go_binance_futures/service/spotledger"
	"go_binance_futures/service/spotquant"
	"go_binance_futures/technology"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

// spotQuantExecutor describes the existing, separately verified 06-4 order
// processor. A nil executor means read-only / no automatic order submission.
// Never replace it with the old Spot Rush or Futures order functions.
type spotQuantExecutor interface {
	ExecuteBuy(context.Context, string, string, string) (models.SpotManagedOrder, error)
	ExecuteSell(context.Context, int64, string) (models.SpotManagedOrder, error)
	Reconcile(context.Context, int64) (models.SpotManagedOrder, error)
}

// The reviewed 06-4 Processor satisfies this interface without an adapter.
// This assertion does not initialize it or permit a real Spot order.
var _ spotQuantExecutor = (*spotexecution.Processor)(nil)

type spotQuantStrategy interface {
	EvaluateEntry(models.SpotSymbols) (line.SpotSignal, error)
	EvaluateExit(models.SpotSymbols, *spotledger.Ledger, int64) (line.SpotSignal, error)
}

// SpotQuantRunner follows the existing Futures two-second loop contract but
// uses only Spot symbols, the dedicated Spot Fill ledger and 06-4 processor.
// Running the scheduler without an explicitly attached executor is safe:
// business switches alone cannot authorize a real Binance order.
type SpotQuantRunner struct {
	db       orm.Ormer
	ledger   *spotledger.Ledger
	strategy spotQuantStrategy
	exec     spotQuantExecutor
	clock    func() time.Time
	running  atomic.Bool // only protects this instance; main.loopRun owns process-tick serialization
	errorMu  sync.Mutex
	lastErr  map[string]string
	warnings map[string]bool
}

func NewSpotQuantRunner(db orm.Ormer, ledger *spotledger.Ledger, strategy spotQuantStrategy, exec spotQuantExecutor) (*SpotQuantRunner, error) {
	if db == nil || ledger == nil || strategy == nil {
		return nil, fmt.Errorf("Spot Quant runner requires DB, Spot Fill ledger and Spot strategy")
	}
	return &SpotQuantRunner{db: db, ledger: ledger, strategy: strategy, exec: exec, clock: time.Now,
		lastErr: make(map[string]string), warnings: make(map[string]bool)}, nil
}

// Spot event identity uses the longest enabled indicator or explicitly
// referenced Kline interval. This conservative choice prevents repeated
// BUY/SELL cycles inside a 5m/1h candle even when rules are edited.
// Without a Kline dependency, use the project's 1m signal baseline.
var spotRuleKlineRef = regexp.MustCompile(`\bkline_(1m|3m|5m|15m|30m|1h|2h|4h|6h|8h|12h|1d|3d|1w|1M)\b`)

var spotCandleDuration = map[string]time.Duration{
	"1m": time.Minute, "3m": 3 * time.Minute, "5m": 5 * time.Minute,
	"15m": 15 * time.Minute, "30m": 30 * time.Minute,
	"1h": time.Hour, "2h": 2 * time.Hour, "4h": 4 * time.Hour,
	"6h": 6 * time.Hour, "8h": 8 * time.Hour, "12h": 12 * time.Hour,
	"1d": 24 * time.Hour, "3d": 72 * time.Hour,
	"1w": 7 * 24 * time.Hour, "1M": 31 * 24 * time.Hour,
}

func spotSignalInterval(coin models.SpotSymbols) (string, error) {
	interval := "1m"
	configText := strings.TrimSpace(coin.Technology)
	if configText != "" {
		var cfg technology.TechnologyConfig
		if err := json.Unmarshal([]byte(configText), &cfg); err != nil {
			return "", fmt.Errorf("invalid Spot indicator config: %w", err)
		}
		for _, group := range [][]technology.IndicatorConfig{
			cfg.MA, cfg.EMA, cfg.MACD, cfg.RSI, cfg.KC, cfg.BOLL,
			cfg.ATR, cfg.ADX, cfg.MFI, cfg.OBV, cfg.CCI, cfg.ROC,
			cfg.KDJ, cfg.Supertrend, cfg.Donchian,
		} {
			for _, item := range group {
				if !item.Enable {
					continue
				}
				duration, ok := spotCandleDuration[item.KlineInterval]
				if !ok {
					return "", fmt.Errorf("unsupported Spot indicator Kline interval %q", item.KlineInterval)
				}
				if duration > spotCandleDuration[interval] {
					interval = item.KlineInterval
				}
			}
		}
	}
	for _, match := range spotRuleKlineRef.FindAllStringSubmatch(coin.Strategy, -1) {
		if spotCandleDuration[match[1]] > spotCandleDuration[interval] {
			interval = match[1]
		}
	}
	return interval, nil
}

// Binance Spot Klines use UTC-aligned start times (weekly begins Monday;
// monthly begins UTC 1st). Return the period's OPEN, not a rolling minute ID.
func spotCandleOpen(t time.Time, interval string) time.Time {
	t = t.UTC()
	switch interval {
	case "1M":
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	case "1w":
		day := t.Truncate(24 * time.Hour)
		return day.Add(-time.Duration((int(day.Weekday())+6)%7) * 24 * time.Hour)
	default:
		duration := spotCandleDuration[interval]
		if duration <= 0 {
			return time.Time{}
		}
		return t.Truncate(duration)
	}
}

// spotQuantClientID includes a canonical Spot Kline OPEN timestamp rather
// than the polling minute. Hash is independent of editable strategy rules.
// This function accepts a candle event time already aligned by caller.
func spotQuantClientID(symbol, side string, positionID int64, eventOpen time.Time) string {
	identity := symbol + "|" + side + "|" + strconv.FormatInt(positionID, 10) +
		"|" + strconv.FormatInt(eventOpen.UTC().Unix(), 10)
	hash := sha256.Sum256([]byte("spot_quant_0672_v2|" + identity))
	return "sq" + hex.EncodeToString(hash[:16]) // 34 characters
}

func (r *SpotQuantRunner) logSymbolError(symbol string, err error) {
	r.errorMu.Lock()
	defer r.errorMu.Unlock()
	if err == nil {
		if _, ok := r.lastErr[symbol]; ok {
			delete(r.lastErr, symbol)
			logs.Info("spot quant: ", symbol, " recovered")
		}
		return
	}
	msg := err.Error()
	if r.lastErr[symbol] != msg {
		r.lastErr[symbol] = msg
		logs.Warning("spot quant: ", symbol, " skip: ", err)
	}
}

func (r *SpotQuantRunner) globalEnabled(ctx context.Context) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	var on int64
	if err := r.db.Raw("SELECT spot_quant_enable FROM config WHERE id=1").QueryRow(&on); err != nil {
		return false, fmt.Errorf("Spot Quant global switch: %w", err)
	}
	return on == 1, nil
}

// Run is a single bounded, non-overlapping cycle; main.go reuses loopRun at
// the same 2s cadence as Futures. Each enabled symbol is processed serially.
func (r *SpotQuantRunner) Run(ctx context.Context) error {
	if r == nil {
		return fmt.Errorf("nil Spot Quant runner")
	}
	if !r.running.CompareAndSwap(false, true) {
		return nil // prior run still active, never enqueue an overlapping cycle
	}
	defer r.running.Store(false)
	if r.exec == nil {
		// Stage 06-7-2 runtime wiring is intentionally non-mutating until
		// isolated Spot Testnet credentials and the real-order Gate pass.
		// Do not even fetch live market history on a disabled order route.
		return nil
	}
	on, err := r.globalEnabled(ctx)
	if err != nil || !on {
		return err
	}
	var problems []error
	for page := 1; ; page++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		symbols, err := spotquant.ListSymbols(r.db, spotquant.SymbolFilters{
			Enabled: "1", Page: page, Limit: 100, Sort: "id",
		})
		if err != nil {
			return fmt.Errorf("Spot Quant enabled symbols: %w", err)
		}
		for _, coin := range symbols.List {
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := r.runSymbol(ctx, coin); err != nil {
				problems = append(problems, fmt.Errorf("%s: %w", coin.Symbol, err))
				r.logSymbolError(coin.Symbol, err)
			} else {
				r.logSymbolError(coin.Symbol, nil)
			}
		}
		if page*100 >= int(symbols.Total) {
			break
		}
	}
	return errors.Join(problems...)
}

// activeSpotPosition projects EVERY existing cycle for the symbol using
// immutable Fill evidence, not the editable Position.status cache. Personal
// single-account volume is small; favor correctness over a mutable-status
// shortcut. A corrupted CLOSED cache cannot hide a pending BUY.
func (r *SpotQuantRunner) activeSpotPosition(symbol string) (models.SpotManagedPosition, bool, error) {
	var active models.SpotManagedPosition
	found := false
	for page := 1; ; page++ {
		view, err := r.ledger.ListPositions(spotledger.PositionFilter{
			Symbol: symbol, Page: page, Limit: 100,
		})
		if err != nil {
			return active, false, err
		}
		for _, stored := range view.List {
			verified, err := r.ledger.Accounting(stored.ID, "")
			if err != nil {
				return active, false, err
			}
			if verified.Position.Status != "OPEN" && verified.Position.Status != "PENDING_BUY" {
				continue
			}
			if found {
				return active, false, fmt.Errorf("%w: multiple verified Spot cycles", spotledger.ErrNeedsReview)
			}
			active = verified.Position
			found = true
		}
		if page*100 >= int(view.Total) {
			break
		}
	}
	return active, found, nil
}

func (r *SpotQuantRunner) reconcileUnsettled(ctx context.Context, symbol string) (bool, error) {
	pending, err := r.ledger.ListUnsettledOrders(symbol)
	if err != nil {
		return false, err
	}
	if len(pending) == 0 {
		return false, nil
	}
	var issues []error
	for _, o := range pending {
		if ctx.Err() != nil {
			return true, ctx.Err()
		}
		if _, err := r.exec.Reconcile(ctx, o.ID); err != nil {
			issues = append(issues, fmt.Errorf("%s/%s: %w", o.Symbol, o.ClientOrderID, err))
		}
	}
	// After ANY reconciliation wait until the next loop to reproject
	// all Fill evidence; never BUY/SELL within the recovery cycle.
	return true, errors.Join(issues...)
}

func (r *SpotQuantRunner) runSymbol(ctx context.Context, coin models.SpotSymbols) error {
	if coin.SpotQuantEnabled != 1 || coin.Symbol == "" {
		return nil
	}
	gate := spotexecution.DBGate{DB: r.db}
	on, err := gate.Enabled(ctx, coin.Symbol)
	if err != nil || !on {
		return err
	}

	// Review every order still unsettled, regardless of the mutable
	// parent Position.status, including orphan/closed-cache rows.
	recovered, err := r.reconcileUnsettled(ctx, coin.Symbol)
	if err != nil || recovered {
		return err
	}
	if coin.Profit == "100" && coin.Loss == "100" && !r.warnings[coin.Symbol] {
		r.warnings[coin.Symbol] = true
		logs.Warning("spot quant: ", coin.Symbol,
			" profit=100/loss=100 means +100%/-100% ROI; confirm before automatic trading")
	}
	active, exists, err := r.activeSpotPosition(coin.Symbol)
	if err != nil {
		return err
	}
	if exists {
		orders, err := r.ledger.ListOrders(active.ID)
		if err != nil {
			return err
		}
		if len(orders) == 0 {
			return fmt.Errorf("%w: active Spot cycle has no verified BUY order", spotledger.ErrNeedsReview)
		}
		// Reconcile all NEW/PARTIALLY_FILLED/PENDING_CANCEL orders
		// above before any decision; current projection is Fill-sourced.
		if active.Status != "OPEN" {
			return nil
		}
		signal, err := r.strategy.EvaluateExit(coin, r.ledger, active.ID)
		if err != nil || signal.Action != "SELL" {
			return err
		}
		interval, err := spotSignalInterval(coin)
		if err != nil {
			return err
		}
		id := spotQuantClientID(coin.Symbol, "SELL", active.ID, spotCandleOpen(r.clock(), interval))
		prior, err := r.ledger.FindOrderByClientID(id)
		if err == nil {
			if prior.Symbol != coin.Symbol || prior.Side != "SELL" || prior.PositionID != active.ID {
				return fmt.Errorf("%w: Spot SELL client id collision", spotledger.ErrNeedsReview)
			}
			return nil // already submitted in this Spot candle, even if FILLED
		}
		if !errors.Is(err, orm.ErrNoRows) {
			return err
		}
		on, err = gate.Enabled(ctx, coin.Symbol)
		if err != nil || !on {
			return err
		}
		_, err = r.exec.ExecuteSell(ctx, active.ID, id)
		return err
	}

	signal, err := r.strategy.EvaluateEntry(coin)
	if err != nil || signal.Action != "BUY" {
		return err
	}
	interval, err := spotSignalInterval(coin)
	if err != nil {
		return err
	}
	id := spotQuantClientID(coin.Symbol, "BUY", 0, spotCandleOpen(r.clock(), interval))
	prior, err := r.ledger.FindOrderByClientID(id)
	if err == nil {
		if prior.Symbol != coin.Symbol || prior.Side != "BUY" {
			return fmt.Errorf("%w: Spot BUY client id collision", spotledger.ErrNeedsReview)
		}
		return nil // do not BUY twice in the same Spot candle
	}
	if !errors.Is(err, orm.ErrNoRows) {
		return err
	}
	on, err = gate.Enabled(ctx, coin.Symbol)
	if err != nil || !on {
		return err
	}
	_, err = r.exec.ExecuteBuy(ctx, coin.Symbol, coin.Usdt, id)
	return err
}

// StartSpotQuantTrade is the 06-7-2 main.go scheduling entry.
// Production has no authorized Exchange mutation executor. Return BEFORE
// any config/ORM access so even an unregistered default DB cannot panic a
// Futures worker. The full Runner is exercised with injected Fake executors.
// Real Testnet/Mainnet injection needs a separate reviewed release gate.
func StartSpotQuantTrade(ctx context.Context) error {
	return ctx.Err()
}

// ReportSpotQuantScheduler suppresses duplicate scheduler errors across the
// process rather than logging the same failure on every two-second tick.
var spotQuantSchedulerLogMu sync.Mutex
var spotQuantSchedulerLogState string

func ReportSpotQuantScheduler(err error) {
	spotQuantSchedulerLogMu.Lock()
	defer spotQuantSchedulerLogMu.Unlock()
	if err == nil {
		spotQuantSchedulerLogState = ""
		return
	}
	if msg := err.Error(); spotQuantSchedulerLogState != msg {
		spotQuantSchedulerLogState = msg
		logs.Warning("spot quant scheduler: ", err)
	}
}

// RunSpotQuantProtected isolates a Spot callback panic from the existing
// Futures/legacy Spot loopRun goroutines, without altering their behavior.
func RunSpotQuantProtected(callback func()) {
	defer func() {
		if recovered := recover(); recovered != nil {
			logs.Warning("spot quant scheduler recovered panic: ", recovered)
		}
	}()
	callback()
}

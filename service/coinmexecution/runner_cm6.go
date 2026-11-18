package coinmexecution

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"sync/atomic"
	"time"

	"go_binance_futures/feature/api/binance"
	"go_binance_futures/models"
	"go_binance_futures/service/coinmquant"

	"github.com/beego/beego/v2/client/orm"
)

const (
	coinMTickSymbolMinimum   = 10 * time.Second
	coinMTickOrderMinimum    = 8 * time.Second
	coinMProtectAuditMinimum = 15 * time.Second
)

type CoinMRunner struct {
	db          orm.Ormer
	processor   *Processor
	strategy    CoinMSignalStrategy
	clock       func() time.Time
	eventOpen   func(context.Context, string, string, time.Time) (time.Time, error)
	environment func() string
	running     atomic.Bool
	marketAt    map[string]time.Time
	orderAt     map[string]time.Time
	cursor      int
}

func NewCoinMRunner(db orm.Ormer, p *Processor, strat CoinMSignalStrategy) (*CoinMRunner, error) {
	if db == nil || p == nil || p.API == nil || p.DB == nil || strat == nil {
		return nil, errors.New("COIN-M runner needs isolated DB, execution processor and strategy")
	}
	r := &CoinMRunner{db: db, processor: p, strategy: strat, clock: time.Now, eventOpen: coinMEventOpen, environment: binance.CoinMEnvironment,
		marketAt: map[string]time.Time{}, orderAt: map[string]time.Time{}}
	return r, nil
}
func coinMEntryEnabled(db orm.Ormer) (bool, error) {
	var n int64
	if err := db.Raw("SELECT coin_m_quant_enable FROM config LIMIT 1").QueryRow(&n); err != nil {
		return false, err
	}
	if n != 0 && n != 1 {
		return false, errors.New("invalid COIN-M global switch")
	}
	return n == 1, nil
}
func CoinMNeedsRuntime(db orm.Ormer) (bool, error) {
	if db == nil {
		return false, errors.New("COIN-M database unavailable")
	}
	global, err := coinMEntryEnabled(db)
	if err != nil {
		return false, err
	}
	if global {
		var enabled int64
		if err = db.Raw("SELECT COUNT(*) FROM delivery_symbols WHERE enable=1 AND contract_type='PERPETUAL'").QueryRow(&enabled); err != nil {
			return false, err
		}
		if enabled > 0 {
			return true, nil
		}
	}
	var rows int64
	if err = db.Raw("SELECT COUNT(*) FROM coin_m_managed_positions WHERE remaining_contracts>0 OR status NOT IN ('CLOSED','CANCELED')").QueryRow(&rows); err != nil {
		return false, err
	}
	if rows > 0 {
		return true, nil
	}
	if err = db.Raw("SELECT COUNT(*) FROM coin_m_managed_orders WHERE status IN ('PENDING_SUBMIT','UNKNOWN','NEW','PARTIALLY_FILLED','ACTIVE','AWAITING_FILLS','TRIGGERED','PENDING_CANCEL') OR (intent!='PROTECT' AND status='FILLED' AND filled_contracts > (SELECT COALESCE(SUM(contracts),0) FROM coin_m_trade_fills WHERE order_id=coin_m_managed_orders.id))").QueryRow(&rows); err != nil {
		return false, err
	}
	return rows > 0, nil
}

var coinMOrderColumns = "id,position_id,symbol,position_side,intent,side,order_type,client_order_id,COALESCE(exchange_order_id,'') AS exchange_order_id,requested_contracts,filled_contracts,status,created_at,updated_at"

func (r *CoinMRunner) unsettled() ([]models.CoinMManagedOrder, error) {
	list := make([]models.CoinMManagedOrder, 0)
	_, err := r.db.Raw("SELECT " + coinMOrderColumns + " FROM coin_m_managed_orders WHERE status IN ('PENDING_SUBMIT','UNKNOWN','NEW','PARTIALLY_FILLED','ACTIVE','AWAITING_FILLS','TRIGGERED','PENDING_CANCEL') OR (intent!='PROTECT' AND status='FILLED' AND filled_contracts > (SELECT COALESCE(SUM(contracts),0) FROM coin_m_trade_fills WHERE order_id=coin_m_managed_orders.id)) ORDER BY id LIMIT 101").QueryRows(&list)
	if err != nil {
		return nil, err
	}
	if len(list) > 100 {
		// Never silently ignore a pending 101st exchange order and infer
		// permission to open/close from a truncated local order projection.
		return nil, errors.New("COIN-M unsettled order scan exceeds 100; cannot authorize new execution")
	}
	return list, nil
}
func (r *CoinMRunner) managed() ([]models.CoinMManagedPosition, error) {
	rows := make([]models.CoinMManagedPosition, 0)
	_, err := r.db.Raw("SELECT id,symbol,position_side,margin_asset,contract_size,entry_price,opened_contracts,remaining_contracts,realized_pnl,status,created_at,updated_at,closed_at FROM coin_m_managed_positions WHERE remaining_contracts>0 OR status NOT IN ('CLOSED','CANCELED') ORDER BY id LIMIT 100").QueryRows(&rows)
	return rows, err
}
func (r *CoinMRunner) symbol(symbol string) (models.DeliverySymbols, error) {
	var coin models.DeliverySymbols
	if err := r.db.QueryTable(new(models.DeliverySymbols)).Filter("Symbol", symbol).
		Filter("ContractType", "PERPETUAL").One(&coin); err != nil {
		return coin, err
	}
	return coin, nil
}
func (r *CoinMRunner) mayRead(key string, min time.Duration) bool {
	at := r.clock()
	if last, ok := r.marketAt[key]; ok && at.Sub(last) < min {
		return false
	}
	r.marketAt[key] = at
	return true
}
func (r *CoinMRunner) mayRecover(key string, min time.Duration) bool {
	at := r.clock()
	if last, ok := r.orderAt[key]; ok && at.Sub(last) < min {
		return false
	}
	r.orderAt[key] = at
	return true
}
func (r *CoinMRunner) Tick(ctx context.Context) error {
	if r == nil {
		return errors.New("nil COIN-M runner")
	}
	if ctx == nil {
		return errors.New("missing COIN-M scheduler context")
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if !r.running.CompareAndSwap(false, true) {
		return nil
	}
	defer r.running.Store(false)
	global, err := coinMEntryEnabled(r.db)
	if err != nil {
		return fmt.Errorf("COIN-M scheduler global gate: %w", err)
	}
	orders, err := r.unsettled()
	if err != nil {
		return fmt.Errorf("COIN-M unsettled ledger: %w", err)
	}
	// Reconcile all old/uncertain orders before ANY new strategy decision.
	// Never POST a new order in the same tick as recovering an unsettled ID.
	if len(orders) > 0 {
		for _, order := range orders {
			if err := ctx.Err(); err != nil {
				return err
			}
			gap := coinMTickOrderMinimum
			if order.Intent == "PROTECT" && order.Status == "ACTIVE" {
				gap = coinMProtectAuditMinimum
			}
			if !r.mayRecover(order.ClientOrderID, gap) {
				continue
			}
			var err error
			if order.Intent == "PROTECT" {
				_, err = r.processor.ReconcileProtection(ctx, order.ClientOrderID)
			} else {
				_, err = r.processor.Reconcile(ctx, order.ClientOrderID)
			}
			if err != nil {
				return fmt.Errorf("COIN-M %s recovery: %w", order.ClientOrderID, err)
			}
			return nil
		}
		// Existing active Algos are allowed while evaluating an owned position,
		// but ANY ordinary unsettled order must still block a new entry or exit.
		for _, o := range orders {
			if o.Intent != "PROTECT" || o.Status != "ACTIVE" {
				return nil
			}
		}
	}
	positions, err := r.managed()
	if err != nil {
		return fmt.Errorf("COIN-M managed inventory: %w", err)
	}
	if len(positions) > 0 {
		// The existing personal-account rule is ONE managed position globally.
		// Existing inventory always takes precedence over candidate entries.
		if len(positions) > 1 {
			return errors.New("multiple active COIN-M Managed cycles require reconciliation")
		}
		pos := positions[0]
		if pos.Status != "OPEN" {
			return nil
		} // partial, unknown and pending cannot authorize anything
		if !r.mayRead("exit|"+pos.Symbol, coinMTickSymbolMinimum) {
			return nil
		}
		coin, err := r.symbol(pos.Symbol)
		if err != nil {
			return err
		}
		return r.managePosition(ctx, coin, pos, orders)
	}
	if !global || !r.processor.API.Ready() {
		return nil
	}
	// Read one COIN-M symbol per two-second tick to cap public Klines and
	// signed account preflight weight independently from USD-M and Spot.
	// Round-robin over ALL enabled pages, not just the first 100 symbols.
	const pageLimit = 100
	first, e := coinmquant.Symbols(r.db, coinmquant.SymbolFilters{Enabled: "1", Page: 1, Limit: pageLimit, Sort: "id"})
	if e != nil {
		return e
	}
	if first.Total == 0 {
		return nil
	}
	target := r.cursor % int(first.Total)
	r.cursor++
	batch := first
	if page := target/pageLimit + 1; page > 1 {
		batch, e = coinmquant.Symbols(r.db, coinmquant.SymbolFilters{Enabled: "1", Page: page, Limit: pageLimit, Sort: "id"})
		if e != nil {
			return e
		}
	}
	index := target % pageLimit
	if index >= len(batch.List) {
		return errors.New("COIN-M symbols changed during paginated scan")
	}
	item := batch.List[index]
	if !r.mayRead("entry|"+item.Symbol, coinMTickSymbolMinimum) {
		return nil
	}
	return r.tryEntry(ctx, item.DeliverySymbols)
}

// A new automatically traded cycle must have a valid executable STOP and
// TAKE_PROFIT configuration. Do not open first and discover that the
// protection price cannot be computed only after the Fill arrives.
func coinMAutoProtectionReady(coin models.DeliverySymbols) error {
	if coin.Leverage < 1 || coin.Leverage > 125 {
		return errors.New("COIN-M auto entry leverage invalid for protective triggers")
	}
	if _, err := positive(coin.TickSize); err != nil {
		return fmt.Errorf("COIN-M auto entry tick size invalid: %w", err)
	}
	bound := newRatMul(ratInt(coin.Leverage), ratInt(100))
	for _, threshold := range []struct{ name, value string }{{"STOP", coin.Loss}, {"TAKE_PROFIT", coin.Profit}} {
		roi, err := positive(threshold.value)
		if err != nil || roi.Cmp(bound) >= 0 {
			return fmt.Errorf("COIN-M auto entry requires positive executable %s inverse ROI threshold", threshold.name)
		}
	}
	return nil
}
func (r *CoinMRunner) tryEntry(ctx context.Context, coin models.DeliverySymbols) error {
	if coin.Enable != 1 || coin.ContractType != "PERPETUAL" {
		return nil
	}
	if err := coinMAutoProtectionReady(coin); err != nil {
		return err
	}
	side, err := r.strategy.Entry(ctx, coin)
	if err != nil {
		return fmt.Errorf("%s strategy entry: %w", coin.Symbol, err)
	}
	if side == "" {
		return nil
	}
	if side != "LONG" && side != "SHORT" {
		return errors.New("invalid COIN-M entry signal side")
	}
	interval, err := coinMSignalInterval(coin)
	if err != nil {
		return err
	}
	event, err := r.eventOpen(ctx, coin.Symbol, interval, r.clock())
	if err != nil {
		return err
	}
	prior, err := r.hasEntryEvent(coin.Symbol, event, interval)
	if err != nil {
		return err
	}
	if prior {
		return nil
	}
	// Double-check DB switches immediately before the DAPI preflight/post,
	// and permit the CM-5 Processor to re-check them again independently.
	global, err := coinMEntryEnabled(r.db)
	if err != nil {
		return err
	}
	cfg, err := r.symbol(coin.Symbol)
	if err != nil {
		return err
	}
	if !global || cfg.Enable != 1 || !r.processor.API.Ready() {
		return nil
	}
	// A user may edit rules, indicators or STOP/TP protection while DAPI
	// candles are being evaluated. Revalidate the refreshed configuration
	// and discard the whole old signal on ANY change to a field used by
	// strategy evaluation or automatic protection. Otherwise a valid
	// preflight from the old snapshot could open an unprotected position.
	if cfg.Strategy != coin.Strategy || cfg.Technology != coin.Technology ||
		cfg.Profit != coin.Profit || cfg.Loss != coin.Loss ||
		cfg.TickSize != coin.TickSize || cfg.Leverage != coin.Leverage {
		return nil
	}
	if err := coinMAutoProtectionReady(cfg); err != nil {
		return err
	}
	client := coinMStableEntryID(r.environment(), coin.Symbol, side, interval, event)
	if client == "" {
		return errors.New("COIN-M entry identity cannot be generated")
	}
	_, err = r.processor.Open(ctx, OpenRequest{Symbol: coin.Symbol, PositionSide: side, ClientOrderID: client, Type: "MARKET"})
	return err
}
func (r *CoinMRunner) hasEntryEvent(symbol string, event time.Time, interval string) (bool, error) {
	var ids []struct {
		ClientOrderID string `orm:"column(client_order_id)"`
	}
	_, err := r.db.Raw("SELECT client_order_id FROM coin_m_managed_orders WHERE symbol=? AND intent='OPEN'", symbol).QueryRows(&ids)
	if err != nil {
		return false, err
	}
	now := r.clock()
	intervalEnd := coinMEventEnd(event, interval)
	for _, o := range ids {
		oldInterval, oldOpen, ok := coinMDecodeEntryID(o.ClientOrderID)
		if !ok {
			// Unrecognized already-owned entry IDs may indicate another CM
			// environment or earlier executor. Never assume safe to re-enter.
			return true, nil
		}
		oldEnd := coinMEventEnd(oldOpen, oldInterval)
		if oldOpen.Before(intervalEnd) && oldEnd.After(event) && now.Before(oldEnd) {
			return true, nil
		}
	}
	return false, nil
}
func coinMStableExitID(symbol string, positionID int64) string {
	if symbol == "" || positionID <= 0 {
		return ""
	}
	h := sha256.Sum256([]byte("cm6_exit_v1|" + symbol + "|" + strconv.FormatInt(positionID, 10)))
	return "cmx" + hex.EncodeToString(h[:15])
}
func (r *CoinMRunner) managePosition(ctx context.Context, coin models.DeliverySymbols, pos models.CoinMManagedPosition, orders []models.CoinMManagedOrder) error {
	if pos.PositionSide != "LONG" && pos.PositionSide != "SHORT" {
		return errors.New("invalid managed COIN-M hedge direction")
	}
	remain, err := positive(pos.RemainingContracts)
	if err != nil {
		return err
	}
	if remain.Sign() == 0 {
		return nil
	}
	if !r.processor.API.Ready() {
		return nil
	}
	mark, err := r.processor.API.Mark(ctx, pos.Symbol)
	if err != nil {
		return err
	}
	summary, err := coinmquant.PositionSummaries(r.db, coinmquant.LedgerFilter{Symbol: pos.Symbol, Limit: 100},
		map[string]coinmquant.Mark{pos.Symbol: {Symbol: pos.Symbol, Price: mark, Time: r.clock().UnixMilli()}})
	if err != nil {
		return err
	}
	var own *coinmquant.PositionSummary
	for i := range summary.List {
		if summary.List[i].ID == pos.ID {
			own = &summary.List[i]
			break
		}
	}
	if own == nil || own.PnLStatus != "verified" || own.MarkPrice == nil {
		return errors.New("COIN-M existing Managed position is not fully verified by Fills and mark")
	}
	exit, err := r.strategy.Exit(ctx, coin, *own)
	if err != nil {
		return err
	}
	if exit {
		// Close() refuses to act while a protective Algo is still ACTIVE.
		// Cancel only explicitly owned protection, and never proceed after a
		// failed/unknown cancel or a child trigger.
		for _, o := range orders {
			if o.PositionID != pos.ID || o.Intent != "PROTECT" || o.Status != "ACTIVE" {
				continue
			}
			if _, err = r.processor.CancelProtection(ctx, o.ClientOrderID); err != nil {
				return err
			}
		}
		_, err = r.processor.Close(ctx, CloseRequest{PositionID: pos.ID, ClientOrderID: coinMStableExitID(pos.Symbol, pos.ID), Type: "MARKET"})
		return err
	}
	// Preserve both opposite-side protective Algos for ongoing positions.
	// Do not create another if any protection is unresolved.
	for _, o := range orders {
		if o.PositionID == pos.ID && o.Intent == "PROTECT" && o.Status != "ACTIVE" &&
			o.Status != "FILLED" && o.Status != "CANCELED" && o.Status != "EXPIRED" && o.Status != "REJECTED" {
			return errors.New("unresolved COIN-M protection cannot be replaced")
		}
	}
	for _, typ := range []string{"STOP_MARKET", "TAKE_PROFIT_MARKET"} {
		found := false
		for _, o := range orders {
			if o.PositionID == pos.ID && o.Intent == "PROTECT" && o.OrderType == typ && o.Status == "ACTIVE" {
				found = true
				break
			}
		}
		if found {
			continue
		}
		trigger, err := coinMProtectionTrigger(coin, pos, typ)
		if err != nil {
			return err
		}
		suffix := "sl"
		if typ == "TAKE_PROFIT_MARKET" {
			suffix = "tp"
		}
		id := coinMProtectionClientID(pos.Symbol, pos.ID, suffix)
		// One Algo request per tick. Next tick installs the other protection.
		created, err := r.processor.PlaceProtection(ctx, ProtectionRequest{
			PositionID: pos.ID, ClientAlgoID: id, Type: typ, TriggerPrice: trigger,
		})
		if err != nil {
			return err
		}
		if created.Status != "ACTIVE" {
			return fmt.Errorf("%w: COIN-M %s protection is not confirmed active", ErrUncertain, typ)
		}
		return nil
	}
	return nil
}
func coinMProtectionClientID(symbol string, positionID int64, suffix string) string {
	h := sha256.Sum256([]byte("cm6_protect_v1|" + symbol + "|" + strconv.FormatInt(positionID, 10) + "|" + suffix))
	return "cmp" + suffix + hex.EncodeToString(h[:14])
}
func coinMProtectionTrigger(coin models.DeliverySymbols, pos models.CoinMManagedPosition, typ string) (string, error) {
	if coin.Leverage <= 0 {
		return "", errors.New("invalid COIN-M leverage")
	}
	entry, err := positive(pos.EntryPrice)
	if err != nil {
		return "", err
	}
	var raw string
	if typ == "STOP_MARKET" {
		raw = coin.Loss
	} else if typ == "TAKE_PROFIT_MARKET" {
		raw = coin.Profit
	} else {
		return "", errors.New("unsupported COIN-M protection")
	}
	pct, err := positive(raw)
	if err != nil {
		return "", err
	}
	multiplier := newRatDiv(pct, ratInt(coin.Leverage*100))
	one := ratInt(1)
	minus := typ == "TAKE_PROFIT_MARKET" && pos.PositionSide == "LONG" ||
		typ == "STOP_MARKET" && pos.PositionSide == "SHORT"
	denominator := new(big.Rat).Set(one)
	if minus {
		denominator.Sub(denominator, multiplier)
	} else {
		denominator.Add(denominator, multiplier)
	}
	if denominator.Sign() <= 0 {
		return "", errors.New("COIN-M protection price denominator nonpositive")
	}
	rawTrigger := newRatDiv(entry, denominator)
	step, err := positive(coin.TickSize)
	if err != nil {
		return "", err
	}
	scaled := newRatDiv(rawTrigger, step)
	quotient := new(big.Int).Quo(scaled.Num(), scaled.Denom())
	needsCeil := typ == "STOP_MARKET" && pos.PositionSide == "LONG" ||
		typ == "TAKE_PROFIT_MARKET" && pos.PositionSide == "SHORT"
	if needsCeil && new(big.Rat).SetInt(quotient).Cmp(scaled) < 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	price := newRatMul(new(big.Rat).SetInt(quotient), step)
	if price.Sign() <= 0 {
		return "", errors.New("COIN-M protection trigger below tick")
	}
	return str(price), nil
}

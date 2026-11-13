package feature

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go_binance_futures/feature/strategy/line"
	"go_binance_futures/models"
	"go_binance_futures/service/spotexecution"
	"go_binance_futures/service/spotledger"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
	"github.com/shopspring/decimal"
)

var spotRunnerDBSeq atomic.Uint64
var spotRunnerModelsOnce sync.Once

const runnerLongRules = `[{"name":"buy","type":"long","enable":true,"code":"NowPrice > 90"},{"name":"sell","type":"close_long","enable":true,"code":"ROI > 8"}]`

func spotRunnerFixture(t *testing.T) (*SpotQuantRunner, *spotRunnerFakeExec, *spotRunnerFakeStrategy, orm.Ormer) {
	t.Helper()
	_ = orm.RegisterDriver("sqlite3", orm.DRSqlite)
	spotRunnerModelsOnce.Do(func() { orm.RegisterModel(new(models.SpotSymbols)) })
	alias := fmt.Sprintf("spot_runner_0672_%d", spotRunnerDBSeq.Add(1))
	if err := orm.RegisterDataBase(alias, "sqlite3", filepath.Join(t.TempDir(), "runner.db")); err != nil {
		t.Fatal(err)
	}
	db := orm.NewOrmUsingDB(alias)
	for _, sql := range []string{
		`CREATE TABLE config(id INTEGER PRIMARY KEY, spot_quant_enable INTEGER NOT NULL)`,
		`INSERT INTO config VALUES(1,0)`,
		`CREATE TABLE spot_symbols(
			id INTEGER PRIMARY KEY, symbol VARCHAR(32) UNIQUE NOT NULL,
			percentChange REAL DEFAULT 0,close VARCHAR(64) DEFAULT '100',
			open VARCHAR(64) DEFAULT '100',low VARCHAR(64) DEFAULT '99',
			high VARCHAR(64) DEFAULT '110',enable INTEGER DEFAULT 0,
			spot_quant_enabled INTEGER DEFAULT 0,updateTime BIGINT DEFAULT 0,
			lastClose VARCHAR(64) DEFAULT '100',lastUpdateTime BIGINT DEFAULT 0,
			baseVolume REAL DEFAULT 0,quoteVolume REAL DEFAULT 0,
			closeQty REAL DEFAULT 0,tradeCount REAL DEFAULT 0,
			tickSize VARCHAR(64) DEFAULT '0.01',stepSize VARCHAR(64) DEFAULT '0.001',
			usdt VARCHAR(64) DEFAULT '100',profit VARCHAR(64) DEFAULT '8',
			loss VARCHAR(64) DEFAULT '6',technology TEXT DEFAULT '{}',
			strategy TEXT DEFAULT '[]',strategy_type VARCHAR(32) DEFAULT 'custom',
			pin BIGINT DEFAULT 0,sort BIGINT DEFAULT 0,type VARCHAR(12) DEFAULT 'USDT')`,
		`CREATE TABLE spot_managed_positions(
			id INTEGER PRIMARY KEY AUTOINCREMENT,symbol VARCHAR(32) NOT NULL,
			base_asset VARCHAR(32) NOT NULL,quote_asset VARCHAR(32) NOT NULL,
			bought_qty DECIMAL(36,18) NOT NULL DEFAULT 0,
			remaining_qty DECIMAL(36,18) NOT NULL DEFAULT 0,
			avg_cost_quote DECIMAL(36,18) NOT NULL DEFAULT 0,
			realized_pnl DECIMAL(36,18) NOT NULL DEFAULT 0,
			status VARCHAR(32) NOT NULL,created_at BIGINT NOT NULL,
			updated_at BIGINT NOT NULL,closed_at BIGINT NOT NULL DEFAULT 0)`,
		`CREATE TABLE spot_managed_orders(
			id INTEGER PRIMARY KEY AUTOINCREMENT,position_id BIGINT NOT NULL,
			symbol VARCHAR(32) NOT NULL,side VARCHAR(4) NOT NULL,
			order_type VARCHAR(24) NOT NULL,client_order_id VARCHAR(64) NOT NULL UNIQUE,
			exchange_order_id VARCHAR(64),requested_qty DECIMAL(36,18) NOT NULL DEFAULT 0,
			filled_qty DECIMAL(36,18) NOT NULL DEFAULT 0,status VARCHAR(32) NOT NULL,
			created_at BIGINT NOT NULL,updated_at BIGINT NOT NULL,
			UNIQUE(symbol,exchange_order_id))`,
		`CREATE TABLE spot_trade_fills(
			id INTEGER PRIMARY KEY AUTOINCREMENT,position_id BIGINT NOT NULL,
			order_id BIGINT NOT NULL,exchange_order_id VARCHAR(64) NOT NULL,
			trade_id VARCHAR(64) NOT NULL,symbol VARCHAR(32) NOT NULL,
			side VARCHAR(4) NOT NULL,qty DECIMAL(36,18) NOT NULL,
			price DECIMAL(36,18) NOT NULL,quote_qty DECIMAL(36,18) NOT NULL,
			commission DECIMAL(36,18) NOT NULL,
			commission_asset VARCHAR(32) NOT NULL,trade_time BIGINT NOT NULL,
			UNIQUE(symbol,exchange_order_id,trade_id))`,
		`CREATE TABLE futures_managed_positions(id INTEGER PRIMARY KEY,marker VARCHAR(64))`,
		`INSERT INTO futures_managed_positions VALUES(1,'untouched')`,
	} {
		if _, err := db.Raw(sql).Exec(); err != nil {
			t.Fatalf("spot runner fixture SQL: %v", err)
		}
	}
	if _, err := db.Raw(`INSERT INTO spot_symbols(id,symbol,spot_quant_enabled,usdt,profit,loss,strategy,technology,type,updateTime)
		VALUES(1,'BTCUSDT',0,'100','8','6',?,'{}','USDT',?)`, runnerLongRules, time.Now().UnixMilli()).Exec(); err != nil {
		t.Fatal(err)
	}
	ledger, err := spotledger.New(db, "sqlite")
	if err != nil {
		t.Fatal(err)
	}
	eval := &spotRunnerFakeStrategy{entry: true}
	ex := &spotRunnerFakeExec{ledger: ledger}
	runner, err := NewSpotQuantRunner(db, ledger, eval, ex)
	if err != nil {
		t.Fatal(err)
	}
	fixed := time.Unix(1794517200, 0)
	runner.clock = func() time.Time { return fixed }
	return runner, ex, eval, db
}

type spotRunnerFakeStrategy struct {
	entry, exit           bool
	entryCalls, exitCalls int
	fail                  error
	onEntry               func()
}

func (s *spotRunnerFakeStrategy) EvaluateEntry(_ models.SpotSymbols) (line.SpotSignal, error) {
	s.entryCalls++
	if s.onEntry != nil {
		s.onEntry()
	}
	if s.fail != nil {
		return line.SpotSignal{}, s.fail
	}
	if s.entry {
		return line.SpotSignal{Action: "BUY", RuleHash: "editable-rule-hash"}, nil
	}
	return line.SpotSignal{}, nil
}
func (s *spotRunnerFakeStrategy) EvaluateExit(_ models.SpotSymbols, ledger *spotledger.Ledger, id int64) (line.SpotSignal, error) {
	s.exitCalls++
	// Match 06-7-1's production Fill-sourced data contract.
	snapshot, err := ledger.Accounting(id, "")
	if err != nil {
		return line.SpotSignal{}, err
	}
	if snapshot.Position.Status != "OPEN" {
		return line.SpotSignal{}, fmt.Errorf("position not OPEN")
	}
	if s.fail != nil {
		return line.SpotSignal{}, s.fail
	}
	if s.exit {
		return line.SpotSignal{Action: "SELL", RuleHash: "editable-close-hash"}, nil
	}
	return line.SpotSignal{}, nil
}

type spotRunnerFakeExec struct {
	ledger                  *spotledger.Ledger
	buys, sells, reconciles int
	unknownBuy              bool
}

func (x *spotRunnerFakeExec) ExecuteBuy(_ context.Context, symbol, budget, clientID string) (models.SpotManagedOrder, error) {
	x.buys++
	if budget != "100" {
		return models.SpotManagedOrder{}, fmt.Errorf("incorrect quote budget %q", budget)
	}
	base := strings.TrimSuffix(symbol, "USDT")
	p, err := x.ledger.CreatePosition(symbol, base, "USDT")
	if err != nil {
		return models.SpotManagedOrder{}, err
	}
	o, _, err := x.ledger.CreateOrder(spotledger.OrderIntent{PositionID: p.ID, Symbol: symbol, Side: "BUY", Type: "MARKET", ClientOrderID: clientID, RequestedQty: "1"})
	if err != nil || x.unknownBuy {
		return o, spotexecution.ErrUncertain
	}
	return x.fill(o, "100")
}

func (x *spotRunnerFakeExec) ExecuteSell(_ context.Context, positionID int64, clientID string) (models.SpotManagedOrder, error) {
	x.sells++
	snapshot, err := x.ledger.Accounting(positionID, "")
	if err != nil {
		return models.SpotManagedOrder{}, err
	}
	o, _, err := x.ledger.CreateOrder(spotledger.OrderIntent{
		PositionID: positionID, Symbol: snapshot.Position.Symbol,
		Side: "SELL", Type: "MARKET", ClientOrderID: clientID,
		RequestedQty: snapshot.Position.RemainingQty,
	})
	if err != nil {
		return o, err
	}
	return x.fill(o, "110")
}

func (x *spotRunnerFakeExec) fill(o models.SpotManagedOrder, price string) (models.SpotManagedOrder, error) {
	exchangeID := fmt.Sprintf("%d", 990000+o.ID)
	if _, err := x.ledger.BindExchangeOrderID(o.ID, exchangeID); err != nil {
		return o, err
	}
	qty := decimal.RequireFromString(o.RequestedQty)
	amount := qty.Mul(decimal.RequireFromString(price))
	_, err := x.ledger.RecordFill(spotledger.FillEvent{
		OrderID: o.ID, Symbol: o.Symbol, ExchangeOrderID: exchangeID,
		TradeID: fmt.Sprintf("%d", 880000+o.ID), Side: o.Side,
		Qty: o.RequestedQty, Price: price, QuoteQty: amount.String(),
		Commission: "0", CommissionAsset: "", TradeTime: time.Now().UnixMilli(),
	})
	if err != nil {
		return o, err
	}
	return x.ledger.GetOrder(o.ID)
}
func (x *spotRunnerFakeExec) Reconcile(_ context.Context, orderID int64) (models.SpotManagedOrder, error) {
	x.reconciles++
	return x.ledger.GetOrder(orderID)
}

func spotRunnerSetSwitches(t *testing.T, db orm.Ormer, global, symbol int) {
	t.Helper()
	if _, err := db.Raw("UPDATE config SET spot_quant_enable=? WHERE id=1", global).Exec(); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Raw("UPDATE spot_symbols SET spot_quant_enabled=? WHERE symbol='BTCUSDT'", symbol).Exec(); err != nil {
		t.Fatal(err)
	}
}

func TestSpotQuant0672SwitchesRepeatedSignalsAndSecondCycle(t *testing.T) {
	r, ex, eval, db := spotRunnerFixture(t)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		if err := r.Run(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if ex.buys != 0 || eval.entryCalls != 0 {
		t.Fatal("global OFF evaluated or submitted Spot trades")
	}
	spotRunnerSetSwitches(t, db, 1, 0)
	for i := 0; i < 5; i++ {
		if err := r.Run(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if ex.buys != 0 {
		t.Fatal("symbol OFF submitted a Spot order")
	}
	spotRunnerSetSwitches(t, db, 1, 1)
	for i := 0; i < 20; i++ {
		if err := r.Run(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if ex.buys != 1 || ex.sells != 0 {
		t.Fatalf("20 polls caused duplicate BUY: %+v", ex)
	}
	eval.exit = true
	spotRunnerSetSwitches(t, db, 0, 1)
	for i := 0; i < 5; i++ {
		if err := r.Run(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if ex.sells != 0 {
		t.Fatal("global OFF triggered automatic SELL")
	}
	spotRunnerSetSwitches(t, db, 1, 1)
	if err := r.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if ex.sells != 1 {
		t.Fatalf("expected exactly one SELL, got %d", ex.sells)
	}
	for i := 0; i < 10; i++ {
		if err := r.Run(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if ex.buys != 1 || ex.sells != 1 {
		t.Fatalf("same-bar reentry occurred buys=%d sells=%d", ex.buys, ex.sells)
	}
	r.clock = func() time.Time { return time.Unix(1794517260, 0) }
	if err := r.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if ex.buys != 2 {
		t.Fatalf("new bar must allow new cycle, buys=%d", ex.buys)
	}
	var sentinel string
	if err := db.Raw("SELECT marker FROM futures_managed_positions WHERE id=1").QueryRow(&sentinel); err != nil || sentinel != "untouched" {
		t.Fatalf("Futures sentinel corrupted: %q %v", sentinel, err)
	}
}

func TestSpotQuant0672UnknownOrderSurvivesRestartAndRuleChanges(t *testing.T) {
	r, ex, _, db := spotRunnerFixture(t)
	ctx := context.Background()
	spotRunnerSetSwitches(t, db, 1, 1)
	ex.unknownBuy = true
	err := r.Run(ctx)
	if !errors.Is(err, spotexecution.ErrUncertain) || ex.buys != 1 {
		t.Fatalf("unknown first BUY not retained: %v buys=%d", err, ex.buys)
	}
	for i := 0; i < 10; i++ {
		if err := r.Run(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if ex.buys != 1 || ex.reconciles != 10 {
		t.Fatalf("unknown order duplicated buys=%d reconciles=%d", ex.buys, ex.reconciles)
	}
	second, err := NewSpotQuantRunner(db, r.ledger, &spotRunnerFakeStrategy{entry: true}, ex)
	if err != nil {
		t.Fatal(err)
	}
	second.clock = r.clock
	for i := 0; i < 3; i++ {
		if err := second.Run(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if ex.buys != 1 || ex.reconciles != 13 {
		t.Fatal("restart duplicated an ambiguous BUY")
	}
	fixed := r.clock()
	if spotQuantClientID("BTCUSDT", "BUY", 0, "1m", spotCandleOpen(fixed, "1m")) != spotQuantClientID("BTCUSDT", "BUY", 0, "1m", spotCandleOpen(fixed.Add(5*time.Second), "1m")) {
		t.Fatal("same-minute id is unstable")
	}
	if spotQuantClientID("BTCUSDT", "BUY", 0, "1m", fixed) == spotQuantClientID("WBTCUSDT", "BUY", 0, "1m", fixed) ||
		spotQuantClientID("BTCUSDT", "BUY", 0, "1m", fixed) == spotQuantClientID("BTCUSDT", "SELL", 123, "1m", fixed) {
		t.Fatal("symbol/side/cycle identity collision")
	}
	if len(spotQuantClientID("BTCUSDT", "BUY", 0, "1m", fixed)) > 36 {
		t.Fatal("Binance client id too long")
	}
}

func TestSpotQuant0672NoExecutorCannotMutateOrEvaluate(t *testing.T) {
	r, ex, eval, db := spotRunnerFixture(t)
	spotRunnerSetSwitches(t, db, 1, 1)
	r.exec = nil // exact production wiring until the real-order Testnet Gate
	for i := 0; i < 20; i++ {
		if err := r.Run(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if ex.buys != 0 || ex.sells != 0 || eval.entryCalls != 0 {
		t.Fatal("read-only production path evaluated or sent an order")
	}
}

func TestSpotQuant0672PartialOrderIsReconciledBeforeAnyNewDecision(t *testing.T) {
	r, ex, eval, db := spotRunnerFixture(t)
	spotRunnerSetSwitches(t, db, 1, 1)
	eval.exit = true
	ex.unknownBuy = true
	if err := r.Run(context.Background()); !errors.Is(err, spotexecution.ErrUncertain) {
		t.Fatalf("seed uncertain: %v", err)
	}
	ex.unknownBuy = false
	for i := 0; i < 4; i++ {
		if err := r.Run(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if ex.buys != 1 || ex.sells != 0 || ex.reconciles != 4 {
		t.Fatalf("PENDING_BUY bypassed: buys=%d sells=%d reconcile=%d", ex.buys, ex.sells, ex.reconciles)
	}
}

func TestSpotQuant0672SwitchOffDuringEntryBlocksSubmission(t *testing.T) {
	r, ex, eval, db := spotRunnerFixture(t)
	spotRunnerSetSwitches(t, db, 1, 1)
	eval.onEntry = func() {
		if _, err := db.Raw("UPDATE spot_symbols SET spot_quant_enabled=0 WHERE symbol='BTCUSDT'").Exec(); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if ex.buys != 0 || ex.sells != 0 {
		t.Fatal("Spot symbol switch changed during evaluation but an order was submitted")
	}
}

func TestSpotQuant0672TwoSymbolsAreExactAndIsolated(t *testing.T) {
	r, ex, _, db := spotRunnerFixture(t)
	spotRunnerSetSwitches(t, db, 1, 1)
	if _, err := db.Raw(`INSERT INTO spot_symbols(id,symbol,spot_quant_enabled,usdt,profit,loss,strategy,technology,type,updateTime)
	VALUES(2,'WBTCUSDT',1,'100','8','6',?,'{}','USDT',?)`, runnerLongRules, time.Now().UnixMilli()).Exec(); err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if ex.buys != 2 {
		t.Fatalf("enabled independent symbols generated %d BUY orders, want 2", ex.buys)
	}
	for _, symbol := range []string{"BTCUSDT", "WBTCUSDT"} {
		page, err := r.ledger.ListPositions(spotledger.PositionFilter{Symbol: symbol, Status: "OPEN", Page: 1, Limit: 10})
		if err != nil || page.Total != 1 {
			t.Fatalf("%s active position leak: %+v %v", symbol, page, err)
		}
	}
	if err := r.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if ex.buys != 2 {
		t.Fatalf("second polling cycle duplicated positions: buys=%d", ex.buys)
	}
}

func TestSpotQuant0672PendingSellIsReconciledBeforeExit(t *testing.T) {
	r, ex, eval, db := spotRunnerFixture(t)
	ctx := context.Background()
	spotRunnerSetSwitches(t, db, 1, 1)
	if err := r.Run(ctx); err != nil {
		t.Fatal(err)
	}
	eval.exit = true
	pos, exists, err := r.activeSpotPosition("BTCUSDT")
	if err != nil || !exists {
		t.Fatalf("missing open Spot inventory: %v", err)
	}
	if _, _, err := r.ledger.CreateOrder(spotledger.OrderIntent{
		PositionID: pos.ID, Symbol: "BTCUSDT", Side: "SELL", Type: "MARKET",
		ClientOrderID: "pending_spot_sell", RequestedQty: "1",
	}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err := r.Run(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if ex.sells != 0 || ex.reconciles != 3 {
		t.Fatalf("unresolved SELL was bypassed: sells=%d reconcile=%d", ex.sells, ex.reconciles)
	}
}

func TestSpotQuant0672OverlappingCyclesSkipWithoutQueue(t *testing.T) {
	r, ex, eval, db := spotRunnerFixture(t)
	spotRunnerSetSwitches(t, db, 1, 1)
	started := make(chan struct{})
	release := make(chan struct{})
	eval.onEntry = func() {
		close(started)
		<-release
	}
	completed := make(chan error, 1)
	go func() { completed <- r.Run(context.Background()) }()
	<-started
	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("overlapping run was not skipped: %v", err)
	}
	close(release)
	if err := <-completed; err != nil {
		t.Fatal(err)
	}
	if ex.buys != 1 || eval.entryCalls != 1 {
		t.Fatalf("overlapping runner repeated a signal: buys=%d calls=%d", ex.buys, eval.entryCalls)
	}
}

// Review F2: one real 5m/1h event must never produce repeated BUY/SELL
// cycles just because the scheduler runs every two seconds or a minute rolls.
func TestReview0672KlineIdentitySurvivesMinuteRollAndRuleEdit(t *testing.T) {
	for _, tc := range []struct {
		interval string
		minutes  int
	}{
		{"5m", 5}, {"1h", 60},
	} {
		t.Run(tc.interval, func(t *testing.T) {
			r, ex, eval, db := spotRunnerFixture(t)
			spotRunnerSetSwitches(t, db, 1, 1)
			tech := fmt.Sprintf(`{"ema":[{"name":"ema_5","enable":true,"kline_interval":"%s","period":5}]}`, tc.interval)
			if _, err := db.Raw("UPDATE spot_symbols SET technology=? WHERE symbol='BTCUSDT'", tech).Exec(); err != nil {
				t.Fatal(err)
			}
			base := time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)
			r.clock = func() time.Time { return base }
			if err := r.Run(context.Background()); err != nil {
				t.Fatal(err)
			}
			eval.exit = true
			r.clock = func() time.Time { return base.Add(time.Minute) }
			if err := r.Run(context.Background()); err != nil {
				t.Fatal(err)
			}
			if ex.buys != 1 || ex.sells != 1 {
				t.Fatalf("expected first complete cycle buys=%d sells=%d", ex.buys, ex.sells)
			}
			if _, err := db.Raw("UPDATE spot_symbols SET strategy=? WHERE symbol='BTCUSDT'",
				`[{"name":"edited long","type":"long","enable":true,"code":"NowPrice>1"}]`).Exec(); err != nil {
				t.Fatal(err)
			}
			for minute := 2; minute < tc.minutes; minute++ {
				instant := base.Add(time.Duration(minute) * time.Minute)
				r.clock = func() time.Time { return instant }
				if err := r.Run(context.Background()); err != nil {
					t.Fatal(err)
				}
			}
			if ex.buys != 1 || ex.sells != 1 {
				t.Fatalf("%s event repeated due minute rollover or edit: buys=%d sells=%d", tc.interval, ex.buys, ex.sells)
			}
			next := base.Add(time.Duration(tc.minutes) * time.Minute)
			r.clock = func() time.Time { return next }
			if err := r.Run(context.Background()); err != nil {
				t.Fatal(err)
			}
			if ex.buys != 2 {
				t.Fatalf("%s new candle did not allow next BUY: %d", tc.interval, ex.buys)
			}
		})
	}
}

func TestReview0672SpotIntervalExplicitReferencesAndUTCAlignment(t *testing.T) {
	coin := models.SpotSymbols{Technology: "{}", Strategy: `[{"name":"x","type":"long","enable":true,"code":"kline_5m.Close[0] > 1"}]`}
	interval, err := spotSignalInterval(coin)
	if err != nil || interval != "5m" {
		t.Fatalf("explicit 5m strategy reference: %q %v", interval, err)
	}
	coin.Technology = `{"ma":[{"name":"ma_1h","enable":true,"kline_interval":"1h","period":5}]}`
	interval, err = spotSignalInterval(coin)
	if err != nil || interval != "1h" {
		t.Fatalf("multi interval should select 1h: %q %v", interval, err)
	}
	now := time.Date(2026, 11, 12, 12, 59, 12, 0, time.UTC)
	if got := spotCandleOpen(now, "1h"); got != time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC) {
		t.Fatalf("wrong 1h candle open: %s", got)
	}
	week := spotCandleOpen(now, "1w")
	if week.Weekday() != time.Monday || week.Hour() != 0 {
		t.Fatalf("weekly Spot candle not UTC Monday: %s", week)
	}
	month := spotCandleOpen(now, "1M")
	if month.Day() != 1 || month.Hour() != 0 {
		t.Fatalf("monthly Spot candle not 1st UTC: %s", month)
	}
	coin.Technology = `{"ema":[{"name":"bad","enable":true,"kline_interval":"unknown"}]}`
	if _, err := spotSignalInterval(coin); err == nil {
		t.Fatal("unrecognized indicator interval accepted")
	}
}

func TestReview0672TamperedCLOSEDCacheStillReconcilesPendingFill(t *testing.T) {
	r, ex, _, db := spotRunnerFixture(t)
	spotRunnerSetSwitches(t, db, 1, 1)
	ex.unknownBuy = true
	if err := r.Run(context.Background()); !errors.Is(err, spotexecution.ErrUncertain) {
		t.Fatalf("seeding uncertain BUY: %v", err)
	}
	if _, err := db.Raw("UPDATE spot_managed_positions SET status='CLOSED' WHERE symbol='BTCUSDT'").Exec(); err != nil {
		t.Fatal(err)
	}
	r.clock = func() time.Time { return time.Unix(1794517800, 0) }
	if err := r.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if ex.buys != 1 || ex.reconciles != 1 {
		t.Fatalf("tampered cached CLOSED bypassed recovery: buys=%d rec=%d", ex.buys, ex.reconciles)
	}
	active, exists, err := r.activeSpotPosition("BTCUSDT")
	if err != nil || !exists || active.Status != "PENDING_BUY" {
		t.Fatalf("Fill source failed to expose PENDING_BUY: %+v exists=%v err=%v", active, exists, err)
	}
}

func TestReview0672ReconcileALLPendingEvenWithoutActivePosition(t *testing.T) {
	r, ex, _, db := spotRunnerFixture(t)
	spotRunnerSetSwitches(t, db, 1, 1)
	for _, id := range []string{"orphan_one", "orphan_two"} {
		if _, err := db.Raw(`INSERT INTO spot_managed_orders(
			position_id,symbol,side,order_type,client_order_id,exchange_order_id,
			requested_qty,filled_qty,status,created_at,updated_at
			) VALUES(999,'BTCUSDT','BUY','MARKET',?,NULL,'1','0','NEW',1000,1000)`, id).Exec(); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if ex.reconciles != 2 || ex.buys != 0 {
		t.Fatalf("orphan pending orders skipped: rec=%d buys=%d", ex.reconciles, ex.buys)
	}
}

func TestSpotQuantMainEntryFailsClosedIfGlobalSwitchUnavailable(t *testing.T) {
	// Mainnet and Testnet share one runtime; without a registered DB
	// and verified global switch, startup must return an error, not panic.
	if err := StartSpotQuantTrade(context.Background()); err == nil {
		t.Fatal("Spot Quant entry bypassed missing global DB switch")
	}
}

// Review F3: even with no unresolved Order, a manually altered cached
// CLOSED flag cannot hide an OPEN Fill-verified position.
func TestReview0672TamperedClosedCacheStillClosesVerifiedOpenPosition(t *testing.T) {
	r, ex, eval, db := spotRunnerFixture(t)
	spotRunnerSetSwitches(t, db, 1, 1)
	if err := r.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Raw("UPDATE spot_managed_positions SET status='CLOSED',remaining_qty='0',avg_cost_quote='1' WHERE symbol='BTCUSDT'").Exec(); err != nil {
		t.Fatal(err)
	}
	eval.exit = true
	if err := r.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if ex.buys != 1 || ex.sells != 1 {
		t.Fatalf("tampered CLOSED cache caused wrong decision: buys=%d sells=%d", ex.buys, ex.sells)
	}
}

// Review F6 + F8: identical errors are state-tracked; historical +/-100%
// default exit thresholds receive one explicit operator warning per symbol.
func TestReview0672ErrorChangesAndDefaultThresholdWarning(t *testing.T) {
	r, ex, eval, db := spotRunnerFixture(t)
	spotRunnerSetSwitches(t, db, 1, 1)
	if _, err := db.Raw("UPDATE spot_symbols SET profit='100',loss='100' WHERE symbol='BTCUSDT'").Exec(); err != nil {
		t.Fatal(err)
	}
	eval.fail = errors.New("malformed Spot indicator")
	for i := 0; i < 5; i++ {
		if err := r.Run(context.Background()); err == nil {
			t.Fatal("invalid indicator unexpectedly emitted BUY")
		}
	}
	if !r.warnings["BTCUSDT"] || len(r.warnings) != 1 || len(r.lastErr) != 1 || ex.buys != 0 {
		t.Fatalf("warning/error state or fail closed broken: warnings=%v errors=%v buys=%d", r.warnings, r.lastErr, ex.buys)
	}
	eval.fail = nil
	if err := r.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(r.lastErr) != 0 || ex.buys != 1 {
		t.Fatalf("recovered error state not cleared: errors=%v buys=%d", r.lastErr, ex.buys)
	}
}

// Clean source checkout can test the callback without executing main.go's
// production DB initialization and without any private app.conf.
func TestReview0672SpotCallbackPanicIsolatedAndLogStateDeduplicated(t *testing.T) {
	RunSpotQuantProtected(func() { panic("isolated Spot callback failure") })
	completed := 0
	RunSpotQuantProtected(func() { completed++ })
	if completed != 1 {
		t.Fatalf("Spot callback panic stopped later loop: %d", completed)
	}
	spotQuantSchedulerLogMu.Lock()
	spotQuantSchedulerLogState = ""
	spotQuantSchedulerLogMu.Unlock()
	for i := 0; i < 12; i++ {
		ReportSpotQuantScheduler(fmt.Errorf("repeated Spot reconciliation error"))
	}
	spotQuantSchedulerLogMu.Lock()
	value := spotQuantSchedulerLogState
	spotQuantSchedulerLogMu.Unlock()
	if value != "repeated Spot reconciliation error" {
		t.Fatalf("deduplication state changed unexpectedly: %q", value)
	}
	ReportSpotQuantScheduler(nil)
	spotQuantSchedulerLogMu.Lock()
	value = spotQuantSchedulerLogState
	spotQuantSchedulerLogMu.Unlock()
	if value != "" {
		t.Fatalf("successful callback failed to clear prior error state: %q", value)
	}
}

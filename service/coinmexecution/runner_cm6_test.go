package coinmexecution

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/client/orm"
	"go_binance_futures/models"
	"go_binance_futures/service/coinmquant"
)

type cm6FakeStrategy struct {
	entry, exit    bool
	entrySide      string
	entries, exits int
	err            error
	onEntry        func()
}

func (s *cm6FakeStrategy) Entry(ctx context.Context, coin models.DeliverySymbols) (string, error) {
	s.entries++
	if s.onEntry != nil {
		s.onEntry()
	}
	if s.err != nil {
		return "", s.err
	}
	if s.entry {
		if s.entrySide != "" {
			return s.entrySide, nil
		}
		return "LONG", nil
	}
	return "", nil
}
func (s *cm6FakeStrategy) Exit(ctx context.Context, coin models.DeliverySymbols, summary coinmquant.PositionSummary) (bool, error) {
	s.exits++
	if summary.PnLStatus != "verified" || summary.MarkPrice == nil || summary.UnrealizedPnL == nil {
		return false, errors.New("fake strategy was handed unverified Fill evidence")
	}
	if s.err != nil {
		return false, s.err
	}
	return s.exit, nil
}
func cm6RunnerFixture(t *testing.T) (*CoinMRunner, *Processor, *mockCM5, *cm6FakeStrategy, orm.Ormer, *time.Time) {
	t.Helper()
	db := cm5Fixture(t)
	api := newMockCM5()
	processor := NewProcessor(db, api)
	strat := &cm6FakeStrategy{entry: true}
	runner, err := NewCoinMRunner(db, processor, strat)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 18, 3, 12, 30, 0, time.UTC)
	runner.clock = func() time.Time { return now }
	runner.environment = func() string { return "fixture" }
	runner.eventOpen = func(_ context.Context, symbol, interval string, _ time.Time) (time.Time, error) {
		if symbol != "BTCUSD_PERP" || interval != "1m" {
			return time.Time{}, errors.New("unexpected COIN-M event symbol or interval")
		}
		return now.Truncate(time.Minute), nil
	}
	return runner, processor, api, strat, db, &now
}
func TestCM6DisabledEntryDoesNotTouchRemote(t *testing.T) {
	r, _, api, s, db, _ := cm6RunnerFixture(t)
	if _, err := db.Raw("UPDATE config SET coin_m_quant_enable=0").Exec(); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Raw("UPDATE delivery_symbols SET enable=0").Exec(); err != nil {
		t.Fatal(err)
	}
	needed, err := CoinMNeedsRuntime(db)
	if err != nil || needed {
		t.Fatalf("OFF and idle should never start private WS: %t %v", needed, err)
	}
	if err := r.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if s.entries != 0 || len(api.calls) != 0 {
		t.Fatalf("OFF executed entry: entries=%d posts=%d", s.entries, len(api.calls))
	}
}
func TestCM6EnabledSymbolButGlobalOffDoesNotRunStrategy(t *testing.T) {
	r, _, api, s, db, _ := cm6RunnerFixture(t)
	if _, err := db.Raw("UPDATE config SET coin_m_quant_enable=0").Exec(); err != nil {
		t.Fatal(err)
	}
	if err := r.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if s.entries != 0 || len(api.calls) != 0 {
		t.Fatal("global OFF still executed strategy")
	}
}
func TestCM6GlobalEnabledButSymbolOffDoesNotRunStrategy(t *testing.T) {
	r, _, api, s, db, _ := cm6RunnerFixture(t)
	if _, err := db.Raw("UPDATE delivery_symbols SET enable=0").Exec(); err != nil {
		t.Fatal(err)
	}
	if err := r.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if s.entries != 0 || len(api.calls) != 0 {
		t.Fatal("symbol OFF still executed entry")
	}
}
func TestCM6PrivateWSDisconnectedBlocksEntry(t *testing.T) {
	r, _, api, _, _, _ := cm6RunnerFixture(t)
	api.ready = false
	if err := r.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(api.calls) != 0 {
		t.Fatal("COIN-M without authenticated WS posted order")
	}
}
func TestCM6SameEventNoSecondEntryAndUnsettledFirst(t *testing.T) {
	r, _, api, strat, db, now := cm6RunnerFixture(t)
	if err := r.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(api.calls) != 1 || api.calls[0].PositionSide != "LONG" || len(api.calls[0].ClientOrderID) != 34 {
		t.Fatalf("COIN-M did not create expected single open intent: %+v", api.calls)
	}
	first := api.calls[0].ClientOrderID
	if !strings.HasPrefix(first, "cm6") {
		t.Fatalf("unexpected fixture-only client ID %q", first)
	}
	// The exchange has not filled the new order. REST reconciles it, and
	// the second/new signal never causes another POST.
	*now = now.Add(12 * time.Second)
	if err := r.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(api.calls) != 1 || strat.entries != 1 {
		t.Fatalf("unsettled order allowed duplicate: calls=%d entries=%d", len(api.calls), strat.entries)
	}
	expectCM5(t, db, "SELECT COUNT(*) FROM coin_m_managed_orders", 1)
	// Even if the original 1m event ends canceled without fills, a restart
	// must not reuse the same bar/identity.
	api.orders[first].Status = "CANCELED"
	*now = now.Add(12 * time.Second)
	if err := r.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	r2, err := NewCoinMRunner(db, NewProcessor(db, api), strat)
	if err != nil {
		t.Fatal(err)
	}
	r2.clock = func() time.Time { return *now }
	r2.environment = func() string { return "fixture" }
	r2.eventOpen = r.eventOpen
	if err := r2.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(api.calls) != 1 {
		t.Fatalf("restart or cancellation repeated same-candle POST: %+v", api.calls)
	}
}
func TestCM6ExistingPositionExitsWithBothEntrySwitchesOff(t *testing.T) {
	r, p, api, s, db, now := cm6RunnerFixture(t)
	ctx := context.Background()
	opening, err := p.Open(ctx, OpenRequest{Symbol: "BTCUSD_PERP", PositionSide: "LONG", ClientOrderID: "cm6existinglongentry", Type: "MARKET"})
	if err != nil {
		t.Fatal(err)
	}
	api.qty["LONG"] = "10"
	if _, err := p.ApplyFill(sampleTrade(api.orders[opening.ClientOrderID], 60001, "10", "50000", "0")); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Raw("UPDATE config SET coin_m_quant_enable=0").Exec(); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Raw("UPDATE delivery_symbols SET enable=0").Exec(); err != nil {
		t.Fatal(err)
	}
	needed, err := CoinMNeedsRuntime(db)
	if err != nil || !needed {
		t.Fatalf("Managed exit must keep private runtime active despite entry switches OFF: %t %v", needed, err)
	}
	s.exit = true
	*now = now.Add(11 * time.Second)
	if err := r.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	if len(api.calls) != 2 || api.calls[1].Side != "SELL" || api.calls[1].PositionSide != "LONG" {
		t.Fatalf("OFF switches stranded existing LONG position: %+v", api.calls)
	}
	if s.entries != 0 || s.exits != 1 {
		t.Fatalf("exit-priority violated: entry=%d exit=%d", s.entries, s.exits)
	}
	expectCM5(t, db, "SELECT COUNT(*) FROM coin_m_managed_orders WHERE intent='CLOSE'", 1)
}
func TestCM6ActiveManagedPositionTakesPriorityOverOtherEntry(t *testing.T) {
	r, p, api, s, _, now := cm6RunnerFixture(t)
	ctx := context.Background()
	opening, err := p.Open(ctx, OpenRequest{Symbol: "BTCUSD_PERP", PositionSide: "SHORT", ClientOrderID: "cm6shortmanagedentry", Type: "MARKET"})
	if err != nil {
		t.Fatal(err)
	}
	api.qty["SHORT"] = "10"
	if _, err := p.ApplyFill(sampleTrade(api.orders[opening.ClientOrderID], 60002, "10", "50000", "0")); err != nil {
		t.Fatal(err)
	}
	// Automatic protection of a SHORT position must be placed before any
	// new LONG/SHORT candidate entry; no cross-side position is created.
	s.exit = false
	*now = now.Add(11 * time.Second)
	if err := r.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	if s.entries != 0 || len(api.calls) != 1 || len(api.openAlgos) != 1 ||
		api.openAlgos[0].OrderType != "STOP_MARKET" {
		t.Fatalf("active SHORT did not receive STOP before new entry: entries=%d orders=%d algos=%+v", s.entries, len(api.calls), api.openAlgos)
	}
}
func TestCM6StableClientIdentityAndOverlap(t *testing.T) {
	open := time.Date(2026, 11, 18, 3, 12, 0, 0, time.UTC)
	id := coinMStableEntryID("mainnet", "BTCUSD_PERP", "LONG", "1m", open)
	if len(id) != 34 || !strings.HasPrefix(id, "cmn") {
		t.Fatalf("bad COIN-M stable ID %q", id)
	}
	if id != coinMStableEntryID("mainnet", "BTCUSD_PERP", "LONG", "1m", open) {
		t.Fatal("client ID not stable")
	}
	if id == coinMStableEntryID("testnet", "BTCUSD_PERP", "LONG", "1m", open) ||
		id == coinMStableEntryID("mainnet", "BTCUSD_PERP", "SHORT", "1m", open) {
		t.Fatal("environment and hedge side must have independent client IDs")
	}
	interval, decoded, ok := coinMDecodeEntryID(id)
	if !ok || interval != "1m" || !decoded.Equal(open) {
		t.Fatalf("client ID cannot preserve event timestamp %s %v %t", interval, decoded, ok)
	}
	if coinMStableEntryID("", "BTCUSD_PERP", "LONG", "1m", open) != "" {
		t.Fatal("unconfigured COIN-M environment generated a client ID")
	}
	r, _, _, _, db, _ := cm6RunnerFixture(t)
	_, err := db.Raw("INSERT INTO coin_m_managed_positions(symbol,position_side,margin_asset,contract_size,entry_price,opened_contracts,remaining_contracts,realized_pnl,status,created_at,updated_at,closed_at) VALUES('BTCUSD_PERP','LONG','BTC',100,50000,0,0,0,'CANCELED',1,1,1)").Exec()
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Raw("INSERT INTO coin_m_managed_orders(position_id,symbol,position_side,intent,side,order_type,client_order_id,exchange_order_id,requested_contracts,filled_contracts,status,created_at,updated_at) VALUES(1,'BTCUSD_PERP','LONG','OPEN','BUY','MARKET',?,NULL,10,0,'CANCELED',1,1)", id).Exec()
	if err != nil {
		t.Fatal(err)
	}
	overlapping, err := r.hasEntryEvent("BTCUSD_PERP", open, "5m")
	if err != nil || !overlapping {
		t.Fatalf("1m order must block overlapping 5m signal: %t %v", overlapping, err)
	}
}
func TestCM6InverseProtectionTriggersUseContractsAndROE(t *testing.T) {
	coin := models.DeliverySymbols{Leverage: 4, Profit: "8", Loss: "6", TickSize: "0.1"}
	pos := models.CoinMManagedPosition{EntryPrice: "50000", PositionSide: "LONG"}
	tp, err := coinMProtectionTrigger(coin, pos, "TAKE_PROFIT_MARKET")
	if err != nil || tp != "51020.4" {
		t.Fatalf("LONG TP must use inverse ROE and price tick: %s %v", tp, err)
	}
	sl, err := coinMProtectionTrigger(coin, pos, "STOP_MARKET")
	if err != nil || sl != "49261.1" {
		t.Fatalf("LONG STOP inverse trigger incorrect: %s %v", sl, err)
	}
	pos.PositionSide = "SHORT"
	tp, err = coinMProtectionTrigger(coin, pos, "TAKE_PROFIT_MARKET")
	if err != nil || tp != "49019.7" {
		t.Fatalf("SHORT TP incorrect: %s %v", tp, err)
	}
	sl, err = coinMProtectionTrigger(coin, pos, "STOP_MARKET")
	if err != nil || sl != "50761.4" {
		t.Fatalf("SHORT STOP incorrect: %s %v", sl, err)
	}
}
func TestCM6StrategySignalsUseCorrectHedgeDirection(t *testing.T) {
	market := func(_ context.Context, _ models.DeliverySymbols) (map[string]interface{}, error) {
		return map[string]interface{}{"NowPrice": float64(55000), "NowTime": int64(1)}, nil
	}
	strategy := coinMStrategy{Market: market, Klines: func(_ context.Context, _ string, interval string, limit int) ([]*futures.Kline, error) {
		if interval != "5m" || limit != 2 {
			t.Fatalf("incorrect COIN-M fallback request %s %d", interval, limit)
		}
		return []*futures.Kline{{Close: "50000"}, {Close: "55000"}}, nil
	}}
	coin := models.DeliverySymbols{Symbol: "BTCUSD_PERP", Leverage: 4, Profit: "8", Loss: "6",
		Strategy: `[{"name":"long","type":"long","enable":true,"code":"NowPrice>50000"},{"name":"short","type":"short","enable":true,"code":"NowPrice<40000"}]`}
	side, err := strategy.Entry(context.Background(), coin)
	if err != nil || side != "LONG" {
		t.Fatalf("LONG entry not evaluated: %s %v", side, err)
	}
	coin.Strategy = `[{"name":"long","type":"long","enable":true,"code":"true"},{"name":"short","type":"short","enable":true,"code":"true"}]`
	if _, err = strategy.Entry(context.Background(), coin); err == nil {
		t.Fatal("ambiguous simultaneous LONG/SHORT accepted")
	}
	coin.Strategy = `[{"name":"exit","type":"close_long","enable":true,"code":"ROI>=8 and Position.Side=='LONG'"}]`
	mark, unreal := "55000", "0.001818181818181818"
	summary := coinmquant.PositionSummary{
		CoinMManagedPosition: models.CoinMManagedPosition{
			ID: 1, Symbol: "BTCUSD_PERP", PositionSide: "LONG", Status: "OPEN",
			EntryPrice: "50000", RemainingContracts: "10", ContractSize: "100",
		},
		MarkPrice: &mark, UnrealizedPnL: &unreal, PnLStatus: "verified",
	}
	close, err := strategy.Exit(context.Background(), coin, summary)
	if err != nil || !close {
		t.Fatalf("inverse COIN-M ROI close signal not evaluated: %v %v", close, err)
	}
	coin.Strategy = `[{"name":"exit","type":"close_long","enable":true,"code":"false"}]`
	close, err = strategy.Exit(context.Background(), coin, summary)
	if err != nil || close {
		t.Fatalf("false close rule bypassed by ROI threshold: %v %v", close, err)
	}
	coin.Strategy = "[]"
	close, err = strategy.Exit(context.Background(), coin, summary)
	if err != nil || !close {
		t.Fatalf("confirmed 5m futures fallback should close: %v %v", close, err)
	}
	strategy.Klines = func(_ context.Context, _, _ string, _ int) ([]*futures.Kline, error) {
		return []*futures.Kline{{Close: "55000"}, {Close: "50000"}}, nil
	}
	close, err = strategy.Exit(context.Background(), coin, summary)
	if err != nil || close {
		t.Fatalf("nonconfirming LONG 5m candles must keep position: %v %v", close, err)
	}
	strategy.Klines = nil
	close, err = strategy.Exit(context.Background(), coin, summary)
	if err == nil || close {
		t.Fatalf("missing 5m DAPI must not authorize exit: %v %v", close, err)
	}
}

// A strategy signal is valid only for the exact rule/indicator snapshot
// used to evaluate it. Editing those fields during an in-flight DAPI
// candle request must never authorize a stale new order.
func TestCM6RuleOrIndicatorEditedDuringEntryInvalidatesSignal(t *testing.T) {
	for _, tc := range []struct{ name, column, value string }{
		{"strategy", "strategy", `[{"name":"edited","type":"long","enable":true,"code":"true"}]`},
		{"technology", "technology", `{"ma":[]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, _, api, strat, db, now := cm6RunnerFixture(t)
			var changeErr error
			strat.onEntry = func() {
				_, changeErr = db.Raw("UPDATE delivery_symbols SET "+tc.column+"=? WHERE symbol='BTCUSD_PERP'", tc.value).Exec()
			}
			if err := r.Tick(context.Background()); err != nil {
				t.Fatal(err)
			}
			if changeErr != nil {
				t.Fatal(changeErr)
			}
			if len(api.calls) != 0 {
				t.Fatalf("stale %s signal POSTed after configuration edit: %+v", tc.name, api.calls)
			}
			strat.onEntry = nil
			*now = now.Add(11 * time.Second)
			if err := r.Tick(context.Background()); err != nil {
				t.Fatal(err)
			}
			if len(api.calls) != 1 {
				t.Fatalf("fresh %s strategy could not re-evaluate next tick: %d", tc.name, len(api.calls))
			}
		})
	}
}

package spotexecution

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"go_binance_futures/models"
	"go_binance_futures/service/spotledger"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
	"github.com/shopspring/decimal"
)

var seq atomic.Uint64

func testEngine(t *testing.T) (*Processor, *fakeExchange, *fakeGate, *spotledger.Ledger, orm.Ormer) {
	t.Helper()
	_ = orm.RegisterDriver("sqlite3", orm.DRSqlite)
	alias := fmt.Sprintf("spot_exec_064_%d", seq.Add(1))
	if err := orm.RegisterDataBase(alias, "sqlite3", filepath.Join(t.TempDir(), "spot_064.db")); err != nil {
		t.Fatal(err)
	}
	db := orm.NewOrmUsingDB(alias)
	for _, sql := range []string{
		`CREATE TABLE config(id INTEGER PRIMARY KEY,spot_quant_enable TINYINT NOT NULL)`,
		`INSERT INTO config(id,spot_quant_enable) VALUES(1,0)`,
		`CREATE TABLE spot_symbols(id INTEGER PRIMARY KEY,symbol VARCHAR(32) UNIQUE NOT NULL,spot_quant_enabled TINYINT NOT NULL)`,
		`INSERT INTO spot_symbols(id,symbol,spot_quant_enabled) VALUES(1,'BTCUSDT',0)`,
		`CREATE TABLE spot_managed_positions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,symbol VARCHAR(32) NOT NULL,
			base_asset VARCHAR(32) NOT NULL,quote_asset VARCHAR(32) NOT NULL,
			bought_qty DECIMAL(36,18) NOT NULL DEFAULT 0,
			remaining_qty DECIMAL(36,18) NOT NULL DEFAULT 0,
			avg_cost_quote DECIMAL(36,18) NOT NULL DEFAULT 0,
			realized_pnl DECIMAL(36,18) NOT NULL DEFAULT 0,
			status VARCHAR(32) NOT NULL,created_at BIGINT NOT NULL,
			updated_at BIGINT NOT NULL,closed_at BIGINT NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE spot_managed_orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,position_id BIGINT NOT NULL,
			symbol VARCHAR(32) NOT NULL,side VARCHAR(4) NOT NULL,
			order_type VARCHAR(24) NOT NULL,client_order_id VARCHAR(64) NOT NULL UNIQUE,
			exchange_order_id VARCHAR(64) NULL,
			requested_qty DECIMAL(36,18) NOT NULL,filled_qty DECIMAL(36,18) NOT NULL DEFAULT 0,
			status VARCHAR(32) NOT NULL,created_at BIGINT NOT NULL,updated_at BIGINT NOT NULL,
			CONSTRAINT uex UNIQUE(symbol,exchange_order_id)
		)`,
		`CREATE TABLE spot_trade_fills (
			id INTEGER PRIMARY KEY AUTOINCREMENT,position_id BIGINT NOT NULL,
			order_id BIGINT NOT NULL,exchange_order_id VARCHAR(64) NOT NULL,
			trade_id VARCHAR(64) NOT NULL,symbol VARCHAR(32) NOT NULL,
			side VARCHAR(4) NOT NULL,qty DECIMAL(36,18) NOT NULL,
			price DECIMAL(36,18) NOT NULL,quote_qty DECIMAL(36,18) NOT NULL,
			commission DECIMAL(36,18) NOT NULL,commission_asset VARCHAR(32) NOT NULL,
			trade_time BIGINT NOT NULL,CONSTRAINT ufill UNIQUE(symbol,exchange_order_id,trade_id)
		)`,
		`CREATE TABLE futures_managed_positions (id INTEGER PRIMARY KEY,marker TEXT NOT NULL)`,
		`INSERT INTO futures_managed_positions(id,marker) VALUES(1,'unchanged')`,
	} {
		if _, err := db.Raw(sql).Exec(); err != nil {
			t.Fatal(err)
		}
	}
	ledger, err := spotledger.New(db, "sqlite")
	if err != nil {
		t.Fatal(err)
	}
	ex := &fakeExchange{
		rules: Rules{Symbol: "BTCUSDT", Base: "BTC", Quote: "USDT", Trading: true, MarketAllowed: true,
			Step: "0.001", MinQty: "0.01", MaxQty: "100", MinNotional: "10", MaxNotional: "100000"},
		price: "100", free: map[string]string{"BTC": "50", "USDT": "2000"},
		orders: map[string]RemoteOrder{}, trades: map[string][]Trade{},
		nextID: 100, fillOnPlace: true,
	}
	gate := &fakeGate{on: true}
	p, err := NewProcessor(ex, gate, ledger)
	if err != nil {
		t.Fatal(err)
	}
	return p, ex, gate, ledger, db
}

type fakeGate struct {
	on        bool
	reads     int
	disableAt int
}

func (g *fakeGate) Enabled(_ context.Context, _ string) (bool, error) {
	g.reads++
	if g.disableAt > 0 && g.reads >= g.disableAt {
		return false, nil
	}
	return g.on, nil
}

type fakeExchange struct {
	rules                            Rules
	price                            string
	free                             map[string]string
	orders                           map[string]RemoteOrder
	trades                           map[string][]Trade
	placeCount, cancelCount, lookups int
	nextID                           int
	fillOnPlace                      bool
	placeReturnsError                bool
	placeErrorAfterAccept            bool
	lookupError                      bool
	feeAsset                         string
}

func (f *fakeExchange) Rules(_ context.Context, _ string) (Rules, error)  { return f.rules, nil }
func (f *fakeExchange) Price(_ context.Context, _ string) (string, error) { return f.price, nil }
func (f *fakeExchange) Free(_ context.Context, asset string) (string, error) {
	return f.free[asset], nil
}
func (f *fakeExchange) FindOrder(_ context.Context, _, id string) (*RemoteOrder, error) {
	f.lookups++
	if f.lookupError {
		return nil, fmt.Errorf("network timeout")
	}
	if o, ok := f.orders[id]; ok {
		clone := o
		return &clone, nil
	}
	return nil, nil
}
func (f *fakeExchange) Place(_ context.Context, req Request) (RemoteOrder, error) {
	f.placeCount++
	if f.placeReturnsError && !f.placeErrorAfterAccept {
		return RemoteOrder{}, fmt.Errorf("timeout after POST")
	}
	f.nextID++
	id := fmt.Sprintf("%d", f.nextID)
	status := "NEW"
	executed := "0"
	if f.fillOnPlace {
		status = "FILLED"
		executed = req.Quantity
	}
	r := RemoteOrder{Symbol: req.Symbol, Side: req.Side, Type: req.Type, ClientOrderID: req.ClientOrderID,
		ExchangeOrderID: id, Status: status, OrigQty: req.Quantity, ExecutedQty: executed}
	f.orders[req.ClientOrderID] = r
	if f.fillOnPlace {
		price := decimal.RequireFromString(f.price)
		qty := decimal.RequireFromString(req.Quantity)
		f.trades[id] = []Trade{{
			Symbol: req.Symbol, Side: req.Side, OrderID: id, TradeID: fmt.Sprintf("%d", f.nextID+1000),
			Qty: req.Quantity, Price: f.price, QuoteQty: qty.Mul(price).String(),
			Fee: "0", FeeAsset: f.feeAsset, Time: 10000 + int64(f.nextID),
		}}
	}
	if f.placeReturnsError {
		return RemoteOrder{}, fmt.Errorf("timeout after exchange accepted POST")
	}
	return r, nil
}
func (f *fakeExchange) Cancel(_ context.Context, _, id string) (RemoteOrder, error) {
	f.cancelCount++
	order, ok := f.orders[id]
	if !ok {
		return RemoteOrder{}, ErrNotFound
	}
	order.Status = "CANCELED"
	f.orders[id] = order
	return order, nil
}
func (f *fakeExchange) Trades(_ context.Context, _, orderID string) ([]Trade, error) {
	return append([]Trade(nil), f.trades[orderID]...), nil
}

func TestStage064DBGateAllSwitchStates(t *testing.T) {
	_, _, _, _, db := testEngine(t)
	gate := DBGate{DB: db}
	ctx := context.Background()
	assertGate := func(want bool) {
		t.Helper()
		got, err := gate.Enabled(ctx, "BTCUSDT")
		if err != nil || got != want {
			t.Fatalf("gate got=%v want=%v: %v", got, want, err)
		}
	}
	assertGate(false)
	if _, err := db.Raw("UPDATE spot_symbols SET spot_quant_enabled=1 WHERE symbol='BTCUSDT'").Exec(); err != nil {
		t.Fatal(err)
	}
	assertGate(false) // global OFF overrides a configured coin
	if _, err := db.Raw("UPDATE spot_quant_enable SET x=1").Exec(); err == nil {
		t.Fatal("unexpected table")
	}
	if _, err := db.Raw("UPDATE config SET spot_quant_enable=1 WHERE id=1").Exec(); err != nil {
		t.Fatal(err)
	}
	assertGate(true)
	if _, err := db.Raw("UPDATE spot_symbols SET spot_quant_enabled=0 WHERE symbol='BTCUSDT'").Exec(); err != nil {
		t.Fatal(err)
	}
	assertGate(false)
	if _, err := gate.Enabled(ctx, "ETHUSDT"); err == nil {
		t.Fatal("missing Spot symbol should fail closed")
	}
	if _, err := db.Raw("UPDATE spot_symbols SET spot_quant_enabled=1 WHERE symbol='BTCUSDT'").Exec(); err != nil {
		t.Fatal(err)
	}
	if _, err := gate.Enabled(ctx, "ETHUSDT"); err == nil {
		t.Fatal("unregistered Spot symbol allowed")
	}
}

func TestStage064FakeBuySellAndWalletIsolation(t *testing.T) {
	p, ex, _, l, db := testEngine(t)
	ctx := context.Background()
	buy, err := p.ExecuteBuy(ctx, "BTCUSDT", "100", "buy_064_1")
	if err != nil {
		t.Fatal(err)
	}
	if ex.placeCount != 1 || buy.Status != "FILLED" {
		t.Fatalf("buy status=%+v places=%d", buy, ex.placeCount)
	}
	view, err := l.Accounting(buy.PositionID, "100")
	if err != nil {
		t.Fatal(err)
	}
	qty := decimal.RequireFromString(buy.RequestedQty)
	if !decimal.RequireFromString(view.Position.RemainingQty).Equal(qty) {
		t.Fatalf("BUY Fill inventory not exact: %+v", view)
	}
	// Wallet has 50 BTC, most unrelated manual holdings. SELL may only use
	// the fraction explicitly bought by this strategy's verified Fill.
	sell, err := p.ExecuteSell(ctx, buy.PositionID, "sell_064_1")
	if err != nil {
		t.Fatal(err)
	}
	if ex.placeCount != 2 || sell.Side != "SELL" {
		t.Fatalf("sell %+v places=%d", sell, ex.placeCount)
	}
	sellQty := decimal.RequireFromString(sell.RequestedQty)
	if !sellQty.Equal(qty) {
		t.Fatalf("SELL must close exact verified inventory: buy=%s sell=%s", qty, sellQty)
	}
	closed, err := l.Accounting(buy.PositionID, "100")
	if err != nil {
		t.Fatal(err)
	}
	if closed.Position.Status != "CLOSED" || !decimal.RequireFromString(closed.Position.RemainingQty).IsZero() {
		t.Fatalf("SELL did not close full strategy position: %+v", closed.Position)
	}
	if sellQty.GreaterThan(decimal.RequireFromString("1")) {
		t.Fatal("strategy sold more than budget")
	}
	again, err := p.ExecuteBuy(ctx, "BTCUSDT", "100", "buy_064_1")
	if err != nil || again.ID != buy.ID || ex.placeCount != 2 {
		t.Fatalf("restart duplicate sent another BUY: %+v %v", again, err)
	}
	again, err = p.ExecuteSell(ctx, buy.PositionID, "sell_064_1")
	if err != nil || again.ID != sell.ID || ex.placeCount != 2 {
		t.Fatalf("restart duplicate sent another SELL: %+v %v", again, err)
	}
	var marker string
	if err := db.Raw("SELECT marker FROM futures_managed_positions WHERE id=1").QueryRow(&marker); err != nil || marker != "unchanged" {
		t.Fatalf("Futures isolation damaged: %q %v", marker, err)
	}
}

func TestStage064GateAndFreeBalanceFailClosed(t *testing.T) {
	for _, tc := range []struct {
		name              string
		global            bool
		disableAt         int
		quoteFree, budget string
		wantLocal         bool
	}{
		{"global_off", false, 0, "1000", "100", false},
		{"disabled_just_before_exchange", true, 3, "1000", "100", true},
		{"low_quote_balance", true, 0, "5", "100", false},
		{"below_notional", true, 0, "1000", "2", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, ex, g, l, _ := testEngine(t)
			g.on = tc.global
			g.disableAt = tc.disableAt
			ex.free["USDT"] = tc.quoteFree
			_, err := p.ExecuteBuy(context.Background(), "BTCUSDT", tc.budget, "guard064")
			if err == nil {
				t.Fatalf("gate allowed order: %s", tc.name)
			}
			if ex.placeCount != 0 {
				t.Fatalf("exchange side effect before gate: %d", ex.placeCount)
			}
			if tc.wantLocal {
				order, err := l.FindOrderByClientID("guard064")
				if err != nil || order.Status != "NEW" {
					t.Fatalf("expected blocked local order, got %+v %v", order, err)
				}
			}
		})
	}
}

func TestStage064TimeoutNoDuplicateAndRestartRecovery(t *testing.T) {
	p, ex, _, l, _ := testEngine(t)
	ctx := context.Background()
	ex.placeReturnsError = true
	// Exchange accepted POST, SDK timed out: lookup by durable client ID
	// discovers and reconciles the original order, never POSTs twice.
	ex.placeErrorAfterAccept = true
	first, err := p.ExecuteBuy(ctx, "BTCUSDT", "100", "ambiguous064")
	if err != nil || first.Status != "FILLED" || ex.placeCount != 1 {
		t.Fatalf("accepted timeout recovery %+v %v, places=%d", first, err, ex.placeCount)
	}
	p2, err := NewProcessor(ex, &fakeGate{on: true}, l)
	if err != nil {
		t.Fatal(err)
	}
	second, err := p2.ExecuteBuy(ctx, "BTCUSDT", "100", "ambiguous064")
	if err != nil || second.ID != first.ID || ex.placeCount != 1 {
		t.Fatalf("retry duplicated trade %+v %v %d", second, err, ex.placeCount)
	}
	// A definite local NEW order with failed POST remains uncertain, and a
	// new processor may reconcile but NEVER repost with the same identity.
	ex2 := &fakeExchange{rules: ex.rules, price: ex.price, free: ex.free, orders: map[string]RemoteOrder{}, trades: map[string][]Trade{}, placeReturnsError: true}
	_, _, _, l2, _ := testEngine(t)
	q, err := NewProcessor(ex2, &fakeGate{on: true}, l2)
	if err != nil {
		t.Fatal(err)
	}
	_, err = q.ExecuteBuy(ctx, "BTCUSDT", "100", "noresponse064")
	if !errors.Is(err, ErrUncertain) || ex2.placeCount != 1 {
		t.Fatalf("network unknown incorrectly treated as safe: %v", err)
	}
	_, err = q.ExecuteBuy(ctx, "BTCUSDT", "100", "noresponse064")
	if !errors.Is(err, ErrNotFound) || ex2.placeCount != 1 {
		t.Fatalf("persisted absent intent resent: %v count=%d", err, ex2.placeCount)
	}
}

func TestStage064CancelCanRunWithGlobalOff(t *testing.T) {
	p, ex, g, l, _ := testEngine(t)
	ex.fillOnPlace = false
	buy, err := p.ExecuteBuy(context.Background(), "BTCUSDT", "100", "cancel064")
	if err != nil || buy.Status != "NEW" {
		t.Fatalf("new BUY %+v %v", buy, err)
	}
	g.on = false
	canceled, err := p.Cancel(context.Background(), buy.ID)
	if err != nil || canceled.Status != "CANCELED" || ex.cancelCount != 1 {
		t.Fatalf("disable blocked cancel or status mismatch: %+v %v", canceled, err)
	}
	fills, err := l.ListFills(buy.PositionID)
	if err != nil || len(fills) != 0 {
		t.Fatalf("cancel forged trade: %+v %v", fills, err)
	}
	if _, err = p.ExecuteBuy(context.Background(), "BTCUSDT", "100", "newbuy064"); !errors.Is(err, ErrDisabled) {
		t.Fatalf("global OFF allowed new BUY: %v", err)
	}
}

func TestStage064IncompleteTradesBlockNewOrders(t *testing.T) {
	p, ex, _, _, _ := testEngine(t)
	ex.fillOnPlace = false
	buy, err := p.ExecuteBuy(context.Background(), "BTCUSDT", "100", "partial064")
	if err != nil {
		t.Fatal(err)
	}
	original := ex.orders["partial064"]
	original.Status = "FILLED"
	original.ExecutedQty = buy.RequestedQty
	ex.orders["partial064"] = original
	ex.trades[original.ExchangeOrderID] = nil
	_, err = p.Reconcile(context.Background(), buy.ID)
	if !errors.Is(err, ErrNeedsReview) {
		t.Fatalf("partial myTrades treated as reconciled: %v", err)
	}
	if _, err = p.ExecuteSell(context.Background(), buy.PositionID, "blocked064"); err == nil {
		t.Fatal("SELL allowed while all BUY fills are missing")
	}
	// Later myTrades recovery restores all actual fills.
	price := decimal.RequireFromString(ex.price)
	qty := decimal.RequireFromString(buy.RequestedQty)
	ex.trades[original.ExchangeOrderID] = []Trade{{Symbol: "BTCUSDT", Side: "BUY", OrderID: original.ExchangeOrderID,
		TradeID: "881001", Qty: buy.RequestedQty, Price: ex.price, QuoteQty: qty.Mul(price).String(), Fee: "0", Time: 1234}}
	restored, err := p.Reconcile(context.Background(), buy.ID)
	if err != nil || restored.Status != "FILLED" {
		t.Fatalf("myTrades recovery not completed: %+v %v", restored, err)
	}
}

func TestStage064PlanQtyExactPrecisionAndFeeReserve(t *testing.T) {
	r := Rules{Symbol: "BTCUSDT", Base: "BTC", Quote: "USDT", Trading: true, MarketAllowed: true,
		Step: "0.001", MinQty: "0.01", MaxQty: "100", MinNotional: "10", MaxNotional: "1000"}
	q, err := planBuy(r, "100", "100", "101")
	if err != nil || q != "0.990" { // fixed() strips trailing zeros; see value below
		if err != nil || q != "0.99" {
			t.Fatalf("budget plan %q: %v", q, err)
		}
	}
	s, err := planSell(r, "100", "0.99", "50")
	if err != nil {
		t.Fatal(err)
	}
	if s != "0.99" {
		t.Fatalf("full inventory SELL must close, got %s", s)
	}

	for _, tc := range []struct{ budget, free string }{
		{"1", "100"}, {"100", "5"}, {"0", "100"}, {"10000000000000000000", "10000000000000000000"},
	} {
		if _, err := planBuy(r, "100", tc.budget, tc.free); err == nil {
			t.Errorf("invalid BUY plan accepted %+v", tc)
		}
	}
}

func TestStage064UserStreamOnlyManagedAndDeduplicated(t *testing.T) {
	p, ex, _, ledger, _ := testEngine(t)
	ex.fillOnPlace = false
	buy, err := p.ExecuteBuy(context.Background(), "BTCUSDT", "100", "ws_buy_064")
	if err != nil || buy.Status != "NEW" {
		t.Fatalf("create pending BUY: %+v %v", buy, err)
	}
	ignored, err := p.HandleExecutionReport(context.Background(), ExecutionReport{
		Symbol: "BTCUSDT", ClientOrderID: "manual_wallet_999", ExchangeOrderID: "9999",
		Side: "BUY", Status: "FILLED", ExecutionType: "TRADE", TradeID: "12",
		LastQty: "1", LastPrice: "100", LastQuoteQty: "100", Fee: "0", TradeTime: 1000,
	})
	if err != nil || ignored {
		t.Fatalf("manual wallet order was claimed: ignored=%v err=%v", ignored, err)
	}
	order, err := ledger.GetOrder(buy.ID)
	if err != nil || order.FilledQty != "0" {
		t.Fatalf("manual event modified Spot ledger: %+v %v", order, err)
	}
	id := ex.orders["ws_buy_064"].ExchangeOrderID
	event := ExecutionReport{
		Symbol: "BTCUSDT", ClientOrderID: "ws_buy_064", ExchangeOrderID: id,
		Side: "BUY", Status: "PARTIALLY_FILLED", ExecutionType: "TRADE",
		TradeID: "111", LastQty: "0.4", LastPrice: "100", LastQuoteQty: "40",
		Fee: "0.0004", FeeAsset: "BTC", TradeTime: 1000,
	}
	managed, err := p.HandleExecutionReport(context.Background(), event)
	if err != nil || !managed {
		t.Fatalf("valid managed WS trade %+v %v", managed, err)
	}
	_, err = p.HandleExecutionReport(context.Background(), event)
	if err != nil {
		t.Fatal(err)
	}
	trades, err := ledger.ListFills(buy.PositionID)
	if err != nil || len(trades) != 1 {
		t.Fatalf("WS duplicated immutable Fill rows: %d %v", len(trades), err)
	}
	remaining := decimal.RequireFromString(buy.RequestedQty).Sub(decimal.RequireFromString("0.4")).String()
	event.TradeID = "112"
	event.LastQty = remaining
	event.LastQuoteQty = decimal.RequireFromString(remaining).Mul(decimal.RequireFromString("100")).String()
	event.Fee = "0"
	event.FeeAsset = ""
	event.TradeTime = 2000
	event.Status = "FILLED"
	_, err = p.HandleExecutionReport(context.Background(), event)
	if err != nil {
		t.Fatal(err)
	}
	local, err := ledger.GetOrder(buy.ID)
	if err != nil || local.Status != "FILLED" {
		t.Fatalf("WS complete order %+v %v", local, err)
	}
	fills, err := ledger.ListFills(buy.PositionID)
	if err != nil || len(fills) != 2 {
		t.Fatalf("WS Fill count %d err=%v", len(fills), err)
	}
	if ex.placeCount != 1 {
		t.Fatal("WS ingestion triggered new order")
	}
}

func TestStage064RecoveryAfterRestartAndUnknownOrder(t *testing.T) {
	p, ex, _, ledger, _ := testEngine(t)
	ex.fillOnPlace = false
	buy, err := p.ExecuteBuy(context.Background(), "BTCUSDT", "100", "recover_064")
	if err != nil {
		t.Fatal(err)
	}
	restarted, err := NewProcessor(ex, &fakeGate{on: false}, ledger)
	if err != nil {
		t.Fatal(err)
	}
	if err = restarted.RecoverOutstanding(context.Background()); err != nil {
		t.Fatalf("read-only restore with switches OFF must work: %v", err)
	}
	remote := ex.orders["recover_064"]
	remote.Status = "FILLED"
	remote.ExecutedQty = buy.RequestedQty
	ex.orders["recover_064"] = remote
	ex.trades[remote.ExchangeOrderID] = []Trade{{
		Symbol: "BTCUSDT", Side: "BUY", OrderID: remote.ExchangeOrderID, TradeID: "41001",
		Qty: buy.RequestedQty, Price: "100",
		QuoteQty: decimal.RequireFromString(buy.RequestedQty).Mul(decimal.RequireFromString("100")).String(),
		Fee:      "0", Time: 5000,
	}}
	if err = restarted.RecoverOutstanding(context.Background()); err != nil {
		t.Fatalf("recovery after WS disconnect: %v", err)
	}
	local, err := ledger.GetOrder(buy.ID)
	if err != nil || local.Status != "FILLED" {
		t.Fatalf("restored Spot order: %+v %v", local, err)
	}
	if ex.placeCount != 1 {
		t.Fatalf("recovery POSTed order again: %d", ex.placeCount)
	}
	// If the user manually removed an exchange order, a persisted local
	// intent remains uncertain, and recovery never silently reposts it.
	delete(ex.orders, "recover_064")
	// Fully FILLED orders already backed by all immutable myTrades don't
	// require repeated REST lookups after Binance archive expiration.
	if err = restarted.RecoverOutstanding(context.Background()); err != nil {
		t.Fatalf("fully reconciled historical order should not block: %v", err)
	}
	if ex.placeCount != 1 {
		t.Fatal("historical order triggered duplicate POST")
	}
}

func TestStage064WalletShortfallBlocksSELLWithoutManualFallback(t *testing.T) {
	p, ex, _, ledger, _ := testEngine(t)
	buy, err := p.ExecuteBuy(context.Background(), "BTCUSDT", "100", "walletshort_buy")
	if err != nil {
		t.Fatal(err)
	}
	// Verified strategy holds ~0.99 BTC, but wallet free is only 0.3 BTC;
	// cannot use whatever manual BTC remains to fill a smaller SELL.
	ex.free["BTC"] = "0.3"
	_, err = p.ExecuteSell(context.Background(), buy.PositionID, "walletshort_sell")
	if !errors.Is(err, ErrNeedsReview) {
		t.Fatalf("manual fallback not rejected: %v", err)
	}
	if ex.placeCount != 1 {
		t.Fatalf("wallet shortfall placed a SELL: %d", ex.placeCount)
	}
	if _, err := ledger.FindOrderByClientID("walletshort_sell"); !errors.Is(err, orm.ErrNoRows) {
		t.Fatalf("short wallet created a local SELL: %v", err)
	}
}

func TestStage064ConflictingRemoteAndTradesAreRejected(t *testing.T) {
	for _, tc := range []string{"remote_symbol", "remote_qty", "trade_symbol", "truncated_trades", "missing_mytrades"} {
		t.Run(tc, func(t *testing.T) {
			p, ex, _, l, _ := testEngine(t)
			ex.fillOnPlace = false
			local, err := p.ExecuteBuy(context.Background(), "BTCUSDT", "100", "mismatch_064")
			if err != nil {
				t.Fatal(err)
			}
			remote := ex.orders["mismatch_064"]
			switch tc {
			case "remote_symbol":
				remote.Symbol = "ETHUSDT"
			case "remote_qty":
				remote.OrigQty = "999"
			case "trade_symbol":
				remote.Status = "FILLED"
				remote.ExecutedQty = local.RequestedQty
				ex.trades[remote.ExchangeOrderID] = []Trade{{Symbol: "ETHUSDT", Side: "BUY", OrderID: remote.ExchangeOrderID, TradeID: "1", Qty: local.RequestedQty, Price: "100", QuoteQty: decimal.RequireFromString(local.RequestedQty).Mul(decimal.RequireFromString("100")).String(), Fee: "0", Time: 1234}}
			case "truncated_trades":
				remote.Status = "FILLED"
				remote.ExecutedQty = local.RequestedQty
				list := make([]Trade, 1000)
				for i := range list {
					list[i] = Trade{Symbol: "BTCUSDT", OrderID: remote.ExchangeOrderID, Side: "BUY", TradeID: fmt.Sprint(i + 1)}
				}
				ex.trades[remote.ExchangeOrderID] = list
			case "missing_mytrades":
				remote.Status = "FILLED"
				remote.ExecutedQty = local.RequestedQty
			}
			ex.orders["mismatch_064"] = remote
			_, err = p.Reconcile(context.Background(), local.ID)
			if !errors.Is(err, ErrNeedsReview) {
				t.Fatalf("bad remote %s not blocked: %v", tc, err)
			}
			if ex.placeCount != 1 {
				t.Fatal("invalid response caused another BUY")
			}
			fills, err := l.ListFills(local.PositionID)
			if err != nil || len(fills) != 0 {
				t.Fatalf("invalid external trades recorded: %d %v", len(fills), err)
			}
		})
	}
}

// 06-4-F1: successful full SELL closes the position, unlocks a fresh BUY
// cycle, and never fabricates a completed sell from a truncated quantity.
func TestReview064FullSellClosesCycleAndNewBuy(t *testing.T) {
	p, ex, _, ledger, _ := testEngine(t)
	buy, err := p.ExecuteBuy(context.Background(), "BTCUSDT", "100", "review064_buy1")
	if err != nil {
		t.Fatal(err)
	}
	sell, err := p.ExecuteSell(context.Background(), buy.PositionID, "review064_sell1")
	if err != nil {
		t.Fatal(err)
	}
	if sell.RequestedQty != buy.RequestedQty {
		t.Fatalf("SELL dust left behind: buy=%s sell=%s", buy.RequestedQty, sell.RequestedQty)
	}
	view, err := ledger.Accounting(buy.PositionID, "110")
	if err != nil {
		t.Fatal(err)
	}
	if view.Position.Status != "CLOSED" || view.Position.RemainingQty != "0.000000000000000000" {
		t.Fatalf("full SELL never closed inventory: %+v", view.Position)
	}
	if _, err = p.ExecuteBuy(context.Background(), "BTCUSDT", "100", "review064_buy2"); err != nil {
		t.Fatalf("cannot begin fresh buy after full SELL: %v", err)
	}
	if ex.placeCount != 3 {
		t.Fatalf("wrong BUY/SELL/new BUY count %d", ex.placeCount)
	}
}
func TestReview064BaseFeeDustIsExplicitReviewNotFakeClose(t *testing.T) {
	r := Rules{Symbol: "BTCUSDT", Base: "BTC", Quote: "USDT", Trading: true, MarketAllowed: true,
		Step: "0.001", MinQty: "0.01", MaxQty: "100", MinNotional: "10"}
	_, err := planSell(r, "100", "0.9896", "1")
	if !errors.Is(err, ErrNeedsReview) {
		t.Fatalf("base fee remainder should require review: %v", err)
	}
}
func TestReview064ExplicitAbsentOrderRecovery(t *testing.T) {
	p, ex, _, ledger, db := testEngine(t)
	ex.placeReturnsError = true
	_, err := p.ExecuteBuy(context.Background(), "BTCUSDT", "100", "unaccepted064")
	if !errors.Is(err, ErrUncertain) {
		t.Fatalf("expected ambiguous POST: %v", err)
	}
	order, err := ledger.FindOrderByClientID("unaccepted064")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = p.ConfirmNeverAccepted(context.Background(), order.ID, false); !errors.Is(err, ErrNeedsReview) {
		t.Fatalf("no operator approval rejected: %v", err)
	}
	if _, err = p.ConfirmNeverAccepted(context.Background(), order.ID, true); !errors.Is(err, ErrUncertain) {
		t.Fatalf("early confirmation rejected: %v", err)
	}
	if _, err = db.Raw("UPDATE spot_managed_orders SET created_at=created_at-1860000 WHERE id=?", order.ID).Exec(); err != nil {
		t.Fatal(err)
	}
	ex.lookupError = true
	if _, err = p.ConfirmNeverAccepted(context.Background(), order.ID, true); !errors.Is(err, ErrUncertain) {
		t.Fatalf("network error mistaken for proof of absence: %v", err)
	}
	ex.lookupError = false
	confirmed, err := p.ConfirmNeverAccepted(context.Background(), order.ID, true)
	if err != nil || confirmed.Status != "REJECTED" {
		t.Fatalf("operator-confirmed absent order not resolved: %+v %v", confirmed, err)
	}
	position, err := ledger.GetPosition(order.PositionID)
	if err != nil || position.Status != "CLOSED" {
		t.Fatalf("empty cycle not released: %+v %v", position, err)
	}
	if _, err = p.ExecuteBuy(context.Background(), "BTCUSDT", "100", "next064"); !errors.Is(err, ErrUncertain) {
		t.Fatalf("new independent cycle should reach exchange once: %v", err)
	}
	if ex.placeCount != 2 {
		t.Fatalf("duplicate original exchange order sent: %d", ex.placeCount)
	}
	if _, err = p.ConfirmNeverAccepted(context.Background(), order.ID, true); !errors.Is(err, ErrNeedsReview) {
		t.Fatalf("terminal confirmed order can be re-marked: %v", err)
	}
	// Older client_id can never be reused even after explicit rejection.
	if _, err = p.ExecuteBuy(context.Background(), "BTCUSDT", "100", "unaccepted064"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("rejected original client ID must not be resent: %v", err)
	}
	if ex.placeCount != 2 {
		t.Fatalf("rejected client ID unexpectedly replayed: %d", ex.placeCount)
	}
	// An unexpected late Fill from that rejected order must not reopen
	// the empty cycle while the next live strategy cycle already exists.
	_, lateErr := ledger.RecordFill(spotledger.FillEvent{
		OrderID: order.ID, Symbol: "BTCUSDT", ExchangeOrderID: "12345", TradeID: "67890",
		Side: "BUY", Qty: "0.1", Price: "100", QuoteQty: "10", Commission: "0", TradeTime: 1000,
	})
	if !errors.Is(lateErr, spotledger.ErrNeedsReview) {
		t.Fatalf("late trade silently reopened explicitly abandoned cycle: %v", lateErr)
	}
}
func TestReview064AbsentOrderProofCannotOverrideAcceptedFill(t *testing.T) {
	p, ex, _, ledger, db := testEngine(t)
	buy, err := p.ExecuteBuy(context.Background(), "BTCUSDT", "100", "accepted064")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Raw("UPDATE spot_managed_orders SET created_at=created_at-1860000 WHERE id=?", buy.ID).Exec(); err != nil {
		t.Fatal(err)
	}
	delete(ex.orders, "accepted064")
	_, err = p.ConfirmNeverAccepted(context.Background(), buy.ID, true)
	if !errors.Is(err, ErrNeedsReview) {
		t.Fatalf("accepted order/Fill must be immutable: %v", err)
	}
	order, err := ledger.GetOrder(buy.ID)
	if err != nil || order.Status != "FILLED" {
		t.Fatalf("accepted order mutated: %+v %v", order, err)
	}
}

// F5: globally disabled NEW SELL must not spend any Binance REST lookup
// weight. Explicit Reconcile and Cancel remain allowed when OFF.
func TestReview064GateOffBeforeAnySellREST(t *testing.T) {
	p, ex, gate, _, _ := testEngine(t)
	buy, err := p.ExecuteBuy(context.Background(), "BTCUSDT", "100", "gate_sell_buy")
	if err != nil {
		t.Fatal(err)
	}
	gate.on = false
	before := ex.lookups
	_, err = p.ExecuteSell(context.Background(), buy.PositionID, "gate_sell_off")
	if !errors.Is(err, ErrDisabled) {
		t.Fatalf("global OFF did not block new SELL: %v", err)
	}
	if ex.lookups != before {
		t.Fatalf("global OFF did %d unnecessary REST order lookups", ex.lookups-before)
	}
	if ex.placeCount != 1 {
		t.Fatalf("global OFF submitted SELL %d", ex.placeCount)
	}
	_, err = p.Reconcile(context.Background(), buy.ID)
	if err != nil {
		t.Fatalf("explicit OFF-time reconciliation should work: %v", err)
	}
}

// F6: old verified FILLED orders must not be re-fetched merely because a
// fresh SELL is requested. The exchange can expire old order history.
func TestReview064OldFullyReconciledBUYDoesNotBlockSELL(t *testing.T) {
	p, ex, _, _, _ := testEngine(t)
	buy, err := p.ExecuteBuy(context.Background(), "BTCUSDT", "100", "old064_buy")
	if err != nil {
		t.Fatal(err)
	}
	delete(ex.orders, "old064_buy")
	before := ex.lookups
	sell, err := p.ExecuteSell(context.Background(), buy.PositionID, "old064_sell")
	if err != nil {
		t.Fatalf("old full BUY lookup blocked new SELL: %v", err)
	}
	if sell.Status != "FILLED" || ex.lookups-before != 1 {
		t.Fatalf("unexpected history lookups=%d sell=%+v", ex.lookups-before, sell)
	}
}
func TestReview064RemoteCheckScope(t *testing.T) {
	nowMS := time.Now().UnixMilli()
	for _, tc := range []struct {
		status, requested, filled string
		updated                   int64
		want                      bool
	}{
		{"NEW", "1", "0", nowMS, true},
		{"PENDING_CANCEL", "1", "0.5", nowMS, true},
		{"PARTIALLY_FILLED", "1", "0.5", nowMS, true},
		{"FILLED", "1", "1", nowMS, false},
		{"FILLED", "1", "0.5", nowMS, true},
		{"CANCELED", "1", "0.5", nowMS, true},
		{"CANCELED", "1", "0.5", nowMS - int64(48*time.Hour/time.Millisecond), false},
		{"EXPIRED", "1", "0.5", nowMS - int64(48*time.Hour/time.Millisecond), false},
		{"REJECTED", "1", "0", nowMS, false},
	} {
		got := needsRemoteCheck(models.SpotManagedOrder{Status: tc.status, RequestedQty: tc.requested, FilledQty: tc.filled, UpdatedAt: tc.updated})
		if got != tc.want {
			t.Errorf("%+v: got %v want %v", tc, got, tc.want)
		}
	}
}

func TestReview064MissingFeeAssetFailsClosedAndZeroFeeWorks(t *testing.T) {
	p, ex, _, ledger, _ := testEngine(t)
	ex.fillOnPlace = false
	buy, err := p.ExecuteBuy(context.Background(), "BTCUSDT", "100", "fee064_buy")
	if err != nil {
		t.Fatal(err)
	}
	remote := ex.orders["fee064_buy"]
	good := ExecutionReport{
		Symbol: "BTCUSDT", ClientOrderID: "fee064_buy", ExchangeOrderID: remote.ExchangeOrderID,
		Side: "BUY", Status: "PARTIALLY_FILLED", ExecutionType: "TRADE",
		TradeID: "9991", LastQty: "0.4", LastPrice: "100", LastQuoteQty: "40", Fee: "0.00000000", FeeAsset: "", TradeTime: 1234,
	}
	handled, err := p.HandleExecutionReport(context.Background(), good)
	if err != nil || !handled {
		t.Fatalf("zero commission without asset should be valid: %v", err)
	}
	bad := good
	bad.TradeID = "9992"
	bad.Fee = "0.01"
	bad.FeeAsset = ""
	if _, err = p.HandleExecutionReport(context.Background(), bad); !errors.Is(err, ErrNeedsReview) {
		t.Fatalf("positive fee without asset was accepted: %v", err)
	}
	fills, err := ledger.ListFills(buy.PositionID)
	if err != nil || len(fills) != 1 {
		t.Fatalf("invalid fee altered ledger: %d %v", len(fills), err)
	}
}

package spotledger

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
	"github.com/shopspring/decimal"
)

var spotLedgerFixtureSeq atomic.Uint64

func fixture(t *testing.T) (*Ledger, string) {
	t.Helper()
	_ = orm.RegisterDriver("sqlite3", orm.DRSqlite)
	alias := fmt.Sprintf("spot_ledger_063_%d", spotLedgerFixtureSeq.Add(1))
	if err := orm.RegisterDataBase(alias, "sqlite3", filepath.Join(t.TempDir(), "ledger.db")); err != nil {
		t.Fatal(err)
	}
	o := orm.NewOrmUsingDB(alias)
	for _, ddl := range []string{
		`CREATE TABLE spot_managed_positions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			symbol VARCHAR(32) NOT NULL,
			base_asset VARCHAR(32) NOT NULL,
			quote_asset VARCHAR(32) NOT NULL,
			bought_qty DECIMAL(36,18) NOT NULL DEFAULT 0,
			remaining_qty DECIMAL(36,18) NOT NULL DEFAULT 0,
			avg_cost_quote DECIMAL(36,18) NOT NULL DEFAULT 0,
			realized_pnl DECIMAL(36,18) NOT NULL DEFAULT 0,
			status VARCHAR(32) NOT NULL,
			created_at BIGINT NOT NULL, updated_at BIGINT NOT NULL,
			closed_at BIGINT NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE spot_managed_orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			position_id BIGINT NOT NULL,
			symbol VARCHAR(32) NOT NULL,side VARCHAR(4) NOT NULL,
			order_type VARCHAR(24) NOT NULL,
			client_order_id VARCHAR(64) NOT NULL UNIQUE,
			exchange_order_id VARCHAR(64) NULL,
			requested_qty DECIMAL(36,18) NOT NULL DEFAULT 0,
			filled_qty DECIMAL(36,18) NOT NULL DEFAULT 0,
			status VARCHAR(32) NOT NULL,created_at BIGINT NOT NULL,updated_at BIGINT NOT NULL,
			CONSTRAINT uq_spot_orders_exchange UNIQUE (symbol,exchange_order_id)
		)`,
		`CREATE TABLE spot_trade_fills (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			position_id BIGINT NOT NULL,order_id BIGINT NOT NULL,
			exchange_order_id VARCHAR(64) NOT NULL,trade_id VARCHAR(64) NOT NULL,
			symbol VARCHAR(32) NOT NULL,side VARCHAR(4) NOT NULL,
			qty DECIMAL(36,18) NOT NULL,price DECIMAL(36,18) NOT NULL,
			quote_qty DECIMAL(36,18) NOT NULL,
			commission DECIMAL(36,18) NOT NULL,commission_asset VARCHAR(32) NOT NULL,
			trade_time BIGINT NOT NULL,
			CONSTRAINT uq_spot_fill_exchange UNIQUE (symbol,exchange_order_id,trade_id)
		)`,
		`CREATE TABLE spot_strategy_templates(id INTEGER PRIMARY KEY,name TEXT)`,
		`CREATE TABLE futures_managed_positions(id INTEGER PRIMARY KEY, marker VARCHAR(64))`,
		`CREATE TABLE futures_managed_orders(id INTEGER PRIMARY KEY, marker VARCHAR(64))`,
		`CREATE TABLE futures_trade_fills(id INTEGER PRIMARY KEY, marker VARCHAR(64))`,
		`INSERT INTO futures_managed_positions(id,marker) VALUES(1,'futures-untouched')`,
		`INSERT INTO futures_managed_orders(id,marker) VALUES(1,'futures-untouched')`,
		`INSERT INTO futures_trade_fills(id,marker) VALUES(1,'futures-untouched')`,
	} {
		if _, err := o.Raw(ddl).Exec(); err != nil {
			t.Fatalf("fixture SQL: %v", err)
		}
	}
	l, err := New(o, "sqlite")
	if err != nil {
		t.Fatal(err)
	}
	return l, alias
}

func createTestOrder(t *testing.T, l *Ledger, pID int64, side, clientID, qty string) int64 {
	t.Helper()
	o, existing, err := l.CreateOrder(OrderIntent{PositionID: pID, Symbol: "BTCUSDT", Side: side, Type: "MARKET", ClientOrderID: clientID, RequestedQty: qty})
	if err != nil || existing {
		t.Fatalf("create %s order: %+v duplicate=%v err=%v", side, o, existing, err)
	}
	return o.ID
}

func event(orderID int64, ex, trade, side, qty, price, quote, fee, asset string, time int64) FillEvent {
	return FillEvent{OrderID: orderID, Symbol: "BTCUSDT", ExchangeOrderID: ex, TradeID: trade,
		Side: side, Qty: qty, Price: price, QuoteQty: quote, Commission: fee, CommissionAsset: asset, TradeTime: time}
}

func decimalClose(t *testing.T, actual, want string) {
	t.Helper()
	a, err := decimal.NewFromString(actual)
	if err != nil {
		t.Fatal(err)
	}
	b, err := decimal.NewFromString(want)
	if err != nil {
		t.Fatal(err)
	}
	if a.Sub(b).Abs().GreaterThan(decimal.RequireFromString("0.000000000001")) {
		t.Fatalf("decimal mismatch: got %s want %s", actual, want)
	}
}
func TestStage063SpotPartialFillsFeesAndFullLifecycle(t *testing.T) {
	l, alias := fixture(t)
	p, err := l.CreatePosition("BTCUSDT", "BTC", "USDT")
	if err != nil || p.Status != "PENDING_BUY" {
		t.Fatalf("create position: %+v %v", p, err)
	}
	if _, err = l.CreatePosition("BTCUSDT", "BTC", "USDT"); err == nil {
		t.Fatal("multiple open strategy cycles created")
	}
	buyID := createTestOrder(t, l, p.ID, "BUY", "spot-063-buy-1", "1.5")
	if _, existing, err := l.CreateOrder(OrderIntent{PositionID: p.ID, Symbol: "BTCUSDT", Side: "BUY", Type: "MARKET", ClientOrderID: "spot-063-buy-1", RequestedQty: "1.50000000"}); err != nil || !existing {
		t.Fatalf("client idempotency: existing=%v err=%v", existing, err)
	}
	if _, _, err = l.CreateOrder(OrderIntent{PositionID: p.ID, Symbol: "BTCUSDT", Side: "SELL", Type: "MARKET", ClientOrderID: "illegal-sell", RequestedQty: "1"}); err == nil {
		t.Fatal("SELL without confirmed BUY")
	}
	if _, err = l.BindExchangeOrderID(buyID, "10001"); err != nil {
		t.Fatal(err)
	}
	first := event(buyID, "10001", "50001", "BUY", "1", "100", "100", "0.01", "BTC", 1000)
	applied, err := l.RecordFill(first)
	if err != nil || !applied.Applied {
		t.Fatalf("first partial BUY: %+v %v", applied, err)
	}
	if applied.Order.Status != "PARTIALLY_FILLED" {
		t.Fatalf("partial status %+v", applied.Order)
	}
	decimalClose(t, applied.Position.RemainingQty, "0.99")
	if !strings.HasPrefix(applied.Position.AvgCostQuote, "101.0101") {
		t.Fatalf("base fee not accounted: %s", applied.Position.AvgCostQuote)
	}
	duplicate, err := l.RecordFill(first)
	if err != nil || duplicate.Applied {
		t.Fatalf("duplicate trade id recorded twice: %+v %v", duplicate, err)
	}
	conflict := first
	conflict.QuoteQty = "101"
	if _, err = l.RecordFill(conflict); err == nil {
		t.Fatal("conflicting duplicate payload accepted")
	}
	second := event(buyID, "10001", "50002", "BUY", "0.5", "110", "55", "0.1", "USDT", 2000)
	r, err := l.RecordFill(second)
	if err != nil {
		t.Fatal(err)
	}
	if r.Order.Status != "FILLED" {
		t.Fatalf("fully bought order: %+v", r.Order)
	}
	decimalClose(t, r.Position.BoughtQty, "1.49")
	decimalClose(t, r.Position.RemainingQty, "1.49")
	decimalClose(t, r.Position.AvgCostQuote, decimal.RequireFromString("155.1").DivRound(decimal.RequireFromString("1.49"), 18).String())
	if _, _, err = l.CreateOrder(OrderIntent{PositionID: p.ID, Symbol: "BTCUSDT", Side: "SELL", Type: "MARKET", ClientOrderID: "too-much-sell", RequestedQty: "1.5"}); err == nil {
		t.Fatal("SELL attempted to claim manual/external inventory")
	}
	sellID := createTestOrder(t, l, p.ID, "SELL", "spot-063-sell-1", "1")
	if _, err = l.BindExchangeOrderID(sellID, "10002"); err != nil {
		t.Fatal(err)
	}
	sell1 := event(sellID, "10002", "60001", "SELL", "0.6", "120", "72", "0.005", "BTC", 3000)
	r, err = l.RecordFill(sell1)
	if err != nil {
		t.Fatal(err)
	}
	if r.Order.Status != "PARTIALLY_FILLED" {
		t.Fatalf("partial sell: %+v", r.Order)
	}
	decimalClose(t, r.Position.RemainingQty, "0.885")
	sell2 := event(sellID, "10002", "60002", "SELL", "0.4", "125", "50", "0.005", "BTC", 4000)
	r, err = l.RecordFill(sell2)
	if err != nil {
		t.Fatal(err)
	}
	if r.Order.Status != "FILLED" {
		t.Fatalf("filled sell: %+v", r.Order)
	}
	decimalClose(t, r.Position.RemainingQty, "0.48")
	if r.Position.Status != "OPEN" {
		t.Fatalf("should still hold 0.48 BTC: %+v", r.Position)
	}
	// Fee charged in BNB is preserved, but cannot be priced as USDT.
	finalSellID := createTestOrder(t, l, p.ID, "SELL", "spot-063-sell-2", "0.48")
	last := event(finalSellID, "10003", "70001", "SELL", "0.48", "130", "62.4", "0.001", "BNB", 5000)
	r, err = l.RecordFill(last)
	if err != nil {
		t.Fatal(err)
	}
	if r.Position.Status != "CLOSED" || r.Position.ClosedAt != 5000 {
		t.Fatalf("not closed: %+v", r.Position)
	}
	decimalClose(t, r.Position.RemainingQty, "0")
	decimalClose(t, r.Position.AvgCostQuote, "0")
	view, err := l.Accounting(p.ID, "130")
	if err != nil {
		t.Fatal(err)
	}
	if view.FullyPriced || view.UnpricedFees["BNB"] == "" || !view.HasMarkPrice {
		t.Fatalf("third asset fee must remain unconverted: %+v", view)
	}
	decimalClose(t, view.UnrealizedPnL, "0")
	decimalClose(t, view.TotalPnL, view.RealizedPnL)
	withoutPrice, err := l.Accounting(p.ID, "")
	if err != nil || withoutPrice.HasMarkPrice || withoutPrice.TotalPnL != "" {
		t.Fatalf("fabricated mark price: %+v %v", withoutPrice, err)
	}
	if _, _, err = l.CreateOrder(OrderIntent{PositionID: p.ID, Symbol: "BTCUSDT", Side: "BUY", Type: "MARKET", ClientOrderID: "after-close", RequestedQty: "1"}); err == nil {
		t.Fatal("closed position silently reused for new strategy cycle")
	}
	// A process restart cannot lose the canonical trade ledger.
	restarted, err := New(orm.NewOrmUsingDB(alias), "sqlite")
	if err != nil {
		t.Fatal(err)
	}
	rebuilt, err := restarted.RebuildPosition(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	decimalClose(t, rebuilt.RealizedPnL, r.Position.RealizedPnL)
	_, err = restarted.CreatePosition("BTCUSDT", "BTC", "USDT")
	if err != nil {
		t.Fatalf("new cycle after close: %v", err)
	}
	orders, err := restarted.ListOrders(p.ID)
	if err != nil || len(orders) != 3 {
		t.Fatalf("persisted orders %d err=%v", len(orders), err)
	}
	fills, err := restarted.ListFills(p.ID)
	if err != nil || len(fills) != 5 {
		t.Fatalf("persisted fills %d err=%v", len(fills), err)
	}
	for _, table := range []string{"futures_managed_positions", "futures_managed_orders", "futures_trade_fills"} {
		var marker string
		if err := orm.NewOrmUsingDB(alias).Raw("SELECT marker FROM " + table + " WHERE id=1").QueryRow(&marker); err != nil || marker != "futures-untouched" {
			t.Fatalf("Futures table %s modified, marker=%q err=%v", table, marker, err)
		}
	}
}

func TestStage063SpotStatusValidationAndRebuild(t *testing.T) {
	l, alias := fixture(t)
	p, err := l.CreatePosition("BTCUSDT", "BTC", "USDT")
	if err != nil {
		t.Fatal(err)
	}
	id := createTestOrder(t, l, p.ID, "BUY", "status-test-buy", "2")
	if _, err = l.SetExchangeStatus(id, "INVALID"); err == nil {
		t.Fatal("unsupported order status")
	}
	if _, err = l.SetExchangeStatus(id, "PENDING_CANCEL"); err != nil {
		t.Fatal(err)
	}
	if _, err = l.SetExchangeStatus(id, "CANCELED"); err != nil {
		t.Fatal(err)
	}
	if _, err = l.SetExchangeStatus(id, "NEW"); err == nil {
		t.Fatal("terminal status reopened")
	}
	// A fill can be reported after cancellation; the canonical fills win.
	r, err := l.RecordFill(event(id, "2001", "8001", "BUY", "0.4", "100", "40", "0", "", 6000))
	if err != nil {
		t.Fatal(err)
	}
	if r.Order.Status != "CANCELED" {
		t.Fatalf("canceled partial status lost: %+v", r.Order)
	}
	decimalClose(t, r.Order.FilledQty, "0.4")
	// A late callback can say FILLED before all fills are available.
	id2 := createTestOrder(t, l, p.ID, "BUY", "status-test-buy2", "1")
	if _, err = l.BindExchangeOrderID(id2, "2002"); err != nil {
		t.Fatal(err)
	}
	if _, err = l.SetExchangeStatus(id2, "FILLED"); err != nil {
		t.Fatal(err)
	}
	if _, err = l.SetExchangeStatus(id2, "CANCELED"); err == nil {
		t.Fatal("FILLED downgraded")
	}
	// New pending orders should be blocked until missing myTrades reconcile.
	if _, _, err = l.CreateOrder(OrderIntent{PositionID: p.ID, Symbol: "BTCUSDT", Side: "BUY", Type: "MARKET", ClientOrderID: "not-recovered", RequestedQty: "1"}); err == nil {
		t.Fatal("new intent created before FILLED exchange order trade recovery")
	}
	// With a recovered full fill, state may safely move on.
	if _, err = l.RecordFill(event(id2, "2002", "8002", "BUY", "1", "110", "110", "0.1", "USDT", 7000)); err != nil {
		t.Fatal(err)
	}
	// Rebuild fixes corrupted projections using immutable trade rows.
	o := orm.NewOrmUsingDB(alias)
	if _, err = o.Raw("UPDATE spot_managed_positions SET remaining_qty=999,avg_cost_quote=0,realized_pnl=123 WHERE id=?", p.ID).Exec(); err != nil {
		t.Fatal(err)
	}
	restored, err := l.RebuildPosition(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	decimalClose(t, restored.RemainingQty, "1.4")
	decimalClose(t, restored.RealizedPnL, "0")
	if restored.AvgCostQuote == "0" {
		t.Fatal("recovered avg cost should be nonzero")
	}
}

func TestStage063SpotRejectedFillsRollbackAndDedup(t *testing.T) {
	l, _ := fixture(t)
	p, err := l.CreatePosition("BTCUSDT", "BTC", "USDT")
	if err != nil {
		t.Fatal(err)
	}
	id := createTestOrder(t, l, p.ID, "BUY", "test-buy", "1")
	for _, bad := range []FillEvent{
		event(id, "100", "bad", "BUY", "1", "100", "100", "0", "", 1234),
		event(id, "100", "1", "SHORT", "1", "100", "100", "0", "", 1234),
		event(id, "100", "1", "SELL", "1", "100", "100", "0", "", 1234),
		event(id, "100", "1", "BUY", "-1", "100", "100", "0", "", 1234),
		event(id, "100", "1", "BUY", "1", "100", "100", "0.1", "", 1234),
		event(id, "100", "1", "BUY", "1", "100", "100", "2", "BTC", 1234),
		event(id, "100", "1", "BUY", "2", "100", "200", "0", "", 1234),
		event(id, "100", "1", "BUY", "1", "100", "1", "0", "", 1234),
	} {
		if _, err = l.RecordFill(bad); err == nil {
			t.Errorf("invalid fill accepted: %+v", bad)
		}
	}
	o, err := l.GetOrder(id)
	if err != nil || o.ExchangeOrderID != "" {
		t.Fatalf("failed transaction persisted order binding: %+v %v", o, err)
	}
	fills, err := l.ListFills(p.ID)
	if err != nil || len(fills) != 0 {
		t.Fatalf("failed transaction persisted fills: %d %v", len(fills), err)
	}
	good := event(id, "100", "1", "BUY", "1", "100", "100", "0", "", 1234)
	var wg sync.WaitGroup
	var mu sync.Mutex
	applied := 0
	var failures []error
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := l.RecordFill(good)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				failures = append(failures, err)
			}
			if r.Applied {
				applied++
			}
		}()
	}
	wg.Wait()
	if len(failures) > 0 || applied != 1 {
		t.Fatalf("concurrent duplicate fill result applied=%d errors=%v", applied, failures)
	}
	fills, err = l.ListFills(p.ID)
	if err != nil || len(fills) != 1 {
		t.Fatalf("replayed fill count %d err=%v", len(fills), err)
	}
}

func TestStage063SpotIdentityAndLimits(t *testing.T) {
	l, _ := fixture(t)
	for _, tc := range []struct{ symbol, base, quote string }{
		{"BTCUSDT", "BTC", "BNB"}, {"BTCUSDT", "USDT", "USDT"}, {"btcusdt", "BTC", "USDT"},
	} {
		if _, err := l.CreatePosition(tc.symbol, tc.base, tc.quote); err == nil {
			t.Fatalf("invalid identity accepted: %+v", tc)
		}
	}
	p, err := l.CreatePosition("BTCUSDT", "BTC", "USDT")
	if err != nil {
		t.Fatal(err)
	}
	for _, intent := range []OrderIntent{
		{PositionID: p.ID, Symbol: "BTCUSDT", Side: "SHORT", Type: "MARKET", ClientOrderID: "bad-short", RequestedQty: "1"},
		{PositionID: p.ID, Symbol: "BTCUSDT", Side: "BUY", Type: "LIMIT", ClientOrderID: "bad-budget", RequestedQty: "1e3"},
		{PositionID: p.ID, Symbol: "ETHUSDT", Side: "BUY", Type: "MARKET", ClientOrderID: "wrong-symbol", RequestedQty: "1"},
		{PositionID: p.ID, Symbol: "BTCUSDT", Side: "BUY", Type: "MARKET", ClientOrderID: "bad-precision", RequestedQty: "0.1234567890123456789"},
	} {
		if _, _, err = l.CreateOrder(intent); err == nil {
			t.Errorf("illegal intent accepted: %+v", intent)
		}
	}
	id := createTestOrder(t, l, p.ID, "BUY", "valid-client", "1")
	if _, _, err = l.CreateOrder(OrderIntent{PositionID: p.ID, Symbol: "BTCUSDT", Side: "BUY", Type: "MARKET", ClientOrderID: "valid-client", RequestedQty: "2"}); err == nil {
		t.Fatal("different order intent reused existing client id")
	}
	if _, err = l.BindExchangeOrderID(id, "abc"); err == nil {
		t.Fatal("nonnumeric exchange id accepted")
	}
	if _, err = l.BindExchangeOrderID(id, "9000"); err != nil {
		t.Fatal(err)
	}
	if _, err = l.BindExchangeOrderID(id, "9000"); err != nil {
		t.Fatal("idempotent same bind failed", err)
	}
	if _, err = l.BindExchangeOrderID(id, "9001"); err == nil {
		t.Fatal("order id rebound")
	}
	// Unrelated to a Binance wallet balance: only 06-3-managed fills count.
	snapshot, err := l.Accounting(p.ID, "50000")
	if err != nil {
		t.Fatal(err)
	}
	decimalClose(t, snapshot.UnrealizedPnL, "0")
	decimalClose(t, snapshot.TotalPnL, "0")
	if !snapshot.FullyPriced {
		t.Fatal("no-fill position cannot have unknown fee")
	}
	if _, err = New(l.db, "unrecognized"); err == nil {
		t.Fatal("unsupported DB dialect accepted")
	}
}

func ExampleLedger() {
	// This package has no Binance client or account-wallet dependency.
	fmt.Println("local Spot ledger, no live order submission")
	// Output: local Spot ledger, no live order submission
}

func TestStage063SpotPricedPNLAndFilters(t *testing.T) {
	l, _ := fixture(t)
	p, err := l.CreatePosition("BTCUSDT", "BTC", "USDT")
	if err != nil {
		t.Fatal(err)
	}
	buyID := createTestOrder(t, l, p.ID, "BUY", "priced-buy", "1")
	_, err = l.RecordFill(event(buyID, "410", "901", "BUY", "1", "100", "100", "1", "USDT", 1000))
	if err != nil {
		t.Fatal(err)
	}
	sellID := createTestOrder(t, l, p.ID, "SELL", "priced-sell", "0.4")
	_, err = l.RecordFill(event(sellID, "411", "902", "SELL", "0.4", "120", "48", "0.5", "USDT", 2000))
	if err != nil {
		t.Fatal(err)
	}
	view, err := l.Accounting(p.ID, "110")
	if err != nil {
		t.Fatal(err)
	}
	if !view.FullyPriced || !view.HasMarkPrice || len(view.UnpricedFees) != 0 {
		t.Fatalf("quote-denominated fees must produce priced result: %+v", view)
	}
	decimalClose(t, view.RealizedPnL, "7.1")
	decimalClose(t, view.UnrealizedPnL, "5.4")
	decimalClose(t, view.TotalPnL, "12.5")
	decimalClose(t, view.Position.RemainingQty, "0.6")
	decimalClose(t, view.Position.AvgCostQuote, "101")
	if _, err = l.Accounting(p.ID, "0"); err == nil {
		t.Fatal("zero mark is invalid")
	}
	if _, err = l.Accounting(p.ID, "-100"); err == nil {
		t.Fatal("negative mark is invalid")
	}
	page, err := l.ListPositions(PositionFilter{Symbol: "btcusdt", Status: "OPEN", Page: 1, Limit: 20})
	if err != nil || page.Total != 1 || len(page.List) != 1 || page.List[0].ID != p.ID {
		t.Fatalf("filter list: %+v %v", page, err)
	}
	for _, f := range []PositionFilter{{Status: "UNKNOWN"}, {Symbol: "BTC%USDT"}} {
		if _, err = l.ListPositions(f); err == nil {
			t.Errorf("unsafe filter accepted: %+v", f)
		}
	}
	orders, err := l.ListOrders(p.ID)
	if err != nil || len(orders) != 2 || orders[0].Side != "BUY" || orders[1].Side != "SELL" {
		t.Fatalf("order relationship: %+v %v", orders, err)
	}
	fills, err := l.ListFills(p.ID)
	if err != nil || len(fills) != 2 || fills[0].OrderID != buyID || fills[1].OrderID != sellID {
		t.Fatalf("fill relationship: %+v %v", fills, err)
	}
}

func TestStage063SpotSimultaneousOpenCycles(t *testing.T) {
	l, _ := fixture(t)
	var wg sync.WaitGroup
	var mu sync.Mutex
	created := 0
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p, err := l.CreatePosition("BTCUSDT", "BTC", "USDT")
			if err != nil {
				return
			}
			mu.Lock()
			created++
			if p.Status != "PENDING_BUY" {
				t.Errorf("invalid new position state %s", p.Status)
			}
			mu.Unlock()
		}()
	}
	wg.Wait()
	if created != 1 {
		t.Fatalf("simultaneous position creation yielded %d open cycles", created)
	}
	page, err := l.ListPositions(PositionFilter{Page: 1, Limit: 100})
	if err != nil || page.Total != 1 {
		t.Fatalf("position count: %+v %v", page, err)
	}
}

func TestStage063SpotOutOfOrderRecoveryAndPendingCancel(t *testing.T) {
	l, _ := fixture(t)
	p, err := l.CreatePosition("BTCUSDT", "BTC", "USDT")
	if err != nil {
		t.Fatal(err)
	}
	buyID := createTestOrder(t, l, p.ID, "BUY", "out-of-order-buy", "1")
	// The newer BUY fill arrives before an older execution with a distinct
	// real trade ID; replay must sort canonical fills by their trade timestamp.
	_, err = l.RecordFill(event(buyID, "1200", "3", "BUY", "0.6", "120", "72", "0", "", 2000))
	if err != nil {
		t.Fatal(err)
	}
	r, err := l.RecordFill(event(buyID, "1200", "2", "BUY", "0.4", "100", "40", "0", "", 1000))
	if err != nil {
		t.Fatal(err)
	}
	decimalClose(t, r.Position.RemainingQty, "1")
	decimalClose(t, r.Position.AvgCostQuote, "112")
	sellID := createTestOrder(t, l, p.ID, "SELL", "cancel-after-partial", "1")
	if _, err = l.SetExchangeStatus(sellID, "PENDING_CANCEL"); err != nil {
		t.Fatal(err)
	}
	r, err = l.RecordFill(event(sellID, "1300", "4", "SELL", "0.5", "130", "65", "0", "", 3000))
	if err != nil {
		t.Fatal(err)
	}
	if r.Order.Status != "PENDING_CANCEL" {
		t.Fatalf("late fill dropped pending cancel: %+v", r.Order)
	}
	if _, err = l.SetExchangeStatus(sellID, "CANCELED"); err != nil {
		t.Fatal(err)
	}
	decimalClose(t, r.Position.RemainingQty, "0.5")
	if _, _, err = l.CreateOrder(OrderIntent{PositionID: p.ID, Symbol: "BTCUSDT", Side: "SELL", Type: "MARKET", ClientOrderID: "oversell-recovery", RequestedQty: "0.6"}); err == nil {
		t.Fatal("oversold previously partially sold position")
	}
	finalID := createTestOrder(t, l, p.ID, "SELL", "close-remainder", "0.5")
	r, err = l.RecordFill(event(finalID, "1301", "5", "SELL", "0.5", "135", "67.5", "0", "", 4000))
	if err != nil || r.Position.Status != "CLOSED" {
		t.Fatalf("late/partial lifecycle: %+v %v", r, err)
	}
	decimalClose(t, r.Position.RealizedPnL, "20.5") // 65+67.5-112
}

func TestStage063SpotBaseFeeOverdraftRollsBack(t *testing.T) {
	l, _ := fixture(t)
	p, err := l.CreatePosition("BTCUSDT", "BTC", "USDT")
	if err != nil {
		t.Fatal(err)
	}
	buyID := createTestOrder(t, l, p.ID, "BUY", "base-buy", "1")
	if _, err = l.RecordFill(event(buyID, "12001", "1", "BUY", "1", "100", "100", "0", "", 1000)); err != nil {
		t.Fatal(err)
	}
	sellID := createTestOrder(t, l, p.ID, "SELL", "base-overdraft-sell", "1")
	// Exchange sells 1 BTC and charges another 0.01 BTC: the strategy
	// never owned 1.01 BTC, so the entire fill+binding must roll back.
	if _, err = l.RecordFill(event(sellID, "12002", "2", "SELL", "1", "110", "110", "0.01", "BTC", 2000)); err == nil {
		t.Fatal("fee in base asset overdrawn the strategy-owned quantity")
	}
	after, err := l.GetOrder(sellID)
	if err != nil || after.ExchangeOrderID != "" || after.FilledQty != "0" {
		t.Fatalf("failed sell mutated order: %+v %v", after, err)
	}
	position, err := l.GetPosition(p.ID)
	if err != nil || position.Status != "OPEN" {
		t.Fatalf("fee overdraft mutated position: %+v %v", position, err)
	}
	decimalClose(t, position.RemainingQty, "1")
	fills, err := l.ListFills(p.ID)
	if err != nil || len(fills) != 1 {
		t.Fatalf("fee overdraft persisted a fill: %d %v", len(fills), err)
	}
}

func TestStage063ExactDecimalScale(t *testing.T) {
	oneUnit := "0.000000000000000001"
	value, err := parseDBDecimal(oneUnit, true)
	if err != nil || value.String() != oneUnit {
		t.Fatalf("18-place decimal accepted exactly: %v %v", value, err)
	}
	formatted, err := numeric(value)
	if err != nil || formatted != "0.000000000000000001" {
		t.Fatalf("fixed-18 numeric format: %q %v", formatted, err)
	}
	for _, tooWide := range []string{"0.0000000000000000001", "1234567890123456789", "1e-18", "0", "-1"} {
		if _, err := parseDBDecimal(tooWide, true); err == nil {
			t.Errorf("invalid positive DECIMAL accepted: %q", tooWide)
		}
	}
	if _, err := numeric(decimal.RequireFromString("9999999999999999999.0")); err == nil {
		t.Fatal("19 integer-digit overflow accepted")
	}
}

// 06-3-F1: no client, server or UI may trust editable materialized PnL or
// inventory to authorize a SELL or claim a precise account-level profit.
func TestReview063TamperedMaterializedInventoryAndAccounting(t *testing.T) {
	l, alias := fixture(t)
	p, err := l.CreatePosition("BTCUSDT", "BTC", "USDT")
	if err != nil {
		t.Fatal(err)
	}
	buyID := createTestOrder(t, l, p.ID, "BUY", "rev063-buy", "1")
	if _, err = l.RecordFill(event(buyID, "3201", "1", "BUY", "1", "100", "100", "0", "", 1000)); err != nil {
		t.Fatal(err)
	}
	sellID := createTestOrder(t, l, p.ID, "SELL", "rev063-sell", "0.4")
	if _, err = l.RecordFill(event(sellID, "3202", "2", "SELL", "0.4", "120", "48", "0", "", 2000)); err != nil {
		t.Fatal(err)
	}
	db := orm.NewOrmUsingDB(alias)
	_, err = db.Raw("UPDATE spot_managed_positions SET remaining_qty=999,avg_cost_quote=1,realized_pnl=999 WHERE id=?", p.ID).Exec()
	if err != nil {
		t.Fatal(err)
	}
	view, err := l.Accounting(p.ID, "110")
	if err != nil {
		t.Fatal(err)
	}
	decimalClose(t, view.Position.RemainingQty, "0.6")
	decimalClose(t, view.Position.AvgCostQuote, "100")
	decimalClose(t, view.RealizedPnL, "8")
	decimalClose(t, view.UnrealizedPnL, "6")
	decimalClose(t, view.TotalPnL, "14")
	raw, err := l.GetPosition(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	decimalClose(t, raw.RemainingQty, "999") // Accounting MUST be read-only
	if _, _, err = l.CreateOrder(OrderIntent{PositionID: p.ID, Symbol: "BTCUSDT", Side: "SELL", Type: "MARKET", ClientOrderID: "rev063-oversell", RequestedQty: "999"}); err == nil {
		t.Fatal("accepted SELL 999 using modified remaining_qty cache")
	}
	if _, err = l.CreatePosition("BTCUSDT", "BTC", "USDT"); err == nil {
		t.Fatal("opened duplicate Spot cycle")
	}
	// A real new intent starts with a transactional re-projection.
	valid, duplicate, err := l.CreateOrder(OrderIntent{PositionID: p.ID, Symbol: "BTCUSDT", Side: "SELL", Type: "MARKET", ClientOrderID: "rev063-verified", RequestedQty: "0.6"})
	if err != nil || duplicate || valid.RequestedQty == "" {
		t.Fatalf("verified SELL refused: %+v dup=%v %v", valid, duplicate, err)
	}
	repaired, err := l.GetPosition(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	decimalClose(t, repaired.RemainingQty, "0.6")
	decimalClose(t, repaired.RealizedPnL, "8")
	// A false CLOSED marker cannot create a second inventory cycle.
	_, err = db.Raw("UPDATE spot_managed_positions SET status='CLOSED',remaining_qty=0 WHERE id=?", p.ID).Exec()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = l.CreatePosition("BTCUSDT", "BTC", "USDT"); err == nil {
		t.Fatal("created a second cycle based on a corrupted CLOSED status")
	}
}

func TestReview063TerminalOrderNeverRewrittenToFilled(t *testing.T) {
	l, _ := fixture(t)
	p, err := l.CreatePosition("BTCUSDT", "BTC", "USDT")
	if err != nil {
		t.Fatal(err)
	}
	buyID := createTestOrder(t, l, p.ID, "BUY", "rev063-terminal", "1")
	if _, err = l.SetExchangeStatus(buyID, "CANCELED"); err != nil {
		t.Fatal(err)
	}
	result, err := l.RecordFill(event(buyID, "3203", "1", "BUY", "1", "100", "100", "0", "", 1000))
	if err != nil {
		t.Fatal(err)
	}
	if result.Order.Status != "CANCELED" {
		t.Fatalf("reproject rewrote terminal CANCELED -> %s", result.Order.Status)
	}
	decimalClose(t, result.Order.FilledQty, "1")
	issues, err := l.ReviewPosition(p.ID)
	if err != nil || len(issues) == 0 {
		t.Fatalf("contradictory exchange state not surfaced: %v %v", issues, err)
	}
	if _, err = l.Accounting(p.ID, "120"); !errors.Is(err, ErrNeedsReview) {
		t.Fatalf("accounting claimed precise PnL for inconsistent order: %v", err)
	}
	if _, _, err = l.CreateOrder(OrderIntent{PositionID: p.ID, Symbol: "BTCUSDT", Side: "SELL", Type: "MARKET", ClientOrderID: "rev063-blocked", RequestedQty: "1"}); !errors.Is(err, ErrNeedsReview) {
		t.Fatalf("issued new order with contradictory terminal status: %v", err)
	}
	rebuilt, err := l.RebuildPosition(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	decimalClose(t, rebuilt.RemainingQty, "1")
	order, err := l.GetOrder(buyID)
	if err != nil || order.Status != "CANCELED" {
		t.Fatalf("restart recovery rewrote terminal order: %+v %v", order, err)
	}
	if _, err = l.SetExchangeStatus(buyID, "FILLED"); err == nil {
		t.Fatal("SetExchangeStatus reopened terminal order")
	}
}

func TestReview063MissingTradesAndOverfillRequireReconciliation(t *testing.T) {
	l, _ := fixture(t)
	p, err := l.CreatePosition("BTCUSDT", "BTC", "USDT")
	if err != nil {
		t.Fatal(err)
	}
	id := createTestOrder(t, l, p.ID, "BUY", "rev063-missing", "1")
	if _, err = l.SetExchangeStatus(id, "FILLED"); err != nil {
		t.Fatal(err)
	}
	issues, err := l.ReviewPosition(p.ID)
	if err != nil || len(issues) == 0 {
		t.Fatalf("FILLED with missing myTrades is not flagged: %v %v", issues, err)
	}
	if _, err = l.Accounting(p.ID, "100"); !errors.Is(err, ErrNeedsReview) {
		t.Fatalf("missing myTrades accounting: %v", err)
	}
	if _, _, err = l.CreateOrder(OrderIntent{PositionID: p.ID, Symbol: "BTCUSDT", Side: "BUY", Type: "MARKET", ClientOrderID: "rev063-early", RequestedQty: "1"}); !errors.Is(err, ErrNeedsReview) {
		t.Fatalf("issued order with missing myTrades: %v", err)
	}
	if _, err = l.RecordFill(event(id, "3204", "1", "BUY", "1.1", "100", "110", "0", "", 1000)); !errors.Is(err, ErrNeedsReview) {
		t.Fatalf("overfilled reconciliation not typed for operator: %v", err)
	}
	fills, err := l.ListFills(p.ID)
	if err != nil || len(fills) != 0 {
		t.Fatalf("overfill corrupted verified ledger: %d %v", len(fills), err)
	}
	if _, err = l.RecordFill(event(id, "3204", "1", "BUY", "1", "100", "100", "0", "", 1000)); err != nil {
		t.Fatal(err)
	}
	issues, err = l.ReviewPosition(p.ID)
	if err != nil || len(issues) != 0 {
		t.Fatalf("recovered myTrades still flagged: %v %v", issues, err)
	}
	view, err := l.Accounting(p.ID, "100")
	if err != nil {
		t.Fatal(err)
	}
	decimalClose(t, view.Position.RemainingQty, "1")
}

package spotledger

import (
	"fmt"
	"net"
	"os"
	"sync"
	"testing"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
	mysqlDriver "github.com/go-sql-driver/mysql"
)

// The MySQL test is opt-in and hard-limited to a purpose-built disposable DB.
// It must NEVER be used against go_binance, go_bn_test or any live schema.
// Example: SPOT_LEDGER_MYSQL_DSN=root:pw@tcp(127.0.0.1:33389)/spotledger063_isolated
func TestStage063MySQLIsolatedLifecycle(t *testing.T) {
	dsn := os.Getenv("SPOT_LEDGER_MYSQL_DSN")
	if dsn == "" {
		t.Skip("SPOT_LEDGER_MYSQL_DSN not set: only isolated MySQL 06-3 DB permitted")
	}
	parsed, err := mysqlDriver.ParseDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.DBName != "spotledger063_isolated" {
		t.Fatal("refusing MySQL write outside disposable spotledger063_isolated database")
	}
	// Local disposable containers only: a database sharing the test name
	// on a remote server must never become an implicit write target.
	host, _, err := net.SplitHostPort(parsed.Addr)
	if parsed.Net != "tcp" || err != nil || (host != "127.0.0.1" && host != "::1") {
		t.Fatal("refusing non-loopback MySQL integration DSN")
	}
	_ = orm.RegisterDriver("mysql", orm.DRMySQL)
	alias := fmt.Sprintf("spotledger063_mysql_%d", spotLedgerFixtureSeq.Add(1))
	if err := orm.RegisterDataBase(alias, "mysql", dsn); err != nil {
		t.Fatal(err)
	}
	o := orm.NewOrmUsingDB(alias)
	for _, ddl := range []string{
		`CREATE TABLE spot_symbols (id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,symbol VARCHAR(32) NOT NULL UNIQUE)`,
		`INSERT INTO spot_symbols(symbol) VALUES('BTCUSDT')`,
		`CREATE TABLE spot_managed_positions (
			id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY, symbol VARCHAR(32) NOT NULL,
			base_asset VARCHAR(32) NOT NULL,quote_asset VARCHAR(32) NOT NULL,
			bought_qty DECIMAL(36,18) NOT NULL DEFAULT 0,
			remaining_qty DECIMAL(36,18) NOT NULL DEFAULT 0,
			avg_cost_quote DECIMAL(36,18) NOT NULL DEFAULT 0,
			realized_pnl DECIMAL(36,18) NOT NULL DEFAULT 0,
			status VARCHAR(32) NOT NULL,
			created_at BIGINT NOT NULL,updated_at BIGINT NOT NULL,closed_at BIGINT NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE spot_managed_orders (
			id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
			position_id BIGINT NOT NULL,symbol VARCHAR(32) NOT NULL,side VARCHAR(4) NOT NULL,
			order_type VARCHAR(24) NOT NULL,client_order_id VARCHAR(64) NOT NULL,
			exchange_order_id VARCHAR(64) NULL,
			requested_qty DECIMAL(36,18) NOT NULL DEFAULT 0,
			filled_qty DECIMAL(36,18) NOT NULL DEFAULT 0,
			status VARCHAR(32) NOT NULL,created_at BIGINT NOT NULL,updated_at BIGINT NOT NULL,
			CONSTRAINT uq_spot_orders_client UNIQUE (client_order_id),
			CONSTRAINT uq_spot_orders_exchange UNIQUE (symbol,exchange_order_id)
		)`,
		`CREATE TABLE spot_trade_fills (
			id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
			position_id BIGINT NOT NULL,order_id BIGINT NOT NULL,
			exchange_order_id VARCHAR(64) NOT NULL,trade_id VARCHAR(64) NOT NULL,
			symbol VARCHAR(32) NOT NULL,side VARCHAR(4) NOT NULL,
			qty DECIMAL(36,18) NOT NULL,price DECIMAL(36,18) NOT NULL,
			quote_qty DECIMAL(36,18) NOT NULL,commission DECIMAL(36,18) NOT NULL,
			commission_asset VARCHAR(32) NOT NULL,trade_time BIGINT NOT NULL,
			CONSTRAINT uq_spot_fill_exchange UNIQUE (symbol,exchange_order_id,trade_id)
		)`,
		`CREATE TABLE futures_managed_positions (id BIGINT PRIMARY KEY, marker VARCHAR(60))`,
		`INSERT INTO futures_managed_positions VALUES(1,'unchanged')`,
	} {
		if _, err := o.Raw(ddl).Exec(); err != nil {
			t.Fatalf("MySQL fixture DDL: %v", err)
		}
	}
	// Match the four non-unique indexes created by the production v25
	// migration; a table-only fixture misses important SQL access paths.
	for _, ddl := range []string{
		"CREATE INDEX idx_spot_positions_symbol_status ON spot_managed_positions(symbol,status)",
		"CREATE INDEX idx_spot_orders_position ON spot_managed_orders(position_id)",
		"CREATE INDEX idx_spot_fills_position ON spot_trade_fills(position_id)",
		"CREATE INDEX idx_spot_fills_order ON spot_trade_fills(order_id)",
	} {
		if _, err := o.Raw(ddl).Exec(); err != nil {
			t.Fatalf("MySQL v25 index fixture: %v", err)
		}
	}
	t.Cleanup(func() {
		// Drop ONLY tables in the explicitly named disposable schema.
		for _, tbl := range []string{"spot_trade_fills", "spot_managed_orders", "spot_managed_positions", "spot_symbols", "futures_managed_positions"} {
			_, _ = o.Raw("DROP TABLE IF EXISTS " + tbl).Exec()
		}
	})
	l, err := New(o, "mysql")
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	var created []models.SpotManagedPosition
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p, err := l.CreatePosition("BTCUSDT", "BTC", "USDT")
			if err == nil {
				mu.Lock()
				created = append(created, p)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if len(created) != 1 {
		t.Fatalf("MySQL FOR UPDATE must serialize active cycles, got %d", len(created))
	}
	p := created[0]
	id := createTestOrder(t, l, p.ID, "BUY", "mysql-063-buy", "1")
	first := event(id, "6601", "7701", "BUY", "0.4", "100", "40", "0.1", "USDT", 1000)
	r, err := l.RecordFill(first)
	if err != nil {
		t.Fatal(err)
	}
	if r.Order.Status != "PARTIALLY_FILLED" {
		t.Fatalf("MySQL partial fill %+v", r.Order)
	}
	dup, err := l.RecordFill(first)
	if err != nil || dup.Applied {
		t.Fatalf("MySQL duplicate fill: %+v %v", dup, err)
	}
	r, err = l.RecordFill(event(id, "6601", "7702", "BUY", "0.6", "110", "66", "0.006", "BTC", 2000))
	if err != nil {
		t.Fatal(err)
	}
	if r.Order.Status != "FILLED" || r.Position.Status != "OPEN" {
		t.Fatalf("MySQL full BUY status %+v %+v", r.Order, r.Position)
	}
	decimalClose(t, r.Position.RemainingQty, "0.994")
	sellID := createTestOrder(t, l, p.ID, "SELL", "mysql-063-sell", "0.994")
	if _, err := l.RecordFill(event(sellID, "6602", "7703", "SELL", "0.994", "120", "119.28", "0.01", "BNB", 3000)); err != nil {
		t.Fatal(err)
	}
	view, err := l.Accounting(p.ID, "120")
	if err != nil || view.FullyPriced || view.UnpricedFees["BNB"] == "" || view.Position.Status != "CLOSED" {
		t.Fatalf("MySQL close/fee result: %+v err=%v", view, err)
	}
	newInstance, err := New(orm.NewOrmUsingDB(alias), "mysql")
	if err != nil {
		t.Fatal(err)
	}
	rebuilt, err := newInstance.RebuildPosition(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	decimalClose(t, rebuilt.RemainingQty, "0")
	orders, err := newInstance.ListOrders(p.ID)
	if err != nil || len(orders) != 2 {
		t.Fatalf("MySQL order relation: %+v %v", orders, err)
	}
	fills, err := newInstance.ListFills(p.ID)
	if err != nil || len(fills) != 3 {
		t.Fatalf("MySQL fill relation: %+v %v", fills, err)
	}
	// F1: a bad materialized snapshot must never falsify account PnL or
	// expand SELL capacity, even in MySQL (the target production dialect).
	if _, err = o.Raw("UPDATE spot_managed_positions SET remaining_qty=999, realized_pnl=999 WHERE id=?", p.ID).Exec(); err != nil {
		t.Fatal(err)
	}
	view, err = newInstance.Accounting(p.ID, "120")
	if err != nil {
		t.Fatal(err)
	}
	decimalClose(t, view.Position.RemainingQty, "0")
	if view.RealizedPnL == "999" {
		t.Fatal("MySQL Accounting trusted corrupted materialized PnL")
	}
	if _, _, err = newInstance.CreateOrder(OrderIntent{PositionID: p.ID, Symbol: "BTCUSDT", Side: "SELL", Type: "MARKET", ClientOrderID: "mysql-tampered-sell", RequestedQty: "999"}); err == nil {
		t.Fatal("MySQL allowed SELL 999 from corrupted cache")
	}
	// F2: MySQL late fully filled canceled order stays CANCELED and blocks
	// subsequent automatic intents until REST reconciliation.
	next, err := newInstance.CreatePosition("BTCUSDT", "BTC", "USDT")
	if err != nil {
		t.Fatal(err)
	}
	canceledID := createTestOrder(t, newInstance, next.ID, "BUY", "mysql-terminal-guard", "1")
	if _, err = newInstance.SetExchangeStatus(canceledID, "CANCELED"); err != nil {
		t.Fatal(err)
	}
	late, err := newInstance.RecordFill(event(canceledID, "6603", "7704", "BUY", "1", "100", "100", "0", "", 4000))
	if err != nil || late.Order.Status != "CANCELED" {
		t.Fatalf("MySQL terminal status overwritten: %+v %v", late, err)
	}
	if _, _, err = newInstance.CreateOrder(OrderIntent{PositionID: next.ID, Symbol: "BTCUSDT", Side: "SELL", Type: "MARKET", ClientOrderID: "mysql-review-block", RequestedQty: "1"}); err == nil {
		t.Fatal("MySQL did not block contradictory canceled-full order")
	}
	var marker string
	if err := o.Raw("SELECT marker FROM futures_managed_positions WHERE id=1").QueryRow(&marker); err != nil || marker != "unchanged" {
		t.Fatalf("MySQL Futures isolation failed: %q %v", marker, err)
	}
}

package line

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync/atomic"
	"testing"

	"go_binance_futures/service/spotledger"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
)

var spotReviewLedgerCounter atomic.Uint64

func spotReviewLedgerFixture(t *testing.T) (*spotledger.Ledger, orm.Ormer, int64) {
	t.Helper()
	_ = orm.RegisterDriver("sqlite3", orm.DRSqlite)
	alias := fmt.Sprintf("spot_review_0671_%d", spotReviewLedgerCounter.Add(1))
	if err := orm.RegisterDataBase(alias, "sqlite3", filepath.Join(t.TempDir(), "spot.db")); err != nil {
		t.Fatal(err)
	}
	db := orm.NewOrmUsingDB(alias)
	for _, sql := range []string{
		`CREATE TABLE spot_managed_positions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			symbol VARCHAR(32) NOT NULL,base_asset VARCHAR(32) NOT NULL,quote_asset VARCHAR(32) NOT NULL,
			bought_qty DECIMAL(36,18) NOT NULL DEFAULT 0,remaining_qty DECIMAL(36,18) NOT NULL DEFAULT 0,
			avg_cost_quote DECIMAL(36,18) NOT NULL DEFAULT 0,realized_pnl DECIMAL(36,18) NOT NULL DEFAULT 0,
			status VARCHAR(32) NOT NULL,created_at BIGINT NOT NULL,updated_at BIGINT NOT NULL,
			closed_at BIGINT NOT NULL DEFAULT 0)`,
		`CREATE TABLE spot_managed_orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			position_id BIGINT NOT NULL,symbol VARCHAR(32) NOT NULL,side VARCHAR(4) NOT NULL,
			order_type VARCHAR(24) NOT NULL,client_order_id VARCHAR(64) NOT NULL UNIQUE,
			exchange_order_id VARCHAR(64),requested_qty DECIMAL(36,18) NOT NULL DEFAULT 0,
			filled_qty DECIMAL(36,18) NOT NULL DEFAULT 0,status VARCHAR(32) NOT NULL,
			created_at BIGINT NOT NULL,updated_at BIGINT NOT NULL,
			UNIQUE(symbol,exchange_order_id))`,
		`CREATE TABLE spot_trade_fills (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			position_id BIGINT NOT NULL,order_id BIGINT NOT NULL,
			exchange_order_id VARCHAR(64) NOT NULL,trade_id VARCHAR(64) NOT NULL,
			symbol VARCHAR(32) NOT NULL,side VARCHAR(4) NOT NULL,
			qty DECIMAL(36,18) NOT NULL,price DECIMAL(36,18) NOT NULL,
			quote_qty DECIMAL(36,18) NOT NULL,
			commission DECIMAL(36,18) NOT NULL,commission_asset VARCHAR(32) NOT NULL,
			trade_time BIGINT NOT NULL,
			UNIQUE(symbol,exchange_order_id,trade_id))`,
	} {
		if _, err := db.Raw(sql).Exec(); err != nil {
			t.Fatal(err)
		}
	}
	l, err := spotledger.New(db, "sqlite")
	if err != nil {
		t.Fatal(err)
	}
	p, err := l.CreatePosition("BTCUSDT", "BTC", "USDT")
	if err != nil {
		t.Fatal(err)
	}
	o, existing, err := l.CreateOrder(spotledger.OrderIntent{
		PositionID: p.ID, Symbol: "BTCUSDT", Side: "BUY", Type: "MARKET",
		ClientOrderID: "spot_review_0671_buy", RequestedQty: "0.8",
	})
	if err != nil || existing {
		t.Fatalf("create order: %v duplicate=%v", err, existing)
	}
	if _, err := l.BindExchangeOrderID(o.ID, "51230001"); err != nil {
		t.Fatal(err)
	}
	applied, err := l.RecordFill(spotledger.FillEvent{
		OrderID: o.ID, Symbol: "BTCUSDT", ExchangeOrderID: "51230001",
		TradeID: "1234", Side: "BUY", Qty: "0.8", Price: "100",
		QuoteQty: "80", Commission: "0", CommissionAsset: "",
		TradeTime: 123456789,
	})
	if err != nil || !applied.Applied || applied.Position.Status != "OPEN" {
		t.Fatalf("verified Fill: %+v %v", applied, err)
	}
	return l, db, p.ID
}

func TestReview0671EvaluateExitCannotTrustCallerMaterializedValues(t *testing.T) {
	l, db, id := spotReviewLedgerFixture(t)
	coin := spotRuleTestCoin()
	e := spotRuleTestEvaluator(110)

	actual, err := e.EvaluateExit(coin, l, id)
	if err != nil || actual.Action != "SELL" || actual.ROI != "10" {
		t.Fatalf("real immutable Fill inventory not evaluated: %+v %v", actual, err)
	}
	// This is the concrete F2 exploit from the review: the mutable cache
	// can contain 999 units at $1, while confirmed Fill is 0.8 at $100.
	if _, err := db.Raw("UPDATE spot_managed_positions SET remaining_qty='999',avg_cost_quote='1' WHERE id=?", id).Exec(); err != nil {
		t.Fatal(err)
	}
	actual, err = e.EvaluateExit(coin, l, id)
	if err != nil || actual.Action != "SELL" || actual.ROI != "10" {
		t.Fatalf("read editable materialized fields instead of Fill projection: %+v %v", actual, err)
	}
	if signal, err := e.EvaluateExit(coin, nil, id); err == nil || signal.Action != "" {
		t.Fatalf("nil ledger bypassed authoritative projection: %+v %v", signal, err)
	}
	if signal, err := e.EvaluateExit(coin, l, 0); err == nil || signal.Action != "" {
		t.Fatalf("invalid id bypassed authoritative projection: %+v %v", signal, err)
	}
}

func TestReview0671EvaluateExitFailsClosedOnFillInconsistency(t *testing.T) {
	l, db, id := spotReviewLedgerFixture(t)
	// Contradictory immutable Fill evidence cannot be repaired by trusting
	// the cached Position or by generating a new SELL intent.
	if _, err := db.Raw("UPDATE spot_trade_fills SET qty='1.2' WHERE trade_id='1234'").Exec(); err != nil {
		t.Fatal(err)
	}
	signal, err := spotRuleTestEvaluator(110).EvaluateExit(spotRuleTestCoin(), l, id)
	if !errors.Is(err, spotledger.ErrNeedsReview) || signal.Action != "" {
		t.Fatalf("inconsistent Fill produced a trading signal: %+v %v", signal, err)
	}
}

package orderhistory

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
	"go_binance_futures/models"
	"go_binance_futures/service/tradefills"
)

var initOnce sync.Once
var nextAlias atomic.Int64

func historyDB(t *testing.T) (orm.Ormer, Store) {
	t.Helper()
	_ = orm.RegisterDriver("sqlite3", orm.DRSqlite)
	initOnce.Do(func() {
		if _, err := orm.GetDB("default"); err != nil {
			if e := orm.RegisterDataBase("default", "sqlite3", "file:s510_history_default?mode=memory&cache=shared"); e != nil {
				t.Fatal(e)
			}
		}
		orm.RegisterModel(new(models.FuturesManagedPosition), new(models.FuturesManagedOrder))
	})
	alias := fmt.Sprintf("s510_history015_%d", nextAlias.Add(1))
	if err := orm.RegisterDataBase(alias, "sqlite3", filepath.Join(t.TempDir(), "history.db")); err != nil {
		t.Fatal(err)
	}
	if err := orm.RunSyncdb(alias, false, false); err != nil {
		t.Fatal(err)
	}
	o := orm.NewOrmUsingDB(alias)
	if _, err := o.Raw(tradefills.SchemaDDL).Exec(); err != nil {
		t.Fatal(err)
	}
	return o, Store{DB: o, Now: func() time.Time { return time.UnixMilli(1800000010000).UTC() },
		ActivatedAt: func(account string) (int64, error) { return 1799999999000, nil }}
}
func positionFixture(t *testing.T, o orm.Ormer, account, owner, symbol, side, status, settlement string, closed bool) int64 {
	t.Helper()
	pos := &models.FuturesManagedPosition{AccountID: account, Owner: owner, Symbol: symbol, PositionSide: side,
		Status: status, SettlementStatus: settlement, ManagedQty: 0, EntryPrice: 100,
		CreatedAt: 1800000000000, UpdatedAt: 1800000005000}
	if closed {
		pos.ClosedAt = 1800000005000
	}
	if settlement == "settled" {
		pos.SettlementProcessedAt = 1800000006000
		pos.SettlementPNLSign = 1
	}
	id, err := o.Insert(pos)
	if err != nil {
		t.Fatal(err)
	}
	return id
}
func managedOrder(t *testing.T, o orm.Ormer, positionID int64, account, owner, symbol, side, client, intent, orderID string, qty float64) models.FuturesManagedOrder {
	t.Helper()
	row := models.FuturesManagedOrder{AccountID: account, Owner: owner, Symbol: symbol, PositionSide: side,
		ClientOrderID: client, Intent: intent, PositionID: &positionID, ExchangeOrderID: orderID,
		RequestedQty: qty, FilledQty: qty, OrderType: "MARKET", Status: "filled",
		CreatedAt: 1800000000000, UpdatedAt: 1800000001000}
	if _, err := o.Insert(&row); err != nil {
		t.Fatal(err)
	}
	return row
}
func fill(t *testing.T, o orm.Ormer, account, symbol, side, orderID, tradeID, client, intent, qty, price, fee, pnl, asset string, pid int64, verified bool) {
	t.Helper()
	v := 0
	if verified {
		v = 1
	}
	_, err := o.Raw(`INSERT INTO futures_trade_fills
 (market,account_id,symbol,order_id,trade_id,position_id,client_order_id,intent,position_side,
 qty,price,commission,commission_asset,realized_pnl,pnl_asset,trade_time,rest_verified)
 VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		tradefills.MarketUSDM, account, symbol, orderID, tradeID, pid, client, intent, side, qty, price, fee, asset, pnl, "USDT",
		1800000001000, v).Exec()
	if err != nil {
		t.Fatal(err)
	}
}
func fullCycle(t *testing.T, o orm.Ormer) int64 {
	t.Helper()
	id := positionFixture(t, o, "main", "auto_strategy", "BTCUSDT", "LONG", "closed", "settled", true)
	managedOrder(t, o, id, "main", "auto_strategy", "BTCUSDT", "LONG", "open-1", "open", "101", 4)
	managedOrder(t, o, id, "main", "auto_strategy", "BTCUSDT", "LONG", "close-1", "close", "102", 1)
	managedOrder(t, o, id, "main", "auto_strategy", "BTCUSDT", "LONG", "close-2", "close", "103", 3)
	fill(t, o, "main", "BTCUSDT", "LONG", "101", "1", "open-1", "open", "2", "100", "0.2", "0", "USDT", id, true)
	fill(t, o, "main", "BTCUSDT", "LONG", "101", "2", "open-1", "open", "2", "120", "0.2", "0", "USDT", id, true)
	fill(t, o, "main", "BTCUSDT", "LONG", "102", "3", "close-1", "close", "1", "130", "0.1", "20", "USDT", id, true)
	fill(t, o, "main", "BTCUSDT", "LONG", "103", "4", "close-2", "close", "3", "140", "0.3", "90", "USDT", id, true)
	return id
}
func TestStage015OneCycleMultipleClosesExactFeesAndNominalReturn(t *testing.T) {
	o, store := historyDB(t)
	id := fullCycle(t, o)
	page, err := store.List(context.Background(), Filter{AccountID: "main", Page: 1, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || len(page.List) != 1 || page.List[0].PositionID != id {
		t.Fatalf("not exactly one position cycle: %+v", page)
	}
	got := page.List[0]
	if got.OpenQty != "4" || got.CloseQty != "4" || got.RemainingQty != "0" {
		t.Fatalf("opened+partial closes distorted: %+v", got)
	}
	if got.OpenAvgPrice == nil || *got.OpenAvgPrice != "110" || got.CloseAvgPrice == nil || *got.CloseAvgPrice != "137.5" {
		t.Fatalf("VWAP weighted exchange fills incorrect: %+v", got)
	}
	if got.NetPnL == nil || *got.NetPnL != "109.2" || got.GrossPnL == nil || *got.GrossPnL != "110" ||
		got.ReturnPct == nil || *got.ReturnPct != "24.818181818181818182" || got.Completeness != "verified" ||
		got.PnLAsset != "USDT" || got.Leverage != nil {
		t.Fatalf("verified net/nominal ROI or missing leverage policy broken: %+v", got)
	}
	if page.PageVerifiedNet["USDT"] != "109.2" {
		t.Fatalf("wrong page-only verified sum: %+v", page)
	}
	detail, err := store.Details(context.Background(), "main", id)
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Orders) != 3 || len(detail.Orders[0].Fills) != 2 || len(detail.Orders[1].Fills) != 1 ||
		len(detail.Orders[2].Fills) != 1 {
		t.Fatalf("expected OPEN + two CLOSE children with immutable Fill rows: %+v", detail)
	}
	for i, child := range detail.Orders {
		if child.Completeness != "verified" || len(child.Fees) != 1 {
			t.Fatalf("completed child fee/verification missing: %+v", child)
		}
		want := []string{"0.4", "0.1", "0.3"}[i]
		if child.Fees[0].Amount != want || child.Fees[0].Asset != "USDT" || child.GrossPnL == nil {
			t.Fatalf("per order exact fee/PnL mismatched: %+v", child)
		}
	}
}

func TestStage015SecondSameSymbolCycleSeparatedAndAccountScoped(t *testing.T) {
	o, store := historyDB(t)
	first := fullCycle(t, o)
	second := positionFixture(t, o, "main", "auto_strategy", "BTCUSDT", "LONG", "active", "open", false)
	// A second cycle can be active, no executions yet; never inherit first.
	managedOrder(t, o, second, "main", "auto_strategy", "BTCUSDT", "LONG", "second-open", "open", "501", 0)
	lead := positionFixture(t, o, "lead", "auto_strategy", "BTCUSDT", "LONG", "closed", "legacy", true)
	// Deliberately leave lead without full history; never show main fills.
	page, err := store.List(context.Background(), Filter{Page: 1, Limit: 1, Symbol: "btcusdt"})
	if err != nil || page.Total != 2 || len(page.List) != 1 {
		t.Fatalf("main pagination/case-fold filter: %+v %v", page, err)
	}
	secondPage, err := store.List(context.Background(), Filter{Page: 2, Limit: 1, Symbol: "BTC"})
	if err != nil || secondPage.Total != 2 || len(secondPage.List) != 1 {
		t.Fatalf("pagination second: %+v %v", secondPage, err)
	}
	seen := map[int64]bool{page.List[0].PositionID: true, secondPage.List[0].PositionID: true}
	if !seen[first] || !seen[second] {
		t.Fatalf("same symbol cycles got merged: %+v", seen)
	}
	leadPage, err := store.List(context.Background(), Filter{AccountID: "lead", Page: 1, Limit: 20})
	if err != nil || leadPage.Total != 1 || leadPage.List[0].PositionID != lead {
		t.Fatalf("cross-account cycle disclosure/mixing: %+v %v", leadPage, err)
	}
	if _, err := store.Details(context.Background(), "lead", first); !errorsIs(err, ErrNotFound) {
		t.Fatalf("main position ID leaked through lead detail: %v", err)
	}
}
func errorsIs(err, expected error) bool { return err == expected }

func TestStage015NoFabricatedPNLOnIncompleteOrMixedFee(t *testing.T) {
	for _, scenario := range []string{"unverified", "missing_fill", "bnb_fee", "legacy", "cross_account_orphan"} {
		t.Run(scenario, func(t *testing.T) {
			o, store := historyDB(t)
			id := fullCycle(t, o)
			switch scenario {
			case "unverified":
				_, _ = o.Raw("UPDATE futures_trade_fills SET rest_verified=0 WHERE trade_id='4'").Exec()
			case "missing_fill":
				_, _ = o.Raw("DELETE FROM futures_trade_fills WHERE trade_id='2'").Exec()
			case "bnb_fee":
				_, _ = o.Raw("UPDATE futures_trade_fills SET commission_asset='BNB' WHERE trade_id='4'").Exec()
			case "legacy":
				_, _ = o.Raw("UPDATE futures_managed_positions SET settlement_status='legacy',settlement_processed_at=0 WHERE id=?", id).Exec()
			case "cross_account_orphan":
				_, _ = o.Raw("UPDATE futures_trade_fills SET client_order_id='foreign-account' WHERE trade_id='4'").Exec()
			}
			list, err := store.List(context.Background(), Filter{Page: 1, Limit: 20})
			if err != nil {
				t.Fatal(err)
			}
			got := list.List[0]
			if got.Completeness != "incomplete" || got.NetPnL != nil || got.ReturnPct != nil ||
				len(list.PageVerifiedNet) != 0 || len(got.Fees) != 0 {
				t.Fatalf("unverified/%s cycle exposes misleading financial result: %+v", scenario, got)
			}
		})
	}
}

func TestStage015InvalidPaginationTimeAndFiltersRejected(t *testing.T) {
	_, store := historyDB(t)
	for _, q := range []Filter{
		{AccountID: "admin", Page: 1, Limit: 20},
		{Page: 0, Limit: 20}, {Page: 1, Limit: 101},
		{Page: 1, Limit: 20, Status: "OPEN); DROP TABLE order"},
		{Page: 1, Limit: 20, Symbol: "BTC' OR 1=1"},
		{Page: 1, Limit: 20, StartTime: 200, EndTime: 100},
	} {
		if _, err := store.List(context.Background(), q); err == nil {
			t.Fatalf("unsafe filters accepted: %+v", q)
		}
	}
}
func TestStage015NoLegacyTableRequiredAndNoMutation(t *testing.T) {
	o, store := historyDB(t)
	id := fullCycle(t, o)
	// New endpoint cannot depend on the retired physical table.
	if _, err := o.Raw("CREATE TABLE `order` (id INTEGER PRIMARY KEY, message TEXT)").Exec(); err != nil {
		t.Fatal(err)
	}
	if _, err := o.Raw("INSERT INTO `order` (id,message) VALUES(1,'must survive')").Exec(); err != nil {
		t.Fatal(err)
	}
	before := 0
	if err := o.Raw("SELECT COUNT(*) FROM futures_trade_fills").QueryRow(&before); err != nil {
		t.Fatal(err)
	}
	if _, err := store.List(context.Background(), Filter{Page: 1, Limit: 20}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Details(context.Background(), "main", id); err != nil {
		t.Fatal(err)
	}
	after := 0
	if err := o.Raw("SELECT COUNT(*) FROM futures_trade_fills").QueryRow(&after); err != nil || after != before {
		t.Fatalf("history API mutated immutable Fill facts: before=%d after=%d %v", before, after, err)
	}
	var message string
	if err := o.Raw("SELECT message FROM `order` WHERE id=1").QueryRow(&message); err != nil || !strings.Contains(message, "survive") {
		t.Fatalf("legacy table was modified: %q %v", message, err)
	}
}

// Algo parent IDs are not execution IDs. The observed Fill must be attributed
// by actual_order_id, not the exchange's algoId, even after order cancellation.
func TestStage015AlgoActualOrderIDAndCanceledPartialClose(t *testing.T) {
	o, store := historyDB(t)
	id := fullCycle(t, o)
	if _, err := o.Raw(`UPDATE futures_managed_orders
        SET exchange_order_id='77002',actual_order_id='102',order_type='TAKE_PROFIT_MARKET',
            requested_qty=5,filled_qty=1,status='canceled',intent='take_profit'
        WHERE client_order_id='close-1'`).Exec(); err != nil {
		t.Fatal(err)
	}
	if _, err := o.Raw(`UPDATE futures_trade_fills SET intent='take_profit' WHERE order_id='102'`).Exec(); err != nil {
		t.Fatal(err)
	}
	first, err := store.Details(context.Background(), "main", id)
	if err != nil {
		t.Fatal(err)
	}
	if first.Position.Completeness != "verified" || len(first.Orders) != 3 ||
		first.Orders[1].Managed.ExchangeOrderID != "77002" ||
		first.Orders[1].Managed.ActualOrderID == nil ||
		*first.Orders[1].Managed.ActualOrderID != "102" ||
		first.Orders[1].FilledQtyFromFills != "1" {
		t.Fatalf("algo actual-order link or canceled partial trade lost: %+v", first)
	}
	if _, err := o.Raw("UPDATE futures_managed_orders SET actual_order_id=NULL WHERE client_order_id='close-1'").Exec(); err != nil {
		t.Fatal(err)
	}
	missing, err := store.List(context.Background(), Filter{Page: 1, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if missing.List[0].Completeness != "incomplete" || missing.List[0].NetPnL != nil {
		t.Fatalf("algoId alone incorrectly certified execution identity: %+v", missing.List[0])
	}
}

// An ACK-only OPEN has no position_id until first real fill. It belongs to
// the existing pending/open-order view, not a fabricated cycle history row.
func TestStage015ZeroFilledUnboundOpenIsNotFabricated(t *testing.T) {
	o, store := historyDB(t)
	row := models.FuturesManagedOrder{AccountID: "main", Owner: "auto_strategy", Symbol: "SOLUSDT",
		PositionSide: "LONG", Intent: "open", ClientOrderID: "pending-unbound",
		RequestedQty: 2, FilledQty: 0, Status: "submitted", OrderType: "LIMIT", CreatedAt: 1800000001000}
	if _, err := o.Insert(&row); err != nil {
		t.Fatal(err)
	}
	page, err := store.List(context.Background(), Filter{Page: 1, Limit: 20})
	if err != nil || page.Total != 0 || len(page.List) != 0 {
		t.Fatalf("zero-fill unbound order fabricated a position cycle: %+v %v", page, err)
	}
}

// Internal source_ref can contain proposal identifiers or future private
// metadata. The history endpoint projects an explicit allowlist instead of
// accidentally exposing new ORM columns on model evolution.
func TestStage015HistoryJSONResponseDoesNotLeakInternalSourceRefs(t *testing.T) {
	o, store := historyDB(t)
	id := fullCycle(t, o)
	const marker = "S510_PRIVATE_PROPOSAL_DO_NOT_EXPORT"
	if _, err := o.Raw("UPDATE futures_managed_positions SET source_ref=? WHERE id=?", marker, id).Exec(); err != nil {
		t.Fatal(err)
	}
	if _, err := o.Raw("UPDATE futures_managed_orders SET source_ref=? WHERE position_id=?", marker, id).Exec(); err != nil {
		t.Fatal(err)
	}
	details, err := store.Details(context.Background(), "main", id)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(details)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), marker) || strings.Contains(string(raw), "source_ref") {
		t.Fatalf("private source_ref leaked through public order history DTO: %s", string(raw))
	}
	if !strings.Contains(string(raw), `"client_order_id":"open-1"`) ||
		!strings.Contains(string(raw), `"order_id":"101"`) {
		t.Fatalf("public trade IDs required by the order details page were lost: %s", string(raw))
	}
	page, err := store.List(context.Background(), Filter{Page: 1, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	jsonPage, err := json.Marshal(page)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(jsonPage), marker) || strings.Contains(string(jsonPage), "source_ref") {
		t.Fatalf("internal source_ref leaked through list: %s", string(jsonPage))
	}
}

// ALL-F4: an old, even REST-verified/settled cycle must not be certified
// as "new history" until its account has an explicit T0. This only affects
// the read-only projection and its page totals, never persisted settlement.
func TestStage016AccountHistoryT0GateFailsClosedWithoutChangingLedger(t *testing.T) {
	o, store := historyDB(t)
	id := fullCycle(t, o)
	cases := []struct {
		name   string
		t0     int64
		status string
		reason string
		net    *string
	}{
		{"not_activated", 0, "incomplete", "history_not_activated", nil},
		{"pre_activation", 1800000000001, "incomplete", "pre_activation_position_cycle", nil},
		{"activated_earlier", 1799999999999, "verified", "", ptr(big.NewRat(546, 5))},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store.ActivatedAt = func(account string) (int64, error) {
				if account != "main" {
					t.Fatalf("history requested wrong account: %s", account)
				}
				return tc.t0, nil
			}
			list, err := store.List(context.Background(), Filter{AccountID: "main", Page: 1, Limit: 20})
			if err != nil || list.Total != 1 {
				t.Fatalf("history query: %v, %+v", err, list)
			}
			item := list.List[0]
			if item.Completeness != tc.status || item.IncompleteReason != tc.reason {
				t.Fatalf("T0 not enforced: %+v", item)
			}
			detail, err := store.Details(context.Background(), "main", id)
			if err != nil {
				t.Fatal(err)
			}
			if tc.net == nil {
				for _, child := range detail.Orders {
					if child.GrossPnL != nil || len(child.Fees) != 0 || child.Completeness == "verified" {
						t.Fatalf("pre-T0 child falsely certified: %+v", child)
					}
				}
			}
			if detail.Position.Completeness != item.Completeness || detail.Position.IncompleteReason != item.IncompleteReason {
				t.Fatalf("detail bypassed history T0: list=%+v detail=%+v", item, detail.Position)
			}
			if tc.net == nil {
				if item.NetPnL != nil || item.ReturnPct != nil || len(list.PageVerifiedNet) != 0 {
					t.Fatalf("old/unactivated PnL leaked: %+v %+v", item, list.PageVerifiedNet)
				}
			} else if item.NetPnL == nil || *item.NetPnL != *tc.net || list.PageVerifiedNet["USDT"] != *tc.net {
				t.Fatalf("post-T0 verified PnL lost: %+v %+v", item, list.PageVerifiedNet)
			}
		})
	}
	var status string
	if err := o.Raw("SELECT settlement_status FROM futures_managed_positions WHERE id=?", id).QueryRow(&status); err != nil || status != "settled" {
		t.Fatalf("read-only T0 gate mutated settlement: status=%q err=%v", status, err)
	}
}

func TestStage016ConfiguredHistoryT0IndependentAndMalformedRejected(t *testing.T) {
	t.Setenv("BINANCE_ORDER_HISTORY_T0_MAIN_MS", "")
	t.Setenv("BINANCE_ORDER_HISTORY_T0_LEAD_MS", "")
	if v, err := ConfiguredHistoryT0("main"); err != nil || v != 0 {
		t.Fatalf("unset T0 must not activate: %d %v", v, err)
	}
	t.Setenv("BINANCE_ORDER_HISTORY_T0_MAIN_MS", "1800000000000")
	t.Setenv("BINANCE_ORDER_HISTORY_T0_LEAD_MS", "1800000001000")
	if v, err := ConfiguredHistoryT0("main"); err != nil || v != 1800000000000 {
		t.Fatalf("main T0 not independent: %d %v", v, err)
	}
	if v, err := ConfiguredHistoryT0("lead"); err != nil || v != 1800000001000 {
		t.Fatalf("lead T0 not independent: %d %v", v, err)
	}
	for _, raw := range []string{"0", "garbage", "-1", "2026-11-09T00:00:00Z", "4102444800001"} {
		t.Setenv("BINANCE_ORDER_HISTORY_T0_MAIN_MS", raw)
		if _, err := ConfiguredHistoryT0("main"); err == nil {
			t.Fatalf("invalid T0 accepted %q", raw)
		}
	}
	if _, err := ConfiguredHistoryT0("attacker"); err == nil {
		t.Fatal("unknown account accepted")
	}
}

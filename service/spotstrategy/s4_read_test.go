package spotstrategy

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"go_binance_futures/models"
)

func TestS4ReadOnlySettingsScopeAndPagination(t *testing.T) {
	setupSpotS1(t)
	db, err := orm.GetDB("default")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err = ListSettings(ctx, db, PageRequest{Symbol: "BTCUSDT' OR 1=1--"}); err == nil {
		t.Fatal("SQL injection filter accepted")
	}
	if _, err = ListSettings(ctx, db, PageRequest{Limit: 51}); err == nil {
		t.Fatal("unbounded list accepted")
	}
	if _, err = ListSettings(ctx, db, PageRequest{Page: -1}); err == nil {
		t.Fatal("negative page accepted")
	}
	result, err := ListSettings(ctx, db, PageRequest{})
	if err != nil || result.Total != 1 || len(result.Items) != 1 {
		t.Fatalf("settings result %+v err=%v", result, err)
	}
	item := result.Items[0]
	if item.Enabled || item.Symbol != "BTCUSDT" || item.QuoteAsset != "USDT" {
		t.Fatalf("untrusted config visible %+v", item)
	}
	if _, err = orm.NewOrm().Insert(&models.SpotSymbols{Symbol: "ETHUSDT", Type: "USDT", Technology: "{}", Strategy: "[]", StrategyBudgetQuote: "0"}); err != nil {
		t.Fatal(err)
	}
	paged, err := ListSettings(ctx, db, PageRequest{Page: 2, Limit: 1})
	if err != nil || paged.Total != 2 || len(paged.Items) != 1 || paged.Items[0].Symbol != "ETHUSDT" {
		t.Fatalf("page 2 %+v %v", paged, err)
	}
}

func TestS4HoldingToOrderToFillReadOnlyAndUnsafeDataHidden(t *testing.T) {
	setupSpotS1(t)
	db, err := orm.GetDB("default")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UnixMilli()
	result, err := db.Exec(`INSERT INTO spot_strategy_holdings
 (account_id,owner,symbol,base_asset,quote_asset,owned_qty,avg_cost_quote,status,active_slot,created_at,updated_at)
 VALUES('main','auto_strategy','BTCUSDT','BTC','USDT','2','100','active','main:auto_strategy:BTCUSDT',?,?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	holdingID, _ := result.LastInsertId()
	result, err = db.Exec(`INSERT INTO spot_strategy_orders
 (holding_id,account_id,owner,symbol,side,order_type,intent,
 client_order_id,signal_key,requested_qty,filled_qty,executed_quote_qty,reserved_qty,status,created_at,updated_at,exchange_order_id)
 VALUES (?,'main','auto_strategy','BTCUSDT','BUY','LIMIT','open',
 's4-own-cid','s4:unique','2','2','200','0','filled',?,?,'7001')`, holdingID, now, now)
	if err != nil {
		t.Fatal(err)
	}
	orderID, _ := result.LastInsertId()
	if _, err = db.Exec(`INSERT INTO spot_strategy_fills
 (account_id,symbol,order_id,trade_id,holding_id,client_order_id,side,qty,price,quote_qty,commission,commission_asset,trade_time,rest_verified)
 VALUES ('main','BTCUSDT','7001','71',?,'s4-own-cid','BUY','2','100','200','0.01','BNB',?,0)`, holdingID, now); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	h, err := ListHoldings(ctx, db, PageRequest{Symbol: "BTCUSDT"})
	if err != nil || h.Total != 1 || len(h.Items) != 1 {
		t.Fatalf("holding list %+v %v", h, err)
	}
	item := h.Items[0]
	if item.Complete || item.OwnedQty != "" || item.CostBasisQuote != "" || item.RealizedGrossQuote != "" ||
		!strings.Contains(item.IncompleteReason, "fill_source_not_verified") {
		t.Fatalf("unverified data falsely exposed: %+v", item)
	}
	orders, err := ListOrders(ctx, db, holdingID, PageRequest{})
	if err != nil || orders.Total != 1 || len(orders.Items) != 1 || orders.Items[0].ID != orderID ||
		orders.Items[0].ClientOrderID != "s4-own-cid" || orders.Items[0].RemainingQty != "0" {
		t.Fatalf("order list %+v %v", orders, err)
	}
	fills, err := ListOrderFills(ctx, db, orderID, PageRequest{})
	if err != nil || fills.Total != 1 || len(fills.Items) != 1 ||
		fills.Items[0].CommissionAsset != "BNB" || fills.Items[0].RESTVerified {
		t.Fatalf("Fill list %+v %v", fills, err)
	}
	filtered, filterErr := ListOrderFills(ctx, db, orderID, PageRequest{Symbol: "ETHUSDT"})
	if filterErr != nil || filtered.Total != 0 || len(filtered.Items) != 0 {
		t.Fatalf("wrong-symbol Fill filter silently ignored: %+v %v", filtered, filterErr)
	}
	matching, matchErr := ListOrderFills(ctx, db, orderID, PageRequest{Symbol: "BTCUSDT"})
	if matchErr != nil || matching.Total != 1 {
		t.Fatalf("same-symbol Fill filter wrong: %+v %v", matching, matchErr)
	}
	if _, err := ListOrderFills(ctx, db, 7001, PageRequest{}); !errors.Is(err, ErrUnmanagedOrder) {
		t.Fatalf("exchangeOrderID without local identity allowed: %v", err)
	}
	if _, err := ListOrders(ctx, db, 999999, PageRequest{}); !errors.Is(err, ErrUnmanagedOrder) {
		t.Fatalf("missing Holding ID error: %v", err)
	}
	for _, req := range []PageRequest{{Limit: 51}, {Status: "withdraw"}, {Symbol: "BTCUSDT;DROP TABLE"}} {
		if _, err := ListHoldings(ctx, db, req); err == nil {
			t.Fatalf("unsafe filter accepted: %+v", req)
		}
	}
	// No new order/fill is created by the read paths.
	var n int
	if err = db.QueryRow("SELECT COUNT(*) FROM spot_strategy_orders").Scan(&n); err != nil || n != 1 {
		t.Fatalf("read method modified orders: count=%d err=%v", n, err)
	}
}

func TestS4OrdersUnknownAndReconcileAreVisibleWithoutRawErrors(t *testing.T) {
	setupSpotS1(t)
	db, _ := orm.GetDB("default")
	now := time.Now().UnixMilli()
	for i, status := range []string{StateUnknown, StateReconcile} {
		_, err := db.Exec(`INSERT INTO spot_strategy_orders
 (account_id,owner,symbol,side,order_type,intent,client_order_id,signal_key,
 requested_qty,filled_qty,executed_quote_qty,reserved_qty,status,last_error,created_at,updated_at)
 VALUES('main','auto_strategy','BTCUSDT','BUY','MARKET','open',?,?,'1','0','0','1',?,?,?,?)`,
			"s4-pending-"+status, "s4:signal:"+status, status,
			"https://api.binance.com/api/v3/order?signature=SECRET", now+int64(i), now+int64(i))
		if err != nil {
			t.Fatal(err)
		}
	}
	result, err := ListOrders(context.Background(), db, 0, PageRequest{})
	if err != nil || result.Total != 2 {
		t.Fatalf("pending list %+v %v", result, err)
	}
	for _, item := range result.Items {
		if item.ErrorCategory == "" || strings.Contains(item.ErrorCategory, "SECRET") {
			t.Fatalf("internal network error leaked or status missing: %+v", item)
		}
	}
}

// A BUY reserves quote notional (e.g., 1000 USDT). Remaining unfilled base
// quantity is requested base minus verified executed base, NOT reserved_qty.
func TestS4RemainingQuantityUsesBaseUnits(t *testing.T) {
	setupSpotS1(t)
	db, err := orm.GetDB("default")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO spot_strategy_orders
 (account_id,owner,symbol,side,order_type,intent,client_order_id,signal_key,
 requested_qty,filled_qty,executed_quote_qty,reserved_qty,status,created_at,updated_at)
 VALUES('main','auto_strategy','BTCUSDT','BUY','LIMIT','open','s4-partial-buy',
 's4:partial:btc','10','4','440','1000','partially_filled',1,1)`)
	if err != nil {
		t.Fatal(err)
	}
	result, err := ListOrders(context.Background(), db, 0, PageRequest{})
	if err != nil || len(result.Items) != 1 {
		t.Fatalf("wrong orders %+v %v", result, err)
	}
	order := result.Items[0]
	if order.RemainingQty != "6" || order.ReservedQty != "1000" || order.ErrorCategory != "" {
		t.Fatalf("base remaining must be 6 BTC, quote reservation 1000 USDT: %+v", order)
	}
}

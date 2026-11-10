package spotstrategy

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// Once account/owner are no longer caller-selectable, SQL must still
// reject cross-owner records and local IDs. Do not "fix" this test by
// removing account_id/owner predicates from the financial tables.
func TestS4FixedInternalScopeRejectsForeignInventoryAndFills(t *testing.T) {
	setupSpotS1(t)
	db, err := orm.GetDB("default")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UnixMilli()
	res, err := db.Exec(`INSERT INTO spot_strategy_holdings
  (account_id,owner,symbol,base_asset,quote_asset,owned_qty,avg_cost_quote,status,active_slot,created_at,updated_at)
  VALUES('main','manual','BTCUSDT','BTC','USDT','10','100','active','main:manual:BTCUSDT',?,?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	foreignHolding, _ := res.LastInsertId()
	res, err = db.Exec(`INSERT INTO spot_strategy_orders
  (holding_id,account_id,owner,symbol,side,order_type,intent,client_order_id,exchange_order_id,signal_key,
  requested_qty,filled_qty,executed_quote_qty,reserved_qty,status,created_at,updated_at)
  VALUES (?,'main','manual','BTCUSDT','BUY','LIMIT','open','s4-manual-cid','9933','s4:manual',
  '10','10','1000','0','filled',?,?)`, foreignHolding, now, now)
	if err != nil {
		t.Fatal(err)
	}
	foreignOrder, _ := res.LastInsertId()
	if _, err = db.Exec(`INSERT INTO spot_strategy_fills
  (account_id,symbol,order_id,trade_id,holding_id,client_order_id,side,qty,price,quote_qty,commission,commission_asset,trade_time,rest_verified)
  VALUES ('main','BTCUSDT','9933','foreign-trade',?,'s4-manual-cid','BUY','10','100','1000','1','USDT',?,1)`, foreignHolding, now); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	hs, err := ListHoldings(ctx, db, PageRequest{})
	if err != nil || hs.Total != 0 || len(hs.Items) != 0 {
		t.Fatalf("manual Holdings leaked: %+v %v", hs, err)
	}
	orders, err := ListOrders(ctx, db, 0, PageRequest{})
	if err != nil || orders.Total != 0 || len(orders.Items) != 0 {
		t.Fatalf("manual orders leaked: %+v %v", orders, err)
	}
	if _, err = ListOrders(ctx, db, foreignHolding, PageRequest{}); !errors.Is(err, ErrUnmanagedOrder) {
		t.Fatalf("other owner Holding ID accepted: %v", err)
	}
	if _, err = ListOrderFills(ctx, db, foreignOrder, PageRequest{}); !errors.Is(err, ErrUnmanagedOrder) {
		t.Fatalf("other owner Order ID leaked Fill: %v", err)
	}
}

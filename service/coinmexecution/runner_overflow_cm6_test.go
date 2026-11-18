package coinmexecution

import (
	"context"
	"strconv"
	"testing"
)

func TestCM6UnsettledScanOverflowNeverAuthorizesAnotherOrder(t *testing.T) {
	runner, _, gateway, strategy, db, _ := cm6RunnerFixture(t)
	for i := 0; i < 101; i++ {
		_, err := db.Raw("INSERT INTO coin_m_managed_orders (position_id,symbol,position_side,intent,side,order_type,client_order_id,exchange_order_id,requested_contracts,filled_contracts,status,created_at,updated_at) VALUES (?,?,?,?,?,?,?,NULL,'1','0','UNKNOWN',1,1)",
			int64(1), "BTCUSD_PERP", "LONG", "OPEN", "BUY", "MARKET", "cm6overflow"+strconv.Itoa(i)).Exec()
		if err != nil {
			t.Fatal(err)
		}
	}
	if _, err := runner.unsettled(); err == nil {
		t.Fatal("a truncated pending order scan was treated as complete")
	}
	if err := runner.Tick(context.Background()); err == nil {
		t.Fatal("101 pending COIN-M orders should fail closed")
	}
	if len(gateway.calls) != 0 || strategy.entries != 0 {
		t.Fatalf("scanner overflow sent a new DAPI order: calls=%d signals=%d", len(gateway.calls), strategy.entries)
	}
}

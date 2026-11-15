package futuresownership

import (
	"context"
	"errors"
	"testing"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/client/orm"
)

// A failed userTrades lookup must not be repeated by the closed-cycle pass
// in the same minute. The following pass must still be able to recover.
func TestClosedCycleFailedOrderRESTIsNotRetriedTwiceInOnePass(t *testing.T) {
	prepareOwnershipDB(t)
	settlementConfig(t, 1, 2)
	s := testService()
	ctx := context.Background()
	p := settlementFilled(t, s, IntentOpen, "min_retry_open", "1101", 1, 100, "0", "0.05")
	p = settlementFilled(t, s, IntentClose, "min_retry_close", "1201", 1, 110, "10", "0.07")
	if p.Status != PositionClosed {
		t.Fatalf("expected closed position, got %+v", p)
	}
	o := orm.NewOrm()
	if _, err := o.Raw("UPDATE futures_trade_fills SET rest_verified=0,commission='0',commission_asset='' WHERE order_id='1201'").Exec(); err != nil {
		t.Fatal(err)
	}
	if err := s.SetOrderStatus(ctx, "min_retry_close", OrderReconcile); err != nil {
		t.Fatal(err)
	}
	broker := &stage012Broker{tradeError: errors.New("REST test unavailable")}
	broker.lookupResult = ExchangeOrder{
		ExchangeOrderID: "1201", ClientOrderID: "min_retry_close",
		Status: "FILLED", FilledQty: 1, AveragePrice: 110,
	}
	r := settlementReconciler(s, "USDT")
	r.Executor.Broker = broker
	first, err := r.ReconcileOwner(ctx, OwnerAutoStrategy)
	if err != nil || first.OrdersUnresolved != 1 || broker.tradeCalls != 1 {
		t.Fatalf("expected one failed REST call this pass: summary=%+v calls=%d err=%v", first, broker.tradeCalls, err)
	}
	if got := loadSettlementPosition(t, p.ID); got.SettlementStatus != SettlementIncomplete {
		t.Fatalf("unverified cycle settled: %+v", got)
	}
	broker.tradeError = nil
	broker.trades = []*futures.AccountTrade{
		ledgerTrade(201, 1201, "1", "110", "0.07", "10", futures.PositionSideTypeLong),
	}
	second, err := r.ReconcileOwner(ctx, OwnerAutoStrategy)
	if err != nil || second.OrdersUnresolved != 0 {
		t.Fatalf("recovery in next pass failed: %+v err=%v", second, err)
	}
	if got := loadSettlementPosition(t, p.ID); got.SettlementStatus != SettlementSettled {
		t.Fatalf("verified cycle not settled after recovery: %+v", got)
	}
}

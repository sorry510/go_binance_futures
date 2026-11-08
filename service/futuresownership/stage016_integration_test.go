package futuresownership

import (
	"context"
	"testing"

	"go_binance_futures/service/orderhistory"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/client/orm"
)

// Integration across 01-1..01-5, using the real persistence & projection code,
// a deterministic REST trade broker and an exact-zero mock account source.
// No Binance network or production DB is touched.
func TestStage016EndToEndPartialOpenCancelRestartSettleAndHistory(t *testing.T) {
	prepareOwnershipDB(t)
	settlementConfig(t, 1, 2)
	ctx := context.Background()
	svc := testService()
	broker := &stage012Broker{}
	broker.submitResult = ExchangeOrder{ExchangeOrderID: "801", ClientOrderID: "s510_016_open",
		Status: "PARTIALLY_FILLED", FilledQty: 2, AveragePrice: 100}
	broker.trades = []*futures.AccountTrade{
		ledgerTrade(81001, 801, "2", "100", "0.10", "0", futures.PositionSideTypeLong),
	}
	executor := Executor{Ownership: svc, Broker: broker}
	_, err := executor.Execute(ctx, OrderRequest{Owner: OwnerAutoStrategy, Symbol: "BTCUSDT",
		PositionSide: "LONG", Intent: IntentOpen, Side: "BUY", OrderType: "LIMIT",
		Quantity: 4, Price: 100, ClientOrderID: "s510_016_open"})
	if err != nil {
		t.Fatal(err)
	}
	position, err := svc.GetPosition(ctx, OwnerAutoStrategy, "BTCUSDT", "LONG")
	if err != nil || position.ManagedQty != 2 {
		t.Fatalf("partial opening owns wrong amount: %+v %v", position, err)
	}
	store := orderhistory.Store{DB: orm.NewOrm(), Now: svc.Now}
	page, err := store.List(ctx, orderhistory.Filter{AccountID: "main", Page: 1, Limit: 20})
	if err != nil || page.Total != 1 || len(page.List) != 1 ||
		page.List[0].OpenQty != "2" || page.List[0].RemainingQty != "2" ||
		page.List[0].NetPnL != nil || len(page.PageVerifiedNet) != 0 {
		t.Fatalf("partially opened and unsettled cycle was misrepresented: %+v err=%v", page, err)
	}

	broker.trades = append(broker.trades, ledgerTrade(81002, 801, "2", "120", "0.12", "0", futures.PositionSideTypeLong))
	if _, err := executor.ApplyObservedExchange(ctx, "s510_016_open", ExchangeOrder{
		ExchangeOrderID: "801", Status: "FILLED", FilledQty: 4, AveragePrice: 110,
	}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 100; i++ {
		if _, err := executor.ApplyObservedExchange(ctx, "s510_016_open", ExchangeOrder{
			ExchangeOrderID: "801", Status: "FILLED", FilledQty: 4, AveragePrice: 110,
		}); err != nil {
			t.Fatalf("duplicate event %d broke replay: %v", i, err)
		}
	}
	position, err = svc.GetPosition(ctx, OwnerAutoStrategy, "BTCUSDT", "LONG")
	if err != nil || position.ManagedQty != 4 || position.EntryPrice != 110 {
		t.Fatalf("cumulative average or dedup broken: %+v err=%v", position, err)
	}
	if _, err = svc.ClaimOrder(ctx, ClaimOrderInput{Owner: OwnerAutoStrategy,
		Symbol: "BTCUSDT", PositionSide: "LONG", Intent: IntentClose,
		ClientOrderID: "s510_016_cancel_close", OrderType: "LIMIT", RequestedQty: 4}); err != nil {
		t.Fatal(err)
	}
	broker.trades = append(broker.trades, ledgerTrade(81003, 802, "1", "130", "0.08", "20", futures.PositionSideTypeLong))
	if _, err = executor.ApplyObservedExchange(ctx, "s510_016_cancel_close", ExchangeOrder{
		ExchangeOrderID: "802", Status: "CANCELED", FilledQty: 1, AveragePrice: 130,
	}); err != nil {
		t.Fatal(err)
	}
	position, err = svc.GetPosition(ctx, OwnerAutoStrategy, "BTCUSDT", "LONG")
	if err != nil || position.ManagedQty != 3 {
		t.Fatalf("cancel discarded actual partial close: %+v %v", position, err)
	}
	page, err = store.List(ctx, orderhistory.Filter{Page: 1, Limit: 20})
	if err != nil || len(page.List) != 1 || page.List[0].CloseQty != "1" ||
		page.List[0].RemainingQty != "3" || page.List[0].NetPnL != nil {
		t.Fatalf("partial canceled exit fabricated full close or PnL: %+v %v", page, err)
	}

	// Fresh service/executor, same durable DB -- no in-memory replay state.
	restarted := Service{Now: svc.Now}
	if _, err = restarted.ClaimOrder(ctx, ClaimOrderInput{Owner: OwnerAutoStrategy,
		Symbol: "BTCUSDT", PositionSide: "LONG", Intent: IntentClose,
		ClientOrderID: "s510_016_final_close", OrderType: "MARKET", RequestedQty: 3}); err != nil {
		t.Fatal(err)
	}
	broker.trades = append(broker.trades, ledgerTrade(81004, 803, "3", "140", "0.15", "90", futures.PositionSideTypeLong))
	resumedExecutor := Executor{Ownership: restarted, Broker: broker}
	if _, err = resumedExecutor.ApplyObservedExchange(ctx, "s510_016_final_close", ExchangeOrder{
		ExchangeOrderID: "803", Status: "FILLED", FilledQty: 3, AveragePrice: 140,
	}); err != nil {
		t.Fatal(err)
	}
	// GetPosition intentionally returns active cycles only; read the
	// persisted closed cycle by its original ID.
	position = loadSettlementPosition(t, position.ID)
	if position.ManagedQty != 0 || position.Status != PositionClosed ||
		position.SettlementStatus != SettlementIncomplete {
		t.Fatalf("flat without exchange proof must stay incomplete: %+v %v", position, err)
	}
	before, err := store.List(ctx, orderhistory.Filter{Page: 1, Limit: 20})
	if err != nil || before.Total != 1 || before.List[0].NetPnL != nil ||
		before.List[0].OpenQty != "4" || before.List[0].CloseQty != "4" ||
		before.List[0].OpenAvgPrice == nil || *before.List[0].OpenAvgPrice != "110" {
		t.Fatalf("history recognized PnL before final verified flat: %+v %v", before, err)
	}
	riskBefore := loadSettlementConfig(t)
	if riskBefore.SettlementScaleBalance != 0 {
		t.Fatalf("risk fired before flat verification: %+v", riskBefore)
	}
	r := settlementReconciler(restarted, "USDT")
	r.Executor.Broker = broker
	result, err := r.ReconcileOwner(ctx, OwnerAutoStrategy)
	if err != nil || result.OrdersUnresolved != 0 {
		t.Fatalf("exact flat recon failed: %+v %v", result, err)
	}
	settled := loadSettlementPosition(t, position.ID)
	if settled.SettlementStatus != SettlementSettled || settled.SettlementProcessedAt == 0 {
		t.Fatalf("verified flat did not settle: %+v", settled)
	}
	page, err = store.List(ctx, orderhistory.Filter{AccountID: "main", Page: 1, Limit: 20})
	if err != nil || page.Total != 1 || page.List[0].Completeness != "verified" ||
		page.List[0].NetPnL == nil || *page.List[0].NetPnL != "109.55" ||
		page.List[0].OpenAvgPrice == nil || *page.List[0].OpenAvgPrice != "110" ||
		page.List[0].CloseAvgPrice == nil || *page.List[0].CloseAvgPrice != "137.5" {
		t.Fatalf("settled cycle summary does not match four true Fills: %+v %v", page, err)
	}
	detail, err := store.Details(ctx, "main", position.ID)
	if err != nil || len(detail.Orders) != 3 ||
		len(detail.Orders[0].Fills) != 2 || len(detail.Orders[1].Fills) != 1 ||
		len(detail.Orders[2].Fills) != 1 {
		t.Fatalf("one cycle should have three child orders and four fills: %+v %v", detail, err)
	}
	var count int
	if err = orm.NewOrm().Raw("SELECT COUNT(*) FROM futures_trade_fills WHERE account_id='main' AND position_id=?", position.ID).QueryRow(&count); err != nil || count != 4 {
		t.Fatalf("100 duplicate exchanges inserted duplicate Fill: n=%d err=%v", count, err)
	}
	riskAfter := loadSettlementConfig(t)
	for i := 0; i < 3; i++ {
		if _, err = r.ReconcileOwner(ctx, OwnerAutoStrategy); err != nil {
			t.Fatalf("restart reconcile #%d: %v", i, err)
		}
	}
	riskLast := loadSettlementConfig(t)
	if riskAfter.SettlementScaleBalance != riskLast.SettlementScaleBalance ||
		riskAfter.SettlementConsecutiveLosses != riskLast.SettlementConsecutiveLosses {
		t.Fatalf("verified settlement/replay duplicated risk: before=%+v after=%+v", riskAfter, riskLast)
	}
}

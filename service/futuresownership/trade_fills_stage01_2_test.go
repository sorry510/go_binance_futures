package futuresownership

import (
	"context"
	"errors"
	"math"
	"testing"

	"go_binance_futures/service/tradefills"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/client/orm"
)

type stage012Broker struct {
	fakeOrderBroker
	trades     []*futures.AccountTrade
	tradeError error
	tradeCalls int
}

func (b *stage012Broker) ListAccountTrades(ctx context.Context, symbol string, id, fromID int64, limit int) ([]*futures.AccountTrade, error) {
	b.tradeCalls++
	if b.tradeError != nil {
		return nil, b.tradeError
	}
	result := make([]*futures.AccountTrade, 0)
	for _, trade := range b.trades {
		if trade.Symbol == symbol && trade.OrderID == id && trade.ID >= fromID {
			result = append(result, trade)
		}
	}
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}
func ledgerTrade(id, orderID int64, qty, price, fee, pnl string, side futures.PositionSideType) *futures.AccountTrade {
	return &futures.AccountTrade{ID: id, OrderID: orderID, Symbol: "BTCUSDT", Quantity: qty, Price: price,
		Commission: fee, CommissionAsset: "USDT", RealizedPnl: pnl, Time: 1800000000000 + id,
		PositionSide: side}
}
func TestStage012RESTPartialCumulativeAverageCancelAndClose(t *testing.T) {
	prepareOwnershipDB(t)
	ctx := context.Background()
	s := testService()
	broker := &stage012Broker{}
	broker.submitResult = ExchangeOrder{ExchangeOrderID: "100", ClientOrderID: "s510_part_open", Status: "PARTIALLY_FILLED", FilledQty: 2, AveragePrice: 100}
	broker.trades = []*futures.AccountTrade{ledgerTrade(1, 100, "2", "100", "0.1", "0", futures.PositionSideTypeLong)}
	executor := Executor{Ownership: s, Broker: broker}
	_, err := executor.Execute(ctx, OrderRequest{Owner: OwnerAutoStrategy, Symbol: "BTCUSDT", PositionSide: "LONG", Intent: IntentOpen,
		Side: "BUY", OrderType: "LIMIT", Quantity: 4, Price: 100, ClientOrderID: "s510_part_open"})
	if err != nil {
		t.Fatal(err)
	}
	pos, err := s.GetPosition(ctx, OwnerAutoStrategy, "BTCUSDT", "LONG")
	if err != nil || pos.ManagedQty != 2 || pos.EntryPrice != 100 {
		t.Fatalf("partial OPEN: %+v %v", pos, err)
	}
	broker.trades = append(broker.trades, ledgerTrade(2, 100, "2", "120", "0.12", "0", futures.PositionSideTypeLong))
	_, err = executor.ApplyObservedExchange(ctx, "s510_part_open", ExchangeOrder{ExchangeOrderID: "100", Status: "FILLED", FilledQty: 4, AveragePrice: 110})
	if err != nil {
		t.Fatal(err)
	}
	pos, err = s.GetPosition(ctx, OwnerAutoStrategy, "BTCUSDT", "LONG")
	if err != nil || math.Abs(pos.EntryPrice-110) > 1e-8 || pos.ManagedQty != 4 {
		t.Fatalf("2@100 + 2@120 must average 110, not 105: %+v %v", pos, err)
	}
	order, _ := s.GetOrder(ctx, "s510_part_open")
	got, err := tradefills.Quantity(orm.NewOrm(), order)
	if err != nil || got.RatString() != "4" {
		t.Fatalf("official open fills: %v %v", got, err)
	}
	calls := broker.tradeCalls
	for i := 0; i < 100; i++ {
		if _, err = executor.ApplyObservedExchange(ctx, "s510_part_open", ExchangeOrder{ExchangeOrderID: "100", Status: "FILLED", FilledQty: 4, AveragePrice: 110}); err != nil {
			t.Fatalf("duplicate receipt %d: %v", i, err)
		}
	}
	if broker.tradeCalls != calls {
		t.Fatalf("fully covered order re-fetched REST trades: %d -> %d", calls, broker.tradeCalls)
	}
	// The first close fills one unit, then Binance cancels the rest.
	_, err = s.ClaimOrder(ctx, ClaimOrderInput{Owner: OwnerAutoStrategy, Symbol: "BTCUSDT", PositionSide: "LONG", Intent: IntentClose,
		ClientOrderID: "s510_close_part", OrderType: "LIMIT", RequestedQty: 4})
	if err != nil {
		t.Fatal(err)
	}
	broker.trades = append(broker.trades, ledgerTrade(3, 200, "1", "130", "0.08", "30", futures.PositionSideTypeLong))
	_, err = executor.ApplyObservedExchange(ctx, "s510_close_part", ExchangeOrder{ExchangeOrderID: "200", Status: "CANCELED", FilledQty: 1, AveragePrice: 130})
	if err != nil {
		t.Fatal(err)
	}
	pos, err = s.GetPosition(ctx, OwnerAutoStrategy, "BTCUSDT", "LONG")
	if err != nil || pos.ManagedQty != 3 {
		t.Fatalf("canceled remainder removed a real partial fill: %+v %v", pos, err)
	}
	order, _ = s.GetOrder(ctx, "s510_close_part")
	if order.Status != OrderCanceled || order.PositionID == nil || *order.PositionID != pos.ID {
		t.Fatalf("partial canceled close: %+v", order)
	}
	got, err = tradefills.Quantity(orm.NewOrm(), order)
	if err != nil || got.RatString() != "1" {
		t.Fatalf("canceled close fill lost: %v %v", got, err)
	}
}

func TestStage012RESTGapFailsClosedAndRecovers(t *testing.T) {
	prepareOwnershipDB(t)
	ctx := context.Background()
	s := testService()
	broker := &stage012Broker{tradeError: errors.New("REST temporarily unavailable")}
	e := Executor{Ownership: s, Broker: broker}
	claimOpen(t, s, OwnerAutoStrategy, "BTCUSDT", "SHORT", "s510_missing", 2)
	_, err := e.ApplyObservedExchange(ctx, "s510_missing", ExchangeOrder{ExchangeOrderID: "400", Status: "FILLED", FilledQty: 2, AveragePrice: 50})
	if err == nil {
		t.Fatal("must surface missing immutable trade details")
	}
	managed, _ := s.GetOrder(ctx, "s510_missing")
	if managed.Status != OrderReconcile || managed.FilledQty != 2 {
		t.Fatalf("unknown details marked complete: %+v", managed)
	}
	broker.tradeError = nil
	broker.trades = []*futures.AccountTrade{ledgerTrade(4, 400, "2", "50", "0.2", "0", futures.PositionSideTypeShort)}
	_, err = e.ApplyObservedExchange(ctx, "s510_missing", ExchangeOrder{ExchangeOrderID: "400", Status: "FILLED", FilledQty: 2, AveragePrice: 50})
	if err != nil {
		t.Fatalf("REST recovery failed: %v", err)
	}
	managed, _ = s.GetOrder(ctx, "s510_missing")
	if managed.Status != OrderFilled {
		t.Fatalf("complete official trades not reconciled: %+v", managed)
	}
	got, err := tradefills.Quantity(orm.NewOrm(), managed)
	if err != nil || got.RatString() != "2" {
		t.Fatalf("REST gap not recovered: %v %v", got, err)
	}
}

func TestStage012AlgoActualOrderIDAndTradeAttribution(t *testing.T) {
	for _, tc := range []struct {
		intent string
		name   string
	}{{IntentTakeProfit, "tp"}, {IntentStopLoss, "sl"}} {
		t.Run(tc.name, func(t *testing.T) {
			prepareOwnershipDB(t)
			ctx := context.Background()
			s := testService()
			opener := claimOpen(t, s, OwnerAgentTrade, "BTCUSDT", "LONG", "s510_open_"+tc.name, 1)
			_, err := s.ApplyFill(ctx, opener.ClientOrderID, 1, 100)
			if err != nil {
				t.Fatal(err)
			}
			_, err = s.ClaimOrder(ctx, ClaimOrderInput{Owner: OwnerAgentTrade, Symbol: "BTCUSDT", PositionSide: "LONG",
				Intent: tc.intent, ClientOrderID: "s510_algo_" + tc.name, OrderType: map[string]string{IntentTakeProfit: "TAKE_PROFIT_MARKET", IntentStopLoss: "STOP_MARKET"}[tc.intent], RequestedQty: 1})
			if err != nil {
				t.Fatal(err)
			}
			broker := &stage012Broker{trades: []*futures.AccountTrade{ledgerTrade(8, 601, "1", "120", "0.02", "20", futures.PositionSideTypeLong)}}
			executor := Executor{Ownership: s, Broker: broker}
			_, err = executor.ApplyObservedExchange(ctx, "s510_algo_"+tc.name, ExchangeOrder{ExchangeOrderID: "501",
				ActualOrderID: "601", Status: "FILLED", FilledQty: 1, AveragePrice: 120})
			if err != nil {
				t.Fatal(err)
			}
			managed, _ := s.GetOrder(ctx, "s510_algo_"+tc.name)
			if managed.ActualOrderID == nil || *managed.ActualOrderID != "601" || managed.ExchangeOrderID != "501" {
				t.Fatalf("Algo ID and actual order ID conflated: %+v", managed)
			}
			got, err := tradefills.Quantity(orm.NewOrm(), managed)
			if err != nil || got.RatString() != "1" {
				t.Fatalf("Algo fill association: %v %v", got, err)
			}
			if err = s.SetActualOrderID(ctx, "s510_algo_"+tc.name, "699"); err == nil {
				t.Fatal("cannot remap verified algo to unrelated exchange order")
			}
		})
	}
}

func TestStage012FillDBFailureKeepsRecoverableReconcile(t *testing.T) {
	prepareOwnershipDB(t)
	ctx := context.Background()
	s := testService()
	broker := &stage012Broker{trades: []*futures.AccountTrade{ledgerTrade(91, 999, "2", "101", "0.1", "0", futures.PositionSideTypeLong)}}
	executor := Executor{Ownership: s, Broker: broker}
	open := claimOpen(t, s, OwnerAutoStrategy, "BTCUSDT", "LONG", "s510_db_fail", 2)
	o := orm.NewOrm()
	if _, err := o.Raw(`CREATE TRIGGER fail_s510_fill BEFORE INSERT ON futures_trade_fills
       BEGIN SELECT RAISE(ABORT,'forced trade ledger failure'); END`).Exec(); err != nil {
		t.Fatal(err)
	}
	_, err := executor.ApplyObservedExchange(ctx, open.ClientOrderID, ExchangeOrder{
		ExchangeOrderID: "999", Status: "FILLED", FilledQty: 2, AveragePrice: 101})
	if _, dropErr := o.Raw("DROP TRIGGER IF EXISTS fail_s510_fill").Exec(); dropErr != nil {
		t.Fatal(dropErr)
	}
	if err == nil {
		t.Fatal("ledger failure must not be reported as fully settled")
	}
	order, loadErr := s.GetOrder(ctx, open.ClientOrderID)
	if loadErr != nil || order.Status != OrderReconcile || order.FilledQty != 2 {
		t.Fatalf("missing real fill must leave recoverable order: %+v %v", order, loadErr)
	}
	_, err = executor.ApplyObservedExchange(ctx, open.ClientOrderID, ExchangeOrder{
		ExchangeOrderID: "999", Status: "FILLED", FilledQty: 2, AveragePrice: 101})
	if err != nil {
		t.Fatalf("after DB fault, recovery must be possible: %v", err)
	}
	order, _ = s.GetOrder(ctx, open.ClientOrderID)
	got, err := tradefills.Quantity(o, order)
	if err != nil || got.RatString() != "2" || order.Status != OrderFilled {
		t.Fatalf("fill DB recovery failed: %+v qty=%v err=%v", order, got, err)
	}
}

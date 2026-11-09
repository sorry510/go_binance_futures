package futuresownership

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/beego/beego/v2/client/orm"
)

func TestTerminalReceiptWithoutQuantityRemainsRecoverable(t *testing.T) {
	prepareOwnershipDB(t)
	ctx := context.Background()
	s := testService()
	claimOpen(t, s, OwnerAutoStrategy, "HYPEUSDT", "SHORT", "aut_hype_missing_fill", 1.04)
	executor := Executor{Ownership: s, Broker: &fakeOrderBroker{}}
	_, err := executor.ApplyObservedExchange(ctx, "aut_hype_missing_fill", ExchangeOrder{
		ClientOrderID: "aut_hype_missing_fill", ExchangeOrderID: "5001",
		Status: "FILLED", FilledQty: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	order, err := s.GetOrder(ctx, "aut_hype_missing_fill")
	if err != nil || order.Status != OrderReconcile {
		t.Fatalf("unknown fill must remain outstanding across restart: %+v err=%v", order, err)
	}
	if _, err := s.GetPosition(ctx, OwnerAutoStrategy, "HYPEUSDT", "SHORT"); err != orm.ErrNoRows {
		t.Fatalf("unconfirmed fill must not create ownership: %v", err)
	}

	_, err = executor.ApplyObservedExchange(ctx, "aut_hype_missing_fill", ExchangeOrder{
		ClientOrderID: "aut_hype_missing_fill", ExchangeOrderID: "5001",
		Status: "FILLED", FilledQty: 1.04, AveragePrice: 18.5,
	})
	if err != nil {
		t.Fatal(err)
	}
	order, err = s.GetOrder(ctx, "aut_hype_missing_fill")
	position, posErr := s.GetPosition(ctx, OwnerAutoStrategy, "HYPEUSDT", "SHORT")
	if err != nil || posErr != nil || order.Status != OrderFilled || position.ManagedQty != 1.04 {
		t.Fatalf("later confirmed fill must restore recorded ownership: order=%+v position=%+v errors %v %v", order, position, err, posErr)
	}
}

// No test broker, nor a production adapter, may silently certify an order
// whose matching REST trade reader is missing.
type reviewNoTradeReader struct{}

func (reviewNoTradeReader) Submit(context.Context, OrderRequest, string) (ExchangeOrder, error) {
	return ExchangeOrder{}, fmt.Errorf("unused")
}
func (reviewNoTradeReader) Lookup(context.Context, string, string, string) (ExchangeOrder, error) {
	return ExchangeOrder{}, fmt.Errorf("unused")
}
func (reviewNoTradeReader) Cancel(context.Context, string, int64, string) error { return nil }

func TestStageReviewNoTradeReaderMustFailClosed(t *testing.T) {
	prepareOwnershipDB(t)
	s := testService()
	claimOpen(t, s, OwnerAutoStrategy, "BTCUSDT", "LONG", "review_missing_reader", 1)
	e := Executor{Ownership: s, Broker: reviewNoTradeReader{}}
	_, err := e.ApplyObservedExchange(context.Background(), "review_missing_reader", ExchangeOrder{
		ExchangeOrderID: "55511", FilledQty: 1, AveragePrice: 101, Status: "FILLED"})
	if err == nil || !strings.Contains(err.Error(), "cannot verify official exchange trades") {
		t.Fatalf("no reader treated as successfully verified: %v", err)
	}
	row, err := s.GetOrder(context.Background(), "review_missing_reader")
	if err != nil || row.Status != OrderReconcile {
		t.Fatalf("missing trade reader did not reserve reconciliation: %+v err=%v", row, err)
	}
	var count int
	if err := orm.NewOrm().Raw("SELECT COUNT(*) FROM futures_trade_fills").QueryRow(&count); err != nil || count != 0 {
		t.Fatalf("missing reader manufactured %d trade fills: %v", count, err)
	}
}

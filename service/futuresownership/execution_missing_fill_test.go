package futuresownership

import (
	"context"
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

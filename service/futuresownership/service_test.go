package futuresownership

import (
	"context"
	"testing"
	"time"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
)

func prepareOwnershipDB(t *testing.T) {
	t.Helper()
	_ = orm.RegisterDriver("sqlite3", orm.DRSqlite)
	if _, err := orm.GetDB("default"); err != nil {
		if err := orm.RegisterDataBase("default", "sqlite3", "file:ownership_test?mode=memory&cache=shared"); err != nil {
			t.Fatal(err)
		}
		orm.RegisterModel(new(models.FuturesManagedPosition), new(models.FuturesManagedOrder))
		if err := orm.RunSyncdb("default", false, false); err != nil {
			t.Fatal(err)
		}
	}
	o := orm.NewOrm()
	if _, err := o.Raw("DELETE FROM futures_managed_orders").Exec(); err != nil {
		t.Fatal(err)
	}
	if _, err := o.Raw("DELETE FROM futures_managed_positions").Exec(); err != nil {
		t.Fatal(err)
	}
}

func testService() Service {
	return Service{Now: func() time.Time { return time.UnixMilli(1_800_000_000_000).UTC() }}
}

func claimOpen(t *testing.T, service Service, owner, symbol, side, clientID string, qty float64) models.FuturesManagedOrder {
	t.Helper()
	row, err := service.ClaimOrder(context.Background(), ClaimOrderInput{
		Owner: owner, Symbol: symbol, PositionSide: side, Intent: IntentOpen,
		ClientOrderID: clientID, RequestedQty: qty, OrderType: "MARKET", SourceRef: "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	return row
}

func TestUnknownAccountPositionIsNeverClaimed(t *testing.T) {
	prepareOwnershipDB(t)
	_, err := testService().GetPosition(context.Background(), OwnerAutoStrategy, "BTCUSDT", "LONG")
	if err != orm.ErrNoRows {
		t.Fatalf("missing ownership must remain unmanaged, got %v", err)
	}
}

func TestClaimOrderIsIdempotentAndRejectsOtherOwner(t *testing.T) {
	prepareOwnershipDB(t)
	service := testService()
	first := claimOpen(t, service, OwnerAutoStrategy, "BTCUSDT", "LONG", "auto_1", 2)
	second := claimOpen(t, service, OwnerAutoStrategy, "BTCUSDT", "LONG", "auto_1", 2)
	if first.ID != second.ID {
		t.Fatalf("same client id must be idempotent: %d != %d", first.ID, second.ID)
	}
	if _, err := service.ClaimOrder(context.Background(), ClaimOrderInput{Owner: OwnerAgentTrade, Symbol: "BTCUSDT", PositionSide: "LONG", Intent: IntentOpen, ClientOrderID: "agent_1", RequestedQty: 1, OrderType: "MARKET"}); err == nil {
		t.Fatal("another owner must not claim the same symbol/side while a managed open order is live")
	}
}

func TestPartialFillOwnsOnlyActuallyFilledQuantity(t *testing.T) {
	prepareOwnershipDB(t)
	service := testService()
	claimOpen(t, service, OwnerAutoStrategy, "ETHUSDT", "LONG", "auto_eth", 10)
	position, err := service.ApplyFill(context.Background(), "auto_eth", 4, 100)
	if err != nil {
		t.Fatal(err)
	}
	if position.ManagedQty != 4 {
		t.Fatalf("managed qty=%v, want actual fill 4", position.ManagedQty)
	}
	position, err = service.ApplyFill(context.Background(), "auto_eth", 7, 110)
	if err != nil {
		t.Fatal(err)
	}
	if position.ManagedQty != 7 {
		t.Fatalf("cumulative fill must add delta only, got %v", position.ManagedQty)
	}
}

func TestReconcileNeverClaimsManualIncreaseAndShrinksAfterManualReduce(t *testing.T) {
	prepareOwnershipDB(t)
	service := testService()
	claimOpen(t, service, OwnerAutoStrategy, "SOLUSDT", "SHORT", "auto_sol", 5)
	if _, err := service.ApplyFill(context.Background(), "auto_sol", 5, 50); err != nil {
		t.Fatal(err)
	}
	position, err := service.ReconcilePosition(context.Background(), OwnerAutoStrategy, "SOLUSDT", "SHORT", 8)
	if err != nil {
		t.Fatal(err)
	}
	if position.ManagedQty != 5 {
		t.Fatalf("manual add must stay unmanaged, managed qty=%v", position.ManagedQty)
	}
	position, err = service.ReconcilePosition(context.Background(), OwnerAutoStrategy, "SOLUSDT", "SHORT", 3)
	if err != nil {
		t.Fatal(err)
	}
	if position.ManagedQty != 3 {
		t.Fatalf("manual reduce must shrink managed qty, got %v", position.ManagedQty)
	}
	qty, err := service.CloseQuantity(context.Background(), OwnerAutoStrategy, "SOLUSDT", "SHORT", 10)
	if err != nil || qty != 3 {
		t.Fatalf("close qty must be capped to managed qty: qty=%v err=%v", qty, err)
	}
	position, err = service.ReconcilePosition(context.Background(), OwnerAutoStrategy, "SOLUSDT", "SHORT", 0)
	if err != nil {
		t.Fatal(err)
	}
	if position.Status != PositionClosed || position.ManagedQty != 0 {
		t.Fatalf("zero account qty must close ownership: %+v", position)
	}
}

func TestCloseFillReducesOnlyManagedQuantity(t *testing.T) {
	prepareOwnershipDB(t)
	service := testService()
	claimOpen(t, service, OwnerAgentTrade, "BNBUSDT", "LONG", "agt_open", 5)
	if _, err := service.ApplyFill(context.Background(), "agt_open", 5, 600); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ClaimOrder(context.Background(), ClaimOrderInput{Owner: OwnerAgentTrade, Symbol: "BNBUSDT", PositionSide: "LONG", Intent: IntentClose, ClientOrderID: "agt_close", RequestedQty: 3, OrderType: "MARKET"}); err != nil {
		t.Fatal(err)
	}
	position, err := service.ApplyFill(context.Background(), "agt_close", 3, 610)
	if err != nil {
		t.Fatal(err)
	}
	if position.ManagedQty != 2 || position.Status != PositionActive {
		t.Fatalf("close fill should leave managed qty 2: %+v", position)
	}
}

func TestMutationRequiresOwnedQuantity(t *testing.T) {
	prepareOwnershipDB(t)
	service := testService()
	_, err := service.ClaimOrder(context.Background(), ClaimOrderInput{Owner: OwnerAutoStrategy, Symbol: "BTCUSDT", PositionSide: "LONG", Intent: IntentClose, ClientOrderID: "close_unmanaged", RequestedQty: 1, OrderType: "MARKET"})
	if err == nil {
		t.Fatal("unmanaged position must not be mutable")
	}
	claimOpen(t, service, OwnerAutoStrategy, "BTCUSDT", "LONG", "open_owned", 2)
	if _, err := service.ApplyFill(context.Background(), "open_owned", 2, 100); err != nil {
		t.Fatal(err)
	}
	_, err = service.ClaimOrder(context.Background(), ClaimOrderInput{Owner: OwnerAutoStrategy, Symbol: "BTCUSDT", PositionSide: "LONG", Intent: IntentClose, ClientOrderID: "close_too_much", RequestedQty: 3, OrderType: "MARKET"})
	if err == nil {
		t.Fatal("mutation quantity above managed qty must be rejected")
	}
}

func TestSameOwnerCannotAddOrCreateSecondOpenOrder(t *testing.T) {
	prepareOwnershipDB(t)
	service := testService()
	claimOpen(t, service, OwnerAutoStrategy, "ADAUSDT", "LONG", "auto_ada_1", 2)
	if _, err := service.ClaimOrder(context.Background(), ClaimOrderInput{Owner: OwnerAutoStrategy, Symbol: "ADAUSDT", PositionSide: "LONG", Intent: IntentOpen, ClientOrderID: "auto_ada_2", RequestedQty: 1, OrderType: "MARKET"}); err == nil {
		t.Fatal("same owner must not create a second live open order for the same symbol/side")
	}
	if _, err := service.ApplyFill(context.Background(), "auto_ada_1", 2, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ClaimOrder(context.Background(), ClaimOrderInput{Owner: OwnerAutoStrategy, Symbol: "ADAUSDT", PositionSide: "LONG", Intent: IntentOpen, ClientOrderID: "auto_ada_3", RequestedQty: 1, OrderType: "MARKET"}); err == nil {
		t.Fatal("same owner must not add to an active managed position in V3 first version")
	}
}

func TestClaimOrderRejectsSecondLiveCloseButAllowsRetryAfterTerminal(t *testing.T) {
	prepareOwnershipDB(t)
	service := testService()
	claimOpen(t, service, OwnerAutoStrategy, "AVAXUSDT", "SHORT", "aut_avax_open", 2)
	if _, err := service.ApplyFill(context.Background(), "aut_avax_open", 2, 20); err != nil {
		t.Fatal(err)
	}
	first := ClaimOrderInput{Owner: OwnerAutoStrategy, Symbol: "AVAXUSDT", PositionSide: "SHORT", Intent: IntentClose, ClientOrderID: "aut_avax_close_1", RequestedQty: 2, OrderType: "MARKET"}
	if _, err := service.ClaimOrder(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	second := first
	second.ClientOrderID = "aut_avax_close_2"
	if _, err := service.ClaimOrder(context.Background(), second); err == nil {
		t.Fatal("second live close for same owner/symbol/side must be rejected")
	}
	if err := service.SetOrderStatus(context.Background(), first.ClientOrderID, OrderCanceled); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ClaimOrder(context.Background(), second); err != nil {
		t.Fatalf("terminal close must allow a later remaining-quantity attempt: %v", err)
	}
}

package binance

import (
	"go_binance_futures/service/tradefills"
	"path/filepath"
	"testing"

	"go_binance_futures/models"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
)

func TestStage2UserDataMirrorIsolatedForSameSymbolAndOrderID(t *testing.T) {
	_ = orm.RegisterDriver("sqlite3", orm.DRSqlite)
	if _, err := orm.GetDB("default"); err != nil {
		if err := orm.RegisterDataBase("default", "sqlite3", filepath.Join(t.TempDir(), "default.db")); err != nil {
			t.Fatal(err)
		}
	}
	alias := "stage2_user_data_mirror"
	if err := orm.RegisterDataBase(alias, "sqlite3", filepath.Join(t.TempDir(), "mirror.db")); err != nil {
		t.Fatal(err)
	}
	orm.RegisterModel(new(models.FuturesOrder), new(models.FuturesPosition), new(models.Symbols), new(models.FuturesManagedOrder))
	if err := orm.RunSyncdb(alias, false, false); err != nil {
		t.Fatal(err)
	}
	o := orm.NewOrmUsingDB(alias)
	if _, err := o.Raw(tradefills.SchemaDDL).Exec(); err != nil {
		t.Fatal(err)
	}
	sdk := futures.NewClient("fake", "fake")
	lead, err := NewAccountClient(LeadAccountID, sdk)
	if err != nil {
		t.Fatal(err)
	}
	leadOrder := func(qty string, when int64) *futures.WsUserDataEvent {
		return &futures.WsUserDataEvent{
			Event: futures.UserDataEventTypeOrderTradeUpdate, Time: when,
			WsUserDataOrderTradeUpdate: futures.WsUserDataOrderTradeUpdate{
				OrderTradeUpdate: futures.WsOrderTradeUpdate{
					Symbol: "BTCUSDT", ID: 777, ClientOrderID: "fakeorder", Side: futures.SideTypeBuy,
					PositionSide: futures.PositionSideTypeLong, Type: futures.OrderTypeMarket,
					Status: futures.OrderStatusTypeFilled, OriginalQty: "1", AccumulatedFilledQty: qty,
				},
			},
		}
	}
	for _, tc := range []struct {
		id     AccountID
		client *AccountClient
		qty    string
	}{
		{MainAccountID, nil, "0.5"}, {LeadAccountID, lead, "0.8"},
	} {
		if err := persistFuturesAccountEventWithDB(o, tc.id, tc.client, leadOrder(tc.qty, 100)); err != nil {
			t.Fatal(err)
		}
	}
	check := func(id string, want string) {
		t.Helper()
		var rows []models.FuturesOrder
		if _, err := o.QueryTable(new(models.FuturesOrder)).Filter("account_id", id).Filter("order_id", "777").All(&rows); err != nil {
			t.Fatal(err)
		}
		if len(rows) != 1 || rows[0].ExecutedQty != want {
			t.Fatalf("order mirror %s rows=%+v want=%s", id, rows, want)
		}
	}
	check("main", "0.5")
	check("lead", "0.8")
	if err := persistFuturesAccountEventWithDB(o, LeadAccountID, lead, leadOrder("1.0", 110)); err != nil {
		t.Fatal(err)
	}
	check("main", "0.5")
	check("lead", "1.0")
	if err := persistFuturesAccountEventWithDB(o, LeadAccountID, lead, leadOrder("0.1", 90)); err != nil {
		t.Fatal(err)
	}
	check("lead", "1.0")
	for _, tc := range []struct {
		id     AccountID
		client *AccountClient
		qty    string
	}{
		{MainAccountID, nil, "0.1"}, {LeadAccountID, lead, "0.2"},
	} {
		evt := &futures.WsUserDataEvent{Event: futures.UserDataEventTypeAccountUpdate, Time: 200,
			WsUserDataAccountUpdate: futures.WsUserDataAccountUpdate{AccountUpdate: futures.WsAccountUpdate{Positions: []futures.WsPosition{
				{Symbol: "BTCUSDT", Side: futures.PositionSideTypeLong, Amount: tc.qty, EntryPrice: "30000", MarkPrice: "31000"},
			}}}}
		if err := persistFuturesAccountEventWithDB(o, tc.id, tc.client, evt); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		id   string
		want string
	}{{"main", "0.1"}, {"lead", "0.2"}} {
		var positions []models.FuturesPosition
		if _, err := o.QueryTable("futures_positions").Filter("account_id", tc.id).Filter("symbol", "BTCUSDT").All(&positions); err != nil {
			t.Fatal(err)
		}
		if len(positions) != 1 || positions[0].Amount != tc.want {
			t.Fatalf("mirror positions %s: %+v", tc.id, positions)
		}
	}
	if err := persistFuturesAccountEventWithDB(o, "unknown", lead, leadOrder("3.0", 300)); err == nil {
		t.Fatal("unknown account must fail")
	}
	if err := persistFuturesAccountEventWithDB(o, LeadAccountID, nil, leadOrder("3.0", 300)); err == nil {
		t.Fatal("nil lead client must fail")
	}
	// The ORDER_TRADE_UPDATE timestamp is older than the mirror's current
	// order state, but the unique exchange execution must still persist.
	late := leadOrder("0.5", 95)
	late.OrderTradeUpdate.ExecutionType = futures.OrderExecutionType("TRADE")
	late.OrderTradeUpdate.TradeID = 999
	late.OrderTradeUpdate.LastFilledQty = "0.5"
	late.OrderTradeUpdate.LastFilledPrice = "100"
	late.OrderTradeUpdate.Commission = "0.05"
	late.OrderTradeUpdate.CommissionAsset = "USDT"
	late.OrderTradeUpdate.RealizedPnL = "1"
	late.OrderTradeUpdate.TradeTime = 95
	for i := 0; i < 100; i++ {
		if err := persistFuturesAccountEventWithDB(o, LeadAccountID, lead, late); err != nil {
			t.Fatal(err)
		}
	}
	var fills int64
	if err := o.Raw("SELECT COUNT(*) FROM futures_trade_fills WHERE account_id='lead' AND order_id='777' AND trade_id='999'").QueryRow(&fills); err != nil || fills != 1 {
		t.Fatalf("late duplicate WS events should persist one fill: %d err=%v", fills, err)
	}
	check("lead", "1.0")
}

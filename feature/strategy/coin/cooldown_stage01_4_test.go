package coin

import (
	"fmt"
	"testing"
	"time"

	"go_binance_futures/models"
	"go_binance_futures/service/tradefills"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
)

func TestStage014LegacyCoinSelectorUsesManagedCloseWithAccountBoundary(t *testing.T) {
	_ = orm.RegisterDriver("sqlite3", orm.DRSqlite)
	orm.RegisterModel(new(models.FuturesManagedOrder))
	alias := "s510_stage014_coin"
	if err := orm.RegisterDataBase(alias, "sqlite3", fmt.Sprintf("file:%s?mode=memory&cache=shared", alias)); err != nil {
		t.Fatal(err)
	}
	if err := orm.RunSyncdb(alias, true, false); err != nil {
		t.Fatal(err)
	}
	o := orm.NewOrmUsingDB(alias)
	if _, err := o.Raw(tradefills.SchemaDDL).Exec(); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UnixMilli()
	data := []struct {
		account, symbol, intent string
		qty                     float64
		delta                   int64
	}{
		{"main", "CLOSEUSDT", "close", 1, -60_000},
		{"main", "PARTIALUSDT", "stop_loss", 0.2, -30_000},
		{"main", "ACKUSDT", "close", 0, -30_000},
		{"main", "OPENUSDT", "open", 1, -30_000},
		{"main", "STALEUSDT", "close", 1, -10 * 60_000},
		{"lead", "LEADUSDT", "close", 1, -30_000},
		{"main", "LATEFILLUSDT", "close", 1, -20 * 60_000},
		{"main", "REFRESHEDUSDT", "close", 1, -10_000},
	}
	for i, entry := range data {
		row := &models.FuturesManagedOrder{
			AccountID: entry.account, Owner: "auto_strategy",
			Symbol: entry.symbol, PositionSide: "LONG", Intent: entry.intent,
			ClientOrderID: fmt.Sprintf("stage014-%d", i), RequestedQty: 1,
			FilledQty: entry.qty, OrderType: "MARKET", Status: "filled",
			CreatedAt: now + entry.delta, UpdatedAt: now + entry.delta,
		}
		if _, err := o.Insert(row); err != nil {
			t.Fatal(err)
		}
		if entry.qty > 0 {
			tradeTime := now + entry.delta
			if entry.symbol == "LATEFILLUSDT" {
				tradeTime = now - 10_000
			}
			if entry.symbol == "REFRESHEDUSDT" {
				tradeTime = now - 60*60_000
			}
			orderID := fmt.Sprintf("%d", i+1)
			if _, err := o.Raw(`INSERT INTO futures_trade_fills
                (market,account_id,symbol,order_id,trade_id,client_order_id,intent,position_side,
                qty,price,commission,commission_asset,realized_pnl,pnl_asset,trade_time,rest_verified)
                VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
				tradefills.MarketUSDM, entry.account, entry.symbol, orderID, orderID,
				row.ClientOrderID, entry.intent, "LONG", fmt.Sprintf("%.8f", entry.qty),
				"100", "0", "USDT", "0", "USDT", tradeTime, 1).Exec(); err != nil {
				t.Fatal(err)
			}
		}
	}
	symbols, err := getLimitMinManagedCloseWithOrm(o, 5)
	if err != nil {
		t.Fatal(err)
	}
	if !symbols["LATEFILLUSDT"] || symbols["REFRESHEDUSDT"] {
		t.Fatalf("late fill / old refreshed trade is based on mutable updated_at: %+v", symbols)
	}
	if !symbols["CLOSEUSDT"] || !symbols["PARTIALUSDT"] {
		t.Fatalf("real main close / partial stop fill was not excluded: %+v", symbols)
	}
	for _, allowed := range []string{"ACKUSDT", "OPENUSDT", "STALEUSDT", "LEADUSDT"} {
		if symbols[allowed] {
			t.Fatalf("invalid cooldown for %s: %+v", allowed, symbols)
		}
	}
	if _, err := o.Raw("DROP TABLE futures_trade_fills").Exec(); err != nil {
		t.Fatal(err)
	}
	if _, err := getLimitMinManagedCloseWithOrm(o, 5); err == nil {
		t.Fatal("selector DB error must not silently permit fresh entries")
	}
}

package feature

import (
	"testing"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
)

func TestCreateTestResultAllowsOnlyOneOpenRowPerSymbol(t *testing.T) {
	_ = orm.RegisterDriver("sqlite3", orm.DRSqlite)
	if err := orm.RegisterDataBase("default", "sqlite3", "file:test_strategy_open_guard?mode=memory&cache=shared"); err != nil {
		t.Fatal(err)
	}
	orm.RegisterModel(new(models.TestStrategyResults))
	if err := orm.RunSyncdb("default", true, false); err != nil {
		t.Fatal(err)
	}

	coin := &models.Symbols{
		Symbol: "GUARDUSDT", Leverage: 1, TickSize: "0.01", StepSize: "0.001",
		Usdt: "10", Profit: "20", Loss: "20",
	}
	first, created, err := createTestResult(coin, 100, "LONG", "entry", "long", "NowPrice > 1", 0.001)
	if err != nil || !created || first == nil {
		t.Fatalf("first open failed: created=%v row=%+v err=%v", created, first, err)
	}
	second, created, err := createTestResult(coin, 101, "LONG", "entry2", "long", "NowPrice > 2", 0.001)
	if err != nil || created || second != nil {
		t.Fatalf("second open must be rejected: created=%v row=%+v err=%v", created, second, err)
	}

	count, err := orm.NewOrm().QueryTable("test_strategy_results").Filter("symbol", coin.Symbol).Filter("close_price", "0").Count()
	if err != nil || count != 1 {
		t.Fatalf("expected exactly one open row, count=%d err=%v", count, err)
	}
}

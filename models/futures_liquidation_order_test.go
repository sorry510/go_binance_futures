package models

import (
	"path/filepath"
	"testing"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
)

func TestFuturesLiquidationOrderSchema(t *testing.T) {
	const alias = "futures_liquidation_order_schema"

	if err := orm.RegisterDriver("sqlite3", orm.DRSqlite); err != nil {
		t.Fatal(err)
	}
	if err := orm.RegisterDataBase(alias, "sqlite3", filepath.Join(t.TempDir(), "test.db")); err != nil {
		t.Fatal(err)
	}
	orm.RegisterModel(new(FuturesLiquidationOrder))

	if err := orm.RunSyncdb(alias, false, false); err != nil {
		t.Fatal(err)
	}

	exists := orm.NewOrmUsingDB(alias).QueryTable(new(FuturesLiquidationOrder)).Exist()
	if exists {
		t.Fatal("new liquidation order table should be empty")
	}
}

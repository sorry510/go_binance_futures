package models

import (
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
)

var futuresLiquidationTestOnce sync.Once
var futuresLiquidationTestSeq atomic.Uint64

func TestFuturesLiquidationOrderSchema(t *testing.T) {
	alias := fmt.Sprintf("futures_liquidation_order_schema_%d", futuresLiquidationTestSeq.Add(1))

	if err := orm.RegisterDriver("sqlite3", orm.DRSqlite); err != nil {
		t.Fatal(err)
	}
	if err := orm.RegisterDataBase(alias, "sqlite3", filepath.Join(t.TempDir(), "test.db")); err != nil {
		t.Fatal(err)
	}
	futuresLiquidationTestOnce.Do(func() { orm.RegisterModel(new(FuturesLiquidationOrder)) })

	if err := orm.RunSyncdb(alias, false, false); err != nil {
		t.Fatal(err)
	}

	exists := orm.NewOrmUsingDB(alias).QueryTable(new(FuturesLiquidationOrder)).Exist()
	if exists {
		t.Fatal("new liquidation order table should be empty")
	}
}

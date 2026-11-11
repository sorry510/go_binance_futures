package market

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
)

var marketHistoryTestOnce sync.Once
var marketHistoryTestErr error
var marketHistoryTestSeq atomic.Uint64

func setupMarketHistoryTest(t *testing.T) string {
	t.Helper()
	marketHistoryTestOnce.Do(func() {
		marketHistoryTestErr = orm.RegisterDriver("sqlite3", orm.DRSqlite)
		if marketHistoryTestErr != nil {
			return
		}
		if _, err := orm.GetDB("default"); err != nil {
			marketHistoryTestErr = orm.RegisterDataBase("default", "sqlite3", "file:market_history_bootstrap?mode=memory&cache=shared")
			if marketHistoryTestErr != nil {
				return
			}
		}
		orm.RegisterModel(new(models.Config), new(models.MarketConditionHistory))
	})
	if marketHistoryTestErr != nil {
		t.Fatal(marketHistoryTestErr)
	}
	alias := fmt.Sprintf("market_history_test_%d", marketHistoryTestSeq.Add(1))
	if err := orm.RegisterDataBase(alias, "sqlite3", filepath.Join(t.TempDir(), "market-history.db")); err != nil {
		t.Fatal(err)
	}
	if err := orm.RunSyncdb(alias, false, false); err != nil {
		t.Fatal(err)
	}
	return alias
}

func TestSaveMarketConditionAppendsHistoryForEverySave(t *testing.T) {
	alias := setupMarketHistoryTest(t)
	o := orm.NewOrmUsingDB(alias)
	_, _ = o.Raw("DELETE FROM market_condition_histories").Exec()
	_, _ = o.Raw("DELETE FROM config").Exec()
	cfg := models.Config{ID: 1, MarketCondition: 3}
	if _, err := o.Insert(&cfg); err != nil {
		t.Fatal(err)
	}

	// SaveMarketCondition uses the default alias in production. Switch the test
	// model calls through a transaction-compatible helper instead of changing
	// the production alias globally.
	if err := saveMarketConditionWithOrmer(context.Background(), o, cfg.ID, 2); err != nil {
		t.Fatal(err)
	}
	if err := saveMarketConditionWithOrmer(context.Background(), o, cfg.ID, 2); err != nil {
		t.Fatal(err)
	}
	var current models.Config
	if err := o.QueryTable(new(models.Config)).Filter("id", cfg.ID).One(&current); err != nil {
		t.Fatal(err)
	}
	if current.MarketCondition != 2 {
		t.Fatalf("market condition=%d, want 2", current.MarketCondition)
	}
	count, err := o.QueryTable(new(models.MarketConditionHistory)).Filter("config_id", cfg.ID).Count()
	if err != nil || count != 2 {
		t.Fatalf("history count=%d err=%v, want 2", count, err)
	}
}

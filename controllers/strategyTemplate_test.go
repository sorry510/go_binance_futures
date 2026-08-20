package controllers

import (
	"path/filepath"
	"testing"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
)

func TestQueryStrategyTemplatePageOrdersByIDDescending(t *testing.T) {
	const alias = "default"

	if err := orm.RegisterDriver("sqlite3", orm.DRSqlite); err != nil {
		t.Fatal(err)
	}
	if err := orm.RegisterDataBase(alias, "sqlite3", filepath.Join(t.TempDir(), "test.db")); err != nil {
		t.Fatal(err)
	}
	orm.RegisterModel(new(models.StrategyTemplates))
	if err := orm.RunSyncdb(alias, false, false); err != nil {
		t.Fatal(err)
	}

	o := orm.NewOrmUsingDB(alias)
	for _, name := range []string{"first", "second", "third", "fourth", "fifth"} {
		if _, err := o.Insert(&models.StrategyTemplates{Name: name}); err != nil {
			t.Fatal(err)
		}
	}

	list, total, err := queryStrategyTemplatePage(o, "", 2, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total != 5 {
		t.Fatalf("total = %d, want 5", total)
	}
	if len(list) != 2 || list[0].Name != "fifth" || list[1].Name != "fourth" {
		t.Fatalf("first page = %#v, want fifth and fourth", list)
	}

	list, total, err = queryStrategyTemplatePage(o, "ir", 20, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 {
		t.Fatalf("filtered total = %d, want 2", total)
	}
	if len(list) != 2 || list[0].Name != "third" || list[1].Name != "first" {
		t.Fatalf("filtered list = %#v, want third and first", list)
	}
}

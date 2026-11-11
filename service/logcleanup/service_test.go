package logcleanup

import (
	"context"
	"sync"
	"testing"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
)

var cleanupDefaultOnce sync.Once
var cleanupDefaultErr error

func TestCleanupUsesStrictLogWhitelist(t *testing.T) {
	if err := orm.RegisterDriver("sqlite3", orm.DRSqlite); err != nil {
		t.Fatal(err)
	}
	cleanupDefaultOnce.Do(func() {
		cleanupDefaultErr = orm.RegisterDataBase("default", "sqlite3", "file:s510_logcleanup_default?mode=memory&cache=shared")
	})
	if cleanupDefaultErr != nil {
		t.Fatal(cleanupDefaultErr)
	}
	o := orm.NewOrm()
	// No test may inherit rows/tables from an earlier -count=N repetition.
	for _, target := range targets {
		if _, err := o.Raw("DROP TABLE IF EXISTS " + target.Table).Exec(); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := o.Raw("DROP TABLE IF EXISTS agent_trade_audits").Exec(); err != nil {
		t.Fatal(err)
	}
	for _, target := range targets {
		if _, err := o.Raw("CREATE TABLE " + target.Table + " (id INTEGER PRIMARY KEY AUTOINCREMENT, " + target.TimeColumn + " INTEGER NOT NULL)").Exec(); err != nil {
			t.Fatal(err)
		}
		if _, err := o.Raw("INSERT INTO "+target.Table+" ("+target.TimeColumn+") VALUES (?), (?)", 100, 300).Exec(); err != nil {
			t.Fatal(err)
		}
	}
	// A trade audit is intentionally outside the cleanup whitelist.
	if _, err := o.Raw("CREATE TABLE agent_trade_audits (id INTEGER PRIMARY KEY AUTOINCREMENT, created_at INTEGER NOT NULL)").Exec(); err != nil {
		t.Fatal(err)
	}
	if _, err := o.Raw("INSERT INTO agent_trade_audits (created_at) VALUES (100)").Exec(); err != nil {
		t.Fatal(err)
	}
	results, err := Cleanup(context.Background(), 200)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != len(targets) {
		t.Fatalf("unexpected cleanup results: %+v", results)
	}
	for _, target := range targets {
		var count int
		if err := o.Raw("SELECT COUNT(*) FROM " + target.Table).QueryRow(&count); err != nil || count != 1 {
			t.Fatalf("%s retained count=%d err=%v", target.Table, count, err)
		}
	}
	var auditCount int
	if err := o.Raw("SELECT COUNT(*) FROM agent_trade_audits").QueryRow(&auditCount); err != nil || auditCount != 1 {
		t.Fatalf("trade audits must never be cleaned: count=%d err=%v", auditCount, err)
	}
}

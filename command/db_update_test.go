package command

import (
	"os"
	"path/filepath"
	"testing"

	"go_binance_futures/models"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
)

func TestSyncDatabaseInitializesAndIsIdempotent(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	projectRoot := filepath.Dir(cwd)
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)

	if err := orm.RegisterDriver("sqlite3", orm.DRSqlite); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(t.TempDir(), "sync.db")
	if err := orm.RegisterDataBase("default", "sqlite3", dbPath); err != nil {
		t.Fatal(err)
	}
	orm.RegisterModel(
		new(models.Config),
		new(models.StrategyTemplates),
		new(models.Symbols),
		new(models.SpotSymbols),
		new(models.AgentSkill),
	)

	if err := SyncDatabase(1); err != nil {
		t.Fatal(err)
	}
	config, err := utils.GetSystemConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.Version != 1 {
		t.Fatalf("expected database version 1, got %d", config.Version)
	}
	o := orm.NewOrm()
	for _, item := range []*models.AgentSkill{
		{Name: "symbol_analysis", DisplayName: "Symbol", Type: "native", Enabled: 1, ChatEnabled: -1},
		{Name: "alert_analysis", DisplayName: "Alert", Type: "native", Enabled: 1, ChatEnabled: -1},
		{Name: "portable_demo", DisplayName: "Portable", Type: "portable", Enabled: 1, ChatEnabled: -1},
	} {
		if _, err := o.Insert(item); err != nil {
			t.Fatal(err)
		}
	}
	if err := SyncDatabase(2); err != nil {
		t.Fatal(err)
	}
	config, err = utils.GetSystemConfig()
	if err != nil || config.Version != 2 {
		t.Fatalf("expected database version 2 after chat migration, config=%+v err=%v", config, err)
	}
	for name, want := range map[string]int{"symbol_analysis": 1, "alert_analysis": 0, "portable_demo": 1} {
		var got int
		if err := o.Raw("SELECT chat_enabled FROM agent_skills WHERE name = ?", name).QueryRow(&got); err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("chat_enabled for %s = %d, want %d", name, got, want)
		}
	}

	if err := SyncDatabase(2); err != nil {
		t.Fatalf("second version-2 sync should be idempotent: %v", err)
	}
}

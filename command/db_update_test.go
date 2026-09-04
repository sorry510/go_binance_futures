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

	if err := SyncDatabase(1); err != nil {
		t.Fatalf("second sync should be idempotent: %v", err)
	}
}

package controllers

import (
	"path/filepath"
	"reflect"
	"testing"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
)

func TestListenSymbolOptionsFilterPersistedPairs(t *testing.T) {
	rows := []localListenSymbolRecord{
		{Symbol: " ETHUSDT ", Type: "USDT"},
		{Symbol: "BTCUSDT", Type: "USDT"},
		{Symbol: "btcusdt", Type: ""},
		{Symbol: "XRPUSDC", Type: "USDC"},
		{Symbol: "NEWFDUSD", Type: "FDUSD"},
		{Symbol: "BADBUSD", Type: "BUSD"},
		{Symbol: "BTCUSDT_261225", Type: "USDT"},
		{Symbol: "WRONGUSDT", Type: "USDC"},
		{Symbol: "", Type: "USDT"},
		{Symbol: "  ", Type: ""},
	}
	want := []string{"BTCUSDT", "ETHUSDT", "NEWFDUSD", "XRPUSDC"}
	if got := filterLocalListenSymbolOptions(rows); !reflect.DeepEqual(got, want) {
		t.Fatalf("filtered local symbol options = %v, want %v", got, want)
	}
	if got := filterLocalListenSymbolOptions(nil); len(got) != 0 {
		t.Fatalf("empty table must yield an empty list, not a Binance API fallback: %v", got)
	}
}

func TestLocalListenSymbolOptionsReadFromSeparateTables(t *testing.T) {
	const alias = "listen_symbol_options_test"
	if err := orm.RegisterDriver("sqlite3", orm.DRSqlite); err != nil {
		t.Fatal(err)
	}
	// Beego's model bootstrap requires a default alias even though the test
	// queries the isolated named database. Never open the configured real DB.
	if _, err := orm.GetDB("default"); err != nil {
		if err := orm.RegisterDataBase("default", "sqlite3", filepath.Join(t.TempDir(), "default.db")); err != nil {
			t.Fatal(err)
		}
	}
	if err := orm.RegisterDataBase(alias, "sqlite3", filepath.Join(t.TempDir(), "symbols.db")); err != nil {
		t.Fatal(err)
	}
	orm.RegisterModel(new(models.Symbols), new(models.SpotSymbols))
	if err := orm.RunSyncdb(alias, false, false); err != nil {
		t.Fatal(err)
	}
	o := orm.NewOrmUsingDB(alias)
	for _, row := range []models.Symbols{
		{Symbol: "BTCUSDT", Type: "USDT", Enable: 0},
		{Symbol: "ETHUSDT", Type: "USDT", Enable: 1},
		{Symbol: "LTCUSDC", Type: "USDC", Enable: 0},
		{Symbol: "OLDUSD", Type: "", Enable: 1},
	} {
		if _, err := o.Insert(&row); err != nil {
			t.Fatalf("insert local futures symbol %s: %v", row.Symbol, err)
		}
	}
	for _, row := range []models.SpotSymbols{
		{Symbol: "SOLUSDT", Type: "USDT", Enable: 0},
		{Symbol: "DOGEUSDT", Type: "USDT", Enable: 1},
		{Symbol: "SOLFDUSD", Type: "FDUSD", Enable: 0},
		{Symbol: "WRONGUSDT", Type: "USDC", Enable: 1},
	} {
		if _, err := o.Insert(&row); err != nil {
			t.Fatalf("insert local spot symbol %s: %v", row.Symbol, err)
		}
	}
	for _, tc := range []struct {
		name        string
		marketType  int64
		exactSymbol string
		want        []string
	}{
		{"futures", 2, "", []string{"BTCUSDT", "ETHUSDT", "LTCUSDC"}},
		{"spot", 1, "", []string{"DOGEUSDT", "SOLFDUSD", "SOLUSDT"}},
		{"futures exact", 2, "BTCUSDT", []string{"BTCUSDT"}},
		{"spot exact", 1, "SOLUSDT", []string{"SOLUSDT"}},
		{"spot only not in futures", 2, "SOLUSDT", []string{}},
		{"futures only not in spot", 1, "BTCUSDT", []string{}},
		{"invalid spot type", 1, "WRONGUSDT", []string{}},
		{"not persisted", 2, "XRPUSDT", []string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := loadLocalListenSymbolOptions(o, tc.marketType, tc.exactSymbol)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("market=%d symbol=%s: got %v want %v", tc.marketType, tc.exactSymbol, got, tc.want)
			}
		})
	}
	if _, err := loadLocalListenSymbolOptions(o, 3, ""); err == nil {
		t.Fatal("invalid listener market type accepted")
	}
}

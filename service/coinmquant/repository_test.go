package coinmquant

import (
	"encoding/json"
	"fmt"
	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
	"go_binance_futures/models"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

var cm3OrmModels sync.Once
var cm3AliasCounter atomic.Uint64

func fixture(t *testing.T) orm.Ormer {
	t.Helper()
	_ = orm.RegisterDriver("sqlite3", orm.DRSqlite)
	if _, err := orm.GetDB("default"); err != nil {
		if err := orm.RegisterDataBase("default", "sqlite3", "file:coinmquant_cm3_default?mode=memory&cache=shared"); err != nil {
			t.Fatal(err)
		}
	}
	cm3OrmModels.Do(func() { orm.RegisterModel(new(models.DeliverySymbols)) })
	alias := "coinm_cm3_" + strings.ReplaceAll(t.Name(), "/", "_") + "_" + fmt.Sprint(cm3AliasCounter.Add(1))
	if err := orm.RegisterDataBase(alias, "sqlite3", filepath.Join(t.TempDir(), "cm3.db")); err != nil {
		t.Fatal(err)
	}
	if err := orm.RunSyncdb(alias, false, false); err != nil {
		t.Fatal(err)
	}
	o := orm.NewOrmUsingDB(alias)
	for _, sql := range []string{
		"ALTER TABLE delivery_symbols ADD COLUMN budget_usd DECIMAL(36,18) NOT NULL DEFAULT 0",
		"CREATE TABLE coin_m_strategy_templates (id INTEGER PRIMARY KEY AUTOINCREMENT, name VARCHAR(128) NOT NULL, technology TEXT NOT NULL, strategy TEXT NOT NULL, create_time BIGINT NOT NULL, update_time BIGINT NOT NULL)",
	} {
		if _, err := o.Raw(sql).Exec(); err != nil {
			t.Fatal(err)
		}
	}
	return o
}
func TestCM3SymbolIsolationBudgetAndFourDirections(t *testing.T) {
	o := fixture(t)
	old := &models.DeliverySymbols{Symbol: "BTCUSD_PERP", Type: "PERP", ContractType: "PERPETUAL", MarginAsset: "BTC", ContractSize: 100, Enable: 0, Usdt: "10", Leverage: 4, MarginType: "CROSSED", Profit: "8", Loss: "6", Strategy: "[]", Technology: "{}", TickSize: "0.1", StepSize: "1"}
	id, err := o.Insert(old)
	if err != nil {
		t.Fatal(err)
	}
	future := &models.DeliverySymbols{Symbol: "BTCUSD_261225", ContractType: "CURRENT_QUARTER"}
	if _, err := o.Insert(future); err != nil {
		t.Fatal(err)
	}
	page, err := Symbols(o, SymbolFilters{Limit: 50})
	if err != nil || page.Total != 1 || len(page.List) != 1 || page.List[0].BudgetUSD != "0" {
		t.Fatalf("incorrect perpetual-only listing: %+v %v", page, err)
	}
	full, err := Symbols(o, SymbolFilters{Symbol: "BTCUSD_PERP", Limit: 50})
	if err != nil || full.Total != 1 {
		t.Fatalf("full CM symbol search must allow literal _PERP: %+v %v", full, err)
	}
	if _, err := PatchSymbol(o, id, []byte(`{"enable":1}`)); err == nil {
		t.Fatal("enabled unconfigured COIN-M symbol")
	}
	if _, err := PatchSymbol(o, id, []byte(`{"usdt":"250"}`)); err == nil {
		t.Fatal("allowed legacy usdt budget editing")
	}
	if _, err := PatchSymbol(o, id, []byte(`{"contract_size":200}`)); err == nil {
		t.Fatal("allowed exchange-controlled contract size editing")
	}
	if _, err := PatchSymbol(o, id, []byte(`{"budget_usd":"250","leverage":4,"marginType":"ISOLATED"}`)); err != nil {
		t.Fatal(err)
	}
	rules := []Rule{{Name: "entry-long", Type: "long", Code: "Close > Open", Enable: true}, {Name: "exit-long", Type: "close_long", Code: "Close < Open", Enable: true}, {Name: "entry-short", Type: "short", Code: "Close < Open", Enable: true}, {Name: "exit-short", Type: "close_short", Code: "Close > Open", Enable: true}}
	raw, _ := json.Marshal(rules)
	payload, _ := json.Marshal(map[string]any{"strategy": string(raw)})
	if _, err := PatchSymbol(o, id, payload); err != nil {
		t.Fatal(err)
	}
	if _, err := PatchSymbol(o, id, []byte(`{"enable":1}`)); err != nil {
		t.Fatal(err)
	}
	s, err := Symbol(o, id)
	if err != nil || s.Enable != 1 || s.MarginType != "ISOLATED" || s.BudgetUSD != "250" || s.Usdt != "10" || s.ContractSize != 100 {
		t.Fatalf("patch altered legacy fields: %+v err=%v", s, err)
	}
	if _, err := PatchSymbol(o, id, []byte(`{"budget_usd":"1e8"}`)); err == nil {
		t.Fatal("allowed exponent budget")
	}
	if _, err := PatchSymbol(o, id, []byte(`{"budget_usd":"0.1234567890123456789"}`)); err == nil {
		t.Fatal("allowed precision greater than DECIMAL(36,18)")
	}
	for _, v := range []string{`{"enable":2}`, `{"leverage":0}`, `{"marginType":"INVALID"}`, `{"strategy":[{"type":"invalid","code":"1","enable":true}]}`} {
		if _, err := PatchSymbol(o, id, []byte(v)); err == nil {
			t.Errorf("accepted invalid config: %s", v)
		}
	}
	if n, err := EnableAll(o, 0); err != nil || n != 1 {
		t.Fatalf("disable-all failed: %d %v", n, err)
	}
	if n, err := EnableAll(o, 1); err != nil || n != 1 {
		t.Fatalf("enable configured failed: %d %v", n, err)
	}
	if _, err := BatchSymbols(o, []int64{id, id}, []byte(`{"enable":0}`)); err == nil {
		t.Fatal("batch duplicate IDs accepted")
	}
}
func TestCM3IndependentStrategyTemplatesCRUD(t *testing.T) {
	o := fixture(t)
	rules := `[{"name":"short","type":"short","code":"Close < Open","enable":true}]`
	payload, _ := json.Marshal(map[string]any{"name": "CM-only", "technology": "{}", "strategy": rules})
	item, err := CreateTemplate(o, payload)
	if err != nil || item.ID < 1 {
		t.Fatalf("create %+v %v", item, err)
	}
	list, err := Templates(o, "CM", 1, 20)
	if err != nil || list.Total != 1 || len(list.List) != 1 || list.List[0].ID != item.ID || list.List[0].Name != "CM-only" {
		t.Fatalf("incorrect CM-only template listing %+v %v", list, err)
	}
	edited, err := EditTemplate(o, item.ID, []byte(`{"name":"CM-renamed"}`))
	if err != nil || edited.Name != "CM-renamed" {
		t.Fatalf("edit %+v %v", edited, err)
	}
	_, action, err := ImportTemplate(o, []byte(`{"name":"CM-renamed","technology":"{}","strategy":"[]"}`))
	if err != nil || action != "updated" {
		t.Fatalf("same-name import %+v %v", action, err)
	}
	if _, err := CreateTemplate(o, []byte(`{"name":"Invalid","strategy":"[{\"type\":\"spot_sell\"}]"}`)); err == nil {
		t.Fatal("accepted invalid CM strategy direction")
	}
	if err := DeleteTemplate(o, item.ID); err != nil {
		t.Fatal(err)
	}
	list, err = Templates(o, "", 1, 20)
	if err != nil || list.Total != 0 {
		t.Fatalf("templates not deleted %+v %v", list, err)
	}
}

func TestCM3BatchSymbolEditsAreAllOrNothing(t *testing.T) {
	o := fixture(t)
	one := &models.DeliverySymbols{Symbol: "BTCUSD_PERP", ContractType: "PERPETUAL", ContractSize: 100, MarginAsset: "BTC", Leverage: 3, Enable: 0, Strategy: "[]"}
	two := &models.DeliverySymbols{Symbol: "ETHUSD_PERP", ContractType: "PERPETUAL", ContractSize: 10, MarginAsset: "ETH", Leverage: 3, Enable: 0, Strategy: "[]"}
	id1, err := o.Insert(one)
	if err != nil {
		t.Fatal(err)
	}
	id2, err := o.Insert(two)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := PatchSymbol(o, id1, []byte(`{"budget_usd":"100","strategy":[{"name":"L","type":"long","enable":true,"code":"Close > Open"}]}`)); err != nil {
		t.Fatal(err)
	}
	// Second symbol has no budget/entry rule: no partial enable of first.
	if _, err := BatchSymbols(o, []int64{id1, id2}, []byte(`{"enable":1}`)); err == nil {
		t.Fatal("expected incomplete batch to be rejected")
	}
	a, err := Symbol(o, id1)
	if err != nil || a.Enable != 0 {
		t.Fatalf("batch partially enabled first symbol: %+v %v", a, err)
	}
	b, err := Symbol(o, id2)
	if err != nil || b.Enable != 0 {
		t.Fatalf("batch partially enabled second symbol: %+v %v", b, err)
	}
	count, err := BatchSymbols(o, []int64{id1, id2}, []byte(`{"budget_usd":"125.5","leverage":5}`))
	if err != nil || count != 2 {
		t.Fatalf("atomic budget/leverage batch failed %d %v", count, err)
	}
	for _, id := range []int64{id1, id2} {
		row, err := Symbol(o, id)
		if err != nil || row.BudgetUSD != "125.5" || row.Leverage != 5 || row.Enable != 0 {
			t.Fatalf("batch update incorrect id=%d %+v %v", id, row, err)
		}
	}
}

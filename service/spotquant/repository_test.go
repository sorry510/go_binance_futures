package spotquant

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
)

const testLongRule = `[{"name":"buy entry","type":"long","code":"NowPrice > 0","enable":true}]`

var registerSpotQuantTestModel sync.Once

func testStore(t *testing.T) orm.Ormer {
	t.Helper()
	if err := orm.RegisterDriver("sqlite3", orm.DRSqlite); err != nil {
		t.Fatal(err)
	}
	if _, err := orm.GetDB("default"); err != nil {
		if err := orm.RegisterDataBase("default", "sqlite3", filepath.Join(t.TempDir(), "default.db")); err != nil {
			t.Fatal(err)
		}
	}
	alias := "spot_quant_062_" + strings.ReplaceAll(t.Name(), "/", "_")
	if err := orm.RegisterDataBase(alias, "sqlite3", filepath.Join(t.TempDir(), "spot.db")); err != nil {
		t.Fatal(err)
	}
	registerSpotQuantTestModel.Do(func() { orm.RegisterModel(new(models.SpotSymbols)) })
	if err := orm.RunSyncdb(alias, false, false); err != nil {
		t.Fatal(err)
	}
	o := orm.NewOrmUsingDB(alias)
	for _, q := range []string{
		`CREATE TABLE spot_strategy_templates(id INTEGER PRIMARY KEY AUTOINCREMENT,name VARCHAR(128) NOT NULL,technology TEXT NOT NULL,strategy TEXT NOT NULL,create_time BIGINT NOT NULL,update_time BIGINT NOT NULL)`,
		`CREATE TABLE strategy_templates(id INTEGER PRIMARY KEY,name TEXT NOT NULL)`,
		`INSERT INTO strategy_templates VALUES(1,'legacy-futures-must-stay')`,
	} {
		if _, err := o.Raw(q).Exec(); err != nil {
			t.Fatal(err)
		}
	}
	return o
}

func TestStage062SpotRuleAndBudgetValidation(t *testing.T) {
	for _, tc := range []struct {
		input string
		valid bool
	}{
		{testLongRule, true},
		{`[{"name":"sell","type":"close_long","code":"NowPrice > 5","enable":true}]`, true},
		{`[{"type":"short","enable":false}]`, false},
		{`[{"type":"close_short","enable":false}]`, false},
		{`[{"name":"a","type":"long","code":"NowPrice >","enable":true}]`, false},
		{`[{"name":"a","type":"long","code":"NowPrice > 0","enable":false}]`, true},
		{`{"type":"long"}`, false},
	} {
		_, err := ParseRules(tc.input)
		if (err == nil) != tc.valid {
			t.Errorf("parse %s: %v", tc.input, err)
		}
	}
	for _, v := range []string{"10", "0.000000000000000001", "123.123456789012345678"} {
		if err := ValidateBudget(v); err != nil {
			t.Errorf("budget %s: %v", v, err)
		}
	}
	for _, v := range []string{"0", "-1", "1e20", "1.1111111111111111111", "oops"} {
		if err := ValidateBudget(v); err == nil {
			t.Errorf("accepted invalid budget: %s", v)
		}
	}
	if err := ValidateTechnology(`{"ema":[{"name":"ema_10","enable":true}]}`); err != nil {
		t.Fatal(err)
	}
	if err := ValidateTechnology(`{"weird":[]}`); err == nil {
		t.Fatal("unsupported technology accepted")
	}
	if err := ValidateActiveStrategy(`[{"name":"sell","type":"close_long","code":"NowPrice>0","enable":true}]`); err == nil {
		t.Fatal("closing-only rule cannot enable Spot Quant")
	}
}

func TestStage062SpotSymbolsSeparateLegacyAndFutures(t *testing.T) {
	o := testStore(t)
	a, err := CreateSymbol(o, "btcusdt")
	if err != nil {
		t.Fatal(err)
	}
	if a.Symbol != "BTCUSDT" || a.Type != "USDT" || a.Enable != 0 || a.SpotQuantEnabled != 0 {
		t.Fatalf("new Spot state %+v", a)
	}
	if _, err := CreateSymbol(o, "BTCUSDT"); err == nil {
		t.Fatal("duplicate symbol accepted")
	}
	if _, err := CreateSymbol(o, "BTCBAD"); err == nil {
		t.Fatal("invalid quote accepted")
	}
	_, err = o.QueryTable(new(models.SpotSymbols)).Filter("ID", a.ID).Update(orm.Params{"Enable": 1, "Close": "234.5"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ApplySymbolPatch(o, a.ID, []byte(`{"spot_quant_enabled":1}`)); err == nil {
		t.Fatal("unconfigured Spot enabled")
	}
	if _, err := ApplySymbolPatch(o, a.ID, []byte(`{"enable":0}`)); err == nil {
		t.Fatal("legacy enable overwritten")
	}
	if _, err := ApplySymbolPatch(o, a.ID, []byte(`{"technology":{"ema":[{"name":"fast","enable":true}]},"strategy":`+strconvQuote(testLongRule)+`,"usdt":"12.345678"}`)); err != nil {
		t.Fatal(err)
	}
	a, err = ApplySymbolPatch(o, a.ID, []byte(`{"spot_quant_enabled":1,"pin":1}`))
	if err != nil {
		t.Fatal(err)
	}
	if a.Enable != 1 || a.Close != "234.5" || a.SpotQuantEnabled != 1 || a.Pin != 1 {
		t.Fatalf("quant patch corrupted Spot %v", a)
	}
	for _, f := range []SymbolFilters{
		{Symbol: "btc", Quote: "USDT", Enabled: "1"},
		{Quote: "FAVORITE"},
		{Pin: true},
	} {
		got, err := ListSymbols(o, f)
		if err != nil || got.Total != 1 || got.List[0].ID != a.ID {
			t.Fatalf("list %+v: %v", f, err)
		}
	}
	if _, err := ListSymbols(o, SymbolFilters{Quote: "BTC"}); err == nil {
		t.Fatal("unsupported quote accepted")
	}
	if _, err := ListSymbols(o, SymbolFilters{Symbol: "BTC%"}); err == nil {
		t.Fatal("LIKE wildcard accepted")
	}
	if _, err := ApplySymbolPatch(o, a.ID, []byte(`{"strategy":[{"name":"no","type":"short","code":"true","enable":false}]}`)); err == nil {
		t.Fatal("SHORT accepted")
	}
	b, err := CreateSymbol(o, "ethusdc")
	if err != nil {
		t.Fatal(err)
	}
	fd, err := CreateSymbol(o, "solfdusd")
	if err != nil {
		t.Fatal(err)
	}
	if fd.Type != "FDUSD" {
		t.Fatalf("FDUSD quote classification failed: %+v", fd)
	}
	for _, tc := range []struct {
		f    SymbolFilters
		want int64
	}{
		{SymbolFilters{Sort: "symbol+", Limit: 1, Page: 1}, a.ID},
		{SymbolFilters{Sort: "symbol+", Limit: 1, Page: 2}, b.ID},
		{SymbolFilters{Quote: "USDC"}, b.ID},
		{SymbolFilters{Quote: "FDUSD"}, fd.ID},
	} {
		page, err := ListSymbols(o, tc.f)
		if err != nil || len(page.List) != 1 || page.List[0].ID != tc.want {
			t.Fatalf("Spot quote/pagination %+v => %+v %v", tc.f, page, err)
		}
	}
	// Invalid second ID rolls the entire update back.
	if _, err := BatchSymbolPatch(o, []int64{a.ID, 999999}, []byte(`{"usdt":"99"}`)); err == nil {
		t.Fatal("batch invalid id accepted")
	}
	a, err = GetSymbol(o, a.ID)
	if err != nil || a.Usdt != "12.345678" {
		t.Fatalf("batch partial write: %+v err=%v", a, err)
	}
	if _, err := BatchSymbolPatch(o, []int64{a.ID, b.ID}, []byte(`{"profit":"10"}`)); err != nil {
		t.Fatal(err)
	}
	count, err := UpdateSymbolsEnabled(o, 1)
	if err != nil || count != 0 {
		t.Fatalf("invalid symbol activated: count=%d err=%v", count, err)
	}
	count, err = UpdateSymbolsEnabled(o, 0)
	if err != nil || count != 3 {
		t.Fatalf("disable all count=%d err=%v", count, err)
	}
	a, err = GetSymbol(o, a.ID)
	if err != nil || a.SpotQuantEnabled != 0 || a.Enable != 1 {
		t.Fatalf("legacy enable was changed: %+v", a)
	}
}

func TestStage062SpotTemplatesCRUDImportAndIsolation(t *testing.T) {
	o := testStore(t)
	template, err := CreateTemplate(o, []byte(`{"name":"spot-test"}`))
	if err != nil {
		t.Fatal(err)
	}
	if template.ID <= 0 || template.Strategy != "" {
		t.Fatalf("new draft %+v", template)
	}
	updated, err := EditTemplate(o, template.ID, []byte(`{"strategy":`+strconvQuote(testLongRule)+`,"technology":{"ema":[]}}`))
	if err != nil {
		t.Fatal(err)
	}
	if updated.Strategy != testLongRule {
		t.Fatalf("template strategy lost: %+v", updated)
	}
	got, err := GetTemplate(o, template.ID)
	if err != nil || got.Name != "spot-test" || got.ID != template.ID {
		t.Fatalf("get %+v, err=%v", got, err)
	}
	p, err := ListTemplates(o, "spot", 1, 10)
	if err != nil || p.Total != 1 || len(p.List) != 1 || p.List[0].ID != template.ID {
		t.Fatalf("list %+v %v", p, err)
	}
	if _, err := EditTemplate(o, template.ID, []byte(`{"strategy":[{"name":"no","type":"short","code":"true","enable":false}]}`)); err == nil {
		t.Fatal("SHORT template edit accepted")
	}
	importJSON := []byte(`{"name":"spot-test","technology":{"ema":[]},"strategy":` + strconvQuote(testLongRule) + `}`)
	_, action, err := ImportTemplate(o, importJSON)
	if err != nil || action != "updated" {
		t.Fatalf("import existing: %s %v", action, err)
	}
	_, action, err = ImportTemplate(o, []byte(`{"name":"spot-imported","technology":{"ema":[]},"strategy":`+strconvQuote(testLongRule)+`}`))
	if err != nil || action != "created" {
		t.Fatalf("import new: %s %v", action, err)
	}
	// JSON export -> import must also round-trip an intentionally inactive
	// draft; reject SHORT regardless of whether the rule is enabled.
	draft := []byte(`{"name":"draft-export","technology":{"ema":[]},"strategy":[]}`)
	exported, action, err := ImportTemplate(o, draft)
	if err != nil || action != "created" || exported.Name != "draft-export" {
		t.Fatalf("inactive draft JSON import: %+v %s %v", exported, action, err)
	}
	_, action, err = ImportTemplate(o, draft)
	if err != nil || action != "updated" {
		t.Fatalf("inactive draft repeat import: %s %v", action, err)
	}
	if _, _, err = ImportTemplate(o, []byte(`{"name":"bad-short","technology":{},"strategy":[{"name":"bad","type":"short","enable":false}]}`)); err == nil {
		t.Fatal("SHORT imported as a disabled rule")
	}
	if err := DeleteTemplate(o, template.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := GetTemplate(o, template.ID); err == nil {
		t.Fatal("deleted template still available")
	}
	var count int
	if err := o.Raw("SELECT COUNT(*) FROM strategy_templates WHERE name='legacy-futures-must-stay'").QueryRow(&count); err != nil || count != 1 {
		t.Fatalf("Futures templates were modified: %d %v", count, err)
	}
}

func strconvQuote(value string) string {
	buf, _ := json.Marshal(value)
	return string(buf)
}

// Review 06-2-F4: direction alone is not enough. Spot must not silently
// accept Futures-only variables even in disabled rules or imported templates.
func TestStage062SpotRejectFuturesOnlyIdentifiers(t *testing.T) {
	for _, code := range []string{
		"ROI > 5",
		"FundingRate.Data[0] > 0",
		"Position.Leverage > 1",
		"Positions[0].MarginType == \"CROSSED\"",
		"MarketCondition == \"3\"",
		"BasicTrend > 0",
	} {
		for _, enabled := range []bool{false, true} {
			payload, _ := json.Marshal([]Rule{{Name: "test", Type: "long", Code: code, Enable: enabled}})
			if _, err := ParseRules(string(payload)); err == nil || !strings.Contains(err.Error(), "futures-only") {
				t.Errorf("allowed futures-only code=%q enabled=%v: %v", code, enabled, err)
			}
		}
	}
	// AST validation must not block string literals containing field names.
	for _, code := range []string{
		"NowPrice > 1 and \"FundingRate\" == \"FundingRate\"",
		"SomeSpotIndicator.Data[0] > 0",
	} {
		payload, _ := json.Marshal([]Rule{{Name: "spot", Type: "long", Code: code, Enable: true}})
		if _, err := ParseRules(string(payload)); err != nil {
			t.Errorf("valid Spot expression %q rejected: %v", code, err)
		}
	}
	o := testStore(t)
	bad := []byte(`{"name":"bad-funding","technology":{},"strategy":[{"name":"x","type":"long","enable":false,"code":"FundingRate > 0"}]}`)
	if _, _, err := ImportTemplate(o, bad); err == nil {
		t.Fatal("Futures-only rule entered Spot templates through JSON import")
	}
}

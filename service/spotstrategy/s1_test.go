package spotstrategy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
	"go_binance_futures/models"
)

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "spot-s1-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	if err := orm.RegisterDriver("sqlite3", orm.DRSqlite); err != nil {
		panic(err)
	}
	if err := orm.RegisterDataBase("default", "sqlite3", filepath.Join(dir, "spot-s1.db")); err != nil {
		panic(err)
	}
	orm.RegisterModel(new(models.Config), new(models.SpotSymbols))
	if err := orm.RunSyncdb("default", false, false); err != nil {
		panic(err)
	}
	o := orm.NewOrm()
	for _, ddl := range SchemaFor(false) {
		if _, err := o.Raw(ddl).Exec(); err != nil {
			panic(err)
		}
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

func setupSpotS1(t *testing.T) {
	t.Helper()
	o := orm.NewOrm()
	for _, table := range []string{"spot_strategy_orders", "spot_strategy_fills", "spot_strategy_holdings", "spot_symbols", "config"} {
		if _, err := o.Raw("DELETE FROM " + table).Exec(); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := o.Insert(&models.Config{SpotAutoTradeMode: ModeLegacy}); err != nil {
		t.Fatal(err)
	}
	if _, err := o.Insert(&models.SpotSymbols{Symbol: "BTCUSDT", Type: "USDT", Strategy: "[]", Technology: "{}", StrategyBudgetQuote: "0"}); err != nil {
		t.Fatal(err)
	}
}

func TestS1ScopeBudgetsAndExpr(t *testing.T) {
	for _, scope := range [][2]string{{"lead", OwnerAuto}, {AccountMain, "manual"}, {AccountMain, ""}, {AccountMain, OwnerAuto}} {
		err := ValidateScope(scope[0], scope[1])
		if (err == nil) != (scope[0] == AccountMain && scope[1] == OwnerAuto) {
			t.Fatalf("scope=%v err=%v", scope, err)
		}
	}
	for _, input := range []string{"1e4", "-1", "0", "0.0000000000000000001", "NaN", "1.2.3", ""} {
		if _, err := NormalizeBudget(input, true); err == nil {
			t.Fatalf("accepted invalid positive budget %q", input)
		}
	}
	if got, err := NormalizeBudget("00012.500000000000000000", true); err != nil || got != "12.5" {
		t.Fatalf("normalized %q %v", got, err)
	}
	rules := `[{"name":"buy","type":"long","enable":true,"code":"NowPrice > 10"},{"name":"sell","type":"close_long","enable":true,"code":"NowPrice < 7"}]`
	if err := ValidateRules(rules, true); err != nil {
		t.Fatal(err)
	}
	if err := ValidateRules(`[{"type":"short","enable":true,"code":"NowPrice < 10"}]`, true); err == nil {
		t.Fatal("spot short cannot trade")
	}
	if err := ValidateRulesWithTechnology(`[{"type":"long","enable":true,"code":"UnknownVar > 10"},{"type":"close_long","enable":true,"code":"NowPrice < 7"}]`, "{}", true); err == nil {
		t.Fatal("unknown runtime variable must be rejected during configuration")
	}
	if err := ValidateRulesWithTechnology(`[{"type":"long","enable":true,"code":"ma7.Data[0] > 0"},{"type":"close_long","enable":true,"code":"NowPrice < 7"}]`,
		`{"ma":[{"name":"ma7","enable":true,"kline_interval":"1m","period":7}]}`, true); err != nil {
		t.Fatalf("configured indicator variables must compile: %v", err)
	}
	if err := ValidateRules(`[{"type":"long","enable":true,"code":"NowPrice > 10"}]`, true); err == nil {
		t.Fatal("missing close_long should fail")
	}
	matched, err := EvaluateRule("NowPrice > 10", map[string]interface{}{"NowPrice": 12.0})
	if err != nil || !matched {
		t.Fatalf("pure spot expr %v %v", matched, err)
	}
	if _, err := EvaluateRule("NowPrice", map[string]interface{}{"NowPrice": 12.0}); err == nil {
		t.Fatal("non bool should fail")
	}
}

func TestS1ConfigurationModeAndMockReservation(t *testing.T) {
	setupSpotS1(t)
	o := orm.NewOrm()
	if err := AuthorizeLegacyOrder(SourceNotice, "BTCUSDT", IntentExit); err != nil {
		t.Fatal(err)
	}
	if err := RequestMode(ModeStrategy); err == nil {
		t.Fatal("strategy mode must remain blocked at S1")
	}
	if err := RequestMode(ModeOff); err == nil {
		t.Fatal("off transition without verified legacy exits must be blocked")
	}
	if err := RequestMode(ModeLegacy); err != nil {
		t.Fatalf("same-mode legacy request must be idempotent: %v", err)
	}
	if mode, err := CurrentMode(); err != nil || mode != ModeLegacy {
		t.Fatalf("rejected mode transition mutated config: %q %v", mode, err)
	}
	if _, err := o.Raw("UPDATE config SET spot_auto_trade_mode='off'").Exec(); err != nil {
		t.Fatal(err)
	}
	for _, intent := range []OrderIntent{IntentNewRisk, IntentExit} {
		if err := AuthorizeLegacyOrder(SourceNotice, "BTCUSDT", intent); err == nil {
			t.Fatal("unverified legacy order allowed in off mode", intent)
		}
	}
	if err := RequestMode(ModeLegacy); err == nil {
		t.Fatal("off to legacy requires verified account/exit reconciliation")
	}
	if _, err := o.Raw("UPDATE config SET spot_auto_trade_mode='legacy'").Exec(); err != nil {
		t.Fatal(err)
	}
	if _, err := o.Raw("UPDATE config SET spot_auto_trade_mode='strategy'").Exec(); err != nil {
		t.Fatal(err)
	}
	if err := AuthorizeLegacyOrder(SourceNotice, "BTCUSDT", IntentExit); err == nil {
		t.Fatal("legacy order allowed in strategy mode")
	}
	if err := AuthorizeLegacyOrder(OrderSource("typo"), "BTCUSDT", IntentExit); err == nil {
		t.Fatal("unknown legacy source must fail closed")
	}
	if err := AuthorizeLegacyOrder(SourceRush, "BTCUSDT", OrderIntent("typo")); err == nil {
		t.Fatal("unknown legacy intent must fail closed")
	}
	if _, err := o.Raw("UPDATE config SET spot_auto_trade_mode='unknown_mode'").Exec(); err != nil {
		t.Fatal(err)
	}
	if err := AuthorizeLegacyOrder(SourceNotice, "BTCUSDT", IntentExit); err == nil {
		t.Fatal("unknown persisted Spot mode must fail closed")
	}

	if _, err := o.Raw("UPDATE config SET spot_auto_trade_mode='legacy'").Exec(); err != nil {
		t.Fatal(err)
	}

	en := true
	rules := `[{"name":"buy","type":"long","enable":true,"code":"NowPrice > 10"},{"name":"sell","type":"close_long","enable":true,"code":"NowPrice < 7"}]`
	_, err := UpdateSetting(Patch{AccountID: "lead", Owner: OwnerAuto, Symbol: "BTCUSDT", Enabled: &en, BudgetQuote: "10", Strategy: &rules})
	if err == nil {
		t.Fatal("invalid scope accepted")
	}
	_, err = UpdateSetting(Patch{AccountID: AccountMain, Owner: OwnerAuto, Symbol: "BTCUSDT", Enabled: &en, BudgetQuote: "0", Strategy: &rules})
	if err == nil {
		t.Fatal("zero budget accepted")
	}
	setting, err := UpdateSetting(Patch{AccountID: AccountMain, Owner: OwnerAuto, Symbol: "BTCUSDT", Enabled: &en, BudgetQuote: "12.5000", Strategy: &rules})
	if err != nil || !setting.Enabled || setting.BudgetQuote != "12.5" || setting.Execution != "mock_only" {
		t.Fatalf("setting=%+v err=%v", setting, err)
	}
	// Existing legacy JSON API must not change the private enable/budget.
	var existing models.SpotSymbols
	if err := o.QueryTable("spot_symbols").Filter("symbol", "BTCUSDT").One(&existing); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(`{"enable":0,"strategy_trade_enabled":0,"strategy_budget_quote":"0"}`), &existing); err != nil {
		t.Fatal(err)
	}
	if existing.StrategyTradeEnabled != 1 || existing.StrategyBudgetQuote != "12.5" {
		t.Fatalf("legacy edit can change private spot config: %+v", existing)
	}

	req := MockReservation{AccountID: AccountMain, Owner: OwnerAuto, Symbol: "BTCUSDT", ClientOrderID: "s510-s1-unique-one", SignalKey: "closed-kline1-hash1", RequestedQty: "4.250"}
	id, err := ReserveMockOrder(req)
	if err != nil || id < 1 {
		t.Fatalf("reserve mock id=%d err=%v", id, err)
	}
	req.ClientOrderID = "s510-s1-unique-two"
	req.SignalKey = "closed-kline2-hash2"
	if _, err := ReserveMockOrder(req); err == nil {
		t.Fatal("second active mock order not rejected")
	}
	var count int
	if err := o.Raw("SELECT COUNT(*) FROM spot_strategy_orders").QueryRow(&count); err != nil || count != 1 {
		t.Fatalf("duplicate mock count=%d err=%v", count, err)
	}
	if _, err := o.Raw("UPDATE spot_strategy_orders SET status='filled',pending_slot=NULL WHERE id=?", id).Exec(); err != nil {
		t.Fatal(err)
	}
	if _, err := ReserveMockOrder(req); err != nil {
		t.Fatal("new cycle mock allowed after confirmed terminal", err)
	}
	req.ClientOrderID = "s510-s1-unique-one" // duplicate original client ID even after terminal
	req.SignalKey = "closed-kline3-hash3"
	if _, err := ReserveMockOrder(req); err == nil {
		t.Fatal("duplicate client ID should always fail")
	}
	if err := o.Raw("SELECT COUNT(*) FROM spot_strategy_orders").QueryRow(&count); err != nil || count != 2 {
		t.Fatalf("unexpected mock rows %d %v", count, err)
	}
}

// An ambiguous global mode (two config rows) must stop all legacy orders,
// not silently choose an arbitrary first row.
func TestS1MultipleConfigRowsFailClosed(t *testing.T) {
	setupSpotS1(t)
	o := orm.NewOrm()
	if _, err := o.Insert(&models.Config{SpotAutoTradeMode: ModeLegacy}); err != nil {
		t.Fatal(err)
	}
	if _, err := CurrentMode(); err == nil {
		t.Fatal("ambiguous config accepted")
	}
	if err := AuthorizeLegacyOrder(SourceRush, "BTCUSDT", IntentNewRisk); err == nil {
		t.Fatal("legacy trade allowed with multiple config rows")
	}
	if err := RequestMode(ModeOff); err == nil {
		t.Fatal("unsafe mode change allowed with multiple config rows")
	}
}

func TestS1ConcurrentFirstBuyIdempotency(t *testing.T) {
	setupSpotS1(t)
	o := orm.NewOrm()
	if _, err := o.Raw("UPDATE spot_symbols SET strategy_trade_enabled=1 WHERE symbol='BTCUSDT'").Exec(); err != nil {
		t.Fatal(err)
	}
	db, err := orm.GetDB("default")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(8)
	var wg sync.WaitGroup
	outcomes := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			_, err := ReserveMockOrder(MockReservation{
				AccountID: AccountMain, Owner: OwnerAuto, Symbol: "BTCUSDT",
				ClientOrderID: "s510-concurrent-" + string(rune('0'+index)),
				SignalKey:     "signal-" + string(rune('0'+index)), RequestedQty: "1.25"})
			outcomes <- err
		}(i)
	}
	wg.Wait()
	close(outcomes)
	ok, bad := 0, 0
	for err := range outcomes {
		if err == nil {
			ok++
		} else if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "busy") || strings.Contains(strings.ToLower(err.Error()), "unique") {
			bad++
		} else {
			t.Fatalf("unexpected concurrent error: %v", err)
		}
	}
	if ok != 1 || bad != 1 {
		t.Fatalf("simultaneous first BUY expected 1/1, got %d/%d", ok, bad)
	}
	var count int
	if err := o.Raw("SELECT COUNT(*) FROM spot_strategy_orders").QueryRow(&count); err != nil || count != 1 {
		t.Fatalf("concurrent row count=%d err=%v", count, err)
	}
}

package spotstrategy

import (
	"strings"
	"testing"

	"github.com/beego/beego/v2/client/orm"
)

// The table switch must only update enabled/budget; rule/indicator payloads
// are optional. Activation still requires positive budget and both validated
// enabled Spot directions, and emergency disable works with corrupt drafts.
func TestS4SpotEnableMinimalPatchAndSafeDisable(t *testing.T) {
	setupSpotS1(t)
	off, on := false, true
	symbol := "BTCUSDT"
	valid := `[{"name":"entry","type":"long","enable":true,"code":"NowPrice > 1"},{"name":"exit","type":"close_long","enable":true,"code":"NowPrice < 1"}]`
	setting, err := UpdateSetting(Patch{Symbol: symbol, Enabled: &off, BudgetQuote: "125", Strategy: &valid})
	if err != nil || setting.Enabled || setting.BudgetQuote != "125" {
		t.Fatalf("setup failed %+v %v", setting, err)
	}
	setting, err = UpdateSetting(Patch{Symbol: symbol, Enabled: &on, BudgetQuote: "125"})
	if err != nil || !setting.Enabled || setting.BudgetQuote != "125" || setting.Strategy != valid {
		t.Fatalf("minimal enable failed: %+v %v", setting, err)
	}
	setting, err = UpdateSetting(Patch{Symbol: symbol, Enabled: &off, BudgetQuote: "125"})
	if err != nil || setting.Enabled || setting.Strategy != valid {
		t.Fatalf("minimal disable failed: %+v %v", setting, err)
	}

	// Invalid rules must fail before changing the stored enable flag.
	empty := `[]`
	if _, err = UpdateSetting(Patch{Symbol: symbol, Enabled: &off, BudgetQuote: "125", Strategy: &empty}); err != nil {
		t.Fatal(err)
	}
	if _, err = UpdateSetting(Patch{Symbol: symbol, Enabled: &on, BudgetQuote: "125"}); err == nil || !strings.Contains(err.Error(), "long and close_long") {
		t.Fatalf("invalid rule enabled via minimal patch: %v", err)
	}
	setting, err = LoadSetting(symbol)
	if err != nil || setting.Enabled {
		t.Fatalf("rejected enable persisted: %+v %v", setting, err)
	}

	// An unsafe existing strategy must never prevent an emergency disable.
	db := orm.NewOrm()
	if _, err = db.Raw("UPDATE spot_symbols SET strategy=?,strategy_trade_enabled=1 WHERE symbol=?", "not-json", symbol).Exec(); err != nil {
		t.Fatal(err)
	}
	setting, err = UpdateSetting(Patch{Symbol: symbol, Enabled: &off, BudgetQuote: "0"})
	if err != nil || setting.Enabled || setting.Strategy != "not-json" {
		t.Fatalf("emergency disable blocked by malformed legacy rules: %+v %v", setting, err)
	}
}

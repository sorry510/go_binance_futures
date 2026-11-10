package spotstrategy

import (
	"context"
	"testing"

	"github.com/beego/beego/v2/client/orm"
)

// Symbol listing and editing must work when the PRIVATE trade-mode config
// is missing/invalid. Account/Owner remain mandatory for order/Holding APIs.
func TestS4SymbolManagementIndependentOfAccountAndMode(t *testing.T) {
	setupSpotS1(t)
	db, err := orm.GetDB("default")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("UPDATE config SET spot_auto_trade_mode='bad_mode'"); err != nil {
		t.Fatal(err)
	}
	verify := func() {
		t.Helper()
		listing, err := ListSettings(context.Background(), db, PageRequest{QuoteAsset: "USDT"})
		if err != nil || listing.Total != 1 || len(listing.Items) != 1 {
			t.Fatalf("symbol listing depends on trade mode: %+v %v", listing, err)
		}
		setting, err := LoadSetting("BTCUSDT")
		if err != nil || setting.Symbol != "BTCUSDT" || setting.QuoteAsset != "USDT" {
			t.Fatalf("symbol lookup depends on trade mode: %+v %v", setting, err)
		}
		disabled := false
		updated, err := UpdateSetting(Patch{Symbol: "BTCUSDT", Enabled: &disabled, BudgetQuote: "5"})
		if err != nil || updated.BudgetQuote != "5" || updated.Enabled {
			t.Fatalf("symbol config write depends on trade mode: %+v %v", updated, err)
		}
	}
	verify()
	// The config row can be absent entirely; it still must not decide which
	// symbols exist or whether normal symbol metadata can be edited.
	if _, err = db.Exec("DELETE FROM config"); err != nil {
		t.Fatal(err)
	}
	verify()
	// A damaged/missing mode must STILL fail closed for actual order gateways.
	if err := AuthorizeLegacyOrder(SourceNotice, "BTCUSDT", IntentNewRisk); err == nil {
		t.Fatal("symbol management accidentally lifted actual order safety gate")
	}
}

func TestS4SymbolListingDoesNotRequireAnyTradingTables(t *testing.T) {
	setupSpotS1(t)
	db, err := orm.GetDB("default")
	if err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"spot_strategy_fills", "spot_strategy_orders", "spot_strategy_holdings"} {
		if _, err = db.Exec("DROP TABLE " + table); err != nil {
			t.Fatal(err)
		}
	}
	listing, err := ListSettings(context.Background(), db, PageRequest{})
	if err != nil || listing.Total != 1 || len(listing.Items) != 1 {
		t.Fatalf("symbol list queried account/holding/order tables: %+v %v", listing, err)
	}
}

package controllers

import (
	"github.com/beego/beego/v2/client/orm"
	"go_binance_futures/service/leadaccount"
	"net/http"
	"testing"
)

func TestLeadRiskSavesToNewDBTable(t *testing.T) {
	prepareLeadControllerSettingsDB(t)
	t.Setenv(leadaccount.AuditPathEnv, "")
	store, err := leadaccount.LeadConfigStoreFromDB()
	if err != nil {
		t.Fatal(err)
	}
	before, err := store.Load()
	if err != nil || before.AllowNewOpens || before.AccountID != "lead" {
		t.Fatalf("unsafe default %+v %v", before, err)
	}
	const body = `{"account_id":"lead","allow_new_opens":false,"limits":{"max_order_notional_usdt":50,"max_total_notional_usdt":150,"max_daily_realized_loss_usdt":0}}`
	req, rec := stage6Controller(t, http.MethodPut, "/futures/lead/config", body, stage6AuthToken(t))
	req.SaveConfig()
	if rec.Code != http.StatusOK {
		t.Fatalf("risk save: %d %s", rec.Code, rec.Body.String())
	}
	current, err := store.Load()
	if err != nil || current.AllowNewOpens || current.Limits.MaxOrderNotionalUSDT != 50 || current.Limits.MaxTotalNotionalUSDT != 150 {
		t.Fatalf("not persisted: %+v %v", current, err)
	}
	req, rec = stage6Controller(t, http.MethodGet, "/futures/lead/config", "", stage6AuthToken(t))
	req.Config()
	if rec.Code != http.StatusOK {
		t.Fatalf("risk read: %d %s", rec.Code, rec.Body.String())
	}
	if stage6ParseResponse(t, rec)["data"].(map[string]any)["risk_configured"] != true {
		t.Fatal("risk config not reported ready")
	}
	var count int
	if err := orm.NewOrm().Raw("SELECT COUNT(*) FROM lead_trade_settings WHERE account_id='lead'").QueryRow(&count); err != nil || count != 1 {
		t.Fatalf("database row missing: %d %v", count, err)
	}
}

func TestLeadRiskDatabaseUnavailableFailsClosed(t *testing.T) {
	prepareLeadControllerSettingsDB(t)
	if _, err := orm.NewOrm().Raw("DROP TABLE lead_trade_settings").Exec(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = orm.NewOrm().Raw(leadaccount.LeadTradeSettingsDDL).Exec() })
	const body = `{"account_id":"lead","allow_new_opens":false,"limits":{"max_order_notional_usdt":50,"max_total_notional_usdt":150,"max_daily_realized_loss_usdt":0}}`
	req, rec := stage6Controller(t, http.MethodPut, "/futures/lead/config", body, stage6AuthToken(t))
	req.SaveConfig()
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("missing DB schema accepted: %d %s", rec.Code, rec.Body.String())
	}
	req, rec = stage6Controller(t, http.MethodGet, "/futures/lead/config", "", stage6AuthToken(t))
	req.Config()
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("missing DB schema reported healthy: %d", rec.Code)
	}
}

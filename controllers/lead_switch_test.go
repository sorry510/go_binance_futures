package controllers

import (
	"net/http"
	"testing"

	"github.com/beego/beego/v2/client/orm"
	"go_binance_futures/service/leadaccount"
)

func TestLeadSwitchControllerPersistsDefaultOffOnOffWithoutExtraVerify(t *testing.T) {
	if leadaccount.LeadEmergencyPaused() {
		t.Skip("another test asserted the intentional one-way process emergency latch; run this ON/OFF test in isolation")
	}
	prepareLeadControllerSettingsDB(t)
	t.Setenv(leadaccount.AuditPathEnv, "")
	previous := ensureLeadHost
	var starts int
	ensureLeadHost = func() error { starts++; return nil } // never touch Binance in unit tests
	t.Cleanup(func() { ensureLeadHost = previous })
	on := `{"account_id":"lead","allow_new_opens":true,"limits":{"max_order_notional_usdt":50,"max_total_notional_usdt":150,"max_daily_realized_loss_usdt":0}}`
	off := `{"account_id":"lead","allow_new_opens":false,"limits":{"max_order_notional_usdt":50,"max_total_notional_usdt":150,"max_daily_realized_loss_usdt":0}}`
	store, err := leadaccount.LeadConfigStoreFromDB()
	if err != nil {
		t.Fatal(err)
	}
	before, err := store.Load()
	if err != nil || before.AllowNewOpens {
		t.Fatal("Lead must be disabled by default")
	}
	ctrl, rec := stage6Controller(t, http.MethodPut, "/futures/lead/config", on, stage6AuthToken(t))
	ctrl.SaveConfig()
	if rec.Code != http.StatusOK || starts != 1 {
		t.Fatalf("manual ON blocked or skipped existing WS bootstrap: %d starts=%d", rec.Code, starts)
	}
	after, err := store.Load()
	if err != nil || !after.AllowNewOpens {
		t.Fatalf("ON not persisted: %+v %v", after, err)
	}
	ctrl, rec = stage6Controller(t, http.MethodPut, "/futures/lead/config", off, stage6AuthToken(t))
	ctrl.SaveConfig()
	if rec.Code != http.StatusOK || starts != 1 {
		t.Fatalf("OFF should not relaunch WS: %d starts=%d", rec.Code, starts)
	}
	after, err = store.Load()
	if err != nil || after.AllowNewOpens {
		t.Fatalf("OFF not persisted: %+v %v", after, err)
	}
}

func TestLeadSwitchControllerNeverPersistsFailedCredentialStartup(t *testing.T) {
	if leadaccount.LeadEmergencyPaused() {
		t.Skip("another test asserted the intentional one-way process emergency latch; run this startup-failure test in isolation")
	}
	prepareLeadControllerSettingsDB(t)
	previous := ensureLeadHost
	ensureLeadHost = func() error { return leadaccount.ErrCredentialsUnavailable }
	t.Cleanup(func() { ensureLeadHost = previous })
	body := `{"account_id":"lead","allow_new_opens":true,"limits":{"max_order_notional_usdt":50,"max_total_notional_usdt":150,"max_daily_realized_loss_usdt":0}}`
	ctrl, rec := stage6Controller(t, http.MethodPut, "/futures/lead/config", body, stage6AuthToken(t))
	ctrl.SaveConfig()
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("missing Lead credentials were accepted: %d", rec.Code)
	}
	var count int
	if err := orm.NewOrm().Raw("SELECT COUNT(*) FROM lead_trade_settings").QueryRow(&count); err != nil || count != 0 {
		t.Fatalf("failed startup wrote enabled DB settings: count=%d err=%v", count, err)
	}
}

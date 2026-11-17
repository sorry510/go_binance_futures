package controllers

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/beego/beego/v2/client/orm"
	"go_binance_futures/service/leadaccount"
)

func lead01PrivateDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestLead01SaveConfigWithoutAuditFileAndCompactAPI(t *testing.T) {
	prepareLeadControllerSettingsDB(t)
	t.Setenv(leadaccount.AuditPathEnv, "")
	body := `{"account_id":"lead","allow_new_opens":false,"limits":{"max_order_notional_usdt":50,"max_total_notional_usdt":150,"max_daily_realized_loss_usdt":0}}`
	ctrl, rec := stage6Controller(t, http.MethodPut, "/futures/lead/config", body, stage6AuthToken(t))
	ctrl.SaveConfig()
	if rec.Code != http.StatusOK {
		t.Fatalf("normal config write depended on audit: %d %s", rec.Code, rec.Body.String())
	}
	saved, err := leadaccount.LeadConfigStoreFromDB()
	if err != nil {
		t.Fatal(err)
	}
	v, err := saved.Load()
	if err != nil || v.AllowNewOpens || !leadaccount.LeadRiskLimitsReady(v.Limits) {
		t.Fatalf("saved config invalid: %+v %v", v, err)
	}
	ctrl, rec = stage6Controller(t, http.MethodGet, "/futures/lead/config", "", stage6AuthToken(t))
	ctrl.Config()
	if rec.Code != http.StatusOK {
		t.Fatalf("GET config failed: %d", rec.Code)
	}
	data := stage6ParseResponse(t, rec)["data"].(map[string]any)
	if data["risk_configured"] != true {
		t.Fatal("risk readiness missing")
	}
	settings := data["settings"].(map[string]any)
	if _, present := settings["enabled"]; present {
		t.Fatal("obsolete second enabled switch exposed")
	}
	limits := settings["limits"].(map[string]any)
	if len(limits) != 3 {
		t.Fatalf("legacy risk fields still exposed: %+v", limits)
	}
	ctrl, rec = stage6Controller(t, http.MethodPost, "/futures/lead/start", "{}", stage6AuthToken(t))
	ctrl.Start()
	if rec.Code != http.StatusLocked {
		t.Fatalf("audit absence bypassed real trade lock: %d", rec.Code)
	}
}

func TestLead01InvalidRiskFormNeverWrites(t *testing.T) {
	prepareLeadControllerSettingsDB(t)
	t.Setenv(leadaccount.AuditPathEnv, "")
	for _, body := range []string{
		`{"account_id":"lead","allow_new_opens":false,"limits":{}}`,
		`{"account_id":"lead","allow_new_opens":false,"limits":{"max_order_notional_usdt":50,"max_total_notional_usdt":0}}`,
		`{"account_id":"lead","allow_new_opens":true,"limits":{"max_order_notional_usdt":50,"max_total_notional_usdt":150}}`,
		`{"account_id":"main","allow_new_opens":false,"limits":{"max_order_notional_usdt":50,"max_total_notional_usdt":150}}`,
	} {
		ctrl, rec := stage6Controller(t, http.MethodPut, "/futures/lead/config", body, stage6AuthToken(t))
		ctrl.SaveConfig()
		if rec.Code != http.StatusBadRequest && rec.Code != http.StatusLocked && rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("invalid or unconfigured Lead settings accepted: %d %s", rec.Code, rec.Body.String())
		}
		var count int
		if err := orm.NewOrm().Raw("SELECT COUNT(*) FROM lead_trade_settings").QueryRow(&count); err != nil || count != 0 {
			t.Fatalf("invalid config created DB settings: %d %v", count, err)
		}
	}
}

func TestLead01CredentialsCanSaveWithoutExtraAuditFile(t *testing.T) {
	prepareLeadControllerSettingsDB(t)
	dir := lead01PrivateDir(t)
	path := filepath.Join(dir, "credentials.enc")
	t.Setenv(leadaccount.StorePathEnv, path)
	t.Setenv(leadaccount.MasterKeyEnv, base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32)))
	t.Setenv(leadaccount.AuditPathEnv, "")
	body := `{"api_key":"lead_test_key","api_secret":"lead_test_secret","portfolio_label":"read only verification label"}`
	ctrl, rec := stage6Controller(t, http.MethodPut, "/futures/lead/credentials", body, stage6AuthToken(t))
	ctrl.SaveCredentials()
	if rec.Code != http.StatusOK {
		t.Fatalf("credential save blocked by missing audit: %d %s", rec.Code, rec.Body.String())
	}
	store, err := leadaccount.NewEncryptedFileStoreFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	creds, err := store.Load()
	if err != nil || creds.APIKey != "lead_test_key" || creds.APISecret != "lead_test_secret" {
		t.Fatal("saved credential not available through encrypted store")
	}
	if strings.Contains(rec.Body.String(), "lead_test_secret") {
		t.Fatal("secret leaked in API output")
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("unsafe credential file permissions: %v %v", info, err)
	}
}

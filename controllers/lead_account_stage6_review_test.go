package controllers

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/beego/beego/v2/client/orm"
	"go_binance_futures/service/leadaccount"
)

func stage6ReviewPrivateDir(t *testing.T) string {
	t.Helper()
	path := t.TempDir()
	if err := os.Chmod(path, 0700); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestStage6ReviewEmergencyPauseWorksWithoutAuditOrSettings(t *testing.T) {
	// Missing storage must never stop a safety-reducing pause. This test only
	// changes the process-local one-way flag, not Binance or production files.
	t.Setenv(leadaccount.AuditPathEnv, "")
	prepareLeadControllerSettingsDB(t)
	// Missing Lead table simulates a database storage outage. The emergency
	// process-local pause must still apply even if persistence is impossible.
	if _, err := orm.NewOrm().Raw("DROP TABLE lead_trade_settings").Exec(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = orm.NewOrm().Raw(leadaccount.LeadTradeSettingsDDL).Exec() })
	ctrl, response := stage6Controller(t, http.MethodPost, "/futures/lead/pause", "{}", stage6AuthToken(t))
	ctrl.Pause()
	if response.Code != http.StatusAccepted {
		t.Fatalf("emergency pause refused without stores: %d %s", response.Code, response.Body.String())
	}
	result := stage6ParseResponse(t, response)["data"].(map[string]any)
	if result["pause_in_memory"] != true || result["pause_persisted"] != false || result["audit_recorded"] != false ||
		result["allow_new_opens"] != false {
		t.Fatalf("unsafe or misleading pause status: %+v", result)
	}
	if !leadaccount.LeadEmergencyPaused() {
		t.Fatal("immediate in-memory one-way pause was not asserted")
	}
	ctrl, status := stage6Controller(t, http.MethodGet, "/futures/lead/status", "", stage6AuthToken(t))
	ctrl.Status()
	state := stage6ParseResponse(t, status)["data"].(map[string]any)
	if state["pause_in_memory"] != true || state["allow_new_opens"] != false {
		t.Fatalf("status dropped safety latch: %+v", state)
	}
}

func TestStage6ReviewPausePersistsEvenWhenAuditMissing(t *testing.T) {
	prepareLeadControllerSettingsDB(t)
	t.Setenv(leadaccount.AuditPathEnv, "")
	ctrl, response := stage6Controller(t, http.MethodPost, "/futures/lead/pause", "{}", stage6AuthToken(t))
	ctrl.Pause()
	if response.Code != http.StatusOK {
		t.Fatalf("pause should be durable without audit: %d %s", response.Code, response.Body.String())
	}
	result := stage6ParseResponse(t, response)["data"].(map[string]any)
	if result["pause_in_memory"] != true || result["pause_persisted"] != true || result["audit_recorded"] != false {
		t.Fatalf("wrong response when audit unavailable: %+v", result)
	}
	store, err := leadaccount.LeadConfigStoreFromDB()
	if err != nil {
		t.Fatal(err)
	}
	config, err := store.Load()
	if err != nil || config.AllowNewOpens {
		t.Fatalf("durable pause failed %+v %v", config, err)
	}
}

func TestStage6ReviewPauseAuditAndSettingsAvailable(t *testing.T) {
	prepareLeadControllerSettingsDB(t)
	directory := stage6ReviewPrivateDir(t)
	auditPath := filepath.Join(directory, "events.jsonl")
	t.Setenv(leadaccount.AuditPathEnv, auditPath)
	ctrl, response := stage6Controller(t, http.MethodPost, "/futures/lead/pause", "{}", stage6AuthToken(t))
	ctrl.Pause()
	if response.Code != http.StatusOK {
		t.Fatalf("paused request failed: %d %s", response.Code, response.Body.String())
	}
	result := stage6ParseResponse(t, response)["data"].(map[string]any)
	if result["pause_in_memory"] != true || result["pause_persisted"] != true || result["audit_recorded"] != true {
		t.Fatalf("pause missing persistence/audit: %+v", result)
	}
	data, err := os.ReadFile(auditPath)
	if err != nil || !containsStage6ReviewPauseAudit(data) {
		t.Fatalf("pause missing audit: %q %v", data, err)
	}
}
func containsStage6ReviewPauseAudit(data []byte) bool {
	return strings.Contains(string(data), "\"action\":\"paused\"")
}
func TestStage6ReviewMetricQueueMissingCannotReportZeroAsMeasured(t *testing.T) {
	ctrl, response := stage6Controller(t, http.MethodGet, "/futures/lead/metrics", "", stage6AuthToken(t))
	ctrl.Metrics()
	if response.Code != http.StatusOK {
		t.Fatalf("metrics unavailable: %d %s", response.Code, response.Body.String())
	}
	result := stage6ParseResponse(t, response)["data"].(map[string]any)
	metrics := result["metrics"].(map[string]any)
	if result["alert_queue_present"] != false || metrics["alert_queue_present"] != false {
		t.Fatalf("metrics falsely claimed unstarted alert queue: %+v", result)
	}
	if result["runtime_connected"] != false || result["ws_healthy"] != false {
		t.Fatal("offline account mislabeled as connected")
	}
}

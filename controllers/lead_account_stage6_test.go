package controllers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go_binance_futures/service/leadaccount"
	"go_binance_futures/utils"

	beegocontext "github.com/beego/beego/v2/server/web/context"
)

func stage6Controller(t *testing.T, method, path, body, token string) (*LeadController, *httptest.ResponseRecorder) {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Accept", "application/json")
	if method != http.MethodGet && method != http.MethodDelete {
		request.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		request.Header.Set("Authorization", token)
	}
	recorder := httptest.NewRecorder()
	ctx := beegocontext.NewContext()
	ctx.Reset(recorder, request)
	ctx.Input.RequestBody = []byte(body)
	ctrl := new(LeadController)
	ctrl.Init(ctx, "LeadController", "", nil)
	return ctrl, recorder
}
func stage6AuthToken(t *testing.T) string {
	t.Helper()
	v, err := utils.GenerateToken("stage6-owner", 1)
	if err != nil {
		t.Fatal(err)
	}
	return "Bearer " + v
}
func stage6ParseResponse(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var v map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("invalid JSON response: %q (%v)", rec.Body.String(), err)
	}
	return v
}
func TestStage61LeadEndpointRequiresIndependentJWT(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPut} {
		path := "/futures/lead/status"
		if method == http.MethodPost {
			path = "/futures/lead/start"
		}
		if method == http.MethodPut {
			path = "/futures/lead/config"
		}
		ctrl, rec := stage6Controller(t, method, path, "{}", "")
		switch method {
		case http.MethodGet:
			ctrl.Status()
		case http.MethodPost:
			ctrl.Start()
		case http.MethodPut:
			ctrl.SaveConfig()
		}
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("unauthorized %s returned status %d body %q", method, rec.Code, rec.Body.String())
		}
	}
}
func TestStage61StatusShowsNoEnabledLiveTradingAndNoSecret(t *testing.T) {
	ctrl, rec := stage6Controller(t, http.MethodGet, "/futures/lead/status", "", "")
	ctrl.Ctx.Request.Header.Set("Authorization", stage6AuthToken(t))
	ctrl.Status()
	if rec.Code != http.StatusOK {
		t.Fatalf("status API failed: %d %s", rec.Code, rec.Body.String())
	}
	raw := rec.Body.String()
	if strings.Contains(raw, "api_secret") || strings.Contains(raw, "secret_key") {
		t.Fatal("secret included in Lead status")
	}
	data := stage6ParseResponse(t, rec)["data"].(map[string]any)
	if data["account_id"] != "lead" || data["enabled"] != false || data["allow_new_opens"] != false {
		t.Fatalf("Lead status allowed trading: %v", data)
	}
}
func TestStage63StartRejectedEvenWithAuthenticatedJWT(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "lead-audit.jsonl")
	t.Setenv(leadaccount.AuditPathEnv, path)
	ctrl, rec := stage6Controller(t, http.MethodPost, "/futures/lead/start", "{}", stage6AuthToken(t))
	ctrl.Start()
	if rec.Code != http.StatusLocked {
		t.Fatalf("live start unexpectedly accepted: %d %s", rec.Code, rec.Body.String())
	}
	contents, err := os.ReadFile(path)
	if err != nil || !bytes.Contains(contents, []byte("start_denied")) {
		t.Fatalf("rejected start not audited: %q %v", contents, err)
	}
}
func TestStage63ConfigRejectsEnableAndMainAccount(t *testing.T) {
	for _, body := range []string{
		`{"account_id":"lead","enabled":true,"allow_new_opens":false,"limits":{}}`,
		`{"account_id":"main","enabled":false,"allow_new_opens":false,"limits":{}}`,
		`{"account_id":"lead","enabled":false,"allow_new_opens":true,"limits":{}}`,
	} {
		ctrl, rec := stage6Controller(t, http.MethodPut, "/futures/lead/config", body, stage6AuthToken(t))
		ctrl.SaveConfig()
		if rec.Code != http.StatusLocked {
			t.Fatalf("unsafe config accepted: %d %s", rec.Code, rec.Body.String())
		}
	}
}
func TestStage64ManualReconcileRemainsLocked(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv(leadaccount.AuditPathEnv, filepath.Join(dir, "lead-audit.jsonl"))
	ctrl, rec := stage6Controller(t, http.MethodPost, "/futures/lead/reconcile", "{}", stage6AuthToken(t))
	ctrl.Reconcile()
	if rec.Code != http.StatusLocked {
		t.Fatalf("unverified reconciliation allowed: %d %s", rec.Code, rec.Body.String())
	}
}

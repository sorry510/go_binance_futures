package controllers

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"go_binance_futures/middlewares"
	"go_binance_futures/service/leadaccount"

	"github.com/beego/beego/v2/server/web"
)

// LeadController exposes management-only endpoints. It intentionally has no
// code path to execute Binance Submit/Cancel, enable the Lead Broker or start
// the Stage 5 WS supervisor. JWT is checked here as defense in depth in case
// a future router accidentally omits the global JwtMiddleware.
type LeadController struct{ web.Controller }

var leadManagerState struct {
	sync.Mutex
	manager           *leadaccount.Manager
	sourceFingerprint [32]byte
}

func getLeadCredentialManager() (*leadaccount.Manager, *leadaccount.EncryptedFileStore, error) {
	store, err := leadaccount.NewEncryptedFileStoreFromEnv()
	if err != nil {
		return nil, nil, err
	}
	leadManagerState.Lock()
	defer leadManagerState.Unlock()
	// Environment credentials may be rotated by an operator between
	// requests. Never reuse an old Manager bound to a different file/key.
	fingerprint := sha256.Sum256([]byte(os.Getenv(leadaccount.MasterKeyEnv) + "\x00" + os.Getenv(leadaccount.StorePathEnv)))
	if leadManagerState.manager == nil || leadManagerState.sourceFingerprint != fingerprint {
		leadManagerState.manager = leadaccount.NewManager(store)
		leadManagerState.sourceFingerprint = fingerprint
	}
	return leadManagerState.manager, store, nil
}
func (c *LeadController) reply(code int, data any, reason string) {
	if reason == "" {
		reason = "success"
	}
	c.Ctx.Output.SetStatus(code)
	c.Ctx.Resp(map[string]any{"code": code, "data": data, "msg": reason})
}
func (c *LeadController) authorized() bool {
	claims, err := middlewares.ValidateAuthorization(c.Ctx.Request.Header.Get("Authorization"))
	if err != nil || claims["Username"] == nil || strings.TrimSpace(c.Ctx.Request.Header.Get("Authorization")) == "" {
		c.reply(http.StatusUnauthorized, nil, "lead_auth_required")
		return false
	}
	// Do not accept ambient cookie authentication for mutable account controls.
	if c.Ctx.Request.Method != http.MethodGet {
		if !strings.HasPrefix(strings.ToLower(c.Ctx.Request.Header.Get("Content-Type")), "application/json") && c.Ctx.Request.Method != http.MethodDelete {
			c.reply(http.StatusUnsupportedMediaType, nil, "json_content_required")
			return false
		}
	}
	return true
}
func (c *LeadController) body(out any) bool {
	if len(c.Ctx.Input.RequestBody) == 0 || len(c.Ctx.Input.RequestBody) > 4096 {
		c.reply(http.StatusBadRequest, nil, "invalid_lead_payload")
		return false
	}
	dec := json.NewDecoder(strings.NewReader(string(c.Ctx.Input.RequestBody)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		c.reply(http.StatusBadRequest, nil, "invalid_lead_payload")
		return false
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		c.reply(http.StatusBadRequest, nil, "invalid_lead_payload")
		return false
	}
	return true
}
func (c *LeadController) audit(ctx context.Context, action, reason string) bool {
	store, err := leadaccount.LeadAuditStoreFromEnv()
	if err != nil {
		c.reply(http.StatusLocked, nil, "lead_audit_not_configured")
		return false
	}
	if err = store.Append(ctx, action, reason); err != nil {
		c.reply(http.StatusServiceUnavailable, nil, "lead_audit_unavailable")
		return false
	}
	return true
}

// Lead observer is an explicit read-only opt-in; no trade broker starts.
func StartLeadReadOnlyOnExplicitOptIn() error { return leadaccount.StartConfiguredReadOnlyHost() }
func (c *LeadController) Status() {
	if !c.authorized() {
		return
	}
	report := map[string]any{"account_id": "lead", "enabled": false, "allow_new_opens": false,
		"readiness": "disabled", "ws_healthy": false, "last_reconciled_at": nil,
		"blocking_reasons":  []string{"stage7_live_authorization_required", "lead_supervisor_not_started"},
		"runtime_connected": false, "gate5_live_verified": false, "gate6_verified": false,
		"gate7_authorized": false, "source": "offline_management",
		"pause_in_memory": leadaccount.LeadEmergencyPaused(),
		"pause_persisted": false}
	if manager, store, err := getLeadCredentialManager(); err == nil {
		verification := manager.Status()
		if stored, e := store.Load(); e == nil {
			verification.KeyHint = stored.MaskedKey()
			verification.PortfolioLabel = stored.PortfolioLabel
		}
		report["verification"] = verification
	} else {
		report["verification"] = map[string]any{"account_id": "lead", "blocking_reasons": []string{"credentials_unavailable"}}
	}
	observer := leadaccount.LeadReadOnlyStatus()
	report["read_only_observer"] = observer
	if observer.Running {
		report["runtime_connected"] = true
		report["ws_healthy"] = observer.Stream.Healthy
		report["readiness"] = "read_only_observing"
	}
	c.reply(http.StatusOK, report, "success")
}
func (c *LeadController) Config() {
	if !c.authorized() {
		return
	}
	store, err := leadaccount.LeadConfigStoreFromEnv()
	if err != nil {
		c.reply(http.StatusOK, map[string]any{"configured": false, "settings": leadaccount.DefaultLeadSettings(), "blocking_reasons": []string{"lead_config_storage_unavailable"}}, "success")
		return
	}
	v, err := store.Load()
	if err != nil {
		c.reply(http.StatusServiceUnavailable, nil, "lead_config_unavailable")
		return
	}
	c.reply(http.StatusOK, map[string]any{"configured": true, "settings": v}, "success")
}
func (c *LeadController) SaveConfig() {
	if !c.authorized() {
		return
	}
	var settings leadaccount.LeadSettings
	if !c.body(&settings) {
		return
	}
	if err := leadaccount.ValidateLeadSettings(settings); err != nil {
		c.reply(http.StatusLocked, nil, "lead_config_not_authorized_or_invalid")
		return
	}
	store, err := leadaccount.LeadConfigStoreFromEnv()
	if err != nil {
		c.reply(http.StatusLocked, nil, "lead_config_storage_unavailable")
		return
	}
	if !c.audit(c.Ctx.Request.Context(), "settings_updated", "authorized_request") {
		return
	}
	if err = store.Save(settings); err != nil {
		c.reply(http.StatusServiceUnavailable, nil, "lead_config_save_failed")
		return
	}
	c.reply(http.StatusOK, map[string]any{"account_id": "lead", "enabled": false, "allow_new_opens": false}, "success")
}
func (c *LeadController) SaveCredentials() {
	if !c.authorized() {
		return
	}
	var data struct {
		APIKey         string `json:"api_key"`
		APISecret      string `json:"api_secret"`
		PortfolioLabel string `json:"portfolio_label"`
	}
	if !c.body(&data) {
		return
	}
	if data.APIKey == "" || data.APISecret == "" || len(data.APIKey) > 512 || len(data.APISecret) > 512 || len(data.PortfolioLabel) > 128 {
		c.reply(http.StatusBadRequest, nil, "invalid_lead_credentials")
		return
	}
	manager, _, err := getLeadCredentialManager()
	if err != nil {
		c.reply(http.StatusLocked, nil, "lead_encryption_not_configured")
		return
	}
	if !c.audit(c.Ctx.Request.Context(), "credentials_saved", "rotation_or_initial_request") {
		return
	}
	leadaccount.StopReadOnlyHost()
	if err = manager.SaveCredentials(leadaccount.Credentials{APIKey: data.APIKey, APISecret: data.APISecret, PortfolioLabel: data.PortfolioLabel}); err != nil {
		c.reply(http.StatusServiceUnavailable, nil, "lead_credential_save_failed")
		return
	}
	c.reply(http.StatusOK, map[string]any{"account_id": "lead", "key_hint": manager.Status().KeyHint, "readiness": "disabled"}, "success")
}
func (c *LeadController) DeleteCredentials() {
	if !c.authorized() {
		return
	}
	if !c.audit(c.Ctx.Request.Context(), "credentials_deleted", "blocked_pending_authoritative_evidence") {
		return
	}
	// Local DB cannot establish that Binance has zero unmanaged positions,
	// protective Algo orders or in-flight submissions. No unsafe deletion.
	c.reply(http.StatusLocked, nil, "lead_credentials_deletion_requires_full_live_reconciliation")
}
func (c *LeadController) Verify() {
	if !c.authorized() {
		return
	}
	manager, _, err := getLeadCredentialManager()
	if err != nil {
		c.reply(http.StatusLocked, nil, "lead_credentials_unavailable")
		return
	}
	if !c.audit(c.Ctx.Request.Context(), "verify_requested", "authorized_read_only") {
		return
	}
	ctx, cancel := context.WithTimeout(c.Ctx.Request.Context(), 50*time.Second)
	defer cancel()
	report, err := manager.VerifyReadOnly(ctx)
	if err != nil {
		c.reply(http.StatusServiceUnavailable, map[string]any{"account_id": "lead", "blocking_reasons": report.BlockingReasons}, "lead_verification_failed")
		return
	}
	c.reply(http.StatusOK, report, "success")
}

// Pause is a one-way safety operation. Unlike permission-expanding changes,
// audit or configuration I/O failure MUST NOT prevent the immediate in-memory
// halt. Return 202 when durable persistence failed: it is unsafe to claim
// the pause survived a restart. Production Stage 7 still remains locked.
func (c *LeadController) Pause() {
	if !c.authorized() {
		return
	}
	leadaccount.LatchLeadEmergencyPause() // before all external I/O

	persisted := false
	audited := false
	reasons := make([]string, 0, 2)
	store, err := leadaccount.LeadConfigStoreFromEnv()
	if err == nil {
		var settings leadaccount.LeadSettings
		settings, err = store.Load()
		if err == nil {
			settings.Enabled = false
			settings.AllowNewOpens = false
			err = store.Save(settings)
		}
	}
	if err == nil {
		persisted = true
	} else {
		reasons = append(reasons, "lead_pause_persistence_unavailable")
		log.Print("lead_trading: emergency pause asserted in memory; durable persistence unavailable")
	}
	audit, err := leadaccount.LeadAuditStoreFromEnv()
	if err == nil {
		err = audit.Append(c.Ctx.Request.Context(), "paused", "authorized_request")
	}
	if err == nil {
		audited = true
	} else {
		reasons = append(reasons, "lead_audit_unavailable")
		log.Print("lead_trading: emergency pause audit unavailable; safety latch remains active")
	}
	code := http.StatusOK
	if !persisted {
		code = http.StatusAccepted
	}
	c.reply(code, map[string]any{
		"account_id": "lead", "enabled": false, "allow_new_opens": false,
		"pause_in_memory": true, "pause_persisted": persisted,
		"audit_recorded": audited, "blocking_reasons": reasons,
	}, "lead_pause_applied")
}
func (c *LeadController) Start() {
	if !c.authorized() {
		return
	}
	if !c.audit(c.Ctx.Request.Context(), "start_denied", "stage7_live_authorization_required") {
		return
	}
	c.reply(http.StatusLocked, map[string]any{"account_id": "lead", "enabled": false, "allow_new_opens": false, "blocking_reasons": []string{"stage7_live_authorization_required"}}, "stage7_live_authorization_required")
}
func (c *LeadController) Reconcile() {
	if !c.authorized() {
		return
	}
	if !c.audit(c.Ctx.Request.Context(), "reconcile_denied", "gate5_live_evidence_required") {
		return
	}
	c.reply(http.StatusLocked, map[string]any{"account_id": "lead", "blocking_reasons": []string{"gate5_live_evidence_required"}}, "lead_reconciliation_not_authorized")
}
func (c *LeadController) Managed() {
	if !c.authorized() {
		return
	}
	limit, _ := strconv.Atoi(c.GetString("limit", "50"))
	owner := strings.TrimSpace(c.GetString("owner"))
	result, err := leadaccount.ListLeadManaged(c.Ctx.Request.Context(), owner, limit)
	if err != nil {
		c.reply(http.StatusServiceUnavailable, nil, "lead_ownership_data_unavailable")
		return
	}
	if strings.HasSuffix(c.Ctx.Request.URL.Path, "/positions") {
		c.reply(http.StatusOK, map[string]any{"account_id": "lead", "positions": result.Positions, "data_source": "ownership_database", "live_verified": false}, "success")
	} else {
		c.reply(http.StatusOK, map[string]any{"account_id": "lead", "orders": result.Orders, "data_source": "ownership_database", "live_verified": false}, "success")
	}
}
func (c *LeadController) Events() {
	if !c.authorized() {
		return
	}
	audit, err := leadaccount.LeadAuditStoreFromEnv()
	if err != nil {
		c.reply(http.StatusOK, map[string]any{"account_id": "lead", "events": []any{}, "source": "audit_not_configured"}, "success")
		return
	}
	limit, _ := strconv.Atoi(c.GetString("limit", "50"))
	rows, err := audit.List(c.Ctx.Request.Context(), limit)
	if err != nil {
		c.reply(http.StatusServiceUnavailable, nil, "lead_audit_unavailable")
		return
	}
	c.reply(http.StatusOK, map[string]any{"account_id": "lead", "events": rows, "source": "local_audit"}, "success")
}
func (c *LeadController) Metrics() {
	if !c.authorized() {
		return
	}
	observer := leadaccount.LeadReadOnlyStatus()
	metrics := leadaccount.LeadUsageSnapshot(nil, nil, time.Now().UTC())
	if observer.Running {
		metrics.WSUpdates = observer.Mirror.Metrics.WSUpdates
		metrics.RESTSnapshots = observer.Mirror.Metrics.RestSnapshots
	}
	c.reply(http.StatusOK, map[string]any{
		"account_id": "lead", "data_source": "binance_api_usage",
		"runtime_connected": observer.Running, "ws_healthy": observer.Stream.Healthy,
		"read_only_observer":  observer,
		"alert_queue_present": false, "alert_queue_source": "supervisor_not_started",
		"metrics": metrics,
	}, "success")
}
func (c *LeadController) Symbols() {
	if !c.authorized() {
		return
	}
	c.reply(http.StatusOK, map[string]any{"account_id": "lead", "symbols": []any{}, "verified": false,
		"blocking_reasons": []string{"lead_whitelist_not_verified"}}, "success")
}

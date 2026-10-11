package leadaccount

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const AuditPathEnv = "BINANCE_LEAD_AUDIT_FILE"

var ErrLeadAuditUnavailable = errors.New("lead_audit_storage_unavailable")

type LeadAuditEvent struct {
	AccountID  string    `json:"account_id"`
	Action     string    `json:"action"`
	Reason     string    `json:"reason"`
	OccurredAt time.Time `json:"occurred_at"`
}
type LeadAuditStore struct {
	mu   sync.Mutex
	path string
}

func NewLeadAuditStore(path string) (*LeadAuditStore, error) {
	if !filepath.IsAbs(path) || path == "/" || path == "" || verifyCredentialPath(path) != nil {
		return nil, ErrLeadAuditUnavailable
	}
	return &LeadAuditStore{path: path}, nil
}
func LeadAuditStoreFromEnv() (*LeadAuditStore, error) {
	p := os.Getenv(AuditPathEnv)
	if p == "" {
		return nil, ErrLeadAuditUnavailable
	}
	return NewLeadAuditStore(p)
}
func auditAction(v string) bool {
	switch v {
	case "credentials_saved", "credentials_deleted", "verify_requested", "verify_failed",
		"settings_updated", "paused", "start_denied", "reconcile_denied",
		"recovery_inspection":
		return true
	}
	return false
}

// Only fixed server-side reason codes may reach the audit file. This avoids
// persisting credential contents accidentally embedded in raw error text.
func auditReason(v string) bool {
	switch v {
	case "authorized_request", "rotation_or_initial_request", "authorized_read_only",
		"blocked_pending_authoritative_evidence", "stage7_live_authorization_required",
		"gate5_live_evidence_required",
		"read_only_not_authoritative", "lead_outstanding_orders_require_reconciliation",
		"read_only_inspection_failed":
		return true
	}
	return false
}
func (s *LeadAuditStore) Append(ctx context.Context, action, reason string) error {
	if s == nil {
		return ErrLeadAuditUnavailable
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if !auditAction(action) || !auditReason(reason) {
		return ErrLeadAuditUnavailable
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if verifyCredentialPath(s.path) != nil {
		return ErrLeadAuditUnavailable
	}
	data, err := json.Marshal(LeadAuditEvent{AccountID: "lead", Action: action, Reason: reason, OccurredAt: time.Now().UTC()})
	if err != nil {
		return ErrLeadAuditUnavailable
	}
	f, err := os.OpenFile(s.path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
	if err != nil {
		return ErrLeadAuditUnavailable
	}
	_, err = f.Write(append(data, '\n'))
	if err == nil {
		err = f.Sync()
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return ErrLeadAuditUnavailable
	}
	return nil
}
func (s *LeadAuditStore) List(ctx context.Context, limit int) ([]LeadAuditEvent, error) {
	if s == nil {
		return nil, ErrLeadAuditUnavailable
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if verifyCredentialPath(s.path) != nil {
		return nil, ErrLeadAuditUnavailable
	}
	// Cap memory so untrusted file growth cannot starve the web process.
	info, err := os.Stat(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return []LeadAuditEvent{}, nil
	}
	if err != nil || info.Size() > 8<<20 {
		return nil, ErrLeadAuditUnavailable
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, ErrLeadAuditUnavailable
	}
	rows := strings.Split(strings.TrimSpace(string(data)), "\n")
	out := make([]LeadAuditEvent, 0, limit)
	for i := len(rows) - 1; i >= 0 && len(out) < limit; i-- {
		if rows[i] == "" {
			continue
		}
		var e LeadAuditEvent
		if json.Unmarshal([]byte(rows[i]), &e) != nil || e.AccountID != "lead" || !auditAction(e.Action) {
			return nil, ErrLeadAuditUnavailable
		}
		out = append(out, e)
	}
	return out, nil
}

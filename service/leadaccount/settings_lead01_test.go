package leadaccount

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLead01NewConfigOnlySecurePersistence(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "risk.json")
	store, err := NewLeadConfigStore(path)
	if err != nil {
		t.Fatal(err)
	}
	s := DefaultLeadSettings()
	s.Limits = LeadRiskConfig{MaxOrderNotionalUSDT: 50, MaxTotalNotionalUSDT: 100, MaxDailyRealizedLossUSDT: 8}
	if err := store.Save(s); err != nil {
		t.Fatal(err)
	}
	reloaded, err := store.Load()
	if err != nil || reloaded.AllowNewOpens || !LeadRiskLimitsReady(reloaded.Limits) {
		t.Fatalf("saved Lead settings: %+v %v", reloaded, err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{`"enabled"`, `"max_drawdown_pct"`, `"max_positions"`, `"max_losing_positions"`} {
		if strings.Contains(string(content), key) {
			t.Fatalf("old field was stored: %s", key)
		}
	}
	for _, key := range []string{`"allow_new_opens"`, `"max_order_notional_usdt"`, `"max_total_notional_usdt"`, `"max_daily_realized_loss_usdt"`} {
		if !strings.Contains(string(content), key) {
			t.Fatalf("new field missing: %s", key)
		}
	}
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("unsafe config mode: %v %v", info, err)
	}
	t.Setenv(ConfigPathEnv, path)
	if err := LeadDurableOpenGate(); !errors.Is(err, ErrLeadLiveGateLocked) {
		t.Fatalf("config unexpectedly unlocked live trading: %v", err)
	}
}

func TestLead01OldConfigFileIsRejectedWithoutMigration(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "risk.json")
	old := `{"account_id":"lead","enabled":false,"allow_new_opens":false,"limits":{"max_order_notional_usdt":50,"max_total_notional_usdt":100,"max_daily_realized_loss_usdt":8,"max_drawdown_pct":12,"max_positions":3,"max_losing_positions":2}}`
	if err := os.WriteFile(path, []byte(old), 0600); err != nil {
		t.Fatal(err)
	}
	store, err := NewLeadConfigStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(); !errors.Is(err, ErrLeadConfigUnavailable) {
		t.Fatalf("old format unexpectedly compatible: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != old {
		t.Fatal("old file was unexpectedly migrated or changed")
	}
	if err := store.Save(LeadSettings{AccountID: "lead", Limits: LeadRiskConfig{MaxOrderNotionalUSDT: 50, MaxTotalNotionalUSDT: 100}}); err != nil {
		t.Fatalf("fresh config cannot be explicitly written: %v", err)
	}
	if _, err := store.Load(); err != nil {
		t.Fatalf("new config failed after explicit save: %v", err)
	}
}

func TestLead01RejectUnknownOrRetiredConfigurationFields(t *testing.T) {
	for _, raw := range []string{
		`{"account_id":"lead","enabled":false,"allow_new_opens":false,"limits":{}}`,
		`{"account_id":"lead","allow_new_opens":false,"limits":{"max_drawdown_pct":5}}`,
		`{"account_id":"lead","allow_new_opens":false,"limits":{"max_positions":3}}`,
		`{"account_id":"lead","allow_new_opens":false,"limits":{"max_losing_positions":3}}`,
		`{"account_id":"lead","allow_new_opens":false,"risk_approval":true,"limits":{}}`,
		`{"account_id":"lead","allow_new_opens":false,"limits":{"max_order_notional_usdt":50,"max_total_notional_usdt":100,"unknown_extra":1}}`,
		`{"account_id":"lead","allow_new_opens":true,"limits":{}}`,
	} {
		dir := t.TempDir()
		if err := os.Chmod(dir, 0700); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, "risk.json")
		if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		store, err := NewLeadConfigStore(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := store.Load(); !errors.Is(err, ErrLeadConfigUnavailable) {
			t.Fatalf("unknown/unsafe config was accepted: %s err=%v", raw, err)
		}
	}
	// A Go struct decode alone does not validate business semantics.
	var v LeadSettings
	if err := json.Unmarshal([]byte(`{"account_id":"lead","allow_new_opens":true,"limits":{}}`), &v); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(ValidateLeadSettings(v), ErrLeadLiveGateLocked) {
		t.Fatal("enabled new entries were accepted")
	}
}

func TestLead01RiskCapPairAndOptionalDailyLoss(t *testing.T) {
	for _, v := range []struct {
		order, total, daily float64
		valid, ready        bool
	}{
		{0, 0, 0, true, false},
		{50, 100, 0, true, true},
		{50, 100, 8, true, true},
		{50, 0, 0, false, false},
		{0, 100, 0, false, false},
		{101, 100, 0, false, false},
		{50, 100, -1, false, true},
	} {
		s := DefaultLeadSettings()
		s.Limits = LeadRiskConfig{MaxOrderNotionalUSDT: v.order, MaxTotalNotionalUSDT: v.total, MaxDailyRealizedLossUSDT: v.daily}
		if (ValidateLeadSettings(s) == nil) != v.valid {
			t.Fatalf("unexpected validity: %+v", v)
		}
		if got := LeadRiskLimitsReady(s.Limits); got != v.ready {
			t.Fatalf("unexpected readiness: %+v got=%v", v, got)
		}
	}
}

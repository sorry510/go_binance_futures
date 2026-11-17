package leadaccount

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	binance "go_binance_futures/feature/api/binance"
	"go_binance_futures/service/futuresownership"
)

func TestStage63SettingsDatabasePersistenceAlwaysDisabled(t *testing.T) {
	prepareLeadSettingsTestDB(t)
	store, err := LeadConfigStoreFromDB()
	if err != nil {
		t.Fatal(err)
	}
	initial, err := store.Load()
	if err != nil || initial.AccountID != "lead" || initial.AllowNewOpens {
		t.Fatalf("unsafe default %+v %v", initial, err)
	}
	settings := DefaultLeadSettings()
	settings.Limits = LeadRiskConfig{MaxOrderNotionalUSDT: 50, MaxTotalNotionalUSDT: 100, MaxDailyRealizedLossUSDT: 8}
	if err := store.Save(settings); err != nil {
		t.Fatal(err)
	}
	recovered, err := store.Load()
	if err != nil || recovered.Limits.MaxOrderNotionalUSDT != 50 || recovered.AllowNewOpens {
		t.Fatalf("DB state %+v %v", recovered, err)
	}
	for _, tc := range []struct {
		name   string
		mutate func(*LeadSettings)
	}{
		{"wrong_account", func(v *LeadSettings) { v.AccountID = "main" }},
		{"negative_loss", func(v *LeadSettings) { v.Limits.MaxDailyRealizedLossUSDT = -1 }},
		{"total_smaller", func(v *LeadSettings) { v.Limits.MaxTotalNotionalUSDT = 1 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := recovered
			tc.mutate(&bad)
			if err := store.Save(bad); err == nil {
				t.Fatal("unsafe config persisted")
			}
			after, e := store.Load()
			if e != nil || after.AllowNewOpens || after.Limits.MaxOrderNotionalUSDT != 50 {
				t.Fatalf("rejected config mutated DB %+v %v", after, e)
			}
		})
	}
}
func TestStage64LeadLocalAuditNeverSerializesSecret(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	store, err := NewLeadAuditStore(filepath.Join(dir, "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Append(context.Background(), "paused", "authorized_request"); err != nil {
		t.Fatal(err)
	}
	if err := store.Append(context.Background(), "start_denied", "stage7_live_authorization_required"); err != nil {
		t.Fatal(err)
	}
	if err := store.Append(context.Background(), "paused", "APIKey=supersecret"); err == nil {
		t.Fatal("untrusted raw input accepted into audit")
	}
	events, err := store.List(context.Background(), 20)
	if err != nil || len(events) != 2 || events[0].Action != "start_denied" || events[1].Action != "paused" {
		t.Fatalf("wrong audit sequence %+v %v", events, err)
	}
	contents, err := os.ReadFile(store.path)
	if err != nil || strings.Contains(string(contents), "supersecret") {
		t.Fatal("audit stored unsafe input")
	}
}
func TestStage64OwnershipAlwaysBindsLeadOnly(t *testing.T) {
	s, err := futuresownership.BindAccount(binance.LeadAccountID)
	if err != nil || s.AccountID != "lead" {
		t.Fatalf("lead ownership not isolated %+v %v", s, err)
	}
}

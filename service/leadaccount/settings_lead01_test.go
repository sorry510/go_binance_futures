package leadaccount

import (
	"errors"
	"github.com/beego/beego/v2/client/orm"
	"testing"
)

func TestLead01DatabaseSettingsDefaultOffAndPersistence(t *testing.T) {
	prepareLeadSettingsTestDB(t)
	store, err := LeadConfigStoreFromDB()
	if err != nil {
		t.Fatal(err)
	}
	initial, err := store.Load()
	if err != nil || initial.AllowNewOpens || initial.AccountID != "lead" || LeadRiskLimitsReady(initial.Limits) {
		t.Fatalf("unsafe DB defaults: %+v %v", initial, err)
	}
	settings := DefaultLeadSettings()
	settings.Limits = LeadRiskConfig{MaxOrderNotionalUSDT: 50, MaxTotalNotionalUSDT: 100, MaxDailyRealizedLossUSDT: 8}
	if err := store.Save(settings); err != nil {
		t.Fatal(err)
	}
	newStore, err := LeadConfigStoreFromDB()
	if err != nil {
		t.Fatal(err)
	}
	got, err := newStore.Load()
	if err != nil || got.AllowNewOpens || !LeadRiskLimitsReady(got.Limits) || got.Limits.MaxDailyRealizedLossUSDT != 8 {
		t.Fatalf("DB persistence: %+v %v", got, err)
	}
	if err := LeadDurableOpenGate(); !errors.Is(err, ErrLeadLiveGateLocked) {
		t.Fatalf("default off authorized trading: %v", err)
	}
	var count int
	if err := orm.NewOrm().Raw("SELECT COUNT(*) FROM lead_trade_settings WHERE account_id='lead'").QueryRow(&count); err != nil || count != 1 {
		t.Fatalf("single lead row: %d %v", count, err)
	}
}

func TestLead01RejectInvalidRiskAndMainScopeWithoutDBMutation(t *testing.T) {
	prepareLeadSettingsTestDB(t)
	store, _ := LeadConfigStoreFromDB()
	settings := DefaultLeadSettings()
	settings.Limits = LeadRiskConfig{MaxOrderNotionalUSDT: 50, MaxTotalNotionalUSDT: 100}
	if err := store.Save(settings); err != nil {
		t.Fatal(err)
	}
	for _, wrong := range []LeadSettings{
		{AccountID: "main", Limits: settings.Limits},
		{AccountID: "lead", AllowNewOpens: true},
		{AccountID: "lead", Limits: LeadRiskConfig{MaxOrderNotionalUSDT: 100, MaxTotalNotionalUSDT: 10}},
		{AccountID: "lead", Limits: LeadRiskConfig{MaxOrderNotionalUSDT: 50, MaxTotalNotionalUSDT: 100, MaxDailyRealizedLossUSDT: -1}},
	} {
		if err := store.Save(wrong); err == nil {
			t.Fatalf("unsafe DB config accepted: %+v", wrong)
		}
	}
	got, err := store.Load()
	if err != nil || got.AllowNewOpens || got.Limits.MaxOrderNotionalUSDT != 50 {
		t.Fatalf("invalid settings changed DB %+v %v", got, err)
	}
}

func TestLead01MissingDatabaseTableFailsClosed(t *testing.T) {
	prepareLeadSettingsTestDB(t)
	if _, err := orm.NewOrm().Raw("DROP TABLE lead_trade_settings").Exec(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = orm.NewOrm().Raw(LeadTradeSettingsDDL).Exec() })
	store, _ := LeadConfigStoreFromDB()
	if _, err := store.Load(); err == nil {
		t.Fatal("missing DB table returned default as authoritative")
	}
	config := DefaultLeadSettings()
	config.AllowNewOpens = true
	config.Limits = LeadRiskConfig{MaxOrderNotionalUSDT: 50, MaxTotalNotionalUSDT: 100}
	if err := store.Save(config); err == nil {
		t.Fatal("write without schema succeeded")
	}
	if err := LeadDurableOpenGate(); !errors.Is(err, ErrLeadConfigUnavailable) {
		t.Fatalf("DB failure did not fail closed: %v", err)
	}
}

func TestLead01RiskCapPairAndOptionalDailyLoss(t *testing.T) {
	for _, v := range []struct {
		order, total, daily float64
		valid, ready        bool
	}{
		{0, 0, 0, true, false}, {50, 100, 0, true, true}, {50, 100, 8, true, true},
		{50, 0, 0, false, false}, {0, 100, 0, false, false}, {101, 100, 0, false, false},
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

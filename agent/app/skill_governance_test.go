package app

import (
	"context"
	"strings"
	"testing"

	"go_binance_futures/agent/governance"
	"go_binance_futures/agent/skillconfig"
	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
)

func setupSkillGovernanceTest(t *testing.T) {
	t.Helper()
	if err := orm.RegisterDriver("sqlite3", orm.DRSqlite); err != nil {
		t.Fatal(err)
	}
	orm.RegisterModel(new(models.AgentSkill))
	if err := orm.RegisterDataBase("default", "sqlite3", "file:agent_app_skill?mode=memory&cache=shared"); err != nil {
		t.Fatal(err)
	}
	if err := orm.RunSyncdb("default", true, false); err != nil {
		t.Fatal(err)
	}
}
func TestSkillDatabaseControlsAdmissionAndCanBeRecreated(t *testing.T) {
	setupSkillGovernanceTest(t)
	originalStore, originalLimiter := defaultSkillStore, defaultLimiter
	defer func() { defaultSkillStore, defaultLimiter = originalStore, originalLimiter }()
	defaultSkillStore = skillconfig.Store{}
	defaultLimiter = governance.New(func() governance.Limits {
		return governance.Limits{PerMinute: 100, PerHour: 1000}
	})
	ctx := context.Background()
	store := skillconfig.Store{}
	item, err := store.Create(ctx, skillconfig.CreateInput{Name: "symbol_analysis", Enabled: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := AdmitSkill("symbol_analysis"); err != nil {
		t.Fatal(err)
	}
	_, err = store.Update(ctx, item.ID, skillconfig.UpdateInput{Enabled: 0})
	if err != nil {
		t.Fatal(err)
	}
	if err := AdmitSkill("symbol_analysis"); err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("disabled skill admission error = %v", err)
	}
	if _, err := store.Update(ctx, item.ID, skillconfig.UpdateInput{Enabled: 1}); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(ctx, item.ID); err != nil {
		t.Fatal(err)
	}
	if err := AdmitSkill("symbol_analysis"); err == nil || !strings.Contains(err.Error(), "not registered") {
		t.Fatalf("deleted skill admission error = %v", err)
	}
	if _, err := store.Create(ctx, skillconfig.CreateInput{Name: "symbol_analysis", Enabled: 1}); err != nil {
		t.Fatal(err)
	}
	if err := AdmitSkill("symbol_analysis"); err != nil {
		t.Fatalf("recreated skill admission error = %v", err)
	}
}

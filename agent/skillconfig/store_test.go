package skillconfig

import (
	"context"
	"sync"
	"testing"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
)

var skillStoreTestOnce sync.Once
var skillStoreTestErr error

func setupSkillStoreTest(t *testing.T) Store {
	t.Helper()
	skillStoreTestOnce.Do(func() {
		skillStoreTestErr = orm.RegisterDriver("sqlite3", orm.DRSqlite)
		if skillStoreTestErr != nil {
			return
		}
		orm.RegisterModel(new(models.AgentSkill))
		skillStoreTestErr = orm.RegisterDataBase("default", "sqlite3", "file:agent_skill_test?mode=memory&cache=shared")
		if skillStoreTestErr == nil {
			skillStoreTestErr = orm.RunSyncdb("default", true, false)
		}
	})
	if skillStoreTestErr != nil {
		t.Fatal(skillStoreTestErr)
	}
	return Store{}
}

func TestStoreCRUDAndDefaultsDoNotRestoreDeletedRows(t *testing.T) {
	store := setupSkillStoreTest(t)
	ctx := context.Background()
	defaults := []CreateInput{
		{Name: "one", DisplayName: "One", Enabled: 1},
		{Name: "two", DisplayName: "Two", Enabled: 1},
	}
	if err := store.EnsureDefaults(ctx, defaults); err != nil {
		t.Fatal(err)
	}
	items, err := store.List(ctx)
	if err != nil || len(items) != 2 {
		t.Fatalf("initial skills=%d err=%v", len(items), err)
	}
	if err := store.Delete(ctx, items[0].ID); err != nil {
		t.Fatal(err)
	}
	if err := store.EnsureDefaults(ctx, defaults); err != nil {
		t.Fatal(err)
	}
	items, err = store.List(ctx)
	if err != nil || len(items) != 1 {
		t.Fatalf("skills after delete=%d err=%v", len(items), err)
	}
	updated, err := store.Update(ctx, items[0].ID, UpdateInput{
		DisplayName: "Updated", Enabled: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Enabled != 0 || updated.DisplayName != "Updated" {
		t.Fatalf("unexpected update: %+v", updated)
	}
}

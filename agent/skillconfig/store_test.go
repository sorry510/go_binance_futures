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
		{Name: "one", DisplayName: "One", Description: "alpha skill", Enabled: 1, ChatEnabled: 1},
		{Name: "two", DisplayName: "Two", Description: "beta skill", Enabled: 1, ChatEnabled: 0},
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
		DisplayName: "Updated", Enabled: 0, ChatEnabled: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Enabled != 0 || updated.ChatEnabled != 1 || updated.DisplayName != "Updated" {
		t.Fatalf("unexpected update: %+v", updated)
	}
}

func TestStoreListPageFiltersTypeKeywordAndPaginates(t *testing.T) {
	store := setupSkillStoreTest(t)
	ctx := context.Background()
	for _, input := range []CreateInput{
		{Name: "page_native_alpha", DisplayName: "Alpha Native", Description: "momentum finder", Enabled: 1, ChatEnabled: 1},
		{Name: "page_native_beta", DisplayName: "Beta Native", Description: "mean reversion", Enabled: 1, ChatEnabled: 1},
	} {
		if _, err := store.Create(ctx, input); err != nil {
			t.Fatal(err)
		}
	}
	result, err := store.ListPage(ctx, ListOptions{Type: "native", Keyword: "momentum", Page: 1, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 1 || len(result.List) != 1 || result.List[0].Name != "page_native_alpha" {
		t.Fatalf("unexpected filtered page: %+v", result)
	}
}

func TestEnsureDefaultsInitializesOnlyUnsetChatEnabled(t *testing.T) {
	store := setupSkillStoreTest(t)
	ctx := context.Background()
	item, err := store.Create(ctx, CreateInput{Name: "chat_default_once", DisplayName: "Chat", Enabled: 1, ChatEnabled: 0})
	if err != nil {
		t.Fatal(err)
	}
	o := store.ormer()
	item.ChatEnabled = -1
	if _, err := o.Update(item, "ChatEnabled"); err != nil {
		t.Fatal(err)
	}
	defaults := []CreateInput{{Name: item.Name, DisplayName: item.DisplayName, Enabled: 1, ChatEnabled: 1}}
	if err := store.EnsureDefaults(ctx, defaults); err != nil {
		t.Fatal(err)
	}
	got, err := store.GetByName(ctx, item.Name)
	if err != nil || got.ChatEnabled != 1 {
		t.Fatalf("unset chat default was not initialized: got=%+v err=%v", got, err)
	}
	got.ChatEnabled = 0
	if _, err := o.Update(got, "ChatEnabled"); err != nil {
		t.Fatal(err)
	}
	if err := store.EnsureDefaults(ctx, defaults); err != nil {
		t.Fatal(err)
	}
	got, _ = store.GetByName(ctx, item.Name)
	if got.ChatEnabled != 0 {
		t.Fatalf("explicit chat setting must not be overwritten: %+v", got)
	}
}

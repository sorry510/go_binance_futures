package task

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
)

var ormStoreTestOnce sync.Once
var ormStoreTestErr error

func setupORMStoreTest(t *testing.T) *ORMStore {
	t.Helper()
	ormStoreTestOnce.Do(func() {
		ormStoreTestErr = orm.RegisterDriver("sqlite3", orm.DRSqlite)
		if ormStoreTestErr != nil {
			return
		}
		orm.RegisterModel(new(models.AgentTask), new(models.AgentTaskEvent))
		ormStoreTestErr = orm.RegisterDataBase("agent_task_test", "sqlite3", "file:agent_task_test?mode=memory&cache=shared")
		if ormStoreTestErr != nil {
			return
		}
		ormStoreTestErr = orm.RunSyncdb("agent_task_test", true, false)
	})
	if ormStoreTestErr != nil {
		t.Fatal(ormStoreTestErr)
	}
	return &ORMStore{Alias: "agent_task_test"}
}

func TestORMStorePersistsTaskEventsAndRedactsSecrets(t *testing.T) {
	store := setupORMStoreTest(t)
	now := time.Now().UTC()
	item := &Task{
		ID: "task-persist", Skill: "symbol_analysis", ConversationID: "conv-1", Status: StatusRunning,
		Stage: "waiting_tool", Progress: 45, Input: `{"symbol":"BTCUSDT","api_key":"secret-value"}`,
		Error: "authorization=top-secret", CreatedAt: now, UpdatedAt: now,
		Events: []Event{{TaskID: "task-persist", Stage: "queued", Message: "Bearer super-secret", Time: now}},
	}
	if err := store.Save(context.Background(), item); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(context.Background(), item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ConversationID != "conv-1" || len(got.Events) != 1 {
		t.Fatalf("unexpected persisted task: %+v", got)
	}
	if strings.Contains(got.Input, "secret-value") || strings.Contains(got.Error, "top-secret") || strings.Contains(got.Events[0].Message, "super-secret") {
		t.Fatalf("sensitive value persisted: %+v", got)
	}
}

func TestORMStoreMarksRunningTasksInterrupted(t *testing.T) {
	store := setupORMStoreTest(t)
	now := time.Now().UTC()
	item := &Task{ID: "task-interrupted", Skill: "alert_analysis", Status: StatusWaitingLLM, Stage: "waiting_llm", Progress: 40, CreatedAt: now, UpdatedAt: now}
	if err := store.Save(context.Background(), item); err != nil {
		t.Fatal(err)
	}
	marked, err := store.MarkInterrupted(context.Background(), now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if marked < 1 {
		t.Fatalf("expected interrupted task, got %d", marked)
	}
	got, err := store.Get(context.Background(), item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != StatusInterrupted || got.CompletedAt == nil || len(got.Events) == 0 || got.Events[len(got.Events)-1].Stage != "interrupted" {
		t.Fatalf("unexpected interrupted task: %+v", got)
	}
}

func TestORMStoreListsPersistedTasks(t *testing.T) {
	store := setupORMStoreTest(t)
	result, err := store.List(context.Background(), ListOptions{Skill: "symbol_analysis", Page: 1, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total < 1 || len(result.List) < 1 {
		t.Fatalf("expected persisted tasks, got %+v", result)
	}
}

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

func TestORMStoreCreatePersistsNewTaskWithoutReadBack(t *testing.T) {
	store := setupORMStoreTest(t)
	now := time.Now().UTC().Truncate(time.Millisecond)
	item := &Task{ID: "create-fast-path", Skill: "symbol_analysis", Status: StatusQueued, Stage: "queued", Input: `{"symbol":"BTCUSDT"}`, CreatedAt: now, UpdatedAt: now}
	if err := store.Create(context.Background(), item); err != nil {
		t.Fatal(err)
	}
	stored, err := store.Get(context.Background(), item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.ID != item.ID || stored.Status != StatusQueued || stored.Skill != item.Skill {
		t.Fatalf("unexpected created task: %+v", stored)
	}
	if err := store.Create(context.Background(), item); err == nil {
		t.Fatal("duplicate Create unexpectedly succeeded")
	}
}

func TestORMStorePersistsTaskEventsAndRedactsSecrets(t *testing.T) {
	store := setupORMStoreTest(t)
	now := time.Now().UTC()
	item := &Task{
		ID: "task-persist", Skill: "symbol_analysis", ConversationID: "conv-1", Status: StatusRunning,
		RuntimeVersion: "1.0.0", SkillVersion: "1.0.0", PromptVersion: "1.0.0", PromptHash: "hash",
		ModelConfigID: 77, InputContractVersion: "input_v1", OutputContractVersion: "output_v1", SkillSource: "native", SkillSourceVersion: "v1",
		ExecutionMode: "react", Steps: []byte(`[{"step_id":"step-001","type":"tool","status":"succeeded"}]`), ResumeCount: 1,
		CheckpointJSON: `{"token":"secret-checkpoint","safe":true}`,
		Stage:          "waiting_tool", Progress: 45, Input: `{"symbol":"BTCUSDT","api_key":"secret-value"}`,
		Error: "authorization=top-secret", CreatedAt: now, UpdatedAt: now,
		Events: []Event{{TaskID: "task-persist", StepID: "step-001", StepType: "tool", Stage: "tool_result", Status: "success", ErrorType: "", Checkpoint: true, Message: "Bearer super-secret", Time: now}},
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
	if got.ExecutionMode != "react" || got.ResumeCount != 1 || !strings.Contains(string(got.Steps), "step-001") {
		t.Fatalf("runtime execution state was not persisted: %+v", got)
	}
	if got.Events[0].StepID != "step-001" || got.Events[0].StepType != "tool" || !got.Events[0].Checkpoint {
		t.Fatalf("structured step event was not persisted: %+v", got.Events[0])
	}
	if strings.Contains(got.CheckpointJSON, "secret-checkpoint") {
		t.Fatalf("checkpoint secret was not redacted: %s", got.CheckpointJSON)
	}
	if err := store.SaveCheckpoint(context.Background(), item.ID, `{"safe":true,"token":"another-secret"}`); err != nil {
		t.Fatal(err)
	}
	// Checkpoint persistence is idempotent. MySQL may report affected_rows=0
	// when the second UPDATE writes the same value, which must not be treated as
	// a missing task.
	if err := store.SaveCheckpoint(context.Background(), item.ID, `{"safe":true,"token":"another-secret"}`); err != nil {
		t.Fatalf("saving an unchanged checkpoint must succeed: %v", err)
	}
	checkpoint, err := store.LoadCheckpoint(context.Background(), item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(checkpoint, `"safe":true`) || strings.Contains(checkpoint, "another-secret") {
		t.Fatalf("unexpected persisted checkpoint: %s", checkpoint)
	}
	if err := store.ClearCheckpoint(context.Background(), item.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.LoadCheckpoint(context.Background(), item.ID); err == nil {
		t.Fatal("expected cleared checkpoint to be unavailable")
	}
	if got.RuntimeVersion != "1.0.0" || got.PromptHash != "hash" || got.ModelConfigID != 77 || got.OutputContractVersion != "output_v1" || got.SkillSource != "native" {
		t.Fatalf("version metadata was not persisted: %+v", got.VersionMetadata())
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

func TestToModelKeepsEmptyRuntimeTextExplicitForStrictSQLSchemas(t *testing.T) {
	now := time.Now().UTC()
	row := toModel(&Task{ID: "strict-schema", Skill: "symbol_analysis", Status: StatusQueued, CreatedAt: now, UpdatedAt: now})
	if row.PlanJSON == nil || row.StepsJSON == nil || row.CheckpointJSON == nil {
		t.Fatalf("runtime text columns must be explicit non-nil values: plan=%v steps=%v checkpoint=%v", row.PlanJSON, row.StepsJSON, row.CheckpointJSON)
	}
	if *row.PlanJSON != "" || *row.StepsJSON != "" || *row.CheckpointJSON != "" {
		t.Fatalf("empty runtime state must persist as empty strings: plan=%q steps=%q checkpoint=%q", *row.PlanJSON, *row.StepsJSON, *row.CheckpointJSON)
	}
}

package workflow

import (
	"context"
	"encoding/json"
	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
	"go_binance_futures/models"
	"sync"
	"testing"
	"time"
)

var workflowStoreOnce sync.Once
var workflowStoreErr error

func setupWorkflowStore(t *testing.T) Store {
	workflowStoreOnce.Do(func() {
		if err := orm.RegisterDriver("sqlite3", orm.DRSqlite); err != nil {
			workflowStoreErr = err
			return
		}
		if err := orm.RegisterDataBase("default", "sqlite3", "file:workflow_store_test?mode=memory&cache=shared"); err != nil {
			workflowStoreErr = err
			return
		}
		orm.RegisterModel(new(models.AgentWorkflowRun))
		workflowStoreErr = orm.RunSyncdb("default", true, false)
	})
	if workflowStoreErr != nil {
		t.Fatal(workflowStoreErr)
	}
	_, _ = orm.NewOrm().Raw("DELETE FROM agent_workflow_runs").Exec()
	return Store{}
}
func TestWorkflowStorePersistsParentAndChildTasks(t *testing.T) {
	store := setupWorkflowStore(t)
	now := time.Now().UnixMilli()
	run := Run{ID: "wf-test", Workflow: StrategyExperiment, SchemaVersion: "strategy_experiment_result_v1", Status: "running", Stage: "summarizing", Input: json.RawMessage(`{"goal":"x"}`), ChildTaskIDs: []string{"task-a", "task-b"}, CreatedAt: now, UpdatedAt: now}
	if err := store.Save(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Workflow != StrategyExperiment || len(got.ChildTaskIDs) != 2 || got.ChildTaskIDs[1] != "task-b" {
		t.Fatalf("unexpected stored run: %+v", got)
	}
}

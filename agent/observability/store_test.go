package observability

import (
	"context"
	"sync"
	"testing"
	"time"

	agentruntime "go_binance_futures/agent/runtime"
	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
)

var obsStoreOnce sync.Once
var obsStoreErr error

func setupObsStore(t *testing.T) Store {
	t.Helper()
	obsStoreOnce.Do(func() {
		if err := orm.RegisterDriver("sqlite3", orm.DRSqlite); err != nil {
			obsStoreErr = err
			return
		}
		obsStoreErr = orm.RegisterDataBase("default", "sqlite3", "file:observability_test?mode=memory&cache=shared")
		if obsStoreErr != nil {
			return
		}
		orm.RegisterModel(new(models.AgentObservation), new(models.AgentChangeEvent), new(models.AgentTask), new(models.AgentMCPServer))
		obsStoreErr = orm.RunSyncdb("default", true, false)
	})
	if obsStoreErr != nil {
		t.Fatal(obsStoreErr)
	}
	o := orm.NewOrmUsingDB("default")
	_, _ = o.Raw("DELETE FROM agent_observations").Exec()
	_, _ = o.Raw("DELETE FROM agent_change_events").Exec()
	_, _ = o.Raw("DELETE FROM agent_tasks").Exec()
	_, _ = o.Raw("DELETE FROM agent_mcp_servers").Exec()
	return Store{}
}

func TestPersistentTraceAndSummary(t *testing.T) {
	store := setupObsStore(t)
	ctx := context.Background()
	now := time.Now().UTC().UnixMilli()
	o := orm.NewOrmUsingDB("default")
	_, err := o.Insert(&models.AgentTask{ID: "task-observe", Skill: "symbol_analysis", Status: "succeeded", Provider: "gemini", Model: "gemini-test", SkillVersion: "2", PromptVersion: "3", FinalModelConfigID: 9, TotalTokens: 1200, Round: 2, CreatedAt: now - 1000, StartedAt: now - 900, CompletedAt: now - 100})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.InsertObservation(ctx, agentruntime.Observation{Type: "context_build", TaskID: "task-observe", Skill: "symbol_analysis", StepID: "step-1", StepType: "llm", Status: "success", ContextTokens: 700, ContextBlocks: 5, TrimmedBlocks: 1, MemorySelected: 2}); err != nil {
		t.Fatal(err)
	}
	if err := store.InsertObservation(ctx, agentruntime.Observation{Type: "tool_call", TaskID: "task-observe", Skill: "symbol_analysis", StepID: "step-2", StepType: "tool", Tool: "mcp.test", ToolSource: "mcp", ProviderRef: "mcp-server:7", Status: "success", DurationMs: 80, CacheHit: true, EvidenceCount: 2}); err != nil {
		t.Fatal(err)
	}
	traces, err := store.ListTraces(ctx, TraceListOptions{TaskID: "task-observe", Page: 1, Limit: 20})
	if err != nil || traces.Total != 2 {
		t.Fatalf("traces=%+v err=%v", traces, err)
	}
	summary, err := store.Summary(ctx, now-5000, now+1000)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Global.Tasks != 1 || summary.Global.Succeeded != 1 || summary.Global.TotalTokens != 1200 {
		t.Fatalf("task summary=%+v", summary.Global)
	}
	if summary.Context.Builds != 1 || summary.Context.MemoryHitRate != 1 || summary.Context.TrimRate != 1 {
		t.Fatalf("context=%+v", summary.Context)
	}
	if len(summary.Tools) != 1 || summary.Tools[0].CacheHitRate != 1 {
		t.Fatalf("tools=%+v", summary.Tools)
	}
}

func TestChangeHistoryPersistsRedactedDetail(t *testing.T) {
	store := setupObsStore(t)
	ctx := context.Background()
	if err := store.RecordChange(ctx, ChangeEvent{Category: "mcp", EntityType: "mcp_server", EntityID: 3, EntityName: "demo", ChangeType: "catalog_changed", BeforeHash: "old", AfterHash: "new", Status: "success", Detail: map[string]any{"authorization": "Bearer secret-value"}}); err != nil {
		t.Fatal(err)
	}
	result, err := store.ListChanges(ctx, ChangeListOptions{Category: "mcp", Page: 1, Limit: 20})
	if err != nil || result.Total != 1 {
		t.Fatalf("changes=%+v err=%v", result, err)
	}
	if result.List[0].DetailJSON == "" || result.List[0].DetailJSON == `{"authorization":"Bearer secret-value"}` {
		t.Fatalf("detail was not redacted: %s", result.List[0].DetailJSON)
	}
}

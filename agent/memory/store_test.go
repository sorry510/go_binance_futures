package memory

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"go_binance_futures/agent/task"
	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
)

var memoryStoreTestOnce sync.Once
var memoryStoreTestErr error

func setupMemoryStoreTest(t *testing.T) *ORMStore {
	t.Helper()
	memoryStoreTestOnce.Do(func() {
		memoryStoreTestErr = orm.RegisterDriver("sqlite3", orm.DRSqlite)
		if memoryStoreTestErr != nil {
			return
		}
		orm.RegisterModel(new(models.AgentMemory))
		memoryStoreTestErr = orm.RegisterDataBase("default", "sqlite3", "file:agent_memory_test?mode=memory&cache=shared")
		if memoryStoreTestErr == nil {
			memoryStoreTestErr = orm.RunSyncdb("default", true, false)
		}
	})
	if memoryStoreTestErr != nil {
		t.Fatal(memoryStoreTestErr)
	}
	db, err := orm.GetDB("default")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("DELETE FROM agent_memories"); err != nil {
		t.Fatal(err)
	}
	return &ORMStore{}
}

func TestMarketHypothesisExpiresAndIsExcludedFromContext(t *testing.T) {
	store := setupMemoryStoreTest(t)
	ctx := context.Background()
	now := time.Now().UTC()
	item, err := store.Create(ctx, CreateInput{Type: TypeMarketHypothesis, Scope: Scope{Skill: "symbol_analysis", Symbol: "BTCUSDT"}, Confidence: 0.8, Content: "short-term squeeze risk", ExpiresAt: now.Add(time.Minute).UnixMilli()})
	if err != nil {
		t.Fatal(err)
	}
	if item.ExpiresAt == nil || item.ExpiresAt.IsZero() {
		t.Fatal("market hypothesis must have ttl")
	}
	if err := store.Expire(ctx, now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	stored, err := store.Get(ctx, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != StatusExpired {
		t.Fatalf("status=%s", stored.Status)
	}
	blocks, err := store.ContextBlocks(ctx, QueryScope{Skill: "symbol_analysis", Symbol: "BTCUSDT", Now: now.Add(2 * time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 0 {
		t.Fatalf("expired memory leaked into context: %+v", blocks)
	}
}

func TestQueryMatchesAllNonEmptyScopes(t *testing.T) {
	store := setupMemoryStoreTest(t)
	ctx := context.Background()
	inputs := []CreateInput{
		{Type: TypeLesson, Scope: Scope{User: DefaultUserScope}, Confidence: 1, Content: "global user lesson"},
		{Type: TypeLesson, Scope: Scope{Skill: "symbol_analysis", Symbol: "BTCUSDT"}, Confidence: 1, Content: "btc skill lesson"},
		{Type: TypeLesson, Scope: Scope{Skill: "symbol_analysis", Symbol: "ETHUSDT"}, Confidence: 1, Content: "eth skill lesson"},
		{Type: TypeStrategyFact, Scope: Scope{Skill: "strategy_builder", Strategy: "s-1"}, Confidence: 0.9, Content: "strategy fact"},
	}
	for _, input := range inputs {
		if _, err := store.Create(ctx, input); err != nil {
			t.Fatal(err)
		}
	}
	items, err := store.Query(ctx, QueryScope{User: DefaultUserScope, Skill: "symbol_analysis", Symbol: "BTCUSDT", Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("matched=%d want=2: %+v", len(items), items)
	}
	joined := items[0].Content + "\n" + items[1].Content
	if !strings.Contains(joined, "global user lesson") || !strings.Contains(joined, "btc skill lesson") || strings.Contains(joined, "eth") {
		t.Fatalf("unexpected scope match: %s", joined)
	}
}

func TestTaskSummaryWriteIsSafeAndIdempotent(t *testing.T) {
	store := setupMemoryStoreTest(t)
	service := Service{Store: store}
	ctx := context.Background()
	item := &task.Task{ID: "task-1", Skill: "symbol_analysis", Status: task.StatusSucceeded}
	first, err := service.PersistTaskSummary(ctx, "symbol_analysis", `{"symbol":"BTCUSDT"}`, nil, item, "BTC trend summary")
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.PersistTaskSummary(ctx, "symbol_analysis", `{"symbol":"BTCUSDT"}`, nil, item, "duplicate")
	if err != nil {
		t.Fatal(err)
	}
	if first == nil || second == nil || first.ID != second.ID {
		t.Fatalf("task summary write is not idempotent: first=%+v second=%+v", first, second)
	}
	if first.Scope.Symbol != "BTCUSDT" || first.Scope.Skill != "symbol_analysis" {
		t.Fatalf("scope not captured: %+v", first.Scope)
	}
	if err := ValidateAutomaticWrite(TypeStrategyFact); err == nil {
		t.Fatal("strategy_fact must require approval for automatic writes")
	}
	if err := ValidateAutomaticWrite(TypeMarketHypothesis); err == nil {
		t.Fatal("market_hypothesis must require approval for automatic writes")
	}
}

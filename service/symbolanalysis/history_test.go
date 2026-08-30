package symbolanalysis

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
)

func TestHistoryServicePersistsAndListsAnalysis(t *testing.T) {
	const alias = "symbol_analysis_history_test"
	if err := orm.RegisterDriver("sqlite3", orm.DRSqlite); err != nil {
		t.Fatal(err)
	}
	if err := orm.RegisterDataBase(alias, "sqlite3", filepath.Join(t.TempDir(), "history.db")); err != nil {
		t.Fatal(err)
	}
	orm.RegisterModel(new(models.SymbolAnalysisHistory))
	if err := orm.RunSyncdb(alias, false, false); err != nil {
		t.Fatal(err)
	}

	service := HistoryService{Alias: alias}
	now := time.Now().UTC()
	result := json.RawMessage(`{"version":"trading_plan_v1","symbol":"ONGUSDT","market_condition":2,"direction":"long","confidence":0.73,"summary":"偏多"}`)
	if err := service.Save(context.Background(), HistorySaveRequest{
		TaskID: "task-1", Symbol: "ONGUSDT", Prompt: "分析",
		Status: "succeeded", Result: result, Provider: "test", Model: "fake",
		AnalysisPrice: 100, CreatedAt: now.Add(-time.Minute).UnixMilli(), CompletedAt: now.UnixMilli(),
	}); err != nil {
		t.Fatal(err)
	}
	if err := service.Save(context.Background(), HistorySaveRequest{
		TaskID: "task-2", Symbol: "ONGUSDT", Prompt: "再次分析",
		Status: "failed", Error: "llm failed", AnalysisPrice: 101,
		CreatedAt: now.UnixMilli(), CompletedAt: now.Add(time.Second).UnixMilli(),
	}); err != nil {
		t.Fatal(err)
	}

	listed, err := service.List(context.Background(), HistoryListOptions{
		Symbol: "ONGUSDT", Status: "succeeded", Page: 1, Limit: 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if listed.Total != 1 || len(listed.List) != 1 {
		t.Fatalf("unexpected history result: %+v", listed)
	}
	item := listed.List[0]
	if item.TaskID != "task-1" || item.Direction != "long" || item.MarketCondition != 2 {
		t.Fatalf("unexpected stored metadata: %+v", item)
	}
	if item.Confidence != 0.73 || item.AnalysisPrice != 100 || item.Summary != "偏多" {
		t.Fatalf("unexpected stored plan fields: %+v", item)
	}
	if string(item.Result) != string(result) {
		t.Fatalf("result json mismatch: %s", string(item.Result))
	}
}

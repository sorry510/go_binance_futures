package alertpipeline

import (
	"context"
	"sync"
	"testing"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
	"go_binance_futures/models"
	signalservice "go_binance_futures/service/signal"
)

var traceStoreOnce sync.Once
var traceStoreErr error

func setupTraceStore(t *testing.T) ORMTraceStore {
	t.Helper()
	traceStoreOnce.Do(func() {
		_ = orm.RegisterDriver("sqlite3", orm.DRSqlite)
		orm.RegisterModel(new(models.AgentAlertPipelineTrace), new(models.AgentTask), new(models.Notification))
		traceStoreErr = orm.RegisterDataBase("default", "sqlite3", "file:alert_trace_test?mode=memory&cache=shared")
	})
	if traceStoreErr != nil {
		t.Fatal(traceStoreErr)
	}
	if err := orm.RunSyncdb("default", true, false); err != nil {
		t.Fatal(err)
	}
	return ORMTraceStore{Alias: "default"}
}

func TestORMTraceStoreUpsertsLatestSignalStateAndPaginates(t *testing.T) {
	store := setupTraceStore(t)
	ctx := context.Background()
	first := Trace{EventID: "evt-1", SignalID: "sig-1", Symbol: "BTCUSDT", Type: signalservice.TypeFastMove, Severity: signalservice.SeverityHigh, Status: "received", CreatedAt: 1000, UpdatedAt: 1000}
	if err := store.Save(ctx, first); err != nil {
		t.Fatal(err)
	}
	first.TaskID = "task-1"
	first.NotificationID = 7
	first.Action = "notify"
	first.Status = "notified"
	first.UpdatedAt = 2000
	if err := store.Save(ctx, first); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(ctx, Trace{EventID: "evt-2", SignalID: "sig-2", Symbol: "ETHUSDT", Type: signalservice.TypeLiquidationSpike, Severity: signalservice.SeverityMedium, Status: "cooldown", CreatedAt: 3000, UpdatedAt: 3000}); err != nil {
		t.Fatal(err)
	}
	o := orm.NewOrmUsingDB("default")
	if _, err := o.Insert(&models.AgentTask{ID: "task-1", Skill: "alert_analysis", Status: "succeeded", Stage: "completed", CreatedAt: 1000, UpdatedAt: 2000}); err != nil {
		t.Fatal(err)
	}
	if _, err := o.Insert(&models.Notification{ID: 7, Title: "alert", Content: "dingding compare", Module: "futures_market_listen", CreateTime: 2000}); err != nil {
		t.Fatal(err)
	}

	result, err := store.List(ctx, TraceListOptions{Page: 1, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 2 || len(result.List) != 1 || result.List[0].SignalID != "sig-2" {
		t.Fatalf("unexpected first page: %+v", result)
	}
	result, err = store.List(ctx, TraceListOptions{Page: 1, Limit: 20, Symbol: "BTCUSDT", Status: "notified"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 1 || len(result.List) != 1 || result.List[0].SignalID != "sig-1" || result.List[0].TaskID != "task-1" || result.List[0].TaskStatus != "succeeded" || result.List[0].Notification == nil || result.List[0].Notification.ID != 7 || result.List[0].CreatedAt != 1000 || result.List[0].UpdatedAt != 2000 {
		t.Fatalf("upsert/filter mismatch: %+v", result)
	}
}

package systemhealth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"go_binance_futures/agent/scheduler"
	"go_binance_futures/appversion"
	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
)

func setupHealthDB(t *testing.T, now time.Time) orm.Ormer {
	t.Helper()
	if err := orm.RegisterDriver("sqlite3", orm.DRSqlite); err != nil {
		t.Fatal(err)
	}
	orm.RegisterModel(
		new(models.Config), new(models.Symbols), new(models.AgentMarketSourceStatus),
		new(models.AgentMCPServer), new(models.LLMConfig), new(models.AgentObservation),
		new(models.AgentTask), new(models.AgentTradeProposal), new(models.FuturesManagedPosition),
		new(models.FuturesManagedOrder),
	)
	if err := orm.RegisterDataBase("default", "sqlite3", filepath.Join(t.TempDir(), "health.db")); err != nil {
		t.Fatal(err)
	}
	if err := orm.RunSyncdb("default", true, false); err != nil {
		t.Fatal(err)
	}
	o := orm.NewOrm()
	cfg := models.Config{ID: 1, Version: appversion.DatabaseSchemaVersion, WsFuturesEnable: 1, AgentOpportunityWatchEnable: 1, AgentOpportunityScanIntervalMin: 60}
	if _, err := o.Insert(&cfg); err != nil {
		t.Fatal(err)
	}
	if _, err := o.Insert(&models.Symbols{Symbol: "BTCUSDT", UpdateTime: now.Add(-time.Minute).UnixMilli()}); err != nil {
		t.Fatal(err)
	}
	if _, err := o.Insert(&models.LLMConfig{Name: "fixture", Provider: "mock", Model: "fixture", Enabled: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := o.Insert(&models.AgentObservation{TaskID: "task-1", Type: "llm_call", Skill: "market_scan", Status: "success", CreatedAt: now.Add(-time.Minute).UnixMilli()}); err != nil {
		t.Fatal(err)
	}
	if _, err := o.Insert(&models.AgentMCPServer{Name: "fixture-mcp", Endpoint: "http://127.0.0.1", Enabled: 1, Status: "healthy", LastSuccessAt: now.Add(-time.Minute).UnixMilli()}); err != nil {
		t.Fatal(err)
	}
	if _, err := o.Insert(&models.AgentMarketSourceStatus{Source: "fixture-source", Status: "healthy", LastSuccessAt: now.Add(-time.Minute).UnixMilli(), UpdatedAt: now.UnixMilli()}); err != nil {
		t.Fatal(err)
	}
	return o
}

func TestReportUsesDeterministicHealthSignals(t *testing.T) {
	now := time.UnixMilli(1800000000000)
	o := setupHealthDB(t, now)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/fapi/v1/time" {
			t.Fatalf("unexpected Binance health path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"serverTime":1800000000000}`))
	}))
	defer server.Close()

	report, err := (Service{}).Report(context.Background(), Options{
		Now: now, CheckBinanceREST: true, BinanceBaseURL: server.URL, IgnoreConfiguredBinanceProxy: true,
		SchedulerRuntimeAvailable: true,
		SchedulerJobs:             []scheduler.JobStatus{{Name: "opportunity_market_scan", Skill: "market_scan", Enabled: true, IntervalSeconds: 3600, LastStatus: "succeeded", LastRunAt: now.Add(-time.Minute).UnixMilli()}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Database.Status != StatusHealthy || report.Database.Version != appversion.DatabaseSchemaVersion {
		t.Fatalf("unexpected database health: %+v", report.Database)
	}
	if report.BinanceREST.Status != StatusHealthy || report.FuturesWS.Status != StatusHealthy {
		t.Fatalf("unexpected Binance health: rest=%+v ws=%+v", report.BinanceREST, report.FuturesWS)
	}
	if report.MCP.Status != StatusHealthy || report.LLM.Status != StatusHealthy || report.Scheduler.Status != StatusHealthy {
		t.Fatalf("unexpected subsystem health: mcp=%+v llm=%+v scheduler=%+v", report.MCP, report.LLM, report.Scheduler)
	}
	if report.Trade.Status != StatusHealthy {
		t.Fatalf("unexpected trade health: %+v", report.Trade)
	}

	if _, err := o.Insert(&models.AgentObservation{TaskID: "task-llm-fail", Type: "llm_call", Skill: "market_scan", Status: "error", Error: "boom provider 500", CreatedAt: now.Add(30 * time.Second).UnixMilli()}); err != nil {
		t.Fatal(err)
	}
	if _, err := o.Insert(&models.AgentTask{ID: "task-ok", Skill: "market_scan", Status: "succeeded", CreatedAt: now.Add(-time.Hour).UnixMilli(), CompletedAt: now.Add(-time.Hour).UnixMilli()}); err != nil {
		t.Fatal(err)
	}
	if _, err := o.Insert(&models.AgentTask{ID: "task-max-rounds", Skill: "symbol_analysis", Status: "failed", Error: "agent reached maximum rounds", Round: 8, MaxRounds: 8, CreatedAt: now.Add(-time.Minute).UnixMilli(), CompletedAt: now.Add(-time.Minute).UnixMilli()}); err != nil {
		t.Fatal(err)
	}
	if _, err := o.Insert(&models.AgentMarketSourceStatus{Source: "paused-source", Status: "disabled", UpdatedAt: now.UnixMilli()}); err != nil {
		t.Fatal(err)
	}
	report, err = (Service{}).Report(context.Background(), Options{Now: now, BinanceBaseURL: server.URL, IgnoreConfiguredBinanceProxy: true, CheckBinanceREST: true})
	if err != nil {
		t.Fatal(err)
	}
	if report.LLM.Status != StatusWarning || report.LLM.LastError != "boom provider 500" {
		t.Fatalf("latest LLM failure not surfaced: %+v", report.LLM)
	}
	if report.Agent.Tasks24h != 2 || report.Agent.Failed24h != 1 || report.Agent.MaxRoundsFailed != 1 {
		t.Fatalf("unexpected agent 24h stats: %+v", report.Agent)
	}
	if report.Scheduler.Status != StatusHealthy || len(report.SchedulerJobs) != 3 || report.SchedulerJobs[1].LastStatus != "succeeded" {
		t.Fatalf("CLI scheduler freshness not surfaced: scheduler=%+v jobs=%+v", report.Scheduler, report.SchedulerJobs)
	}
	if report.MarketIntelligence.Status != StatusWarning {
		t.Fatalf("non-healthy market source must not be counted as healthy: %+v", report.MarketIntelligence)
	}

	if _, err := o.Raw("UPDATE config SET version=? WHERE id=1", appversion.DatabaseSchemaVersion-1).Exec(); err != nil {
		t.Fatal(err)
	}
	report, err = (Service{}).Report(context.Background(), Options{Now: now, BinanceBaseURL: server.URL, IgnoreConfiguredBinanceProxy: true, CheckBinanceREST: true})
	if err != nil {
		t.Fatal(err)
	}
	if report.Database.Status != StatusError || report.Database.Version != appversion.DatabaseSchemaVersion-1 || report.Database.RequiredVersion != appversion.DatabaseSchemaVersion || report.Overall != StatusError {
		t.Fatalf("schema mismatch not surfaced: database=%+v overall=%s", report.Database, report.Overall)
	}
	if _, err := o.Raw("UPDATE config SET version=? WHERE id=1", appversion.DatabaseSchemaVersion).Exec(); err != nil {
		t.Fatal(err)
	}

	if _, err := o.Insert(&models.AgentTradeProposal{ProposalID: "uncertain", SourceTaskID: "task-1", Symbol: "BTCUSDT", Side: "LONG", Status: "execution_uncertain", CreatedAt: now.UnixMilli(), UpdatedAt: now.UnixMilli()}); err != nil {
		t.Fatal(err)
	}
	if _, err := o.Insert(&models.FuturesManagedPosition{Owner: "agent_trade", Symbol: "BTCUSDT", PositionSide: "LONG", ManagedQty: 1, Status: "reconcile_required", SourceRef: "uncertain", CreatedAt: now.UnixMilli(), UpdatedAt: now.UnixMilli()}); err != nil {
		t.Fatal(err)
	}
	if _, err := o.Insert(&models.FuturesManagedOrder{Owner: "agent_trade", Symbol: "BTCUSDT", PositionSide: "LONG", Intent: "stop", ClientOrderID: "fixture-reconcile-order", OrderType: "STOP_MARKET", Status: "reconcile_required", SourceRef: "uncertain", CreatedAt: now.UnixMilli(), UpdatedAt: now.UnixMilli()}); err != nil {
		t.Fatal(err)
	}
	report, err = (Service{}).Report(context.Background(), Options{Now: now, BinanceBaseURL: server.URL, IgnoreConfiguredBinanceProxy: true, CheckBinanceREST: true, SchedulerRuntimeAvailable: true, SchedulerJobs: []scheduler.JobStatus{}})
	if err != nil {
		t.Fatal(err)
	}
	if report.Trade.Status != StatusError || report.Trade.ExecutionUncertain != 1 || report.Trade.ReconcileRequired != 2 {
		t.Fatalf("trade safety issue not surfaced: %+v", report.Trade)
	}
	if report.Overall != StatusError {
		t.Fatalf("overall must reflect trade safety error: %+v", report)
	}
}

func TestSchedulerDefinitionsRespectConfig(t *testing.T) {
	items := schedulerDefinitions(models.Config{MarketConditionIsAuto: 1, AgentMarketRegimeScheduleEnable: 1, AgentMarketRegimeIntervalMin: 30, AgentOpportunityWatchEnable: 1, AgentOpportunityScanIntervalMin: 60})
	if len(items) != 3 || !items[0].Enabled || items[0].IntervalSeconds != 1800 || !items[1].Enabled || items[2].Enabled {
		t.Fatalf("unexpected scheduler definitions: %+v", items)
	}
}

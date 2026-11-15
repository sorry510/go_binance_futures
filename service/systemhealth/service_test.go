package systemhealth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"go_binance_futures/agent/scheduler"
	"go_binance_futures/appversion"
	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
)

var healthDBOnce sync.Once
var healthDBErr error

func setupHealthDB(t *testing.T, now time.Time) orm.Ormer {
	t.Helper()
	if err := orm.RegisterDriver("sqlite3", orm.DRSqlite); err != nil {
		t.Fatal(err)
	}
	healthDBOnce.Do(func() {
		orm.RegisterModel(
			new(models.Config), new(models.Symbols), new(models.SpotSymbols), new(models.DeliverySymbols), new(models.AgentMarketSourceStatus),
			new(models.AgentMCPServer), new(models.LLMConfig), new(models.AgentObservation),
			new(models.AgentTask), new(models.AgentTradeProposal), new(models.FuturesManagedPosition),
			new(models.FuturesManagedOrder),
		)
		healthDBErr = orm.RegisterDataBase("default", "sqlite3", "file:s510_systemhealth_default?mode=memory&cache=shared")
	})
	if healthDBErr != nil {
		t.Fatal(healthDBErr)
	}
	if err := orm.RunSyncdb("default", true, false); err != nil {
		t.Fatal(err)
	}
	o := orm.NewOrm()
	// Fill is deliberately not a Beego model; create only a SQLite test table.
	if _, err := o.Raw(`CREATE TABLE IF NOT EXISTS futures_trade_fills (
		account_id TEXT, symbol TEXT, order_id TEXT, trade_id TEXT,
		qty DECIMAL(36,18), rest_verified INTEGER
	)`).Exec(); err != nil {
		t.Fatal(err)
	}
	if _, err := o.Raw("DELETE FROM futures_trade_fills").Exec(); err != nil {
		t.Fatal(err)
	}
	cfg := models.Config{ID: 1, Version: appversion.DatabaseSchemaVersion, AgentOpportunityWatchEnable: 1, AgentOpportunityScanIntervalMin: 60}
	if _, err := o.Insert(&cfg); err != nil {
		t.Fatal(err)
	}
	if _, err := o.Insert(&models.Symbols{Symbol: "BTCUSDT", UpdateTime: now.Add(-time.Minute).UnixMilli()}); err != nil {
		t.Fatal(err)
	}
	if _, err := o.Insert(&models.SpotSymbols{Symbol: "BTCUSDT", UpdateTime: now.Add(-time.Minute).UnixMilli()}); err != nil {
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
	if report.MarketWS.Status != StatusHealthy || len(report.WSMarkets) != 3 ||
		report.WSMarkets[0].Market != "futures" || report.WSMarkets[0].Status != StatusHealthy ||
		report.WSMarkets[1].Market != "spot" || report.WSMarkets[1].Status != StatusHealthy ||
		report.WSMarkets[2].Market != "delivery" || report.WSMarkets[2].Enabled || report.WSMarkets[2].Status != StatusDisabled {
		t.Fatalf("unexpected WS market statuses: overall=%+v markets=%+v", report.MarketWS, report.WSMarkets)
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
	if _, err := o.Insert(&models.FuturesManagedPosition{AccountID: "main", Owner: "agent_trade", Symbol: "BTCUSDT", PositionSide: "LONG", ManagedQty: 1, Status: "reconcile_required", SourceRef: "uncertain", CreatedAt: now.UnixMilli(), UpdatedAt: now.UnixMilli()}); err != nil {
		t.Fatal(err)
	}
	if _, err := o.Insert(&models.FuturesManagedOrder{AccountID: "main", Owner: "agent_trade", Symbol: "BTCUSDT", PositionSide: "LONG", Intent: "stop", ClientOrderID: "fixture-reconcile-order", OrderType: "STOP_MARKET", Status: "reconcile_required", SourceRef: "uncertain", CreatedAt: now.UnixMilli(), UpdatedAt: now.UnixMilli()}); err != nil {
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

func TestMarketWSStatusAndDisabledDelivery(t *testing.T) {
	now := time.UnixMilli(1800000000000)
	o := setupHealthDB(t, now)
	healthy := checkFuturesWS(o, now)
	overall, markets := checkMarketWS(o, now, healthy, false)
	if overall.Status != StatusHealthy || markets[2].Status != StatusDisabled {
		t.Fatalf("disabled delivery must not degrade health: %+v %+v", overall, markets)
	}
	if _, err := o.Raw("UPDATE spot_symbols SET updateTime=? WHERE symbol=?", now.Add(-16*time.Minute).UnixMilli(), "BTCUSDT").Exec(); err != nil {
		t.Fatal(err)
	}
	overall, markets = checkMarketWS(o, now, healthy, false)
	if overall.Status != StatusError || markets[1].Status != StatusError || markets[2].Status != StatusDisabled {
		t.Fatalf("stale spot should degrade WS while disabled delivery stays ignored: %+v %+v", overall, markets)
	}
	if _, err := o.Raw("UPDATE spot_symbols SET updateTime=? WHERE symbol=?", now.Add(-time.Minute).UnixMilli(), "BTCUSDT").Exec(); err != nil {
		t.Fatal(err)
	}
	if _, err := o.Insert(&models.DeliverySymbols{Symbol: "BTCUSD_PERP", UpdateTime: now.Add(-time.Minute).UnixMilli()}); err != nil {
		t.Fatal(err)
	}
	overall, markets = checkMarketWS(o, now, healthy, true)
	if overall.Status != StatusHealthy || !markets[2].Enabled || markets[2].Status != StatusHealthy {
		t.Fatalf("enabled delivery should use its own timestamp: %+v %+v", overall, markets)
	}
	if _, err := o.Raw("UPDATE delivery_symbols SET updateTime=? WHERE symbol=?", now.Add(-10*time.Minute).UnixMilli(), "BTCUSD_PERP").Exec(); err != nil {
		t.Fatal(err)
	}
	overall, markets = checkMarketWS(o, now, healthy, true)
	if overall.Status != StatusWarning || markets[2].Status != StatusWarning {
		t.Fatalf("delivery freshness warning not propagated: %+v %+v", overall, markets)
	}
}

func TestTradeHealthSeparatesUnverifiedFillsFromExecutionRisk(t *testing.T) {
	now := time.UnixMilli(1800000000000)
	o := setupHealthDB(t, now)
	closedID, err := o.Insert(&models.FuturesManagedPosition{
		AccountID: "main", Owner: "auto_strategy", Symbol: "ETHUSDT", PositionSide: "SHORT",
		Status: "closed", SettlementStatus: "incomplete", CreatedAt: now.UnixMilli(), ClosedAt: now.UnixMilli(),
	})
	if err != nil {
		t.Fatal(err)
	}
	activeID, err := o.Insert(&models.FuturesManagedPosition{
		AccountID: "main", Owner: "auto_strategy", Symbol: "XRPUSDT", PositionSide: "LONG",
		Status: "active", ManagedQty: 10, SettlementStatus: "open", CreatedAt: now.UnixMilli(),
	})
	if err != nil {
		t.Fatal(err)
	}
	insertOrder := func(symbol, client, exchange string, positionID int64, requested, filled float64) {
		t.Helper()
		if _, err := o.Insert(&models.FuturesManagedOrder{
			AccountID: "main", Owner: "auto_strategy", Symbol: symbol, PositionSide: "LONG",
			Intent: "open", Status: "reconcile_required", ClientOrderID: client,
			ExchangeOrderID: exchange, PositionID: &positionID,
			RequestedQty: requested, FilledQty: filled, CreatedAt: now.UnixMilli(),
		}); err != nil {
			t.Fatal(err)
		}
	}
	insertOrder("ETHUSDT", "filled-closed", "1001", closedID, 2, 2)
	insertOrder("XRPUSDT", "filled-active", "1002", activeID, 10, 10)
	for _, row := range []struct {
		symbol, order, trade string
		qty                  float64
	}{
		{"ETHUSDT", "1001", "4001", 2},
		{"XRPUSDT", "1002", "4002", 10},
	} {
		if _, err := o.Raw("INSERT INTO futures_trade_fills(account_id,symbol,order_id,trade_id,qty,rest_verified) VALUES ('main',?,?,?,?,0)", row.symbol, row.order, row.trade, row.qty).Exec(); err != nil {
			t.Fatal(err)
		}
	}
	report := checkTrade(o)
	if report.Status != StatusWarning || report.ReconcileRequired != 0 ||
		report.FillVerificationPending != 2 || report.HistoricalFillVerification != 1 ||
		report.ActiveFillVerification != 1 || report.Count != 2 {
		t.Fatalf("complete local fills pending REST should warn, not show execution failure: %+v", report)
	}
	// Missing fill evidence must never be downgraded to a warning.
	insertOrder("XRPUSDT", "unknown-open", "1003", activeID, 10, 0)
	report = checkTrade(o)
	if report.Status != StatusError || report.ReconcileRequired != 1 ||
		report.FillVerificationPending != 2 || report.Count != 3 {
		t.Fatalf("unknown execution must stay an error: %+v", report)
	}
	// Even a completed local order with a mismatched ledger is risky.
	insertOrder("ETHUSDT", "mismatch", "1004", closedID, 5, 5)
	if _, err := o.Raw("INSERT INTO futures_trade_fills(account_id,symbol,order_id,trade_id,qty,rest_verified) VALUES ('main','ETHUSDT','1004','4004',2,0)").Exec(); err != nil {
		t.Fatal(err)
	}
	report = checkTrade(o)
	if report.ReconcileRequired != 2 || report.FillVerificationPending != 2 {
		t.Fatalf("ledger mismatch must not be hidden: %+v", report)
	}
}

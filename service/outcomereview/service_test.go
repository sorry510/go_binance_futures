package outcomereview

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
)

var outcomeDBOnce sync.Once
var outcomeDBErr error

func setupOutcomeDB(t *testing.T) {
	t.Helper()
	outcomeDBOnce.Do(func() {
		if err := orm.RegisterDriver("sqlite3", orm.DRSqlite); err != nil {
			outcomeDBErr = err
			return
		}
		orm.RegisterModel(new(models.AgentBacktestRun), new(models.AgentBacktestTrade), new(models.AgentTradeProposal), new(models.FuturesManagedPosition), new(models.FuturesManagedOrder))
		dir, err := os.MkdirTemp("", "outcome-review-test-*")
		if err != nil {
			outcomeDBErr = err
			return
		}
		if err := orm.RegisterDataBase("default", "sqlite3", filepath.Join(dir, "outcome.db")); err != nil {
			outcomeDBErr = err
			return
		}
		outcomeDBErr = orm.RunSyncdb("default", true, false)
	})
	if outcomeDBErr != nil {
		t.Fatal(outcomeDBErr)
	}
	o := orm.NewOrm()
	for _, table := range []string{"futures_managed_orders", "futures_managed_positions", "agent_trade_proposals", "agent_backtest_trades", "agent_backtest_runs"} {
		if _, err := o.Raw("DELETE FROM " + table).Exec(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestValidateFilter(t *testing.T) {
	if validateFilter(Filter{StartTime: 20, EndTime: 10}) == nil {
		t.Fatal("expected invalid time range")
	}
	if validateFilter(Filter{Side: "BUY"}) == nil {
		t.Fatal("expected invalid side")
	}
	if validateFilter(Filter{Side: "LONG"}) != nil {
		t.Fatal("LONG must be valid")
	}
}

func TestLiveManagedWhereUsesProposalDimensions(t *testing.T) {
	where, args := liveManagedWhere(Filter{
		StartTime: 10, EndTime: 20, Symbol: "btcusdt", Side: "long", MarketCondition: 3,
	}, "mp")
	for _, want := range []string{"mp.source_ref IN", "p.created_at>=?", "p.created_at<=?", "p.symbol=?", "p.side=?", "p.market_condition=?"} {
		if !strings.Contains(where, want) {
			t.Fatalf("managed filter missing %q: %s", want, where)
		}
	}
	if got, want := fmt.Sprint(args), "[10 20 BTCUSDT LONG 3]"; got != want {
		t.Fatalf("unexpected managed args: got %s want %s", got, want)
	}
}

func TestBacktestRunWhereRestrictsTradeDimensions(t *testing.T) {
	where, args := backtestRunWhere(Filter{StrategyTemplateID: 7, Side: "short", MarketCondition: 2}, "r")
	for _, want := range []string{"r.strategy_template_id=?", "EXISTS", "tf.side=?", "tf.market_condition=?"} {
		if !strings.Contains(where, want) {
			t.Fatalf("run filter missing %q: %s", want, where)
		}
	}
	if got, want := fmt.Sprint(args), "[7 SHORT 2]"; got != want {
		t.Fatalf("unexpected run args: got %s want %s", got, want)
	}
}

func TestBacktestSQLAggregateFixture(t *testing.T) {
	setupOutcomeDB(t)
	o := orm.NewOrm()
	run := models.AgentBacktestRun{
		RunID: "run-1", DatasetID: "dataset-1", DatasetSpecHash: "spec", DataHash: "data",
		StrategyTemplateID: 7, StrategyTemplateName: "fixture", StrategyVersion: "hash",
		TechnologyJSON: "{}", StrategyJSON: "[]", EngineVersion: "fixture", MarketConditionModel: "fixture",
		ResolutionMode: "standard_1m", ResolutionModel: "standard_1m_v1", Symbol: "BTCUSDT", ExecutionInterval: "1m",
		StartTime: 100, EndTime: 200, InitialEquity: 1000, PositionSizePct: 1, Leverage: 4,
		Status: "succeeded", Stage: "completed", Progress: 100, MetricsJSON: `{"max_drawdown_pct":7.5}`,
		CreatedAt: 90, UpdatedAt: 210, CompletedAt: 210,
	}
	if _, err := o.Insert(&run); err != nil {
		t.Fatal(err)
	}
	trades := []models.AgentBacktestTrade{
		{RunID: run.RunID, Sequence: 1, Symbol: "BTCUSDT", Side: "LONG", EntryTime: 110, ExitTime: 120, EntryPrice: 100, ExitPrice: 110, Quantity: 1, NetPnL: 10, GrossPnL: 11, Fees: 1, FundingPnL: 0.5, HoldingMs: 1000, ExitReason: "take_profit", MarketCondition: 1},
		{RunID: run.RunID, Sequence: 2, Symbol: "BTCUSDT", Side: "SHORT", EntryTime: 130, ExitTime: 150, EntryPrice: 110, ExitPrice: 114, Quantity: 1, NetPnL: -4, GrossPnL: -3.5, Fees: 0.5, FundingPnL: -0.2, HoldingMs: 3000, ExitReason: "stop_loss", MarketCondition: 3},
	}
	for i := range trades {
		if _, err := o.Insert(&trades[i]); err != nil {
			t.Fatal(err)
		}
	}

	got, err := (Service{}).Backtest(context.Background(), Filter{StrategyTemplateID: 7, Symbol: "btcusdt"})
	if err != nil {
		t.Fatal(err)
	}
	if got.RunCount != 1 || got.TradeCount != 2 || math.Abs(got.NetPnL-6) > 1e-9 || math.Abs(got.ReturnPct-0.6) > 1e-9 {
		t.Fatalf("unexpected SQL summary: %+v", got)
	}
	if !got.MaxDrawdownAvailable || math.Abs(got.MaxDrawdownPct-7.5) > 1e-9 || math.Abs(got.ProfitFactor-2.5) > 1e-9 {
		t.Fatalf("unexpected SQL metrics: %+v", got)
	}
	if len(got.BySymbol) != 1 || len(got.BySide) != 2 || len(got.ByMarketCondition) != 2 {
		t.Fatalf("unexpected SQL groups: %+v", got)
	}
	if got.ByMarketCondition[0].Key != "1" || got.ByMarketCondition[1].Key != "3" {
		t.Fatalf("unexpected market condition keys: %+v", got.ByMarketCondition)
	}

	longOnly, err := (Service{}).Backtest(context.Background(), Filter{StrategyTemplateID: 7, Side: "long"})
	if err != nil {
		t.Fatal(err)
	}
	if longOnly.RunCount != 1 || longOnly.TradeCount != 1 || longOnly.NetPnL != 10 || math.Abs(longOnly.ReturnPct-1) > 1e-9 {
		t.Fatalf("unexpected LONG summary: %+v", longOnly)
	}
	if longOnly.MaxDrawdownAvailable || longOnly.MaxDrawdownPct != 0 {
		t.Fatalf("direction-level drawdown must be unavailable: %+v", longOnly)
	}
}

func TestLiveSummaryIncludesTraceableRecentProposalsAndActiveQty(t *testing.T) {
	setupOutcomeDB(t)
	o := orm.NewOrm()
	proposals := []models.AgentTradeProposal{
		{ProposalID: "p-old", SourceTaskID: "task-old", Symbol: "BTCUSDT", Side: "LONG", Status: "closed", RiskStatus: "pass", MarketCondition: 1, CreatedAt: 100, UpdatedAt: 120, ExpiresAt: 1000, ExecutedAt: 110},
		{ProposalID: "p-new", SourceTaskID: "task-new", Symbol: "ETHUSDT", Side: "SHORT", Status: "executed", RiskStatus: "pass", MarketCondition: 3, CreatedAt: 200, UpdatedAt: 220, ExpiresAt: 1000, ExecutedAt: 210},
	}
	for i := range proposals {
		if _, err := o.Insert(&proposals[i]); err != nil {
			t.Fatal(err)
		}
	}
	positions := []models.FuturesManagedPosition{
		{Owner: "agent_trade", Symbol: "ETHUSDT", PositionSide: "SHORT", ManagedQty: 2, Status: "open", SourceRef: "p-new", CreatedAt: 205, UpdatedAt: 205},
		{Owner: "agent_trade", Symbol: "ETHUSDT", PositionSide: "SHORT", ManagedQty: 0, Status: "open", SourceRef: "p-new", CreatedAt: 206, UpdatedAt: 206},
		{Owner: "agent_trade", Symbol: "BTCUSDT", PositionSide: "LONG", ManagedQty: 0, Status: "closed", SourceRef: "p-old", CreatedAt: 105, UpdatedAt: 130, ClosedAt: 130},
	}
	for i := range positions {
		if _, err := o.Insert(&positions[i]); err != nil {
			t.Fatal(err)
		}
	}
	order := models.FuturesManagedOrder{Owner: "agent_trade", Symbol: "ETHUSDT", PositionSide: "SHORT", Intent: "stop_loss", ClientOrderID: "client-p-new", RequestedQty: 2, Status: "submitted", SourceRef: "p-new", CreatedAt: 207, UpdatedAt: 207}
	if _, err := o.Insert(&order); err != nil {
		t.Fatal(err)
	}

	got, err := (Service{}).Live(context.Background(), Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Proposals != 2 || got.Executed != 2 || got.OpenPositions != 1 || got.ClosedPositions != 1 || got.ManagedOrders != 1 {
		t.Fatalf("unexpected live summary: %+v", got)
	}
	if len(got.RecentProposals) != 2 || got.RecentProposals[0].ProposalID != "p-new" || got.RecentProposals[1].ProposalID != "p-old" {
		t.Fatalf("unexpected recent proposal order: %+v", got.RecentProposals)
	}

	filtered, err := (Service{}).Live(context.Background(), Filter{Symbol: "ethusdt", Side: "short", MarketCondition: 3})
	if err != nil {
		t.Fatal(err)
	}
	if filtered.Proposals != 1 || filtered.OpenPositions != 1 || filtered.ClosedPositions != 0 || filtered.ManagedOrders != 1 || len(filtered.RecentProposals) != 1 || filtered.RecentProposals[0].ProposalID != "p-new" {
		t.Fatalf("unexpected filtered live summary: %+v", filtered)
	}
}

package opportunity

import (
	"context"
	"sync"
	"testing"
	"time"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
)

var opportunityTestOnce sync.Once
var opportunityTestErr error

func setupOpportunityService(t *testing.T) Service {
	t.Helper()
	opportunityTestOnce.Do(func() {
		if err := orm.RegisterDriver("sqlite3", orm.DRSqlite); err != nil {
			opportunityTestErr = err
			return
		}
		if err := orm.RegisterDataBase("default", "sqlite3", "file:opportunity_service_test?mode=memory&cache=shared"); err != nil {
			opportunityTestErr = err
			return
		}
		db, err := orm.GetDB("default")
		if err != nil {
			opportunityTestErr = err
			return
		}
		// SQLite shared-memory tests use one connection so concurrent Pipeline
		// workers exercise application locking instead of SQLite table locks.
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)
		orm.RegisterModel(new(models.AgentOpportunity))
		opportunityTestErr = orm.RunSyncdb("default", true, false)
	})
	if opportunityTestErr != nil {
		t.Fatal(opportunityTestErr)
	}
	if _, err := orm.NewOrm().Raw("DELETE FROM agent_opportunities").Exec(); err != nil {
		t.Fatal(err)
	}
	fixed := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	return Service{Now: func() time.Time { return fixed }}
}

func TestCreateDeduplicatesSameSource(t *testing.T) {
	service := setupOpportunityService(t)
	ctx := context.Background()
	input := CreateInput{
		Symbol: "btcusdt", Direction: DirectionLong,
		SourceType: "fast_move", SourceID: "sig-1",
		AnalysisStatus: AnalysisSucceeded, AnalysisTaskID: "task-1",
		Summary: "momentum confirmed", Confidence: 0.82,
	}
	first, created, err := service.Create(ctx, input)
	if err != nil || !created {
		t.Fatalf("first create: created=%v row=%+v err=%v", created, first, err)
	}
	second, created, err := service.Create(ctx, input)
	if err != nil || created {
		t.Fatalf("replay create: created=%v row=%+v err=%v", created, second, err)
	}
	if second.OpportunityID != first.OpportunityID {
		t.Fatalf("source replay created another opportunity: first=%s second=%s", first.OpportunityID, second.OpportunityID)
	}
	result, err := service.List(ctx, ListOptions{})
	if err != nil || result.Total != 1 {
		t.Fatalf("expected one durable opportunity, total=%d err=%v", result.Total, err)
	}
}

func TestCooldownUsesSymbolAndSourceType(t *testing.T) {
	service := setupOpportunityService(t)
	ctx := context.Background()
	row, _, err := service.Create(ctx, CreateInput{
		Symbol: "ETHUSDT", Direction: DirectionNeutral,
		SourceType: "fast_move", SourceID: "sig-cooldown",
		AnalysisStatus: AnalysisPending,
	})
	if err != nil {
		t.Fatal(err)
	}
	hit, recent, err := service.InCooldown(ctx, "ethusdt", "fast_move", service.now().Add(-30*time.Minute))
	if err != nil || !hit || recent.OpportunityID != row.OpportunityID {
		t.Fatalf("expected cooldown hit, hit=%v recent=%+v err=%v", hit, recent, err)
	}
	hit, _, err = service.InCooldown(ctx, "ETHUSDT", "market_event", service.now().Add(-30*time.Minute))
	if err != nil || hit {
		t.Fatalf("different source category must not be suppressed: hit=%v err=%v", hit, err)
	}
}

func TestReviewExpireAndProposalEligibility(t *testing.T) {
	service := setupOpportunityService(t)
	ctx := context.Background()
	row, _, err := service.Create(ctx, CreateInput{
		Symbol: "SOLUSDT", Direction: DirectionLong,
		SourceType: "market_scan", SourceID: "scan-1",
		AnalysisStatus: AnalysisSucceeded, AnalysisTaskID: "task-sol",
		Confidence: 0.9, ExpiresAt: service.now().Add(5 * time.Minute).UnixMilli(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !CanCreateProposal(row, service.now()) {
		t.Fatal("fresh succeeded directional opportunity must be proposal-eligible")
	}
	reviewed, err := service.MarkReviewed(ctx, row.OpportunityID)
	if err != nil || reviewed.Status != StatusReviewed || reviewed.ReviewedAt == 0 {
		t.Fatalf("mark reviewed failed: row=%+v err=%v", reviewed, err)
	}
	service.Now = func() time.Time { return time.Date(2026, 9, 13, 12, 10, 0, 0, time.UTC) }
	count, err := service.Expire(ctx)
	if err != nil || count != 1 {
		t.Fatalf("expire count=%d err=%v", count, err)
	}
	expired, err := service.Get(ctx, row.OpportunityID)
	if err != nil || expired.Status != StatusExpired {
		t.Fatalf("expected expired row: %+v err=%v", expired, err)
	}
	if CanCreateProposal(expired, service.now()) {
		t.Fatal("expired opportunity must not be proposal-eligible")
	}
}

func TestUpdateAnalysisAndEligibilityGuard(t *testing.T) {
	service := setupOpportunityService(t)
	ctx := context.Background()
	row, _, err := service.Create(ctx, CreateInput{
		Symbol: "XRPUSDT", Direction: DirectionNeutral,
		SourceType: "market_event", SourceID: "evt-1",
		AnalysisStatus: AnalysisPending,
	})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := service.UpdateAnalysis(ctx, row.OpportunityID, AnalysisUpdate{
		Direction: DirectionShort, TaskID: "task-xrp", Summary: "bearish follow-through",
		Confidence: 0.76, MarketCondition: 2, Status: AnalysisSucceeded,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Direction != DirectionShort || updated.AnalysisTaskID != "task-xrp" || updated.AnalysisStatus != AnalysisSucceeded {
		t.Fatalf("unexpected analysis update: %+v", updated)
	}
	if !CanCreateProposal(updated, service.now()) {
		t.Fatal("succeeded short opportunity should be eligible")
	}
	updated.AnalysisStatus = AnalysisDataMissing
	if CanCreateProposal(updated, service.now()) {
		t.Fatal("data_missing opportunity must not be proposal-eligible")
	}
	updated.AnalysisStatus = AnalysisSucceeded
	updated.Direction = DirectionNeutral
	if CanCreateProposal(updated, service.now()) {
		t.Fatal("neutral opportunity must not be proposal-eligible")
	}
}

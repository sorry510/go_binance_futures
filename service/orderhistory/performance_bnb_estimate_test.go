package orderhistory

import (
	"context"
	"testing"
)

func TestPerformanceOracle1BNBClosedIncompleteShowsEstimatedResults(t *testing.T) {
	store, id, status := makeBNBCycle(t)
	report, err := store.Performance(context.Background(), PerformanceFilter{Days: 30})
	if err != nil {
		t.Fatal(err)
	}
	if report.CandidateCycles != 1 || report.VerifiedCycles != 0 || report.EstimatedCycles != 1 ||
		report.ExcludedCycles != 0 || len(report.Assets) != 0 || len(report.EstimatedAssets) != 1 {
		t.Fatalf("BNB cycle lost or falsely marked as settled: %+v", report)
	}
	asset := report.EstimatedAssets[0]
	if asset.Asset != "USDT" || asset.Trades != 1 || asset.Wins != 1 || asset.NetPnL != "105.2" ||
		asset.GrossPnL != "110" || asset.Commission != "4.8" || asset.WinRatePct != "100" ||
		len(asset.BySymbol) != 1 || asset.BySymbol[0].Name != "BTCUSDT" || len(asset.ByRule) != 1 ||
		report.BNBPriceUSDT != "600" || report.BNBPriceUpdatedAt != 1800000009000 {
		t.Fatalf("estimated BNB performance statistics wrong: %+v report=%+v", asset, report)
	}
	if status() != "incomplete" {
		t.Fatal("performance read modified settlement")
	}
	// Only a display estimate is added. The official verified bucket stays empty.
	if report.Assets == nil {
		t.Fatal("verified asset bucket should be an empty, non-nil slice")
	}
	detail, err := store.Details(context.Background(), "main", id)
	if err != nil || detail.Position.NetPnL != nil || detail.Position.SettlementStatus != "incomplete" {
		t.Fatalf("view-only estimate leaked into certified settlement: %+v %v", detail.Position, err)
	}
}

func TestPerformanceEstimatedBNBUsesCurrentLocalPriceNoT0(t *testing.T) {
	store, _, status := makeBNBCycle(t)
	store.ActivatedAt = func(string) (int64, error) { return 0, nil }
	oldReport, err := store.Performance(context.Background(), PerformanceFilter{Days: 7})
	if err != nil || oldReport.EstimatedCycles != 1 || oldReport.EstimatedAssets[0].NetPnL != "105.2" {
		t.Fatalf("estimate blocked by T0: %+v %v", oldReport, err)
	}
	if _, err = store.db().Raw("UPDATE symbols SET close='650' WHERE symbol='BNBUSDT'").Exec(); err != nil {
		t.Fatal(err)
	}
	updated, err := store.Performance(context.Background(), PerformanceFilter{Days: 7})
	if err != nil || updated.EstimatedCycles != 1 || updated.EstimatedAssets[0].Commission != "5.2" ||
		updated.EstimatedAssets[0].NetPnL != "104.8" || updated.VerifiedCycles != 0 || status() != "incomplete" {
		t.Fatalf("performance did not revalue local current BNB quote: %+v %v", updated, err)
	}
	// A fresh quote is required. Missing price never becomes fake profit.
	if _, err = store.db().Raw("DELETE FROM symbols WHERE symbol='BNBUSDT'").Exec(); err != nil {
		t.Fatal(err)
	}
	noPrice, err := store.Performance(context.Background(), PerformanceFilter{Days: 7})
	if err != nil || noPrice.EstimatedCycles != 0 || noPrice.ExcludedCycles != 1 ||
		len(noPrice.EstimatedAssets) != 0 || noPrice.VerifiedCycles != 0 {
		t.Fatalf("BNB without local quote incorrectly counted: %+v %v", noPrice, err)
	}
}

func TestPerformanceNeverMergesEstimatedWithVerifiedOrBadFills(t *testing.T) {
	store, id, status := makeBNBCycle(t)
	now := store.now().UnixMilli()
	uf5Cycle(t, store.db(), "main", "auto_strategy", "ETHUSDT", "LONG", "USDT", "auto_strategy:ETHUSDT", now-4000, "3")
	report, err := store.Performance(context.Background(), PerformanceFilter{Days: 30})
	if err != nil || report.CandidateCycles != 2 || report.VerifiedCycles != 1 || report.EstimatedCycles != 1 ||
		len(report.Assets) != 1 || len(report.EstimatedAssets) != 1 ||
		report.Assets[0].NetPnL != "2" || report.EstimatedAssets[0].NetPnL != "105.2" {
		t.Fatalf("verified/estimated buckets mixed: %+v %v", report, err)
	}
	// A genuinely unverified Fill must never enter either profit population.
	if _, err = store.db().Raw("UPDATE futures_trade_fills SET rest_verified=0 WHERE position_id=? AND trade_id='4'", id).Exec(); err != nil {
		t.Fatal(err)
	}
	invalid, err := store.Performance(context.Background(), PerformanceFilter{Days: 30})
	if err != nil || invalid.CandidateCycles != 2 || invalid.EstimatedCycles != 0 || invalid.VerifiedCycles != 1 ||
		invalid.ExcludedCycles != 1 || len(invalid.EstimatedAssets) != 0 || status() != "incomplete" {
		t.Fatalf("nonverified BNB fill incorrectly included: %+v %v", invalid, err)
	}
}

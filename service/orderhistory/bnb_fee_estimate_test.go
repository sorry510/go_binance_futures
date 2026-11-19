package orderhistory

import (
	"context"
	"testing"
)

func makeBNBCycle(t *testing.T) (Store, int64, func() string) {
	t.Helper()
	o, store := historyDB(t)
	id := fullCycle(t, o)
	if _, err := o.Raw("UPDATE futures_managed_positions SET settlement_status='incomplete',settlement_processed_at=0,settlement_pnl_sign=0 WHERE id=?", id).Exec(); err != nil {
		t.Fatal(err)
	}
	if _, err := o.Raw("UPDATE futures_trade_fills SET commission_asset='BNB',commission='0.002',pnl_asset='' WHERE position_id=?", id).Exec(); err != nil {
		t.Fatal(err)
	}
	if _, err := o.Raw("INSERT INTO symbols (symbol, close, updateTime, type) VALUES ('BNBUSDT', '600', 1800000009000, 'USDT')").Exec(); err != nil {
		t.Fatal(err)
	}
	readStatus := func() string {
		var status string
		if err := o.Raw("SELECT settlement_status FROM futures_managed_positions WHERE id=?", id).QueryRow(&status); err != nil {
			t.Fatal(err)
		}
		return status
	}
	return store, id, readStatus
}
func TestBNBFeeEstimateClosedCycleWithoutMutatingSettlement(t *testing.T) {
	store, id, status := makeBNBCycle(t)
	page, err := store.List(context.Background(), Filter{AccountID: "main", Page: 1, Limit: 20})
	if err != nil || len(page.List) != 1 {
		t.Fatalf("list %v %+v", err, page)
	}
	row := page.List[0]
	if row.PositionID != id || row.Completeness != "estimated" || !row.BNBFeeEstimated || row.SettlementStatus != "incomplete" ||
		row.EstimatedFeeUSDT == nil || *row.EstimatedFeeUSDT != "4.8" ||
		row.EstimatedNetPnL == nil || *row.EstimatedNetPnL != "105.2" ||
		row.BNBFeePriceUSDT == nil || *row.BNBFeePriceUSDT != "600" ||
		row.RealizedPnL == nil || *row.RealizedPnL != "105.2" ||
		row.TotalPnL == nil || *row.TotalPnL != "105.2" ||
		row.NetPnL != nil || row.GrossPnL != nil {
		t.Fatalf("BNB closed estimated view wrong: %+v", row)
	}
	if status() != "incomplete" {
		t.Fatal("read-only estimate modified settlement")
	}
	if page.PageVerifiedNet["USDT"] != "" {
		t.Fatal("estimated amount leaked into verified PnL")
	}
	details, err := store.Details(context.Background(), "main", id)
	if err != nil || !details.Position.BNBFeeEstimated || len(details.Orders) != 3 {
		t.Fatalf("details %v %+v", err, details.Position)
	}
}
func TestBNBFeeEstimateNeverHidesMissingFillsOrPrices(t *testing.T) {
	store, id, status := makeBNBCycle(t)
	if _, err := store.db().Raw("UPDATE symbols SET close='0' WHERE symbol='BNBUSDT'").Exec(); err != nil {
		t.Fatal(err)
	}
	row, err := store.Details(context.Background(), "main", id)
	if err != nil || row.Position.BNBFeeEstimated || row.Position.Completeness != "incomplete" {
		t.Fatalf("missing quote marked normal: %+v %v", row.Position, err)
	}
	if _, err := store.db().Raw("UPDATE symbols SET close='600' WHERE symbol='BNBUSDT'").Exec(); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db().Raw("UPDATE futures_trade_fills SET rest_verified=0 WHERE position_id=? AND trade_id='4'", id).Exec(); err != nil {
		t.Fatal(err)
	}
	row, err = store.Details(context.Background(), "main", id)
	if err != nil || row.Position.BNBFeeEstimated || row.Position.Completeness != "incomplete" || row.Position.IncompleteReason != "fee_and_pnl_not_rest_verified" {
		t.Fatalf("unverified fill marked normal: %+v %v", row.Position, err)
	}
	if status() != "incomplete" {
		t.Fatal("read-only estimate modified settlement")
	}
}

// An unapproved T0 must still block verified PnL, but not an explicitly
// labelled indicative fee estimate backed by fully REST-confirmed Fills.
func TestBNBFeeEstimateWorksWithoutCertifiedHistoryT0(t *testing.T) {
	store, id, status := makeBNBCycle(t)
	store.ActivatedAt = func(string) (int64, error) { return 0, nil }
	row, err := store.Details(context.Background(), "main", id)
	if err != nil || row.Position.Completeness != "estimated" || !row.Position.BNBFeeEstimated ||
		row.Position.EstimatedNetPnL == nil || *row.Position.EstimatedNetPnL != "105.2" ||
		row.Position.NetPnL != nil || status() != "incomplete" {
		t.Fatalf("T0 incorrectly gates read-only estimate or certifies settlement: %+v %v", row.Position, err)
	}
}

// Changing the local symbols snapshot must immediately revalue historical BNB
// commissions. This intentionally uses CURRENT price, not historical candles.
func TestBNBFeeEstimateUsesCurrentLocalBNBPrice(t *testing.T) {
	store, id, status := makeBNBCycle(t)
	original, err := store.Details(context.Background(), "main", id)
	if err != nil || original.Position.EstimatedFeeUSDT == nil || *original.Position.EstimatedFeeUSDT != "4.8" {
		t.Fatalf("unexpected original quote valuation: %+v %v", original.Position, err)
	}
	if _, err = store.db().Raw("UPDATE symbols SET close='650',updateTime=1800000010000 WHERE symbol='BNBUSDT'").Exec(); err != nil {
		t.Fatal(err)
	}
	updated, err := store.Details(context.Background(), "main", id)
	if err != nil || updated.Position.EstimatedFeeUSDT == nil || *updated.Position.EstimatedFeeUSDT != "5.2" ||
		updated.Position.EstimatedNetPnL == nil || *updated.Position.EstimatedNetPnL != "104.8" ||
		updated.Position.BNBFeePriceUSDT == nil || *updated.Position.BNBFeePriceUSDT != "650" ||
		updated.Position.BNBFeePriceUpdatedAt != 1800000010000 ||
		updated.Position.TotalPnL == nil || *updated.Position.TotalPnL != "104.8" ||
		updated.Position.NetPnL != nil || status() != "incomplete" {
		t.Fatalf("current local BNB quote not applied or settlement mutated: %+v %v", updated.Position, err)
	}
}

func TestBNBFeeEstimateMissingLocalQuoteRemainsIncomplete(t *testing.T) {
	store, id, _ := makeBNBCycle(t)
	if _, err := store.db().Raw("DELETE FROM symbols WHERE symbol='BNBUSDT'").Exec(); err != nil {
		t.Fatal(err)
	}
	result, err := store.Details(context.Background(), "main", id)
	if err != nil || result.Position.BNBFeeEstimated || result.Position.Completeness != "incomplete" {
		t.Fatalf("missing local BNB price must not be fabricated: %+v %v", result.Position, err)
	}
}

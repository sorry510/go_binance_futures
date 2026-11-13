package spotexecution

import (
	"context"
	"errors"
	"testing"
	"time"

	"go_binance_futures/service/spotledger"

	"github.com/shopspring/decimal"
)

// 06-6 runs the 06-4 execution processor against the ACTUAL 06-3 SQL
// ledger and the 06-5 read-model, rather than isolated mock repositories.
// SQLite is temporary and all remote API behavior is a Fake Exchange.
// Futures history, legacy Spot, Binance Mainnet and Testnet are not used.
func TestStage066EndToEndBUYSELLLedgerReadModelAndSecondCycle(t *testing.T) {
	_, ex, _, ledger, db := testEngine(t)
	for _, sql := range []string{
		"ALTER TABLE spot_symbols ADD COLUMN close VARCHAR(64) DEFAULT '100'",
		"ALTER TABLE spot_symbols ADD COLUMN updateTime BIGINT DEFAULT 0",
	} {
		if _, err := db.Raw(sql).Exec(); err != nil {
			t.Fatal(err)
		}
	}
	ts := time.Now().UnixMilli()
	if _, err := db.Raw("UPDATE spot_symbols SET close=?, updateTime=? WHERE symbol='BTCUSDT'", "110", ts).Exec(); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Raw("UPDATE config SET spot_quant_enable=1 WHERE id=1").Exec(); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Raw("UPDATE spot_symbols SET spot_quant_enabled=1 WHERE symbol='BTCUSDT'").Exec(); err != nil {
		t.Fatal(err)
	}
	gate := DBGate{DB: db}
	verified, err := NewProcessor(ex, gate, ledger)
	if err != nil {
		t.Fatal(err)
	}
	read, err := spotledger.NewSpotReadService(db, "sqlite")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	buy, err := verified.ExecuteBuy(ctx, "BTCUSDT", "100", "stage066_e2e_buy1")
	if err != nil {
		t.Fatalf("first isolated BUY: %v", err)
	}
	if buy.Status != "FILLED" || ex.placeCount != 1 {
		t.Fatalf("BUY response: %+v / place=%d", buy, ex.placeCount)
	}
	seen, err := read.Position(buy.PositionID)
	if err != nil {
		t.Fatal(err)
	}
	if !seen.HasMarkPrice || !seen.FullyPriced || seen.Position.Status != "OPEN" {
		t.Fatalf("06-5 read model did not observe 06-4 Fill: %+v", seen)
	}
	sell, err := verified.ExecuteSell(ctx, buy.PositionID, "stage066_e2e_sell1")
	if err != nil {
		t.Fatalf("full inventory SELL: %v", err)
	}
	if sell.Status != "FILLED" || sell.RequestedQty != buy.RequestedQty {
		t.Fatalf("SELL did not close verified inventory: %+v vs %+v", sell, buy)
	}
	closed, err := read.Position(buy.PositionID)
	if err != nil {
		t.Fatal(err)
	}
	if closed.Position.Status != "CLOSED" || !decimal.RequireFromString(closed.Position.RemainingQty).IsZero() {
		t.Fatalf("settled read model failed to CLOSE: %+v", closed)
	}
	if !closed.FullyPriced || closed.TotalPnL == "" {
		t.Fatalf("full no-fee lifecycle lacks verifiable net Spot PnL: %+v", closed)
	}
	summary, err := read.Positions(spotledger.PositionFilter{Symbol: "BTCUSDT", Page: 1, Limit: 1})
	if err != nil || summary.Total != 1 || len(summary.Summary) != 1 {
		t.Fatalf("06-5 cross-page summary: %+v %v", summary, err)
	}
	orders, err := read.Orders(spotledger.OrderFilter{PositionID: buy.PositionID, Page: 1, Limit: 10})
	if err != nil || orders.Total != 2 || len(orders.List) != 2 {
		t.Fatalf("06-5 order list does not match BUY/SELL: %+v %v", orders, err)
	}
	fills, err := ledger.ListFills(buy.PositionID)
	if err != nil || len(fills) != 2 {
		t.Fatalf("missing immutable BUY/SELL fills: %d %v", len(fills), err)
	}
	if ex.placeCount != 2 {
		t.Fatalf("unexpected external POST count: %d", ex.placeCount)
	}
	secondBuy, err := verified.ExecuteBuy(ctx, "BTCUSDT", "100", "stage066_e2e_buy2")
	if err != nil || secondBuy.PositionID == buy.PositionID || ex.placeCount != 3 {
		t.Fatalf("new Spot cycle not independent: %+v %v count=%d", secondBuy, err, ex.placeCount)
	}
	result, err := read.Positions(spotledger.PositionFilter{Symbol: "BTCUSDT", Page: 1, Limit: 1})
	if err != nil || result.Total != 2 || len(result.List) != 1 || result.Summary[0].Positions != 2 {
		t.Fatalf("cross-page count/summary mismatch: %+v %v", result, err)
	}
	var sentinel string
	if err := db.Raw("SELECT marker FROM futures_managed_positions WHERE id=1").QueryRow(&sentinel); err != nil || sentinel != "unchanged" {
		t.Fatalf("Futures test sentinel changed: %s / %v", sentinel, err)
	}
}

// Restore a prior uncertain exchange response without resubmitting a single
// order. Disabling either switch stops new orders, NOT read-only restoration.
func TestStage066RestartRecoveryGateAndNoDoublePOST(t *testing.T) {
	p, ex, _, ledger, db := testEngine(t)
	ctx := context.Background()
	ex.placeReturnsError = true
	ex.placeErrorAfterAccept = true
	_, err := p.ExecuteBuy(ctx, "BTCUSDT", "100", "stage066_timeout_buy")
	if err != nil {
		t.Fatalf("accepted-but-timeout Fake order was not recovered: %v", err)
	}
	if ex.placeCount != 1 {
		t.Fatal("timeout caused duplicate order POST")
	}
	g := DBGate{DB: db}
	if _, err := db.Raw("UPDATE config SET spot_quant_enable=1 WHERE id=1").Exec(); err != nil {
		t.Fatal(err)
	}
	verified, err := NewProcessor(ex, g, ledger)
	if err != nil {
		t.Fatal(err)
	}
	// Symbol switch remains OFF and is verified independently.
	if _, err := verified.ExecuteBuy(ctx, "BTCUSDT", "100", "stage066_next_buy"); !errors.Is(err, ErrDisabled) {
		t.Fatalf("single-symbol switch failed to block a new BUY: %v", err)
	}
	if err := verified.RecoverOutstanding(ctx); err != nil {
		t.Fatalf("read-only recovery must work while symbol switch OFF: %v", err)
	}
	if ex.placeCount != 1 {
		t.Fatal("recovery placed second order")
	}
	if _, err := db.Raw("UPDATE spot_symbols SET spot_quant_enabled=1 WHERE symbol='BTCUSDT'").Exec(); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Raw("UPDATE config SET spot_quant_enable=0 WHERE id=1").Exec(); err != nil {
		t.Fatal(err)
	}
	if _, err := verified.ExecuteBuy(ctx, "BTCUSDT", "100", "stage066_next_buy"); !errors.Is(err, ErrDisabled) {
		t.Fatalf("global switch failed to block new BUY: %v", err)
	}
	if err := verified.RecoverOutstanding(ctx); err != nil {
		t.Fatalf("read-only recovery failed with global OFF: %v", err)
	}
	if ex.placeCount != 1 {
		t.Fatalf("global OFF recovery sent external orders: %d", ex.placeCount)
	}
	loaded, err := ledger.FindOrderByClientID("stage066_timeout_buy")
	if err != nil || loaded.Status != "FILLED" {
		t.Fatalf("lost persistent client_id reconciliation: %+v %v", loaded, err)
	}
	if _, err := verified.Reconcile(ctx, loaded.ID); err != nil {
		t.Fatalf("explicit REST recovery while global OFF: %v", err)
	}
	if ex.placeCount != 1 {
		t.Fatalf("read-only Reconcile sent a POST: %d", ex.placeCount)
	}
	if _, err := ledger.GetPosition(loaded.PositionID); err != nil {
		t.Fatalf("durable position missing: %v", err)
	}
}

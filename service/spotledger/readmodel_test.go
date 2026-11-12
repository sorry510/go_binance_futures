package spotledger

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"github.com/shopspring/decimal"
)

func readModelFixture(t *testing.T) (*Ledger, *SpotReadService, orm.Ormer) {
	t.Helper()
	l, alias := fixture(t)
	db := orm.NewOrmUsingDB(alias)
	if _, err := db.Raw("CREATE TABLE spot_symbols (symbol VARCHAR(32) PRIMARY KEY,close VARCHAR(64) NOT NULL,updateTime BIGINT NOT NULL)").Exec(); err != nil {
		t.Fatal(err)
	}
	read, err := NewSpotReadService(db, "sqlite")
	if err != nil {
		t.Fatal(err)
	}
	return l, read, db
}
func TestStage065ReadModelPagedSummaryAndExactAccounting(t *testing.T) {
	l, read, db := readModelFixture(t)
	ts := time.Now().UnixMilli()
	for _, symbol := range []string{"BTCUSDT", "ETHUSDC"} {
		if _, err := db.Raw("INSERT INTO spot_symbols (symbol,close,updateTime) VALUES(?,?,?)", symbol, "110", ts).Exec(); err != nil {
			t.Fatal(err)
		}
	}
	p, err := l.CreatePosition("BTCUSDT", "BTC", "USDT")
	if err != nil {
		t.Fatal(err)
	}
	buyID := createTestOrder(t, l, p.ID, "BUY", "view-buy", "1")
	if _, err := l.RecordFill(event(buyID, "50001", "1", "BUY", "1", "100", "100", "0", "", ts-2000)); err != nil {
		t.Fatal(err)
	}
	sellID := createTestOrder(t, l, p.ID, "SELL", "view-sell", "0.4")
	if _, err := l.RecordFill(event(sellID, "50002", "2", "SELL", "0.4", "120", "48", "0", "", ts-1000)); err != nil {
		t.Fatal(err)
	}
	second, err := l.CreatePosition("ETHUSDC", "ETH", "USDC")
	if err != nil {
		t.Fatal(err)
	}
	buy2, duplicate, err := l.CreateOrder(OrderIntent{PositionID: second.ID, Symbol: "ETHUSDC", Side: "BUY", Type: "MARKET", ClientOrderID: "view-eth-buy", RequestedQty: "2"})
	if err != nil || duplicate {
		t.Fatalf("ETH order: %+v %v", buy2, err)
	}
	if _, err = l.RecordFill(FillEvent{OrderID: buy2.ID, Symbol: "ETHUSDC", ExchangeOrderID: "50003", TradeID: "3", Side: "BUY", Qty: "2", Price: "100", QuoteQty: "200", Commission: "0", TradeTime: ts}); err != nil {
		t.Fatal(err)
	}
	// Pagination restricts rows, but summary aggregates all matching
	// positions and keeps different quote assets as different units.
	got, err := read.Positions(PositionFilter{Page: 1, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if got.Total != 2 || len(got.List) != 1 || len(got.Summary) != 2 {
		t.Fatalf("page-scoped summary: %+v", got)
	}
	if got.Summary[0].QuoteAsset != "USDT" || got.Summary[1].QuoteAsset != "USDC" {
		t.Fatalf("quote currencies combined: %+v", got.Summary)
	}
	if !got.Summary[0].FullyPriced || !got.Summary[1].FullyPriced {
		t.Fatalf("valid quotes not priced: %+v", got.Summary)
	}
	decimalClose(t, got.Summary[0].TotalPnL, "14")
	decimalClose(t, got.Summary[1].TotalPnL, "20")
	if got.Summary[0].RealizedPnL != "8.000000000000000000" {
		t.Fatalf("wrong verified realized quote pnl %s", got.Summary[0].RealizedPnL)
	}
	filtered, err := read.Positions(PositionFilter{Symbol: "BTC", SymbolLike: true, Page: 1, Limit: 20})
	if err != nil || filtered.Total != 1 || len(filtered.Summary) != 1 || filtered.Summary[0].QuoteAsset != "USDT" {
		t.Fatalf("fuzzy symbol search: %+v %v", filtered, err)
	}
	orders, err := read.Orders(OrderFilter{Symbol: "btc", Page: 1, Limit: 1})
	if err != nil || orders.Total != 2 || len(orders.List) != 1 {
		t.Fatalf("order filter/limit: %+v %v", orders, err)
	}
	if _, err = read.Orders(OrderFilter{Status: "SHORT"}); err == nil {
		t.Fatal("SHORT status accepted")
	}
	if _, err = read.Positions(PositionFilter{Symbol: "BTC%"}); err == nil {
		t.Fatal("wildcard symbol accepted")
	}
}

func TestStage065ReadModelStalePriceAndUnpricedFees(t *testing.T) {
	l, read, db := readModelFixture(t)
	now := time.Now().UnixMilli()
	_, err := db.Raw("INSERT INTO spot_symbols(symbol,close,updateTime) VALUES(?,?,?)", "BTCUSDT", "110", now-6*60*1000).Exec()
	if err != nil {
		t.Fatal(err)
	}
	p, err := l.CreatePosition("BTCUSDT", "BTC", "USDT")
	if err != nil {
		t.Fatal(err)
	}
	id := createTestOrder(t, l, p.ID, "BUY", "stale-fee-buy", "1")
	if _, err = l.RecordFill(event(id, "5101", "1", "BUY", "1", "100", "100", "0.001", "BNB", now)); err != nil {
		t.Fatal(err)
	}
	view, err := read.Position(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if view.HasMarkPrice || view.TotalPnL != "" || view.FullyPriced || view.UnpricedFees["BNB"] == "" {
		t.Fatalf("stale/third-fee was represented as complete PnL: %+v", view)
	}
	all, err := read.Positions(PositionFilter{Page: 1, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(all.Summary) != 1 || all.Summary[0].FullyPriced || all.Summary[0].MissingPrices != 1 || all.Summary[0].UnpricedPositions != 1 || all.Summary[0].TotalPnL != "" {
		t.Fatalf("incomplete PnL aggregate falsely completed: %+v", all.Summary)
	}
	if _, err = db.Raw("UPDATE spot_symbols SET close='110',updateTime=? WHERE symbol='BTCUSDT'", now).Exec(); err != nil {
		t.Fatal(err)
	}
	fresh, err := read.Position(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !fresh.HasMarkPrice || fresh.FullyPriced || fresh.TotalPnL != "" {
		t.Fatalf("fresh price should not hide unpriced BNB fee: %+v", fresh)
	}
}
func TestStage065ReadModelErrorRequiresReviewWithoutMutableProfit(t *testing.T) {
	l, read, db := readModelFixture(t)
	ts := time.Now().UnixMilli()
	if _, err := db.Raw("INSERT INTO spot_symbols(symbol,close,updateTime) VALUES(?,?,?)", "BTCUSDT", "100", ts).Exec(); err != nil {
		t.Fatal(err)
	}
	p, err := l.CreatePosition("BTCUSDT", "BTC", "USDT")
	if err != nil {
		t.Fatal(err)
	}
	id := createTestOrder(t, l, p.ID, "BUY", "review-view-buy", "1")
	if _, err = l.SetExchangeStatus(id, "FILLED"); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Raw("UPDATE spot_managed_positions SET remaining_qty=999,realized_pnl=999 WHERE id=?", p.ID).Exec(); err != nil {
		t.Fatal(err)
	}
	view, err := read.Position(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !view.NeedsReview || view.RealizedPnL != "" || view.TotalPnL != "" || view.HasMarkPrice || view.Position.RemainingQty != "" || view.Position.RealizedPnL != "" || !strings.Contains(view.ReviewReason, "reconciliation") {
		t.Fatalf("unreconciled fake PnL exposed as real: %+v", view)
	}
	page, err := read.Positions(PositionFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if page.Summary[0].ReviewPositions != 1 || page.Summary[0].FullyPriced || page.Summary[0].TotalPnL != "" {
		t.Fatalf("unverified amounts added to totals: %+v", page.Summary)
	}
	// All monetary accumulation is exact DECIMAL, never JS/Go float.
	if _, err := decimal.NewFromString(page.Summary[0].RealizedPnL); err != nil {
		t.Fatal(err)
	}
}

// F1 regression: the 06-4 order recovery depends on an exact symbol. Only
// the UI read-only list may explicitly request substring matching.
func TestReview065ExactSymbolRecoveryNoCrossCoin(t *testing.T) {
	l, read, _ := readModelFixture(t)
	for _, p := range []struct{ sym, base string }{
		{"BTCUSDT", "BTC"}, {"WBTCUSDT", "WBTC"}, {"ETHUSDT", "ETH"},
	} {
		if _, err := l.CreatePosition(p.sym, p.base, "USDT"); err != nil {
			t.Fatal(err)
		}
	}
	exact, err := l.ListPositions(PositionFilter{Symbol: "BTCUSDT", Page: 1, Limit: 100})
	if err != nil || exact.Total != 1 || len(exact.List) != 1 || exact.List[0].Symbol != "BTCUSDT" {
		t.Fatalf("execution/recovery exact BTCUSDT spilled to WBTCUSDT: %+v err=%v", exact, err)
	}
	partial, err := read.Positions(PositionFilter{Symbol: "BTC", SymbolLike: true, Page: 1, Limit: 20})
	if err != nil || partial.Total != 2 {
		t.Fatalf("UI opt-in partial lookup should match BTC+WBTC: %+v err=%v", partial, err)
	}
	legacy, err := l.ListPositions(PositionFilter{Symbol: "btcusdt", Page: 1, Limit: 20})
	if err != nil || legacy.Total != 1 || legacy.List[0].Symbol != "BTCUSDT" {
		t.Fatalf("case-normalized exact identifier regression: %+v err=%v", legacy, err)
	}
}

// F2 regression: CLOSED plus unavailable Spot price and a BNB commission
// cannot publish net total_pnl. The third-asset fee remains unpriced.
func TestReview065ClosedUnpricedFeeNeverPublishesTotalPnL(t *testing.T) {
	l, read, _ := readModelFixture(t)
	p, err := l.CreatePosition("BTCUSDT", "BTC", "USDT")
	if err != nil {
		t.Fatal(err)
	}
	buy := createTestOrder(t, l, p.ID, "BUY", "review065_closed_buy", "1")
	if _, err = l.RecordFill(event(buy, "61001", "11", "BUY", "1", "100", "100", "0.001", "BNB", 1000)); err != nil {
		t.Fatal(err)
	}
	sell := createTestOrder(t, l, p.ID, "SELL", "review065_closed_sell", "1")
	if _, err = l.RecordFill(event(sell, "61002", "12", "SELL", "1", "110", "110", "0", "", 2000)); err != nil {
		t.Fatal(err)
	}
	view, err := read.Position(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if view.Position.Status != "CLOSED" || view.HasMarkPrice || view.FullyPriced ||
		view.TotalPnL != "" || view.UnrealizedPnL == "" || view.UnpricedFees["BNB"] == "" {
		t.Fatalf("CLOSED third-asset commission exposed incomplete net PnL: %+v", view)
	}
	paged, err := read.Positions(PositionFilter{Page: 1, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(paged.Summary) != 1 || paged.Summary[0].FullyPriced || paged.Summary[0].TotalPnL != "" {
		t.Fatalf("cross-page summary exposed incomplete net PnL: %+v", paged.Summary)
	}
}

func TestReview065SummarySizeFailClosedBeforeReplayingThousandsOfFills(t *testing.T) {
	_, read, db := readModelFixture(t)
	for i := 0; i < 11; i++ {
		values := make([]string, 0, 100)
		for k := 0; k < 100; k++ {
			values = append(values, "('BTCUSDT','BTC','USDT','PENDING_BUY',1,1)")
		}
		sql := "INSERT INTO spot_managed_positions (symbol,base_asset,quote_asset,status,created_at,updated_at) VALUES " + strings.Join(values, ",")
		if _, err := db.Raw(sql).Exec(); err != nil {
			t.Fatal(err)
		}
	}
	page, err := read.Positions(PositionFilter{Limit: 20})
	if !errors.Is(err, ErrSummaryTooLarge) || page.Total != 0 || len(page.List) != 0 {
		t.Fatalf("too many Spot positions returned partial misleading total: %+v %v", page, err)
	}
}

package orderhistory

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func uf6AssertSameSummary(t *testing.T, store Store, q Filter) {
	t.Helper()
	legacy, err := store.List(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	only, err := store.ListPageOnly(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	summary, err := store.Summary(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	if !legacy.SummaryIncluded || only.SummaryIncluded ||
		only.Total != legacy.Total || only.Page != legacy.Page || only.Limit != legacy.Limit ||
		!reflect.DeepEqual(only.List, legacy.List) ||
		!reflect.DeepEqual(only.PageVerifiedNet, legacy.PageVerifiedNet) ||
		!reflect.DeepEqual(summary.FilteredPnL, legacy.FilteredPnL) ||
		summary.PNLUnpriced != legacy.PNLUnpriced ||
		summary.Total != legacy.Total ||
		summary.CalculatedAt <= 0 {
		t.Fatalf("UF6 must preserve every result/NULL/asset, only skip full scan on page: q=%+v\nlegacy=%+v\npage=%+v\nsummary=%+v", q, legacy, only, summary)
	}
	if len(only.FilteredPnL) != 0 || only.PNLUnpriced != 0 {
		t.Fatalf("page-only accidentally recomputed complete filter: %+v", only)
	}
}
func TestUF6LegacyListAndSeparatedSummaryExactAllFilters(t *testing.T) {
	o, store := historyDB(t)
	fullCycle(t, o)
	active := positionFixture(t, o, "main", "auto_strategy", "BTCUSDT", "SHORT", "active", "open", false)
	if _, err := o.Raw("UPDATE futures_managed_positions SET managed_qty=1 WHERE id=?", active).Exec(); err != nil {
		t.Fatal(err)
	}
	managedOrder(t, o, active, "main", "auto_strategy", "BTCUSDT", "SHORT", "uf6-short-open", "open", "1099", 1)
	fill(t, o, "main", "BTCUSDT", "SHORT", "1099", "uf6-1", "uf6-short-open", "open", "1", "100", "0.01", "0", "USDT", active, false)
	unknown := positionFixture(t, o, "main", "manual_claim", "XRPUSDT", "LONG", "active", "open", false)
	if _, err := o.Raw("UPDATE futures_managed_positions SET managed_qty=1 WHERE id=?", unknown).Exec(); err != nil {
		t.Fatal(err)
	}
	usdc := positionFixture(t, o, "main", "notice_auto_order", "SOLUSDC", "LONG", "active", "open", false)
	if _, err := o.Raw("UPDATE futures_managed_positions SET managed_qty=2,entry_price=50 WHERE id=?", usdc).Exec(); err != nil {
		t.Fatal(err)
	}
	lead := positionFixture(t, o, "lead", "auto_strategy", "BTCUSDT", "LONG", "active", "open", false)
	if _, err := o.Raw("UPDATE futures_managed_positions SET managed_qty=3 WHERE id=?", lead).Exec(); err != nil {
		t.Fatal(err)
	}
	for _, quote := range []struct{ symbol, close, asset string }{
		{"BTCUSDT", "110", "USDT"}, {"SOLUSDC", "60", "USDC"},
	} {
		if _, err := o.Raw("INSERT INTO symbols (symbol,close,updateTime,type) VALUES (?,?,?,?)", quote.symbol, quote.close, int64(1800000008000), quote.asset).Exec(); err != nil {
			t.Fatal(err)
		}
	}
	filters := []Filter{
		{AccountID: "main", Page: 1, Limit: 1},
		{AccountID: "main", Page: 2, Limit: 2},
		{AccountID: "main", Page: 999, Limit: 3},
		{AccountID: "main", Page: 1, Limit: 20},
		{AccountID: "main", Owner: "auto_strategy", Page: 1, Limit: 2},
		{AccountID: "main", Owner: "notice_auto_order", Page: 1, Limit: 5},
		{AccountID: "main", Symbol: "BTC", Page: 1, Limit: 2},
		{AccountID: "main", Symbol: "SOLUSDC", Page: 1, Limit: 2},
		{AccountID: "main", PositionSide: "SHORT", Page: 1, Limit: 2},
		{AccountID: "main", Status: "closed", Page: 1, Limit: 2},
		{AccountID: "main", Settlement: "open", Page: 1, Limit: 1},
		{AccountID: "main", StartTime: 1800000000000, EndTime: 1800000006000, Page: 1, Limit: 2},
		{AccountID: "main", StartTime: 1800000010000, EndTime: 1800000090000, Page: 1, Limit: 2},
		{AccountID: "lead", Page: 1, Limit: 20},
	}
	for _, q := range filters {
		uf6AssertSameSummary(t, store, q)
	}
	summary, err := store.Summary(context.Background(), Filter{AccountID: "main"})
	if err != nil || summary.FilteredPnL["USDT"].Realized != "110" ||
		summary.FilteredPnL["USDT"].Unrealized != "-10" ||
		summary.FilteredPnL["USDT"].Total != "100" ||
		summary.FilteredPnL["USDC"].Unrealized != "20" ||
		summary.PNLUnpriced != 1 || summary.PriceUpdatedAt != 1800000008000 {
		t.Fatalf("cross-asset/unpriced and quote snapshot changed: %+v %v", summary, err)
	}
}
func TestUF6ManualOverrideAndIncompleteFillNotCertifiedBySummary(t *testing.T) {
	o, store := historyDB(t)
	id := fullCycle(t, o)
	if _, err := o.Raw("UPDATE futures_managed_positions SET history_status_override='active',history_completeness_override='manual_reviewed' WHERE id=?", id).Exec(); err != nil {
		t.Fatal(err)
	}
	if _, err := o.Raw("UPDATE futures_trade_fills SET rest_verified=0 WHERE trade_id='4' AND position_id=?", id).Exec(); err != nil {
		t.Fatal(err)
	}
	if _, err := o.Raw("INSERT INTO symbols (symbol,close,updateTime,type) VALUES (?,?,?,?)", "BTCUSDT", "140", int64(1800000008000), "USDT").Exec(); err != nil {
		t.Fatal(err)
	}
	uf6AssertSameSummary(t, store, Filter{Page: 1, Limit: 1})
	legacy, err := store.List(context.Background(), Filter{Page: 1, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if legacy.List[0].NetPnL != nil || legacy.List[0].SystemCompleteness != "incomplete" {
		t.Fatalf("manual override cannot certify REST fill: %+v", legacy.List[0])
	}
	// A late REST Fill update must be observable at the next explicit
	// summary refresh. The server does not cache financial results.
	before, err := store.Summary(context.Background(), Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = o.Raw("UPDATE futures_trade_fills SET price='150' WHERE trade_id='4' AND position_id=?", id).Exec(); err != nil {
		t.Fatal(err)
	}
	after, err := store.Summary(context.Background(), Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(before.FilteredPnL, after.FilteredPnL) {
		t.Fatalf("fresh Fill/price update did not invalidate server summary: %v -> %v", before, after)
	}
	uf6AssertSameSummary(t, store, Filter{Page: 1, Limit: 1})
}
func TestUF6InvalidFiltersAndCanceledRequestsNeverReturnValidEmptySummary(t *testing.T) {
	_, store := historyDB(t)
	for _, q := range []Filter{
		{AccountID: "unexpected", Page: 1, Limit: 20},
		{Owner: "'; DROP TABLE test", Page: 1, Limit: 20},
		{Symbol: "BTC%';", Page: 1, Limit: 20},
		{StartTime: 42, EndTime: 41, Page: 1, Limit: 20},
		{PositionSide: "BOTH", Page: 1, Limit: 20},
	} {
		_, err := store.Summary(context.Background(), q)
		if !errors.Is(err, ErrInvalidQuery) {
			t.Fatalf("unsafe summary filter accepted: %+v %v", q, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := store.ListPageOnly(ctx, Filter{Page: 1, Limit: 20})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled page returned fake empty page: %v", err)
	}
	_, err = store.Summary(ctx, Filter{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled summary returned fake zero PnL: %v", err)
	}
}
func TestUF6EmptyAndOutOfRangePagesKeepAllFilterTotals(t *testing.T) {
	o, store := historyDB(t)
	_, err := store.Summary(context.Background(), Filter{})
	if err != nil {
		t.Fatal(err)
	}
	empty, err := store.ListPageOnly(context.Background(), Filter{Page: 2, Limit: 10})
	if err != nil || empty.Total != 0 || len(empty.List) != 0 || empty.SummaryIncluded {
		t.Fatalf("unexpected empty fast page: %+v %v", empty, err)
	}
	fullCycle(t, o)
	uf6AssertSameSummary(t, store, Filter{Page: 5000, Limit: 1})
}

// Cross-check the exposed generated-at metadata is a calculation time and
// doesn't invent a Binance mark/quote timestamp where no quote exists.
func TestUF6CancellationMidSummaryNeverReturnsPartialFinancialTotals(t *testing.T) {
	_, store := historyDB(t)
	uf6PopulateBenchmark(t, store, 250) // three 100-position projection batches
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	clockReads := 0
	store.Now = func() time.Time {
		clockReads++
		if clockReads == 3 {
			cancel()
		}
		return time.UnixMilli(1800000010000).UTC()
	}
	got, err := store.Summary(ctx, Filter{})
	if !errors.Is(err, context.Canceled) ||
		got.Total != 0 || len(got.FilteredPnL) != 0 || got.PNLUnpriced != 0 {
		t.Fatalf("cancellation mid-batch must fail closed, never return partial PnL: %+v %v", got, err)
	}
	if clockReads < 3 {
		t.Fatalf("did not reach an active projection batch: %d", clockReads)
	}
}

func TestUF6NoQuotePriceTimestampIsZeroAndGenerationIsLocal(t *testing.T) {
	o, store := historyDB(t)
	fullCycle(t, o)
	store.Now = func() time.Time { return time.UnixMilli(1800000015000).UTC() }
	summary, err := store.Summary(context.Background(), Filter{})
	if err != nil || summary.CalculatedAt != 1800000015000 || summary.PriceUpdatedAt != 0 {
		t.Fatalf("cannot claim local price is fresh when unavailable: %+v %v", summary, err)
	}
}

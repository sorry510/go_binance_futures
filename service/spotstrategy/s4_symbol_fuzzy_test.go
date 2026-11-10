package spotstrategy

import (
	"context"
	"errors"
	"testing"

	"github.com/beego/beego/v2/client/orm"
	"go_binance_futures/models"
)

func TestS4SymbolListFuzzySearchBeforePagination(t *testing.T) {
	setupSpotS1(t)
	o := orm.NewOrm()
	for _, coin := range []models.SpotSymbols{
		{Symbol: "WBTCUSDT", Type: "USDT", Strategy: "[]", Technology: "{}", StrategyBudgetQuote: "0"},
		{Symbol: "ETHUSDT", Type: "USDT", Strategy: "[]", Technology: "{}", StrategyBudgetQuote: "0"},
		{Symbol: "BTCUSDC", Type: "USDC", Strategy: "[]", Technology: "{}", StrategyBudgetQuote: "0"},
		{Symbol: "SOLFDUSD", Type: "FDUSD", Strategy: "[]", Technology: "{}", StrategyBudgetQuote: "0"},
	} {
		if _, err := o.Insert(&coin); err != nil {
			t.Fatal(err)
		}
	}
	db, err := orm.GetDB("default")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	cases := []struct {
		query, quote, status string
		total                int64
	}{
		{"BTC", "", "", 3},
		{"btc", "", "", 3},
		{"btc ", "", "", 3},
		{" Btc ", "", "", 3},
		{"B", "", "", 3},
		{"BTC", "USDT", "", 2},
		{"BTC", "USDC", "", 1},
		{"USDT", "", "", 3},
		{"BTC", "", "enabled", 0},
		{"BTC", "", "disabled", 3},
		{"XRP", "", "", 0},
	}
	for _, tc := range cases {
		got, err := ListSettings(ctx, db, PageRequest{Symbol: tc.query, QuoteAsset: tc.quote, Status: tc.status, Page: 1, Limit: 1})
		if err != nil || got.Total != tc.total || len(got.Items) != min(int(tc.total), 1) {
			t.Fatalf("query=%q quote=%q status=%q total=%d got=%+v err=%v",
				tc.query, tc.quote, tc.status, tc.total, got, err)
		}
		for _, row := range got.Items {
			if tc.quote != "" && row.QuoteAsset != tc.quote {
				t.Fatalf("quote mismatch: %+v", row)
			}
		}
		if tc.total > 1 {
			second, err := ListSettings(ctx, db, PageRequest{Symbol: tc.query, QuoteAsset: tc.quote, Status: tc.status, Page: 2, Limit: 1})
			if err != nil || second.Total != tc.total || len(second.Items) != 1 || second.Items[0].Symbol == got.Items[0].Symbol {
				t.Fatalf("bad page2 query=%q first=%+v second=%+v err=%v", tc.query, got, second, err)
			}
		}
	}
	for _, invalid := range []string{"B%", "B_", "BTC' OR 1=1", "B/C", "B;DROP", "*"} {
		if _, err := ListSettings(ctx, db, PageRequest{Symbol: invalid}); !errors.Is(err, ErrInvalidS4Filter) {
			t.Fatalf("unsafe symbol fragment %q accepted: %v", invalid, err)
		}
	}
	// A short fuzzy list filter does not turn Holding/order queries into LIKE:
	// those paths still do exact-symbol comparisons.
	owned, err := ListHoldings(ctx, db, PageRequest{Symbol: "BTC"})
	if err != nil || owned.Total != 0 {
		t.Fatalf("short Holding filter unexpectedly fuzzy: %+v %v", owned, err)
	}
}

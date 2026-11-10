package spotstrategy

import (
	"context"
	"errors"
	"testing"

	"github.com/beego/beego/v2/client/orm"
	"go_binance_futures/models"
)

func TestS4ConfigQuoteFilterCountsBeforePagination(t *testing.T) {
	setupSpotS1(t)
	o := orm.NewOrm()
	for _, item := range []models.SpotSymbols{
		{Symbol: "ETHUSDT", Type: "USDT", Strategy: "[]", Technology: "{}", StrategyBudgetQuote: "0"},
		{Symbol: "BNBUSDT", Type: "USDT", Strategy: "[]", Technology: "{}", StrategyBudgetQuote: "0"},
		{Symbol: "ETHUSDC", Type: "USDC", Strategy: "[]", Technology: "{}", StrategyBudgetQuote: "0"},
		{Symbol: "XRPUSDC", Type: "USDC", Strategy: "[]", Technology: "{}", StrategyBudgetQuote: "0"},
		{Symbol: "SOLFDUSD", Type: "FDUSD", Strategy: "[]", Technology: "{}", StrategyBudgetQuote: "0"},
		{Symbol: "STALEBUSD", Type: "BUSD", Strategy: "[]", Technology: "{}", StrategyBudgetQuote: "0"},
		{Symbol: "FAKEUSDC", Type: "USDT", Strategy: "[]", Technology: "{}", StrategyBudgetQuote: "0"},
	} {
		if _, err := o.Insert(&item); err != nil {
			t.Fatal(err)
		}
	}
	db, err := orm.GetDB("default")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, tc := range []struct {
		quote string
		total int64
	}{
		{"", 6}, {"USDT", 3}, {"USDC", 2}, {"FDUSD", 1},
	} {
		first, err := ListSettings(ctx, db, PageRequest{QuoteAsset: tc.quote, Limit: 1, Page: 1})
		if err != nil || first.Total != tc.total || len(first.Items) != 1 {
			t.Fatalf("quote=%q got %+v err=%v", tc.quote, first, err)
		}
		if tc.quote != "" && first.Items[0].QuoteAsset != tc.quote {
			t.Fatalf("wrong-quote row: %+v", first.Items)
		}
		if tc.total > 1 {
			second, err := ListSettings(ctx, db, PageRequest{QuoteAsset: tc.quote, Limit: 1, Page: 2})
			if err != nil || second.Total != tc.total || len(second.Items) != 1 || second.Items[0].Symbol == first.Items[0].Symbol {
				t.Fatalf("bad per-quote pagination first=%+v second=%+v err=%v", first, second, err)
			}
			if tc.quote != "" && second.Items[0].QuoteAsset != tc.quote {
				t.Fatal("cross quote leakage")
			}
		}
	}
	for _, quote := range []string{"usd", "BUSD", "USDT OR 1=1", "USDT ", "BTC", "1"} {
		_, err := ListSettings(ctx, db, PageRequest{QuoteAsset: quote})
		if !errors.Is(err, ErrInvalidS4Filter) {
			t.Fatalf("invalid quote %q accepted: %v", quote, err)
		}
	}
}

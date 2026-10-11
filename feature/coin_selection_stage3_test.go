package feature

import (
	"fmt"
	"testing"
	"time"

	"go_binance_futures/feature/api/binance"
	"go_binance_futures/models"
	"go_binance_futures/scanner"
)

func TestStage3LeadWhitelistAppliedBeforeTop60Ranking(t *testing.T) {
	now := time.Now().UnixMilli()
	all := make([]*models.Symbols, 0, 61)
	for i := 0; i < 61; i++ {
		all = append(all, &models.Symbols{
			Symbol: fmt.Sprintf("X%02dUSDT", i), Type: "USDT", Enable: 1,
			UpdateTime: now, LastUpdateTime: now - 1000, LastClose: "1.07",
			Close: "1.08", Open: "1.00", Low: "0.98", High: "1.10",
			PercentChange: 8, QuoteVolume: 120_000_000, TradeCount: 220_000,
		})
	}
	opts := scanner.SmartLocalV2Options{Limit: scanner.DefaultSmartLocalV2PoolSize}
	main := scanner.SmartLocalV2FromSymbols(all, nil, opts)
	if len(main.Candidates) != 60 {
		t.Fatalf("main candidates=%d want 60", len(main.Candidates))
	}
	for _, item := range main.Candidates {
		if item.Symbol == "X60USDT" {
			t.Fatal("fixture: X60 unexpectedly inside Main's top 60")
		}
	}
	allowed := map[string]binance.LeadTradingSymbol{
		"X60USDT": {Symbol: "X60USDT", QuoteAsset: "USDT"},
	}
	leadUniverse := leadEligibleUniverse(all, allowed)
	lead := scanner.SmartLocalV2FromSymbols(leadUniverse, nil, opts)
	if len(lead.Candidates) != 1 || lead.Candidates[0].Symbol != "X60USDT" {
		t.Fatalf("Lead must consider symbols outside Main Top60: %+v", lead.Candidates)
	}
}
func TestStage3SharedExitPriorityGolden(t *testing.T) {
	cases := []struct {
		name             string
		roi              float64
		auto, permission bool
		want             tradeExitReason
		calls            int
	}{
		{"auto_over_loss", -0.08, true, true, tradeExitAuto, 0},
		{"stop_loss", -0.08, false, true, tradeExitLoss, 1},
		{"take_profit", 0.1, false, true, tradeExitProfit, 1},
		{"loss_blocked", -0.08, false, false, tradeExitHold, 1},
		{"neutral", 0, false, true, tradeExitHold, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			got := evaluateTradeExitWithRules(tc.roi, 0.08, 0.06,
				func() bool { return tc.auto },
				func() bool { calls++; return tc.permission },
			)
			if got != tc.want || calls != tc.calls {
				t.Fatalf("result=%v calls=%d want=%v calls=%d", got, calls, tc.want, tc.calls)
			}
		})
	}
}

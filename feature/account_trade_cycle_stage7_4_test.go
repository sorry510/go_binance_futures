package feature

import (
	"context"
	"sync"
	"testing"

	"github.com/adshao/go-binance/v2/futures"
	binance "go_binance_futures/feature/api/binance"
	"go_binance_futures/models"
	"go_binance_futures/types"
)

func TestStage74TwoAccountFourEnableModesSharedSignals(t *testing.T) {
	cases := []struct {
		name       string
		main, lead bool
	}{
		{"both_off", false, false},
		{"main_only", true, false},
		{"lead_only", false, true},
		{"both_on", true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			main, mainOpened, _, _ := stage3MockRunner(t, []string{"BTCUSDT"})
			main.AccountID = binance.MainAccountID
			main.Mock = false
			main.LeadSymbols = nil
			main.leadMockPermit = nil
			main.PreflightOpen = nil
			main.SelectCoins = func(_ *models.Config, coins []*models.Symbols) ([]*models.Symbols, error) { return coins[:1], nil }
			main.EvaluateEntry = stage46BothDirections
			main.AllowNewOpens = tc.main
			lead, leadOpened, _, _ := stage3MockRunner(t, []string{"BTCUSDT"})
			lead.SelectCoins = func(_ *models.Config, coins []*models.Symbols) ([]*models.Symbols, error) { return coins[:1], nil }
			lead.EvaluateEntry = stage46BothDirections
			lead.AllowNewOpens = tc.lead
			lead.PreflightOpen = func(context.Context, *models.Symbols, futures.PositionSideType, float64, float64) error { return nil }
			var wg sync.WaitGroup
			for i := 0; i < 8; i++ {
				wg.Add(2)
				go func() { defer wg.Done(); runAccountTradeCycle(main) }()
				go func() { defer wg.Done(); runAccountTradeCycle(lead) }()
				wg.Wait()
			}
			wantMain, wantLead := 0, 0
			if tc.main {
				wantMain = 16
			}
			if tc.lead {
				wantLead = 16
			}
			if *mainOpened != wantMain || *leadOpened != wantLead {
				t.Fatalf("two-account signal drift main=%d want=%d lead=%d want=%d", *mainOpened, wantMain, *leadOpened, wantLead)
			}
		})
	}
}
func TestStage74LeadPreflightFailureDoesNotChangeMainIntents(t *testing.T) {
	main, mainOpened, _, _ := stage3MockRunner(t, []string{"BTCUSDT"})
	main.AccountID = binance.MainAccountID
	main.Mock = false
	main.LeadSymbols = nil
	main.leadMockPermit = nil
	main.PreflightOpen = nil
	main.SelectCoins = func(_ *models.Config, coins []*models.Symbols) ([]*models.Symbols, error) { return coins[:1], nil }
	main.EvaluateEntry = stage46BothDirections
	lead, leadOpened, _, _ := stage3MockRunner(t, []string{"BTCUSDT"})
	lead.SelectCoins = func(_ *models.Config, coins []*models.Symbols) ([]*models.Symbols, error) { return coins[:1], nil }
	lead.EvaluateEntry = stage46BothDirections
	lead.PreflightOpen = func(context.Context, *models.Symbols, futures.PositionSideType, float64, float64) error {
		return context.DeadlineExceeded
	}
	for i := 0; i < 12; i++ {
		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); runAccountTradeCycle(main) }()
		go func() { defer wg.Done(); runAccountTradeCycle(lead) }()
		wg.Wait()
	}
	if *mainOpened != 24 || *leadOpened != 0 {
		t.Fatalf("Lead outage touched Main golden trade intents: main=%d lead=%d", *mainOpened, *leadOpened)
	}
}
func TestStage74NoProductionLeadTradingLoop(t *testing.T) {
	// Running the two-account simulation is intentionally restricted to tests.
	// The only production StartTrade uses the existing Main runner and cannot
	// instantiate the test-only Lead permit.
	r, _, _, _ := stage3MockRunner(t, []string{"BTCUSDT"})
	r.Mock = false
	r.leadMockPermit = nil
	if err := r.validate(); err == nil {
		t.Fatal("unreviewed live Lead runner passed validation")
	}
	_ = types.FuturesPosition{}
}

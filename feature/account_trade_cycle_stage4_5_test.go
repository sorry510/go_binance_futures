package feature

import (
	"context"
	"errors"
	"testing"

	"github.com/adshao/go-binance/v2/futures"
	binance "go_binance_futures/feature/api/binance"
	"go_binance_futures/models"
	"go_binance_futures/service/leadaccount"
)

func TestStage45LeadFaultNeverStopsMainTradeRunner(t *testing.T) {
	// No shared lead/global config, no actual network or private Binance calls.
	lead, leadOpens, _, _ := stage3MockRunner(t, []string{"BTCUSDT"})
	lead.PreflightOpen = func(context.Context, *models.Symbols, futures.PositionSideType, float64, float64) error {
		return leadaccount.ErrLeadPendingReconcile
	}
	main, mainOpens, _, _ := stage3MockRunner(t, []string{"BTCUSDT"})
	main.AccountID = binance.MainAccountID
	main.Mock = false
	main.LeadSymbols = nil
	main.leadMockPermit = nil
	main.PreflightOpen = nil
	if err := main.validate(); err != nil {
		t.Fatalf("Main adapter unexpectedly requires Lead gate: %v", err)
	}
	runAccountTradeCycle(lead)
	runAccountTradeCycle(main)
	if *leadOpens != 0 || *mainOpens == 0 {
		t.Fatalf("Lead fault stopped Main cycle: lead=%d main=%d", *leadOpens, *mainOpens)
	}
}

func TestStage45LeadPreflightErrorDoesNotRemoveSafeExitHooks(t *testing.T) {
	runner, opens, _, _ := stage3MockRunner(t, []string{"BTCUSDT"})
	runner.PreflightOpen = func(context.Context, *models.Symbols, futures.PositionSideType, float64, float64) error {
		return errors.New("private raw error remains unlogged")
	}
	// Validate and check preflight, but do not alter EvaluateExit or Close
	// callbacks which are needed for the previously opened Lead positions.
	prev := runner.EvaluateExit
	if prev == nil || runner.SubmitClose == nil || runner.SyncPositions == nil {
		t.Fatal("lead exit or reconciliation callbacks unexpectedly removed")
	}
	runAccountTradeCycle(runner)
	if *opens != 0 {
		t.Fatalf("risk rejection bypassed: %d", *opens)
	}
	if runner.EvaluateExit == nil || runner.SubmitClose == nil {
		t.Fatal("pausing openings destroyed exit route")
	}
}

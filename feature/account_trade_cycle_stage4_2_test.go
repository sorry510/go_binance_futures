package feature

import (
	"context"
	"errors"
	"testing"

	"github.com/adshao/go-binance/v2/futures"
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/feature/strategy"
	"go_binance_futures/models"
	"go_binance_futures/service/leadaccount"
	"go_binance_futures/types"
)

func TestStage42LeadPreflightDeniesAllOrdersWhenRiskNotAuthorized(t *testing.T) {
	r, opens, _, history := stage3MockRunner(t, []string{"BTCUSDT", "ETHUSDT"})
	risk := leadaccount.NewRiskController()
	r.PreflightOpen = func(ctx context.Context, coin *models.Symbols, side futures.PositionSideType, qty, price float64) error {
		d := risk.CheckOpen(ctx, leadaccount.RiskOpenOrder{AccountID: binance.LeadAccountID, Symbol: coin.Symbol, Side: string(side), Quantity: qty, ReferencePrice: price}, leadaccount.RiskSnapshot{AccountID: binance.LeadAccountID}, r.LeadSymbols)
		if d.Allowed {
			return nil
		}
		return errors.New("blocked by default-deny Stage 4-2 risk gate")
	}
	runAccountTradeCycle(r)
	if *opens != 0 || *history != 0 {
		t.Fatalf("Lead risk gate was bypassed, opens=%d history=%d", *opens, *history)
	}
}
func TestStage42LeadDirectionPreflightIndependent(t *testing.T) {
	r, _, _, _ := stage3MockRunner(t, []string{"BTCUSDT"})
	r.LoadSymbols = func() ([]*models.Symbols, error) {
		return []*models.Symbols{{Symbol: "BTCUSDT", Usdt: "100", Leverage: 4, TickSize: "0.1", StepSize: "0.01", Profit: "8", Loss: "6"}}, nil
	}
	r.SelectCoins = func(_ *models.Config, all []*models.Symbols) ([]*models.Symbols, error) { return all, nil }
	r.EvaluateEntry = func(*models.Symbols, []types.FuturesPosition) strategy.OpenResult {
		return strategy.OpenResult{CanLong: true, CanShort: true}
	}
	called := []string{}
	r.PreflightOpen = func(_ context.Context, _ *models.Symbols, side futures.PositionSideType, qty, price float64) error {
		called = append(called, string(side))
		if side == futures.PositionSideTypeLong {
			return errors.New("long denied")
		}
		return nil
	}
	submitted := []string{}
	r.SubmitOpen = func(snap *tradeCycleAccountSnapshot, sym string, qty, price float64, side futures.SideType, pos futures.PositionSideType, kind futures.OrderType, hash string) (*futures.CreateOrderResponse, error) {
		submitted = append(submitted, string(pos))
		snap.RecordPendingOpen(sym, side, pos, int64(len(submitted)))
		return &futures.CreateOrderResponse{OrderID: int64(len(submitted))}, nil
	}
	runAccountTradeCycle(r)
	if len(called) != 2 || called[0] != "LONG" || called[1] != "SHORT" || len(submitted) != 1 || submitted[0] != "SHORT" {
		t.Fatalf("lead preflight improperly skipped other direction: calls=%v orders=%v", called, submitted)
	}
}
func TestStage42MainPathDoesNotCallLeadPreflight(t *testing.T) {
	r, opens, _, _ := stage3MockRunner(t, []string{"BTCUSDT"})
	r.AccountID = binance.MainAccountID
	r.Mock = false
	r.leadMockPermit = nil
	r.LeadSymbols = nil
	calls := 0
	r.PreflightOpen = func(context.Context, *models.Symbols, futures.PositionSideType, float64, float64) error {
		calls++
		return errors.New("never read Lead risk state from Main")
	}
	runAccountTradeCycle(r)
	if calls != 0 || *opens == 0 {
		t.Fatalf("Main affected by Lead risk hook, calls=%d opens=%d", calls, *opens)
	}
}
func TestStage42LeadRunnerMissingPreflightRejected(t *testing.T) {
	r, _, _, _ := stage3MockRunner(t, []string{"BTCUSDT"})
	r.PreflightOpen = nil
	if err := r.validate(); err == nil {
		t.Fatal("Lead trade loop accepted missing risk gate")
	}
}

package feature

import (
	"context"
	"errors"
	"testing"

	"github.com/adshao/go-binance/v2/futures"
	binance "go_binance_futures/feature/api/binance"
	"go_binance_futures/models"
	"go_binance_futures/service/futuresownership"
	"go_binance_futures/types"
)

func TestLead03IndependentPendingDoesNotStopUnrelatedManagedExit(t *testing.T) {
	live := []types.FuturesPosition{
		{Symbol: "BTCUSDT", Side: "LONG", Amount: "1", EntryPrice: "100", MarkPrice: "100", Leverage: 4},
		{Symbol: "ETHUSDT", Side: "SHORT", Amount: "-2", EntryPrice: "100", MarkPrice: "95", Leverage: 4},
	}
	managed := []models.FuturesManagedPosition{
		{AccountID: "lead", Owner: futuresownership.OwnerAutoStrategy, Symbol: "BTCUSDT", PositionSide: "LONG", Status: futuresownership.PositionActive, ManagedQty: 1, EntryPrice: 100, SourceRef: "btc"},
		{AccountID: "lead", Owner: futuresownership.OwnerAutoStrategy, Symbol: "ETHUSDT", PositionSide: "SHORT", Status: futuresownership.PositionActive, ManagedQty: 1.5, EntryPrice: 100, SourceRef: "eth"},
	}
	pending := []models.FuturesManagedOrder{{
		AccountID: "lead", Owner: futuresownership.OwnerAutoStrategy, Symbol: "BTCUSDT", PositionSide: "LONG",
		ClientOrderID: "lead_pending_btc", Status: futuresownership.OrderSubmitted,
	}}
	verified, blocked, err := verifyLeadManagedBySymbol(live, managed, pending)
	if err != nil || !blocked["BTCUSDT"] || blocked["ETHUSDT"] || len(verified) != 1 ||
		verified[0].Position.Symbol != "ETHUSDT" || verified[0].Position.Amount != "-1.5" {
		t.Fatalf("unrelated ETH exit blocked by BTC: %+v blocked=%+v err=%v", verified, blocked, err)
	}
	r, port, coin, _ := stage032Hooks(t, "MARKET")
	// BTC has a normal exchange-accepted pending order. Its repeated entry
	// must be rejected by the same shared preflight path as Main.
	r.leadPendingSymbols = blocked
	if err := r.PreflightOpen(context.Background(), coin, futures.PositionSideTypeLong, 1, 100); !errors.Is(err, ErrLeadStrategyTicket) {
		t.Fatalf("BTC pending did not block repeated open: %v", err)
	}
	if port.preflights != 0 || port.opens != 0 {
		t.Fatal("blocked coin reached broker")
	}
	r.leadPendingSymbols = nil
	if err := r.PreflightOpen(context.Background(), coin, futures.PositionSideTypeLong, 1, 100); err != nil {
		t.Fatal(err)
	}
	r.leadPendingSymbols = blocked
	if _, err := r.SubmitOpen(newTradeCycleAccountSnapshot(nil, nil), "BTCUSDT", 1, 0,
		futures.SideTypeBuy, futures.PositionSideTypeLong, futures.OrderTypeMarket, "hash"); !errors.Is(err, ErrLeadStrategyTicket) {
		t.Fatalf("new pending after preflight did not prevent open: %v", err)
	}
	if port.opens != 0 {
		t.Fatal("race with pending order reached Binance")
	}
}

func TestLead03MalformedPendingOrForeignManagedAlwaysFails(t *testing.T) {
	snapshot := []types.FuturesPosition{{Symbol: "ETHUSDT", Side: "LONG", Amount: "1", EntryPrice: "100", MarkPrice: "100", Leverage: 4}}
	managed := []models.FuturesManagedPosition{{AccountID: "lead", Owner: futuresownership.OwnerAutoStrategy, Symbol: "ETHUSDT",
		PositionSide: "LONG", Status: futuresownership.PositionActive, ManagedQty: 1, EntryPrice: 100}}
	base := models.FuturesManagedOrder{AccountID: "lead", Symbol: "BTCUSDT", PositionSide: "LONG", ClientOrderID: "lead_order"}
	cases := []struct {
		name  string
		alter func(*models.FuturesManagedOrder)
	}{
		{"foreign", func(p *models.FuturesManagedOrder) { p.AccountID = "main" }},
		{"no_symbol", func(p *models.FuturesManagedOrder) { p.Symbol = "" }},
		{"bad_hedge", func(p *models.FuturesManagedOrder) { p.PositionSide = "BOTH" }},
		{"no_client_id", func(p *models.FuturesManagedOrder) { p.ClientOrderID = "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			order := base
			tc.alter(&order)
			if _, _, err := verifyLeadManagedBySymbol(snapshot, managed, []models.FuturesManagedOrder{order}); !errors.Is(err, errLeadTradeIOUnverified) {
				t.Fatalf("invalid Lead pending accepted: %v", err)
			}
		})
	}
	managed[0].AccountID = "main"
	if _, _, err := verifyLeadManagedBySymbol(snapshot, managed, []models.FuturesManagedOrder{base}); !errors.Is(err, errLeadTradeIOUnverified) {
		t.Fatalf("invalid Lead ownership accepted: %v", err)
	}
}

func TestLead03NoPendingWorksWithCurrentSharedRunner(t *testing.T) {
	r, port, coin, _ := stage032Hooks(t, "MARKET")
	r.leadPendingSymbols = map[string]bool{}
	if err := r.PreflightOpen(context.Background(), coin, futures.PositionSideTypeShort, 1, 100); err != nil {
		t.Fatal(err)
	}
	if _, err := r.SubmitOpen(newTradeCycleAccountSnapshot(nil, nil), "BTCUSDT", 1, 0,
		futures.SideTypeSell, futures.PositionSideTypeShort, futures.OrderTypeMarket, "hash"); err != nil {
		t.Fatal(err)
	}
	if port.opens != 1 {
		t.Fatalf("safe opening failed: %d", port.opens)
	}
}

func TestLead03MainSharedRunnerRemainsUntouched(t *testing.T) {
	// UF-1: Main's entry switch is now an entry-only gate. Lead isolation
	// remains identical for both OFF and ON Main configurations.
	for _, enabled := range []int{0, 1} {
		r := mainTradeRunner(&models.Config{FutureEnable: enabled})
		if r.AccountID != binance.MainAccountID || r.LeadSymbols != nil ||
			r.leadPendingSymbols != nil || r.AllowNewOpens != (enabled == 1) {
			t.Fatalf("Main account mutated by Lead scope: enable=%d runner=%+v", enabled, r)
		}
	}
}

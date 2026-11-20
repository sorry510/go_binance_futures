package feature

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/adshao/go-binance/v2/futures"

	"go_binance_futures/feature/api/binance"
	"go_binance_futures/feature/strategy"
	"go_binance_futures/models"
	"go_binance_futures/types"
)

func TestTradeOpenCooldownBoundaryAndExtension(t *testing.T) {
	var gate tradeOpenCooldown
	start := time.Unix(1000, 0)
	gate.blockFrom(start)
	if !gate.blocksAt(start) || !gate.blocksAt(start.Add(3*time.Minute-time.Millisecond)) {
		t.Fatal("new entries must be blocked for the full 3 minutes")
	}
	if gate.blocksAt(start.Add(3 * time.Minute)) {
		t.Fatal("new entries must resume exactly at the 3-minute boundary")
	}
	// A later successful CLOSE must extend the NEW-ENTRY cooldown; it must
	// not prevent the close itself, even when the entry cooldown nearly ends.
	gate.blockFrom(start.Add(2*time.Minute + 55*time.Second))
	if !gate.blocksAt(start.Add(5*time.Minute+54*time.Second)) ||
		gate.blocksAt(start.Add(5*time.Minute+55*time.Second)) {
		t.Fatal("close-time cooldown did not extend the new-entry boundary")
	}
	gate.blockFrom(start) // older events must never shorten an active cooldown
	if !gate.blocksAt(start.Add(5*time.Minute + 54*time.Second)) {
		t.Fatal("older event shortened the cooldown")
	}
}

func TestTradeOpenCooldownSeparateMainAndLeadAccounts(t *testing.T) {
	now := time.Now()
	var main, lead tradeOpenCooldown
	main.blockFrom(now)
	if !main.blocksAt(now) || lead.blocksAt(now) {
		t.Fatal("Main cooldown must not block Lead entries")
	}
	lead.blockFrom(now.Add(5 * time.Second))
	if !lead.blocksAt(now.Add(10*time.Second)) || main.blocksAt(now.Add(3*time.Minute)) {
		t.Fatal("Lead cooldown must be independent of Main")
	}
	if got := mainTradeRunner(&models.Config{}).OpenCooldown; got != &mainTradeOpenCooldown {
		t.Fatal("production Main runner is not using the Main cooldown")
	}
}

func TestTradeOpenCooldownStillClosesAllPositionsAndOnlyBlocksNewEntries(t *testing.T) {
	r, opens, _ := stage3MockRunner(t, []string{"BTCUSDT", "ETHUSDT"})
	r.AccountID = binance.MainAccountID
	r.Mock = false
	r.leadMockPermit = nil
	r.LeadSymbols = nil
	r.OpenCooldown = &tradeOpenCooldown{}
	r.AllowNewOpens = true
	r.Sleep = func(time.Duration) { t.Fatal("successful close or open must never sleep") }
	positions := []types.FuturesPosition{
		{Symbol: "BTCUSDT", Side: "LONG", Amount: "2", Leverage: 4, MarkPrice: "100", UnrealizedProfit: "-10"},
		{Symbol: "ETHUSDT", Side: "SHORT", Amount: "-3", Leverage: 4, MarkPrice: "100", UnrealizedProfit: "-12"},
	}
	r.ReadPositions = func(context.Context) ([]types.FuturesPosition, error) { return positions, nil }
	r.SyncPositions = func(p []types.FuturesPosition) ([]ownedTradePosition, error) {
		return []ownedTradePosition{
			{Position: p[0], Owner: "auto_strategy", SourceRef: "auto_strategy:BTCUSDT"},
			{Position: p[1], Owner: "auto_strategy", SourceRef: "auto_strategy:ETHUSDT"},
		}, nil
	}
	r.EvaluateExit = func(strategy.CloseParams, float64, float64) tradeExitReason {
		return tradeExitAuto
	}
	var attempts []string
	r.SubmitClose = func(_, _, symbol string, _ float64, _ futures.PositionSideType) (*futures.CreateOrderResponse, error) {
		attempts = append(attempts, symbol)
		return &futures.CreateOrderResponse{OrderID: int64(len(attempts))}, nil
	}
	r.SelectCoins = func(*models.Config, []*models.Symbols) ([]*models.Symbols, error) {
		t.Fatal("must not evaluate new entries after successful closes")
		return nil, nil
	}
	start := time.Now()
	runAccountTradeCycle(r)
	if !reflect.DeepEqual(attempts, []string{"BTCUSDT", "ETHUSDT"}) || *opens != 0 {
		t.Fatalf("cooldown blocked exits or allowed new entries: closes=%v opens=%d", attempts, *opens)
	}
	if !r.OpenCooldown.blocksAt(time.Now()) {
		t.Fatal("successful closes must start new-entry cooldown")
	}
	if elapsed := time.Since(start); elapsed >= time.Second {
		t.Fatalf("close cycle blocked unexpectedly for %v", elapsed)
	}

	// Existing gate must prevent opening while permitting future exit scans.
	r.ReadPositions = func(context.Context) ([]types.FuturesPosition, error) { return nil, nil }
	r.SyncPositions = func([]types.FuturesPosition) ([]ownedTradePosition, error) { return nil, nil }
	runAccountTradeCycle(r)
	if *opens != 0 {
		t.Fatal("cooldown incorrectly allowed new entry")
	}

	// Deterministically advance the gate beyond expiry, without waiting 3 minutes.
	r.OpenCooldown.mu.Lock()
	r.OpenCooldown.blockedUntil = time.Now().Add(-time.Second)
	r.OpenCooldown.mu.Unlock()
	r.SelectCoins = func(_ *models.Config, coins []*models.Symbols) ([]*models.Symbols, error) {
		return coins[:1], nil
	}
	r.EvaluateEntry = func(*models.Symbols, []types.FuturesPosition) strategy.OpenResult {
		return strategy.OpenResult{CanLong: true}
	}
	runAccountTradeCycle(r)
	if *opens != 1 || !r.OpenCooldown.blocksAt(time.Now()) {
		t.Fatalf("after expiry new entry must resume and recool: opens=%d blocked=%v", *opens, r.OpenCooldown.blocksAt(time.Now()))
	}
}

func TestTradeOpenCooldownFailedCloseDoesNotExtendCooldown(t *testing.T) {
	r, _, _ := stage3MockRunner(t, []string{"BTCUSDT"})
	r.AccountID = binance.MainAccountID
	r.Mock = false
	r.leadMockPermit = nil
	r.LeadSymbols = nil
	r.OpenCooldown = &tradeOpenCooldown{}
	r.AllowNewOpens = false
	r.ReadPositions = func(context.Context) ([]types.FuturesPosition, error) {
		return []types.FuturesPosition{{Symbol: "BTCUSDT", Side: "LONG", Amount: "1", Leverage: 4, MarkPrice: "100", UnrealizedProfit: "-10"}}, nil
	}
	r.SyncPositions = func(positions []types.FuturesPosition) ([]ownedTradePosition, error) {
		return []ownedTradePosition{{Position: positions[0], Owner: "auto_strategy", SourceRef: "auto_strategy:BTCUSDT"}}, nil
	}
	r.EvaluateExit = func(strategy.CloseParams, float64, float64) tradeExitReason {
		return tradeExitAuto
	}
	r.SubmitClose = func(string, string, string, float64, futures.PositionSideType) (*futures.CreateOrderResponse, error) {
		return nil, errors.New("test-only rejected close submission")
	}
	runAccountTradeCycle(r)
	if r.OpenCooldown.blocksAt(time.Now()) {
		t.Fatal("failed close must not create a successful-submission cooldown")
	}
}
